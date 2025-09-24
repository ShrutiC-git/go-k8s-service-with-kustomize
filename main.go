package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/streadway/amqp"
)

// Publisher holds the rabbitmq channel
type Publisher struct {
	channel *amqp.Channel
}

// Global publisher instance
var app Publisher

var rabbitHost = getEnv("RABBIT_HOST", "rabbitmq.messaging.svc.cluster.local")

func getEnv(key, default_string string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return default_string
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@" + rabbitHost + ":5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ. error:  %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	// We don't close the channel here, as it's long-lived

	// Declare the queue here once on startup
	_, err = ch.QueueDeclare(
		"checkout.events", // queue name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	app.channel = ch
	log.Println("Successfully connected to RabbitMQ and declared queue.")

	http.HandleFunc("/checkout", checkoutHandler)
	log.Println("Checkout service starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func checkoutHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("userId")
	amount := r.URL.Query().Get("amount")

	if userId == "" || amount == "" {
		http.Error(w, "Missing userId or amount", http.StatusBadRequest)
		return
	}

	event := map[string]interface{}{
		"userId": userId,
		"amount": amount,
	}

	if err := app.publishEvent(event); err != nil {
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"event":  event,
	})

}

func (p *Publisher) publishEvent(event map[string]interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	err = p.channel.Publish(
		"",                // exchange
		"checkout.events", // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{ContentType: "application/json", Body: body},
	)
	if err != nil {
		return fmt.Errorf("Failed to publish a message: %v", err)
	}
	log.Printf("Published message: %s", body)
	return nil
}
