#!/usr/bin/env bash
# Copyright (C) 2024, Nuklai. All rights reserved.
# See the file LICENSE for licensing terms.

set -e

# Ensure the database name is set
if [ -z "$POSTGRES_DBNAME" ]; then
  echo "POSTGRES_DBNAME is not set. Exiting."
  exit 1
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE "$POSTGRES_DBNAME";
EOSQL
