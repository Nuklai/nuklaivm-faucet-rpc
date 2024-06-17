// Copyright (C) 2024, Nuklai. All rights reserved.
// See the file LICENSE for licensing terms.

package database

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/nuklai/nuklai-faucet/config"
)

type DB struct {
	conn *sql.DB
}

type Transaction struct {
	TxID        string `json:"txID"`
	Destination string `json:"destination"`
	Amount      uint64 `json:"amount"`
	Timestamp   int64  `json:"timestamp"`
}

func NewDB(config *config.Config) (*DB, error) {
	connStr := "host=" + config.PostgresHost +
		" port=" + strconv.Itoa(config.PostgresPort) +
		" user=" + config.PostgresUser +
		" password=" + config.PostgresPassword +
		" dbname=" + config.PostgresDBName +
		" sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		return nil, err
	}

	query := `CREATE TABLE IF NOT EXISTS transactions (
        txid TEXT PRIMARY KEY,
        destination TEXT,
        amount BIGINT,
        timestamp BIGINT
    )`
	_, err = db.Exec(query)
	if err != nil {
		log.Printf("Error creating table: %v", err)
		return nil, err
	}

	log.Println("Database initialized successfully")
	return &DB{conn: db}, nil
}

func (db *DB) SaveTransaction(txID, destination string, amount uint64) error {
	timestamp := time.Now().Unix()
	log.Printf("Saving transaction: txID=%s, destination=%s, amount=%d, timestamp=%d", txID, destination, amount, timestamp)
	query := `INSERT INTO transactions (txid, destination, amount, timestamp) VALUES ($1, $2, $3, $4)`
	_, err := db.conn.Exec(query, txID, destination, amount, timestamp)
	if err != nil {
		log.Printf("Error saving transaction: %v", err)
	}
	return err
}

func (db *DB) GetTransaction(txID string) (*Transaction, error) {
	var txn Transaction
	query := `SELECT txid, destination, amount, timestamp FROM transactions WHERE txid = $1`
	row := db.conn.QueryRow(query, txID)
	err := row.Scan(&txn.TxID, &txn.Destination, &txn.Amount, &txn.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No transaction found with txID: %s", txID)
		} else {
			log.Printf("Error fetching transaction: %v", err)
		}
		return nil, err
	}
	return &txn, nil
}

func (db *DB) GetAllTransactions() ([]Transaction, error) {
	var transactions []Transaction
	query := `SELECT txid, destination, amount, timestamp FROM transactions`
	rows, err := db.conn.Query(query)
	if err != nil {
		log.Printf("Error fetching all transactions: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var txn Transaction
		if err := rows.Scan(&txn.TxID, &txn.Destination, &txn.Amount, &txn.Timestamp); err != nil {
			log.Printf("Error scanning transaction row: %v", err)
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error in rows: %v", err)
		return nil, err
	}

	return transactions, nil
}

func (db *DB) Close() {
	log.Println("Closing database connection")
	db.conn.Close()
}
