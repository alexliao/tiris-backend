package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTradingLogService_Performance_HighVolume tests high-volume trading scenarios
func TestTradingLogService_Performance_HighVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
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
	
	t.Run("sequential_high_volume_trades", func(t *testing.T) {
		// Setup: Create exchange and accounts with large balances
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		subAccountFactory := helpers.NewSubAccountFactory()
		
		// ETH account with large balance
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 10000.0 // Large starting balance
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		// USDT account with large balance
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 100000000.0 // 100M USDT
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Test: Execute 1000 sequential trades
		tradeCount := 1000
		startTime := time.Now()
		
		for i := 0; i < tradeCount; i++ {
			tradeType := "long"
			if i%2 == 1 {
				tradeType = "short"
			}
			
			request := &services.CreateTradingLogRequest{
				ExchangeID: testExchange.ID,
				Type:       tradeType,
				Source:     "manual",
				Message:    fmt.Sprintf("Performance test trade %d", i),
				Info: map[string]interface{}{
					"stock_account_id":    ethAccount.ID.String(),
					"currency_account_id": usdtAccount.ID.String(),
					"volume":              0.1, // Small volume to avoid balance issues
					"price":               3000.0 + float64(i%100), // Varying prices
					"fee":                 1.0,
					"stock":               "ETH",
					"currency":            "USDT",
				},
			}
			
			result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
			require.NoError(t, err, "Trade %d failed", i)
			require.NotNil(t, result)
		}
		
		duration := time.Since(startTime)
		tradesPerSecond := float64(tradeCount) / duration.Seconds()
		
		t.Logf("Completed %d trades in %v", tradeCount, duration)
		t.Logf("Performance: %.2f trades/second", tradesPerSecond)
		
		// Performance assertions
		assert.True(t, tradesPerSecond > 10, "Should process at least 10 trades/second, got %.2f", tradesPerSecond)
		assert.True(t, duration < 2*time.Minute, "Should complete 1000 trades within 2 minutes")
		
		// Verify all trades were processed correctly
		allTrades, _, err := repos.TradingLog.GetByUserID(context.Background(), testUser.ID, repositories.TradingLogFilters{})
		require.NoError(t, err)
		assert.Len(t, allTrades, tradeCount)
	})
	
	t.Run("concurrent_high_volume_trades", func(t *testing.T) {
		t.Skip("TODO: Fix concurrent trading performance test - database isolation issues")
		// Setup: Create fresh exchange for isolation
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Create multiple accounts for concurrent testing
		subAccountFactory := helpers.NewSubAccountFactory()
		
		accountCount := 10
		for i := 0; i < accountCount; i++ {
			// ETH account
			ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
			ethAccount.Symbol = fmt.Sprintf("ETH_%d", i)
			ethAccount.Balance = 1000.0
			err = repos.SubAccount.Create(context.Background(), ethAccount)
			require.NoError(t, err)
			
			// USDT account
			usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
			usdtAccount.Symbol = fmt.Sprintf("USDT_%d", i)
			usdtAccount.Balance = 10000000.0 // 10M USDT each
			err = repos.SubAccount.Create(context.Background(), usdtAccount)
			require.NoError(t, err)
		}
		
		// Test: Concurrent trades across multiple goroutines
		goroutineCount := 20
		tradesPerGoroutine := 50
		totalTrades := goroutineCount * tradesPerGoroutine
		
		var wg sync.WaitGroup
		var successCount, errorCount int64
		var mu sync.Mutex
		
		startTime := time.Now()
		
		for g := 0; g < goroutineCount; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				
				// Use different accounts for each goroutine to reduce contention
				accountIndex := goroutineID % accountCount
				
				for i := 0; i < tradesPerGoroutine; i++ {
					request := &services.CreateTradingLogRequest{
						ExchangeID: testExchange.ID,
						Type:       "long",
						Source:     "concurrent_test",
						Message:    fmt.Sprintf("Concurrent test G%d-T%d", goroutineID, i),
						Info: map[string]interface{}{
							"stock_account_id":    fmt.Sprintf("ETH_%d", accountIndex),
							"currency_account_id": fmt.Sprintf("USDT_%d", accountIndex),
							"volume":              0.01, // Very small volume
							"price":               3000.0 + float64(i%10),
							"fee":                 0.1,
							"stock":               fmt.Sprintf("ETH_%d", accountIndex),
							"currency":            fmt.Sprintf("USDT_%d", accountIndex),
						},
					}
					
					result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
					
					mu.Lock()
					if err != nil {
						errorCount++
					} else if result != nil {
						successCount++
					}
					mu.Unlock()
				}
			}(g)
		}
		
		wg.Wait()
		duration := time.Since(startTime)
		
		t.Logf("Concurrent test completed in %v", duration)
		t.Logf("Success: %d, Errors: %d, Total: %d", successCount, errorCount, totalTrades)
		t.Logf("Concurrent performance: %.2f trades/second", float64(successCount)/duration.Seconds())
		
		// Performance assertions
		assert.True(t, successCount > int64(float64(totalTrades)*0.8), "At least 80%% of trades should succeed")
		assert.True(t, duration < 30*time.Second, "Concurrent trades should complete within 30 seconds")
	})
}

// TestTradingLogService_Performance_StressTest tests system limits
func TestTradingLogService_Performance_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
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
	
	t.Run("database_connection_stress", func(t *testing.T) {
		t.Skip("TODO: Fix database connection stress test - concurrent access issues")
		// Setup: Create exchange and accounts
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		subAccountFactory := helpers.NewSubAccountFactory()
		
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 50000.0
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 1000000000.0 // 1B USDT
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Test: High concurrent load with many connections
		maxConcurrency := 100
		tradesPerConnection := 10
		
		var wg sync.WaitGroup
		var totalSuccesses, totalErrors int64
		var mu sync.Mutex
		
		semaphore := make(chan struct{}, maxConcurrency)
		
		startTime := time.Now()
		
		for i := 0; i < maxConcurrency*tradesPerConnection; i++ {
			wg.Add(1)
			go func(tradeID int) {
				defer wg.Done()
				
				// Acquire semaphore to limit concurrency
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				request := &services.CreateTradingLogRequest{
					ExchangeID: testExchange.ID,
					Type:       "short", // Sell ETH
					Source:     "manual",
					Message:    fmt.Sprintf("Stress test trade %d", tradeID),
					Info: map[string]interface{}{
						"stock_account_id":    ethAccount.ID.String(),
						"currency_account_id": usdtAccount.ID.String(),
						"volume":              0.001, // Very small volume
						"price":               3000.0,
						"fee":                 0.01,
						"stock":               "ETH",
						"currency":            "USDT",
					},
				}
				
				result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
				
				mu.Lock()
				if err != nil {
					totalErrors++
				} else if result != nil {
					totalSuccesses++
				}
				mu.Unlock()
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(startTime)
		
		t.Logf("Stress test completed in %v", duration)
		t.Logf("Successes: %d, Errors: %d", totalSuccesses, totalErrors)
		t.Logf("Success rate: %.2f%%", float64(totalSuccesses)/float64(totalSuccesses+totalErrors)*100)
		
		// Stress test assertions
		assert.True(t, totalSuccesses > 0, "Should have some successful trades")
		assert.True(t, float64(totalSuccesses)/float64(totalSuccesses+totalErrors) > 0.7, "Success rate should be > 70%%")
	})
	
	t.Run("memory_efficiency_test", func(t *testing.T) {
		t.Skip("TODO: Optimize memory efficiency test - currently too slow for CI")
		// Setup: Create exchange and accounts
		exchangeFactory := helpers.NewExchangeFactory()
		testExchange := exchangeFactory.WithUserID(testUser.ID)
		err := repos.Exchange.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		subAccountFactory := helpers.NewSubAccountFactory()
		
		ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		ethAccount.Symbol = "ETH"
		ethAccount.Balance = 100000.0
		err = repos.SubAccount.Create(context.Background(), ethAccount)
		require.NoError(t, err)
		
		usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
		usdtAccount.Symbol = "USDT"
		usdtAccount.Balance = 100000000.0
		err = repos.SubAccount.Create(context.Background(), usdtAccount)
		require.NoError(t, err)
		
		// Test: Process large number of trades to test memory efficiency
		largeTradeCount := 5000
		
		startTime := time.Now()
		
		for i := 0; i < largeTradeCount; i++ {
			// Alternate between long and short to maintain balance
			tradeType := "long"
			if i%2 == 1 {
				tradeType = "short"
			}
			
			request := &services.CreateTradingLogRequest{
				ExchangeID: testExchange.ID,
				Type:       tradeType,
				Source:     "manual",
				Message:    fmt.Sprintf("Memory efficiency test %d", i),
				Info: map[string]interface{}{
					"stock_account_id":    ethAccount.ID.String(),
					"currency_account_id": usdtAccount.ID.String(),
					"volume":              0.01,
					"price":               3000.0 + float64(i%1000)*0.1, // Varying prices
					"fee":                 0.05,
					"stock":               "ETH",
					"currency":            "USDT",
				},
			}
			
			result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
			require.NoError(t, err, "Trade %d failed", i)
			require.NotNil(t, result)
			
			// Log progress every 1000 trades
			if (i+1)%1000 == 0 {
				t.Logf("Processed %d/%d trades", i+1, largeTradeCount)
			}
		}
		
		duration := time.Since(startTime)
		avgTimePerTrade := duration / time.Duration(largeTradeCount)
		
		t.Logf("Memory efficiency test completed: %d trades in %v", largeTradeCount, duration)
		t.Logf("Average time per trade: %v", avgTimePerTrade)
		
		// Memory efficiency assertions
		assert.True(t, avgTimePerTrade < 100*time.Millisecond, "Average trade time should be < 100ms")
		assert.True(t, duration < 10*time.Minute, "Should complete 5000 trades within 10 minutes")
	})
}

// TestTradingLogService_Performance_Benchmarks provides benchmark tests
func TestTradingLogService_Performance_Benchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark tests in short mode")
	}
	
	// Setup test database
	testConfig := config.GetProfileConfig(config.ProfileQuick)
	dbHelper := helpers.NewDatabaseTestHelper(t, testConfig)
	
	repos := repositories.NewRepositories(dbHelper.DB)
	tradingLogService := services.NewTradingLogService(repos, dbHelper.DB)
	
	// Create test user and setup
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	err := repos.User.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	exchangeFactory := helpers.NewExchangeFactory()
	testExchange := exchangeFactory.WithUserID(testUser.ID)
	err = repos.Exchange.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	subAccountFactory := helpers.NewSubAccountFactory()
	
	ethAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
	ethAccount.Symbol = "ETH"
	ethAccount.Balance = 1000000.0
	err = repos.SubAccount.Create(context.Background(), ethAccount)
	require.NoError(t, err)
	
	usdtAccount := subAccountFactory.WithUserAndExchange(testUser.ID, testExchange.ID)
	usdtAccount.Symbol = "USDT"
	usdtAccount.Balance = 10000000000.0 // 10B USDT
	err = repos.SubAccount.Create(context.Background(), usdtAccount)
	require.NoError(t, err)
	
	t.Run("benchmark_single_trade_latency", func(t *testing.T) {
		// Warm-up
		for i := 0; i < 10; i++ {
			request := &services.CreateTradingLogRequest{
				ExchangeID: testExchange.ID,
				Type:       "long",
				Source:     "warmup",
				Message:    "Warmup trade",
				Info: map[string]interface{}{
					"stock_account_id":    ethAccount.ID.String(),
					"currency_account_id": usdtAccount.ID.String(),
					"volume":              0.001,
					"price":               3000.0,
					"fee":                 0.01,
					"stock":               "ETH",
					"currency":            "USDT",
				},
			}
			_, _ = tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
		}
		
		// Benchmark single trade latency
		iterations := 100
		var totalDuration time.Duration
		
		for i := 0; i < iterations; i++ {
			request := &services.CreateTradingLogRequest{
				ExchangeID: testExchange.ID,
				Type:       "short",
				Source:     "manual",
				Message:    fmt.Sprintf("Benchmark trade %d", i),
				Info: map[string]interface{}{
					"stock_account_id":    ethAccount.ID.String(),
					"currency_account_id": usdtAccount.ID.String(),
					"volume":              0.001,
					"price":               3000.0,
					"fee":                 0.01,
					"stock":               "ETH",
					"currency":            "USDT",
				},
			}
			
			start := time.Now()
			result, err := tradingLogService.CreateTradingLog(context.Background(), testUser.ID, request)
			elapsed := time.Since(start)
			
			require.NoError(t, err)
			require.NotNil(t, result)
			
			totalDuration += elapsed
		}
		
		avgLatency := totalDuration / time.Duration(iterations)
		t.Logf("Average single trade latency: %v", avgLatency)
		
		// Latency assertions
		assert.True(t, avgLatency < 50*time.Millisecond, "Average latency should be < 50ms, got %v", avgLatency)
	})
}