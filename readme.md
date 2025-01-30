# Banking Ledger Service

## Overview
This is a backend service for a banking ledger that manages transactions, accounts, and balances efficiently. The service is built using Golang and supports high transaction volumes with horizontal scalability.

## Features
- Account creation and management
- Transaction processing (deposits, withdrawals, transfers)
- Balance tracking
- Unit tests for core functionalities

## Technologies Used
- **Golang** - Backend logic
- **PostgreSQL** - Database for account and transaction storage
- **MongoDB** - Database for logging transactions
- **RabbitMQ** - Queuing transactions
- **Docker** - Containerization
- **REST** - API communication

## Setup

### Prerequisites
- Go 1.21+
- PostgreSQL
- Docker & Docker Compose
- Kubernetes (for production scaling)

## API Endpoints

### Create Account
- `POST /create` - Create a new account.
- Here it wont allow to create an account with same phone number, it should always be a 10 digit number.
- User ID and account ID will automatically be generated.
- ```{
  "username": "Alex Martin",
  "phone": "9879876561"
### Deposit Amount
- `POST /transaction` - Deposit amount to your account.
- ```
  "account_id": "331491007",
  "type": "deposit",
  "amount": 50000
### Withdraw Amount
- `POST /transaction` - Deposit amount to your account.
- ```
  "account_id": "331491007",
  "type": "withdrawal",
  "amount": 500
  