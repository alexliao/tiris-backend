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

type TradingLogServiceTestSuite struct {
	suite.Suite
	mockRepos         *repositories.Repositories
	tradingLogService *TradingLogService
	ctx               context.Context
}

func (suite *TradingLogServiceTestSuite) SetupTest() {
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
	suite.tradingLogService = NewTradingLogService(suite.mockRepos)
}

func (suite *TradingLogServiceTestSuite) TestCreateTradingLog_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	subAccountID := uuid.New()
	transactionID := uuid.New()

	req := &CreateTradingLogRequest{
		ExchangeID:    exchangeID,
		SubAccountID:  &subAccountID,
		TransactionID: &transactionID,
		Type:          "trade",
		Source:        "manual",
		Message:       "Test trading log",
		Info:          map[string]interface{}{"custom": "data"},
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Type:   "binance",
	}

	subAccount := &models.SubAccount{
		ID:     subAccountID,
		UserID: userID,
	}

	transaction := &models.Transaction{
		ID:     transactionID,
		UserID: userID,
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockTransactionRepo := suite.mockRepos.Transaction.(*mocks.MockTransactionRepository)
	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)
	mockTransactionRepo.On("GetByID", suite.ctx, transactionID).Return(transaction, nil)
	mockTradingLogRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.TradingLog")).Return(nil)

	// Act
	result, err := suite.tradingLogService.CreateTradingLog(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), userID, result.UserID)
	assert.Equal(suite.T(), exchangeID, result.ExchangeID)
	assert.Equal(suite.T(), &subAccountID, result.SubAccountID)
	assert.Equal(suite.T(), &transactionID, result.TransactionID)
	assert.Equal(suite.T(), req.Type, result.Type)
	assert.Equal(suite.T(), req.Source, result.Source)
	assert.Equal(suite.T(), req.Message, result.Message)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
	mockTransactionRepo.AssertExpectations(suite.T())
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestCreateTradingLog_MinimalRequest() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()

	req := &CreateTradingLogRequest{
		ExchangeID: exchangeID,
		Type:       "trade",
		Source:     "bot",
		Message:    "Bot generated trading log",
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Type:   "binance",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockTradingLogRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.TradingLog")).Return(nil)

	// Act
	result, err := suite.tradingLogService.CreateTradingLog(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), userID, result.UserID)
	assert.Equal(suite.T(), exchangeID, result.ExchangeID)
	assert.Nil(suite.T(), result.SubAccountID)
	assert.Nil(suite.T(), result.TransactionID)
	assert.Equal(suite.T(), req.Type, result.Type)
	assert.Equal(suite.T(), req.Source, result.Source)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestCreateTradingLog_ExchangeNotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()

	req := &CreateTradingLogRequest{
		ExchangeID: exchangeID,
		Type:       "trade",
		Source:     "manual",
		Message:    "Test trading log",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(nil, nil)

	// Act
	result, err := suite.tradingLogService.CreateTradingLog(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange not found")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestCreateTradingLog_WrongUser() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	otherUserID := uuid.New()

	req := &CreateTradingLogRequest{
		ExchangeID: exchangeID,
		Type:       "trade",
		Source:     "manual",
		Message:    "Test trading log",
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: otherUserID, // Different user
		Type:   "binance",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)

	// Act
	result, err := suite.tradingLogService.CreateTradingLog(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange not found")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetUserTradingLogs_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &TradingLogQueryRequest{
		Limit:  50,
		Offset: 0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	tradingLogs := []*models.TradingLog{
		{
			ID:         uuid.New(),
			UserID:     userID,
			ExchangeID: uuid.New(),
			Type:       "trade",
			Source:     "manual",
			Message:    "Manual trade log",
			Timestamp:  fixtures.FixedTime,
			Info:       datatypes.JSON(infoJSON),
		},
		{
			ID:         uuid.New(),
			UserID:     userID,
			ExchangeID: uuid.New(),
			Type:       "order",
			Source:     "bot",
			Message:    "Bot order log",
			Timestamp:  fixtures.FixedTime.Add(time.Hour),
			Info:       datatypes.JSON(infoJSON),
		},
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTradingLogRepo.On("GetByUserID", suite.ctx, userID, filters).Return(tradingLogs, int64(2), nil)

	// Act
	result, err := suite.tradingLogService.GetUserTradingLogs(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.TradingLogs, 2)
	assert.Equal(suite.T(), int64(2), result.Total)
	assert.Equal(suite.T(), 50, result.Limit)
	assert.Equal(suite.T(), 0, result.Offset)
	assert.False(suite.T(), result.HasMore)
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetUserTradingLogs_WithFilters() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	logType := "trade"
	source := "manual"
	startDate := fixtures.FixedTime
	endDate := fixtures.FixedTime.Add(24 * time.Hour)

	req := &TradingLogQueryRequest{
		Type:      &logType,
		Source:    &source,
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     100,
		Offset:    0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	tradingLogs := []*models.TradingLog{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      logType,
			Source:    source,
			Message:   "Filtered trading log",
			Timestamp: startDate.Add(time.Hour),
			Info:      datatypes.JSON(infoJSON),
		},
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTradingLogRepo.On("GetByUserID", suite.ctx, userID, filters).Return(tradingLogs, int64(1), nil)

	// Act
	result, err := suite.tradingLogService.GetUserTradingLogs(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.TradingLogs, 1)
	assert.Equal(suite.T(), logType, result.TradingLogs[0].Type)
	assert.Equal(suite.T(), source, result.TradingLogs[0].Source)
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetUserTradingLogs_InvalidDateRange() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	startDate := fixtures.FixedTime.Add(time.Hour)
	endDate := fixtures.FixedTime // End before start

	req := &TradingLogQueryRequest{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	// Act
	result, err := suite.tradingLogService.GetUserTradingLogs(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "start date cannot be after end date")
}

func (suite *TradingLogServiceTestSuite) TestGetSubAccountTradingLogs_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	req := &TradingLogQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	subAccount := &models.SubAccount{
		ID:     subAccountID,
		UserID: userID,
		Name:   "Test SubAccount",
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	tradingLogs := []*models.TradingLog{
		{
			ID:           uuid.New(),
			UserID:       userID,
			SubAccountID: &subAccountID,
			Type:         "trade",
			Source:       "manual",
			Message:      "SubAccount trade log",
			Timestamp:    fixtures.FixedTime,
			Info:         datatypes.JSON(infoJSON),
		},
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)

	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTradingLogRepo.On("GetBySubAccountID", suite.ctx, subAccountID, filters).Return(tradingLogs, int64(1), nil)

	// Act
	result, err := suite.tradingLogService.GetSubAccountTradingLogs(suite.ctx, userID, subAccountID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.TradingLogs, 1)
	assert.Equal(suite.T(), &subAccountID, result.TradingLogs[0].SubAccountID)
	mockSubAccountRepo.AssertExpectations(suite.T())
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetExchangeTradingLogs_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	req := &TradingLogQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Name:   "Test Exchange",
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	tradingLogs := []*models.TradingLog{
		{
			ID:         uuid.New(),
			UserID:     userID,
			ExchangeID: exchangeID,
			Type:       "trade",
			Source:     "manual",
			Message:    "Exchange trade log",
			Timestamp:  fixtures.FixedTime,
			Info:       datatypes.JSON(infoJSON),
		},
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTradingLogRepo.On("GetByExchangeID", suite.ctx, exchangeID, filters).Return(tradingLogs, int64(1), nil)

	// Act
	result, err := suite.tradingLogService.GetExchangeTradingLogs(suite.ctx, userID, exchangeID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.TradingLogs, 1)
	assert.Equal(suite.T(), exchangeID, result.TradingLogs[0].ExchangeID)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetTradingLog_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	tradingLogID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	tradingLog := &models.TradingLog{
		ID:        tradingLogID,
		UserID:    userID,
		Type:      "trade",
		Source:    "manual",
		Message:   "Test trading log",
		Timestamp: fixtures.FixedTime,
		Info:      datatypes.JSON(infoJSON),
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	mockTradingLogRepo.On("GetByID", suite.ctx, tradingLogID).Return(tradingLog, nil)

	// Act
	result, err := suite.tradingLogService.GetTradingLog(suite.ctx, userID, tradingLogID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), tradingLogID, result.ID)
	assert.Equal(suite.T(), userID, result.UserID)
	assert.Equal(suite.T(), "trade", result.Type)
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetTradingLog_NotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	tradingLogID := uuid.New()

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	mockTradingLogRepo.On("GetByID", suite.ctx, tradingLogID).Return(nil, nil)

	// Act
	result, err := suite.tradingLogService.GetTradingLog(suite.ctx, userID, tradingLogID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "trading log not found")
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestDeleteTradingLog_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	tradingLogID := uuid.New()

	tradingLog := &models.TradingLog{
		ID:     tradingLogID,
		UserID: userID,
		Source: "manual", // Manual logs can be deleted
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	mockTradingLogRepo.On("GetByID", suite.ctx, tradingLogID).Return(tradingLog, nil)
	mockTradingLogRepo.On("Delete", suite.ctx, tradingLogID).Return(nil)

	// Act
	err := suite.tradingLogService.DeleteTradingLog(suite.ctx, userID, tradingLogID)

	// Assert
	assert.NoError(suite.T(), err)
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestDeleteTradingLog_BotGenerated() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	tradingLogID := uuid.New()

	tradingLog := &models.TradingLog{
		ID:     tradingLogID,
		UserID: userID,
		Source: "bot", // Bot logs cannot be deleted
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	mockTradingLogRepo.On("GetByID", suite.ctx, tradingLogID).Return(tradingLog, nil)

	// Act
	err := suite.tradingLogService.DeleteTradingLog(suite.ctx, userID, tradingLogID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot delete bot-generated trading logs")
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetTradingLogsByTimeRange_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	startTime := fixtures.FixedTime
	endTime := fixtures.FixedTime.Add(24 * time.Hour)
	req := &TradingLogQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	allTradingLogs := []*models.TradingLog{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      "trade",
			Source:    "manual",
			Message:   "User's trading log",
			Timestamp: startTime.Add(time.Hour),
			Info:      datatypes.JSON(infoJSON),
		},
		{
			ID:        uuid.New(),
			UserID:    uuid.New(), // Different user - should be filtered out
			Type:      "trade",
			Source:    "manual",
			Message:   "Other user's trading log",
			Timestamp: startTime.Add(2 * time.Hour),
			Info:      datatypes.JSON(infoJSON),
		},
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: &startTime,
		EndDate:   &endTime,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}
	mockTradingLogRepo.On("GetByTimeRange", suite.ctx, startTime, endTime, filters).Return(allTradingLogs, int64(2), nil)

	// Act
	result, err := suite.tradingLogService.GetTradingLogsByTimeRange(suite.ctx, userID, startTime, endTime, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.TradingLogs, 1) // Only user's trading log
	assert.Equal(suite.T(), userID, result.TradingLogs[0].UserID)
	assert.Equal(suite.T(), int64(1), result.Total) // Recalculated for user
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestListAllTradingLogs_Success() {
	// Arrange
	req := &TradingLogQueryRequest{
		Limit:  100,
		Offset: 0,
	}

	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	tradingLogs := []*models.TradingLog{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      "trade",
			Source:    "manual",
			Message:   "Admin view trading log 1",
			Timestamp: fixtures.FixedTime,
			Info:      datatypes.JSON(infoJSON),
		},
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      "order",
			Source:    "bot",
			Message:   "Admin view trading log 2",
			Timestamp: fixtures.FixedTime.Add(time.Hour),
			Info:      datatypes.JSON(infoJSON),
		},
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	// Should use broad time range for admin queries
	startTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	mockTradingLogRepo.On("GetByTimeRange", suite.ctx, startTime, 
		mock.AnythingOfType("time.Time"), mock.MatchedBy(func(f repositories.TradingLogFilters) bool {
			return f.Limit == 100 && f.Offset == 0
		})).Return(tradingLogs, int64(2), nil)

	// Act
	result, err := suite.tradingLogService.ListAllTradingLogs(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.TradingLogs, 2)
	assert.Equal(suite.T(), int64(2), result.Total)
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestGetTradingLogByID_Success() {
	// Arrange
	tradingLogID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	tradingLog := &models.TradingLog{
		ID:        tradingLogID,
		UserID:    uuid.New(),
		Type:      "trade",
		Source:    "manual",
		Message:   "Admin view trading log",
		Timestamp: fixtures.FixedTime,
		Info:      datatypes.JSON(infoJSON),
	}

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	mockTradingLogRepo.On("GetByID", suite.ctx, tradingLogID).Return(tradingLog, nil)

	// Act
	result, err := suite.tradingLogService.GetTradingLogByID(suite.ctx, tradingLogID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), tradingLogID, result.ID)
	assert.Equal(suite.T(), "trade", result.Type)
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func (suite *TradingLogServiceTestSuite) TestDefaultPagination() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &TradingLogQueryRequest{} // No limit specified

	mockTradingLogRepo := suite.mockRepos.TradingLog.(*mocks.MockTradingLogRepository)
	mockTradingLogRepo.On("GetByUserID", suite.ctx, userID, 
		mock.MatchedBy(func(f repositories.TradingLogFilters) bool {
			return f.Limit == 100 // Default limit should be 100
		})).Return([]*models.TradingLog{}, int64(0), nil)

	// Act
	result, err := suite.tradingLogService.GetUserTradingLogs(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 100, result.Limit) // Should default to 100
	mockTradingLogRepo.AssertExpectations(suite.T())
}

func TestTradingLogServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TradingLogServiceTestSuite))
}