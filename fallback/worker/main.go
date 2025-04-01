package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

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

	go func() {
		<-ctx.Done()
		os.Exit(0)
	}()

	kafkaAvailable := true
	rabbitAvailable := true

	for {
		if kafkaAvailable {
			err := consumeKafkaMessages(ctx)
			if err != nil {
				log.Printf("Kafka consumption failed: %v. Falling back to RabbitMQ...", err)
				kafkaAvailable = false

				go func() {
					// Retry Kafka connection after 5 seconds
					time.Sleep(5 * time.Second)
					kafkaAvailable = true
				}()
			}
		}

		if !kafkaAvailable && rabbitAvailable {
			err := consumeRabbitMQMessages(ctx)
			if err != nil {
				log.Printf("RabbitMQ consumption failed: %v. Falling back to Kafka...", err)
				rabbitAvailable = false

				go func() {
					// Retry RabbitMQ connection after 5 seconds
					time.Sleep(5 * time.Second)
					rabbitAvailable = true
				}()
			}
		}

		if !kafkaAvailable && !rabbitAvailable {
			log.Println("Both Kafka and RabbitMQ are unavailable. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}
}

func consumeKafkaMessages(ctx context.Context) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:29092"},
		Topic:   "payments",
		GroupID: "payments-consumer",
	})
	defer r.Close()

	msg, err := r.ReadMessage(ctx)
	if err != nil {
		log.Printf("Error reading Kafka message: %v", err)
		return err
	}

	log.Printf("Received Kafka message: %s", string(msg.Value))
	var pi PaymentInfo
	if err := json.Unmarshal(msg.Value, &pi); err != nil {
		log.Printf("Error unmarshalling Kafka message: %v", err)
		return err
	}
	if err := makePayment(ctx, pi); err != nil {
		log.Printf("Error making payment: %v", err)
		return err
	}
	log.Printf("Payment processed for order %s", pi.PublicOrderID)

	return nil
}

func consumeRabbitMQMessages(ctx context.Context) error {
	conn, err := amqp.Dial("amqp://user:password@localhost:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
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
		return err
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
		return err
	}

	err = ch.QueueBind(
		q.Name,
		"",
		"payments",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		log.Println("Shutting down RabbitMQ consumer...")
		return nil
	case msg := <-msgs:
		log.Printf("Received RabbitMQ message: %s", msg.Body)
		var pi PaymentInfo
		if err := json.Unmarshal(msg.Body, &pi); err != nil {
			log.Printf("Error unmarshalling RabbitMQ message: %v", err)
			msg.Nack(false, false)
			return err
		}
		if err := makePayment(ctx, pi); err != nil {
			log.Printf("Error making payment: %v", err)
			msg.Nack(false, false)
			return err
		}
		msg.Ack(false)
		log.Printf("Payment processed for order %s", pi.PublicOrderID)
	}

	return nil
}

func makePayment(ctx context.Context, pi PaymentInfo) error {
	// Simulate making a payment
	log.Printf("Making payment for order %s with price %.2f", pi.PublicOrderID, pi.Price)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		// Simulate payment processing
	}

	return nil
}
