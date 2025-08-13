package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/fixtures"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
)

type TransactionServiceTestSuite struct {
	suite.Suite
	mockRepos          *repositories.Repositories
	transactionService *TransactionService
	ctx                context.Context
}

func (suite *TransactionServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()

	// Create mock repositories
	suite.mockRepos = &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service with mocked dependencies
	suite.transactionService = NewTransactionService(suite.mockRepos)
}

func (suite *TransactionServiceTestSuite) TestGetUserTransactions_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &TransactionQueryRequest{
		Limit:  50,
		Offset: 0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	transactions := []*models.Transaction{
		{
			ID:             uuid.New(),
			UserID:         userID,
			ExchangeID:     uuid.New(),
			SubAccountID:   uuid.New(),
			Timestamp:      fixtures.FixedTime,
			Direction:      "credit",
			Reason:         "deposit",
			Amount:         100.0,
			ClosingBalance: 100.0,
			Info:           datatypes.JSON(infoJSON),
		},
		{
			ID:             uuid.New(),
			UserID:         userID,
			ExchangeID:     uuid.New(),
			SubAccountID:   uuid.New(),
			Timestamp:      fixtures.FixedTime.Add(time.Hour),
			Direction:      "debit",
			Reason:         "withdrawal",
			Amount:         50.0,
			ClosingBalance: 50.0,
			Info:           datatypes.JSON(infoJSON),
		},
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTransactionRepo.On("GetByUserID", suite.ctx, userID, filters).Return(transactions, int64(2), nil)

	// Act
	result, err := suite.transactionService.GetUserTransactions(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Transactions, 2)
	assert.Equal(suite.T(), int64(2), result.Total)
	assert.Equal(suite.T(), 50, result.Limit)
	assert.Equal(suite.T(), 0, result.Offset)
	assert.False(suite.T(), result.HasMore)
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetUserTransactions_WithFilters() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	direction := "credit"
	reason := "deposit"
	minAmount := 50.0
	maxAmount := 200.0
	startDate := fixtures.FixedTime
	endDate := fixtures.FixedTime.Add(24 * time.Hour)

	req := &TransactionQueryRequest{
		Direction: &direction,
		Reason:    &reason,
		MinAmount: &minAmount,
		MaxAmount: &maxAmount,
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     100,
		Offset:    0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	transactions := []*models.Transaction{
		{
			ID:             uuid.New(),
			UserID:         userID,
			Direction:      direction,
			Reason:         reason,
			Amount:         100.0,
			ClosingBalance: 100.0,
			Timestamp:      startDate.Add(time.Hour),
			Info:           datatypes.JSON(infoJSON),
		},
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTransactionRepo.On("GetByUserID", suite.ctx, userID, filters).Return(transactions, int64(1), nil)

	// Act
	result, err := suite.transactionService.GetUserTransactions(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Transactions, 1)
	assert.Equal(suite.T(), direction, result.Transactions[0].Direction)
	assert.Equal(suite.T(), reason, result.Transactions[0].Reason)
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetUserTransactions_InvalidDateRange() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	startDate := fixtures.FixedTime.Add(time.Hour)
	endDate := fixtures.FixedTime // End before start

	req := &TransactionQueryRequest{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	// Act
	result, err := suite.transactionService.GetUserTransactions(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "start date cannot be after end date")
}

func (suite *TransactionServiceTestSuite) TestGetUserTransactions_InvalidAmountRange() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	minAmount := 200.0
	maxAmount := 100.0 // Max less than min

	req := &TransactionQueryRequest{
		MinAmount: &minAmount,
		MaxAmount: &maxAmount,
	}

	// Act
	result, err := suite.transactionService.GetUserTransactions(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "min amount cannot be greater than max amount")
}

func (suite *TransactionServiceTestSuite) TestGetSubAccountTransactions_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	req := &TransactionQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	subAccount := &models.SubAccount{
		ID:     subAccountID,
		UserID: userID,
		Name:   "Test SubAccount",
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	transactions := []*models.Transaction{
		{
			ID:             uuid.New(),
			UserID:         userID,
			SubAccountID:   subAccountID,
			Direction:      "credit",
			Amount:         100.0,
			ClosingBalance: 100.0,
			Timestamp:      fixtures.FixedTime,
			Info:           datatypes.JSON(infoJSON),
		},
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)

	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTransactionRepo.On("GetBySubAccountID", suite.ctx, subAccountID, filters).Return(transactions, int64(1), nil)

	// Act
	result, err := suite.transactionService.GetSubAccountTransactions(suite.ctx, userID, subAccountID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Transactions, 1)
	assert.Equal(suite.T(), subAccountID, result.Transactions[0].SubAccountID)
	mockSubAccountRepo.AssertExpectations(suite.T())
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetSubAccountTransactions_SubAccountNotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	req := &TransactionQueryRequest{}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(nil, nil)

	// Act
	result, err := suite.transactionService.GetSubAccountTransactions(suite.ctx, userID, subAccountID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "sub-account not found")
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetExchangeTransactions_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	req := &TransactionQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Name:   "Test Exchange",
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	transactions := []*models.Transaction{
		{
			ID:             uuid.New(),
			UserID:         userID,
			ExchangeID:     exchangeID,
			Direction:      "credit",
			Amount:         100.0,
			ClosingBalance: 100.0,
			Timestamp:      fixtures.FixedTime,
			Info:           datatypes.JSON(infoJSON),
		},
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTransactionRepo.On("GetByExchangeID", suite.ctx, exchangeID, filters).Return(transactions, int64(1), nil)

	// Act
	result, err := suite.transactionService.GetExchangeTransactions(suite.ctx, userID, exchangeID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Transactions, 1)
	assert.Equal(suite.T(), exchangeID, result.Transactions[0].ExchangeID)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetTransaction_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	transactionID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	transaction := &models.Transaction{
		ID:             transactionID,
		UserID:         userID,
		ExchangeID:     uuid.New(),
		SubAccountID:   uuid.New(),
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         100.0,
		ClosingBalance: 100.0,
		Timestamp:      fixtures.FixedTime,
		Info:           datatypes.JSON(infoJSON),
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	mockTransactionRepo.On("GetByID", suite.ctx, transactionID).Return(transaction, nil)

	// Act
	result, err := suite.transactionService.GetTransaction(suite.ctx, userID, transactionID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), transactionID, result.ID)
	assert.Equal(suite.T(), userID, result.UserID)
	assert.Equal(suite.T(), "credit", result.Direction)
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetTransaction_NotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	transactionID := uuid.New()

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	mockTransactionRepo.On("GetByID", suite.ctx, transactionID).Return(nil, nil)

	// Act
	result, err := suite.transactionService.GetTransaction(suite.ctx, userID, transactionID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "transaction not found")
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetTransaction_WrongUser() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	otherUserID := uuid.New()
	transactionID := uuid.New()

	transaction := &models.Transaction{
		ID:     transactionID,
		UserID: otherUserID, // Different user
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	mockTransactionRepo.On("GetByID", suite.ctx, transactionID).Return(transaction, nil)

	// Act
	result, err := suite.transactionService.GetTransaction(suite.ctx, userID, transactionID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "transaction not found")
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetTransactionsByTimeRange_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	startTime := fixtures.FixedTime
	endTime := fixtures.FixedTime.Add(24 * time.Hour)
	req := &TransactionQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	allTransactions := []*models.Transaction{
		{
			ID:             uuid.New(),
			UserID:         userID,
			Direction:      "credit",
			Amount:         100.0,
			ClosingBalance: 100.0,
			Timestamp:      startTime.Add(time.Hour),
			Info:           datatypes.JSON(infoJSON),
		},
		{
			ID:             uuid.New(),
			UserID:         uuid.New(), // Different user - should be filtered out
			Direction:      "credit",
			Amount:         50.0,
			ClosingBalance: 50.0,
			Timestamp:      startTime.Add(2 * time.Hour),
			Info:           datatypes.JSON(infoJSON),
		},
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: &startTime,
		EndDate:   &endTime,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTransactionRepo.On("GetByTimeRange", suite.ctx, startTime, endTime, filters).Return(allTransactions, int64(2), nil)

	// Act
	result, err := suite.transactionService.GetTransactionsByTimeRange(suite.ctx, userID, startTime, endTime, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Transactions, 1) // Only user's transaction
	assert.Equal(suite.T(), userID, result.Transactions[0].UserID)
	assert.Equal(suite.T(), int64(1), result.Total) // Recalculated for user
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetTransactionsByTimeRange_InvalidTimeRange() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	startTime := fixtures.FixedTime.Add(time.Hour)
	endTime := fixtures.FixedTime // End before start
	req := &TransactionQueryRequest{}

	// Act
	result, err := suite.transactionService.GetTransactionsByTimeRange(suite.ctx, userID, startTime, endTime, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "start time cannot be after end time")
}

func (suite *TransactionServiceTestSuite) TestListAllTransactions_Success() {
	// Arrange
	req := &TransactionQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	transactions := []*models.Transaction{
		{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			Direction:      "credit",
			Amount:         100.0,
			ClosingBalance: 100.0,
			Timestamp:      fixtures.FixedTime,
			Info:           datatypes.JSON(infoJSON),
		},
		{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			Direction:      "debit",
			Amount:         50.0,
			ClosingBalance: 50.0,
			Timestamp:      fixtures.FixedTime.Add(time.Hour),
			Info:           datatypes.JSON(infoJSON),
		},
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	// Should use broad time range for admin queries
	startTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	mockTransactionRepo.On("GetByTimeRange", suite.ctx, startTime, 
		mock.AnythingOfType("time.Time"), mock.MatchedBy(func(f repositories.TransactionFilters) bool {
			return f.Limit == 100 && f.Offset == 0
		})).Return(transactions, int64(2), nil)

	// Act
	result, err := suite.transactionService.ListAllTransactions(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Transactions, 2)
	assert.Equal(suite.T(), int64(2), result.Total)
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestGetTransactionByID_Success() {
	// Arrange
	transactionID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	transaction := &models.Transaction{
		ID:             transactionID,
		UserID:         uuid.New(),
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         100.0,
		ClosingBalance: 100.0,
		Timestamp:      fixtures.FixedTime,
		Info:           datatypes.JSON(infoJSON),
	}

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	mockTransactionRepo.On("GetByID", suite.ctx, transactionID).Return(transaction, nil)

	// Act
	result, err := suite.transactionService.GetTransactionByID(suite.ctx, transactionID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), transactionID, result.ID)
	assert.Equal(suite.T(), "credit", result.Direction)
	mockTransactionRepo.AssertExpectations(suite.T())
}

func (suite *TransactionServiceTestSuite) TestDefaultPagination() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &TransactionQueryRequest{} // No limit specified

	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	mockTransactionRepo.On("GetByUserID", suite.ctx, userID, 
		mock.MatchedBy(func(f repositories.TransactionFilters) bool {
			return f.Limit == 100 // Default limit should be 100
		})).Return([]*models.Transaction{}, int64(0), nil)

	// Act
	result, err := suite.transactionService.GetUserTransactions(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 100, result.Limit) // Should default to 100
	mockTransactionRepo.AssertExpectations(suite.T())
}

func TestTransactionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionServiceTestSuite))
}