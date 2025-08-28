package mocks

import (
	"context"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) Disable(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// Note: UserStats doesn't exist in models, this method might not be used

// MockTradingRepository is a mock implementation of ExchangeRepository
type MockTradingRepository struct {
	mock.Mock
}

func (m *MockTradingRepository) Create(ctx context.Context, exchange *models.Trading) error {
	args := m.Called(ctx, exchange)
	return args.Error(0)
}

func (m *MockTradingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Trading, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trading), args.Error(1)
}

func (m *MockTradingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Trading, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Trading), args.Error(1)
}

func (m *MockTradingRepository) Update(ctx context.Context, exchange *models.Trading) error {
	args := m.Called(ctx, exchange)
	return args.Error(0)
}

func (m *MockTradingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTradingRepository) GetByUserIDAndType(ctx context.Context, userID uuid.UUID, exchangeType string) ([]*models.Trading, error) {
	args := m.Called(ctx, userID, exchangeType)
	return args.Get(0).([]*models.Trading), args.Error(1)
}

// MockSubAccountRepository is a mock implementation of SubAccountRepository
type MockSubAccountRepository struct {
	mock.Mock
}

func (m *MockSubAccountRepository) Create(ctx context.Context, subAccount *models.SubAccount) error {
	args := m.Called(ctx, subAccount)
	return args.Error(0)
}

func (m *MockSubAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SubAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SubAccount), args.Error(1)
}

func (m *MockSubAccountRepository) GetByUserID(ctx context.Context, userID uuid.UUID, tradingID *uuid.UUID) ([]*models.SubAccount, error) {
	args := m.Called(ctx, userID, tradingID)
	return args.Get(0).([]*models.SubAccount), args.Error(1)
}

func (m *MockSubAccountRepository) GetByTradingID(ctx context.Context, tradingID uuid.UUID) ([]*models.SubAccount, error) {
	args := m.Called(ctx, tradingID)
	return args.Get(0).([]*models.SubAccount), args.Error(1)
}

func (m *MockSubAccountRepository) Update(ctx context.Context, subAccount *models.SubAccount) error {
	args := m.Called(ctx, subAccount)
	return args.Error(0)
}

func (m *MockSubAccountRepository) UpdateBalance(ctx context.Context, subAccountID uuid.UUID, newBalance float64, amount float64, direction, reason string, info interface{}) (*uuid.UUID, error) {
	args := m.Called(ctx, subAccountID, newBalance, amount, direction, reason, info)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

func (m *MockSubAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSubAccountRepository) GetBySymbol(ctx context.Context, userID uuid.UUID, symbol string) ([]*models.SubAccount, error) {
	args := m.Called(ctx, userID, symbol)
	return args.Get(0).([]*models.SubAccount), args.Error(1)
}

// MockTransactionRepository is a mock implementation of TransactionRepository
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters repositories.TransactionFilters) ([]*models.Transaction, int64, error) {
	args := m.Called(ctx, userID, filters)
	return args.Get(0).([]*models.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionRepository) GetBySubAccountID(ctx context.Context, subAccountID uuid.UUID, filters repositories.TransactionFilters) ([]*models.Transaction, int64, error) {
	args := m.Called(ctx, subAccountID, filters)
	return args.Get(0).([]*models.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionRepository) GetByTradingID(ctx context.Context, tradingID uuid.UUID, filters repositories.TransactionFilters) ([]*models.Transaction, int64, error) {
	args := m.Called(ctx, tradingID, filters)
	return args.Get(0).([]*models.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionRepository) GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filters repositories.TransactionFilters) ([]*models.Transaction, int64, error) {
	args := m.Called(ctx, startTime, endTime, filters)
	return args.Get(0).([]*models.Transaction), args.Get(1).(int64), args.Error(2)
}

// MockTradingLogRepository is a mock implementation of TradingLogRepository
type MockTradingLogRepository struct {
	mock.Mock
}

func (m *MockTradingLogRepository) Create(ctx context.Context, tradingLog *models.TradingLog) error {
	args := m.Called(ctx, tradingLog)
	return args.Error(0)
}

func (m *MockTradingLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TradingLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TradingLog), args.Error(1)
}

func (m *MockTradingLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters repositories.TradingLogFilters) ([]*models.TradingLog, int64, error) {
	args := m.Called(ctx, userID, filters)
	return args.Get(0).([]*models.TradingLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockTradingLogRepository) GetBySubAccountID(ctx context.Context, subAccountID uuid.UUID, filters repositories.TradingLogFilters) ([]*models.TradingLog, int64, error) {
	args := m.Called(ctx, subAccountID, filters)
	return args.Get(0).([]*models.TradingLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockTradingLogRepository) GetByTradingID(ctx context.Context, tradingID uuid.UUID, filters repositories.TradingLogFilters) ([]*models.TradingLog, int64, error) {
	args := m.Called(ctx, tradingID, filters)
	return args.Get(0).([]*models.TradingLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockTradingLogRepository) GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filters repositories.TradingLogFilters) ([]*models.TradingLog, int64, error) {
	args := m.Called(ctx, startTime, endTime, filters)
	return args.Get(0).([]*models.TradingLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockTradingLogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockOAuthTokenRepository is a mock implementation of OAuthTokenRepository
type MockOAuthTokenRepository struct {
	mock.Mock
}

func (m *MockOAuthTokenRepository) Create(ctx context.Context, token *models.OAuthToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockOAuthTokenRepository) GetByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthToken, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthToken), args.Error(1)
}

func (m *MockOAuthTokenRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthToken, error) {
	args := m.Called(ctx, provider, providerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OAuthToken), args.Error(1)
}

func (m *MockOAuthTokenRepository) Update(ctx context.Context, token *models.OAuthToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockOAuthTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOAuthTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockEventProcessingRepository is a mock implementation of EventProcessingRepository
type MockEventProcessingRepository struct {
	mock.Mock
}

func (m *MockEventProcessingRepository) Create(ctx context.Context, event *models.EventProcessing) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProcessingRepository) GetByEventID(ctx context.Context, eventID string) (*models.EventProcessing, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EventProcessing), args.Error(1)
}

func (m *MockEventProcessingRepository) GetByEventType(ctx context.Context, eventType string, filters repositories.EventProcessingFilters) ([]*models.EventProcessing, int64, error) {
	args := m.Called(ctx, eventType, filters)
	return args.Get(0).([]*models.EventProcessing), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventProcessingRepository) Update(ctx context.Context, event *models.EventProcessing) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProcessingRepository) MarkAsProcessed(ctx context.Context, eventID string) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

func (m *MockEventProcessingRepository) MarkAsFailed(ctx context.Context, eventID string, errorMessage string, retryCount int) error {
	args := m.Called(ctx, eventID, errorMessage, retryCount)
	return args.Error(0)
}

func (m *MockEventProcessingRepository) GetFailedEvents(ctx context.Context, maxRetries int) ([]*models.EventProcessing, error) {
	args := m.Called(ctx, maxRetries)
	return args.Get(0).([]*models.EventProcessing), args.Error(1)
}

func (m *MockEventProcessingRepository) DeleteOldEvents(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

// MockJWTManager is a mock implementation of JWTManagerInterface
type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) GenerateToken(userID uuid.UUID, username, email, role string) (string, error) {
	args := m.Called(userID, username, email, role)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) GenerateTokenPair(userID uuid.UUID, username, email, role string) (*auth.TokenPair, error) {
	args := m.Called(userID, username, email, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockJWTManager) ValidateToken(tokenString string) (*auth.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Claims), args.Error(1)
}

func (m *MockJWTManager) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	args := m.Called(tokenString)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockJWTManager) RefreshToken(refreshToken, username, email, role string) (string, error) {
	args := m.Called(refreshToken, username, email, role)
	return args.String(0), args.Error(1)
}

// MockOAuthManager is a mock implementation of OAuthManagerInterface
type MockOAuthManager struct {
	mock.Mock
}

func (m *MockOAuthManager) GetAuthURL(provider auth.OAuthProvider, state string) (string, error) {
	args := m.Called(provider, state)
	return args.String(0), args.Error(1)
}

func (m *MockOAuthManager) ExchangeCodeForToken(provider auth.OAuthProvider, code string) (*oauth2.Token, error) {
	args := m.Called(provider, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *MockOAuthManager) GetUserInfo(provider auth.OAuthProvider, token *oauth2.Token) (*auth.OAuthUser, error) {
	args := m.Called(provider, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.OAuthUser), args.Error(1)
}

// MockExchangeService is a mock implementation of ExchangeServiceInterface
type MockExchangeService struct {
	mock.Mock
}

func (m *MockExchangeService) CreateExchange(ctx context.Context, userID uuid.UUID, req *services.CreateTradingRequest) (*services.TradingResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TradingResponse), args.Error(1)
}

func (m *MockExchangeService) GetUserExchanges(ctx context.Context, userID uuid.UUID) ([]*services.TradingResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*services.TradingResponse), args.Error(1)
}

func (m *MockExchangeService) GetExchange(ctx context.Context, userID, tradingID uuid.UUID) (*services.TradingResponse, error) {
	args := m.Called(ctx, userID, tradingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TradingResponse), args.Error(1)
}

func (m *MockExchangeService) UpdateExchange(ctx context.Context, userID, tradingID uuid.UUID, req *services.UpdateTradingRequest) (*services.TradingResponse, error) {
	args := m.Called(ctx, userID, tradingID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TradingResponse), args.Error(1)
}

func (m *MockExchangeService) DeleteExchange(ctx context.Context, userID, tradingID uuid.UUID) error {
	args := m.Called(ctx, userID, tradingID)
	return args.Error(0)
}

func (m *MockExchangeService) ListExchanges(ctx context.Context, limit, offset int) ([]*services.TradingResponse, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*services.TradingResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockExchangeService) GetExchangeByID(ctx context.Context, tradingID uuid.UUID) (*services.TradingResponse, error) {
	args := m.Called(ctx, tradingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TradingResponse), args.Error(1)
}

// MockRepositories combines all mock repositories
type MockRepositories struct {
	User            repositories.UserRepository
	Exchange        repositories.TradingRepository
	SubAccount      repositories.SubAccountRepository
	Transaction     repositories.TransactionRepository
	TradingLog      repositories.TradingLogRepository
	OAuthToken      repositories.OAuthTokenRepository
	EventProcessing repositories.EventProcessingRepository
}

// NewMockRepositories creates a new mock repositories instance
func NewMockRepositories() *MockRepositories {
	return &MockRepositories{
		User:            &MockUserRepository{},
		Exchange:        &MockTradingRepository{},
		SubAccount:      &MockSubAccountRepository{},
		Transaction:     &MockTransactionRepository{},
		TradingLog:      &MockTradingLogRepository{},
		OAuthToken:      &MockOAuthTokenRepository{},
		EventProcessing: &MockEventProcessingRepository{},
	}
}
