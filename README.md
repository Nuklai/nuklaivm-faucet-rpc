# Nuklai Faucet

## Build & Run from Source

- Build

  ```bash
  ./scripts/build.sh
  ```

- Run

  ```bash
  cp .env.example .env;
  ./build/nuklai-faucet
  ```

  NOTE: Make sure to have the correct values for Postgres on your .env file

- Database Operations

  You can use the scripts/db.sh script to interact with the SQLite database.

- Get All Transactions:

  ```bash
  ./scripts/db.sh get-all-transactions
  ```

- Get Transactions by TxID:

  ```bash
  ./scripts/db.sh get-transaction-by-txid <TxID>
  ```

- Get Transactions by user:

  ```bash
  ./scripts/db.sh get-transactions-by-user <WalletAddress>
  ```

## Build & Run with Docker

- Build

  ```bash
  ./scripts/build.sh docker
  ```

- Run

  ```bash
  ./scripts/run_docker.sh start
  ```

- Stop the docker containers

  ```bash
  ./scripts/run_docker.sh stop
  ```

  - Check the logs from docker containers

  ```bash
  ./scripts/run_docker.sh logs
  ```
