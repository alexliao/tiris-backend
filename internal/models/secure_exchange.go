package models

import (
	"encoding/json"
	"errors"
	"time"

	"tiris-backend/pkg/security"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecureExchange extends the Exchange model with encrypted API keys and enhanced security
type SecureExchange struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Name             string         `gorm:"type:varchar(100);not null" json:"name"`
	Type             string         `gorm:"type:varchar(50);not null;index" json:"type"`
	EncryptedAPIKey  string         `gorm:"type:text;not null;column:encrypted_api_key" json:"-"`
	EncryptedSecret  string         `gorm:"type:text;not null;column:encrypted_api_secret" json:"-"`
	APIKeyHash       string         `gorm:"type:varchar(64);not null;index;column:api_key_hash" json:"-"`
	Status           string         `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Info             JSON `gorm:"type:jsonb;default:'{}'" json:"info"`
	LastUsedAt       *time.Time     `gorm:"column:last_used_at" json:"last_used_at,omitempty"`
	FailureCount     int            `gorm:"default:0;column:failure_count" json:"-"`
	LastFailureAt    *time.Time     `gorm:"column:last_failure_at" json:"-"`
	SecuritySettings JSON `gorm:"type:jsonb;default:'{}';column:security_settings" json:"security_settings"`

	CreatedAt time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User        User         `gorm:"foreignKey:UserID" json:"-"`
	SubAccounts []SubAccount `json:"-"`
}

// TableName returns the table name for SecureExchange
func (SecureExchange) TableName() string {
	return "exchanges"
}

// ExchangeManager handles secure operations on exchange data
type ExchangeManager struct {
	encryptionManager *security.EncryptionManager
	apiKeyManager     *security.APIKeyManager
}

// NewExchangeManager creates a new exchange manager with encryption capabilities
func NewExchangeManager(masterKey, signingKey string) (*ExchangeManager, error) {
	encMgr, err := security.NewEncryptionManager(masterKey)
	if err != nil {
		return nil, err
	}

	apiMgr, err := security.NewAPIKeyManager(masterKey, signingKey)
	if err != nil {
		return nil, err
	}

	return &ExchangeManager{
		encryptionManager: encMgr,
		apiKeyManager:     apiMgr,
	}, nil
}

// SetAPICredentials encrypts and stores API credentials
func (em *ExchangeManager) SetAPICredentials(exchange *SecureExchange, apiKey, apiSecret string) error {
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	if apiSecret == "" {
		return errors.New("API secret cannot be empty")
	}

	// Encrypt the API key and secret
	encryptedKey, err := em.encryptionManager.Encrypt(apiKey)
	if err != nil {
		return err
	}

	encryptedSecret, err := em.encryptionManager.Encrypt(apiSecret)
	if err != nil {
		return err
	}

	// Create hash for indexing and comparison
	apiKeyHash := em.apiKeyManager.HashAPIKey(apiKey)

	exchange.EncryptedAPIKey = encryptedKey
	exchange.EncryptedSecret = encryptedSecret
	exchange.APIKeyHash = apiKeyHash

	return nil
}

// GetAPICredentials decrypts and returns API credentials
func (em *ExchangeManager) GetAPICredentials(exchange *SecureExchange) (apiKey, apiSecret string, err error) {
	apiKey, err = em.encryptionManager.Decrypt(exchange.EncryptedAPIKey)
	if err != nil {
		return "", "", err
	}

	apiSecret, err = em.encryptionManager.Decrypt(exchange.EncryptedSecret)
	if err != nil {
		return "", "", err
	}

	return apiKey, apiSecret, nil
}

// GetMaskedAPIKey returns a masked version of the API key for display
func (em *ExchangeManager) GetMaskedAPIKey(exchange *SecureExchange) (string, error) {
	apiKey, err := em.encryptionManager.Decrypt(exchange.EncryptedAPIKey)
	if err != nil {
		return "", err
	}

	return security.MaskSensitiveData(apiKey, 4), nil
}

// ValidateAPIKey checks if the stored API key hash matches the provided key
func (em *ExchangeManager) ValidateAPIKey(exchange *SecureExchange, apiKey string) bool {
	expectedHash := em.apiKeyManager.HashAPIKey(apiKey)
	return expectedHash == exchange.APIKeyHash
}

// UpdateLastUsed updates the last used timestamp
func (em *ExchangeManager) UpdateLastUsed(db *gorm.DB, exchangeID uuid.UUID) error {
	now := time.Now()
	return db.Model(&SecureExchange{}).
		Where("id = ?", exchangeID).
		Update("last_used_at", now).Error
}

// RecordFailure records an API failure
func (em *ExchangeManager) RecordFailure(db *gorm.DB, exchangeID uuid.UUID) error {
	now := time.Now()
	return db.Model(&SecureExchange{}).
		Where("id = ?", exchangeID).
		Updates(map[string]interface{}{
			"failure_count":    gorm.Expr("failure_count + 1"),
			"last_failure_at":  now,
		}).Error
}

// ResetFailureCount resets the failure count after successful operation
func (em *ExchangeManager) ResetFailureCount(db *gorm.DB, exchangeID uuid.UUID) error {
	return db.Model(&SecureExchange{}).
		Where("id = ?", exchangeID).
		Updates(map[string]interface{}{
			"failure_count":   0,
			"last_failure_at": nil,
		}).Error
}

// ShouldDisableExchange checks if exchange should be disabled due to failures
func (em *ExchangeManager) ShouldDisableExchange(exchange *SecureExchange, maxFailures int, failureWindow time.Duration) bool {
	if exchange.FailureCount >= maxFailures {
		if exchange.LastFailureAt != nil && time.Since(*exchange.LastFailureAt) < failureWindow {
			return true
		}
	}
	return false
}

// RotateAPIKey generates a new API key for internal use
func (em *ExchangeManager) RotateAPIKey(exchange *SecureExchange) (string, error) {
	newAPIKey, err := em.apiKeyManager.GenerateAPIKey(security.PrefixExchange, 64)
	if err != nil {
		return "", err
	}

	// Update the exchange with the new encrypted key
	encryptedKey, err := em.encryptionManager.Encrypt(newAPIKey)
	if err != nil {
		return "", err
	}

	exchange.EncryptedAPIKey = encryptedKey
	exchange.APIKeyHash = em.apiKeyManager.HashAPIKey(newAPIKey)

	return newAPIKey, nil
}

// SecuritySettings represents security configuration for an exchange
type SecuritySettings struct {
	MaxFailures         int           `json:"max_failures"`
	FailureWindow       time.Duration `json:"failure_window"`
	RequireIPWhitelist  bool          `json:"require_ip_whitelist"`
	AllowedIPs          []string      `json:"allowed_ips"`
	RateLimitEnabled    bool          `json:"rate_limit_enabled"`
	MaxRequestsPerHour  int           `json:"max_requests_per_hour"`
	AlertOnFailure      bool          `json:"alert_on_failure"`
	AutoDisableOnAbuse  bool          `json:"auto_disable_on_abuse"`
	LastSecurityAuditAt *time.Time    `json:"last_security_audit_at"`
}

// GetSecuritySettings returns the security settings for an exchange
func (em *ExchangeManager) GetSecuritySettings(exchange *SecureExchange) (*SecuritySettings, error) {
	settings := &SecuritySettings{
		MaxFailures:        10,
		FailureWindow:      time.Hour,
		RequireIPWhitelist: false,
		RateLimitEnabled:   true,
		MaxRequestsPerHour: 1000,
		AlertOnFailure:     true,
		AutoDisableOnAbuse: true,
	}

	if len(exchange.SecuritySettings) > 0 {
		// Convert JSON map to bytes first, then unmarshal to settings
		data, err := json.Marshal(exchange.SecuritySettings)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, settings); err != nil {
			return nil, err
		}
	}

	return settings, nil
}

// UpdateSecuritySettings updates the security settings for an exchange
func (em *ExchangeManager) UpdateSecuritySettings(exchange *SecureExchange, settings *SecuritySettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	// Convert []byte to JSON map
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}
	
	exchange.SecuritySettings = JSON(jsonMap)
	return nil
}

// BeforeCreate hook to ensure API keys are encrypted
func (se *SecureExchange) BeforeCreate(tx *gorm.DB) error {
	if se.EncryptedAPIKey == "" || se.EncryptedSecret == "" {
		return errors.New("API credentials must be set before creating exchange")
	}
	return nil
}

// BeforeUpdate hook to validate updates
func (se *SecureExchange) BeforeUpdate(tx *gorm.DB) error {
	// Don't allow updating encrypted fields directly
	if tx.Statement.Changed("encrypted_api_key", "encrypted_api_secret", "api_key_hash") {
		return errors.New("encrypted fields cannot be updated directly")
	}
	return nil
}

// ExchangeResponse represents the API response format for exchanges
type ExchangeResponse struct {
	ID               uuid.UUID      `json:"id"`
	UserID           uuid.UUID      `json:"user_id"`
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	MaskedAPIKey     string         `json:"masked_api_key"`
	Status           string         `json:"status"`
	Info             JSON `json:"info"`
	LastUsedAt       *time.Time     `json:"last_used_at,omitempty"`
	SecuritySettings JSON `json:"security_settings"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// ToResponse converts SecureExchange to ExchangeResponse with masked credentials
func (se *SecureExchange) ToResponse(exchangeManager *ExchangeManager) (*ExchangeResponse, error) {
	maskedKey, err := exchangeManager.GetMaskedAPIKey(se)
	if err != nil {
		maskedKey = "***"
	}

	return &ExchangeResponse{
		ID:               se.ID,
		UserID:           se.UserID,
		Name:             se.Name,
		Type:             se.Type,
		MaskedAPIKey:     maskedKey,
		Status:           se.Status,
		Info:             se.Info,
		LastUsedAt:       se.LastUsedAt,
		SecuritySettings: se.SecuritySettings,
		CreatedAt:        se.CreatedAt,
		UpdatedAt:        se.UpdatedAt,
	}, nil
}