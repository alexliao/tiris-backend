package test

import (
	"context"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestTradingLogService_CreateTradingLog tests the CreateTradingLog functionality
func TestTradingLogService_CreateTradingLog(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        mockTradingRepo,
		SubAccount:      mockSubAccountRepo,
		Transaction:     mockTransactionRepo,
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTradingPlatform := tradingFactory.WithUserID(userID)
	testTradingPlatform.ID = tradingID

	// Test successful trading log creation (basic)
	t.Run("successful_creation_basic", func(t *testing.T) {
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:       "trade",
			Source:     "manual",
			Message:    "Manual trade executed successfully",
			Info:       map[string]interface{}{"symbol": "BTCUSDT", "side": "buy"},
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTradingPlatform, nil).Once()
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, tradingID, result.TradingID)
		assert.Equal(t, request.Type, result.Type)
		assert.Equal(t, request.Source, result.Source)
		assert.Equal(t, request.Message, result.Message)
		assert.Nil(t, result.SubAccountID)
		assert.Nil(t, result.TransactionID)
		assert.NotEmpty(t, result.Info)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test successful creation with sub-account and transaction
	t.Run("successful_creation_with_references", func(t *testing.T) {
		subAccountID := uuid.New()
		transactionID := uuid.New()

		subAccountFactory := helpers.NewSubAccountFactory()
		testSubAccount := subAccountFactory.WithUserAndTrading(userID, tradingID)
		testSubAccount.ID = subAccountID

		transactionFactory := helpers.NewTransactionFactory()
		testTransaction := transactionFactory.WithUserID(userID)
		testTransaction.ID = transactionID

		request := &services.CreateTradingLogRequest{
			TradingID:    tradingID,
			SubAccountID:  &subAccountID,
			TransactionID: &transactionID,
			Type:          "strategy",
			Source:        "bot",
			Message:       "Grid strategy triggered: Position opened",
			Info:          map[string]interface{}{"strategy": "grid", "action": "open_position"},
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTradingPlatform, nil).Once()
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(testTransaction, nil).Once()
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, tradingID, result.TradingID)
		assert.Equal(t, subAccountID, *result.SubAccountID)
		assert.Equal(t, transactionID, *result.TransactionID)
		assert.Equal(t, request.Type, result.Type)
		assert.Equal(t, request.Source, result.Source)
		assert.Equal(t, request.Message, result.Message)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading platform not found
	t.Run("trading_not_found", func(t *testing.T) {
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:       "trade",
			Source:     "manual",
			Message:    "Test message",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test trading platform belongs to different user
	t.Run("trading_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserTradingPlatform := tradingFactory.WithUserID(differentUserID)
		wrongUserTradingPlatform.ID = tradingID

		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:       "trade",
			Source:     "manual",
			Message:    "Test message",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(wrongUserTradingPlatform, nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test sub-account belongs to different user
	t.Run("subaccount_wrong_user", func(t *testing.T) {
		subAccountID := uuid.New()
		differentUserID := uuid.New()

		subAccountFactory := helpers.NewSubAccountFactory()
		wrongUserSubAccount := subAccountFactory.WithUserAndTrading(differentUserID, tradingID)
		wrongUserSubAccount.ID = subAccountID

		request := &services.CreateTradingLogRequest{
			TradingID:   tradingID,
			SubAccountID: &subAccountID,
			Type:         "trade",
			Source:       "manual",
			Message:      "Test message",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTradingPlatform, nil).Once()
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(wrongUserSubAccount, nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test transaction belongs to different user
	t.Run("transaction_wrong_user", func(t *testing.T) {
		transactionID := uuid.New()
		differentUserID := uuid.New()

		transactionFactory := helpers.NewTransactionFactory()
		wrongUserTransaction := transactionFactory.WithUserID(differentUserID)
		wrongUserTransaction.ID = transactionID

		request := &services.CreateTradingLogRequest{
			TradingID:    tradingID,
			TransactionID: &transactionID,
			Type:          "trade",
			Source:        "manual",
			Message:       "Test message",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTradingPlatform, nil).Once()
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(wrongUserTransaction, nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_GetUserTradingLogs tests the GetUserTradingLogs functionality
func TestTradingLogService_GetUserTradingLogs(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingLogFactory := helpers.NewTradingLogFactory()

	// Test successful retrieval with default pagination
	t.Run("successful_retrieval_default", func(t *testing.T) {
		testTradingLogs := []*models.TradingLog{
			tradingLogFactory.WithUserID(userID),
			tradingLogFactory.WithUserID(userID),
		}
		testTradingLogs[0].Type = "trade"
		testTradingLogs[1].Type = "strategy"

		request := &services.TradingLogQueryRequest{}
		expectedFilters := repositories.TradingLogFilters{
			Limit: 100, // Default limit
		}

		// Setup mock expectations
		mockTradingLogRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTradingLogs, int64(2), nil).Once()

		// Execute test
		result, err := tradingLogService.GetUserTradingLogs(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.TradingLogs, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 100, result.Limit)
		assert.Equal(t, 0, result.Offset)
		assert.False(t, result.HasMore)
		assert.Equal(t, "trade", result.TradingLogs[0].Type)
		assert.Equal(t, "strategy", result.TradingLogs[1].Type)

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test with filtering parameters
	t.Run("successful_retrieval_filtered", func(t *testing.T) {
		logType := "strategy"
		source := "bot"
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		request := &services.TradingLogQueryRequest{
			Type:      &logType,
			Source:    &source,
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     50,
			Offset:    10,
		}

		expectedFilters := repositories.TradingLogFilters{
			Type:      &logType,
			Source:    &source,
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     50,
			Offset:    10,
		}

		testTradingLogs := []*models.TradingLog{
			tradingLogFactory.BotLog(),
		}
		testTradingLogs[0].UserID = userID

		// Setup mock expectations
		mockTradingLogRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTradingLogs, int64(1), nil).Once()

		// Execute test
		result, err := tradingLogService.GetUserTradingLogs(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.TradingLogs, 1)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, 50, result.Limit)
		assert.Equal(t, 10, result.Offset)
		assert.False(t, result.HasMore)

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test validation errors
	t.Run("invalid_date_range", func(t *testing.T) {
		startDate := time.Now()
		endDate := time.Now().Add(-24 * time.Hour) // End before start

		request := &services.TradingLogQueryRequest{
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		// Execute test
		result, err := tradingLogService.GetUserTradingLogs(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "start date cannot be after end date")
	})

	// Test pagination with HasMore
	t.Run("pagination_has_more", func(t *testing.T) {
		testTradingLogs := []*models.TradingLog{
			tradingLogFactory.WithUserID(userID),
		}

		request := &services.TradingLogQueryRequest{
			Limit:  10,
			Offset: 0,
		}

		expectedFilters := repositories.TradingLogFilters{
			Limit:  10,
			Offset: 0,
		}

		// Setup mock expectations - return total > limit + offset
		mockTradingLogRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTradingLogs, int64(25), nil).Once()

		// Execute test
		result, err := tradingLogService.GetUserTradingLogs(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.HasMore) // 0 + 10 < 25

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_GetSubAccountTradingLogs tests the GetSubAccountTradingLogs functionality
func TestTradingLogService_GetSubAccountTradingLogs(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccount := subAccountFactory.WithUserAndTrading(userID, tradingID)
	testSubAccount.ID = subAccountID

	tradingLogFactory := helpers.NewTradingLogFactory()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{
			Limit: 50,
		}

		testTradingLogs := []*models.TradingLog{
			tradingLogFactory.WithSubAccountID(subAccountID),
			tradingLogFactory.WithSubAccountID(subAccountID),
		}
		testTradingLogs[0].UserID = userID
		testTradingLogs[1].UserID = userID

		expectedFilters := repositories.TradingLogFilters{
			Limit: 50,
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		mockTradingLogRepo.On("GetBySubAccountID", mock.Anything, subAccountID, expectedFilters).
			Return(testTradingLogs, int64(2), nil).Once()

		// Execute test
		result, err := tradingLogService.GetSubAccountTradingLogs(context.Background(), userID, subAccountID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.TradingLogs, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 50, result.Limit)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test sub-account not found
	t.Run("subaccount_not_found", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingLogService.GetSubAccountTradingLogs(context.Background(), userID, subAccountID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_GetTradingLogs tests the GetTradingLogs functionality
func TestTradingLogService_GetTradingLogs(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTradingPlatform := tradingFactory.WithUserID(userID)
	testTradingPlatform.ID = tradingID

	tradingLogFactory := helpers.NewTradingLogFactory()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{
			Limit: 25,
		}

		testTradingLogs := []*models.TradingLog{
			tradingLogFactory.WithTradingID(tradingID),
			tradingLogFactory.WithTradingID(tradingID),
		}
		testTradingLogs[0].UserID = userID
		testTradingLogs[1].UserID = userID

		expectedFilters := repositories.TradingLogFilters{
			Limit: 25,
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTradingPlatform, nil).Once()
		mockTradingLogRepo.On("GetByTradingID", mock.Anything, tradingID, expectedFilters).
			Return(testTradingLogs, int64(2), nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLogs(context.Background(), userID, tradingID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.TradingLogs, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 25, result.Limit)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading platform not found
	t.Run("trading_not_found", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLogs(context.Background(), userID, tradingID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_GetTradingLog tests the GetTradingLog functionality
func TestTradingLogService_GetTradingLog(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingLogID := uuid.New()
	tradingLogFactory := helpers.NewTradingLogFactory()
	testTradingLog := tradingLogFactory.WithUserID(userID)
	testTradingLog.ID = tradingLogID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(testTradingLog, nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, tradingLogID, result.ID)
		assert.Equal(t, userID, result.UserID)

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading log not found
	t.Run("trading_log_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading log not found")

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading log belongs to different user
	t.Run("trading_log_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserTradingLog := tradingLogFactory.WithUserID(differentUserID)
		wrongUserTradingLog.ID = tradingLogID

		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(wrongUserTradingLog, nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading log not found")

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_DeleteTradingLog tests the DeleteTradingLog functionality
func TestTradingLogService_DeleteTradingLog(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingLogID := uuid.New()
	tradingLogFactory := helpers.NewTradingLogFactory()

	// Test successful deletion of manual log
	t.Run("successful_deletion_manual", func(t *testing.T) {
		testTradingLog := tradingLogFactory.WithUserID(userID)
		testTradingLog.ID = tradingLogID
		testTradingLog.Source = "manual"

		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(testTradingLog, nil).Once()
		mockTradingLogRepo.On("Delete", mock.Anything, tradingLogID).
			Return(nil).Once()

		// Execute test
		err := tradingLogService.DeleteTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.NoError(t, err)

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test deletion of bot-generated log (should fail)
	t.Run("deletion_bot_log_failed", func(t *testing.T) {
		testTradingLog := tradingLogFactory.BotLog()
		testTradingLog.ID = tradingLogID
		testTradingLog.UserID = userID
		testTradingLog.Source = "bot"

		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(testTradingLog, nil).Once()

		// Execute test
		err := tradingLogService.DeleteTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete bot-generated trading logs")

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading log not found
	t.Run("trading_log_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(nil, nil).Once()

		// Execute test
		err := tradingLogService.DeleteTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trading log not found")

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading log belongs to different user
	t.Run("trading_log_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserTradingLog := tradingLogFactory.WithUserID(differentUserID)
		wrongUserTradingLog.ID = tradingLogID
		wrongUserTradingLog.Source = "manual"

		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(wrongUserTradingLog, nil).Once()

		// Execute test
		err := tradingLogService.DeleteTradingLog(context.Background(), userID, tradingLogID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trading log not found")

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_GetTradingLogsByTimeRange tests the GetTradingLogsByTimeRange functionality
func TestTradingLogService_GetTradingLogsByTimeRange(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	anotherUserID := uuid.New()
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	tradingLogFactory := helpers.NewTradingLogFactory()

	// Test successful retrieval with user filtering
	t.Run("successful_retrieval_filtered", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{
			Limit: 50,
		}

		// Mix of user's trading logs and other user's trading logs
		allTradingLogs := []*models.TradingLog{
			tradingLogFactory.WithUserID(userID),        // User's log
			tradingLogFactory.WithUserID(anotherUserID), // Another user's log
			tradingLogFactory.WithUserID(userID),        // User's log
		}

		expectedFilters := repositories.TradingLogFilters{
			StartDate: &startTime,
			EndDate:   &endTime,
			Limit:     50,
		}

		// Setup mock expectations
		mockTradingLogRepo.On("GetByTimeRange", mock.Anything, startTime, endTime, expectedFilters).
			Return(allTradingLogs, int64(3), nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLogsByTimeRange(context.Background(), userID, startTime, endTime, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should only return 2 trading logs (user's logs only)
		assert.Len(t, result.TradingLogs, 2)
		assert.Equal(t, int64(2), result.Total) // Filtered total
		assert.Equal(t, 50, result.Limit)

		// All returned trading logs should belong to the user
		for _, log := range result.TradingLogs {
			assert.Equal(t, userID, log.UserID)
		}

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test invalid time range
	t.Run("invalid_time_range", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{}

		// Start time after end time
		invalidStartTime := time.Now()
		invalidEndTime := time.Now().Add(-24 * time.Hour)

		// Execute test
		result, err := tradingLogService.GetTradingLogsByTimeRange(context.Background(), userID, invalidStartTime, invalidEndTime, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "start time cannot be after end time")
	})
}

// TestTradingLogService_ListAllTradingLogs tests the ListAllTradingLogs functionality (admin)
func TestTradingLogService_ListAllTradingLogs(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	tradingLogFactory := helpers.NewTradingLogFactory()

	// Test successful retrieval with default time range
	t.Run("successful_retrieval_default", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{
			Limit: 75,
		}

		testTradingLogs := []*models.TradingLog{
			tradingLogFactory.Build(),
			tradingLogFactory.Build(),
		}

		// Setup mock expectations - we expect the call with any start/end time
		mockTradingLogRepo.On("GetByTimeRange", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("repositories.TradingLogFilters")).
			Return(testTradingLogs, int64(2), nil).Once()

		// Execute test
		result, err := tradingLogService.ListAllTradingLogs(context.Background(), request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.TradingLogs, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 75, result.Limit)

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_GetTradingLogByID tests the GetTradingLogByID functionality (admin)
func TestTradingLogService_GetTradingLogByID(t *testing.T) {
	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	tradingLogID := uuid.New()
	tradingLogFactory := helpers.NewTradingLogFactory()
	testTradingLog := tradingLogFactory.Build()
	testTradingLog.ID = tradingLogID

	// Test successful retrieval (admin)
	t.Run("successful_admin_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(testTradingLog, nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLogByID(context.Background(), tradingLogID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, tradingLogID, result.ID)
		// Admin can access any user's trading log

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Test trading log not found (admin)
	t.Run("trading_log_not_found_admin", func(t *testing.T) {
		// Setup mock expectations
		mockTradingLogRepo.On("GetByID", mock.Anything, tradingLogID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingLogService.GetTradingLogByID(context.Background(), tradingLogID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading log not found")

		// Verify mock expectations
		mockTradingLogRepo.AssertExpectations(t)
	})
}

// Performance test for trading log operations
func TestTradingLogService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database
	mockDB := &gorm.DB{}

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingLogFactory := helpers.NewTradingLogFactory()
	testTradingLogs := []*models.TradingLog{
		tradingLogFactory.WithUserID(userID),
		tradingLogFactory.WithUserID(userID),
		tradingLogFactory.WithUserID(userID),
	}

	t.Run("get_user_trading_logs_performance", func(t *testing.T) {
		request := &services.TradingLogQueryRequest{
			Limit: 10,
		}

		expectedFilters := repositories.TradingLogFilters{
			Limit: 10,
		}

		// Setup mock for multiple calls
		mockTradingLogRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTradingLogs, int64(3), nil).Times(100)

		timer := helpers.NewPerformanceTimer()
		timer.Start()

		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := tradingLogService.GetUserTradingLogs(context.Background(), userID, request)
			require.NoError(t, err)
		}

		timer.Stop()

		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 GetUserTradingLogs operations should complete within 1 second")

		mockTradingLogRepo.AssertExpectations(t)
	})
}
