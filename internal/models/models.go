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

// ExchangeBinding constants
const (
	// Exchange types
	ExchangeBinance  = "binance"
	ExchangeKraken   = "kraken"
	ExchangeGate     = "gate"
	ExchangeCoinbase = "coinbase"
	ExchangeVirtual  = "virtual"

	// Binding types
	ExchangeBindingTypePrivate = "private"
	ExchangeBindingTypePublic  = "public"

	// Binding statuses
	ExchangeBindingStatusActive   = "active"
	ExchangeBindingStatusInactive = "inactive"
	ExchangeBindingStatusError    = "error"
)

// ExchangeBinding represents an exchange connection configuration
type ExchangeBinding struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	Exchange  string     `gorm:"type:varchar(50);not null;index" json:"exchange"`
	Type      string     `gorm:"type:varchar(20);not null;index" json:"type"`
	APIKey    string     `gorm:"type:text" json:"-"`
	APISecret string     `gorm:"type:text" json:"-"`
	Status    string     `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Info      JSON       `gorm:"type:jsonb" json:"info"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships  
	User     *User     `gorm:"-:all" json:"-"` // Disable GORM relationship handling
	Tradings []Trading `json:"-"`
}

// TableName returns the table name for ExchangeBinding
func (ExchangeBinding) TableName() string {
	return "exchange_bindings"
}

// Validate validates the exchange binding
func (eb *ExchangeBinding) Validate() error {
	if eb.Name == "" {
		return errors.New("name is required")
	}

	validExchanges := []string{ExchangeBinance, ExchangeKraken, ExchangeGate, ExchangeCoinbase, ExchangeVirtual}
	isValidExchange := false
	for _, exchange := range validExchanges {
		if eb.Exchange == exchange {
			isValidExchange = true
			break
		}
	}
	if !isValidExchange {
		return errors.New("invalid exchange")
	}

	if eb.Type != ExchangeBindingTypePrivate && eb.Type != ExchangeBindingTypePublic {
		return errors.New("invalid type")
	}

	validStatuses := []string{ExchangeBindingStatusActive, ExchangeBindingStatusInactive, ExchangeBindingStatusError}
	isValidStatus := false
	for _, status := range validStatuses {
		if eb.Status == status {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return errors.New("invalid status")
	}

	// Private bindings must have a user
	if eb.Type == ExchangeBindingTypePrivate && eb.UserID == nil {
		return errors.New("private bindings must have a user")
	}

	// Private bindings must have API credentials
	if eb.Type == ExchangeBindingTypePrivate && (eb.APIKey == "" || eb.APISecret == "") {
		return errors.New("private bindings must have API credentials")
	}

	return nil
}

// IsPrivate returns true if the binding is private
func (eb *ExchangeBinding) IsPrivate() bool {
	return eb.Type == ExchangeBindingTypePrivate
}

// IsPublic returns true if the binding is public
func (eb *ExchangeBinding) IsPublic() bool {
	return eb.Type == ExchangeBindingTypePublic
}

// HasCredentials returns true if the binding has API credentials
func (eb *ExchangeBinding) HasCredentials() bool {
	return eb.APIKey != "" && eb.APISecret != ""
}

// GetMaskedAPIKey returns a masked version of the API key for display
func (eb *ExchangeBinding) GetMaskedAPIKey() string {
	if eb.APIKey == "" {
		return ""
	}
	if len(eb.APIKey) <= 8 {
		return "***"
	}
	return eb.APIKey[:4] + "..." + eb.APIKey[len(eb.APIKey)-4:]
}

// ToResponse converts ExchangeBinding to ExchangeBindingResponse
func (eb *ExchangeBinding) ToResponse() *ExchangeBindingResponse {
	return &ExchangeBindingResponse{
		ID:           eb.ID,
		UserID:       eb.UserID,
		Name:         eb.Name,
		Exchange:     eb.Exchange,
		Type:         eb.Type,
		MaskedAPIKey: eb.GetMaskedAPIKey(),
		Status:       eb.Status,
		Info:         eb.Info,
		CreatedAt:    eb.CreatedAt,
		UpdatedAt:    eb.UpdatedAt,
	}
}

// ExchangeBindingResponse represents the API response for exchange bindings
type ExchangeBindingResponse struct {
	ID           uuid.UUID  `json:"id"`
	UserID       *uuid.UUID `json:"user_id"`
	Name         string     `json:"name"`
	Exchange     string     `json:"exchange"`
	Type         string     `json:"type"`
	MaskedAPIKey string     `json:"masked_api_key"`
	Status       string     `json:"status"`
	Info         JSON       `json:"info"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Trading represents a trading connection
type Trading struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ExchangeBindingID uuid.UUID `gorm:"type:uuid;not null;index" json:"exchange_binding_id"`
	Name              string    `gorm:"type:varchar(100);not null" json:"name"`
	Type              string    `gorm:"type:varchar(50);not null;index" json:"type"`
	Status            string    `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Info              JSON      `gorm:"type:jsonb" json:"info"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User            User             `gorm:"foreignKey:UserID" json:"-"`
	ExchangeBinding ExchangeBinding  `gorm:"foreignKey:ExchangeBindingID" json:"-"`
	SubAccounts     []SubAccount     `json:"-"`
}

// TableName returns the table name for Trading
func (Trading) TableName() string {
	return "tradings"
}

// Validate validates the trading
func (t *Trading) Validate() error {
	if t.Name == "" {
		return errors.New("name cannot be empty")
	}

	if t.UserID == uuid.Nil {
		return errors.New("user ID cannot be nil")
	}

	if t.ExchangeBindingID == uuid.Nil {
		return errors.New("exchange binding ID cannot be nil")
	}

	validTypes := []string{TradingTypeReal, TradingTypeVirtual, TradingTypeBacktest}
	isValidType := false
	for _, tradingType := range validTypes {
		if t.Type == tradingType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("invalid trading type")
	}

	validStatuses := []string{TradingStatusActive, TradingStatusInactive, TradingStatusPaused}
	isValidStatus := false
	for _, status := range validStatuses {
		if t.Status == status {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return errors.New("invalid status")
	}

	return nil
}

// IsReal returns true if the trading is real
func (t *Trading) IsReal() bool {
	return t.Type == TradingTypeReal
}

// IsVirtual returns true if the trading is virtual
func (t *Trading) IsVirtual() bool {
	return t.Type == TradingTypeVirtual
}

// IsBacktest returns true if the trading is backtest
func (t *Trading) IsBacktest() bool {
	return t.Type == TradingTypeBacktest
}

// IsActive returns true if the trading is active
func (t *Trading) IsActive() bool {
	return t.Status == TradingStatusActive
}

// ToResponse converts Trading to TradingResponse
func (t *Trading) ToResponse() *TradingResponse {
	response := &TradingResponse{
		ID:                t.ID,
		UserID:            t.UserID,
		ExchangeBindingID: t.ExchangeBindingID,
		Name:              t.Name,
		Type:              t.Type,
		Status:            t.Status,
		Info:              t.Info,
		CreatedAt:         t.CreatedAt,
		UpdatedAt:         t.UpdatedAt,
	}

	// Include exchange binding information if loaded
	if t.ExchangeBinding.ID != uuid.Nil {
		response.ExchangeBindingName = t.ExchangeBinding.Name
		response.Exchange = t.ExchangeBinding.Exchange
		response.ExchangeBindingType = t.ExchangeBinding.Type
	}

	return response
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

// Exchange Binding Request/Response Models

// CreateExchangeBindingRequest represents a request to create a new exchange binding
type CreateExchangeBindingRequest struct {
	UserID    *uuid.UUID `json:"user_id"`
	Name      string     `json:"name" binding:"required"`
	Exchange  string     `json:"exchange" binding:"required"`
	Type      string     `json:"type" binding:"required"`
	APIKey    string     `json:"api_key"`
	APISecret string     `json:"api_secret"`
	Info      JSON       `json:"info,omitempty"`
}

// Validate validates the create exchange binding request
func (r *CreateExchangeBindingRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}

	if r.Exchange == "" {
		return errors.New("exchange is required")
	}

	if r.Type == "" {
		return errors.New("type is required")
	}

	validExchanges := []string{ExchangeBinance, ExchangeKraken, ExchangeGate, ExchangeCoinbase, ExchangeVirtual}
	isValidExchange := false
	for _, exchange := range validExchanges {
		if r.Exchange == exchange {
			isValidExchange = true
			break
		}
	}
	if !isValidExchange {
		return errors.New("invalid exchange")
	}

	if r.Type != ExchangeBindingTypePrivate && r.Type != ExchangeBindingTypePublic {
		return errors.New("invalid type")
	}

	// Private bindings must have API credentials
	if r.Type == ExchangeBindingTypePrivate && (r.APIKey == "" || r.APISecret == "") {
		return errors.New("API credentials are required for private bindings")
	}

	// Private bindings must have a user
	if r.Type == ExchangeBindingTypePrivate && r.UserID == nil {
		return errors.New("user ID is required for private bindings")
	}

	return nil
}

// ToExchangeBinding converts the request to an ExchangeBinding model
func (r *CreateExchangeBindingRequest) ToExchangeBinding() *ExchangeBinding {
	return &ExchangeBinding{
		UserID:    r.UserID,
		Name:      r.Name,
		Exchange:  r.Exchange,
		Type:      r.Type,
		APIKey:    r.APIKey,
		APISecret: r.APISecret,
		Status:    ExchangeBindingStatusActive,
		Info:      r.Info,
	}
}

// UpdateExchangeBindingRequest represents a request to update an exchange binding
type UpdateExchangeBindingRequest struct {
	Name      *string `json:"name,omitempty"`
	APIKey    *string `json:"api_key,omitempty"`
	APISecret *string `json:"api_secret,omitempty"`
	Status    *string `json:"status,omitempty"`
	Info      JSON    `json:"info,omitempty"`
}

// Validate validates the update exchange binding request
func (r *UpdateExchangeBindingRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return errors.New("name cannot be empty")
	}

	if r.Status != nil {
		validStatuses := []string{ExchangeBindingStatusActive, ExchangeBindingStatusInactive, ExchangeBindingStatusError}
		isValidStatus := false
		for _, status := range validStatuses {
			if *r.Status == status {
				isValidStatus = true
				break
			}
		}
		if !isValidStatus {
			return errors.New("invalid status")
		}
	}

	return nil
}

// ToUpdateMap converts the request to a map for database updates
func (r *UpdateExchangeBindingRequest) ToUpdateMap() map[string]interface{} {
	updates := make(map[string]interface{})

	if r.Name != nil {
		updates["name"] = *r.Name
	}
	if r.APIKey != nil {
		updates["api_key"] = *r.APIKey
	}
	if r.APISecret != nil {
		updates["api_secret"] = *r.APISecret
	}
	if r.Status != nil {
		updates["status"] = *r.Status
	}
	if r.Info != nil {
		updates["info"] = r.Info
	}

	return updates
}

// Trading Request/Response Models (Updated)

// Trading constants
const (
	// Trading types
	TradingTypeReal     = "real"
	TradingTypeVirtual  = "virtual"
	TradingTypeBacktest = "backtest"

	// Trading statuses
	TradingStatusActive   = "active"
	TradingStatusInactive = "inactive"
	TradingStatusPaused   = "paused"
)

// CreateTradingRequest represents a request to create a new trading
type CreateTradingRequest struct {
	UserID            uuid.UUID `json:"user_id"`
	ExchangeBindingID uuid.UUID `json:"exchange_binding_id" binding:"required"`
	Name              string    `json:"name" binding:"required"`
	Type              string    `json:"type" binding:"required"`
	Info              JSON      `json:"info,omitempty"`
}

// Validate validates the create trading request
func (r *CreateTradingRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}

	if r.Type == "" {
		return errors.New("type is required")
	}

	validTypes := []string{TradingTypeReal, TradingTypeVirtual, TradingTypeBacktest}
	isValidType := false
	for _, tradingType := range validTypes {
		if r.Type == tradingType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("invalid trading type")
	}

	if r.ExchangeBindingID == uuid.Nil {
		return errors.New("exchange binding ID is required")
	}

	if r.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}

	return nil
}

// ToTrading converts the request to a Trading model
func (r *CreateTradingRequest) ToTrading() *Trading {
	return &Trading{
		UserID:            r.UserID,
		ExchangeBindingID: r.ExchangeBindingID,
		Name:              r.Name,
		Type:              r.Type,
		Status:            TradingStatusActive,
		Info:              r.Info,
	}
}

// UpdateTradingRequest represents a request to update a trading
type UpdateTradingRequest struct {
	Name              *string    `json:"name,omitempty"`
	ExchangeBindingID *uuid.UUID `json:"exchange_binding_id,omitempty"`
	Status            *string    `json:"status,omitempty"`
	Info              JSON       `json:"info,omitempty"`
}

// Validate validates the update trading request
func (r *UpdateTradingRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return errors.New("name cannot be empty")
	}

	if r.Status != nil {
		validStatuses := []string{TradingStatusActive, TradingStatusInactive, TradingStatusPaused}
		isValidStatus := false
		for _, status := range validStatuses {
			if *r.Status == status {
				isValidStatus = true
				break
			}
		}
		if !isValidStatus {
			return errors.New("invalid status")
		}
	}

	return nil
}

// ToUpdateMap converts the request to a map for database updates
func (r *UpdateTradingRequest) ToUpdateMap() map[string]interface{} {
	updates := make(map[string]interface{})

	if r.Name != nil {
		updates["name"] = *r.Name
	}
	if r.ExchangeBindingID != nil {
		updates["exchange_binding_id"] = *r.ExchangeBindingID
	}
	if r.Status != nil {
		updates["status"] = *r.Status
	}
	if r.Info != nil {
		updates["info"] = r.Info
	}

	return updates
}

// TradingResponse represents the API response for tradings
type TradingResponse struct {
	ID                    uuid.UUID `json:"id"`
	UserID                uuid.UUID `json:"user_id"`
	ExchangeBindingID     uuid.UUID `json:"exchange_binding_id"`
	Name                  string    `json:"name"`
	Type                  string    `json:"type"`
	Status                string    `json:"status"`
	Info                  JSON      `json:"info"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	ExchangeBindingName   string    `json:"exchange_binding_name,omitempty"`
	Exchange              string    `json:"exchange,omitempty"`
	ExchangeBindingType   string    `json:"exchange_binding_type,omitempty"`
}

// Common Request/Response Models

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

// PaginationResult represents pagination result metadata
type PaginationResult struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// Error definitions
var (
	// Exchange Binding errors
	ErrExchangeBindingNotFound   = errors.New("exchange binding not found")
	ErrExchangeBindingNameExists = errors.New("exchange binding with this name already exists")
	ErrExchangeBindingInUse      = errors.New("exchange binding is currently in use")

	// Trading errors
	ErrTradingNotFound   = errors.New("trading not found")
	ErrTradingNameExists = errors.New("trading with this name already exists")
)