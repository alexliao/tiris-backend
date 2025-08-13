package repositories

import "errors"

// Common repository errors
var (
	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidUser       = errors.New("invalid user data")

	// Exchange errors
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyExists = errors.New("exchange already exists")
	ErrInvalidExchange       = errors.New("invalid exchange data")

	// Sub-account errors
	ErrSubAccountNotFound      = errors.New("sub-account not found")
	ErrSubAccountAlreadyExists = errors.New("sub-account already exists")
	ErrInvalidSubAccount       = errors.New("invalid sub-account data")

	// Transaction errors
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidTransaction  = errors.New("invalid transaction data")

	// Trading log errors
	ErrTradingLogNotFound = errors.New("trading log not found")
	ErrInvalidTradingLog  = errors.New("invalid trading log data")

	// OAuth token errors
	ErrOAuthTokenNotFound = errors.New("OAuth token not found")
	ErrInvalidOAuthToken  = errors.New("invalid OAuth token data")

	// Event processing errors
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidEvent  = errors.New("invalid event data")

	// General errors
	ErrDatabaseConnection = errors.New("database connection error")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInternalError      = errors.New("internal server error")
)
