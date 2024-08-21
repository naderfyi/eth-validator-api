# Ethereum Validator API

## Overview
This application provides a RESTful API to interact with Ethereum blockchain data, specifically for retrieving block rewards and sync committee duties.

## Endpoints

### GET /blockreward/{slot}
- Retrieves the block reward for a specific Ethereum slot.
- **Parameters:**
  - `slot`: The slot number (integer).
- **Response:**
  - `status`: Indicates if the block was produced via MEV relay or a vanilla process.
  - `reward`: The reward in GWEI received for the block.

### GET /syncduties/{slot}
- Retrieves a list of validators that had sync committee duties for a specific slot.
- **Parameters:**
  - `slot`: The slot number (integer).
- **Response:**
  - `validators`: A list of public keys of the validators.

## Getting Started

### Prerequisites
- Go (Golang) installed on your machine.

### Building the Application
```bash
go mod tidy
go build -o eth-validator-api
```

### Running the Application
```bash
./eth-validator-api
```

### Example API Calls

- **Retrieve Block Reward:**
  ```bash
  curl http://localhost:8080/blockreward/9778014
  ```

- **Retrieve Sync Duties:**
  ```bash
  curl http://localhost:8080/syncduties/9778014
  ```

## Design Choices
- **Gin Framework:** Chosen for its simplicity and performance in creating RESTful APIs in Go.
- **Modular Handlers:** Each API route is handled in a separate file for better maintainability.

