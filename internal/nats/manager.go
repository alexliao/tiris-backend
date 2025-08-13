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
