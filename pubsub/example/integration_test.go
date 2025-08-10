package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ds-testing-user/etcd-fuzzing/pubsub"
)

func TestIntegrationWithEmulator(t *testing.T) {
	// Skip if not running with emulator
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		t.Skip("Skipping integration test: PUBSUB_EMULATOR_HOST not set")
	}

	// Test cases with different message types
	testCases := []struct {
		name       string
		data       []byte
		attributes map[string]string
	}{
		{
			name: "Simple message",
			data: []byte("Hello, Integration Test!"),
			attributes: map[string]string{
				"type": "greeting",
			},
		},
		{
			name: "Message with multiple attributes",
			data: []byte("Test message with attributes"),
			attributes: map[string]string{
				"type":     "test",
				"priority": "high",
				"source":   "integration_test",
			},
		},
		{
			name: "Empty message with attributes",
			data: []byte{},
			attributes: map[string]string{
				"type": "empty",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new client for each test case
			// Sanitize name for topic/subscription IDs
			sanitizedName := strings.ReplaceAll(strings.ToLower(tc.name), " ", "-")
			cfg := pubsub.Config{
				ProjectID:      "test-project",
				TopicID:        fmt.Sprintf("test-topic-%s", sanitizedName),
				SubscriptionID: fmt.Sprintf("test-sub-%s", sanitizedName),
				AckMode:        pubsub.AckModeAck,
				SubConfig: &pubsub.SubscriptionConfig{
					AckDeadline:       10 * time.Second,
					RetentionDuration: 24 * time.Hour,
				},
			}

			client, err := pubsub.NewPubSubClient(cfg)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			// Add a small delay to ensure the subscription is ready
			time.Sleep(1 * time.Second)

			// Publish message
			msgID, err := client.PublishMessage(tc.data, tc.attributes, 5*time.Second)
			if err != nil {
				t.Fatalf("Failed to publish message: %v", err)
			}
			if msgID == "" {
				t.Fatal("Expected non-empty message ID")
			}

			// Receive message
			msg, err := client.ReceiveMessage(5 * time.Second)
			if err != nil {
				t.Fatalf("Failed to receive message: %v", err)
			}

			// Verify message contents
			if string(msg.Data) != string(tc.data) {
				t.Errorf("Expected message data %q, got %q", string(tc.data), string(msg.Data))
			}

			// Verify all attributes are present
			for key, expectedValue := range tc.attributes {
				if actualValue, ok := msg.Attributes[key]; !ok {
					t.Errorf("Missing attribute %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("Attribute %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}

			// Verify no extra attributes
			for key := range msg.Attributes {
				if _, ok := tc.attributes[key]; !ok {
					t.Errorf("Unexpected attribute %q", key)
				}
			}
		})
	}
}

func TestIntegrationConcurrent(t *testing.T) {
	// Skip if not running with emulator
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		t.Skip("Skipping integration test: PUBSUB_EMULATOR_HOST not set")
	}

	// Number of messages to send concurrently
	const messageCount = 5 // Reduced for better debugging
	receivedMessages := make(map[string]bool)
	var mu sync.Mutex

	// Use a unique topic/subscription for this test to avoid conflicts
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	topicID := fmt.Sprintf("concurrent-test-topic-%s", testID)
	subscriptionID := fmt.Sprintf("concurrent-test-sub-%s", testID)

	// Create a single client for both publishing and receiving
	cfg := pubsub.Config{
		ProjectID:      "test-project",
		TopicID:        topicID,
		SubscriptionID: subscriptionID,
		AckMode:        pubsub.AckModeAck,
		SubConfig: &pubsub.SubscriptionConfig{
			AckDeadline:       10 * time.Second,
			RetentionDuration: 24 * time.Hour,
		},
	}
	client, err := pubsub.NewPubSubClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize the receivedMessages map
	for i := 0; i < messageCount; i++ {
		receivedMessages[string(rune('A'+i))] = false
	}

	// Add longer delay to ensure subscription is fully ready
	time.Sleep(2 * time.Second)

	// Create a channel to signal when all messages are received
	doneCh := make(chan struct{})
	var received int32 // Use atomic operations for thread safety

	// Start receiving messages in a goroutine BEFORE publishing
	go func() {
		defer close(doneCh)
		// Use ReceiveMessage in a loop with longer timeout per message
		for atomic.LoadInt32(&received) < messageCount {
			msg, err := client.ReceiveMessage(3 * time.Second)
			if err != nil {
				// Log errors but continue trying
				if !strings.Contains(err.Error(), "timeout") {
					t.Logf("Receive error (continuing): %v", err)
				}
				continue
			}
			if msg == nil {
				continue
			}

			msgContent := string(msg.Data)
			t.Logf("Received message: %s", msgContent)

			func() {
				mu.Lock()
				defer mu.Unlock()
				if _, ok := receivedMessages[msgContent]; !ok {
					t.Errorf("Received unexpected message: %s", msgContent)
				} else if receivedMessages[msgContent] {
					t.Errorf("Received duplicate message: %s", msgContent)
				} else {
					receivedMessages[msgContent] = true
					newReceived := atomic.AddInt32(&received, 1)
					t.Logf("Successfully processed message %s (%d/%d)", msgContent, newReceived, messageCount)
				}
			}()

			if atomic.LoadInt32(&received) >= messageCount {
				break
			}
		}
	}()

	// Add a small delay to ensure receiver is ready
	time.Sleep(500 * time.Millisecond)

	// Publish messages after receiver is ready
	for i := 0; i < messageCount; i++ {
		msgData := []byte(string(rune('A' + i)))
		t.Logf("Publishing message: %s", string(msgData))
		msgID, err := client.PublishMessage(msgData, map[string]string{"index": string(rune('A' + i))}, 5*time.Second)
		if err != nil {
			t.Errorf("Failed to publish message %d: %v", i, err)
		} else {
			t.Logf("Successfully published message %s with ID: %s", string(msgData), msgID)
		}
		// Small delay between publications to avoid overwhelming
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all messages to be received or timeout
	select {
	case <-doneCh:
		t.Logf("All messages processed successfully")
	case <-time.After(15 * time.Second): // Increased timeout
		finalReceived := atomic.LoadInt32(&received)
		t.Errorf("Timeout waiting for messages. Received %d/%d. Missing messages: %v",
			finalReceived, messageCount, getMissingMessages(receivedMessages))
	}

	// Verify all messages were received
	mu.Lock()
	defer mu.Unlock()
	for msg, wasReceived := range receivedMessages {
		if !wasReceived {
			t.Errorf("Message %q was not received", msg)
		}
	}
}

// Helper function to get a list of messages that haven't been received
func getMissingMessages(messages map[string]bool) []string {
	var missing []string
	for msg, received := range messages {
		if !received {
			missing = append(missing, msg)
		}
	}
	return missing
}
