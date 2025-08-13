package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID       uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Username string         `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Email    string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Avatar   *string        `gorm:"type:text" json:"avatar,omitempty"`
	Settings datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"settings"`
	Info     datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	CreatedAt time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	OAuthTokens  []OAuthToken  `json:"-"`
	Exchanges    []Exchange    `json:"-"`
	SubAccounts  []SubAccount  `json:"-"`
	Transactions []Transaction `json:"-"`
	TradingLogs  []TradingLog  `json:"-"`
}

// OAuthToken represents OAuth authentication tokens
type OAuthToken struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Provider       string         `gorm:"type:varchar(20);not null;index" json:"provider"`
	ProviderUserID string         `gorm:"type:varchar(255);not null" json:"provider_user_id"`
	AccessToken    string         `gorm:"type:text;not null" json:"-"`
	RefreshToken   *string        `gorm:"type:text" json:"-"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
	Info           datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	CreatedAt time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// Exchange represents a trading exchange connection
type Exchange struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Type      string         `gorm:"type:varchar(50);not null;index" json:"type"`
	APIKey    string         `gorm:"type:text;not null" json:"-"`
	APISecret string         `gorm:"type:text;not null" json:"-"`
	Status    string         `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Info      datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	CreatedAt time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User        User         `gorm:"foreignKey:UserID" json:"-"`
	SubAccounts []SubAccount `json:"-"`
}

// SubAccount represents a trading sub-account
type SubAccount struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	ExchangeID uuid.UUID      `gorm:"type:uuid;not null;index" json:"exchange_id"`
	Name       string         `gorm:"type:varchar(100);not null" json:"name"`
	Symbol     string         `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Balance    float64        `gorm:"type:decimal(20,8);default:0" json:"balance"`
	Info       datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	CreatedAt time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User         User          `gorm:"foreignKey:UserID" json:"-"`
	Exchange     Exchange      `gorm:"foreignKey:ExchangeID" json:"-"`
	Transactions []Transaction `json:"-"`
	TradingLogs  []TradingLog  `json:"-"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index:idx_transactions_user_timestamp" json:"user_id"`
	ExchangeID     uuid.UUID      `gorm:"type:uuid;not null;index:idx_transactions_exchange_timestamp" json:"exchange_id"`
	SubAccountID   uuid.UUID      `gorm:"type:uuid;not null;index:idx_transactions_sub_account_timestamp" json:"sub_account_id"`
	Timestamp      time.Time      `gorm:"not null;default:now();index:idx_transactions_user_timestamp,sort:desc;index:idx_transactions_exchange_timestamp,sort:desc;index:idx_transactions_sub_account_timestamp,sort:desc" json:"timestamp"`
	Direction      string         `gorm:"type:varchar(10);not null;check:direction IN ('debit', 'credit');index" json:"direction"`
	Reason         string         `gorm:"type:varchar(50);not null;index" json:"reason"`
	Amount         float64        `gorm:"type:decimal(20,8);not null" json:"amount"`
	ClosingBalance float64        `gorm:"type:decimal(20,8);not null" json:"closing_balance"`
	Price          *float64       `gorm:"type:decimal(20,8)" json:"price,omitempty"`
	QuoteSymbol    *string        `gorm:"type:varchar(20)" json:"quote_symbol,omitempty"`
	Info           datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	// Relationships (no DeletedAt for time-series data)
	User       User        `gorm:"foreignKey:UserID" json:"-"`
	Exchange   Exchange    `gorm:"foreignKey:ExchangeID" json:"-"`
	SubAccount SubAccount  `gorm:"foreignKey:SubAccountID" json:"-"`
	TradingLog *TradingLog `json:"-"`
}

// TradingLog represents a trading operation log entry
type TradingLog struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index:idx_trading_logs_user_timestamp" json:"user_id"`
	ExchangeID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_trading_logs_exchange_timestamp" json:"exchange_id"`
	SubAccountID  *uuid.UUID     `gorm:"type:uuid;index:idx_trading_logs_sub_account_timestamp" json:"sub_account_id,omitempty"`
	TransactionID *uuid.UUID     `gorm:"type:uuid;index" json:"transaction_id,omitempty"`
	Timestamp     time.Time      `gorm:"not null;default:now();index:idx_trading_logs_user_timestamp,sort:desc;index:idx_trading_logs_exchange_timestamp,sort:desc;index:idx_trading_logs_sub_account_timestamp,sort:desc" json:"timestamp"`
	Type          string         `gorm:"type:varchar(50);not null;index" json:"type"`
	Source        string         `gorm:"type:varchar(20);not null;check:source IN ('manual', 'bot');index" json:"source"`
	Message       string         `gorm:"type:text;not null" json:"message"`
	Info          datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	// Relationships (no DeletedAt for time-series data)
	User        User         `gorm:"foreignKey:UserID" json:"-"`
	Exchange    Exchange     `gorm:"foreignKey:ExchangeID" json:"-"`
	SubAccount  *SubAccount  `gorm:"foreignKey:SubAccountID" json:"-"`
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"-"`
}

// EventProcessing represents NATS event processing tracking
type EventProcessing struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	EventID      string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"event_id"`
	EventType    string         `gorm:"type:varchar(100);not null;index" json:"event_type"`
	UserID       *uuid.UUID     `gorm:"type:uuid;index" json:"user_id,omitempty"`
	SubAccountID *uuid.UUID     `gorm:"type:uuid;index" json:"sub_account_id,omitempty"`
	ProcessedAt  time.Time      `gorm:"default:now();index" json:"processed_at"`
	Status       string         `gorm:"type:varchar(20);default:'processed';index" json:"status"`
	RetryCount   int            `gorm:"default:0" json:"retry_count"`
	ErrorMessage *string        `gorm:"type:text" json:"error_message,omitempty"`
	Info         datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"info"`

	// Relationships
	User       *User       `gorm:"foreignKey:UserID" json:"-"`
	SubAccount *SubAccount `gorm:"foreignKey:SubAccountID" json:"-"`
}

// TableName overrides for TimescaleDB hypertables
func (Transaction) TableName() string {
	return "transactions"
}

func (TradingLog) TableName() string {
	return "trading_logs"
}
