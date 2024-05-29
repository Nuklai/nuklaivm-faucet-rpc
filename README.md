# Nuklai Faucet

## Build & Run from Source

To build the binary from the source, use the following command:

```bash
./scripts/build.sh
```

Before running, copy the example environment file to .env and configure it with the correct values:

```bash
cp .env.example .env;
```

Then, run the application:

```bash
./build/nuklai-faucet
```

NOTE: Make sure to have the correct values for PostgreSQL in your .env file.

### Database Operations

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

To build the Docker image, use the following command:

```bash
./scripts/build.sh docker
```

Start the Docker containers:

```bash
./scripts/run_docker.sh start
```

To stop the Docker containers:

```bash
./scripts/run_docker.sh stop
```

To view the logs of the Docker container:

```bash
./scripts/run_docker.sh logs
```
