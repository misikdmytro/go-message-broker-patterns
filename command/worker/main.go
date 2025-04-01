package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	_ "github.com/lib/pq"
)

type SendEmailPayload struct {
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := sql.Open("postgres", "host=localhost user=username password=password dbname=db sslmode=disable")
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

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			continue
		}
		defer tx.Rollback()

		rows, err := tx.QueryContext(
			ctx,
			"SELECT id, command_type, payload FROM commands WHERE executed_at IS NULL LIMIT 10 FOR UPDATE SKIP LOCKED",
		)
		if err != nil {
			log.Printf("Error executing query: %v", err)
			continue
		}

		var executed []int64
		for rows.Next() {
			var id int64
			var commandType string
			var payload []byte

			if err := rows.Scan(&id, &commandType, &payload); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			if commandType == "send_email" {
				var emailPayload SendEmailPayload
				if err := json.Unmarshal(payload, &emailPayload); err != nil {
					log.Printf("Error unmarshalling payload: %v", err)
					continue
				}

				if err := sendEmailCommand(ctx, emailPayload); err != nil {
					log.Printf("Error sending email command: %v", err)
					continue
				}
			} else {
				log.Printf("Unknown command type: %s", commandType)
				continue
			}

			executed = append(executed, id)
		}

		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
			continue
		}

		for _, id := range executed {
			_, err := tx.ExecContext(ctx, "UPDATE commands SET executed_at = $1 WHERE id = $2", time.Now(), id)
			if err != nil {
				log.Printf("Error updating row: %v", err)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
		}
	}
}

func sendEmailCommand(ctx context.Context, payload SendEmailPayload) error {
	// Simulate sending an email
	log.Printf("Sending email to '%s' with subject '%s'", payload.Email, payload.Subject)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
		// Simulate email sending delay
	}

	return nil
}
