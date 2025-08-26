package services

import "strings"

// isUniqueConstraintViolation checks if the error is a unique constraint violation
func isUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// Check for PostgreSQL unique constraint violation patterns
	return strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "violates unique constraint") ||
		strings.Contains(errStr, "already exists")
}

// getSpecificConstraintViolation returns specific error message based on constraint name in error
func getSpecificConstraintViolation(err error) string {
	if err == nil {
		return ""
	}
	errStr := strings.ToLower(err.Error())

	// Check for specific constraint violations by examining constraint names
	if strings.Contains(errStr, "exchanges_user_name_active_unique") {
		return "exchange name already exists"
	}
	if strings.Contains(errStr, "exchanges_user_api_key_active_unique") {
		return "api key already exists"
	}
	if strings.Contains(errStr, "exchanges_user_api_secret_active_unique") {
		return "api secret already exists"
	}
	if strings.Contains(errStr, "sub_accounts_exchange_name_active_unique") {
		return "sub-account name already exists for this exchange"
	}

	// Fallback for generic unique constraint violations
	if isUniqueConstraintViolation(err) {
		return "unique constraint violation"
	}

	return ""
}
