package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrading_Validation(t *testing.T) {
	userID := uuid.New()
	exchangeBindingID := uuid.New()

	tests := []struct {
		name        string
		trading     Trading
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid_real_trading",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "My BTC Trading",
				Type:              "real",
				Status:            "active",
			},
			shouldError: false,
		},
		{
			name: "valid_virtual_trading",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "Virtual Trading",
				Type:              "virtual",
				Status:            "active",
			},
			shouldError: false,
		},
		{
			name: "valid_backtest_trading",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "Backtest Strategy",
				Type:              "backtest",
				Status:            "active",
			},
			shouldError: false,
		},
		{
			name: "invalid_type",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "Invalid Type",
				Type:              "invalid_type",
				Status:            "active",
			},
			shouldError: true,
			errorMsg:    "invalid trading type",
		},
		{
			name: "invalid_status",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "Invalid Status",
				Type:              "real",
				Status:            "invalid_status",
			},
			shouldError: true,
			errorMsg:    "invalid status",
		},
		{
			name: "empty_name",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "",
				Type:              "real",
				Status:            "active",
			},
			shouldError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name: "nil_user_id",
			trading: Trading{
				UserID:            uuid.Nil,
				ExchangeBindingID: exchangeBindingID,
				Name:              "No User",
				Type:              "real",
				Status:            "active",
			},
			shouldError: true,
			errorMsg:    "user ID cannot be nil",
		},
		{
			name: "nil_exchange_binding_id",
			trading: Trading{
				UserID:            userID,
				ExchangeBindingID: uuid.Nil,
				Name:              "No Exchange Binding",
				Type:              "real",
				Status:            "active",
			},
			shouldError: true,
			errorMsg:    "exchange binding ID cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trading.Validate()
			
			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTrading_IsReal(t *testing.T) {
	tests := []struct {
		name     string
		trading  Trading
		expected bool
	}{
		{
			name: "real_trading",
			trading: Trading{
				Type: "real",
			},
			expected: true,
		},
		{
			name: "virtual_trading",
			trading: Trading{
				Type: "virtual",
			},
			expected: false,
		},
		{
			name: "backtest_trading",
			trading: Trading{
				Type: "backtest",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.trading.IsReal()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrading_IsVirtual(t *testing.T) {
	tests := []struct {
		name     string
		trading  Trading
		expected bool
	}{
		{
			name: "real_trading",
			trading: Trading{
				Type: "real",
			},
			expected: false,
		},
		{
			name: "virtual_trading",
			trading: Trading{
				Type: "virtual",
			},
			expected: true,
		},
		{
			name: "backtest_trading",
			trading: Trading{
				Type: "backtest",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.trading.IsVirtual()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrading_IsBacktest(t *testing.T) {
	tests := []struct {
		name     string
		trading  Trading
		expected bool
	}{
		{
			name: "real_trading",
			trading: Trading{
				Type: "real",
			},
			expected: false,
		},
		{
			name: "virtual_trading",
			trading: Trading{
				Type: "virtual",
			},
			expected: false,
		},
		{
			name: "backtest_trading",
			trading: Trading{
				Type: "backtest",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.trading.IsBacktest()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrading_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		trading  Trading
		expected bool
	}{
		{
			name: "active_trading",
			trading: Trading{
				Status: "active",
			},
			expected: true,
		},
		{
			name: "inactive_trading",
			trading: Trading{
				Status: "inactive",
			},
			expected: false,
		},
		{
			name: "paused_trading",
			trading: Trading{
				Status: "paused",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.trading.IsActive()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrading_ToResponse(t *testing.T) {
	userID := uuid.New()
	tradingID := uuid.New()
	exchangeBindingID := uuid.New()
	now := time.Now()

	// Create mock exchange binding
	exchangeBinding := ExchangeBinding{
		ID:       exchangeBindingID,
		Name:     "My Binance",
		Exchange: "binance",
		Type:     "private",
	}

	trading := Trading{
		ID:                tradingID,
		UserID:            userID,
		ExchangeBindingID: exchangeBindingID,
		Name:              "My BTC Trading",
		Type:              "real",
		Status:            "active",
		Info:              JSON{"strategy": "scalping"},
		CreatedAt:         now,
		UpdatedAt:         now,
		ExchangeBinding:   exchangeBinding,
	}

	response := trading.ToResponse()

	assert.Equal(t, tradingID, response.ID)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, exchangeBindingID, response.ExchangeBindingID)
	assert.Equal(t, "My BTC Trading", response.Name)
	assert.Equal(t, "real", response.Type)
	assert.Equal(t, "active", response.Status)
	assert.Equal(t, JSON{"strategy": "scalping"}, response.Info)
	assert.Equal(t, now, response.CreatedAt)
	assert.Equal(t, now, response.UpdatedAt)
	
	// Check exchange binding info is included
	assert.Equal(t, "My Binance", response.ExchangeBindingName)
	assert.Equal(t, "binance", response.Exchange)
	assert.Equal(t, "private", response.ExchangeBindingType)
}

func TestTrading_TableName(t *testing.T) {
	trading := Trading{}
	assert.Equal(t, "tradings", trading.TableName())
}

// Test the trading constants
func TestTradingConstants(t *testing.T) {
	// Test valid types
	validTypes := []string{
		TradingTypeReal,
		TradingTypeVirtual,
		TradingTypeBacktest,
	}

	for _, tradingType := range validTypes {
		assert.NotEmpty(t, tradingType)
	}

	// Test valid statuses
	validStatuses := []string{
		TradingStatusActive,
		TradingStatusInactive,
		TradingStatusPaused,
	}

	for _, status := range validStatuses {
		assert.NotEmpty(t, status)
	}
}

func TestCreateTradingRequest_Validate(t *testing.T) {
	userID := uuid.New()
	exchangeBindingID := uuid.New()

	tests := []struct {
		name        string
		request     CreateTradingRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid_request",
			request: CreateTradingRequest{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "My Trading",
				Type:              "real",
			},
			shouldError: false,
		},
		{
			name: "missing_name",
			request: CreateTradingRequest{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "",
				Type:              "real",
			},
			shouldError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing_type",
			request: CreateTradingRequest{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "My Trading",
				Type:              "",
			},
			shouldError: true,
			errorMsg:    "type is required",
		},
		{
			name: "invalid_type",
			request: CreateTradingRequest{
				UserID:            userID,
				ExchangeBindingID: exchangeBindingID,
				Name:              "My Trading",
				Type:              "invalid",
			},
			shouldError: true,
			errorMsg:    "invalid trading type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			
			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateTradingRequest_Validate(t *testing.T) {
	exchangeBindingID := uuid.New()

	tests := []struct {
		name        string
		request     UpdateTradingRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid_name_update",
			request: UpdateTradingRequest{
				Name: stringPtr("New Name"),
			},
			shouldError: false,
		},
		{
			name: "valid_exchange_binding_update",
			request: UpdateTradingRequest{
				ExchangeBindingID: &exchangeBindingID,
			},
			shouldError: false,
		},
		{
			name: "valid_status_update",
			request: UpdateTradingRequest{
				Status: stringPtr("inactive"),
			},
			shouldError: false,
		},
		{
			name: "empty_name",
			request: UpdateTradingRequest{
				Name: stringPtr(""),
			},
			shouldError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name: "invalid_status",
			request: UpdateTradingRequest{
				Status: stringPtr("invalid"),
			},
			shouldError: true,
			errorMsg:    "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			
			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}