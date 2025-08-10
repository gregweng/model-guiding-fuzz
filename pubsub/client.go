package pubsub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

// PubSubClient represents a client for interacting with Google Cloud PubSub
type PubSubClient struct {
	client        *pubsub.Client
	topic         *pubsub.Topic
	subscription  *pubsub.Subscription
	messageBuffer []*pubsub.Message
	bufferMutex   sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	ackMode       AckMode

	// Continuous receive state
	receiverStarted bool
	receiverMutex   sync.Mutex
	messageChan     chan *pubsub.Message
	errorChan       chan error
	receiverOnce    sync.Once
}

// AckMode defines how messages should be acknowledged
type AckMode int

const (
	// AckModeNack indicates messages should not be acknowledged (redelivered)
	AckModeNack AckMode = iota
	// AckModeAck indicates messages should be acknowledged (not redelivered)
	AckModeAck
)

// SubscriptionConfig holds configuration for the subscription
type SubscriptionConfig struct {
	// AckDeadline is the maximum time after a subscriber receives a message
	// before the subscriber should acknowledge the message. Default: 10s.
	AckDeadline time.Duration

	// RetentionDuration is the minimum duration to retain a message after it is published.
	// Default: 7 days.
	RetentionDuration time.Duration

	// ExpirationPolicy specifies the policy for subscription expiration.
	// Default: never expire.
	ExpirationPolicy time.Duration

	// Filter is a filter expression that restricts the messages delivered to
	// the subscription. Default: no filter.
	Filter string
}

// Config holds the configuration for PubSubClient
type Config struct {
	ProjectID      string
	TopicID        string
	SubscriptionID string
	Credentials    string // Path to service account JSON file
	AckMode        AckMode
	SubConfig      *SubscriptionConfig // Optional subscription configuration
}

// NewPubSubClient creates a new PubSubClient instance
func NewPubSubClient(cfg Config) (*PubSubClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var opts []option.ClientOption
	if cfg.Credentials != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.Credentials))
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID, opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub client: %v", err)
	}

	topic := client.Topic(cfg.TopicID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to check topic existence: %v", err)
	}
	if !exists {
		topic, err = client.CreateTopic(ctx, cfg.TopicID)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create topic: %v", err)
		}
	}

	sub := client.Subscription(cfg.SubscriptionID)
	exists, err = sub.Exists(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to check subscription existence: %v", err)
	}
	if !exists {
		subCfg := pubsub.SubscriptionConfig{
			Topic: topic,
		}

		// Apply custom subscription configuration if provided
		if cfg.SubConfig != nil {
			if cfg.SubConfig.AckDeadline > 0 {
				subCfg.AckDeadline = cfg.SubConfig.AckDeadline
			}
			if cfg.SubConfig.RetentionDuration > 0 {
				subCfg.RetentionDuration = cfg.SubConfig.RetentionDuration
			}
			if cfg.SubConfig.ExpirationPolicy > 0 {
				subCfg.ExpirationPolicy = cfg.SubConfig.ExpirationPolicy
			}
			if cfg.SubConfig.Filter != "" {
				subCfg.Filter = cfg.SubConfig.Filter
			}
		}

		sub, err = client.CreateSubscription(ctx, cfg.SubscriptionID, subCfg)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create subscription: %v", err)
		}
	}

	return &PubSubClient{
		client:       client,
		topic:        topic,
		subscription: sub,
		ctx:          ctx,
		cancel:       cancel,
		ackMode:      cfg.AckMode,
		messageChan:  make(chan *pubsub.Message, 100), // Buffer for messages
		errorChan:    make(chan error, 10),            // Buffer for errors
	}, nil
}

// PublishMessage publishes a message to the configured topic with an optional timeout
func (c *PubSubClient) PublishMessage(data []byte, attributes map[string]string, timeout time.Duration) (string, error) {
	msg := &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	}

	ctx := c.ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(c.ctx, timeout)
		defer cancel()
	}

	result := c.topic.Publish(ctx, msg)
	id, err := result.Get(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout publishing message: %v", err)
		}
		return "", fmt.Errorf("failed to publish message: %v", err)
	}
	return id, nil
}

// startContinuousReceiver starts a background goroutine that continuously receives messages
func (c *PubSubClient) startContinuousReceiver() {
	c.receiverOnce.Do(func() {
		c.receiverMutex.Lock()
		c.receiverStarted = true
		c.receiverMutex.Unlock()

		go func() {
			defer func() {
				c.receiverMutex.Lock()
				c.receiverStarted = false
				c.receiverMutex.Unlock()
			}()

			err := c.subscription.Receive(c.ctx, func(ctx context.Context, msg *pubsub.Message) {
				// Check if context is cancelled before sending
				select {
				case <-c.ctx.Done():
					return
				case <-ctx.Done():
					return
				default:
				}

				// Try to send message, but don't block if context is cancelled
				select {
				case c.messageChan <- msg:
					// Message queued successfully
				case <-c.ctx.Done():
					// Client context cancelled, stop trying
					return
				case <-ctx.Done():
					// Message context cancelled
					return
				default:
					// Channel is full, drop the message and nack it
					msg.Nack()
				}
			})

			// Only send error if context is not cancelled and channel is available
			if err != nil && err != context.Canceled {
				select {
				case c.errorChan <- err:
				case <-c.ctx.Done():
				default:
				}
			}
		}()
	})
}

// ReceiveMessage receives a single message from the subscription
func (c *PubSubClient) ReceiveMessage(timeout time.Duration) (*pubsub.Message, error) {
	// Check buffer first
	c.bufferMutex.Lock()
	if len(c.messageBuffer) > 0 {
		msg := c.messageBuffer[0]
		c.messageBuffer = c.messageBuffer[1:]
		c.bufferMutex.Unlock()
		return msg, nil
	}
	c.bufferMutex.Unlock()

	// Start continuous receiver if not already started
	c.startContinuousReceiver()

	// Wait for a message with timeout
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	select {
	case msg, ok := <-c.messageChan:
		if !ok {
			return nil, fmt.Errorf("message channel closed")
		}
		if msg != nil {
			if c.ackMode == AckModeAck {
				msg.Ack()
			} else {
				msg.Nack()
			}
			return msg, nil
		}
		return nil, fmt.Errorf("received nil message")
	case err, ok := <-c.errorChan:
		if !ok {
			return nil, fmt.Errorf("error channel closed")
		}
		return nil, fmt.Errorf("receiver error: %v", err)
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("timeout waiting for message")
		}
		return nil, ctx.Err()
	}
}

// BufferMessage adds a message to the buffer for testing purposes
func (c *PubSubClient) BufferMessage(msg *pubsub.Message) {
	c.bufferMutex.Lock()
	defer c.bufferMutex.Unlock()
	c.messageBuffer = append(c.messageBuffer, msg)
}

// Close closes the PubSub client and cleans up resources
func (c *PubSubClient) Close() error {
	c.cancel()     // This will stop the continuous receiver
	c.topic.Stop() // Stop accepting new publish requests

	// Wait for the receiver to shut down gracefully
	for i := 0; i < 100; i++ { // Max 1 second wait
		c.receiverMutex.Lock()
		started := c.receiverStarted
		c.receiverMutex.Unlock()

		if !started {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close pubsub client: %v", err)
	}
	return nil
}
