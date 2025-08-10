package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ds-testing-user/etcd-fuzzing/pubsub"
)

func main() {
	// Ensure the emulator host is set
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		log.Fatal("Please set PUBSUB_EMULATOR_HOST environment variable")
	}

	// Create a new PubSub client
	cfg := pubsub.Config{
		ProjectID:      "test-project",
		TopicID:        "test-topic",
		SubscriptionID: "test-sub",
		AckMode:        pubsub.AckModeAck, // Messages will be acknowledged when received
		SubConfig: &pubsub.SubscriptionConfig{
			AckDeadline:       10 * time.Second,
			RetentionDuration: 24 * time.Hour,
		},
	}

	client, err := pubsub.NewPubSubClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Publish a message
	msgData := []byte("Hello, PubSub Emulator!")
	attrs := map[string]string{
		"sender": "example",
		"time":   time.Now().Format(time.RFC3339),
	}

	msgID, err := client.PublishMessage(msgData, attrs, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}
	fmt.Printf("Published message with ID: %s\n", msgID)

	// Receive the message
	msg, err := client.ReceiveMessage(5 * time.Second)
	if err != nil {
		log.Fatalf("Failed to receive message: %v", err)
	}

	fmt.Printf("Received message: %s\n", string(msg.Data))
	fmt.Printf("Message attributes: %v\n", msg.Attributes)
}
