package models

import (
	"encoding/json"
	"errors"
	"time"

	"tiris-backend/pkg/security"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecureTrading extends the Trading model with encrypted API keys and enhanced security
type SecureTrading struct {
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

// TableName returns the table name for SecureTrading
func (SecureTrading) TableName() string {
	return "tradings"
}

// TradingManager handles secure operations on trading data
type TradingManager struct {
	encryptionManager *security.EncryptionManager
	apiKeyManager     *security.APIKeyManager
}

// NewTradingManager creates a new trading manager with encryption capabilities
func NewTradingManager(masterKey, signingKey string) (*TradingManager, error) {
	encMgr, err := security.NewEncryptionManager(masterKey)
	if err != nil {
		return nil, err
	}

	apiMgr, err := security.NewAPIKeyManager(masterKey, signingKey)
	if err != nil {
		return nil, err
	}

	return &TradingManager{
		encryptionManager: encMgr,
		apiKeyManager:     apiMgr,
	}, nil
}

// SetAPICredentials encrypts and stores API credentials
func (em *TradingManager) SetAPICredentials(trading *SecureTrading, apiKey, apiSecret string) error {
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

	trading.EncryptedAPIKey = encryptedKey
	trading.EncryptedSecret = encryptedSecret
	trading.APIKeyHash = apiKeyHash

	return nil
}

// GetAPICredentials decrypts and returns API credentials
func (em *TradingManager) GetAPICredentials(trading *SecureTrading) (apiKey, apiSecret string, err error) {
	apiKey, err = em.encryptionManager.Decrypt(trading.EncryptedAPIKey)
	if err != nil {
		return "", "", err
	}

	apiSecret, err = em.encryptionManager.Decrypt(trading.EncryptedSecret)
	if err != nil {
		return "", "", err
	}

	return apiKey, apiSecret, nil
}

// GetMaskedAPIKey returns a masked version of the API key for display
func (em *TradingManager) GetMaskedAPIKey(trading *SecureTrading) (string, error) {
	apiKey, err := em.encryptionManager.Decrypt(trading.EncryptedAPIKey)
	if err != nil {
		return "", err
	}

	return security.MaskSensitiveData(apiKey, 4), nil
}

// ValidateAPIKey checks if the stored API key hash matches the provided key
func (em *TradingManager) ValidateAPIKey(trading *SecureTrading, apiKey string) bool {
	expectedHash := em.apiKeyManager.HashAPIKey(apiKey)
	return expectedHash == trading.APIKeyHash
}

// UpdateLastUsed updates the last used timestamp
func (em *TradingManager) UpdateLastUsed(db *gorm.DB, tradingID uuid.UUID) error {
	now := time.Now()
	return db.Model(&SecureTrading{}).
		Where("id = ?", tradingID).
		Update("last_used_at", now).Error
}

// RecordFailure records an API failure
func (em *TradingManager) RecordFailure(db *gorm.DB, tradingID uuid.UUID) error {
	now := time.Now()
	return db.Model(&SecureTrading{}).
		Where("id = ?", tradingID).
		Updates(map[string]interface{}{
			"failure_count":    gorm.Expr("failure_count + 1"),
			"last_failure_at":  now,
		}).Error
}

// ResetFailureCount resets the failure count after successful operation
func (em *TradingManager) ResetFailureCount(db *gorm.DB, tradingID uuid.UUID) error {
	return db.Model(&SecureTrading{}).
		Where("id = ?", tradingID).
		Updates(map[string]interface{}{
			"failure_count":   0,
			"last_failure_at": nil,
		}).Error
}

// ShouldDisableTrading checks if trading should be disabled due to failures
func (em *TradingManager) ShouldDisableTrading(trading *SecureTrading, maxFailures int, failureWindow time.Duration) bool {
	if trading.FailureCount >= maxFailures {
		if trading.LastFailureAt != nil && time.Since(*trading.LastFailureAt) < failureWindow {
			return true
		}
	}
	return false
}

// RotateAPIKey generates a new API key for internal use
func (em *TradingManager) RotateAPIKey(trading *SecureTrading) (string, error) {
	newAPIKey, err := em.apiKeyManager.GenerateAPIKey(security.PrefixExchange, 64)
	if err != nil {
		return "", err
	}

	// Update the trading with the new encrypted key
	encryptedKey, err := em.encryptionManager.Encrypt(newAPIKey)
	if err != nil {
		return "", err
	}

	trading.EncryptedAPIKey = encryptedKey
	trading.APIKeyHash = em.apiKeyManager.HashAPIKey(newAPIKey)

	return newAPIKey, nil
}

// SecuritySettings represents security configuration for an trading
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

// GetSecuritySettings returns the security settings for an trading
func (em *TradingManager) GetSecuritySettings(trading *SecureTrading) (*SecuritySettings, error) {
	settings := &SecuritySettings{
		MaxFailures:        10,
		FailureWindow:      time.Hour,
		RequireIPWhitelist: false,
		RateLimitEnabled:   true,
		MaxRequestsPerHour: 1000,
		AlertOnFailure:     true,
		AutoDisableOnAbuse: true,
	}

	if len(trading.SecuritySettings) > 0 {
		// Convert JSON map to bytes first, then unmarshal to settings
		data, err := json.Marshal(trading.SecuritySettings)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, settings); err != nil {
			return nil, err
		}
	}

	return settings, nil
}

// UpdateSecuritySettings updates the security settings for an trading
func (em *TradingManager) UpdateSecuritySettings(trading *SecureTrading, settings *SecuritySettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	// Convert []byte to JSON map
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}
	
	trading.SecuritySettings = JSON(jsonMap)
	return nil
}

// BeforeCreate hook to ensure API keys are encrypted
func (se *SecureTrading) BeforeCreate(tx *gorm.DB) error {
	if se.EncryptedAPIKey == "" || se.EncryptedSecret == "" {
		return errors.New("API credentials must be set before creating trading")
	}
	return nil
}

// BeforeUpdate hook to validate updates
func (se *SecureTrading) BeforeUpdate(tx *gorm.DB) error {
	// Don't allow updating encrypted fields directly
	if tx.Statement.Changed("encrypted_api_key", "encrypted_api_secret", "api_key_hash") {
		return errors.New("encrypted fields cannot be updated directly")
	}
	return nil
}

