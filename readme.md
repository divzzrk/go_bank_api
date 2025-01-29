# Golang Backend Banking Ledger Service

## Features
- **Account Management**: Create accounts with specified initial balances.
- **Transaction Handling**: Process deposits and withdrawals securely.
- **Ledger Maintenance**: Maintain a detailed transaction log for each account.
- **Consistency & Integrity**: Implement ACID-like guarantees to prevent double-spending.
- **Scalability**: Support high transaction volumes with horizontal scaling.
- **Asynchronous Processing**: Utilize a message broker (e.g., RabbitMQ/Kafka) to handle transaction requests efficiently.
- **Comprehensive Testing**: Include unit, integration, and feature tests.

## Tech Stack
- **Programming Language**: Golang
- **Database**:
  - PostgreSQL for account balances
  - MongoDB/DynamoDB for transaction logs
- **Message Broker**: RabbitMQ/Kafka
- **API Framework**: Gin
- **Testing**: Go testing framework with mocks
- **Containerization**: Docker & Docker Compose

## Architecture Overview
The system follows a microservices-like architecture with the following components:
1. **API Gateway**: Exposes RESTful endpoints for interacting with accounts and transactions.
2. **Transaction Processor**: Handles transaction requests asynchronously via a message broker.
3. **Database Layer**: Uses PostgreSQL for accounts and MongoDB/DynamoDB for transaction history.
4. **Message Queue**: Ensures reliable processing of transaction requests.

## Setup Instructions

### Prerequisites
- Install [Golang](https://golang.org/dl/)
- Install [Docker](https://www.docker.com/get-started)
- Install [Docker Compose](https://docs.docker.com/compose/install/)

### Running the Application
1. Clone the repository:
   ```sh
   git clone https://github.com/your-repo/banking-ledger.git
   cd banking-ledger
   ```
2. Start services using Docker Compose:
   ```sh
   docker-compose up --build
   ```
3. The API will be available at `http://localhost:8080`

### API Endpoints
| Method | Endpoint | Description |
|--------|---------|-------------|
| `POST` | `/create_account` | Create a new account |
| `POST` | `/transaction` | Process a deposit/withdrawal |
| `GET` | `/transaction/:account_id` | Retrieve transaction history |

## Testing
Run unit and integration tests using:
```sh
make test
```


