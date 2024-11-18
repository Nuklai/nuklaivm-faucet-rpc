#!/usr/bin/env bash
# Copyright (C) 2024, Nuklai. All rights reserved.
# See the file LICENSE for licensing terms.

# Load environment variables from .env file
if [ ! -f .env ]; then
  echo ".env file not found!"
  exit 1
fi

# Source the .env file to load environment variables
source .env

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

# Function to build and start the Faucet service
function start_faucet_service() {
  echo "Building and starting Faucet service..."

  # Build the Faucet service
  ./scripts/build.sh

  # Start the Faucet service
  ./build/nuklai-faucet &
  faucet_pid=$!

  echo "Faucet service started with PID $faucet_pid"
}

# Function to stop services
function stop_services() {
  echo "Stopping PostgreSQL container..."
  docker stop nuklai-faucet-postgres || true
  docker rm nuklai-faucet-postgres || true

  echo "Removing custom network..."
  docker network rm nuklai-faucet-network || true

  echo "Stopping Faucet service..."
  if [ -n "$faucet_pid" ]; then
    kill $faucet_pid || true
    echo "Faucet service stopped"
  fi
}

case "$1" in
  start)
    stop_services
    create_network
    start_postgres
    start_faucet_service
    ;;
  stop)
    stop_services
    ;;
  *)
    echo "Usage: $0 {start|stop}"
    ;;
esac
