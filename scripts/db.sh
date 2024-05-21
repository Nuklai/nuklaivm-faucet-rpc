#!/usr/bin/env bash
# Copyright (C) 2024, AllianceBlock. All rights reserved.
# See the file LICENSE for licensing terms.

DB_PATH=".nuklai-faucet/db/faucet.db"

function get_transaction_by_txid() {
  local txid="$1"
  echo "Getting transaction with TxID: $txid"
  sqlite3 $DB_PATH "SELECT * FROM transactions WHERE txid='$txid';"
}

function get_all_transactions() {
  echo "Getting all transactions"
  sqlite3 $DB_PATH "SELECT * FROM transactions;"
}

function get_transactions_by_user() {
  local user_address="$1"
  echo "Getting transactions for user: $user_address"
  sqlite3 $DB_PATH "SELECT * FROM transactions WHERE destination='$user_address';"
}

function usage() {
  echo "Usage: $0 {get-transaction-by-txid|get-all-transactions|get-transactions-by-user} [args]"
}

# Ensure at least one argument is provided
if [ $# -eq 0 ]; then
  usage
  exit 1
fi

case "$1" in
  get-transaction-by-txid)
    if [ -z "$2" ]; then
      echo "TxID is required"
      usage
      exit 1
    fi
    get_transaction_by_txid "$2"
    ;;
  get-all-transactions)
    get_all_transactions
    ;;
  get-transactions-by-user)
    if [ -z "$2" ]; then
      echo "User address is required"
      usage
      exit 1
    fi
    get_transactions_by_user "$2"
    ;;
  *)
    usage
    ;;
esac
