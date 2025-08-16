package test

import (
	"context"
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

// TestExchangeService_CreateExchange tests the CreateExchange functionality
func TestExchangeService_CreateExchange(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	userID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	
	// Test successful exchange creation
	t.Run("successful_creation", func(t *testing.T) {
		request := &services.CreateExchangeRequest{
			Name:      "binance-main",
			Type:      "binance",
			APIKey:    "test_api_key_12345",
			APISecret: "test_api_secret_67890",
		}
		
		// Setup mock expectations - no existing exchanges
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return([]*models.Exchange{}, nil).Once()
		mockExchangeRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Exchange")).
			Return(nil).Once()
		
		// Execute test
		result, err := exchangeService.CreateExchange(context.Background(), userID, request)
		
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
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test maximum exchanges limit
	t.Run("maximum_exchanges_reached", func(t *testing.T) {
		request := &services.CreateExchangeRequest{
			Name:      "new-exchange",
			Type:      "binance",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
		}
		
		// Create 10 existing exchanges
		existingExchanges := make([]*models.Exchange, 10)
		for i := 0; i < 10; i++ {
			exchange := exchangeFactory.Build()
			exchange.UserID = userID
			existingExchanges[i] = exchange
		}
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return(existingExchanges, nil).Once()
		
		// Execute test
		result, err := exchangeService.CreateExchange(context.Background(), userID, request)
		
		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "maximum number of exchanges reached")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test duplicate exchange name
	t.Run("duplicate_exchange_name", func(t *testing.T) {
		existingName := "existing-exchange"
		request := &services.CreateExchangeRequest{
			Name:      existingName,
			Type:      "binance",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
		}
		
		// Create existing exchange with same name
		existingExchange := exchangeFactory.Build()
		existingExchange.UserID = userID
		existingExchange.Name = existingName
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return([]*models.Exchange{existingExchange}, nil).Once()
		
		// Execute test
		result, err := exchangeService.CreateExchange(context.Background(), userID, request)
		
		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange name already exists")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// TestExchangeService_GetUserExchanges tests the GetUserExchanges functionality
func TestExchangeService_GetUserExchanges(t *testing.T) {
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	userID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	
	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Create test exchanges
		testExchanges := []*models.Exchange{
			exchangeFactory.WithUserID(userID),
			exchangeFactory.WithUserID(userID),
		}
		testExchanges[0].Name = "binance-main"
		testExchanges[1].Name = "okx-trading"
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return(testExchanges, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetUserExchanges(context.Background(), userID)
		
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
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test empty result
	t.Run("no_exchanges", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return([]*models.Exchange{}, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetUserExchanges(context.Background(), userID)
		
		// Verify results
		require.NoError(t, err)
		assert.Len(t, result, 0)
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// TestExchangeService_GetExchange tests the GetExchange functionality
func TestExchangeService_GetExchange(t *testing.T) {
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID
	
	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetExchange(context.Background(), userID, exchangeID)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, exchangeID, result.ID)
		assert.Equal(t, userID, result.UserID)
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetExchange(context.Background(), userID, exchangeID)
		
		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange not found")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test exchange belongs to different user
	t.Run("exchange_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserExchange := exchangeFactory.WithUserID(differentUserID)
		wrongUserExchange.ID = exchangeID
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(wrongUserExchange, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetExchange(context.Background(), userID, exchangeID)
		
		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange not found")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// TestExchangeService_UpdateExchange tests the UpdateExchange functionality
func TestExchangeService_UpdateExchange(t *testing.T) {
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID
	testExchange.Name = "original-name"
	
	// Test successful name update
	t.Run("successful_name_update", func(t *testing.T) {
		newName := "updated-exchange-name"
		request := &services.UpdateExchangeRequest{
			Name: &newName,
		}
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return([]*models.Exchange{testExchange}, nil).Once() // Only current exchange
		mockExchangeRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Exchange")).
			Return(nil).Once()
		
		// Execute test
		result, err := exchangeService.UpdateExchange(context.Background(), userID, exchangeID, request)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newName, result.Name)
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test name conflict with another exchange
	t.Run("name_conflict", func(t *testing.T) {
		conflictingName := "existing-exchange"
		request := &services.UpdateExchangeRequest{
			Name: &conflictingName,
		}
		
		// Create another exchange with conflicting name
		anotherExchange := exchangeFactory.WithUserID(userID)
		anotherExchange.ID = uuid.New()
		anotherExchange.Name = conflictingName
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return([]*models.Exchange{testExchange, anotherExchange}, nil).Once()
		
		// Execute test
		result, err := exchangeService.UpdateExchange(context.Background(), userID, exchangeID, request)
		
		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange name already exists")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test API key and status update
	t.Run("successful_api_key_status_update", func(t *testing.T) {
		newAPIKey := "new_api_key_12345"
		newStatus := "inactive"
		request := &services.UpdateExchangeRequest{
			APIKey: &newAPIKey,
			Status: &newStatus,
		}
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockExchangeRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Exchange")).
			Return(nil).Once()
		
		// Execute test
		result, err := exchangeService.UpdateExchange(context.Background(), userID, exchangeID, request)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newStatus, result.Status)
		// API key should be masked
		assert.Contains(t, result.APIKey, "****")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// TestExchangeService_DeleteExchange tests the DeleteExchange functionality
func TestExchangeService_DeleteExchange(t *testing.T) {
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	userID := uuid.New()
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	testExchange := exchangeFactory.WithUserID(userID)
	testExchange.ID = exchangeID
	
	// Test successful deletion
	t.Run("successful_deletion", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockSubAccountRepo.On("GetByExchangeID", mock.Anything, exchangeID).
			Return([]*models.SubAccount{}, nil).Once() // No sub-accounts
		mockExchangeRepo.On("Delete", mock.Anything, exchangeID).
			Return(nil).Once()
		
		// Execute test
		err := exchangeService.DeleteExchange(context.Background(), userID, exchangeID)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})
	
	// Test deletion with existing sub-accounts
	t.Run("deletion_with_subaccounts", func(t *testing.T) {
		subAccountFactory := helpers.NewSubAccountFactory()
		existingSubAccounts := []*models.SubAccount{
			subAccountFactory.WithUserAndExchange(userID, exchangeID),
		}
		
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		mockSubAccountRepo.On("GetByExchangeID", mock.Anything, exchangeID).
			Return(existingSubAccounts, nil).Once()
		
		// Execute test
		err := exchangeService.DeleteExchange(context.Background(), userID, exchangeID)
		
		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete exchange with existing sub-accounts")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})
	
	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()
		
		// Execute test
		err := exchangeService.DeleteExchange(context.Background(), userID, exchangeID)
		
		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exchange not found")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// TestExchangeService_GetExchangeByID tests the GetExchangeByID functionality (admin)
func TestExchangeService_GetExchangeByID(t *testing.T) {
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	exchangeID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	testExchange := exchangeFactory.Build()
	testExchange.ID = exchangeID
	
	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(testExchange, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetExchangeByID(context.Background(), exchangeID)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, exchangeID, result.ID)
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
	
	// Test exchange not found
	t.Run("exchange_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockExchangeRepo.On("GetByID", mock.Anything, exchangeID).
			Return(nil, nil).Once()
		
		// Execute test
		result, err := exchangeService.GetExchangeByID(context.Background(), exchangeID)
		
		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exchange not found")
		
		// Verify mock expectations
		mockExchangeRepo.AssertExpectations(t)
	})
}

// Performance test for exchange operations
func TestExchangeService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Create mocks
	mockExchangeRepo := &mocks.MockExchangeRepository{}
	
	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        mockExchangeRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	// Create service
	exchangeService := services.NewExchangeService(repos)
	
	// Create test data
	userID := uuid.New()
	exchangeFactory := helpers.NewExchangeFactory()
	testExchanges := []*models.Exchange{
		exchangeFactory.WithUserID(userID),
		exchangeFactory.WithUserID(userID),
		exchangeFactory.WithUserID(userID),
	}
	
	t.Run("get_user_exchanges_performance", func(t *testing.T) {
		// Setup mock for multiple calls
		mockExchangeRepo.On("GetByUserID", mock.Anything, userID).
			Return(testExchanges, nil).Times(100)
		
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := exchangeService.GetUserExchanges(context.Background(), userID)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 GetUserExchanges operations should complete within 1 second")
		
		mockExchangeRepo.AssertExpectations(t)
	})
}