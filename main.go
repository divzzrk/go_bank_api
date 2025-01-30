// Package main implements a banking application that handles user account
// management and transaction processing using PostgreSQL, MongoDB, and RabbitMQ.
// The application provides an API for creating users, processing transactions,
// retrieving transaction history, and listing all users.
//
// Key Components:
// - PostgreSQL: Used to store user and account information.
// - MongoDB: Used to log transactions.
// - RabbitMQ: Used to queue and process transactions asynchronously.
// - Gin: HTTP web framework used for handling API requests.
//
// This application demonstrates a microservices architecture with integration
// between database systems and message queues for resilient transaction processing.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// Main entry point for the application. Connects to PostgreSQL and MongoDB,
// initializes DB tables, starts a transaction consumer in a goroutine,
// and starts a Gin web server.
func main() {
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer mongoClient.Disconnect(context.TODO())

	mongoDB := mongoClient.Database("banking_app")
	transactionsCollection := mongoDB.Collection("transactions")

	// Initialize DB tables
	initializePostgres(db)

	// Initialize RabbitMQ
	rabbitMQ, err := NewRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitMQ.Close()

	// Start transaction consumer in a goroutine
	consumer := NewTransactionConsumer(db, transactionsCollection, rabbitMQ)
	go consumer.Start()

	r := gin.Default()

	// Routes
	r.POST("/create", func(c *gin.Context) { createUser(c, db) })
	// r.POST("/transaction", func(c *gin.Context) { processTransaction(c, db, transactionsCollection) })
	r.GET("/transaction/:account_id", func(c *gin.Context) { getTransactionHistory(c, transactionsCollection) })
	r.GET("/users", func(c *gin.Context) { getUsers(c, db) })
	r.POST("/transaction", func(c *gin.Context) {
		var qt QueuedTransaction
		if err := c.ShouldBindJSON(&qt); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the transaction
		if qt.AccountID == "" && qt.Type != "transfer" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
			return
		}
		if qt.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
			return
		}
		if qt.Type != "deposit" && qt.Type != "withdrawal" && qt.Type != "transfer" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction type"})
			return
		}
		var account_id string

		if qt.Type == "deposit" || qt.Type == "withdrawal" || qt.Type == "transfer" {
			if qt.Type == "deposit" {
				account_id = qt.AccountID
			} else {
				account_id = qt.ToAccountID
			}
			var currentBalance float64
			err := db.QueryRow("SELECT balance FROM users WHERE account_id = $1", account_id).Scan(&currentBalance)
			if err != nil {
				log.Printf("Database error querying balance: %v", err)
				if err == sql.ErrNoRows {
					c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				}
				return
			}
		}

		if qt.Type == "withdrawal" || qt.Type == "transfer" {
			if qt.Type == "withdrawal" {
				account_id = qt.AccountID
			} else {
				account_id = qt.FromAccountID
			}

			var currentBalance float64
			err := db.QueryRow("SELECT balance FROM users WHERE account_id = $1", account_id).Scan(&currentBalance)
			if err != nil {
				log.Printf("Database error querying balance: %v", err)
				if err == sql.ErrNoRows {
					c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				}
				return
			}
			log.Printf("Current balance: %f", currentBalance)
			if currentBalance < qt.Amount {
				log.Printf("Insufficient balance: %f < %f", currentBalance, qt.Amount)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
				return
			}
		}

		// Publish to queue
		if err := rabbitMQ.PublishTransaction(qt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue transaction"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"message":    "Transaction queued successfully",
			"account_id": qt.AccountID,
			"amount":     qt.Amount,
			"type":       qt.Type,
		})
	})

	log.Fatal(r.Run(":8000"))
}

// Initializes PostgreSQL tables
func initializePostgres(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT NOT NULL,
		phone TEXT UNIQUE NOT NULL,
		account_id TEXT UNIQUE NOT NULL,
		balance FLOAT NOT NULL DEFAULT 0
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS accounts (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		balance FLOAT NOT NULL DEFAULT 0
	)`)
	// user_id INTEGER REFERENCES users(id),

	if err != nil {
		log.Fatal(err)
	}
}

// createUser handles the creation of a new user account.
//
// It binds JSON input to a User struct, validates the phone number,
// generates unique IDs for the user and account, and inserts the user
// and account into the PostgreSQL database. If any errors occur during
// these processes, it returns an appropriate JSON error response. On
// success, it returns a JSON response with the created user.

func createUser(c *gin.Context, db *sql.DB) {
	var user User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validatePhone(user.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number"})
		return
	}
	user.AccountID = fmt.Sprintf("%s%d", user.ID, rand.Intn(1000000000))

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = tx.QueryRow("INSERT INTO users (username, phone, account_id, balance) VALUES ($1, $2, $3, 0) RETURNING id",
		user.Username, user.Phone, user.AccountID).Scan(&user.ID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_, err = tx.Exec("INSERT INTO accounts (user_id, balance) VALUES ($1, 0)", user.ID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// validatePhone checks if a phone number string is valid.
//
// It removes all non-digit characters (e.g., spaces, dashes, etc.),
// and checks if the remaining string consists of exactly 10 digits.
// If the phone number is valid, it returns true; otherwise, it returns false.
func validatePhone(phone string) bool {
	// Remove non-digit characters (e.g., spaces, dashes, etc.)
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, " ", "")

	// Ensure the phone number has 10 digits
	re := regexp.MustCompile(`^\d{10}$`)
	return re.MatchString(phone)
}

// processTransaction handles the actual transaction processing
//
// It logs the incoming transaction request, verifies the amount is positive,
// queries the current balance from the database, processes the transaction
// (either deposit or withdrawal), updates the balance in the database,
// logs the transaction in MongoDB, and returns a JSON response with the
// previous balance, new balance, amount, and type of transaction.
//
// If any errors occur during processing, it returns an appropriate JSON
// error response.
func processTransaction(c *gin.Context, db *sql.DB, transactionsCollection *mongo.Collection) {
	var transaction TransactionLog
	if err := c.ShouldBindJSON(&transaction); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log incoming transaction request
	log.Printf("Processing transaction: AccountID=%s, Type=%s, Amount=%f",
		transaction.AccountID, transaction.Type, transaction.Amount)

	// Verify the amount is positive
	if transaction.Amount <= 0 {
		log.Printf("Invalid amount: %f", transaction.Amount)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	var currentBalance float64
	var toAccountBalance float64
	var account_id string
	if transaction.Type == "transfer" {
		account_id = transaction.FromAccountID
	} else {
		account_id = transaction.AccountID
	}
	err := db.QueryRow("SELECT balance FROM users WHERE account_id = $1", account_id).Scan(&currentBalance)
	if err != nil {
		log.Printf("Database error querying balance: %v", err)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}
	if transaction.Type == "transfer" {

		err := db.QueryRow("SELECT balance FROM users WHERE account_id = $1", transaction.ToAccountID).Scan(&toAccountBalance)
		if err != nil {
			log.Printf("Database error querying balance: %v", err)
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			}
			return
		}
	}

	log.Printf("Current balance: %f", currentBalance)

	// Process the transaction
	var newBalance float64
	switch transaction.Type {
	case "deposit":
		newBalance = currentBalance + transaction.Amount
		log.Printf("Processing deposit: %f + %f = %f", currentBalance, transaction.Amount, newBalance)
	case "withdrawal":
		newBalance = currentBalance - transaction.Amount
		log.Printf("Processing withdrawal: %f - %f = %f", currentBalance, transaction.Amount, newBalance)
	case "transfer":
		// TODO: implement transfer
		newBalance = currentBalance - transaction.Amount
		toAccountBalance += transaction.Amount
		log.Printf("Processing transfer: %f - %f = %f", currentBalance, transaction.Amount, newBalance)

	default:
		log.Printf("Invalid transaction type: %s", transaction.Type)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction type"})
		return
	}

	if transaction.Type == "transfer" {
		account_id = transaction.AccountID
	} else {
		account_id = transaction.FromAccountID
	}
	// Update the balance in the database
	result, err := db.Exec("UPDATE users SET balance = $1 WHERE account_id = $2", newBalance, account_id)
	if err != nil {
		log.Printf("Error updating balance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Rows affected by update: %d", rowsAffected)

	if transaction.Type == "transfer" {

		result, err := db.Exec("UPDATE users SET balance = $1 WHERE account_id = $2", toAccountBalance, transaction.ToAccountID)
		if err != nil {
			log.Printf("Error updating balance: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Printf("Rows affected by update: %d", rowsAffected)
	}

	// Log the transaction in MongoDB
	transaction.CreatedAt = time.Now()
	transaction.CurrentBalance = newBalance

	_, err = transactionsCollection.InsertOne(context.TODO(), transaction)
	if err != nil {
		log.Printf("Error logging to MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Transaction successful",
		"previous_balance": currentBalance,
		"new_balance":      newBalance,
		"amount":           transaction.Amount,
		"type":             transaction.Type,
	})
}

// getTransactionHistory fetches the transaction history for a given account ID.
//
// It queries the database for all transactions for the given account ID, and
// sends a JSON response with all the transactions if successful. If no
// transactions are found, it sends a 404 response with an error message. If
// there is an error querying the database, it sends a 500 response with an
// error message.
func getTransactionHistory(c *gin.Context, transactionsCollection *mongo.Collection) {
	accountID := c.Param("account_id")

	filter := bson.M{"account_id": accountID}
	cursor, err := transactionsCollection.Find(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transaction history"})
		return
	}
	defer cursor.Close(context.TODO())

	var transactions []TransactionLog
	if err = cursor.All(context.TODO(), &transactions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transaction history"})
		return
	}

	if len(transactions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No transactions found"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// getUsers retrieves all users from the database.
//
// It queries the database for all users, and sends a JSON response with all
// the users if successful. If no users are found, it sends a 404 response with
// an error message. If there is an error querying the database, it sends a 500
// response with an error message.
func getUsers(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, username, phone, account_id, balance FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Phone, &u.AccountID, &u.Balance); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, u)
	}

	if len(users) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No users found"})
		return
	}

	c.JSON(http.StatusOK, users)
}
