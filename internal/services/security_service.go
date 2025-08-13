package services

import (
	"context"
	"fmt"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/pkg/security"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecurityService provides comprehensive security operations
type SecurityService struct {
	db                *gorm.DB
	redis             *redis.Client
	rateLimiter       *security.RateLimiter
	auditLogger       *security.AuditLogger
	apiKeyManager     *security.APIKeyManager
	exchangeManager   *models.ExchangeManager
	encryptionManager *security.EncryptionManager
}

// NewSecurityService creates a new security service
func NewSecurityService(db *gorm.DB, redisClient *redis.Client, masterKey, signingKey string) (*SecurityService, error) {
	rateLimiter := security.NewRateLimiter(redisClient, "tiris:security")
	auditLogger := security.NewAuditLogger(db)
	
	apiKeyManager, err := security.NewAPIKeyManager(masterKey, signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key manager: %v", err)
	}

	exchangeManager, err := models.NewExchangeManager(masterKey, signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create exchange manager: %v", err)
	}

	encryptionManager, err := security.NewEncryptionManager(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption manager: %v", err)
	}

	return &SecurityService{
		db:                db,
		redis:             redisClient,
		rateLimiter:       rateLimiter,
		auditLogger:       auditLogger,
		apiKeyManager:     apiKeyManager,
		exchangeManager:   exchangeManager,
		encryptionManager: encryptionManager,
	}, nil
}

// CreateUserAPIKey creates a new API key for a user
func (ss *SecurityService) CreateUserAPIKey(ctx context.Context, userID uuid.UUID, name string, permissions []string) (*UserAPIKey, error) {
	// Generate new API key
	apiKey, err := ss.apiKeyManager.GenerateAPIKey(security.PrefixUser, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %v", err)
	}

	// Encrypt the API key for storage
	encryptedKey, err := ss.apiKeyManager.EncryptAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %v", err)
	}

	// Create API key record
	keyRecord := &UserAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         name,
		EncryptedKey: encryptedKey,
		KeyHash:      ss.apiKeyManager.HashAPIKey(apiKey),
		Permissions:  permissions,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := ss.db.WithContext(ctx).Create(keyRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to save API key: %v", err)
	}

	// Log API key creation
	ss.auditLogger.LogSecurityEvent(
		ctx,
		security.ActionAPIKeyCreate,
		&userID,
		"",
		map[string]interface{}{
			"api_key_id":   keyRecord.ID,
			"api_key_name": name,
			"permissions":  permissions,
		},
		nil,
	)

	// Return the key with the plain text API key (only time it's visible)
	keyRecord.PlaintextKey = &apiKey
	return keyRecord, nil
}

// ValidateAPIKey validates an API key and returns associated user information
func (ss *SecurityService) ValidateAPIKey(ctx context.Context, apiKey string) (*APIKeyValidationResult, error) {
	// Validate format
	if err := ss.apiKeyManager.ValidateAPIKey(apiKey); err != nil {
		return nil, err
	}

	// Get key hash for lookup
	keyHash := ss.apiKeyManager.HashAPIKey(apiKey)

	// Find the API key record
	var keyRecord UserAPIKey
	if err := ss.db.WithContext(ctx).
		Where("key_hash = ? AND is_active = true", keyHash).
		First(&keyRecord).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &APIKeyValidationResult{
				Valid: false,
				Error: "API key not found or inactive",
			}, nil
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	// Check if key has expired
	if keyRecord.ExpiresAt != nil && time.Now().After(*keyRecord.ExpiresAt) {
		return &APIKeyValidationResult{
			Valid: false,
			Error: "API key has expired",
		}, nil
	}

	// Update last used timestamp
	now := time.Now()
	ss.db.WithContext(ctx).
		Model(&keyRecord).
		Update("last_used_at", now)

	return &APIKeyValidationResult{
		Valid:       true,
		UserID:      keyRecord.UserID,
		APIKeyID:    keyRecord.ID,
		Permissions: keyRecord.Permissions,
		LastUsedAt:  &now,
	}, nil
}

// RotateAPIKey generates a new API key and deactivates the old one
func (ss *SecurityService) RotateAPIKey(ctx context.Context, userID, apiKeyID uuid.UUID) (*UserAPIKey, error) {
	// Find existing key
	var existingKey UserAPIKey
	if err := ss.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", apiKeyID, userID).
		First(&existingKey).Error; err != nil {
		return nil, fmt.Errorf("API key not found: %v", err)
	}

	// Generate new API key
	newAPIKey, err := ss.apiKeyManager.GenerateAPIKey(security.PrefixUser, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new API key: %v", err)
	}

	// Encrypt the new API key
	encryptedKey, err := ss.apiKeyManager.EncryptAPIKey(newAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt new API key: %v", err)
	}

	// Use transaction to ensure atomicity
	tx := ss.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Deactivate old key
	if err := tx.Model(&existingKey).Update("is_active", false).Error; err != nil {
		return nil, fmt.Errorf("failed to deactivate old key: %v", err)
	}

	// Create new key record
	newKeyRecord := &UserAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         existingKey.Name + " (Rotated)",
		EncryptedKey: encryptedKey,
		KeyHash:      ss.apiKeyManager.HashAPIKey(newAPIKey),
		Permissions:  existingKey.Permissions,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := tx.Create(newKeyRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to create new API key: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Log API key rotation
	ss.auditLogger.LogSecurityEvent(
		ctx,
		security.ActionAPIKeyUpdate,
		&userID,
		"",
		map[string]interface{}{
			"action":        "rotate",
			"old_key_id":    existingKey.ID,
			"new_key_id":    newKeyRecord.ID,
			"api_key_name":  existingKey.Name,
		},
		nil,
	)

	newKeyRecord.PlaintextKey = &newAPIKey
	return newKeyRecord, nil
}

// CheckRateLimit checks if a request should be rate limited
func (ss *SecurityService) CheckRateLimit(ctx context.Context, identifier, ruleName string) (*security.RateLimitResult, error) {
	rules := security.DefaultRules()
	rule, exists := rules[ruleName]
	if !exists {
		rule = rules["api_general"]
	}

	return ss.rateLimiter.CheckRateLimit(ctx, identifier, ruleName, rule)
}

// GetSecurityAlerts retrieves recent security alerts
func (ss *SecurityService) GetSecurityAlerts(ctx context.Context, since time.Time, limit int) ([]security.AuditEvent, error) {
	return ss.auditLogger.GetSecurityAlerts(ctx, since, limit)
}

// GetSuspiciousActivity analyzes and returns suspicious activity patterns
func (ss *SecurityService) GetSuspiciousActivity(ctx context.Context, timeWindow time.Duration) ([]security.SuspiciousActivity, error) {
	return ss.auditLogger.GetSuspiciousActivity(ctx, timeWindow)
}

// EncryptSensitiveData encrypts sensitive data using the encryption manager
func (ss *SecurityService) EncryptSensitiveData(data string) (string, error) {
	return ss.encryptionManager.Encrypt(data)
}

// DecryptSensitiveData decrypts sensitive data using the encryption manager
func (ss *SecurityService) DecryptSensitiveData(encryptedData string) (string, error) {
	return ss.encryptionManager.Decrypt(encryptedData)
}

// CreateSecureExchange creates a new exchange with encrypted API credentials
func (ss *SecurityService) CreateSecureExchange(ctx context.Context, userID uuid.UUID, name, exchangeType, apiKey, apiSecret string) (*models.SecureExchange, error) {
	exchange := &models.SecureExchange{
		ID:     uuid.New(),
		UserID: userID,
		Name:   name,
		Type:   exchangeType,
		Status: "active",
	}

	// Set encrypted API credentials
	if err := ss.exchangeManager.SetAPICredentials(exchange, apiKey, apiSecret); err != nil {
		return nil, fmt.Errorf("failed to encrypt API credentials: %v", err)
	}

	// Save to database
	if err := ss.db.WithContext(ctx).Create(exchange).Error; err != nil {
		return nil, fmt.Errorf("failed to create exchange: %v", err)
	}

	// Log exchange creation
	ss.auditLogger.LogSecurityEvent(
		ctx,
		security.ActionExchangeCreate,
		&userID,
		"",
		map[string]interface{}{
			"exchange_id":   exchange.ID,
			"exchange_name": name,
			"exchange_type": exchangeType,
		},
		nil,
	)

	return exchange, nil
}

// GetExchangeCredentials retrieves and decrypts exchange API credentials
func (ss *SecurityService) GetExchangeCredentials(ctx context.Context, userID, exchangeID uuid.UUID) (apiKey, apiSecret string, err error) {
	var exchange models.SecureExchange
	if err := ss.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", exchangeID, userID).
		First(&exchange).Error; err != nil {
		return "", "", fmt.Errorf("exchange not found: %v", err)
	}

	return ss.exchangeManager.GetAPICredentials(&exchange)
}

// AuditDataAccess logs data access for compliance
func (ss *SecurityService) AuditDataAccess(ctx context.Context, userID *uuid.UUID, action security.AuditAction, resourceType, resourceID, ipAddress string, success bool) error {
	return ss.auditLogger.LogDataAccess(ctx, userID, action, resourceType, resourceID, ipAddress, success)
}

// Supporting types

// UserAPIKey represents an API key for a user
type UserAPIKey struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	Name         string     `json:"name" gorm:"type:varchar(100);not null"`
	EncryptedKey string     `json:"-" gorm:"type:text;not null"`
	KeyHash      string     `json:"-" gorm:"type:varchar(64);not null;index"`
	Permissions  []string   `json:"permissions" gorm:"type:text[]"`
	IsActive     bool       `json:"is_active" gorm:"default:true;index"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at" gorm:"default:now()"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"default:now()"`

	// This field is only populated when creating/rotating keys
	PlaintextKey *string `json:"plaintext_key,omitempty" gorm:"-"`

	// Relationships
	User models.User `json:"-" gorm:"foreignKey:UserID"`
}

// TableName returns the table name for UserAPIKey
func (UserAPIKey) TableName() string {
	return "user_api_keys"
}

// APIKeyValidationResult represents the result of API key validation
type APIKeyValidationResult struct {
	Valid       bool       `json:"valid"`
	UserID      uuid.UUID  `json:"user_id,omitempty"`
	APIKeyID    uuid.UUID  `json:"api_key_id,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// MaskedAPIKey returns a masked version of the API key for safe display
func (uak *UserAPIKey) MaskedAPIKey() string {
	if uak.PlaintextKey != nil {
		return security.MaskSensitiveData(*uak.PlaintextKey, 4)
	}
	return "****"
}

// HasPermission checks if the API key has a specific permission
func (uak *UserAPIKey) HasPermission(permission string) bool {
	for _, p := range uak.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}