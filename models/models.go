package models

import "time"

// User struct
type User struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Phone     string  `json:"phone"`
	AccountID string  `json:"account_id"`
	Balance   float64 `json:"balance"`
}

// Transaction struct for MongoDB logs
type TransactionLog struct {
	ID             string    `bson:"_id,omitempty" json:"id,omitempty"`
	AccountID      string    `bson:"account_id" json:"account_id,omitempty"`
	FromAccountID  string    `bson:"from_account_id" json:"from_account_id,omitempty"`
	ToAccountID    string    `bson:"to_account_id" json:"to_account_id,omitempty"`
	Type           string    `bson:"type" json:"type"`
	Amount         float64   `bson:"amount" json:"amount"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	CurrentBalance float64   `bson:"current_balance" json:"current_balance"`
}

type Transaction struct {
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        float64   `json:"amount"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
}
