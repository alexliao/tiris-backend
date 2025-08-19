package services

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// TradingLogInfo represents the structured info field for trading logs
type TradingLogInfo struct {
	StockAccountID    uuid.UUID `json:"stock_account_id" binding:"required"`
	CurrencyAccountID uuid.UUID `json:"currency_account_id" binding:"required"`
	Price             float64   `json:"price" binding:"required,gt=0"`
	Volume            float64   `json:"volume" binding:"required,gt=0"`
	Stock             string    `json:"stock" binding:"required,min=1,max=20"`
	Currency          string    `json:"currency" binding:"required,min=1,max=20"`
	Fee               float64   `json:"fee" binding:"gte=0"`
}

// ValidationError represents a trading log validation error
type ValidationError struct {
	Field   string
	Message string
	Type    string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s in %s: %s", e.Field, e.Type, e.Message)
}

// TradingLogValidator handles validation of trading log data
type TradingLogValidator struct{}

// NewTradingLogValidator creates a new trading log validator
func NewTradingLogValidator() *TradingLogValidator {
	return &TradingLogValidator{}
}

// ValidateType validates if the trading log type is supported for business logic processing
// This method currently allows all types to pass through - restriction can be added later if needed
func (v *TradingLogValidator) ValidateType(logType string) error {
	// For now, allow all non-empty types
	// Business logic validation will occur in ValidateInfoStructure for specific types
	if len(logType) == 0 {
		return &ValidationError{
			Field:   "type",
			Message: "type cannot be empty",
			Type:    "trading_log",
		}
	}

	return nil
}

// ValidateInfoStructure validates the info field structure for trading operations
func (v *TradingLogValidator) ValidateInfoStructure(info map[string]interface{}, logType string) (*TradingLogInfo, error) {
	// Check if this is a trading operation type that requires structured info
	if !v.isBusinessLogicType(logType) {
		// For non-business logic types, we don't validate the structure
		return nil, nil
	}

	// Extract and validate required fields
	tradingInfo := &TradingLogInfo{}

	// Validate stock_account_id
	if stockAccountIDRaw, exists := info["stock_account_id"]; exists {
		if stockAccountIDStr, ok := stockAccountIDRaw.(string); ok {
			stockAccountID, err := uuid.Parse(stockAccountIDStr)
			if err != nil {
				return nil, &ValidationError{
					Field:   "stock_account_id",
					Message: "must be a valid UUID",
					Type:    logType,
				}
			}
			tradingInfo.StockAccountID = stockAccountID
		} else {
			return nil, &ValidationError{
				Field:   "stock_account_id",
				Message: "must be a string UUID",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "stock_account_id",
			Message: "is required",
			Type:    logType,
		}
	}

	// Validate currency_account_id
	if currencyAccountIDRaw, exists := info["currency_account_id"]; exists {
		if currencyAccountIDStr, ok := currencyAccountIDRaw.(string); ok {
			currencyAccountID, err := uuid.Parse(currencyAccountIDStr)
			if err != nil {
				return nil, &ValidationError{
					Field:   "currency_account_id",
					Message: "must be a valid UUID",
					Type:    logType,
				}
			}
			tradingInfo.CurrencyAccountID = currencyAccountID
		} else {
			return nil, &ValidationError{
				Field:   "currency_account_id",
				Message: "must be a string UUID",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "currency_account_id",
			Message: "is required",
			Type:    logType,
		}
	}

	// Validate price
	if priceRaw, exists := info["price"]; exists {
		if price, ok := v.extractFloat64(priceRaw); ok && price > 0 {
			tradingInfo.Price = price
		} else {
			return nil, &ValidationError{
				Field:   "price",
				Message: "must be a positive number",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "price",
			Message: "is required",
			Type:    logType,
		}
	}

	// Validate volume
	if volumeRaw, exists := info["volume"]; exists {
		if volume, ok := v.extractFloat64(volumeRaw); ok && volume > 0 {
			tradingInfo.Volume = volume
		} else {
			return nil, &ValidationError{
				Field:   "volume",
				Message: "must be a positive number",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "volume",
			Message: "is required",
			Type:    logType,
		}
	}

	// Validate stock
	if stockRaw, exists := info["stock"]; exists {
		if stock, ok := stockRaw.(string); ok && len(stock) > 0 && len(stock) <= 20 {
			tradingInfo.Stock = stock
		} else {
			return nil, &ValidationError{
				Field:   "stock",
				Message: "must be a non-empty string with maximum 20 characters",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "stock",
			Message: "is required",
			Type:    logType,
		}
	}

	// Validate currency
	if currencyRaw, exists := info["currency"]; exists {
		if currency, ok := currencyRaw.(string); ok && len(currency) > 0 && len(currency) <= 20 {
			tradingInfo.Currency = currency
		} else {
			return nil, &ValidationError{
				Field:   "currency",
				Message: "must be a non-empty string with maximum 20 characters",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "currency",
			Message: "is required",
			Type:    logType,
		}
	}

	// Validate fee
	if feeRaw, exists := info["fee"]; exists {
		if fee, ok := v.extractFloat64(feeRaw); ok && fee >= 0 {
			tradingInfo.Fee = fee
		} else {
			return nil, &ValidationError{
				Field:   "fee",
				Message: "must be a non-negative number",
				Type:    logType,
			}
		}
	} else {
		return nil, &ValidationError{
			Field:   "fee",
			Message: "is required",
			Type:    logType,
		}
	}

	// Additional validation: ensure stock and currency accounts are different
	if tradingInfo.StockAccountID == tradingInfo.CurrencyAccountID {
		return nil, &ValidationError{
			Field:   "accounts",
			Message: "stock_account_id and currency_account_id must be different",
			Type:    logType,
		}
	}

	return tradingInfo, nil
}

// ValidateDecimalPrecision ensures financial values have appropriate precision
func (v *TradingLogValidator) ValidateDecimalPrecision(value float64, fieldName string, maxDecimals int) error {
	// For financial validation, use a more practical approach
	// Convert with the maximum precision we care about plus one to detect excess
	formatStr := fmt.Sprintf("%%.%df", maxDecimals+3)
	valueStr := fmt.Sprintf(formatStr, value)
	
	// Find decimal point
	decimalIndex := -1
	for i, char := range valueStr {
		if char == '.' {
			decimalIndex = i
			break
		}
	}
	
	if decimalIndex != -1 && decimalIndex < len(valueStr)-1 {
		// Get the decimal part
		decimalPart := valueStr[decimalIndex+1:]
		
		// Remove trailing zeros
		decimalPart = trimTrailingZeros(decimalPart)
		
		// Count significant decimal places
		decimals := len(decimalPart)
		
		if decimals > maxDecimals {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("must have at most %d decimal places", maxDecimals),
				Type:    "precision",
			}
		}
	}
	
	return nil
}

// trimTrailingZeros removes trailing zeros from a string
func trimTrailingZeros(s string) string {
	// Remove trailing zeros
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	return s
}

// isBusinessLogicType checks if the type requires business logic processing
func (v *TradingLogValidator) isBusinessLogicType(logType string) bool {
	businessLogicTypes := map[string]bool{
		"long":      true,
		"short":     true,
		"stop_loss": true,
	}
	return businessLogicTypes[logType]
}

// extractFloat64 safely extracts float64 from interface{} with support for int and float types
func (v *TradingLogValidator) extractFloat64(value interface{}) (float64, bool) {
	val := reflect.ValueOf(value)
	
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), true
	default:
		return 0, false
	}
}