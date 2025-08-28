package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSON is a custom type for handling JSON data in PostgreSQL
type JSON map[string]interface{}

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan non-[]byte into JSON")
	}
	
	if len(bytes) == 0 {
		*j = make(JSON)
		return nil
	}
	
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(make(JSON))
	}
	return json.Marshal(j)
}

// User represents a user in the system
type User struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Email    string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Avatar   *string   `gorm:"type:text" json:"avatar,omitempty"`
	Settings JSON      `gorm:"type:jsonb" json:"settings"`
	Info     JSON      `gorm:"type:jsonb" json:"info"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	OAuthTokens  []OAuthToken  `json:"-"`
	Tradings     []Trading     `json:"-"`
	SubAccounts  []SubAccount  `json:"-"`
	Transactions []Transaction `json:"-"`
	TradingLogs  []TradingLog  `json:"-"`
}

// OAuthToken represents OAuth authentication tokens
type OAuthToken struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Provider       string     `gorm:"type:varchar(20);not null;index" json:"provider"`
	ProviderUserID string     `gorm:"type:varchar(255);not null" json:"provider_user_id"`
	AccessToken    string     `gorm:"type:text;not null" json:"-"`
	RefreshToken   *string    `gorm:"type:text" json:"-"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Info           JSON       `gorm:"type:jsonb" json:"info"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// Trading represents a trading connection
type Trading struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Type      string    `gorm:"type:varchar(50);not null;index" json:"type"`
	APIKey    string    `gorm:"type:text;not null" json:"-"`
	APISecret string    `gorm:"type:text;not null" json:"-"`
	Status    string    `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Info      JSON      `gorm:"type:jsonb" json:"info"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User        User         `gorm:"foreignKey:UserID" json:"-"`
	SubAccounts []SubAccount `json:"-"`
}

// SubAccount represents a trading sub-account
type SubAccount struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TradingID uuid.UUID `gorm:"type:uuid;not null;index" json:"trading_id"`
	Name       string    `gorm:"type:varchar(100);not null" json:"name"`
	Symbol     string    `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Balance    float64   `gorm:"type:decimal(20,8);default:0" json:"balance"`
	Info       JSON      `gorm:"type:jsonb" json:"info"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User        User         `gorm:"foreignKey:UserID" json:"-"`
	Trading     Trading      `gorm:"foreignKey:TradingID" json:"-"`
	Transactions []Transaction `json:"-"`
	TradingLogs  []TradingLog  `json:"-"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index:idx_transactions_user_timestamp" json:"user_id"`
	TradingID     uuid.UUID `gorm:"type:uuid;not null;index:idx_transactions_trading_timestamp" json:"trading_id"`
	SubAccountID  uuid.UUID `gorm:"type:uuid;not null;index:idx_transactions_sub_account_timestamp" json:"sub_account_id"`
	Timestamp     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_transactions_user_timestamp,sort:desc;index:idx_transactions_trading_timestamp,sort:desc;index:idx_transactions_sub_account_timestamp,sort:desc" json:"timestamp"`
	Direction      string    `gorm:"type:varchar(10);not null;check:direction IN ('debit', 'credit');index" json:"direction"`
	Reason         string    `gorm:"type:varchar(50);not null;index" json:"reason"`
	Amount         float64   `gorm:"type:decimal(20,8);not null" json:"amount"`
	ClosingBalance float64   `gorm:"type:decimal(20,8);not null" json:"closing_balance"`
	Price          *float64  `gorm:"type:decimal(20,8)" json:"price,omitempty"`
	QuoteSymbol    *string   `gorm:"type:varchar(20)" json:"quote_symbol,omitempty"`
	Info           JSON      `gorm:"type:jsonb" json:"info"`

	// Relationships (no DeletedAt for time-series data)
	User       User        `gorm:"foreignKey:UserID" json:"-"`
	Trading    Trading     `gorm:"foreignKey:TradingID" json:"-"`
	SubAccount SubAccount  `gorm:"foreignKey:SubAccountID" json:"-"`
	TradingLog *TradingLog `gorm:"-" json:"-"`
}

// TradingLog represents a trading operation log entry
type TradingLog struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_trading_logs_user_timestamp" json:"user_id"`
	TradingID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_trading_logs_trading_timestamp" json:"trading_id"`
	SubAccountID  *uuid.UUID `gorm:"type:uuid;index:idx_trading_logs_sub_account_timestamp" json:"sub_account_id,omitempty"`
	TransactionID *uuid.UUID `gorm:"type:uuid;index" json:"transaction_id,omitempty"`
	Timestamp     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_trading_logs_user_timestamp,sort:desc;index:idx_trading_logs_trading_timestamp,sort:desc;index:idx_trading_logs_sub_account_timestamp,sort:desc" json:"timestamp"`
	Type          string     `gorm:"type:varchar(50);not null;index" json:"type"`
	Source        string     `gorm:"type:varchar(20);not null;check:source IN ('manual', 'bot');index" json:"source"`
	Message       string     `gorm:"type:text;not null" json:"message"`
	Info          JSON       `gorm:"type:jsonb" json:"info"`

	// Relationships (no DeletedAt for time-series data)
	User        User         `gorm:"foreignKey:UserID" json:"-"`
	Trading     Trading      `gorm:"foreignKey:TradingID" json:"-"`
	SubAccount  *SubAccount  `gorm:"foreignKey:SubAccountID" json:"-"`
	Transaction *Transaction `gorm:"-" json:"-"`
}

// EventProcessing represents NATS event processing tracking
type EventProcessing struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID      string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"event_id"`
	EventType    string     `gorm:"type:varchar(100);not null;index" json:"event_type"`
	UserID       *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	SubAccountID *uuid.UUID `gorm:"type:uuid;index" json:"sub_account_id,omitempty"`
	ProcessedAt  time.Time  `gorm:"default:CURRENT_TIMESTAMP;index" json:"processed_at"`
	Status       string     `gorm:"type:varchar(20);default:'processed';index" json:"status"`
	RetryCount   int        `gorm:"default:0" json:"retry_count"`
	ErrorMessage *string    `gorm:"type:text" json:"error_message,omitempty"`
	Info         JSON       `gorm:"type:jsonb" json:"info"`

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