package test

import (
	"testing"

	"tiris-backend/internal/services"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTradingLogValidator_ValidateType tests trading log type validation
func TestTradingLogValidator_ValidateType(t *testing.T) {
	validator := services.NewTradingLogValidator()

	testCases := []struct {
		name        string
		logType     string
		expectError bool
	}{
		{"valid_long", "long", false},
		{"valid_short", "short", false},
		{"valid_stop_loss", "stop_loss", false},
		{"valid_deposit", "deposit", false},
		{"valid_withdraw", "withdraw", false},
		{"valid_other_type", "trade", false},
		{"valid_strategy_type", "strategy", false},
		{"valid_market_type", "market", false},
		{"empty_type", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateType(tc.logType)

			if tc.expectError {
				require.Error(t, err)
				validationErr, ok := err.(*services.ValidationError)
				require.True(t, ok, "Expected ValidationError")
				assert.Equal(t, "type", validationErr.Field)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestTradingLogValidator_ValidateInfoStructure tests info field validation
func TestTradingLogValidator_ValidateInfoStructure(t *testing.T) {
	validator := services.NewTradingLogValidator()

	t.Run("valid_long_position_info", func(t *testing.T) {
		info := map[string]interface{}{
			"stock_account_id":    uuid.New().String(),
			"currency_account_id": uuid.New().String(),
			"price":               3000.0,
			"volume":              2.5,
			"stock":               "ETH",
			"currency":            "USDT",
			"fee":                 12.5,
		}

		tradingInfo, err := validator.ValidateInfoStructure(info, "long")

		require.NoError(t, err)
		require.NotNil(t, tradingInfo)
		assert.Equal(t, 3000.0, tradingInfo.Price)
		assert.Equal(t, 2.5, tradingInfo.Volume)
		assert.Equal(t, "ETH", tradingInfo.Stock)
		assert.Equal(t, "USDT", tradingInfo.Currency)
		assert.Equal(t, 12.5, tradingInfo.Fee)
	})

	t.Run("non_business_logic_type", func(t *testing.T) {
		info := map[string]interface{}{
			"some_field": "some_value",
		}

		tradingInfo, err := validator.ValidateInfoStructure(info, "trade")

		require.NoError(t, err)
		assert.Nil(t, tradingInfo) // Should return nil for non-business logic types
	})

	t.Run("missing_required_fields", func(t *testing.T) {
		testCases := []struct {
			name         string
			info         map[string]interface{}
			missingField string
		}{
			{
				"missing_stock_account_id",
				map[string]interface{}{
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"stock_account_id",
			},
			{
				"missing_currency_account_id",
				map[string]interface{}{
					"stock_account_id": uuid.New().String(),
					"price":            3000.0,
					"volume":           2.0,
					"stock":            "ETH",
					"currency":         "USDT",
					"fee":              12.0,
				},
				"currency_account_id",
			},
			{
				"missing_price",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"price",
			},
			{
				"missing_volume",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"volume",
			},
			{
				"missing_stock",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"stock",
			},
			{
				"missing_currency",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"fee":                 12.0,
				},
				"currency",
			},
			{
				"missing_fee",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "USDT",
				},
				"fee",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tradingInfo, err := validator.ValidateInfoStructure(tc.info, "long")

				require.Error(t, err)
				assert.Nil(t, tradingInfo)

				validationErr, ok := err.(*services.ValidationError)
				require.True(t, ok, "Expected ValidationError")
				assert.Equal(t, tc.missingField, validationErr.Field)
				assert.Contains(t, validationErr.Message, "is required")
			})
		}
	})

	t.Run("invalid_field_values", func(t *testing.T) {
		testCases := []struct {
			name          string
			info          map[string]interface{}
			expectedField string
			expectedMsg   string
		}{
			{
				"invalid_stock_account_id",
				map[string]interface{}{
					"stock_account_id":    "invalid-uuid",
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"stock_account_id",
				"must be a valid UUID",
			},
			{
				"negative_price",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               -1000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"price",
				"must be a positive number",
			},
			{
				"zero_volume",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              0.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"volume",
				"must be a positive number",
			},
			{
				"negative_fee",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "USDT",
					"fee":                 -5.0,
				},
				"fee",
				"must be a non-negative number",
			},
			{
				"empty_stock_symbol",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "",
					"currency":            "USDT",
					"fee":                 12.0,
				},
				"stock",
				"must be a non-empty string",
			},
			{
				"long_currency_symbol",
				map[string]interface{}{
					"stock_account_id":    uuid.New().String(),
					"currency_account_id": uuid.New().String(),
					"price":               3000.0,
					"volume":              2.0,
					"stock":               "ETH",
					"currency":            "VERYLONGCURRENCYSYMBOL",
					"fee":                 12.0,
				},
				"currency",
				"maximum 20 characters",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tradingInfo, err := validator.ValidateInfoStructure(tc.info, "long")

				require.Error(t, err)
				assert.Nil(t, tradingInfo)

				validationErr, ok := err.(*services.ValidationError)
				require.True(t, ok, "Expected ValidationError")
				assert.Equal(t, tc.expectedField, validationErr.Field)
				assert.Contains(t, validationErr.Message, tc.expectedMsg)
			})
		}
	})

	t.Run("same_accounts", func(t *testing.T) {
		sameAccountID := uuid.New().String()
		info := map[string]interface{}{
			"stock_account_id":    sameAccountID,
			"currency_account_id": sameAccountID,
			"price":               3000.0,
			"volume":              2.0,
			"stock":               "ETH",
			"currency":            "USDT",
			"fee":                 12.0,
		}

		tradingInfo, err := validator.ValidateInfoStructure(info, "long")

		require.Error(t, err)
		assert.Nil(t, tradingInfo)

		validationErr, ok := err.(*services.ValidationError)
		require.True(t, ok, "Expected ValidationError")
		assert.Equal(t, "accounts", validationErr.Field)
		assert.Contains(t, validationErr.Message, "must be different")
	})

	t.Run("numeric_type_conversion", func(t *testing.T) {
		info := map[string]interface{}{
			"stock_account_id":    uuid.New().String(),
			"currency_account_id": uuid.New().String(),
			"price":               int64(3000),  // Test int to float conversion
			"volume":              float32(2.5), // Test float32 to float64 conversion
			"stock":               "ETH",
			"currency":            "USDT",
			"fee":                 int(12), // Test int to float conversion
		}

		tradingInfo, err := validator.ValidateInfoStructure(info, "short")

		require.NoError(t, err)
		require.NotNil(t, tradingInfo)
		assert.Equal(t, 3000.0, tradingInfo.Price)
		assert.Equal(t, float64(2.5), tradingInfo.Volume)
		assert.Equal(t, 12.0, tradingInfo.Fee)
	})

	t.Run("valid_deposit_info", func(t *testing.T) {
		accountID := uuid.New()
		info := map[string]interface{}{
			"account_id": accountID.String(),
			"amount":     1000.50,
			"currency":   "USDT",
		}

		tradingInfo, err := validator.ValidateInfoStructure(info, "deposit")

		require.NoError(t, err)
		assert.NotNil(t, tradingInfo)
		assert.Equal(t, accountID, tradingInfo.StockAccountID) // Account ID maps to StockAccountID
		assert.Equal(t, 1000.50, tradingInfo.Volume)           // Amount maps to Volume
		assert.Equal(t, "USDT", tradingInfo.Stock)             // Currency maps to Stock
		assert.Equal(t, 1.0, tradingInfo.Price)                // Default price
		assert.Equal(t, 0.0, tradingInfo.Fee)                  // No fees for deposits
	})

	t.Run("valid_withdraw_info", func(t *testing.T) {
		accountID := uuid.New()
		info := map[string]interface{}{
			"account_id": accountID.String(),
			"amount":     500.25,
			"currency":   "BTC",
		}

		tradingInfo, err := validator.ValidateInfoStructure(info, "withdraw")

		require.NoError(t, err)
		assert.NotNil(t, tradingInfo)
		assert.Equal(t, accountID, tradingInfo.StockAccountID) // Account ID maps to StockAccountID
		assert.Equal(t, 500.25, tradingInfo.Volume)            // Amount maps to Volume
		assert.Equal(t, "BTC", tradingInfo.Stock)              // Currency maps to Stock
		assert.Equal(t, 1.0, tradingInfo.Price)                // Default price
		assert.Equal(t, 0.0, tradingInfo.Fee)                  // No fees for withdrawals
	})

	t.Run("deposit_withdraw_validation_errors", func(t *testing.T) {
		testCases := []struct {
			name          string
			info          map[string]interface{}
			logType       string
			expectedError string
		}{
			{
				"deposit_missing_account_id",
				map[string]interface{}{"amount": 100.0, "currency": "USD"},
				"deposit",
				"account_id",
			},
			{
				"withdraw_missing_amount",
				map[string]interface{}{"account_id": uuid.New().String(), "currency": "USD"},
				"withdraw",
				"amount",
			},
			{
				"deposit_missing_currency",
				map[string]interface{}{"account_id": uuid.New().String(), "amount": 100.0},
				"deposit",
				"currency",
			},
			{
				"withdraw_invalid_account_id",
				map[string]interface{}{"account_id": "invalid-uuid", "amount": 100.0, "currency": "USD"},
				"withdraw",
				"account_id",
			},
			{
				"deposit_negative_amount",
				map[string]interface{}{"account_id": uuid.New().String(), "amount": -100.0, "currency": "USD"},
				"deposit",
				"amount",
			},
			{
				"withdraw_zero_amount",
				map[string]interface{}{"account_id": uuid.New().String(), "amount": 0.0, "currency": "USD"},
				"withdraw",
				"amount",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := validator.ValidateInfoStructure(tc.info, tc.logType)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			})
		}
	})
}

// TestTradingLogValidator_ValidateDecimalPrecision tests decimal precision validation
func TestTradingLogValidator_ValidateDecimalPrecision(t *testing.T) {
	validator := services.NewTradingLogValidator()

	testCases := []struct {
		name        string
		value       float64
		maxDecimals int
		expectError bool
	}{
		{"valid_2_decimals", 123.45, 2, false},
		{"valid_8_decimals", 123.12345678, 8, false},
		{"valid_no_decimals", 123.0, 8, false},
		{"valid_trailing_zeros", 123.450000, 8, false},
		{"invalid_too_many_decimals", 123.123456789, 8, true},
		{"invalid_3_decimals_max_2", 123.456, 2, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateDecimalPrecision(tc.value, "test_field", tc.maxDecimals)

			if tc.expectError {
				require.Error(t, err)
				validationErr, ok := err.(*services.ValidationError)
				require.True(t, ok, "Expected ValidationError")
				assert.Equal(t, "test_field", validationErr.Field)
				assert.Contains(t, validationErr.Message, "decimal places")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
