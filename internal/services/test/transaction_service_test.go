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
)

// TestTransactionService_GetUserTransactions tests the GetUserTransactions functionality
func TestTransactionService_GetUserTransactions(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	userID := uuid.New()
	transactionFactory := helpers.NewTransactionFactory()

	// Test successful retrieval with default pagination
	t.Run("successful_retrieval_default", func(t *testing.T) {
		testTransactions := []*models.Transaction{
			transactionFactory.WithUserID(userID),
			transactionFactory.WithUserID(userID),
		}
		testTransactions[0].Reason = "deposit"
		testTransactions[1].Reason = "withdrawal"

		request := &services.TransactionQueryRequest{}
		expectedFilters := repositories.TransactionFilters{
			Limit: 100, // Default limit
		}

		// Setup mock expectations
		mockTransactionRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTransactions, int64(2), nil).Once()

		// Execute test
		result, err := transactionService.GetUserTransactions(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 100, result.Limit)
		assert.Equal(t, 0, result.Offset)
		assert.False(t, result.HasMore)
		assert.Equal(t, "deposit", result.Transactions[0].Reason)
		assert.Equal(t, "withdrawal", result.Transactions[1].Reason)

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test with filtering parameters
	t.Run("successful_retrieval_filtered", func(t *testing.T) {
		direction := "credit"
		reason := "deposit"
		minAmount := 100.0
		maxAmount := 1000.0
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		request := &services.TransactionQueryRequest{
			Direction: &direction,
			Reason:    &reason,
			MinAmount: &minAmount,
			MaxAmount: &maxAmount,
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     50,
			Offset:    10,
		}

		expectedFilters := repositories.TransactionFilters{
			Direction: &direction,
			Reason:    &reason,
			MinAmount: &minAmount,
			MaxAmount: &maxAmount,
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     50,
			Offset:    10,
		}

		testTransactions := []*models.Transaction{
			transactionFactory.DepositTransaction(500.0, 1500.0),
		}
		testTransactions[0].UserID = userID

		// Setup mock expectations
		mockTransactionRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTransactions, int64(1), nil).Once()

		// Execute test
		result, err := transactionService.GetUserTransactions(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 1)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, 50, result.Limit)
		assert.Equal(t, 10, result.Offset)
		assert.False(t, result.HasMore)

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test validation errors
	t.Run("invalid_date_range", func(t *testing.T) {
		startDate := time.Now()
		endDate := time.Now().Add(-24 * time.Hour) // End before start

		request := &services.TransactionQueryRequest{
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		// Execute test
		result, err := transactionService.GetUserTransactions(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "start date cannot be after end date")
	})

	// Test invalid amount range
	t.Run("invalid_amount_range", func(t *testing.T) {
		minAmount := 1000.0
		maxAmount := 100.0 // Max less than min

		request := &services.TransactionQueryRequest{
			MinAmount: &minAmount,
			MaxAmount: &maxAmount,
		}

		// Execute test
		result, err := transactionService.GetUserTransactions(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "min amount cannot be greater than max amount")
	})

	// Test pagination with HasMore
	t.Run("pagination_has_more", func(t *testing.T) {
		testTransactions := []*models.Transaction{
			transactionFactory.WithUserID(userID),
		}

		request := &services.TransactionQueryRequest{
			Limit:  10,
			Offset: 0,
		}

		expectedFilters := repositories.TransactionFilters{
			Limit:  10,
			Offset: 0,
		}

		// Setup mock expectations - return total > limit + offset
		mockTransactionRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTransactions, int64(25), nil).Once()

		// Execute test
		result, err := transactionService.GetUserTransactions(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.HasMore) // 0 + 10 < 25

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})
}

// TestTransactionService_GetSubAccountTransactions tests the GetSubAccountTransactions functionality
func TestTransactionService_GetSubAccountTransactions(t *testing.T) {
	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	subAccountID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccount := subAccountFactory.WithUserAndTrading(userID, exchangeID)
	testSubAccount.ID = subAccountID

	transactionFactory := helpers.NewTransactionFactory()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		request := &services.TransactionQueryRequest{
			Limit: 50,
		}

		testTransactions := []*models.Transaction{
			transactionFactory.WithSubAccountID(subAccountID),
			transactionFactory.WithSubAccountID(subAccountID),
		}
		testTransactions[0].UserID = userID
		testTransactions[1].UserID = userID

		expectedFilters := repositories.TransactionFilters{
			Limit: 50,
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		mockTransactionRepo.On("GetBySubAccountID", mock.Anything, subAccountID, expectedFilters).
			Return(testTransactions, int64(2), nil).Once()

		// Execute test
		result, err := transactionService.GetSubAccountTransactions(context.Background(), userID, subAccountID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 50, result.Limit)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test sub-account not found
	t.Run("subaccount_not_found", func(t *testing.T) {
		request := &services.TransactionQueryRequest{}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(nil, nil).Once()

		// Execute test
		result, err := transactionService.GetSubAccountTransactions(context.Background(), userID, subAccountID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test sub-account belongs to different user
	t.Run("subaccount_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserSubAccount := subAccountFactory.WithUserAndTrading(differentUserID, exchangeID)
		wrongUserSubAccount.ID = subAccountID

		request := &services.TransactionQueryRequest{}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(wrongUserSubAccount, nil).Once()

		// Execute test
		result, err := transactionService.GetSubAccountTransactions(context.Background(), userID, subAccountID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestTransactionService_GetExchangeTransactions tests the GetExchangeTransactions functionality
func TestTransactionService_GetExchangeTransactions(t *testing.T) {
	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}
	mockExchangeRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID

	transactionFactory := helpers.NewTransactionFactory()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		request := &services.TransactionQueryRequest{
			Limit: 25,
		}

		testTransactions := []*models.Transaction{
			transactionFactory.WithTradingID(exchangeID),
			transactionFactory.WithTradingID(exchangeID),
		}
		testTransactions[0].UserID = userID
		testTransactions[1].UserID = userID

		expectedFilters := repositories.TransactionFilters{
			Limit: 25,
		}

		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockTransactionRepo.On("GetByTradingID", mock.Anything, exchangeID, expectedFilters).
			Return(testTransactions, int64(2), nil).Once()

		// Execute test
		result, err := transactionService.GetExchangeTransactions(context.Background(), userID, exchangeID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 25, result.Limit)

		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		request := &services.TransactionQueryRequest{}

		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()

		// Execute test
		result, err := transactionService.GetExchangeTransactions(context.Background(), userID, exchangeID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})

	// Test exchange belongs to different user
	t.Run("exchange_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserExchange := exchangeFactory.WithUserID(differentUserID)
		wrongUserExchange.ID = exchangeID

		request := &services.TransactionQueryRequest{}

		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(wrongUserExchange, nil).Once()

		// Execute test
		result, err := transactionService.GetExchangeTransactions(context.Background(), userID, exchangeID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// TestTransactionService_GetTransaction tests the GetTransaction functionality
func TestTransactionService_GetTransaction(t *testing.T) {
	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	userID := uuid.New()
	transactionID := uuid.New()
	transactionFactory := helpers.NewTransactionFactory()
	testTransaction := transactionFactory.WithUserID(userID)
	testTransaction.ID = transactionID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(testTransaction, nil).Once()

		// Execute test
		result, err := transactionService.GetTransaction(context.Background(), userID, transactionID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, transactionID, result.ID)
		assert.Equal(t, userID, result.UserID)

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test transaction not found
	t.Run("transaction_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(nil, nil).Once()

		// Execute test
		result, err := transactionService.GetTransaction(context.Background(), userID, transactionID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction not found")

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test transaction belongs to different user
	t.Run("transaction_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserTransaction := transactionFactory.WithUserID(differentUserID)
		wrongUserTransaction.ID = transactionID

		// Setup mock expectations
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(wrongUserTransaction, nil).Once()

		// Execute test
		result, err := transactionService.GetTransaction(context.Background(), userID, transactionID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction not found")

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})
}

// TestTransactionService_GetTransactionsByTimeRange tests the GetTransactionsByTimeRange functionality
func TestTransactionService_GetTransactionsByTimeRange(t *testing.T) {
	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	userID := uuid.New()
	anotherUserID := uuid.New()
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	transactionFactory := helpers.NewTransactionFactory()

	// Test successful retrieval with user filtering
	t.Run("successful_retrieval_filtered", func(t *testing.T) {
		request := &services.TransactionQueryRequest{
			Limit: 50,
		}

		// Mix of user's transactions and other user's transactions
		allTransactions := []*models.Transaction{
			transactionFactory.WithUserID(userID),        // User's transaction
			transactionFactory.WithUserID(anotherUserID), // Another user's transaction
			transactionFactory.WithUserID(userID),        // User's transaction
		}

		expectedFilters := repositories.TransactionFilters{
			StartDate: &startTime,
			EndDate:   &endTime,
			Limit:     50,
		}

		// Setup mock expectations
		mockTransactionRepo.On("GetByTimeRange", mock.Anything, startTime, endTime, expectedFilters).
			Return(allTransactions, int64(3), nil).Once()

		// Execute test
		result, err := transactionService.GetTransactionsByTimeRange(context.Background(), userID, startTime, endTime, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should only return 2 transactions (user's transactions only)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, int64(2), result.Total) // Filtered total
		assert.Equal(t, 50, result.Limit)

		// All returned transactions should belong to the user
		for _, tx := range result.Transactions {
			assert.Equal(t, userID, tx.UserID)
		}

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test invalid time range
	t.Run("invalid_time_range", func(t *testing.T) {
		request := &services.TransactionQueryRequest{}

		// Start time after end time
		invalidStartTime := time.Now()
		invalidEndTime := time.Now().Add(-24 * time.Hour)

		// Execute test
		result, err := transactionService.GetTransactionsByTimeRange(context.Background(), userID, invalidStartTime, invalidEndTime, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "start time cannot be after end time")
	})

	// Test empty result
	t.Run("no_user_transactions", func(t *testing.T) {
		request := &services.TransactionQueryRequest{}

		// Only other users' transactions
		allTransactions := []*models.Transaction{
			transactionFactory.WithUserID(anotherUserID),
		}

		expectedFilters := repositories.TransactionFilters{
			StartDate: &startTime,
			EndDate:   &endTime,
			Limit:     100, // Default limit
		}

		// Setup mock expectations
		mockTransactionRepo.On("GetByTimeRange", mock.Anything, startTime, endTime, expectedFilters).
			Return(allTransactions, int64(1), nil).Once()

		// Execute test
		result, err := transactionService.GetTransactionsByTimeRange(context.Background(), userID, startTime, endTime, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 0)   // No user transactions
		assert.Equal(t, int64(0), result.Total) // Filtered total

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})
}

// TestTransactionService_ListAllTransactions tests the ListAllTransactions functionality (admin)
func TestTransactionService_ListAllTransactions(t *testing.T) {
	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	transactionFactory := helpers.NewTransactionFactory()

	// Test successful retrieval with default time range
	t.Run("successful_retrieval_default", func(t *testing.T) {
		request := &services.TransactionQueryRequest{
			Limit: 75,
		}

		testTransactions := []*models.Transaction{
			transactionFactory.Build(),
			transactionFactory.Build(),
		}

		// The service will use a broad time range (2020-01-01 to now)
		// We need to match these times in our mock

		// Setup mock expectations - we expect the call with any start/end time
		mockTransactionRepo.On("GetByTimeRange", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("repositories.TransactionFilters")).
			Return(testTransactions, int64(2), nil).Once()

		// Execute test
		result, err := transactionService.ListAllTransactions(context.Background(), request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 75, result.Limit)

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test with custom date range
	t.Run("successful_retrieval_custom_dates", func(t *testing.T) {
		startDate := time.Now().Add(-48 * time.Hour)
		endDate := time.Now().Add(-24 * time.Hour)

		request := &services.TransactionQueryRequest{
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     20,
		}

		testTransactions := []*models.Transaction{
			transactionFactory.Build(),
		}

		expectedFilters := repositories.TransactionFilters{
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     20,
		}

		// Setup mock expectations
		mockTransactionRepo.On("GetByTimeRange", mock.Anything, startDate, endDate, expectedFilters).
			Return(testTransactions, int64(1), nil).Once()

		// Execute test
		result, err := transactionService.ListAllTransactions(context.Background(), request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Transactions, 1)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, 20, result.Limit)

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})
}

// TestTransactionService_GetTransactionByID tests the GetTransactionByID functionality (admin)
func TestTransactionService_GetTransactionByID(t *testing.T) {
	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	transactionID := uuid.New()
	transactionFactory := helpers.NewTransactionFactory()
	testTransaction := transactionFactory.Build()
	testTransaction.ID = transactionID

	// Test successful retrieval (admin)
	t.Run("successful_admin_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(testTransaction, nil).Once()

		// Execute test
		result, err := transactionService.GetTransactionByID(context.Background(), transactionID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, transactionID, result.ID)
		// Admin can access any user's transaction

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})

	// Test transaction not found (admin)
	t.Run("transaction_not_found_admin", func(t *testing.T) {
		// Setup mock expectations
		mockTransactionRepo.On("GetByID", mock.Anything, transactionID).
			Return(nil, nil).Once()

		// Execute test
		result, err := transactionService.GetTransactionByID(context.Background(), transactionID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction not found")

		// Verify mock expectations
		mockTransactionRepo.AssertExpectations(t)
	})
}

// Performance test for transaction operations
func TestTransactionService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	transactionService := services.NewTransactionService(repos)

	// Create test data
	userID := uuid.New()
	transactionFactory := helpers.NewTransactionFactory()
	testTransactions := []*models.Transaction{
		transactionFactory.WithUserID(userID),
		transactionFactory.WithUserID(userID),
		transactionFactory.WithUserID(userID),
	}

	t.Run("get_user_transactions_performance", func(t *testing.T) {
		request := &services.TransactionQueryRequest{
			Limit: 10,
		}

		expectedFilters := repositories.TransactionFilters{
			Limit: 10,
		}

		// Setup mock for multiple calls
		mockTransactionRepo.On("GetByUserID", mock.Anything, userID, expectedFilters).
			Return(testTransactions, int64(3), nil).Times(100)

		timer := helpers.NewPerformanceTimer()
		timer.Start()

		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := transactionService.GetUserTransactions(context.Background(), userID, request)
			require.NoError(t, err)
		}

		timer.Stop()

		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 GetUserTransactions operations should complete within 1 second")

		mockTransactionRepo.AssertExpectations(t)
	})
}
