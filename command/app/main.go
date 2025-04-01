package main

import (
	"context"
	"database/sql"
	"fmt"
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

	body := fmt.Sprintf("Your order with ID %s has been successfully created. Thank you for shopping with us!", publicOrderID)

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO commands(command_type, payload, created_at) 
			VALUES ('send_email', 
				json_build_object('email', 'user@user.com', 'subject', 'Order Confirmation - Thank You for Your Purchase!', 'body', $1::text)::jsonb, 
				$2)`,
		body,
		createdAt,
	)
	if err != nil {
		log.Fatalf("Error inserting command: %v", err)
	}

	log.Printf("Order %s created successfully", publicOrderID)
}
