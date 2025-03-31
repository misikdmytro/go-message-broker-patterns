package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := sql.Open("postgres", "host=localhost user=username password=password dbname=db sslmode=disable")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback()

	publicOrderID := uuid.NewString()
	createdAt := time.Now()
	price := 42.42

	orderStmt, err := tx.PrepareContext(
		ctx,
		"INSERT INTO orders(public_id, price, created_at) VALUES ($1, $2, $3)",
	)
	if err != nil {
		log.Fatalf("Error preparing statement: %v", err)
	}
	defer orderStmt.Close()

	outboxStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO outbox(event_type, event_id, payload, created_at) 
			VALUES ('order_created', $1::text, json_build_object('order_id', $1::text, 'price', $2::text, 'created_at', $3::timestamptz)::jsonb, $3::timestamptz)`,
	)
	if err != nil {
		log.Fatalf("Error preparing statement: %v", err)
	}
	defer outboxStmt.Close()

	if _, err := orderStmt.ExecContext(ctx, publicOrderID, price, createdAt); err != nil {
		log.Fatalf("Error executing order statement: %v", err)
	}

	if _, err := outboxStmt.ExecContext(ctx, publicOrderID, price, createdAt); err != nil {
		log.Fatalf("Error executing outbox statement: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Error committing transaction: %v", err)
	}

	log.Println("Transaction committed successfully")
}
