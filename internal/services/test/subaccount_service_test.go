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

// TestSubAccountService_CreateSubAccount tests the CreateSubAccount functionality
func TestSubAccountService_CreateSubAccount(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        mockTradingRepo,
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTrading := tradingFactory.WithUserID(userID)
	testTrading.ID = tradingID

	// Test successful sub-account creation
	t.Run("successful_creation", func(t *testing.T) {
		request := &services.CreateSubAccountRequest{
			TradingID: tradingID,
			Name:       "main-spot",
			Symbol:     "USDT",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		mockSubAccountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.SubAccount")).
			Return(nil).Once()

		// Execute test
		result, err := subAccountService.CreateSubAccount(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, tradingID, result.TradingID)
		assert.Equal(t, request.Name, result.Name)
		assert.Equal(t, request.Symbol, result.Symbol)
		assert.Equal(t, 0.0, result.Balance) // New accounts start with zero balance

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test trading not found
	t.Run("trading_not_found", func(t *testing.T) {
		request := &services.CreateSubAccountRequest{
			TradingID: tradingID,
			Name:       "test-account",
			Symbol:     "USDT",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(nil, nil).Once()

		// Execute test
		result, err := subAccountService.CreateSubAccount(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test trading belongs to different user
	t.Run("trading_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserTrading := tradingFactory.WithUserID(differentUserID)
		wrongUserTrading.ID = tradingID

		request := &services.CreateSubAccountRequest{
			TradingID: tradingID,
			Name:       "test-account",
			Symbol:     "USDT",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(wrongUserTrading, nil).Once()

		// Execute test
		result, err := subAccountService.CreateSubAccount(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trading not found")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
	})

	// Test duplicate sub-account name - now handled by database constraint
	t.Run("duplicate_name", func(t *testing.T) {
		request := &services.CreateSubAccountRequest{
			TradingID: tradingID,
			Name:       "duplicate-name",
			Symbol:     "USDT",
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		// Database returns unique constraint error with specific constraint name
		mockSubAccountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.SubAccount")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"sub_accounts_trading_name_active_unique\"")).Once()

		// Execute test
		result, err := subAccountService.CreateSubAccount(context.Background(), userID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account name already exists")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestSubAccountService_GetUserSubAccounts tests the GetUserSubAccounts functionality
func TestSubAccountService_GetUserSubAccounts(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	mockTradingRepo := &mocks.MockTradingRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        mockTradingRepo,
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()

	// Test successful retrieval without trading filter
	t.Run("successful_retrieval_all", func(t *testing.T) {
		testSubAccounts := []*models.SubAccount{
			subAccountFactory.WithUserAndTrading(userID, tradingID),
			subAccountFactory.WithUserAndTrading(userID, tradingID),
		}
		testSubAccounts[0].Name = "spot-account"
		testSubAccounts[1].Name = "futures-account"

		// Setup mock expectations
		mockSubAccountRepo.On("GetByUserID", mock.Anything, userID, (*uuid.UUID)(nil)).
			Return(testSubAccounts, nil).Once()

		// Execute test
		result, err := subAccountService.GetUserSubAccounts(context.Background(), userID, nil)

		// Verify results
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "spot-account", result[0].Name)
		assert.Equal(t, "futures-account", result[1].Name)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test successful retrieval with trading filter
	t.Run("successful_retrieval_filtered", func(t *testing.T) {
		tradingFactory := helpers.NewTradingFactory()
		testTrading := tradingFactory.WithUserID(userID)
		testTrading.ID = tradingID

		testSubAccounts := []*models.SubAccount{
			subAccountFactory.WithUserAndTrading(userID, tradingID),
		}
		testSubAccounts[0].Name = "binance-spot"

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		mockSubAccountRepo.On("GetByUserID", mock.Anything, userID, &tradingID).
			Return(testSubAccounts, nil).Once()

		// Execute test
		result, err := subAccountService.GetUserSubAccounts(context.Background(), userID, &tradingID)

		// Verify results
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "binance-spot", result[0].Name)
		assert.Equal(t, tradingID, result[0].TradingID)

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test empty result
	t.Run("no_subaccounts", func(t *testing.T) {
		// Setup mock expectations
		mockSubAccountRepo.On("GetByUserID", mock.Anything, userID, (*uuid.UUID)(nil)).
			Return([]*models.SubAccount{}, nil).Once()

		// Execute test
		result, err := subAccountService.GetUserSubAccounts(context.Background(), userID, nil)

		// Verify results
		require.NoError(t, err)
		assert.Len(t, result, 0)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestSubAccountService_GetSubAccount tests the GetSubAccount functionality
func TestSubAccountService_GetSubAccount(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccount := subAccountFactory.WithUserAndTrading(userID, tradingID)
	testSubAccount.ID = subAccountID

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()

		// Execute test
		result, err := subAccountService.GetSubAccount(context.Background(), userID, subAccountID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, subAccountID, result.ID)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, tradingID, result.TradingID)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test sub-account not found
	t.Run("subaccount_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(nil, nil).Once()

		// Execute test
		result, err := subAccountService.GetSubAccount(context.Background(), userID, subAccountID)

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
		wrongUserSubAccount := subAccountFactory.WithUserAndTrading(differentUserID, tradingID)
		wrongUserSubAccount.ID = subAccountID

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(wrongUserSubAccount, nil).Once()

		// Execute test
		result, err := subAccountService.GetSubAccount(context.Background(), userID, subAccountID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestSubAccountService_UpdateSubAccount tests the UpdateSubAccount functionality
func TestSubAccountService_UpdateSubAccount(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccount := subAccountFactory.WithUserAndTrading(userID, tradingID)
	testSubAccount.ID = subAccountID
	testSubAccount.Name = "original-name"
	testSubAccount.Symbol = "USDT"
	testSubAccount.Balance = 1000.0

	// Test successful name update
	t.Run("successful_name_update", func(t *testing.T) {
		newName := "updated-account-name"
		request := &services.UpdateSubAccountRequest{
			Name: &newName,
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		mockSubAccountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.SubAccount")).
			Return(nil).Once()

		// Execute test
		result, err := subAccountService.UpdateSubAccount(context.Background(), userID, subAccountID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newName, result.Name)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test name conflict - now handled by database constraint
	t.Run("name_conflict", func(t *testing.T) {
		conflictingName := "existing-account"
		request := &services.UpdateSubAccountRequest{
			Name: &conflictingName,
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		// Database returns unique constraint error with specific constraint name
		mockSubAccountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.SubAccount")).
			Return(fmt.Errorf("duplicate key value violates unique constraint \"sub_accounts_trading_name_active_unique\"")).Once()

		// Execute test
		result, err := subAccountService.UpdateSubAccount(context.Background(), userID, subAccountID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sub-account name already exists")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test symbol and balance update
	t.Run("successful_symbol_balance_update", func(t *testing.T) {
		newSymbol := "BTC"
		newBalance := 2500.0
		request := &services.UpdateSubAccountRequest{
			Symbol:  &newSymbol,
			Balance: &newBalance,
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		mockSubAccountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.SubAccount")).
			Return(nil).Once()

		// Execute test
		result, err := subAccountService.UpdateSubAccount(context.Background(), userID, subAccountID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newSymbol, result.Symbol)
		assert.Equal(t, newBalance, result.Balance)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestSubAccountService_UpdateBalance tests the UpdateBalance functionality
func TestSubAccountService_UpdateBalance(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccount := subAccountFactory.WithUserAndTrading(userID, tradingID)
	testSubAccount.ID = subAccountID
	testSubAccount.Balance = 1000.0

	// Test successful credit update
	t.Run("successful_credit", func(t *testing.T) {
		request := &services.UpdateBalanceRequest{
			Amount:    500.0,
			Direction: "credit",
			Reason:    "deposit",
			Info:      map[string]interface{}{"method": "bank_transfer"},
		}

		transactionID := uuid.New()
		expectedNewBalance := 1500.0

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Times(2) // Called twice: once in UpdateBalance, once in GetSubAccount
		mockSubAccountRepo.On("UpdateBalance", mock.Anything, subAccountID, expectedNewBalance, request.Amount, request.Direction, request.Reason, request.Info).
			Return(&transactionID, nil).Once()

		// Execute test
		result, err := subAccountService.UpdateBalance(context.Background(), userID, subAccountID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test successful debit update
	t.Run("successful_debit", func(t *testing.T) {
		request := &services.UpdateBalanceRequest{
			Amount:    300.0,
			Direction: "debit",
			Reason:    "withdrawal",
			Info:      map[string]interface{}{"address": "0x123..."},
		}

		transactionID := uuid.New()
		expectedNewBalance := 700.0

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Times(2)
		mockSubAccountRepo.On("UpdateBalance", mock.Anything, subAccountID, expectedNewBalance, request.Amount, request.Direction, request.Reason, request.Info).
			Return(&transactionID, nil).Once()

		// Execute test
		result, err := subAccountService.UpdateBalance(context.Background(), userID, subAccountID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test insufficient balance
	t.Run("insufficient_balance", func(t *testing.T) {
		request := &services.UpdateBalanceRequest{
			Amount:    1500.0, // More than current balance (1000.0)
			Direction: "debit",
			Reason:    "withdrawal",
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()

		// Execute test
		result, err := subAccountService.UpdateBalance(context.Background(), userID, subAccountID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "insufficient balance")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test invalid direction
	t.Run("invalid_direction", func(t *testing.T) {
		request := &services.UpdateBalanceRequest{
			Amount:    500.0,
			Direction: "invalid",
			Reason:    "test",
		}

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()

		// Execute test
		result, err := subAccountService.UpdateBalance(context.Background(), userID, subAccountID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid direction")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestSubAccountService_DeleteSubAccount tests the DeleteSubAccount functionality
func TestSubAccountService_DeleteSubAccount(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccount := subAccountFactory.WithUserAndTrading(userID, tradingID)
	testSubAccount.ID = subAccountID
	testSubAccount.Balance = 0.0 // Zero balance for successful deletion

	// Test successful deletion
	t.Run("successful_deletion", func(t *testing.T) {
		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(testSubAccount, nil).Once()
		mockSubAccountRepo.On("Delete", mock.Anything, subAccountID).
			Return(nil).Once()

		// Execute test
		err := subAccountService.DeleteSubAccount(context.Background(), userID, subAccountID)

		// Verify results
		require.NoError(t, err)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test deletion with positive balance
	t.Run("deletion_with_balance", func(t *testing.T) {
		subAccountWithBalance := subAccountFactory.WithUserAndTrading(userID, tradingID)
		subAccountWithBalance.ID = subAccountID
		subAccountWithBalance.Balance = 100.0 // Positive balance

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(subAccountWithBalance, nil).Once()

		// Execute test
		err := subAccountService.DeleteSubAccount(context.Background(), userID, subAccountID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete sub-account with positive balance")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test sub-account not found
	t.Run("subaccount_not_found", func(t *testing.T) {
		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(nil, nil).Once()

		// Execute test
		err := subAccountService.DeleteSubAccount(context.Background(), userID, subAccountID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test sub-account belongs to different user
	t.Run("subaccount_wrong_user", func(t *testing.T) {
		differentUserID := uuid.New()
		wrongUserSubAccount := subAccountFactory.WithUserAndTrading(differentUserID, tradingID)
		wrongUserSubAccount.ID = subAccountID
		wrongUserSubAccount.Balance = 0.0

		// Setup mock expectations
		mockSubAccountRepo.On("GetByID", mock.Anything, subAccountID).
			Return(wrongUserSubAccount, nil).Once()

		// Execute test
		err := subAccountService.DeleteSubAccount(context.Background(), userID, subAccountID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sub-account not found")

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// TestSubAccountService_GetSubAccountsBySymbol tests the GetSubAccountsBySymbol functionality
func TestSubAccountService_GetSubAccountsBySymbol(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()

	// Test successful retrieval by symbol
	t.Run("successful_retrieval", func(t *testing.T) {
		symbol := "USDT"
		testSubAccounts := []*models.SubAccount{
			subAccountFactory.WithUserAndTrading(userID, tradingID),
			subAccountFactory.WithUserAndTrading(userID, tradingID),
		}
		testSubAccounts[0].Symbol = symbol
		testSubAccounts[1].Symbol = symbol
		testSubAccounts[0].Name = "binance-usdt"
		testSubAccounts[1].Name = "okx-usdt"

		// Setup mock expectations
		mockSubAccountRepo.On("GetBySymbol", mock.Anything, userID, symbol).
			Return(testSubAccounts, nil).Once()

		// Execute test
		result, err := subAccountService.GetSubAccountsBySymbol(context.Background(), userID, symbol)

		// Verify results
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, symbol, result[0].Symbol)
		assert.Equal(t, symbol, result[1].Symbol)
		assert.Equal(t, "binance-usdt", result[0].Name)
		assert.Equal(t, "okx-usdt", result[1].Name)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})

	// Test empty result
	t.Run("no_subaccounts_for_symbol", func(t *testing.T) {
		symbol := "BTC"

		// Setup mock expectations
		mockSubAccountRepo.On("GetBySymbol", mock.Anything, userID, symbol).
			Return([]*models.SubAccount{}, nil).Once()

		// Execute test
		result, err := subAccountService.GetSubAccountsBySymbol(context.Background(), userID, symbol)

		// Verify results
		require.NoError(t, err)
		assert.Len(t, result, 0)

		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
	})
}

// Performance test for sub-account operations
func TestSubAccountService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        &mocks.MockTradingRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	subAccountService := services.NewSubAccountService(repos)

	// Create test data
	userID := uuid.New()
	subAccountFactory := helpers.NewSubAccountFactory()
	testSubAccounts := []*models.SubAccount{
		subAccountFactory.WithUserID(userID),
		subAccountFactory.WithUserID(userID),
		subAccountFactory.WithUserID(userID),
	}

	t.Run("get_user_subaccounts_performance", func(t *testing.T) {
		// Setup mock for multiple calls
		mockSubAccountRepo.On("GetByUserID", mock.Anything, userID, (*uuid.UUID)(nil)).
			Return(testSubAccounts, nil).Times(100)

		timer := helpers.NewPerformanceTimer()
		timer.Start()

		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := subAccountService.GetUserSubAccounts(context.Background(), userID, nil)
			require.NoError(t, err)
		}

		timer.Stop()

		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 GetUserSubAccounts operations should complete within 1 second")

		mockSubAccountRepo.AssertExpectations(t)
	})
}
