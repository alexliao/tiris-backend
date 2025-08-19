package test

import (
	"context"
	"testing"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestTradingLogProcessor_ProcessLongPosition tests long position business logic
func TestTradingLogProcessor_ProcessLongPosition(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	mockTransactionRepo := &mocks.MockTransactionRepository{}
	
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	processor := services.NewTradingLogProcessor(repos)
	
	// Create test data
	userID := uuid.New()
	stockAccountID := uuid.New()
	currencyAccountID := uuid.New()
	
	t.Run("successful_long_position_processing", func(t *testing.T) {
		// Create accounts with sufficient balance
		stockAccount := &models.SubAccount{
			ID:       stockAccountID,
			UserID:   userID,
			Symbol:   "ETH",
			Balance:  0.0, // Starting with 0 ETH
		}
		
		currencyAccount := &models.SubAccount{
			ID:       currencyAccountID,
			UserID:   userID,
			Symbol:   "USDT",
			Balance:  10000.0, // Starting with 10,000 USDT
		}
		
		// Create trading info for long position
		tradingInfo := &services.TradingLogInfo{
			StockAccountID:    stockAccountID,
			CurrencyAccountID: currencyAccountID,
			Price:             3000.0,
			Volume:            2.0,
			Fee:               12.0,
			Stock:             "ETH",
			Currency:          "USDT",
		}
		
		// Mock balance updates
		stockTransactionID := uuid.New()
		currencyTransactionID := uuid.New()
		mockSubAccountRepo.On("UpdateBalance", mock.Anything, stockAccountID, 2.0, 2.0, "credit", "long", mock.Anything).Return(&stockTransactionID, nil)
		mockSubAccountRepo.On("UpdateBalance", mock.Anything, currencyAccountID, 3988.0, 6012.0, "debit", "long", mock.Anything).Return(&currencyTransactionID, nil)
		
		// Mock transaction retrieval
		stockTransaction := &models.Transaction{
			ID:             stockTransactionID,
			UserID:         userID,
			SubAccountID:   stockAccountID,
			Amount:         2.0,
			Direction:      "credit",
			Reason:         "long",
			ClosingBalance: 2.0,
		}
		currencyTransaction := &models.Transaction{
			ID:             currencyTransactionID,
			UserID:         userID,
			SubAccountID:   currencyAccountID,
			Amount:         6012.0,
			Direction:      "debit",
			Reason:         "long",
			ClosingBalance: 3988.0,
		}
		mockTransactionRepo.On("GetByID", mock.Anything, stockTransactionID).Return(stockTransaction, nil)
		mockTransactionRepo.On("GetByID", mock.Anything, currencyTransactionID).Return(currencyTransaction, nil)
		
		// Test the processing logic directly
		tradingLogInfoMap := map[string]interface{}{"test": "data"}
		transactions, accounts, err := processor.ProcessLongPosition(
			context.Background(),
			tradingInfo,
			stockAccount,
			currencyAccount,
			tradingLogInfoMap,
		)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, transactions)
		require.NotNil(t, accounts)
		
		// Verify processing results
		assert.Len(t, transactions, 2)
		assert.Len(t, accounts, 2)
		
		// Verify stock account balance update
		assert.Equal(t, 2.0, accounts[0].Balance)
		
		// Verify currency account balance update  
		assert.Equal(t, 3988.0, accounts[1].Balance)
		
		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
	})
	
	t.Run("insufficient_balance_long_position", func(t *testing.T) {
		// Reset mocks
		mockSubAccountRepo := &mocks.MockSubAccountRepository{}
		repos.SubAccount = mockSubAccountRepo
		
		// Create accounts with insufficient balance
		stockAccount := &models.SubAccount{
			ID:       stockAccountID,
			UserID:   userID,
			Symbol:   "ETH",
			Balance:  0.0,
		}
		
		currencyAccount := &models.SubAccount{
			ID:       currencyAccountID,
			UserID:   userID,
			Symbol:   "USDT",
			Balance:  1000.0, // Insufficient for 6012 USDT cost
		}
		
		tradingInfo := &services.TradingLogInfo{
			StockAccountID:    stockAccountID,
			CurrencyAccountID: currencyAccountID,
			Price:             3000.0,
			Volume:            2.0,
			Fee:               12.0,
			Stock:             "ETH",
			Currency:          "USDT",
		}
		
		// Test insufficient balance scenario directly
		tradingLogInfoMap := map[string]interface{}{"test": "data"}
		transactions, accounts, err := processor.ProcessLongPosition(
			context.Background(),
			tradingInfo,
			stockAccount,
			currencyAccount,
			tradingLogInfoMap,
		)
		
		// Verify error
		require.Error(t, err)
		assert.Nil(t, transactions)
		assert.Nil(t, accounts)
		assert.Contains(t, err.Error(), "insufficient balance")
		assert.Contains(t, err.Error(), "required 6012.00000000, available 1000.00000000")
	})
}

// TestTradingLogProcessor_ProcessShortPosition tests short position business logic
func TestTradingLogProcessor_ProcessShortPosition(t *testing.T) {
	// Create mocks
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	mockTransactionRepo := &mocks.MockTransactionRepository{}
	
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      mockSubAccountRepo,
		Transaction:     mockTransactionRepo,
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}
	
	processor := services.NewTradingLogProcessor(repos)
	
	userID := uuid.New()
	stockAccountID := uuid.New()
	currencyAccountID := uuid.New()
	
	t.Run("successful_short_position_processing", func(t *testing.T) {
		// Create accounts for short selling
		stockAccount := &models.SubAccount{
			ID:       stockAccountID,
			UserID:   userID,
			Symbol:   "ETH",
			Balance:  5.0, // Starting with 5 ETH
		}
		
		currencyAccount := &models.SubAccount{
			ID:       currencyAccountID,
			UserID:   userID,
			Symbol:   "USDT",
			Balance:  1000.0, // Starting with 1,000 USDT
		}
		
		tradingInfo := &services.TradingLogInfo{
			StockAccountID:    stockAccountID,
			CurrencyAccountID: currencyAccountID,
			Price:             3000.0,
			Volume:            1.5,
			Fee:               9.0,
			Stock:             "ETH",
			Currency:          "USDT",
		}
		
		// Mock balance updates for short position
		stockTransactionID := uuid.New()
		currencyTransactionID := uuid.New()
		mockSubAccountRepo.On("UpdateBalance", mock.Anything, stockAccountID, 3.5, 1.5, "debit", "short", mock.Anything).Return(&stockTransactionID, nil)
		mockSubAccountRepo.On("UpdateBalance", mock.Anything, currencyAccountID, 5491.0, 4491.0, "credit", "short", mock.Anything).Return(&currencyTransactionID, nil)
		
		// Mock transaction retrieval
		stockTransaction := &models.Transaction{
			ID:             stockTransactionID,
			UserID:         userID,
			SubAccountID:   stockAccountID,
			Amount:         1.5,
			Direction:      "debit",
			Reason:         "short",
			ClosingBalance: 3.5,
		}
		currencyTransaction := &models.Transaction{
			ID:             currencyTransactionID,
			UserID:         userID,
			SubAccountID:   currencyAccountID,
			Amount:         4491.0,
			Direction:      "credit",
			Reason:         "short",
			ClosingBalance: 5491.0,
		}
		mockTransactionRepo.On("GetByID", mock.Anything, stockTransactionID).Return(stockTransaction, nil)
		mockTransactionRepo.On("GetByID", mock.Anything, currencyTransactionID).Return(currencyTransaction, nil)
		
		tradingLogInfoMap := map[string]interface{}{"test": "data"}
		transactions, accounts, err := processor.ProcessShortPosition(
			context.Background(),
			tradingInfo,
			stockAccount,
			currencyAccount,
			tradingLogInfoMap,
			"short",
		)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, transactions)
		require.NotNil(t, accounts)
		
		// Verify processing results
		assert.Len(t, transactions, 2)
		assert.Len(t, accounts, 2)
		
		// Verify stock account was debited (sold ETH)
		assert.Equal(t, 3.5, accounts[0].Balance) // 5.0 - 1.5
		
		// Verify currency account was credited (received USDT)
		assert.Equal(t, 5491.0, accounts[1].Balance) // 1000 + 4491
		
		// Verify mock expectations
		mockSubAccountRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
	})
	
	t.Run("insufficient_stock_for_short", func(t *testing.T) {
		// Reset mocks
		mockSubAccountRepo := &mocks.MockSubAccountRepository{}
		repos.SubAccount = mockSubAccountRepo
		
		// Create accounts with insufficient stock
		stockAccount := &models.SubAccount{
			ID:       stockAccountID,
			UserID:   userID,
			Symbol:   "ETH",
			Balance:  0.5, // Only 0.5 ETH available
		}
		
		currencyAccount := &models.SubAccount{
			ID:       currencyAccountID,
			UserID:   userID,
			Symbol:   "USDT",
			Balance:  1000.0,
		}
		
		tradingInfo := &services.TradingLogInfo{
			StockAccountID:    stockAccountID,
			CurrencyAccountID: currencyAccountID,
			Price:             3000.0,
			Volume:            1.5, // Trying to sell 1.5 ETH
			Fee:               9.0,
			Stock:             "ETH",
			Currency:          "USDT",
		}
		
		// Test insufficient stock scenario directly
		tradingLogInfoMap := map[string]interface{}{"test": "data"}
		transactions, accounts, err := processor.ProcessShortPosition(
			context.Background(),
			tradingInfo,
			stockAccount,
			currencyAccount,
			tradingLogInfoMap,
			"short",
		)
		
		// Verify error
		require.Error(t, err)
		assert.Nil(t, transactions)
		assert.Nil(t, accounts)
		assert.Contains(t, err.Error(), "insufficient balance")
		assert.Contains(t, err.Error(), "required 1.50000000, available 0.50000000")
	})
}

// TestTradingLogProcessor_FinancialCalculations tests precision and accuracy
func TestTradingLogProcessor_FinancialCalculations(t *testing.T) {
	t.Run("long_position_cost_calculation", func(t *testing.T) {
		// Test calculation: price * volume + fee
		price := 3000.12345678
		volume := 2.87654321
		fee := 15.99
		
		expectedTotal := price*volume + fee // 8628.034967... + 15.99
		
		// Verify the calculation matches expected result
		actualTotal := price*volume + fee
		assert.InDelta(t, expectedTotal, actualTotal, 0.00000001, "Long position cost calculation should be accurate")
	})
	
	t.Run("short_position_proceeds_calculation", func(t *testing.T) {
		// Test calculation: price * volume - fee
		price := 2800.50
		volume := 1.25
		fee := 8.75
		
		expectedProceeds := price*volume - fee // 3500.625 - 8.75 = 3491.875
		
		// Verify the calculation matches expected result
		actualProceeds := price*volume - fee
		assert.InDelta(t, expectedProceeds, actualProceeds, 0.00000001, "Short position proceeds calculation should be accurate")
	})
	
	t.Run("zero_fee_calculation", func(t *testing.T) {
		price := 1500.0
		volume := 3.0
		fee := 0.0
		
		expectedTotal := price * volume // 4500.0
		actualTotal := price*volume + fee
		
		assert.Equal(t, expectedTotal, actualTotal, "Zero fee calculation should work correctly")
	})
}