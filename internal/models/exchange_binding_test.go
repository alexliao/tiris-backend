package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeBinding_Validation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name        string
		binding     ExchangeBinding
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid_private_binding",
			binding: ExchangeBinding{
				UserID:    &userID,
				Name:      "My Binance",
				Exchange:  "binance",
				Type:      "private",
				APIKey:    "test_api_key",
				APISecret: "test_api_secret",
				Status:    "active",
			},
			shouldError: false,
		},
		{
			name: "valid_public_binding",
			binding: ExchangeBinding{
				UserID:    nil,
				Name:      "Public Binance",
				Exchange:  "binance",
				Type:      "public",
				APIKey:    "",
				APISecret: "",
				Status:    "active",
			},
			shouldError: false,
		},
		{
			name: "invalid_private_binding_no_user",
			binding: ExchangeBinding{
				UserID:    nil,
				Name:      "Invalid Private",
				Exchange:  "binance",
				Type:      "private",
				APIKey:    "test_key",
				APISecret: "test_secret",
				Status:    "active",
			},
			shouldError: true,
			errorMsg:    "private bindings must have a user",
		},
		{
			name: "invalid_private_binding_no_api_key",
			binding: ExchangeBinding{
				UserID:    &userID,
				Name:      "No API Key",
				Exchange:  "binance",
				Type:      "private",
				APIKey:    "",
				APISecret: "test_secret",
				Status:    "active",
			},
			shouldError: true,
			errorMsg:    "private bindings must have API credentials",
		},
		{
			name: "invalid_exchange",
			binding: ExchangeBinding{
				UserID:    &userID,
				Name:      "Invalid Exchange",
				Exchange:  "invalid_exchange",
				Type:      "private",
				APIKey:    "test_key",
				APISecret: "test_secret",
				Status:    "active",
			},
			shouldError: true,
			errorMsg:    "invalid exchange",
		},
		{
			name: "invalid_type",
			binding: ExchangeBinding{
				UserID:    &userID,
				Name:      "Invalid Type",
				Exchange:  "binance",
				Type:      "invalid_type",
				APIKey:    "test_key",
				APISecret: "test_secret",
				Status:    "active",
			},
			shouldError: true,
			errorMsg:    "invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.binding.Validate()
			
			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExchangeBinding_IsPrivate(t *testing.T) {
	tests := []struct {
		name     string
		binding  ExchangeBinding
		expected bool
	}{
		{
			name: "private_binding",
			binding: ExchangeBinding{
				Type: "private",
			},
			expected: true,
		},
		{
			name: "public_binding",
			binding: ExchangeBinding{
				Type: "public",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.binding.IsPrivate()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExchangeBinding_IsPublic(t *testing.T) {
	tests := []struct {
		name     string
		binding  ExchangeBinding
		expected bool
	}{
		{
			name: "private_binding",
			binding: ExchangeBinding{
				Type: "private",
			},
			expected: false,
		},
		{
			name: "public_binding",
			binding: ExchangeBinding{
				Type: "public",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.binding.IsPublic()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExchangeBinding_HasCredentials(t *testing.T) {
	tests := []struct {
		name     string
		binding  ExchangeBinding
		expected bool
	}{
		{
			name: "has_credentials",
			binding: ExchangeBinding{
				APIKey:    "test_key",
				APISecret: "test_secret",
			},
			expected: true,
		},
		{
			name: "no_api_key",
			binding: ExchangeBinding{
				APIKey:    "",
				APISecret: "test_secret",
			},
			expected: false,
		},
		{
			name: "no_api_secret",
			binding: ExchangeBinding{
				APIKey:    "test_key",
				APISecret: "",
			},
			expected: false,
		},
		{
			name: "no_credentials",
			binding: ExchangeBinding{
				APIKey:    "",
				APISecret: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.binding.HasCredentials()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExchangeBinding_GetMaskedAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "normal_key",
			apiKey:   "abcdefghijklmnop",
			expected: "abcd...mnop",
		},
		{
			name:     "short_key",
			apiKey:   "abc",
			expected: "***",
		},
		{
			name:     "empty_key",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "long_key",
			apiKey:   "abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "abcd...7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := ExchangeBinding{
				APIKey: tt.apiKey,
			}
			result := binding.GetMaskedAPIKey()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExchangeBinding_ToResponse(t *testing.T) {
	userID := uuid.New()
	bindingID := uuid.New()
	now := time.Now()

	binding := ExchangeBinding{
		ID:        bindingID,
		UserID:    &userID,
		Name:      "My Binance",
		Exchange:  "binance",
		Type:      "private",
		APIKey:    "test_api_key_12345",
		APISecret: "test_api_secret",
		Status:    "active",
		Info:      JSON{"test": "data"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := binding.ToResponse()

	assert.Equal(t, bindingID, response.ID)
	assert.Equal(t, &userID, response.UserID)
	assert.Equal(t, "My Binance", response.Name)
	assert.Equal(t, "binance", response.Exchange)
	assert.Equal(t, "private", response.Type)
	assert.Equal(t, "test...2345", response.MaskedAPIKey)
	assert.Equal(t, "active", response.Status)
	assert.Equal(t, JSON{"test": "data"}, response.Info)
	assert.Equal(t, now, response.CreatedAt)
	assert.Equal(t, now, response.UpdatedAt)
}

func TestExchangeBinding_TableName(t *testing.T) {
	binding := ExchangeBinding{}
	assert.Equal(t, "exchange_bindings", binding.TableName())
}

// Test the exchange binding constants
func TestExchangeBindingConstants(t *testing.T) {
	// Test valid exchanges
	validExchanges := []string{
		ExchangeBinance,
		ExchangeKraken,
		ExchangeGate,
		ExchangeCoinbase,
		ExchangeVirtual,
	}

	for _, exchange := range validExchanges {
		assert.NotEmpty(t, exchange)
	}

	// Test valid types
	assert.Equal(t, "private", ExchangeBindingTypePrivate)
	assert.Equal(t, "public", ExchangeBindingTypePublic)

	// Test valid statuses
	validStatuses := []string{
		ExchangeBindingStatusActive,
		ExchangeBindingStatusInactive,
		ExchangeBindingStatusError,
	}

	for _, status := range validStatuses {
		assert.NotEmpty(t, status)
	}
}