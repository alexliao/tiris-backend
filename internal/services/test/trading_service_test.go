package test

import (
	"context"
	"fmt"
	"testing"

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

// TestExchangeService_CreateTrading tests the CreateTrading functionality
func TestExchangeService_CreateTrading(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	userID := uuid.New()

	// Test successful exchange creation
	t.Run("successful_creation", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:      "binance-main",
			Type:      "binance",
			APIKey:    "test_api_key_12345",
			APISecret: "test_api_secret_67890",
		}

		// Setup mock expectations
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(nil).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, request.Name, result.Name)
		assert.Equal(t, request.Type, result.Type)
		assert.Equal(t, "active", result.Status)
		// API key should be masked
		assert.Contains(t, result.APIKey, "****")
		assert.NotEqual(t, request.APIKey, result.APIKey)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})


	// Test duplicate exchange name - now handled by database constraint
	t.Run("duplicate_exchange_name", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:      "existing-exchange",
			Type:      "binance",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
		}

		// Setup mock expectations - database returns unique constraint error
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"exchanges_user_name_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange name already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test duplicate API key error - now handled by database constraint
	t.Run("duplicate_api_key", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:      "new-exchange",
			Type:      "binance",
			APIKey:    "existing_api_key_123",
			APISecret: "new_api_secret",
		}

		// Setup mock expectations - database returns unique constraint error
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"exchanges_user_api_key_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api key already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test duplicate API secret error - now handled by database constraint
	t.Run("duplicate_api_secret", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:      "new-exchange",
			Type:      "binance",
			APIKey:    "new_api_key",
			APISecret: "existing_api_secret_789",
		}

		// Setup mock expectations - database returns unique constraint error
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"exchanges_user_api_secret_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api secret already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})
}

// TestExchangeService_GetUserTradings tests the GetUserTradings functionality
func TestExchangeService_GetUserTradings(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	userID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Create test exchanges
		testExchanges := []*models.Trading{
			exchangeFactory.WithUserID(userID),
			exchangeFactory.WithUserID(userID),
		}
		testExchanges[0].Name = "binance-main"
		testExchanges[1].Name = "okx-trading"

		// Setup mock expectations
		mockTradingRepo.On("GetByUserID", mock.Anything, userID).
			Return(testExchanges, nil).Once()

		// Execute test
		result, err := tradingService.GetUserTradings(context.Background(), userID)

		// Verify results
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "binance-main", result[0].Name)
		assert.Equal(t, "okx-trading", result[1].Name)

		// Verify API keys are masked
		for _, exchange := range result {
			assert.Contains(t, exchange.APIKey, "****")
		}

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test empty result
	t.Run("no_exchanges", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByUserID", mock.Anything, userID).
			Return([]*models.Trading{}, nil).Once()

		// Execute test
		result, err := tradingService.GetUserTradings(context.Background(), userID)

		// Verify results
		require.NoError(t, err)
		assert.Len(t, result, 0)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})
}

// TestExchangeService_GetTrading tests the GetTrading functionality
func TestExchangeService_GetTrading(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()

		// Execute test
		result, err := tradingService.GetTrading(context.Background(), userID, exchangeID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, exchangeID, result.ID)
		assert.Equal(t, userID, result.UserID)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingService.GetTrading(context.Background(), userID, exchangeID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test exchange belongs to different user
	t.Run("exchange_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserExchange := exchangeFactory.WithUserID(differentUserID)
		wrongUserExchange.ID = exchangeID

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(wrongUserExchange, nil).Once()

		// Execute test
		result, err := tradingService.GetTrading(context.Background(), userID, exchangeID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})
}

// TestExchangeService_UpdateTrading tests the UpdateTrading functionality
func TestExchangeService_UpdateTrading(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID
	testExchange.Name = "original-name"

	// Test successful name update
	t.Run("successful_name_update", func(t *testing.T) {
		newName := "updated-exchange-name"
		request := &services.UpdateTradingRequest{
			Name: &newName,
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(nil).Once()

		// Execute test
		result, err := tradingService.UpdateTrading(context.Background(), userID, exchangeID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newName, result.Name)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test name conflict with another exchange - now handled by database constraint
	t.Run("name_conflict", func(t *testing.T) {
		conflictingName := "existing-exchange"
		request := &services.UpdateTradingRequest{
			Name: &conflictingName,
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		// Database returns unique constraint error with specific constraint name
		mockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"exchanges_user_name_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.UpdateTrading(context.Background(), userID, exchangeID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange name already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test API key and status update
	t.Run("successful_api_key_status_update", func(t *testing.T) {
		newAPIKey := "new_api_key_12345"
		newStatus := "inactive"
		request := &services.UpdateTradingRequest{
			APIKey: &newAPIKey,
			Status: &newStatus,
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(nil).Once()

		// Execute test
		result, err := tradingService.UpdateTrading(context.Background(), userID, exchangeID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newStatus, result.Status)
		// API key should be masked
		assert.Contains(t, result.APIKey, "****")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test API key conflict with another exchange - now handled by database constraint
	t.Run("api_key_conflict", func(t *testing.T) {
		// Create fresh mocks for this test
		freshMockExchangeRepo := &mocks.MockTradingRepository{}
		freshRepos := &repositories.Repositories{
			User:            &mocks.MockUserRepository{},
			Trading:        freshMockExchangeRepo,
			SubAccount:      &mocks.MockSubAccountRepository{},
			Transaction:     &mocks.MockTransactionRepository{},
			TradingLog:      &mocks.MockTradingLogRepository{},
			OAuthToken:      &mocks.MockOAuthTokenRepository{},
			EventProcessing: &mocks.MockEventProcessingRepository{},
		}
		freshExchangeService := services.NewTradingService(freshRepos)

		conflictingAPIKey := "existing_api_key_456"
		request := &services.UpdateTradingRequest{
			APIKey: &conflictingAPIKey,
		}

		// Setup mock expectations
		freshMockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		// Database returns unique constraint error with specific constraint name
		freshMockExchangeRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"exchanges_user_api_key_active_unique\"")).Once()

		// Execute test
		result, err := freshExchangeService.UpdateTrading(context.Background(), userID, exchangeID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api key already exists")

		// Verify mock expectations
		freshMockExchangeRepo.AssertExpectations(t)
	})

	// Test API secret conflict with another exchange - now handled by database constraint
	t.Run("api_secret_conflict", func(t *testing.T) {
		// Create fresh mocks for this test
		freshMockExchangeRepo := &mocks.MockTradingRepository{}
		freshRepos := &repositories.Repositories{
			User:            &mocks.MockUserRepository{},
			Trading:        freshMockExchangeRepo,
			SubAccount:      &mocks.MockSubAccountRepository{},
			Transaction:     &mocks.MockTransactionRepository{},
			TradingLog:      &mocks.MockTradingLogRepository{},
			OAuthToken:      &mocks.MockOAuthTokenRepository{},
			EventProcessing: &mocks.MockEventProcessingRepository{},
		}
		freshExchangeService := services.NewTradingService(freshRepos)

		conflictingAPISecret := "existing_api_secret_456"
		request := &services.UpdateTradingRequest{
			APISecret: &conflictingAPISecret,
		}

		// Setup mock expectations
		freshMockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		// Database returns unique constraint error with specific constraint name
		freshMockExchangeRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"exchanges_user_api_secret_active_unique\"")).Once()

		// Execute test
		result, err := freshExchangeService.UpdateTrading(context.Background(), userID, exchangeID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api secret already exists")

		// Verify mock expectations
		freshMockExchangeRepo.AssertExpectations(t)
	})
}

// TestExchangeService_DeleteTrading tests the DeleteTrading functionality
func TestExchangeService_DeleteTrading(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID

	// Test successful deletion
	t.Run("successful_deletion", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockSubAccountRepo.On("GetByTradingID", mock.Anything, exchangeID).
			Return([]*models.SubAccount{}, nil).Once() // No sub-accounts
		mockTradingRepo.On("Delete", mock.Anything, exchangeID).
			Return(nil).Once()

		// Execute test
		err := tradingService.DeleteTrading(context.Background(), userID, exchangeID)

		// Verify results
		require.NoError(t, err)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test deletion with existing sub-accounts
	t.Run("deletion_with_subaccounts", func(t *testing.T) {
		subAccountFactory := helpers.NewSubAccountFactory()
		existingSubAccounts := []*models.SubAccount{
			subAccountFactory.WithUserAndTrading(userID, exchangeID),
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockSubAccountRepo.On("GetByTradingID", mock.Anything, exchangeID).
			Return(existingSubAccounts, nil).Once()

		// Execute test
		err := tradingService.DeleteTrading(context.Background(), userID, exchangeID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete trading platform with existing sub-accounts")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()

		// Execute test
		err := tradingService.DeleteTrading(context.Background(), userID, exchangeID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})
}

// TestExchangeService_GetTradingByID tests the GetTradingByID functionality (admin)
func TestExchangeService_GetTradingByID(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()
	testExchange := exchangeFactory.Build()
	testExchange.ID = exchangeID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()

		// Execute test
		result, err := tradingService.GetTradingByID(context.Background(), exchangeID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, exchangeID, result.ID)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingService.GetTradingByID(context.Background(), exchangeID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading platform not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})
}

// Performance test for exchange operations
func TestExchangeService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	tradingService := services.NewTradingService(repos)

	// Create test data
	userID := uuid.New()
	exchangeFactory := helpers.NewTradingFactory()
	testExchanges := []*models.Trading{
		exchangeFactory.WithUserID(userID),
		exchangeFactory.WithUserID(userID),
		exchangeFactory.WithUserID(userID),
	}

	t.Run("get_user_exchanges_performance", func(t *testing.T) {
		// Setup mock for multiple calls
		mockTradingRepo.On("GetByUserID", mock.Anything, userID).
			Return(testExchanges, nil).Times(100)

		timer := helpers.NewPerformanceTimer()
		timer.Start()

		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := tradingService.GetUserTradings(context.Background(), userID)
			require.NoError(t, err)
		}

		timer.Stop()

		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 GetUserTradings operations should complete within 1 second")

		mockTradingRepo.AssertExpectations(t)
	})
}
