package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type OrderFullfillmentPayload struct {
	OrderID string `json:"order_id"`
}

func main() {
	const maxRetries = 3

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := sql.Open("sqlite3", "../queue.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	timeout := 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeout):
		}

		rows, err := db.QueryContext(
			ctx,
			"SELECT id, type, payload, retry_count, next_attempt_at WHERE next_attempt_at <= ? AND retry_count < ?",
			time.Now(),
			maxRetries,
		)
		if err != nil {
			log.Printf("Error executing query: %v", err)
			continue
		}

		for rows.Next() {
			var id int64
			var messageType string
			var payload []byte
			var retryCount int
			var nextAttemptAt time.Time

			if err := rows.Scan(&id, &messageType, &payload, &retryCount, &nextAttemptAt); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			if messageType == "order_fullfillment" {
				var orderPayload OrderFullfillmentPayload
				if err := json.Unmarshal(payload, &orderPayload); err != nil {
					log.Printf("Error unmarshalling payload: %v", err)
					scheduleLater(ctx, db, id, retryCount)

					continue
				}

				if err := fullfillOrder(ctx, orderPayload); err != nil {
					log.Printf("Error fulfilling order: %v", err)
					scheduleLater(ctx, db, id, retryCount)

					continue
				}

				markAsCompleted(ctx, db, id)
			}
		}
	}
}

func fullfillOrder(ctx context.Context, payload OrderFullfillmentPayload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	log.Printf("Fullfilled order with payload: %v", payload)
	return nil
}

func scheduleLater(ctx context.Context, db *sql.DB, id int64, retryCount int) {
	secondsToRetry := 10 * (retryCount + 1)
	_, err := db.ExecContext(
		ctx,
		"UPDATE outbox SET retry_count = retry_count + 1, next_attempt_at = ?, last_attempt_at = ? WHERE id = ?",
		time.Now().Add(time.Duration(secondsToRetry)*time.Second),
		time.Now(),
		id,
	)

	if err != nil {
		log.Printf("Error updating row: %v", err)
		return
	}
}

func markAsCompleted(ctx context.Context, db *sql.DB, id int64) {
	_, err := db.ExecContext(
		ctx,
		"UPDATE outbox SET next_attempt_at = NULL, last_attempt_at = ? WHERE id = ?",
		time.Now(),
		id,
	)
	if err != nil {
		log.Printf("Error deleting row: %v", err)
	}
}
