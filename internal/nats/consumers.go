package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"tiris-backend/internal/repositories"

	"github.com/nats-io/nats.go"
)

// EventConsumer manages event consumption from NATS streams
type EventConsumer struct {
	client *Client
	repos  *repositories.Repositories
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEventConsumer creates a new event consumer
func NewEventConsumer(client *Client, repos *repositories.Repositories) *EventConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventConsumer{
		client: client,
		repos:  repos,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts consuming events from all streams
func (ec *EventConsumer) Start() error {
	log.Println("Starting NATS event consumers...")

	// Start order event consumer
	if err := ec.startOrderEventConsumer(); err != nil {
		return fmt.Errorf("failed to start order event consumer: %w", err)
	}

	// Start balance event consumer
	if err := ec.startBalanceEventConsumer(); err != nil {
		return fmt.Errorf("failed to start balance event consumer: %w", err)
	}

	// Start error event consumer
	if err := ec.startErrorEventConsumer(); err != nil {
		return fmt.Errorf("failed to start error event consumer: %w", err)
	}

	// Start signal event consumer
	if err := ec.startSignalEventConsumer(); err != nil {
		return fmt.Errorf("failed to start signal event consumer: %w", err)
	}

	// Start heartbeat event consumer
	if err := ec.startHeartbeatEventConsumer(); err != nil {
		return fmt.Errorf("failed to start heartbeat event consumer: %w", err)
	}

	log.Println("All NATS event consumers started successfully")
	return nil
}

// Stop stops all event consumers
func (ec *EventConsumer) Stop() {
	log.Println("Stopping NATS event consumers...")
	ec.cancel()
}

// startOrderEventConsumer starts consuming order events
func (ec *EventConsumer) startOrderEventConsumer() error {
	// Create a durable pull consumer for order events
	consumerConfig := &nats.ConsumerConfig{
		Durable:       "order-processor",
		FilterSubject: "trading.orders.*",
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	}

	// Create or update the consumer
	_, err := ec.client.js.AddConsumer("TRADING", consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create order consumer: %w", err)
	}

	// Subscribe to the consumer
	sub, err := ec.client.js.PullSubscribe("trading.orders.*", "order-processor", nats.Bind("TRADING", "order-processor"))
	if err != nil {
		return fmt.Errorf("failed to pull subscribe to order events: %w", err)
	}

	// Start consuming messages in background
	go func() {
		for {
			select {
			case <-ec.ctx.Done():
				sub.Unsubscribe()
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err == nats.ErrTimeout {
						continue
					}
					log.Printf("Error fetching order messages: %v", err)
					continue
				}

				for _, msg := range msgs {
					if err := ec.handleOrderEvent(msg); err != nil {
						log.Printf("Error handling order event: %v", err)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	log.Printf("Subscribed to order events: trading.orders.*")
	return nil
}

// startBalanceEventConsumer starts consuming balance events
func (ec *EventConsumer) startBalanceEventConsumer() error {
	// Create a durable pull consumer for balance events
	consumerConfig := &nats.ConsumerConfig{
		Durable:       "balance-processor",
		FilterSubject: "trading.balance.*",
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	}

	// Create or update the consumer
	_, err := ec.client.js.AddConsumer("TRADING", consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create balance consumer: %w", err)
	}

	// Subscribe to the consumer
	sub, err := ec.client.js.PullSubscribe("trading.balance.*", "balance-processor", nats.Bind("TRADING", "balance-processor"))
	if err != nil {
		return fmt.Errorf("failed to pull subscribe to balance events: %w", err)
	}

	// Start consuming messages in background
	go func() {
		for {
			select {
			case <-ec.ctx.Done():
				sub.Unsubscribe()
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err == nats.ErrTimeout {
						continue
					}
					log.Printf("Error fetching balance messages: %v", err)
					continue
				}

				for _, msg := range msgs {
					if err := ec.handleBalanceEvent(msg); err != nil {
						log.Printf("Error handling balance event: %v", err)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	log.Printf("Subscribed to balance events: trading.balance.*")
	return nil
}

// startErrorEventConsumer starts consuming error events
func (ec *EventConsumer) startErrorEventConsumer() error {
	// Create a durable pull consumer for error events
	consumerConfig := &nats.ConsumerConfig{
		Durable:       "error-processor",
		FilterSubject: "trading.errors",
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	}

	// Create or update the consumer
	_, err := ec.client.js.AddConsumer("TRADING_ERRORS", consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create error consumer: %w", err)
	}

	// Subscribe to the consumer
	sub, err := ec.client.js.PullSubscribe("trading.errors", "error-processor", nats.Bind("TRADING_ERRORS", "error-processor"))
	if err != nil {
		return fmt.Errorf("failed to pull subscribe to error events: %w", err)
	}

	// Start consuming messages in background
	go func() {
		for {
			select {
			case <-ec.ctx.Done():
				sub.Unsubscribe()
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err == nats.ErrTimeout {
						continue
					}
					log.Printf("Error fetching error messages: %v", err)
					continue
				}

				for _, msg := range msgs {
					if err := ec.handleErrorEvent(msg); err != nil {
						log.Printf("Error handling error event: %v", err)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	log.Printf("Subscribed to error events: trading.errors")
	return nil
}

// startSignalEventConsumer starts consuming signal events
func (ec *EventConsumer) startSignalEventConsumer() error {
	// Create a durable pull consumer for signal events
	consumerConfig := &nats.ConsumerConfig{
		Durable:       "signal-processor",
		FilterSubject: "trading.signals",
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	}

	// Create or update the consumer
	_, err := ec.client.js.AddConsumer("TRADING", consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create signal consumer: %w", err)
	}

	// Subscribe to the consumer
	sub, err := ec.client.js.PullSubscribe("trading.signals", "signal-processor", nats.Bind("TRADING", "signal-processor"))
	if err != nil {
		return fmt.Errorf("failed to pull subscribe to signal events: %w", err)
	}

	// Start consuming messages in background
	go func() {
		for {
			select {
			case <-ec.ctx.Done():
				sub.Unsubscribe()
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err == nats.ErrTimeout {
						continue
					}
					log.Printf("Error fetching signal messages: %v", err)
					continue
				}

				for _, msg := range msgs {
					if err := ec.handleSignalEvent(msg); err != nil {
						log.Printf("Error handling signal event: %v", err)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	log.Printf("Subscribed to signal events: trading.signals")
	return nil
}

// startHeartbeatEventConsumer starts consuming heartbeat events
func (ec *EventConsumer) startHeartbeatEventConsumer() error {
	// Create a durable pull consumer for heartbeat events
	consumerConfig := &nats.ConsumerConfig{
		Durable:       "heartbeat-processor",
		FilterSubject: "system.heartbeat",
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	}

	// Create or update the consumer
	_, err := ec.client.js.AddConsumer("SYSTEM", consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat consumer: %w", err)
	}

	// Subscribe to the consumer
	sub, err := ec.client.js.PullSubscribe("system.heartbeat", "heartbeat-processor", nats.Bind("SYSTEM", "heartbeat-processor"))
	if err != nil {
		return fmt.Errorf("failed to pull subscribe to heartbeat events: %w", err)
	}

	// Start consuming messages in background
	go func() {
		for {
			select {
			case <-ec.ctx.Done():
				sub.Unsubscribe()
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
				if err != nil {
					if err == nats.ErrTimeout {
						continue
					}
					log.Printf("Error fetching heartbeat messages: %v", err)
					continue
				}

				for _, msg := range msgs {
					if err := ec.handleHeartbeatEvent(msg); err != nil {
						log.Printf("Error handling heartbeat event: %v", err)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	log.Printf("Subscribed to heartbeat events: system.heartbeat")
	return nil
}

// handleOrderEvent processes order events
func (ec *EventConsumer) handleOrderEvent(msg *nats.Msg) error {
	var event OrderEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal order event: %w", err)
	}

	// Check for duplicate event
	if exists, err := ec.isEventProcessed(event.EventID); err != nil {
		return fmt.Errorf("failed to check event deduplication: %w", err)
	} else if exists {
		log.Printf("Skipping duplicate order event: %s", event.EventID)
		return nil
	}

	// Process the order event
	log.Printf("Processing order event: %s - %s - %s", event.EventType, event.OrderID, event.Status)

	// Create trading log entry
	if err := ec.createTradingLogFromOrderEvent(&event); err != nil {
		return fmt.Errorf("failed to create trading log: %w", err)
	}

	// Mark event as processed
	if err := ec.markEventAsProcessed(event.EventID, string(event.EventType), &event.UserID, &event.SubAccountID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

// handleBalanceEvent processes balance events
func (ec *EventConsumer) handleBalanceEvent(msg *nats.Msg) error {
	var event BalanceEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal balance event: %w", err)
	}

	// Check for duplicate event
	if exists, err := ec.isEventProcessed(event.EventID); err != nil {
		return fmt.Errorf("failed to check event deduplication: %w", err)
	} else if exists {
		log.Printf("Skipping duplicate balance event: %s", event.EventID)
		return nil
	}

	// Process the balance event
	log.Printf("Processing balance event: %s - %s - %f -> %f",
		event.EventType, event.Symbol, event.PreviousBalance, event.NewBalance)

	// Update balance and create transaction
	transactionID, err := ec.repos.SubAccount.UpdateBalance(
		ec.ctx,
		event.SubAccountID,
		event.NewBalance,
		event.Amount,
		event.Direction,
		event.Reason,
		event.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Create trading log entry
	if err := ec.createTradingLogFromBalanceEvent(&event, transactionID); err != nil {
		return fmt.Errorf("failed to create trading log: %w", err)
	}

	// Mark event as processed
	if err := ec.markEventAsProcessed(event.EventID, string(event.EventType), &event.UserID, &event.SubAccountID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

// handleErrorEvent processes error events
func (ec *EventConsumer) handleErrorEvent(msg *nats.Msg) error {
	var event ErrorEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal error event: %w", err)
	}

	// Check for duplicate event
	if exists, err := ec.isEventProcessed(event.EventID); err != nil {
		return fmt.Errorf("failed to check event deduplication: %w", err)
	} else if exists {
		log.Printf("Skipping duplicate error event: %s", event.EventID)
		return nil
	}

	// Process the error event
	log.Printf("Processing error event: %s - %s - %s", event.EventType, event.Severity, event.ErrorMessage)

	// Create trading log entry for error
	if err := ec.createTradingLogFromErrorEvent(&event); err != nil {
		return fmt.Errorf("failed to create trading log: %w", err)
	}

	// Mark event as processed
	if err := ec.markEventAsProcessed(event.EventID, string(event.EventType), &event.UserID, event.SubAccountID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

// handleSignalEvent processes signal events
func (ec *EventConsumer) handleSignalEvent(msg *nats.Msg) error {
	var event SignalEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal signal event: %w", err)
	}

	// Check for duplicate event
	if exists, err := ec.isEventProcessed(event.EventID); err != nil {
		return fmt.Errorf("failed to check event deduplication: %w", err)
	} else if exists {
		log.Printf("Skipping duplicate signal event: %s", event.EventID)
		return nil
	}

	// Process the signal event
	log.Printf("Processing signal event: %s - %s - %s - %.2f confidence",
		event.EventType, event.SignalType, event.Symbol, event.Confidence)

	// Create trading log entry for signal
	if err := ec.createTradingLogFromSignalEvent(&event); err != nil {
		return fmt.Errorf("failed to create trading log: %w", err)
	}

	// Mark event as processed
	if err := ec.markEventAsProcessed(event.EventID, string(event.EventType), &event.UserID, event.SubAccountID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

// handleHeartbeatEvent processes heartbeat events
func (ec *EventConsumer) handleHeartbeatEvent(msg *nats.Msg) error {
	var event HeartbeatEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal heartbeat event: %w", err)
	}

	// Check for duplicate event (though heartbeats are typically not critical to deduplicate)
	if exists, err := ec.isEventProcessed(event.EventID); err != nil {
		return fmt.Errorf("failed to check event deduplication: %w", err)
	} else if exists {
		return nil
	}

	// Process the heartbeat event
	log.Printf("Processing heartbeat event: %s - %s - %s", event.EventType, event.Component, event.Status)

	// Mark event as processed
	if err := ec.markEventAsProcessed(event.EventID, string(event.EventType), &event.UserID, nil); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}
