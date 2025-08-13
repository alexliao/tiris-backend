package nats

import (
	"fmt"
	"log"

	"tiris-backend/internal/config"
	"tiris-backend/internal/repositories"
)

// Manager manages NATS client and event consumers
type Manager struct {
	client   *Client
	consumer *EventConsumer
	cfg      config.NATSConfig
}

// NewManager creates a new NATS manager
func NewManager(cfg config.NATSConfig, repos *repositories.Repositories) (*Manager, error) {
	// Create NATS client
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	// Create event consumer
	consumer := NewEventConsumer(client, repos)

	return &Manager{
		client:   client,
		consumer: consumer,
		cfg:      cfg,
	}, nil
}

// Start starts the NATS manager and begins consuming events
func (m *Manager) Start() error {
	log.Println("Starting NATS manager...")

	// Start consuming events
	if err := m.consumer.Start(); err != nil {
		return fmt.Errorf("failed to start event consumer: %w", err)
	}

	log.Println("NATS manager started successfully")
	return nil
}

// Stop stops the NATS manager and closes connections
func (m *Manager) Stop() {
	log.Println("Stopping NATS manager...")

	if m.consumer != nil {
		m.consumer.Stop()
	}

	if m.client != nil {
		m.client.Close()
	}

	log.Println("NATS manager stopped")
}

// GetClient returns the NATS client for publishing events
func (m *Manager) GetClient() *Client {
	return m.client
}

// HealthCheck performs a health check on the NATS connection
func (m *Manager) HealthCheck() error {
	if m.client == nil {
		return fmt.Errorf("NATS client is not initialized")
	}
	return m.client.HealthCheck()
}

// IsConnected returns true if the NATS connection is active
func (m *Manager) IsConnected() bool {
	if m.client == nil || m.client.conn == nil {
		return false
	}
	return m.client.conn.IsConnected()
}

// GetConnectionStats returns detailed connection statistics
func (m *Manager) GetConnectionStats() map[string]interface{} {
	if m.client == nil || m.client.conn == nil {
		return nil
	}

	stats := m.client.conn.Stats()
	return map[string]interface{}{
		"connected_url":     m.client.conn.ConnectedUrl(),
		"client_id":         m.client.conn.Opts.Name,
		"discovered_urls":   m.client.conn.DiscoveredServers(),
		"bytes_sent":        stats.OutBytes,
		"bytes_received":    stats.InBytes,
		"messages_sent":     stats.OutMsgs,
		"messages_received": stats.InMsgs,
		"reconnects":        stats.Reconnects,
		"last_error":        func() string {
			if err := m.client.conn.LastError(); err != nil {
				return err.Error()
			}
			return ""
		}(),
	}
}

// PublishEvent publishes an event to the appropriate stream
func (m *Manager) PublishEvent(event interface{}) error {
	if m.client == nil {
		return fmt.Errorf("NATS client is not initialized")
	}

	// Validate the event
	if err := ValidateEvent(event); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Marshal the event
	data, err := MarshalEvent(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Determine the subject based on event type
	var subject string
	switch e := event.(type) {
	case *OrderEvent:
		subject = GetSubject(e.EventType)
	case *BalanceEvent:
		subject = GetSubject(e.EventType)
	case *ErrorEvent:
		subject = GetSubject(e.EventType)
	case *SignalEvent:
		subject = GetSubject(e.EventType)
	case *HeartbeatEvent:
		subject = GetSubject(e.EventType)
	default:
		return fmt.Errorf("unknown event type")
	}

	// Publish the event
	return m.client.Publish(subject, data)
}

// PublishEventAsync publishes an event asynchronously
func (m *Manager) PublishEventAsync(event interface{}) error {
	if m.client == nil {
		return fmt.Errorf("NATS client is not initialized")
	}

	// Validate the event
	if err := ValidateEvent(event); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Marshal the event
	data, err := MarshalEvent(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Determine the subject based on event type
	var subject string
	switch e := event.(type) {
	case *OrderEvent:
		subject = GetSubject(e.EventType)
	case *BalanceEvent:
		subject = GetSubject(e.EventType)
	case *ErrorEvent:
		subject = GetSubject(e.EventType)
	case *SignalEvent:
		subject = GetSubject(e.EventType)
	case *HeartbeatEvent:
		subject = GetSubject(e.EventType)
	default:
		return fmt.Errorf("unknown event type")
	}

	// Publish the event asynchronously
	_, err = m.client.PublishAsync(subject, data)
	return err
}

// GetStreamInfo returns information about a stream
func (m *Manager) GetStreamInfo(streamName string) (interface{}, error) {
	if m.client == nil {
		return nil, fmt.Errorf("NATS client is not initialized")
	}
	return m.client.GetStreamInfo(streamName)
}

// GetConsumerInfo returns information about a consumer
func (m *Manager) GetConsumerInfo(streamName, consumerName string) (interface{}, error) {
	if m.client == nil {
		return nil, fmt.Errorf("NATS client is not initialized")
	}
	return m.client.GetConsumerInfo(streamName, consumerName)
}
