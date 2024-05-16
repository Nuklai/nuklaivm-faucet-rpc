# Nuklai Faucet

## Build & Run from Source

- Build

  ```bash
  ./scripts/build.sh
  ```

- Run

  ```bash
  ./build/nuklai-faucet ./config.json
  ```

## Build & Run with Docker

- Build

  ```bash
  docker build -t nuklai-faucet .
  ```

- Run

  ```bash
  docker container rm -f nuklai-faucet;
  docker run -d -p 10591:10591 --name nuklai-faucet nuklai-faucet;
  ```

- Read the logs

  ```bash
  docker container logs -f nuklai-faucet
  ```
