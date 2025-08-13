package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidAPIKey    = errors.New("invalid API key format")
	ErrAPIKeyExpired    = errors.New("API key has expired")
	ErrInvalidSignature = errors.New("invalid API key signature")
)

// APIKeyPrefix represents different types of API keys
type APIKeyPrefix string

const (
	// PrefixExchange for exchange API keys
	PrefixExchange APIKeyPrefix = "txc_"
	// PrefixUser for user API keys
	PrefixUser APIKeyPrefix = "usr_"
	// PrefixService for service-to-service API keys
	PrefixService APIKeyPrefix = "svc_"
	// PrefixWebhook for webhook API keys
	PrefixWebhook APIKeyPrefix = "whk_"
)

// APIKeyManager handles API key generation, validation, and encryption
type APIKeyManager struct {
	encryptionManager *EncryptionManager
	signingKey        []byte
}

// APIKeyInfo contains metadata about an API key
type APIKeyInfo struct {
	ID          string    `json:"id"`
	Prefix      string    `json:"prefix"`
	Name        string    `json:"name,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	IsActive    bool      `json:"is_active"`
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(masterKey, signingKey string) (*APIKeyManager, error) {
	encMgr, err := NewEncryptionManager(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption manager: %v", err)
	}

	// Use SHA-256 of signing key for consistency
	signingKeyHash := sha256.Sum256([]byte(signingKey))

	return &APIKeyManager{
		encryptionManager: encMgr,
		signingKey:        signingKeyHash[:],
	}, nil
}

// GenerateAPIKey generates a new API key with the specified prefix
func (akm *APIKeyManager) GenerateAPIKey(prefix APIKeyPrefix, length int) (string, error) {
	if length < 32 {
		length = 32 // Minimum secure length
	}

	// Generate random bytes
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}

	// Encode to base64 and clean up for URL safety
	keyPart := base64.URLEncoding.EncodeToString(randomBytes)
	keyPart = strings.TrimRight(keyPart, "=") // Remove padding

	// Create the full key with prefix
	fullKey := string(prefix) + keyPart

	// Add signature for integrity
	signature := akm.signAPIKey(fullKey)
	signedKey := fullKey + "." + signature[:8] // Use first 8 chars of signature

	return signedKey, nil
}

// ValidateAPIKey validates the format and signature of an API key
func (akm *APIKeyManager) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return ErrInvalidAPIKey
	}

	// Check if key contains signature
	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return fmt.Errorf("%w: missing signature", ErrInvalidAPIKey)
	}

	keyPart, providedSig := parts[0], parts[1]

	// Validate signature
	expectedSig := akm.signAPIKey(keyPart)
	if providedSig != expectedSig[:8] {
		return ErrInvalidSignature
	}

	// Validate prefix
	validPrefix := false
	prefixes := []APIKeyPrefix{PrefixExchange, PrefixUser, PrefixService, PrefixWebhook}
	for _, prefix := range prefixes {
		if strings.HasPrefix(keyPart, string(prefix)) {
			validPrefix = true
			break
		}
	}

	if !validPrefix {
		return fmt.Errorf("%w: invalid prefix", ErrInvalidAPIKey)
	}

	return nil
}

// ExtractPrefix extracts the prefix from an API key
func (akm *APIKeyManager) ExtractPrefix(apiKey string) (APIKeyPrefix, error) {
	if err := akm.ValidateAPIKey(apiKey); err != nil {
		return "", err
	}

	parts := strings.Split(apiKey, ".")
	keyPart := parts[0]

	prefixes := []APIKeyPrefix{PrefixExchange, PrefixUser, PrefixService, PrefixWebhook}
	for _, prefix := range prefixes {
		if strings.HasPrefix(keyPart, string(prefix)) {
			return prefix, nil
		}
	}

	return "", fmt.Errorf("%w: unknown prefix", ErrInvalidAPIKey)
}

// EncryptAPIKey encrypts an API key for secure storage
func (akm *APIKeyManager) EncryptAPIKey(apiKey string) (string, error) {
	if err := akm.ValidateAPIKey(apiKey); err != nil {
		return "", fmt.Errorf("invalid API key: %v", err)
	}

	return akm.encryptionManager.Encrypt(apiKey)
}

// DecryptAPIKey decrypts an API key from storage
func (akm *APIKeyManager) DecryptAPIKey(encryptedKey string) (string, error) {
	decrypted, err := akm.encryptionManager.Decrypt(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %v", err)
	}

	if decrypted != "" {
		if err := akm.ValidateAPIKey(decrypted); err != nil {
			return "", fmt.Errorf("decrypted key validation failed: %v", err)
		}
	}

	return decrypted, nil
}

// MaskAPIKey masks an API key for safe display
func (akm *APIKeyManager) MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return MaskSensitiveData(apiKey, 4)
	}

	keyPart, sigPart := parts[0], parts[1]
	
	// Extract prefix
	var prefix string
	prefixes := []string{string(PrefixExchange), string(PrefixUser), string(PrefixService), string(PrefixWebhook)}
	for _, p := range prefixes {
		if strings.HasPrefix(keyPart, p) {
			prefix = p
			keyPart = keyPart[len(p):]
			break
		}
	}

	// Mask the key part but keep prefix and signature visible
	if len(keyPart) > 8 {
		maskedKey := keyPart[:4] + strings.Repeat("*", len(keyPart)-8) + keyPart[len(keyPart)-4:]
		return prefix + maskedKey + "." + sigPart
	}

	return prefix + "****." + sigPart
}

// HashAPIKey creates a secure hash of the API key for comparison
func (akm *APIKeyManager) HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey + string(akm.signingKey)))
	return hex.EncodeToString(hash[:])
}

// IsAPIKeyFormat checks if a string looks like an API key
func IsAPIKeyFormat(key string) bool {
	// Basic format check: prefix_base64.signature
	pattern := `^(txc_|usr_|svc_|whk_)[A-Za-z0-9_-]+\.[A-Za-z0-9]{8}$`
	matched, err := regexp.MatchString(pattern, key)
	return err == nil && matched
}

// signAPIKey creates a signature for the API key
func (akm *APIKeyManager) signAPIKey(keyPart string) string {
	data := append([]byte(keyPart), akm.signingKey...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// APIKeyPermissions defines permissions for different API key types
type APIKeyPermissions struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
	Admin  bool `json:"admin"`
}

// GetDefaultPermissions returns default permissions for each API key type
func GetDefaultPermissions(prefix APIKeyPrefix) APIKeyPermissions {
	switch prefix {
	case PrefixExchange:
		return APIKeyPermissions{Read: true, Write: true, Delete: false, Admin: false}
	case PrefixUser:
		return APIKeyPermissions{Read: true, Write: true, Delete: true, Admin: false}
	case PrefixService:
		return APIKeyPermissions{Read: true, Write: false, Delete: false, Admin: false}
	case PrefixWebhook:
		return APIKeyPermissions{Read: false, Write: true, Delete: false, Admin: false}
	default:
		return APIKeyPermissions{Read: false, Write: false, Delete: false, Admin: false}
	}
}