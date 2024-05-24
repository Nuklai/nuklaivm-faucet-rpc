package database

import (
	"database/sql"
	"log"
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
	log.Printf("Opening database at path: %s", path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Printf("Error opening database: %v", err)
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
		log.Printf("Error creating table: %v", err)
		return nil, err
	}

	log.Println("Database initialized successfully")
	return &DB{conn: db}, nil
}

func (db *DB) SaveTransaction(txID, destination string, amount uint64) error {
	timestamp := time.Now().Unix()
	log.Printf("Saving transaction: txID=%s, destination=%s, amount=%d, timestamp=%d", txID, destination, amount, timestamp)
	query := `INSERT INTO transactions (txid, destination, amount, timestamp) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, txID, destination, amount, timestamp)
	if err != nil {
		log.Printf("Error saving transaction: %v", err)
	}
	return err
}

func (db *DB) GetTransaction(txID string) (*Transaction, error) {
	var txn Transaction
	query := `SELECT txid, destination, amount, timestamp FROM transactions WHERE txid = ?`
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
