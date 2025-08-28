package repositories

import (
	"context"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.User, int64, error)
}

// OAuthTokenRepository defines the interface for OAuth token operations
type OAuthTokenRepository interface {
	Create(ctx context.Context, token *models.OAuthToken) error
	GetByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthToken, error)
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthToken, error)
	Update(ctx context.Context, token *models.OAuthToken) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// TradingRepository defines the interface for trading operations
type TradingRepository interface {
	Create(ctx context.Context, trading *models.Trading) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Trading, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Trading, error)
	Update(ctx context.Context, trading *models.Trading) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByUserIDAndType(ctx context.Context, userID uuid.UUID, tradingType string) ([]*models.Trading, error)
}

// SubAccountRepository defines the interface for sub-account operations
type SubAccountRepository interface {
	Create(ctx context.Context, subAccount *models.SubAccount) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SubAccount, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, tradingID *uuid.UUID) ([]*models.SubAccount, error)
	GetByTradingID(ctx context.Context, tradingID uuid.UUID) ([]*models.SubAccount, error)
	Update(ctx context.Context, subAccount *models.SubAccount) error
	UpdateBalance(ctx context.Context, subAccountID uuid.UUID, newBalance float64, amount float64, direction, reason string, info interface{}) (*uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetBySymbol(ctx context.Context, userID uuid.UUID, symbol string) ([]*models.SubAccount, error)
}

// TransactionRepository defines the interface for transaction operations
type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, filters TransactionFilters) ([]*models.Transaction, int64, error)
	GetBySubAccountID(ctx context.Context, subAccountID uuid.UUID, filters TransactionFilters) ([]*models.Transaction, int64, error)
	GetByTradingID(ctx context.Context, tradingID uuid.UUID, filters TransactionFilters) ([]*models.Transaction, int64, error)
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filters TransactionFilters) ([]*models.Transaction, int64, error)
}

// TradingLogRepository defines the interface for trading log operations
type TradingLogRepository interface {
	Create(ctx context.Context, log *models.TradingLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.TradingLog, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, filters TradingLogFilters) ([]*models.TradingLog, int64, error)
	GetBySubAccountID(ctx context.Context, subAccountID uuid.UUID, filters TradingLogFilters) ([]*models.TradingLog, int64, error)
	GetByTradingID(ctx context.Context, tradingID uuid.UUID, filters TradingLogFilters) ([]*models.TradingLog, int64, error)
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filters TradingLogFilters) ([]*models.TradingLog, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventProcessingRepository defines the interface for event processing operations
type EventProcessingRepository interface {
	Create(ctx context.Context, event *models.EventProcessing) error
	GetByEventID(ctx context.Context, eventID string) (*models.EventProcessing, error)
	GetByEventType(ctx context.Context, eventType string, filters EventProcessingFilters) ([]*models.EventProcessing, int64, error)
	Update(ctx context.Context, event *models.EventProcessing) error
	MarkAsProcessed(ctx context.Context, eventID string) error
	MarkAsFailed(ctx context.Context, eventID string, errorMessage string, retryCount int) error
	GetFailedEvents(ctx context.Context, maxRetries int) ([]*models.EventProcessing, error)
	DeleteOldEvents(ctx context.Context, olderThan time.Time) error
}

// Filter structs for complex queries
type TransactionFilters struct {
	Direction *string
	Reason    *string
	StartDate *time.Time
	EndDate   *time.Time
	MinAmount *float64
	MaxAmount *float64
	Limit     int
	Offset    int
}

type TradingLogFilters struct {
	Type      *string
	Source    *string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}

type EventProcessingFilters struct {
	Status    *string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}
