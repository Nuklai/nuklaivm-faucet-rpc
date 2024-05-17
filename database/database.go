package database

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	query := `CREATE TABLE IF NOT EXISTS transactions (
		txid TEXT PRIMARY KEY,
		destination TEXT,
		amount INTEGER,
		timestamp INTEGER
	)`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &DB{conn: db}, nil
}

func (db *DB) SaveTransaction(txID, destination string, amount uint64) error {
	timestamp := time.Now().Unix()
	query := `INSERT INTO transactions (txid, destination, amount, timestamp) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, txID, destination, amount, timestamp)
	return err
}

func (db *DB) GetTransaction(txID string) (*Transaction, error) {
	var txn Transaction
	query := `SELECT txid, destination, amount, timestamp FROM transactions WHERE txid = ?`
	row := db.conn.QueryRow(query, txID)
	err := row.Scan(&txn.TxID, &txn.Destination, &txn.Amount, &txn.Timestamp)
	if err != nil {
		return nil, err
	}
	return &txn, nil
}

func (db *DB) GetAllTransactions() ([]Transaction, error) {
	var transactions []Transaction
	query := `SELECT txid, destination, amount, timestamp FROM transactions`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var txn Transaction
		if err := rows.Scan(&txn.TxID, &txn.Destination, &txn.Amount, &txn.Timestamp); err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	return transactions, nil
}

func (db *DB) Close() {
	db.conn.Close()
}
