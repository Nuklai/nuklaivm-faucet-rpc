#!/usr/bin/env bash
# Copyright (C) 2024, Nuklai. All rights reserved.
# See the file LICENSE for licensing terms.

# Check if .env file exists
if [ ! -f .env ]; then
  echo ".env file not found!"
  exit 1
fi

# Source the .env file to load environment variables
source .env

# Read the .env file and construct the --env options for docker run
env_vars=$(grep -v '^#' .env | xargs -I {} echo --env {} | xargs)

# Function to create a custom Docker network
function create_network() {
  echo "Creating custom Docker network..."
  docker network create nuklai-faucet-network || true
}

# Function to start the PostgreSQL container
function start_postgres() {
  echo "Starting PostgreSQL container..."

  # Remove any existing data volume to ensure clean initialization
  docker volume rm postgres_data_faucet || true

  # Run the PostgreSQL container with the constructed --env options
  docker run -d --name nuklai-faucet-postgres --network nuklai-faucet-network \
      --env POSTGRES_USER=${POSTGRES_USER} \
      --env POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
      --env POSTGRES_DBNAME=${POSTGRES_DBNAME} \
      -p ${POSTGRES_PORT:-5432}:5432 \
      -v postgres_data_faucet:/var/lib/postgresql/data \
      -v $(pwd)/docker-entrypoint-initdb.d:/docker-entrypoint-initdb.d \
      postgres:13

  echo "Waiting for PostgreSQL to become healthy..."
  until docker exec nuklai-faucet-postgres pg_isready -U $POSTGRES_USER -d $POSTGRES_DBNAME; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 1
  done

  echo "PostgreSQL is up and running"
}

# Function to start the Faucet container
function start_faucet() {
  echo "Starting Faucet container..."

# Ensure PostgreSQL is fully ready before starting Faucet
  echo "Checking if PostgreSQL is ready..."
  until docker exec nuklai-faucet-postgres pg_isready -U $POSTGRES_USER -d $POSTGRES_DBNAME; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 1
  done

  # Run the Faucet container with the constructed --env options
  docker run -d -p 10591:10591 --name nuklai-faucet --network nuklai-faucet-network $env_vars nuklai-faucet

  echo "Faucet container started"
}

# Function to stop and remove the containers
function stop_services() {
  echo "Stopping Faucet container..."
  docker stop nuklai-faucet || true
  docker rm nuklai-faucet || true

  echo "Stopping PostgreSQL container..."
  docker stop nuklai-faucet-postgres || true
  docker rm nuklai-faucet-postgres || true

  echo "Removing custom network..."
  docker network rm nuklai-faucet-network || true
}

case "$1" in
  start)
    stop_services
    create_network
    start_postgres
    start_faucet
    ;;
  stop)
    stop_services
    ;;
  logs)
    docker logs -f nuklai-faucet
    ;;
  *)
    echo "Usage: $0 {start|stop|logs}"
    ;;
esac
