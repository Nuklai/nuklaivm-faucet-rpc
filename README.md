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

- Database Operations

  You can use the scripts/db.sh script to interact with the SQLite database.

  - Get Transactions by TxID:

    ```bash
    ./scripts/db.sh get-transaction-by-txid <TxID>
    ```

  - Get All Transactions:

    ```bash
    ./scripts/db.sh get-all-transactions
    ```

## Build & Run with Docker

- Build

  ```bash
  docker build -t nuklai-faucet .
  ```

- Run

  ```bash
  docker container rm -f nuklai-faucet;
  docker run --env-file .env -d -p 10591:10591 --name nuklai-faucet nuklai-faucet
  ```

- Read the logs

  ```bash
  docker container logs -f nuklai-faucet
  ```
