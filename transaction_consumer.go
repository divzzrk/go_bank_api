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
	err = tx.QueryRow("SELECT balance FROM users WHERE account_id = $1 FOR UPDATE", qt.AccountID).Scan(&currentBalance)
	if err != nil {
		return err
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
	default:
		return fmt.Errorf("invalid transaction type")
	}

	// Update balance
	_, err = tx.Exec("UPDATE users SET balance = $1 WHERE account_id = $2", newBalance, qt.AccountID)
	if err != nil {
		return err
	}

	// Log transaction in MongoDB
	transactionLog := TransactionLog{
		AccountID:      qt.AccountID,
		Type:           qt.Type,
		Amount:         qt.Amount,
		CreatedAt:      time.Now(),
		CurrentBalance: newBalance,
	}

	_, err = tc.mongoCollection.InsertOne(context.TODO(), transactionLog)
	if err != nil {
		return err
	}

	return tx.Commit()
}
