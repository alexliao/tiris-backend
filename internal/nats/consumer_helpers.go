package nats

import (
	"fmt"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
)

// isEventProcessed checks if an event has already been processed
func (ec *EventConsumer) isEventProcessed(eventID string) (bool, error) {
	event, err := ec.repos.EventProcessing.GetByEventID(ec.ctx, eventID)
	if err != nil {
		return false, err
	}
	return event != nil, nil
}

// markEventAsProcessed marks an event as successfully processed
func (ec *EventConsumer) markEventAsProcessed(eventID, eventType string, userID *uuid.UUID, subAccountID *uuid.UUID) error {
	event := &models.EventProcessing{
		EventID:      eventID,
		EventType:    eventType,
		UserID:       userID,
		SubAccountID: subAccountID,
		Status:       "processed",
		ProcessedAt:  time.Now(),
	}
	return ec.repos.EventProcessing.Create(ec.ctx, event)
}

// createTradingLogFromOrderEvent creates a trading log entry from an order event
func (ec *EventConsumer) createTradingLogFromOrderEvent(event *OrderEvent) error {
	metadataMap := map[string]interface{}{
		"order_id":          event.OrderID,
		"symbol":            event.Symbol,
		"side":              event.Side,
		"type":              event.Type,
		"amount":            event.Amount,
		"price":             event.Price,
		"status":            event.Status,
		"event_id":          event.EventID,
		"original_metadata": event.Metadata,
	}

	log := &models.TradingLog{
		UserID:       event.UserID,
		ExchangeID:   event.ExchangeID,
		SubAccountID: &event.SubAccountID,
		Timestamp:    event.Timestamp,
		Type:         fmt.Sprintf("order_%s", getOrderAction(event.EventType)),
		Source:       "bot",
		Message:      event.Message,
		Info:         models.JSON(metadataMap),
	}

	return ec.repos.TradingLog.Create(ec.ctx, log)
}

// createTradingLogFromBalanceEvent creates a trading log entry from a balance event
func (ec *EventConsumer) createTradingLogFromBalanceEvent(event *BalanceEvent, transactionID *uuid.UUID) error {
	metadataMap := map[string]interface{}{
		"symbol":            event.Symbol,
		"previous_balance":  event.PreviousBalance,
		"new_balance":       event.NewBalance,
		"amount":            event.Amount,
		"direction":         event.Direction,
		"reason":            event.Reason,
		"related_order_id":  event.RelatedOrderID,
		"event_id":          event.EventID,
		"original_metadata": event.Metadata,
	}

	message := fmt.Sprintf("Balance updated: %s %f %s (was %f, now %f)",
		event.Direction, event.Amount, event.Symbol, event.PreviousBalance, event.NewBalance)

	log := &models.TradingLog{
		UserID:        event.UserID,
		ExchangeID:    event.ExchangeID,
		SubAccountID:  &event.SubAccountID,
		TransactionID: transactionID,
		Timestamp:     event.Timestamp,
		Type:          "balance_update",
		Source:        "bot",
		Message:       message,
		Info:          models.JSON(metadataMap),
	}

	return ec.repos.TradingLog.Create(ec.ctx, log)
}

// createTradingLogFromErrorEvent creates a trading log entry from an error event
func (ec *EventConsumer) createTradingLogFromErrorEvent(event *ErrorEvent) error {
	metadataMap := map[string]interface{}{
		"error_code":        event.ErrorCode,
		"severity":          event.Severity,
		"component":         event.Component,
		"stack_trace":       event.StackTrace,
		"event_id":          event.EventID,
		"original_metadata": event.Metadata,
	}

	message := fmt.Sprintf("[%s] %s: %s", event.Severity, event.Component, event.ErrorMessage)

	log := &models.TradingLog{
		UserID:       event.UserID,
		ExchangeID:   event.ExchangeID,
		SubAccountID: event.SubAccountID,
		Timestamp:    event.Timestamp,
		Type:         "system_error",
		Source:       "bot",
		Message:      message,
		Info:         models.JSON(metadataMap),
	}

	return ec.repos.TradingLog.Create(ec.ctx, log)
}

// createTradingLogFromSignalEvent creates a trading log entry from a signal event
func (ec *EventConsumer) createTradingLogFromSignalEvent(event *SignalEvent) error {
	metadataMap := map[string]interface{}{
		"signal_type":       event.SignalType,
		"symbol":            event.Symbol,
		"confidence":        event.Confidence,
		"price":             event.Price,
		"strategy":          event.Strategy,
		"reasoning":         event.Reasoning,
		"event_id":          event.EventID,
		"original_metadata": event.Metadata,
	}

	message := fmt.Sprintf("Trading signal: %s %s (%.1f%% confidence) - %s",
		event.SignalType, event.Symbol, event.Confidence*100, event.Reasoning)

	log := &models.TradingLog{
		UserID:       event.UserID,
		ExchangeID:   event.ExchangeID,
		SubAccountID: event.SubAccountID,
		Timestamp:    event.Timestamp,
		Type:         "trading_signal",
		Source:       "bot",
		Message:      message,
		Info:         models.JSON(metadataMap),
	}

	return ec.repos.TradingLog.Create(ec.ctx, log)
}

// getOrderAction extracts the action from order event type
func getOrderAction(eventType EventType) string {
	switch eventType {
	case EventOrderCreated:
		return "created"
	case EventOrderFilled:
		return "filled"
	case EventOrderCancelled:
		return "cancelled"
	case EventOrderFailed:
		return "failed"
	default:
		return "unknown"
	}
}
