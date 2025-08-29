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

// TestTradingService_CreateTrading tests the CreateTrading functionality
func TestTradingService_CreateTrading(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		ExchangeBinding: mockExchangeBindingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	userID := uuid.New()

	// Test successful trading creation
	t.Run("successful_creation", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:              "binance-main",
			Type:              "real",
			ExchangeBindingID: uuid.New(),
		}

		// Setup mock expectations for exchange binding validation
		mockExchangeBindingRepo.On("GetByID", mock.Anything, request.ExchangeBindingID).
			Return(&models.ExchangeBinding{
				ID:     request.ExchangeBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
			
		// Setup mock expectations for trading creation
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Run(func(args mock.Arguments) {
				// Simulate database loading the ExchangeBinding relationship
				trading := args.Get(1).(*models.Trading)
				trading.ExchangeBinding = models.ExchangeBinding{
					ID:       request.ExchangeBindingID,
					UserID:   &userID,
					Name:     "Binance Main",
					Exchange: "binance",
					Type:     "private",
					APIKey:   "test_key",
				}
			}).Return(nil).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, request.Name, result.Name)
		assert.Equal(t, request.Type, result.Type)
		assert.Equal(t, "active", result.Status)
		// ExchangeBinding information should be included
		assert.NotNil(t, result.ExchangeBinding)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})


	// Test duplicate trading name - now handled by database constraint
	t.Run("duplicate_trading_name", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:              "existing-trading",
			Type:              "real",
			ExchangeBindingID: uuid.New(),
		}

		// Setup mock expectations for exchange binding validation
		mockExchangeBindingRepo.On("GetByID", mock.Anything, request.ExchangeBindingID).
			Return(&models.ExchangeBinding{
				ID:     request.ExchangeBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
			
		// Setup mock expectations - database returns unique constraint error
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"tradings_user_name_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading name already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test duplicate API key error - now handled by database constraint
	t.Run("duplicate_api_key", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:              "new-trading",
			Type:              "real",
			ExchangeBindingID: uuid.New(),
		}

		// Setup mock expectations for exchange binding validation
		mockExchangeBindingRepo.On("GetByID", mock.Anything, request.ExchangeBindingID).
			Return(&models.ExchangeBinding{
				ID:     request.ExchangeBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
			
		// Setup mock expectations - database returns unique constraint error
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"tradings_user_api_key_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api key already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test duplicate API secret error - now handled by database constraint
	t.Run("duplicate_api_secret", func(t *testing.T) {
		request := &services.CreateTradingRequest{
			Name:              "new-trading",
			Type:              "real",
			ExchangeBindingID: uuid.New(),
		}

		// Setup mock expectations for exchange binding validation
		mockExchangeBindingRepo.On("GetByID", mock.Anything, request.ExchangeBindingID).
			Return(&models.ExchangeBinding{
				ID:     request.ExchangeBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
			
		// Setup mock expectations - database returns unique constraint error
		mockTradingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"tradings_user_api_secret_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.CreateTrading(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api secret already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})
}

// TestTradingService_GetUserTradings tests the GetUserTradings functionality
func TestTradingService_GetUserTradings(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		ExchangeBinding: mockExchangeBindingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	userID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Create test tradings
		testTradings := []*models.Trading{
			tradingFactory.WithUserID(userID),
			tradingFactory.WithUserID(userID),
		}
		testTradings[0].Name = "binance-main"
		testTradings[1].Name = "okx-trading"

		// Setup mock expectations
		mockTradingRepo.On("GetByUserID", mock.Anything, userID).
			Return(testTradings, nil).Once()

		// Execute test
		result, err := tradingService.GetUserTradings(context.Background(), userID)

		// Verify results
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "binance-main", result[0].Name)
		assert.Equal(t, "okx-trading", result[1].Name)

		// Verify exchange binding information is included
		for _, trading := range result {
			if trading.ExchangeBinding != nil {
				assert.NotEmpty(t, trading.ExchangeBinding.Name)
			}
		}

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test empty result
	t.Run("no_tradings", func(t *testing.T) {
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
		mockExchangeBindingRepo.AssertExpectations(t)
	})
}

// TestTradingService_GetTrading tests the GetTrading functionality
func TestTradingService_GetTrading(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		ExchangeBinding: mockExchangeBindingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTrading := tradingFactory.WithUserID(userID)
	testTrading.ID = tradingID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()

		// Execute test
		result, err := tradingService.GetTrading(context.Background(), userID, tradingID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, tradingID, result.ID)
		assert.Equal(t, userID, result.UserID)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test trading not found
	t.Run("trading_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingService.GetTrading(context.Background(), userID, tradingID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test trading belongs to different user
	t.Run("trading_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserTrading := tradingFactory.WithUserID(differentUserID)
		wrongUserTrading.ID = tradingID

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(wrongUserTrading, nil).Once()

		// Execute test
		result, err := tradingService.GetTrading(context.Background(), userID, tradingID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})
}

// TestTradingService_UpdateTrading tests the UpdateTrading functionality
func TestTradingService_UpdateTrading(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		ExchangeBinding: mockExchangeBindingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	exchangeBindingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTrading := tradingFactory.WithUserID(userID)
	testTrading.ID = tradingID
	testTrading.Name = "original-name"
	testTrading.ExchangeBindingID = exchangeBindingID
	testTrading.ExchangeBinding = models.ExchangeBinding{
		ID:       exchangeBindingID,
		UserID:   &userID,
		Name:     "Test Exchange Binding",
		Exchange: "binance",
		Type:     "private",
		APIKey:   "test_key",
	}

	// Test successful name update
	t.Run("successful_name_update", func(t *testing.T) {
		newName := "updated-trading-name"
		request := &services.UpdateTradingRequest{
			Name: &newName,
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		mockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(nil).Once()

		// Execute test
		result, err := tradingService.UpdateTrading(context.Background(), userID, tradingID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newName, result.Name)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test name conflict with another trading - now handled by database constraint
	t.Run("name_conflict", func(t *testing.T) {
		conflictingName := "existing-trading"
		request := &services.UpdateTradingRequest{
			Name: &conflictingName,
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		// Database returns unique constraint error with specific constraint name
		mockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"tradings_user_name_active_unique\"")).Once()

		// Execute test
		result, err := tradingService.UpdateTrading(context.Background(), userID, tradingID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading name already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test API key and status update
	t.Run("successful_api_key_status_update", func(t *testing.T) {
		newBindingID := uuid.New()
		newStatus := "inactive"
		request := &services.UpdateTradingRequest{
			ExchangeBindingID: &newBindingID,
			Status:            &newStatus,
		}

		// Setup mock expectations - UpdateTrading calls GetByID once
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		// Exchange binding validation mock expectations
		mockExchangeBindingRepo.On("GetByID", mock.Anything, newBindingID).
			Return(&models.ExchangeBinding{
				ID:     newBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
			
		mockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Run(func(args mock.Arguments) {
				// Simulate database update - update the trading's ExchangeBinding fields
				trading := args.Get(1).(*models.Trading)
				trading.ExchangeBindingID = newBindingID
				trading.ExchangeBinding = models.ExchangeBinding{
					ID:       newBindingID,
					UserID:   &userID,
					Name:     "Updated Exchange Binding",
					Exchange: "binance",
					Type:     "private",
					APIKey:   "new_key",
				}
			}).Return(nil).Once()

		// Execute test
		result, err := tradingService.UpdateTrading(context.Background(), userID, tradingID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newStatus, result.Status)
		// Exchange binding information should be included
		assert.NotNil(t, result.ExchangeBinding)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test API key conflict with another trading - now handled by database constraint
	t.Run("api_key_conflict", func(t *testing.T) {
		// Create fresh mocks for this test
		freshMockTradingRepo := &mocks.MockTradingRepository{}
		freshMockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}
		freshRepos := &repositories.Repositories{
			User:            &mocks.MockUserRepository{},
			Trading:         freshMockTradingRepo,
			ExchangeBinding: freshMockExchangeBindingRepo,
			SubAccount:      &mocks.MockSubAccountRepository{},
			Transaction:     &mocks.MockTransactionRepository{},
			TradingLog:      &mocks.MockTradingLogRepository{},
			OAuthToken:      &mocks.MockOAuthTokenRepository{},
			EventProcessing: &mocks.MockEventProcessingRepository{},
		}
		freshExchangeBindingService := services.NewExchangeBindingService(freshRepos.ExchangeBinding)
		freshTradingService := services.NewTradingService(freshRepos, freshExchangeBindingService)

		conflictingBindingID := uuid.New()
		request := &services.UpdateTradingRequest{
			ExchangeBindingID: &conflictingBindingID,
		}

		// Setup mock expectations
		freshMockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		// Exchange binding validation mock expectations
		freshMockExchangeBindingRepo.On("GetByID", mock.Anything, conflictingBindingID).
			Return(&models.ExchangeBinding{
				ID:     conflictingBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
		
		// Database returns unique constraint error with specific constraint name
		freshMockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"tradings_user_api_key_active_unique\"")).Once()

		// Execute test
		result, err := freshTradingService.UpdateTrading(context.Background(), userID, tradingID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api key already exists")

		// Verify mock expectations
		freshMockTradingRepo.AssertExpectations(t)
		freshMockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test API secret conflict with another trading - now handled by database constraint
	t.Run("api_secret_conflict", func(t *testing.T) {
		// Create fresh mocks for this test
		freshMockTradingRepo := &mocks.MockTradingRepository{}
		freshMockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}
		freshRepos := &repositories.Repositories{
			User:            &mocks.MockUserRepository{},
			Trading:         freshMockTradingRepo,
			ExchangeBinding: freshMockExchangeBindingRepo,
			SubAccount:      &mocks.MockSubAccountRepository{},
			Transaction:     &mocks.MockTransactionRepository{},
			TradingLog:      &mocks.MockTradingLogRepository{},
			OAuthToken:      &mocks.MockOAuthTokenRepository{},
			EventProcessing: &mocks.MockEventProcessingRepository{},
		}
		freshExchangeBindingService := services.NewExchangeBindingService(freshRepos.ExchangeBinding)
		freshTradingService := services.NewTradingService(freshRepos, freshExchangeBindingService)

		conflictingBindingID := uuid.New()
		request := &services.UpdateTradingRequest{
			ExchangeBindingID: &conflictingBindingID,
		}

		// Setup mock expectations
		freshMockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		// Exchange binding validation mock expectations
		freshMockExchangeBindingRepo.On("GetByID", mock.Anything, conflictingBindingID).
			Return(&models.ExchangeBinding{
				ID:     conflictingBindingID,
				UserID: &userID, // Private binding owned by user
				Type:   "private",
			}, nil).Once()
		
		// Database returns unique constraint error with specific constraint name
		freshMockTradingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Trading")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"tradings_user_api_secret_active_unique\"")).Once()

		// Execute test
		result, err := freshTradingService.UpdateTrading(context.Background(), userID, tradingID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "api secret already exists")

		// Verify mock expectations
		freshMockTradingRepo.AssertExpectations(t)
		freshMockExchangeBindingRepo.AssertExpectations(t)
	})
}

// TestTradingService_DeleteTrading tests the DeleteTrading functionality
func TestTradingService_DeleteTrading(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}
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

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTrading := tradingFactory.WithUserID(userID)
	testTrading.ID = tradingID

	// Test successful deletion
	t.Run("successful_deletion", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		mockSubAccountRepo.On("GetByTradingID", mock.Anything, tradingID).
			Return([]*models.SubAccount{}, nil).Once() // No sub-accounts
		mockTradingRepo.On("Delete", mock.Anything, tradingID).
			Return(nil).Once()

		// Execute test
		err := tradingService.DeleteTrading(context.Background(), userID, tradingID)

		// Verify results
		require.NoError(t, err)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test deletion with existing sub-accounts
	t.Run("deletion_with_subaccounts", func(t *testing.T) {
		subAccountFactory := helpers.NewSubAccountFactory()
		existingSubAccounts := []*models.SubAccount{
			subAccountFactory.WithUserAndTrading(userID, tradingID),
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		mockSubAccountRepo.On("GetByTradingID", mock.Anything, tradingID).
			Return(existingSubAccounts, nil).Once()

		// Execute test
		err := tradingService.DeleteTrading(context.Background(), userID, tradingID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete trading with existing sub-accounts")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test trading not found
	t.Run("trading_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(nil, nil).Once()

		// Execute test
		err := tradingService.DeleteTrading(context.Background(), userID, tradingID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trading not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})
}

// TestTradingService_GetTradingByID tests the GetTradingByID functionality (admin)
func TestTradingService_GetTradingByID(t *testing.T) {
	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		ExchangeBinding: mockExchangeBindingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTrading := tradingFactory.Build()
	testTrading.ID = tradingID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()

		// Execute test
		result, err := tradingService.GetTradingByID(context.Background(), tradingID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, tradingID, result.ID)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})

	// Test trading not found
	t.Run("trading_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(nil, nil).Once()

		// Execute test
		result, err := tradingService.GetTradingByID(context.Background(), tradingID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockExchangeBindingRepo.AssertExpectations(t)
	})
}

// Performance test for trading operations
func TestTradingService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockExchangeBindingRepo := &mocks.MockExchangeBindingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         mockTradingRepo,
		ExchangeBinding: mockExchangeBindingRepo,
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create exchange binding service and trading service
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)

	// Create test data
	userID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTradings := []*models.Trading{
		tradingFactory.WithUserID(userID),
		tradingFactory.WithUserID(userID),
		tradingFactory.WithUserID(userID),
	}

	t.Run("get_user_tradings_performance", func(t *testing.T) {
		// Setup mock for multiple calls
		mockTradingRepo.On("GetByUserID", mock.Anything, userID).
			Return(testTradings, nil).Times(100)

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
