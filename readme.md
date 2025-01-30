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
- Throws insuffient balance if there is no sufficient balance
- ```
  "account_id": "331491007",
  "type": "withdrawal",
  "amount": 500
### Transfer Amount
- `POST /transaction` - To transfer amount from one account to another.
- Throws insuffient balance if there is no sufficient balance
- Both the account ids should be valid else a 404 error stating account not found will be thrown
- ```
  "from_account_id": "331491007",
  "to_account_id": "323477227",
  "type": "transfer",
  "amount": 500
### Get Transaction Details
- `GET /transaction/:account_id` - Get transaction details for an account.
- Throws insuffient balance if there is no sufficient balance
- Both the account ids should be valid else a 404 error stating account not found will be thrown
- ```
  http://localhost:8000/transaction/your_account_id
### Get All Users
- `GET /users` - Get all the users.
- ```
  http://localhost:8000/users


### Output of unit test
```bash
golang_bank_apis % go test -v ./...
=== RUN   TestValidatePhone
=== RUN   TestValidatePhone/Valid_phone_number
=== RUN   TestValidatePhone/Valid_phone_with_dashes
=== RUN   TestValidatePhone/Valid_phone_with_spaces
=== RUN   TestValidatePhone/Invalid_phone_-_too_short
=== RUN   TestValidatePhone/Invalid_phone_-_too_long
=== RUN   TestValidatePhone/Invalid_phone_-_contains_letters
=== RUN   TestValidatePhone/Empty_phone
--- PASS: TestValidatePhone (0.00s)
    --- PASS: TestValidatePhone/Valid_phone_number (0.00s)
    --- PASS: TestValidatePhone/Valid_phone_with_dashes (0.00s)
    --- PASS: TestValidatePhone/Valid_phone_with_spaces (0.00s)
    --- PASS: TestValidatePhone/Invalid_phone_-_too_short (0.00s)
    --- PASS: TestValidatePhone/Invalid_phone_-_too_long (0.00s)
    --- PASS: TestValidatePhone/Invalid_phone_-_contains_letters (0.00s)
    --- PASS: TestValidatePhone/Empty_phone (0.00s)
=== RUN   TestCreateUser
=== RUN   TestCreateUser/Valid_user_creation
2025/01/30 15:08:46 Received user: {ID: Username:testuser Phone:1234567890 AccountID: Balance:0}
    main_test.go:128: Response Status: 201
    main_test.go:129: Response Body: {"id":"1","username":"testuser","phone":"1234567890","account_id":"787869862","balance":0}
=== RUN   TestCreateUser/Invalid_phone_number
2025/01/30 15:08:46 Received user: {ID: Username:testuser Phone:123 AccountID: Balance:0}
    main_test.go:128: Response Status: 400
    main_test.go:129: Response Body: {"error":"Invalid phone number"}
=== RUN   TestCreateUser/Invalid_user_name
2025/01/30 15:08:46 Received user: {ID: Username:abc Phone:9898767654 AccountID: Balance:0}
    main_test.go:128: Response Status: 400
    main_test.go:129: Response Body: {"error":"Username must be at least 5 characters"}
--- PASS: TestCreateUser (0.00s)
    --- PASS: TestCreateUser/Valid_user_creation (0.00s)
    --- PASS: TestCreateUser/Invalid_phone_number (0.00s)
    --- PASS: TestCreateUser/Invalid_user_name (0.00s)
=== RUN   TestGetUsers
=== RUN   TestGetUsers/Successfully_retrieve_users
=== RUN   TestGetUsers/No_users_found
--- PASS: TestGetUsers (0.00s)
    --- PASS: TestGetUsers/Successfully_retrieve_users (0.00s)
    --- PASS: TestGetUsers/No_users_found (0.00s)
PASS
ok      api     0.274s```