package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"time"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := sql.Open("postgres", "host=localhost user=username password=password dbname=db sslmode=disable")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	w := &kafka.Writer{
		Addr:                   kafka.TCP("localhost:29092"),
		Topic:                  "orders",
		AllowAutoTopicCreation: true,
	}

	timeout := 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeout):
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			continue
		}
		defer tx.Rollback()

		rows, err := tx.QueryContext(
			ctx,
			"SELECT event_id, event_type, payload FROM outbox WHERE processed_at IS NULL LIMIT 10 FOR UPDATE SKIP LOCKED",
		)
		if err != nil {
			log.Printf("Error executing query: %v", err)
			continue
		}

		var processed []string
		for rows.Next() {
			var eventId, eventType string
			var payload []byte

			if err := rows.Scan(&eventId, &eventType, &payload); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			msg := kafka.Message{
				Key:   []byte(eventId),
				Value: payload,
			}

			if err := w.WriteMessages(ctx, msg); err != nil {
				log.Printf("Error writing message to Kafka: %v", err)
				continue
			}

			log.Printf("Sent message to Kafka: %s", eventId)
			processed = append(processed, eventId)
		}

		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
			continue
		}

		for _, eventId := range processed {
			_, err = tx.ExecContext(ctx, "UPDATE outbox SET processed_at = $1 WHERE event_id = $2", time.Now(), eventId)
			if err != nil {
				log.Printf("Error updating row: %v", err)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
		}
	}
}
