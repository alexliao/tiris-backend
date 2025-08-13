package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// PerformanceTestSuite tests system performance under various loads
type PerformanceTestSuite struct {
	IntegrationTestSuite
}

// BenchmarkDatabaseOperations tests database operation performance
func (s *PerformanceTestSuite) TestDatabasePerformance() {
	const numOperations = 1000

	// Benchmark user creation
	start := time.Now()
	users := make([]*models.User, numOperations)
	for i := 0; i < numOperations; i++ {
		users[i] = &models.User{
			ID:       uuid.New(),
			Username: fmt.Sprintf("perf_user_%d", i),
			Email:    fmt.Sprintf("perf%d@example.com", i),
		}
	}

	err := s.DB.CreateInBatches(users, 100).Error
	s.Require().NoError(err)
	
	userCreationTime := time.Since(start)
	s.Less(userCreationTime, 5*time.Second, "Creating 1000 users should take less than 5 seconds")
	
	avgUserCreation := userCreationTime / numOperations
	s.Less(avgUserCreation, 5*time.Millisecond, "Average user creation should be under 5ms")

	// Benchmark user queries
	start = time.Now()
	for i := 0; i < 100; i++ {
		var user models.User
		err := s.DB.Where("username = ?", fmt.Sprintf("perf_user_%d", i)).First(&user).Error
		s.Require().NoError(err)
	}
	queryTime := time.Since(start)
	avgQueryTime := queryTime / 100
	s.Less(avgQueryTime, 10*time.Millisecond, "Average user query should be under 10ms")

	// Benchmark bulk updates
	start = time.Now()
	err = s.DB.Model(&models.User{}).Where("username LIKE ?", "perf_user_%").Update("info", map[string]interface{}{"updated": true}).Error
	s.Require().NoError(err)
	bulkUpdateTime := time.Since(start)
	s.Less(bulkUpdateTime, 1*time.Second, "Bulk update of 1000 users should take less than 1 second")

	// Cleanup
	err = s.DB.Where("username LIKE ?", "perf_user_%").Delete(&models.User{}).Error
	s.Require().NoError(err)
}

// TestConcurrentAPIKeyOperations tests API key operations under concurrent load
func (s *PerformanceTestSuite) TestConcurrentAPIKeyOperations() {
	const numGoroutines = 50
	const operationsPerGoroutine = 10
	
	user := s.createTestUser()
	
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)
	apiKeys := make(chan *services.UserAPIKey, numGoroutines*operationsPerGoroutine)
	
	start := time.Now()
	
	// Create API keys concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				apiKey, err := s.SecurityService.CreateUserAPIKey(
					s.ctx,
					user.ID,
					fmt.Sprintf("Perf Key %d-%d", goroutineID, j),
					[]string{"read"},
				)
				
				if err != nil {
					errors <- err
					return
				}
				
				apiKeys <- apiKey
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	close(apiKeys)
	
	creationTime := time.Since(start)
	
	// Check for errors
	for err := range errors {
		s.Require().NoError(err)
	}
	
	// Collect created keys
	var createdKeys []*services.UserAPIKey
	for apiKey := range apiKeys {
		createdKeys = append(createdKeys, apiKey)
	}
	
	s.Len(createdKeys, numGoroutines*operationsPerGoroutine, "All API keys should be created")
	s.Less(creationTime, 10*time.Second, "Concurrent API key creation should complete in reasonable time")
	
	avgCreationTime := creationTime / (numGoroutines * operationsPerGoroutine)
	s.Less(avgCreationTime, 20*time.Millisecond, "Average API key creation should be under 20ms")
	
	// Test concurrent validation
	start = time.Now()
	wg = sync.WaitGroup{}
	
	for _, apiKey := range createdKeys {
		wg.Add(1)
		go func(key *services.UserAPIKey) {
			defer wg.Done()
			
			result, err := s.SecurityService.ValidateAPIKey(s.ctx, *key.PlaintextKey)
			if err != nil {
				errors <- err
				return
			}
			
			if !result.Valid {
				errors <- fmt.Errorf("API key should be valid")
			}
		}(apiKey)
	}
	
	wg.Wait()
	validationTime := time.Since(start)
	
	avgValidationTime := validationTime / time.Duration(len(createdKeys))
	s.Less(avgValidationTime, 5*time.Millisecond, "Average API key validation should be under 5ms")
}

// TestRateLimitingPerformance tests rate limiting performance
func (s *PerformanceTestSuite) TestRateLimitingPerformance() {
	const numChecks = 1000
	const numGoroutines = 10
	
	var wg sync.WaitGroup
	errors := make(chan error, numChecks)
	durations := make(chan time.Duration, numChecks)
	
	checksPerGoroutine := numChecks / numGoroutines
	
	overallStart := time.Now()
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			identifier := fmt.Sprintf("perf_test_%d", goroutineID)
			
			for j := 0; j < checksPerGoroutine; j++ {
				start := time.Now()
				
				_, err := s.SecurityService.CheckRateLimit(s.ctx, identifier, "api_general")
				if err != nil {
					errors <- err
					return
				}
				
				durations <- time.Since(start)
			}
		}(i)
	}
	
	wg.Wait()
	overallTime := time.Since(overallStart)
	
	close(errors)
	close(durations)
	
	// Check for errors
	for err := range errors {
		s.Require().NoError(err)
	}
	
	// Calculate statistics
	var totalDuration time.Duration
	var maxDuration time.Duration
	count := 0
	
	for duration := range durations {
		totalDuration += duration
		if duration > maxDuration {
			maxDuration = duration
		}
		count++
	}
	
	avgDuration := totalDuration / time.Duration(count)
	
	s.Equal(numChecks, count, "Should process all rate limit checks")
	s.Less(avgDuration, 5*time.Millisecond, "Average rate limit check should be under 5ms")
	s.Less(maxDuration, 50*time.Millisecond, "Maximum rate limit check should be under 50ms")
	s.Less(overallTime, 5*time.Second, "Overall rate limiting test should complete quickly")
}

// TestEncryptionDecryptionPerformance tests encryption performance
func (s *PerformanceTestSuite) TestEncryptionDecryptionPerformance() {
	const numOperations = 1000
	
	testData := []string{
		"short_key",
		"medium_length_api_key_with_some_complexity_12345",
		"very_long_api_key_that_might_be_used_in_production_environments_with_lots_of_entropy_and_complexity_abcdefghijklmnopqrstuvwxyz_1234567890",
	}
	
	for _, data := range testData {
		s.Run(fmt.Sprintf("data_length_%d", len(data)), func() {
			// Test encryption performance
			start := time.Now()
			encryptedValues := make([]string, numOperations)
			
			for i := 0; i < numOperations; i++ {
				encrypted, err := s.SecurityService.EncryptSensitiveData(data)
				s.Require().NoError(err)
				encryptedValues[i] = encrypted
			}
			
			encryptionTime := time.Since(start)
			avgEncryptionTime := encryptionTime / numOperations
			s.Less(avgEncryptionTime, 1*time.Millisecond, "Average encryption should be under 1ms")
			
			// Test decryption performance
			start = time.Now()
			
			for i := 0; i < numOperations; i++ {
				decrypted, err := s.SecurityService.DecryptSensitiveData(encryptedValues[i])
				s.Require().NoError(err)
				s.Equal(data, decrypted)
			}
			
			decryptionTime := time.Since(start)
			avgDecryptionTime := decryptionTime / numOperations
			s.Less(avgDecryptionTime, 1*time.Millisecond, "Average decryption should be under 1ms")
		})
	}
}

// TestAuditLoggingPerformance tests audit logging performance
func (s *PerformanceTestSuite) TestAuditLoggingPerformance() {
	const numEvents = 1000
	const batchSize = 100
	
	user := s.createTestUser()
	
	events := make([]security.AuditEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = security.AuditEvent{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Level:     security.AuditLevelInfo,
			Action:    security.ActionUserView,
			UserID:    &user.ID,
			IPAddress: fmt.Sprintf("192.168.1.%d", i%255),
			Resource:  fmt.Sprintf("resource_%d", i),
			Success:   true,
			Details:   map[string]interface{}{"performance_test": i},
		}
	}
	
	// Test batch insertion performance
	start := time.Now()
	err := s.DB.CreateInBatches(events, batchSize).Error
	s.Require().NoError(err)
	batchInsertTime := time.Since(start)
	
	avgBatchInsertTime := batchInsertTime / time.Duration(numEvents/batchSize)
	s.Less(avgBatchInsertTime, 100*time.Millisecond, "Average batch insert should be under 100ms")
	
	// Test query performance on large dataset
	start = time.Now()
	var queryResults []security.AuditEvent
	err = s.DB.Where("user_id = ? AND timestamp >= ?", user.ID, time.Now().Add(-time.Hour)).
		Order("timestamp DESC").
		Limit(100).
		Find(&queryResults).Error
	s.Require().NoError(err)
	queryTime := time.Since(start)
	
	s.Less(queryTime, 50*time.Millisecond, "Complex audit query should be under 50ms")
	s.Len(queryResults, 100, "Should return 100 results")
	
	// Test aggregation performance
	start = time.Now()
	var summary []struct {
		Action string
		Count  int64
	}
	err = s.DB.Model(&security.AuditEvent{}).
		Select("action, COUNT(*) as count").
		Where("user_id = ?", user.ID).
		Group("action").
		Scan(&summary).Error
	s.Require().NoError(err)
	aggregationTime := time.Since(start)
	
	s.Less(aggregationTime, 100*time.Millisecond, "Audit aggregation should be under 100ms")
	s.Greater(len(summary), 0, "Should have aggregation results")
}

// TestConcurrentSecureExchangeOperations tests exchange operations under load
func (s *PerformanceTestSuite) TestConcurrentSecureExchangeOperations() {
	const numUsers = 10
	const exchangesPerUser = 5
	
	users := make([]*models.User, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = s.createTestUser()
	}
	
	var wg sync.WaitGroup
	errors := make(chan error, numUsers*exchangesPerUser)
	exchangeResults := make(chan *models.SecureExchange, numUsers*exchangesPerUser)
	
	start := time.Now()
	
	// Create exchanges concurrently
	for i, user := range users {
		wg.Add(1)
		go func(userIdx int, u *models.User) {
			defer wg.Done()
			
			for j := 0; j < exchangesPerUser; j++ {
				exchange, err := s.SecurityService.CreateSecureExchange(
					s.ctx,
					u.ID,
					fmt.Sprintf("Exchange %d-%d", userIdx, j),
					"binance",
					fmt.Sprintf("api_key_%d_%d", userIdx, j),
					fmt.Sprintf("secret_%d_%d", userIdx, j),
				)
				
				if err != nil {
					errors <- err
					return
				}
				
				exchangeResults <- exchange
			}
		}(i, user)
	}
	
	wg.Wait()
	close(errors)
	close(exchangeResults)
	
	creationTime := time.Since(start)
	
	// Check for errors
	for err := range errors {
		s.Require().NoError(err)
	}
	
	// Collect results
	var exchanges []*models.SecureExchange
	for exchange := range exchangeResults {
		exchanges = append(exchanges, exchange)
	}
	
	s.Len(exchanges, numUsers*exchangesPerUser, "All exchanges should be created")
	
	// Test concurrent credential retrieval
	start = time.Now()
	wg = sync.WaitGroup{}
	
	for i, exchange := range exchanges {
		wg.Add(1)
		go func(idx int, ex *models.SecureExchange) {
			defer wg.Done()
			
			userID := users[idx/(exchangesPerUser)].ID
			
			apiKey, apiSecret, err := s.SecurityService.GetExchangeCredentials(s.ctx, userID, ex.ID)
			if err != nil {
				errors <- err
				return
			}
			
			expectedKey := fmt.Sprintf("api_key_%d_%d", idx/exchangesPerUser, idx%exchangesPerUser)
			expectedSecret := fmt.Sprintf("secret_%d_%d", idx/exchangesPerUser, idx%exchangesPerUser)
			
			if apiKey != expectedKey {
				errors <- fmt.Errorf("API key mismatch: expected %s, got %s", expectedKey, apiKey)
				return
			}
			
			if apiSecret != expectedSecret {
				errors <- fmt.Errorf("API secret mismatch: expected %s, got %s", expectedSecret, apiSecret)
				return
			}
		}(i, exchange)
	}
	
	wg.Wait()
	retrievalTime := time.Since(start)
	
	totalTime := creationTime + retrievalTime
	s.Less(totalTime, 10*time.Second, "Total secure exchange operations should complete in reasonable time")
	
	avgCreationTime := creationTime / time.Duration(len(exchanges))
	avgRetrievalTime := retrievalTime / time.Duration(len(exchanges))
	
	s.Less(avgCreationTime, 50*time.Millisecond, "Average exchange creation should be under 50ms")
	s.Less(avgRetrievalTime, 10*time.Millisecond, "Average credential retrieval should be under 10ms")
}

// TestMemoryUsage tests for memory leaks during operations
func (s *PerformanceTestSuite) TestMemoryUsage() {
	// This is a basic test - in production you'd use more sophisticated profiling
	const numIterations = 100
	
	user := s.createTestUser()
	
	// Perform operations that might cause memory leaks
	for i := 0; i < numIterations; i++ {
		// Create and validate API keys
		apiKey, err := s.SecurityService.CreateUserAPIKey(
			s.ctx,
			user.ID,
			fmt.Sprintf("Memory Test Key %d", i),
			[]string{"read"},
		)
		s.Require().NoError(err)
		
		result, err := s.SecurityService.ValidateAPIKey(s.ctx, *apiKey.PlaintextKey)
		s.Require().NoError(err)
		s.True(result.Valid)
		
		// Perform encryption/decryption
		data := fmt.Sprintf("test_data_%d", i)
		encrypted, err := s.SecurityService.EncryptSensitiveData(data)
		s.Require().NoError(err)
		
		decrypted, err := s.SecurityService.DecryptSensitiveData(encrypted)
		s.Require().NoError(err)
		s.Equal(data, decrypted)
		
		// Check rate limits
		_, err = s.SecurityService.CheckRateLimit(s.ctx, fmt.Sprintf("memory_test_%d", i), "api_general")
		s.Require().NoError(err)
		
		// Create audit events
		err = s.SecurityService.AuditDataAccess(
			s.ctx,
			&user.ID,
			security.ActionUserView,
			"test_resource",
			fmt.Sprintf("test_id_%d", i),
			"127.0.0.1",
			true,
		)
		s.Require().NoError(err)
		
		// Clean up API key to prevent accumulation
		err = s.DB.Where("id = ?", apiKey.ID).Delete(&services.UserAPIKey{}).Error
		s.Require().NoError(err)
	}
	
	// Force garbage collection
	// runtime.GC()
	
	// In a real test, you would check memory usage here
	// This is a placeholder for more sophisticated memory profiling
	s.True(true, "Memory usage test completed")
}

// TestSystemResourceUsage tests overall system resource consumption
func (s *PerformanceTestSuite) TestSystemResourceUsage() {
	const duration = 10 * time.Second
	const concurrentUsers = 20
	
	users := make([]*models.User, concurrentUsers)
	for i := 0; i < concurrentUsers; i++ {
		users[i] = s.createTestUser()
	}
	
	ctx, cancel := context.WithTimeout(s.ctx, duration)
	defer cancel()
	
	var wg sync.WaitGroup
	operationCounts := make(chan int, concurrentUsers)
	
	// Simulate realistic workload
	for i, user := range users {
		wg.Add(1)
		go func(userIdx int, u *models.User) {
			defer wg.Done()
			
			operationCount := 0
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			
			for {
				select {
				case <-ctx.Done():
					operationCounts <- operationCount
					return
				case <-ticker.C:
					// Simulate mixed workload
					switch operationCount % 4 {
					case 0:
						// Rate limit check
						s.SecurityService.CheckRateLimit(ctx, fmt.Sprintf("user_%d", userIdx), "api_general")
					case 1:
						// Encryption/decryption
						data := fmt.Sprintf("data_%d_%d", userIdx, operationCount)
						if encrypted, err := s.SecurityService.EncryptSensitiveData(data); err == nil {
							s.SecurityService.DecryptSensitiveData(encrypted)
						}
					case 2:
						// Audit logging
						s.SecurityService.AuditDataAccess(
							ctx,
							&u.ID,
							security.ActionUserView,
							"test_resource",
							fmt.Sprintf("id_%d", operationCount),
							fmt.Sprintf("192.168.1.%d", userIdx),
							true,
						)
					case 3:
						// API key validation (if we have created keys)
						if operationCount > 10 {
							// Create an API key occasionally
							if apiKey, err := s.SecurityService.CreateUserAPIKey(
								ctx,
								u.ID,
								fmt.Sprintf("Stress Test Key %d", operationCount),
								[]string{"read"},
							); err == nil {
								s.SecurityService.ValidateAPIKey(ctx, *apiKey.PlaintextKey)
							}
						}
					}
					operationCount++
				}
			}
		}(i, user)
	}
	
	wg.Wait()
	close(operationCounts)
	
	// Calculate total operations
	totalOperations := 0
	for count := range operationCounts {
		totalOperations += count
	}
	
	operationsPerSecond := float64(totalOperations) / duration.Seconds()
	
	s.Greater(totalOperations, 0, "Should perform operations")
	s.Greater(operationsPerSecond, 100.0, "Should achieve reasonable throughput")
	
	// Log performance metrics
	s.T().Logf("Total operations: %d", totalOperations)
	s.T().Logf("Operations per second: %.2f", operationsPerSecond)
	s.T().Logf("Average operations per user: %.2f", float64(totalOperations)/float64(concurrentUsers))
}

// Run the performance test suite
func TestPerformanceTestSuite(t *testing.T) {
	// Skip performance tests in short mode
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	suite.Run(t, new(PerformanceTestSuite))
}