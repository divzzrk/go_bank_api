package main

import (
	"encoding/json"
	"os"

	"github.com/streadway/amqp"
)

const (
	TransactionQueue = "transaction_queue"
)

// QueuedTransaction represents a transaction message
type QueuedTransaction struct {
	AccountID     string  `json:"account_id,omitempty"`
	FromAccountID string  `json:"from_account_id,omitempty"`
	ToAccountID   string  `json:"to_account_id,omitempty"`
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
}

// RabbitMQ connection wrapper
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// Initialize RabbitMQ connection
func NewRabbitMQ() (*RabbitMQ, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URI"))
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare queue
	_, err = ch.QueueDeclare(
		TransactionQueue, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{
		conn:    conn,
		channel: ch,
	}, nil
}

// Close connections
func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// PublishTransaction publishes a transaction to the queue
func (r *RabbitMQ) PublishTransaction(transaction QueuedTransaction) error {
	body, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	return r.channel.Publish(
		"",               // exchange
		TransactionQueue, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
}
