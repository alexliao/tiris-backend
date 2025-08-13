package nats

import (
	"context"
	"fmt"
	"log"
	"time"

	"tiris-backend/internal/config"

	"github.com/nats-io/nats.go"
)

// Client represents a NATS JetStream client
type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
	cfg  config.NATSConfig
}

// NewClient creates a new NATS JetStream client
func NewClient(cfg config.NATSConfig) (*Client, error) {
	// Connect to NATS server
	opts := []nats.Option{
		nats.Name(cfg.ClientID),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Reconnect indefinitely
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			} else {
				log.Println("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %v", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &Client{
		conn: conn,
		js:   js,
		cfg:  cfg,
	}

	// Initialize streams
	if err := client.initializeStreams(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	log.Println("NATS JetStream client initialized successfully")
	return client, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		log.Println("NATS connection closed")
	}
}

// Publish publishes a message to a subject
func (c *Client) Publish(subject string, data []byte) error {
	_, err := c.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message to %s: %w", subject, err)
	}
	return nil
}

// PublishAsync publishes a message asynchronously
func (c *Client) PublishAsync(subject string, data []byte) (nats.PubAckFuture, error) {
	ack, err := c.js.PublishAsync(subject, data)
	if err != nil {
		return nil, fmt.Errorf("failed to publish async message to %s: %w", subject, err)
	}
	return ack, nil
}

// Subscribe creates a subscription to a subject
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.js.Subscribe(subject, handler, nats.Durable(c.cfg.DurableName))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}
	return sub, nil
}

// QueueSubscribe creates a queue subscription
func (c *Client) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.js.QueueSubscribe(subject, queue, handler, nats.Durable(c.cfg.DurableName))
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to %s: %w", subject, err)
	}
	return sub, nil
}

// GetStreamInfo returns information about a stream
func (c *Client) GetStreamInfo(streamName string) (*nats.StreamInfo, error) {
	info, err := c.js.StreamInfo(streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info for %s: %w", streamName, err)
	}
	return info, nil
}

// GetConsumerInfo returns information about a consumer
func (c *Client) GetConsumerInfo(streamName, consumerName string) (*nats.ConsumerInfo, error) {
	info, err := c.js.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info for %s/%s: %w", streamName, consumerName, err)
	}
	return info, nil
}

// HealthCheck checks the health of the NATS connection
func (c *Client) HealthCheck() error {
	if c.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	if !c.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not connected")
	}

	// Test JetStream connectivity by getting account info
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.js.AccountInfo(nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("NATS JetStream health check failed: %w", err)
	}

	return nil
}

// initializeStreams creates the required streams for the application
func (c *Client) initializeStreams() error {
	streams := []StreamConfig{
		{
			Name:        "TRADING",
			Description: "Trading events stream",
			Subjects:    []string{"trading.orders.*", "trading.balance.*", "trading.signals.*"},
			MaxAge:      24 * time.Hour * 30, // 30 days
			Storage:     nats.FileStorage,
			Replicas:    1,
		},
		{
			Name:        "TRADING_ERRORS",
			Description: "Trading error events stream",
			Subjects:    []string{"trading.errors"},
			MaxAge:      24 * time.Hour * 7, // 7 days
			Storage:     nats.FileStorage,
			Replicas:    1,
		},
		{
			Name:        "SYSTEM",
			Description: "System events stream",
			Subjects:    []string{"system.*"},
			MaxAge:      24 * time.Hour * 7, // 7 days
			Storage:     nats.FileStorage,
			Replicas:    1,
		},
	}

	for _, streamCfg := range streams {
		if err := c.createOrUpdateStream(streamCfg); err != nil {
			return err
		}
	}

	return nil
}

// StreamConfig represents a stream configuration
type StreamConfig struct {
	Name        string
	Description string
	Subjects    []string
	MaxAge      time.Duration
	Storage     nats.StorageType
	Replicas    int
}

// createOrUpdateStream creates or updates a stream
func (c *Client) createOrUpdateStream(cfg StreamConfig) error {
	streamConfig := &nats.StreamConfig{
		Name:        cfg.Name,
		Description: cfg.Description,
		Subjects:    cfg.Subjects,
		MaxAge:      cfg.MaxAge,
		Storage:     cfg.Storage,
		Replicas:    cfg.Replicas,
	}

	// Try to get existing stream
	_, err := c.js.StreamInfo(cfg.Name)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = c.js.AddStream(streamConfig)
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", cfg.Name, err)
		}
		log.Printf("Created NATS stream: %s", cfg.Name)
	} else {
		// Stream exists, update it
		_, err = c.js.UpdateStream(streamConfig)
		if err != nil {
			return fmt.Errorf("failed to update stream %s: %w", cfg.Name, err)
		}
		log.Printf("Updated NATS stream: %s", cfg.Name)
	}

	return nil
}
