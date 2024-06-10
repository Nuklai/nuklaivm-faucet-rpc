# Nuklai Faucet

## Disclaimer

**IMPORTANT NOTICE:** This project is currently in the alpha stage of development and is not intended for use in production environments. The software may contain bugs, incomplete features, or other issues that could cause it to malfunction. Use at your own risk.

We welcome contributions and feedback to help improve this project, but please be aware that the codebase is still under active development. It is recommended to thoroughly test any changes or additions before deploying them in a production environment.

Thank you for your understanding and support!

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

## How does the faucet work?

The faucet service is designed to distribute test NAI tokens to users, primarily for testing purposes on the Nuklai blockchain. The main components of the service include the main server setup, configuration management, database interaction, a manager that handles the faucet logic, and an RPC server for client interactions.

### Workflow

1. **User Requests a Challenge**:

   - The user calls the `Challenge` method on the JSON-RPC server.
   - The server responds with the current salt and difficulty.

2. **User Solves the Challenge**:

   - The user computes a solution for the provided challenge.
   - The user submits the solution via the `SolveChallenge` method.
   - The server verifies the solution:
     - If valid, it transfers the specified amount of tokens to the user's address.
     - The transaction is saved in the PostgreSQL database.

3. **Challenge Rotation**:

   - The manager periodically rotates the salt and adjusts the difficulty based on the number of solutions.

4. **Health Check**:

   - A simple health check endpoint is available at `/health` to verify the service is running.

5. **Dynamic Configuration**:
   - An authorized admin can update the RPC URL using the `UpdateNuklaiRPC` method with the correct admin token.

This setup ensures the faucet service can handle requests efficiently, manage challenges dynamically, and provide necessary endpoints for client interactions.
