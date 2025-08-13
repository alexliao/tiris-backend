package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of trading event
type EventType string

const (
	// Order Events
	EventOrderCreated   EventType = "trading.orders.created"
	EventOrderFilled    EventType = "trading.orders.filled"
	EventOrderCancelled EventType = "trading.orders.cancelled"
	EventOrderFailed    EventType = "trading.orders.failed"

	// Balance Events
	EventBalanceUpdated  EventType = "trading.balance.updated"
	EventBalanceLocked   EventType = "trading.balance.locked"
	EventBalanceUnlocked EventType = "trading.balance.unlocked"

	// System Events
	EventSystemError     EventType = "trading.errors"
	EventSignalGenerated EventType = "trading.signals"
	EventBotHeartbeat    EventType = "system.heartbeat"
)

// BaseEvent represents the common fields for all events
type BaseEvent struct {
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type"`
	Timestamp  time.Time `json:"timestamp"`
	UserID     uuid.UUID `json:"user_id"`
	ExchangeID uuid.UUID `json:"exchange_id"`
	Source     string    `json:"source"` // "tiris-bot", "manual", etc.
	Version    string    `json:"version"`
}

// OrderEvent represents order-related events
type OrderEvent struct {
	BaseEvent
	SubAccountID uuid.UUID              `json:"sub_account_id"`
	OrderID      string                 `json:"order_id"`
	Symbol       string                 `json:"symbol"`
	Side         string                 `json:"side"` // "buy", "sell"
	Type         string                 `json:"type"` // "market", "limit", etc.
	Amount       float64                `json:"amount"`
	Price        *float64               `json:"price,omitempty"`
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BalanceEvent represents balance update events
type BalanceEvent struct {
	BaseEvent
	SubAccountID    uuid.UUID              `json:"sub_account_id"`
	Symbol          string                 `json:"symbol"`
	PreviousBalance float64                `json:"previous_balance"`
	NewBalance      float64                `json:"new_balance"`
	Amount          float64                `json:"amount"`
	Direction       string                 `json:"direction"` // "debit", "credit"
	Reason          string                 `json:"reason"`
	RelatedOrderID  *string                `json:"related_order_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorEvent represents error events
type ErrorEvent struct {
	BaseEvent
	SubAccountID *uuid.UUID             `json:"sub_account_id,omitempty"`
	ErrorCode    string                 `json:"error_code"`
	ErrorMessage string                 `json:"error_message"`
	Severity     string                 `json:"severity"` // "low", "medium", "high", "critical"
	Component    string                 `json:"component"`
	StackTrace   *string                `json:"stack_trace,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SignalEvent represents trading signal events
type SignalEvent struct {
	BaseEvent
	SubAccountID *uuid.UUID             `json:"sub_account_id,omitempty"`
	SignalType   string                 `json:"signal_type"` // "buy", "sell", "hold"
	Symbol       string                 `json:"symbol"`
	Confidence   float64                `json:"confidence"` // 0.0 to 1.0
	Price        *float64               `json:"price,omitempty"`
	Strategy     string                 `json:"strategy"`
	Reasoning    string                 `json:"reasoning"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// HeartbeatEvent represents system heartbeat events
type HeartbeatEvent struct {
	BaseEvent
	Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Component string                 `json:"component"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// NewBaseEvent creates a new base event with common fields
func NewBaseEvent(eventType EventType, userID, exchangeID uuid.UUID, source string) BaseEvent {
	return BaseEvent{
		EventID:    uuid.New().String(),
		EventType:  eventType,
		Timestamp:  time.Now().UTC(),
		UserID:     userID,
		ExchangeID: exchangeID,
		Source:     source,
		Version:    "1.0",
	}
}

// NewOrderEvent creates a new order event
func NewOrderEvent(eventType EventType, userID, exchangeID, subAccountID uuid.UUID, source string) *OrderEvent {
	return &OrderEvent{
		BaseEvent:    NewBaseEvent(eventType, userID, exchangeID, source),
		SubAccountID: subAccountID,
	}
}

// NewBalanceEvent creates a new balance event
func NewBalanceEvent(userID, exchangeID, subAccountID uuid.UUID, source string) *BalanceEvent {
	return &BalanceEvent{
		BaseEvent:    NewBaseEvent(EventBalanceUpdated, userID, exchangeID, source),
		SubAccountID: subAccountID,
	}
}

// NewErrorEvent creates a new error event
func NewErrorEvent(userID, exchangeID uuid.UUID, source string) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent: NewBaseEvent(EventSystemError, userID, exchangeID, source),
	}
}

// NewSignalEvent creates a new signal event
func NewSignalEvent(userID, exchangeID uuid.UUID, source string) *SignalEvent {
	return &SignalEvent{
		BaseEvent: NewBaseEvent(EventSignalGenerated, userID, exchangeID, source),
	}
}

// NewHeartbeatEvent creates a new heartbeat event
func NewHeartbeatEvent(userID, exchangeID uuid.UUID, source, component string) *HeartbeatEvent {
	return &HeartbeatEvent{
		BaseEvent: NewBaseEvent(EventBotHeartbeat, userID, exchangeID, source),
		Component: component,
	}
}

// MarshalEvent marshals an event to JSON
func MarshalEvent(event interface{}) ([]byte, error) {
	return json.Marshal(event)
}

// UnmarshalEvent unmarshals JSON data to the appropriate event type
func UnmarshalEvent(data []byte, eventType EventType) (interface{}, error) {
	switch eventType {
	case EventOrderCreated, EventOrderFilled, EventOrderCancelled, EventOrderFailed:
		var event OrderEvent
		err := json.Unmarshal(data, &event)
		return &event, err

	case EventBalanceUpdated, EventBalanceLocked, EventBalanceUnlocked:
		var event BalanceEvent
		err := json.Unmarshal(data, &event)
		return &event, err

	case EventSystemError:
		var event ErrorEvent
		err := json.Unmarshal(data, &event)
		return &event, err

	case EventSignalGenerated:
		var event SignalEvent
		err := json.Unmarshal(data, &event)
		return &event, err

	case EventBotHeartbeat:
		var event HeartbeatEvent
		err := json.Unmarshal(data, &event)
		return &event, err

	default:
		var event BaseEvent
		err := json.Unmarshal(data, &event)
		return &event, err
	}
}

// GetSubject returns the NATS subject for an event type
func GetSubject(eventType EventType) string {
	return string(eventType)
}

// ValidateEvent performs basic validation on an event
func ValidateEvent(event interface{}) error {
	switch e := event.(type) {
	case *OrderEvent:
		if e.EventID == "" || e.UserID == uuid.Nil || e.ExchangeID == uuid.Nil || e.SubAccountID == uuid.Nil {
			return fmt.Errorf("missing required fields in OrderEvent")
		}
	case *BalanceEvent:
		if e.EventID == "" || e.UserID == uuid.Nil || e.ExchangeID == uuid.Nil || e.SubAccountID == uuid.Nil {
			return fmt.Errorf("missing required fields in BalanceEvent")
		}
	case *ErrorEvent:
		if e.EventID == "" || e.UserID == uuid.Nil || e.ExchangeID == uuid.Nil {
			return fmt.Errorf("missing required fields in ErrorEvent")
		}
	case *SignalEvent:
		if e.EventID == "" || e.UserID == uuid.Nil || e.ExchangeID == uuid.Nil {
			return fmt.Errorf("missing required fields in SignalEvent")
		}
	case *HeartbeatEvent:
		if e.EventID == "" || e.UserID == uuid.Nil || e.ExchangeID == uuid.Nil {
			return fmt.Errorf("missing required fields in HeartbeatEvent")
		}
	default:
		return fmt.Errorf("unknown event type")
	}
	return nil
}
