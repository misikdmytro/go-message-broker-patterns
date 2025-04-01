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
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/segmentio/kafka-go"
)

type PaymentInfo struct {
	PublicOrderID string  `json:"public_order_id"`
	Price         float64 `json:"price"`
}

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
		Topic:                  "payments",
		AllowAutoTopicCreation: true,
	}

	conn, err := amqp.Dial("amqp://user:password@localhost:5672/")
	if err != nil {
		log.Fatalf("Error connecting to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error opening channel: %v", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"payments",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error declaring exchange: %v", err)
	}

	q, err := ch.QueueDeclare(
		"payments",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error declaring queue: %v", err)
	}

	err = ch.QueueBind(
		q.Name,
		"",
		"payments",
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error binding queue: %v", err)
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
		log.Fatalf("Error preparing statement: %v", err)
	}

	err = publishToKafka(ctx, w, publicOrderID, price)
	if err != nil {
		log.Printf("Error publishing to Kafka: %v", err)

		err = publishToRabbitMQ(ctx, ch, publicOrderID, price)
		if err != nil {
			log.Fatalf("Error publishing to RabbitMQ: %v", err)
		}

		log.Printf("Message sent to RabbitMQ: %s", publicOrderID)
	} else {
		log.Printf("Message sent to Kafka: %s", publicOrderID)
	}

}

func publishToKafka(ctx context.Context, w *kafka.Writer, publicOrderID string, price float64) error {
	pi := PaymentInfo{
		PublicOrderID: publicOrderID,
		Price:         price,
	}

	b, err := json.Marshal(pi)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(publicOrderID),
		Value: b,
	}

	err = w.WriteMessages(ctx, msg)
	if err != nil {
		return err
	}

	log.Printf("Message sent to Kafka: %s", string(msg.Value))
	return nil
}

func publishToRabbitMQ(ctx context.Context, ch *amqp.Channel, publicOrderID string, price float64) error {
	pi := PaymentInfo{
		PublicOrderID: publicOrderID,
		Price:         price,
	}

	b, err := json.Marshal(pi)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		ctx,
		"payments",
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return err
	}

	log.Printf("Message sent to RabbitMQ: %s", string(b))
	return nil
}
