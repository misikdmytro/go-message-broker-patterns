package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func ensureSchema(ctx context.Context, db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS message_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		payload BLOB NOT NULL,
		retry_count INTEGER DEFAULT 0,
		last_attempt_at TIMESTAMP,
		next_attempt_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_next_attempt_at ON message_queue(next_attempt_at);
	`
	_, err := db.ExecContext(ctx, schema)
	return err
}

type OrderFullfillmentPayload struct {
	OrderID string `json:"order_id"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := sql.Open("postgres", "host=localhost user=username password=password dbname=db sslmode=disable")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	queueDB, err := sql.Open("sqlite3", "../queue.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer queueDB.Close()

	if err := ensureSchema(ctx, queueDB); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	publicOrderID := uuid.NewString()
	createdAt := time.Now()
	price := 42.42

	_, err = db.ExecContext(
		ctx,
		"INSERT INTO orders(public_id, price, created_at) VALUES ($1, $2, $3)",
		publicOrderID,
		price,
		createdAt,
	)
	if err != nil {
		log.Fatalf("Error inserting order: %v", err)
	}

	fullfillment := OrderFullfillmentPayload{
		OrderID: publicOrderID,
	}

	b, err := json.Marshal(fullfillment)
	if err != nil {
		log.Fatalf("Error marshalling payload: %v", err)
	}

	_, err = queueDB.ExecContext(
		ctx,
		"INSERT INTO message_queue (type, payload, next_attempt_at, created_at) VALUES (?, ?, ?, ?)",
		"order_fullfillment",
		b,
		createdAt,
		createdAt,
	)
	if err != nil {
		log.Fatalf("Error inserting message into queue: %v", err)
	}

	log.Printf("Inserted message into queue: %s", publicOrderID)
}
