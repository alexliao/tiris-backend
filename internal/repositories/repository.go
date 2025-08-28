package repositories

import (
	"gorm.io/gorm"
)

// Repositories contains all repository instances
type Repositories struct {
	User            UserRepository
	OAuthToken      OAuthTokenRepository
	Trading         TradingRepository
	SubAccount      SubAccountRepository
	Transaction     TransactionRepository
	TradingLog      TradingLogRepository
	EventProcessing EventProcessingRepository
}

// NewRepositories creates a new repository container with all repositories
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:            NewUserRepository(db),
		OAuthToken:      NewOAuthTokenRepository(db),
		Trading:         NewTradingRepository(db),
		SubAccount:      NewSubAccountRepository(db),
		Transaction:     NewTransactionRepository(db),
		TradingLog:      NewTradingLogRepository(db),
		EventProcessing: NewEventProcessingRepository(db),
	}
}
