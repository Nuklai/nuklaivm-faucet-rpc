# Nuklai Faucet

## Disclaimer

**IMPORTANT NOTICE:** This project is currently in the alpha stage of development and is not intended for use in production environments. The software may contain bugs, incomplete features, or other issues that could cause it to malfunction. Use at your own risk.

We welcome contributions and feedback to help improve this project, but please be aware that the codebase is still under active development. It is recommended to thoroughly test any changes or additions before deploying them in a production environment.

Thank you for your understanding and support!

## Build & Run from Source

Before running, copy the example environment file to .env and configure it with the correct values:

```bash
cp .env.example .env;
```

Then, run the application:

```bash
./scripts/run.sh [start|stop]
```

## Build & Run with Docker

To build the Docker image, use the following command:

```bash
./scripts/build.sh docker
```

Start the Docker containers:

```bash
./scripts/run_docker.sh [start|stop|logs]
```

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

### RPC API Endpoints

You can interact with the JSON RPC API to request NAI and do other things.

- Check health status

```bash
curl 127.0.0.1:10591/health
```

- Get the faucet address that's used for funding other accounts

```bash
curl -X POST --data '{
    "jsonrpc": "2.0",
    "method": "faucet.faucetAddress",
    "params": {},
    "id": 1
}' -H 'content-type:application/json;' 127.0.0.1:10591/faucet
```

- Request some test NAI at `00e296a1f7085fcc3f18580f1bbe0b7c56b0ec7e962372105dafc7d86f20a0765b`

```bash
curl -X POST --data '{
    "jsonrpc": "2.0",
    "method": "faucet.requestTestFunds",
    "params": {
        "address": "00e296a1f7085fcc3f18580f1bbe0b7c56b0ec7e962372105dafc7d86f20a0765b"
    },
    "id": 1
}' -H 'content-type:application/json;' 127.0.0.1:10591/faucet
```

## How does the faucet work?

The faucet service is designed to distribute test NAI tokens to users, primarily for testing purposes on the Nuklai blockchain. The main components of the service include the main server setup, configuration management, database interaction, a manager that handles the faucet logic, and an RPC server for client interactions.
