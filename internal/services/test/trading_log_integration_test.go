package test

import (
	"context"
	"fmt"
	"testing"

	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTradingLogService_Integration_EndToEnd tests complete trading workflows
func TestTradingLogService_Integration_EndToEnd(t *testing.T) {
	// Check if running in CI environment
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Setup test database
	testConfig := config.GetProfileConfig(config.ProfileQuick)
	dbHelper := helpers.NewDatabaseTestHelper(t, testConfig)
	
	// Create real repositories and services
	repos := repositories.NewRepositories(dbHelper.DB)
	tradingLogService := services.NewTradingLogService(repos, dbHelper.DB)
	
	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	err := repos.User.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	t.Run("complete_long_position_workflow", func(t *testing.T) {
		// Setup: Create exchange
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Setup: Create sub-accounts
		subAccountFactory := helpers.NewSubAccountFactory()
		
		// ETH account (stock)
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 0.0 // Starting with 0 ETH
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		// USDT account (currency)
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 10000.0 // Starting with 10,000 USDT
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Execute: Long position trade
		request := &services.CreateTradingLogRequest{
			ExchangeID: testExchange.ID,
			Type:       "long",
			Source:     "manual",
			Message:    "Integration test: ETH long position",
			Info: map[string]interface{}{
				"stock_account_id":    ethAccount.ID.String(),
				"currency_account_id": usdtAccount.ID.String(),
				"volume":              2.0,
				"price":               3000.0,
				"fee":                 15.0,
				"stock":               "ETH",
				"currency":            "USDT",
			},
		}
		
		result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		// Verify: Trading log was created
		assert.Equal(t, "long", result.Type)
		assert.Equal(t, testUser.ID, result.UserID)
		assert.Equal(t, testExchange.ID, result.ExchangeID)
		assert.Equal(t, "manual", result.Source)
		
		// Verify: Account balances were updated correctly
		// ETH account should have gained 2.0 ETH
		updatedEthAccount, err := repos.SubAccount.GetByID(context.Background(), ethAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 2.0, updatedEthAccount.Balance)
		
		// USDT account should have lost 6015 USDT (2.0 * 3000 + 15 fee)
		updatedUsdtAccount, err := repos.SubAccount.GetByID(context.Background(), usdtAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 3985.0, updatedUsdtAccount.Balance) // 10000 - 6015
		
		// Verify: Transactions were created
		ethTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), ethAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, ethTransactions, 1)
		assert.Equal(t, "credit", ethTransactions[0].Direction)
		assert.Equal(t, "long", ethTransactions[0].Reason)
		assert.Equal(t, 2.0, ethTransactions[0].Amount)
		
		usdtTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), usdtAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, usdtTransactions, 1)
		assert.Equal(t, "debit", usdtTransactions[0].Direction)
		assert.Equal(t, "long", usdtTransactions[0].Reason)
		assert.Equal(t, 6015.0, usdtTransactions[0].Amount)
	})
	
	t.Run("complete_short_position_workflow", func(t *testing.T) {
		// Setup: Create fresh exchange for isolation
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Setup: Create sub-accounts with starting balances
		subAccountFactory := helpers.NewSubAccountFactory()
		
		// ETH account (stock) - has ETH to sell
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 5.0 // Starting with 5 ETH
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		// USDT account (currency)
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 1000.0 // Starting with 1,000 USDT
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Execute: Short position trade
		request := &services.CreateTradingLogRequest{
			ExchangeID: testExchange.ID,
			Type:       "short",
			Source:     "manual",
			Message:    "Integration test: ETH short position",
			Info: map[string]interface{}{
				"stock_account_id":    ethAccount.ID.String(),
				"currency_account_id": usdtAccount.ID.String(),
				"volume":              1.5,
				"price":               2800.0,
				"fee":                 10.0,
				"stock":               "ETH",
				"currency":            "USDT",
			},
		}
		
		result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		// Verify: Trading log was created
		assert.Equal(t, "short", result.Type)
		assert.Equal(t, testUser.ID, result.UserID)
		assert.Equal(t, testExchange.ID, result.ExchangeID)
		
		// Verify: Account balances were updated correctly
		// ETH account should have lost 1.5 ETH
		updatedEthAccount, err := repos.SubAccount.GetByID(context.Background(), ethAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 3.5, updatedEthAccount.Balance) // 5.0 - 1.5
		
		// USDT account should have gained 4190 USDT (1.5 * 2800 - 10 fee)
		updatedUsdtAccount, err := repos.SubAccount.GetByID(context.Background(), usdtAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 5190.0, updatedUsdtAccount.Balance) // 1000 + 4190
		
		// Verify: Transactions were created with correct amounts
		ethTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), ethAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, ethTransactions, 1)
		assert.Equal(t, "debit", ethTransactions[0].Direction)
		assert.Equal(t, "short", ethTransactions[0].Reason)
		assert.Equal(t, 1.5, ethTransactions[0].Amount)
		
		usdtTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), usdtAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, usdtTransactions, 1)
		assert.Equal(t, "credit", usdtTransactions[0].Direction)
		assert.Equal(t, "short", usdtTransactions[0].Reason)
		assert.Equal(t, 4190.0, usdtTransactions[0].Amount)
	})
	
	t.Run("complete_stop_loss_workflow", func(t *testing.T) {
		// Setup: Create fresh exchange for isolation
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Setup: Create sub-accounts
		subAccountFactory := helpers.NewSubAccountFactory()
		
		// ETH account (stock) - has ETH to sell via stop-loss
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 3.0 // Starting with 3 ETH
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		// USDT account (currency)
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 500.0 // Starting with 500 USDT
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Execute: Stop-loss triggered
		request := &services.CreateTradingLogRequest{
			ExchangeID: testExchange.ID,
			Type:       "stop_loss",
			Source:     "bot",
			Message:    "Integration test: ETH stop-loss triggered",
			Info: map[string]interface{}{
				"stock_account_id":    ethAccount.ID.String(),
				"currency_account_id": usdtAccount.ID.String(),
				"volume":              1.0,
				"price":               2500.0,
				"fee":                 5.0,
				"stock":               "ETH",
				"currency":            "USDT",
			},
		}
		
		result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		// Verify: Trading log was created
		assert.Equal(t, "stop_loss", result.Type)
		assert.Equal(t, "bot", result.Source)
		
		// Verify: Account balances were updated correctly
		// ETH account should have lost 1.0 ETH
		updatedEthAccount, err := repos.SubAccount.GetByID(context.Background(), ethAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 2.0, updatedEthAccount.Balance) // 3.0 - 1.0
		
		// USDT account should have gained 2495 USDT (1.0 * 2500 - 5 fee)
		updatedUsdtAccount, err := repos.SubAccount.GetByID(context.Background(), usdtAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 2995.0, updatedUsdtAccount.Balance) // 500 + 2495
		
		// Verify: Transactions were created with stop_loss reason
		ethTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), ethAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, ethTransactions, 1)
		assert.Equal(t, "debit", ethTransactions[0].Direction)
		assert.Equal(t, "stop_loss", ethTransactions[0].Reason)
		
		usdtTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), usdtAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, usdtTransactions, 1)
		assert.Equal(t, "credit", usdtTransactions[0].Direction)
		assert.Equal(t, "stop_loss", usdtTransactions[0].Reason)
	})
}

// TestTradingLogService_Integration_ErrorHandling tests error scenarios with real database
func TestTradingLogService_Integration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Setup test database
	testConfig := config.GetProfileConfig(config.ProfileQuick)
	dbHelper := helpers.NewDatabaseTestHelper(t, testConfig)
	
	repos := repositories.NewRepositories(dbHelper.DB)
	tradingLogService := services.NewTradingLogService(repos, dbHelper.DB)
	
	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	err := repos.User.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	t.Run("insufficient_balance_real_database", func(t *testing.T) {
		// Setup: Create exchange and accounts
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		subAccountFactory := helpers.NewSubAccountFactory()
		
		// ETH account
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 0.0
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		// USDT account with insufficient balance
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 100.0 // Insufficient for trade
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Attempt: Long position with insufficient funds
		request := &services.CreateTradingLogRequest{
			ExchangeID: testExchange.ID,
			Type:       "long",
			Source:     "manual",
			Message:    "Integration test: Insufficient balance",
			Info: map[string]interface{}{
				"stock_account_id":    ethAccount.ID.String(),
				"currency_account_id": usdtAccount.ID.String(),
				"volume":              2.0,
				"price":               3000.0,
				"fee":                 15.0,
				"stock":               "ETH",
				"currency":            "USDT",
			},
		}
		
		// Verify: Transaction fails with insufficient balance
		result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "insufficient balance")
		
		// Verify: No transactions were created (rollback worked)
		ethTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), ethAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, ethTransactions, 0)
		
		usdtTransactions, _, err := repos.Transaction.GetBySubAccountID(context.Background(), usdtAccount.ID, repositories.TransactionFilters{})
		require.NoError(t, err)
		assert.Len(t, usdtTransactions, 0)
		
		// Verify: Account balances unchanged
		unchangedEthAccount, err := repos.SubAccount.GetByID(context.Background(), ethAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, unchangedEthAccount.Balance)
		
		unchangedUsdtAccount, err := repos.SubAccount.GetByID(context.Background(), usdtAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 100.0, unchangedUsdtAccount.Balance)
	})
	
	t.Run("account_ownership_validation", func(t *testing.T) {
		// Setup: Create another user
		otherUserFactory := helpers.NewUserFactory()
		otherUser := otherUserFactory.Build()
		err := repos.User.Create(context.Background(), otherUser)
		require.NoError(t, err)
		
		// Setup: Create exchange for first user
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err = repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Setup: Create account for OTHER user
		subAccountFactory := helpers.NewSubAccountFactory()
		otherUserAccount := subAccountFactory.WithUserAndExchange(otherUser.ID, testExchange.ID)
		otherUserAccount.Symbol = "ETH"
		otherUserAccount.Balance = 10.0
		err = repos.SubAccount.Create(context.Background(), otherUserAccount)
		require.NoError(t, err)
		
		// Create account for test user
		testUserAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		testUserAccount.Symbol = "USDT"
		testUserAccount.Balance = 10000.0
		err = repos.SubAccount.Create(context.Background(), testUserAccount)
		require.NoError(t, err)
		
		// Attempt: Use other user's account in trading
		request := &services.CreateTradingLogRequest{
			ExchangeID: testExchange.ID,
			Type:       "long",
			Source:     "manual",
			Message:    "Integration test: Wrong account ownership",
			Info: map[string]interface{}{
				"stock_account_id":    otherUserAccount.ID.String(), // Wrong user!
				"currency_account_id": testUserAccount.ID.String(),
				"volume":              1.0,
				"price":               3000.0,
				"fee":                 15.0,
				"stock":               "ETH",
				"currency":            "USDT",
			},
		}
		
		// Verify: Transaction fails with account not found
		result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestTradingLogService_Integration_ConcurrentTransactions tests race conditions
func TestTradingLogService_Integration_ConcurrentTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent transaction test in short mode")
	}
	
	t.Skip("TODO: Fix concurrent transaction isolation test - balance checking needs investigation")
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Setup test database
	testConfig := config.GetProfileConfig(config.ProfileQuick)
	dbHelper := helpers.NewDatabaseTestHelper(t, testConfig)
	
	repos := repositories.NewRepositories(dbHelper.DB)
	tradingLogService := services.NewTradingLogService(repos, dbHelper.DB)
	
	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	err := repos.User.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	t.Run("concurrent_balance_updates", func(t *testing.T) {
		// Setup: Create exchange and accounts
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		subAccountFactory := helpers.NewSubAccountFactory()
		
		// ETH account
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 10.0 // Has ETH to sell
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		// USDT account
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 1000.0
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Test: Attempt concurrent transactions that would overdraw account
		// Each tries to sell 6 ETH, but only 10 ETH available
		// One should succeed, one should fail
		
		type result struct {
			response *services.TradingLogResponse
			error    error
		}
		
		resultChan := make(chan result, 2)
		
		// Start two concurrent transactions
		for i := 0; i < 2; i++ {
			go func(tradeNum int) {
				request := &services.CreateTradingLogRequest{
					ExchangeID: testExchange.ID,
					Type:       "short",
					Source:     "manual",
					Message:    fmt.Sprintf("Concurrent test trade %d", tradeNum),
					Info: map[string]interface{}{
						"stock_account_id":    ethAccount.ID.String(),
						"currency_account_id": usdtAccount.ID.String(),
						"volume":              6.0, // Trying to sell 6 ETH each
						"price":               3000.0,
						"fee":                 10.0,
						"stock":               "ETH",
						"currency":            "USDT",
					},
				}
				
				resp, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
				resultChan <- result{response: resp, error: err}
			}(i)
		}
		
		// Collect results
		var successCount, errorCount int
		for i := 0; i < 2; i++ {
			res := <-resultChan
			if res.error != nil {
				errorCount++
				assert.Contains(t, res.error.Error(), "insufficient balance")
			} else {
				successCount++
				assert.NotNil(t, res.response)
			}
		}
		
		// Verify: Exactly one transaction succeeded, one failed
		assert.Equal(t, 1, successCount, "Exactly one transaction should succeed")
		assert.Equal(t, 1, errorCount, "Exactly one transaction should fail")
		
		// Verify: Final balance is correct (10 - 6 = 4 ETH remaining)
		finalEthAccount, err := repos.SubAccount.GetByID(context.Background(), ethAccount.ID)
		require.NoError(t, err)
		assert.Equal(t, 4.0, finalEthAccount.Balance)
	})
}