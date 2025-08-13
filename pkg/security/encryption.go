package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrInvalidKeySize    = errors.New("invalid key size")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrEncryptionFailed  = errors.New("encryption failed")
)

// EncryptionManager handles AES-256-GCM encryption/decryption for sensitive data
type EncryptionManager struct {
	key []byte
}

// NewEncryptionManager creates a new encryption manager with the provided key
func NewEncryptionManager(masterKey string) (*EncryptionManager, error) {
	if len(masterKey) < 32 {
		return nil, fmt.Errorf("%w: master key must be at least 32 characters", ErrInvalidKeySize)
	}

	// Use PBKDF2 to derive a 32-byte key from the master key
	salt := []byte("tiris-backend-salt-v1") // Fixed salt for consistent key derivation
	key := pbkdf2.Key([]byte(masterKey), salt, 100000, 32, sha256.New)

	return &EncryptionManager{
		key: key,
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (em *EncryptionManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(em.key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: failed to generate nonce: %v", ErrEncryptionFailed, err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (em *EncryptionManager) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64 encoding: %v", ErrInvalidCiphertext, err)
	}

	block, err := aes.NewCipher(em.key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	// Extract nonce and encrypted data
	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// MaskSensitiveData masks sensitive data for logging/display
func MaskSensitiveData(data string, showLength int) string {
	if data == "" {
		return ""
	}

	if len(data) <= showLength*2 {
		return "***"
	}

	if showLength <= 0 {
		showLength = 4
	}

	return data[:showLength] + strings.Repeat("*", len(data)-showLength*2) + data[len(data)-showLength:]
}

// GenerateSecureKey generates a cryptographically secure random key
func GenerateSecureKey(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("key length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure key: %v", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateKeyStrength validates if a key meets security requirements
func ValidateKeyStrength(key string) error {
	if len(key) < 32 {
		return errors.New("key must be at least 32 characters long")
	}

	// Check for basic complexity (at least one number, one lowercase, one uppercase)
	var hasUpper, hasLower, hasDigit bool
	for _, char := range key {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("key must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}