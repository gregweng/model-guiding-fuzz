package pubsub

import (
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
)

func TestPubSubClientReceive(t *testing.T) {
	testCases := []struct {
		name      string
		ackMode   AckMode
		subConfig *SubscriptionConfig
	}{
		{
			name:    "With message acknowledgment",
			ackMode: AckModeAck,
		},
		{
			name:    "Without message acknowledgment",
			ackMode: AckModeNack,
		},
		{
			name:    "With custom subscription config",
			ackMode: AckModeAck,
			subConfig: &SubscriptionConfig{
				AckDeadline:       20 * time.Second,
				RetentionDuration: 48 * time.Hour,
				ExpirationPolicy:  72 * time.Hour,
				Filter:            "attributes.type = 'test'",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test client
			cfg := Config{
				ProjectID:      "test-project",
				TopicID:        "test-topic",
				SubscriptionID: "test-sub",
				AckMode:        tc.ackMode,
				SubConfig:      tc.subConfig,
			}
			client, err := NewPubSubClient(cfg)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			// Test buffering and receiving messages
			testMsg := &pubsub.Message{
				Data: []byte("test message"),
				Attributes: map[string]string{
					"key": "value",
				},
			}
			client.BufferMessage(testMsg)

			// Try receiving the message
			msg, err := client.ReceiveMessage(time.Second)
			if err != nil {
				t.Fatalf("Failed to receive message: %v", err)
			}

			// Verify message contents
			if string(msg.Data) != "test message" {
				t.Errorf("Expected message data 'test message', got '%s'", string(msg.Data))
			}
			if msg.Attributes["key"] != "value" {
				t.Errorf("Expected attribute value 'value', got '%s'", msg.Attributes["key"])
			}
		})
	}
}

func TestPubSubClientPublish(t *testing.T) {
	// Create test client
	cfg := Config{
		ProjectID:      "test-project",
		TopicID:        "test-topic",
		SubscriptionID: "test-sub",
	}
	client, err := NewPubSubClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test publishing message with timeout
	data := []byte("test publish message")
	attrs := map[string]string{"test": "publish"}

	// Test with no timeout
	msgID, err := client.PublishMessage(data, attrs, 0)
	if err != nil {
		t.Fatalf("Failed to publish message without timeout: %v", err)
	}
	if msgID == "" {
		t.Error("Expected non-empty message ID")
	}

	// Test with reasonable timeout
	msgID, err = client.PublishMessage(data, attrs, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to publish message with timeout: %v", err)
	}
	if msgID == "" {
		t.Error("Expected non-empty message ID")
	}

	// Test with very short timeout
	_, err = client.PublishMessage(data, attrs, 1*time.Nanosecond)
	if err == nil {
		t.Error("Expected timeout error for very short timeout")
	}
}

func TestPubSubClientTimeout(t *testing.T) {
	if os.Getenv("PUBSUB_EMULATOR_HOST") != "" {
		t.Skip("Skipping timeout test when using emulator")
	}

	// Create test client
	cfg := Config{
		ProjectID:      "test-project",
		TopicID:        "test-topic",
		SubscriptionID: "test-sub",
	}
	client, err := NewPubSubClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test receive timeout
	msg, err := client.ReceiveMessage(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if msg != nil {
		t.Error("Expected nil message on timeout")
	}
}

func TestPubSubClientErrors(t *testing.T) {
	if os.Getenv("PUBSUB_EMULATOR_HOST") != "" {
		t.Skip("Skipping credential test when using emulator")
	}

	// Test invalid credentials
	cfg := Config{
		ProjectID:      "test-project",
		TopicID:        "test-topic",
		SubscriptionID: "test-sub",
		Credentials:    "invalid-path.json",
	}
	_, err := NewPubSubClient(cfg)
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}

	// Test with nil subscription config
	cfg = Config{
		ProjectID:      "test-project",
		TopicID:        "test-topic",
		SubscriptionID: "test-sub",
	}
	client, err := NewPubSubClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test publishing with nil attributes
	_, err = client.PublishMessage([]byte("test"), nil, 0)
	if err != nil {
		t.Errorf("Expected success with nil attributes, got error: %v", err)
	}

	// Test publishing with negative timeout
	_, err = client.PublishMessage([]byte("test"), nil, -1*time.Second)
	if err != nil {
		t.Errorf("Expected success with negative timeout (treated as no timeout), got error: %v", err)
	}
}
