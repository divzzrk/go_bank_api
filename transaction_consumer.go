package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// TransactionConsumer handles processing of queued transactions
type TransactionConsumer struct {
	db              *sql.DB
	mongoCollection *mongo.Collection
	rabbitMQ        *RabbitMQ
}

// NewTransactionConsumer creates a new consumer
func NewTransactionConsumer(db *sql.DB, mongoCollection *mongo.Collection, rabbitMQ *RabbitMQ) *TransactionConsumer {
	return &TransactionConsumer{
		db:              db,
		mongoCollection: mongoCollection,
		rabbitMQ:        rabbitMQ,
	}
}

// Start begins consuming transactions
func (tc *TransactionConsumer) Start() {
	msgs, err := tc.rabbitMQ.channel.Consume(
		TransactionQueue, // queue
		"",               // consumer
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			log.Printf("Received a transaction: %s", d.Body)

			var qt QueuedTransaction
			if err := json.Unmarshal(d.Body, &qt); err != nil {
				log.Printf("Error unmarshaling transaction: %v", err)
				d.Nack(false, true) // reject and requeue
				continue
			}

			err := tc.processTransaction(qt)
			if err != nil {
				log.Printf("Error processing transaction: %v", err)
				d.Nack(false, true) // reject and requeue
				continue
			}

			d.Ack(false) // acknowledge successful processing
		}
	}()

	log.Printf(" [*] Waiting for transaction messages. To exit press CTRL+C")
	<-forever
}

// processTransaction handles the actual transaction processing
func (tc *TransactionConsumer) processTransaction(qt QueuedTransaction) error {
	// Start a database transaction
	tx, err := tc.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentBalance float64
	var toAccountBalance float64
	var account_id string
	if qt.Type == "transfer" {
		account_id = qt.FromAccountID
	} else {
		account_id = qt.AccountID
	}
	err = tx.QueryRow("SELECT balance FROM users WHERE account_id = $1 FOR UPDATE", account_id).Scan(&currentBalance)
	if err != nil {
		return err
	}

	if qt.Type == "transfer" {
		err := tx.QueryRow("SELECT balance FROM users WHERE account_id = $1", qt.ToAccountID).Scan(&toAccountBalance)
		if err != nil {
			return err
		}
	}

	var newBalance float64
	switch qt.Type {
	case "deposit":
		newBalance = currentBalance + qt.Amount
	case "withdrawal":
		if currentBalance < qt.Amount {
			return fmt.Errorf("insufficient balance")
		}
		newBalance = currentBalance - qt.Amount
	case "transfer":
		newBalance = currentBalance - qt.Amount
		toAccountBalance += qt.Amount
		log.Printf("Processing transfer: %f - %f = %f", currentBalance, qt.Amount, newBalance)

	default:
		return fmt.Errorf("invalid transaction type")
	}
	if qt.Type != "transfer" {
		account_id = qt.AccountID
	} else {
		account_id = qt.FromAccountID
	}
	// Update balance
	_, err = tx.Exec("UPDATE users SET balance = $1 WHERE account_id = $2", newBalance, account_id)
	if err != nil {
		return err
	}

	if qt.Type == "transfer" {
		_, err = tx.Exec("UPDATE users SET balance = $1 WHERE account_id = $2", toAccountBalance, qt.ToAccountID)
		if err != nil {
			return err
		}
	}
	if qt.Type != "transfer" {

	}
	// Log transaction in MongoDB
	transactionLog := TransactionLog{
		Type:           qt.Type,
		Amount:         qt.Amount,
		CreatedAt:      time.Now(),
		CurrentBalance: newBalance,
	}

	if qt.Type != "transfer" {
		transactionLog.AccountID = qt.AccountID
	} else {
		transactionLog.FromAccountID = qt.FromAccountID
		transactionLog.ToAccountID = qt.ToAccountID
		transactionLog.AccountID = qt.FromAccountID
	}
	_, err = tc.mongoCollection.InsertOne(context.TODO(), transactionLog)
	if err != nil {
		return err
	}
	if qt.Type == "transfer" {
		transactionLog.AccountID = qt.ToAccountID
		_, err = tc.mongoCollection.InsertOne(context.TODO(), transactionLog)
		if err != nil {
			return err
		}
	}

	return tx.Commit()

}
