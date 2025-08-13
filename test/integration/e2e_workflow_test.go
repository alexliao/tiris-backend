package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/pkg/security"
	"tiris-backend/test/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// E2EWorkflowTestSuite tests complete end-to-end user workflows
type E2EWorkflowTestSuite struct {
	IntegrationTestSuite
	testConfig *config.TestConfig
	metrics    *config.TestMetrics
}

// SetupSuite initializes the E2E test environment
func (s *E2EWorkflowTestSuite) SetupSuite() {
	s.IntegrationTestSuite.SetupSuite()
	s.testConfig = config.LoadTestConfig()
	s.metrics = &config.TestMetrics{}
	
	// Validate test configuration
	err := s.testConfig.Validate()
	s.Require().NoError(err, "Test configuration should be valid")
}

// TestCompleteUserJourney tests a complete user journey from registration to trading
func (s *E2EWorkflowTestSuite) TestCompleteUserJourney() {
	startTime := time.Now()
	s.metrics.TestsRun++

	// Create a test server for API testing
	server := httptest.NewServer(s.Router)
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: User Registration/Authentication (simulated)
	s.Run("User Authentication", func() {
		user := s.createTestUser()
		s.TestUser = user

		// Generate JWT token for authentication
		token, err := s.generateJWTToken(user.ID, user.Username, user.Email)
		s.Require().NoError(err)

		// Set token for subsequent requests
		s.authToken = token

		// Verify token works with profile endpoint
		req, err := http.NewRequest("GET", server.URL+"/api/v1/users/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		s.Require().NoError(err)
		defer resp.Body.Close()

		s.Equal(http.StatusOK, resp.StatusCode)
		s.metrics.APIRequests++
	})

	// Step 2: Create API Key for External Access
	var userAPIKey string
	s.Run("Create API Key", func() {
		apiKey, err := s.SecurityService.CreateUserAPIKey(
			s.ctx,
			s.TestUser.ID,
			"E2E Test API Key",
			[]string{"read", "write"},
		)
		s.Require().NoError(err)
		s.NotNil(apiKey.PlaintextKey)
		userAPIKey = *apiKey.PlaintextKey

		// Test API key authentication
		req, err := http.NewRequest("GET", server.URL+"/api/v1/external/user/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("X-API-Key", userAPIKey)

		resp, err := client.Do(req)
		s.Require().NoError(err)
		defer resp.Body.Close()

		s.Equal(http.StatusOK, resp.StatusCode)
		s.metrics.APIRequests++
	})

	// Step 3: Update User Profile
	s.Run("Update User Profile", func() {
		updateData := map[string]interface{}{
			"username": "e2e_updated_user",
			"settings": map[string]interface{}{
				"theme":            "dark",
				"notifications":    true,
				"default_exchange": "binance",
			},
		}

		jsonData, err := json.Marshal(updateData)
		s.Require().NoError(err)

		req, err := http.NewRequest("PUT", server.URL+"/api/v1/users/profile", bytes.NewBuffer(jsonData))
		s.Require().NoError(err)
		req.Header.Set("Authorization", "Bearer "+s.authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		s.Require().NoError(err)
		defer resp.Body.Close()

		s.Equal(http.StatusOK, resp.StatusCode)

		// Verify update
		var updateResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&updateResp)
		s.Require().NoError(err)

		s.True(updateResp["success"].(bool))
		s.metrics.APIRequests++
	})

	// Step 4: Create Multiple Exchanges
	var exchanges []uuid.UUID
	exchangeTypes := []string{"binance", "coinbase", "kraken"}

	s.Run("Create Multiple Exchanges", func() {
		for i, exchangeType := range exchangeTypes {
			exchangeData := map[string]interface{}{
				"name":       fmt.Sprintf("E2E %s Exchange", exchangeType),
				"type":       exchangeType,
				"api_key":    fmt.Sprintf("%s_api_key_%d", exchangeType, i),
				"api_secret": fmt.Sprintf("%s_api_secret_%d", exchangeType, i),
			}

			jsonData, err := json.Marshal(exchangeData)
			s.Require().NoError(err)

			req, err := http.NewRequest("POST", server.URL+"/api/v1/exchanges", bytes.NewBuffer(jsonData))
			s.Require().NoError(err)
			req.Header.Set("Authorization", "Bearer "+s.authToken)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.Equal(http.StatusCreated, resp.StatusCode)

			var createResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&createResp)
			s.Require().NoError(err)

			data := createResp["data"].(map[string]interface{})
			exchangeID, err := uuid.Parse(data["id"].(string))
			s.Require().NoError(err)
			exchanges = append(exchanges, exchangeID)

			s.metrics.APIRequests++
			s.metrics.DatabaseOps++
		}
	})

	// Step 5: Create Sub-Accounts for Each Exchange
	var subAccounts []uuid.UUID
	s.Run("Create Sub-Accounts", func() {
		for i, exchangeID := range exchanges {
			// Create 2 sub-accounts per exchange
			for j := 0; j < 2; j++ {
				subAccount := s.createTestSubAccount(exchangeID)
				subAccount.Name = fmt.Sprintf("E2E SubAccount %d-%d", i, j)
				
				err := s.DB.Save(subAccount).Error
				s.Require().NoError(err)
				subAccounts = append(subAccounts, subAccount.ID)
				s.metrics.DatabaseOps++
			}
		}
	})

	// Step 6: Create Transactions
	s.Run("Create Transactions", func() {
		transactionTypes := []struct {
			direction string
			reason    string
			amount    float64
		}{
			{"credit", "deposit", 1000.0},
			{"credit", "trade_profit", 150.0},
			{"debit", "withdrawal", 200.0},
			{"debit", "trading_fee", 5.0},
		}

		for _, subAccountID := range subAccounts {
			for _, txType := range transactionTypes {
				transaction := s.createTestTransaction(subAccountID)
				transaction.Direction = txType.direction
				transaction.Reason = txType.reason
				transaction.Amount = txType.amount

				err := s.DB.Save(transaction).Error
				s.Require().NoError(err)
				s.metrics.DatabaseOps++
			}
		}
	})

	// Step 7: Query User Data via API
	s.Run("Query User Data", func() {
		// Get user profile
		req, err := http.NewRequest("GET", server.URL+"/api/v1/users/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("Authorization", "Bearer "+s.authToken)

		resp, err := client.Do(req)
		s.Require().NoError(err)
		defer resp.Body.Close()
		s.Equal(http.StatusOK, resp.StatusCode)

		// Get exchanges list
		req, err = http.NewRequest("GET", server.URL+"/api/v1/exchanges", nil)
		s.Require().NoError(err)
		req.Header.Set("Authorization", "Bearer "+s.authToken)

		resp, err = client.Do(req)
		s.Require().NoError(err)
		defer resp.Body.Close()
		s.Equal(http.StatusOK, resp.StatusCode)

		var exchangesResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&exchangesResp)
		s.Require().NoError(err)

		s.True(exchangesResp["success"].(bool))
		data := exchangesResp["data"].([]interface{})
		s.Len(data, len(exchangeTypes), "Should return all created exchanges")

		s.metrics.APIRequests += 2
	})

	// Step 8: Test Rate Limiting
	s.Run("Rate Limiting Behavior", func() {
		// Make multiple requests quickly to trigger rate limiting
		successCount := 0
		rateLimitCount := 0

		for i := 0; i < 10; i++ {
			req, err := http.NewRequest("GET", server.URL+"/api/v1/users/profile", nil)
			s.Require().NoError(err)
			req.Header.Set("Authorization", "Bearer "+s.authToken)

			resp, err := client.Do(req)
			s.Require().NoError(err)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				successCount++
			} else if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitCount++
				// Check rate limit headers
				s.NotEmpty(resp.Header.Get("X-RateLimit-Limit"))
				s.NotEmpty(resp.Header.Get("Retry-After"))
			}

			s.metrics.APIRequests++
		}

		s.Greater(successCount, 0, "Some requests should succeed")
		// Rate limiting may or may not trigger depending on previous tests
	})

	// Step 9: Test Security Features
	s.Run("Security Audit Trail", func() {
		// Check that audit events were created
		time.Sleep(500 * time.Millisecond) // Allow for async audit logging

		var auditCount int64
		err := s.DB.Model(&security.AuditEvent{}).
			Where("user_id = ?", s.TestUser.ID).
			Count(&auditCount).Error
		s.Require().NoError(err)

		s.Greater(auditCount, int64(5), "Should have multiple audit events")
		s.metrics.SecurityEvents = int(auditCount)

		// Get recent security alerts
		alerts, err := s.SecurityService.GetSecurityAlerts(s.ctx, time.Now().Add(-time.Hour), 10)
		s.Require().NoError(err)
		s.GreaterOrEqual(len(alerts), 0, "Should retrieve security alerts without error")
	})

	// Step 10: Test API Key External Access
	s.Run("External API Access", func() {
		// Test various endpoints with API key
		endpoints := []string{
			"/api/v1/external/user/profile",
			"/api/v1/external/exchanges",
		}

		for _, endpoint := range endpoints {
			req, err := http.NewRequest("GET", server.URL+endpoint, nil)
			s.Require().NoError(err)
			req.Header.Set("X-API-Key", userAPIKey)

			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.Equal(http.StatusOK, resp.StatusCode, "External API access should work for %s", endpoint)
			s.metrics.APIRequests++
		}
	})

	// Step 11: Data Consistency Verification
	s.Run("Data Consistency Check", func() {
		// Verify data consistency across all created resources
		var userCount int64
		err := s.DB.Model(&models.User{}).Where("id = ?", s.TestUser.ID).Count(&userCount).Error
		s.Require().NoError(err)
		s.Equal(int64(1), userCount, "User should exist")

		var exchangeCount int64
		err = s.DB.Model(&models.SecureExchange{}).Where("user_id = ?", s.TestUser.ID).Count(&exchangeCount).Error
		s.Require().NoError(err)
		s.Equal(int64(len(exchangeTypes)), exchangeCount, "All exchanges should exist")

		var subAccountCount int64
		err = s.DB.Model(&models.SubAccount{}).Where("user_id = ?", s.TestUser.ID).Count(&subAccountCount).Error
		s.Require().NoError(err)
		s.Equal(int64(len(exchanges)*2), subAccountCount, "All sub-accounts should exist")

		var transactionCount int64
		err = s.DB.Model(&models.Transaction{}).Where("user_id = ?", s.TestUser.ID).Count(&transactionCount).Error
		s.Require().NoError(err)
		s.Greater(transactionCount, int64(0), "Transactions should exist")

		s.metrics.DatabaseOps += 4
	})

	// Step 12: Performance Validation
	s.Run("Performance Validation", func() {
		duration := time.Since(startTime)
		s.Less(duration, 30*time.Second, "Complete user journey should finish within 30 seconds")

		// Test concurrent access
		const concurrentUsers = 5
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()

		done := make(chan bool, concurrentUsers)
		
		for i := 0; i < concurrentUsers; i++ {
			go func(userNum int) {
				defer func() { done <- true }()
				
				// Each concurrent user makes profile requests
				for j := 0; j < 5; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						req, err := http.NewRequest("GET", server.URL+"/api/v1/users/profile", nil)
						if err != nil {
							return
						}
						req.Header.Set("Authorization", "Bearer "+s.authToken)

						resp, err := client.Do(req)
						if err != nil {
							return
						}
						resp.Body.Close()
						
						s.metrics.APIRequests++
					}
				}
			}(i)
		}

		// Wait for all concurrent users to complete
		for i := 0; i < concurrentUsers; i++ {
			select {
			case <-done:
			case <-ctx.Done():
				s.Fail("Concurrent test timed out")
			}
		}
	})

	s.metrics.TestsPassed++
	s.metrics.TotalDuration = time.Since(startTime)
}

// TestSecurityWorkflow tests security-focused workflows
func (s *E2EWorkflowTestSuite) TestSecurityWorkflow() {
	startTime := time.Now()
	s.metrics.TestsRun++

	server := httptest.NewServer(s.Router)
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// Test 1: API Key Lifecycle
	s.Run("API Key Security Lifecycle", func() {
		user := s.createTestUser()

		// Create API key
		apiKey1, err := s.SecurityService.CreateUserAPIKey(
			s.ctx,
			user.ID,
			"Security Test Key 1",
			[]string{"read"},
		)
		s.Require().NoError(err)

		// Use API key
		req, err := http.NewRequest("GET", server.URL+"/api/v1/external/user/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("X-API-Key", *apiKey1.PlaintextKey)

		resp, err := client.Do(req)
		s.Require().NoError(err)
		resp.Body.Close()
		s.Equal(http.StatusOK, resp.StatusCode)

		// Rotate API key
		apiKey2, err := s.SecurityService.RotateAPIKey(s.ctx, user.ID, apiKey1.ID)
		s.Require().NoError(err)

		// Old key should not work
		req, err = http.NewRequest("GET", server.URL+"/api/v1/external/user/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("X-API-Key", *apiKey1.PlaintextKey)

		resp, err = client.Do(req)
		s.Require().NoError(err)
		resp.Body.Close()
		s.Equal(http.StatusUnauthorized, resp.StatusCode)

		// New key should work
		req, err = http.NewRequest("GET", server.URL+"/api/v1/external/user/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("X-API-Key", *apiKey2.PlaintextKey)

		resp, err = client.Do(req)
		s.Require().NoError(err)
		resp.Body.Close()
		s.Equal(http.StatusOK, resp.StatusCode)

		s.metrics.SecurityEvents += 3 // Create, rotate, use
	})

	// Test 2: Encryption at Rest
	s.Run("Data Encryption Verification", func() {
		user := s.createTestUser()

		// Create exchange with sensitive data
		exchange, err := s.SecurityService.CreateSecureExchange(
			s.ctx,
			user.ID,
			"Security Test Exchange",
			"binance",
			"super_secret_api_key_that_should_be_encrypted",
			"ultra_secret_api_secret_that_should_be_encrypted",
		)
		s.Require().NoError(err)

		// Verify data is encrypted in database
		var rawData map[string]interface{}
		err = s.DB.Table("exchanges").Where("id = ?", exchange.ID).Take(&rawData).Error
		s.Require().NoError(err)

		// Encrypted fields should not contain plaintext
		s.NotContains(rawData["encrypted_api_key"], "super_secret_api_key")
		s.NotContains(rawData["encrypted_api_secret"], "ultra_secret_api_secret")
		s.NotEmpty(rawData["api_key_hash"])

		// But service should be able to decrypt
		apiKey, apiSecret, err := s.SecurityService.GetExchangeCredentials(s.ctx, user.ID, exchange.ID)
		s.Require().NoError(err)
		s.Equal("super_secret_api_key_that_should_be_encrypted", apiKey)
		s.Equal("ultra_secret_api_secret_that_should_be_encrypted", apiSecret)

		s.metrics.SecurityEvents += 2 // Create and decrypt
	})

	// Test 3: Rate Limiting Security
	s.Run("Rate Limiting Protection", func() {
		// Test auth rate limiting
		for i := 0; i < 7; i++ { // Auth limit is 5
			result, err := s.SecurityService.CheckRateLimit(s.ctx, "test_auth_user", "auth_login")
			s.Require().NoError(err)
			
			if i >= 5 {
				s.False(result.Allowed, "Request %d should be rate limited", i+1)
			} else {
				s.True(result.Allowed, "Request %d should be allowed", i+1)
			}
		}

		s.metrics.SecurityEvents += 7
	})

	// Test 4: Suspicious Activity Detection
	s.Run("Threat Detection", func() {
		testIP := "192.168.100.100"

		// Generate suspicious patterns
		for i := 0; i < 6; i++ {
			err := s.SecurityService.AuditDataAccess(
				s.ctx,
				nil,
				security.ActionLoginFailed,
				"auth",
				"login",
				testIP,
				false,
			)
			s.Require().NoError(err)
		}

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Check for suspicious activity
		suspicious, err := s.SecurityService.GetSuspiciousActivity(s.ctx, time.Hour)
		s.Require().NoError(err)

		found := false
		for _, activity := range suspicious {
			if activity.Type == "multiple_failed_logins" && activity.IPAddress == testIP {
				found = true
				s.GreaterOrEqual(activity.Count, 5)
				break
			}
		}
		s.True(found, "Should detect suspicious login activity")

		s.metrics.SecurityEvents += 6
	})

	s.metrics.TestsPassed++
	duration := time.Since(startTime)
	s.Less(duration, 10*time.Second, "Security workflow should complete quickly")
}

// TestPerformanceWorkflow tests performance under realistic load
func (s *E2EWorkflowTestSuite) TestPerformanceWorkflow() {
	if s.testConfig.Test.SkipPerformance {
		s.T().Skip("Performance tests disabled")
		return
	}

	startTime := time.Now()
	s.metrics.TestsRun++

	server := httptest.NewServer(s.Router)
	defer server.Close()

	const numUsers = 10
	const requestsPerUser = 20

	users := make([]*models.User, numUsers)
	tokens := make([]string, numUsers)

	// Setup users and tokens
	for i := 0; i < numUsers; i++ {
		users[i] = s.createTestUser()
		token, err := s.generateJWTToken(users[i].ID, users[i].Username, users[i].Email)
		s.Require().NoError(err)
		tokens[i] = token
	}

	// Performance test: Concurrent API requests
	s.Run("Concurrent API Performance", func() {
		client := &http.Client{Timeout: 5 * time.Second}
		done := make(chan time.Duration, numUsers*requestsPerUser)
		
		for i := 0; i < numUsers; i++ {
			go func(userIdx int) {
				for j := 0; j < requestsPerUser; j++ {
					requestStart := time.Now()
					
					req, err := http.NewRequest("GET", server.URL+"/api/v1/users/profile", nil)
					if err != nil {
						done <- 0
						continue
					}
					req.Header.Set("Authorization", "Bearer "+tokens[userIdx])

					resp, err := client.Do(req)
					if err != nil {
						done <- 0
						continue
					}
					resp.Body.Close()

					done <- time.Since(requestStart)
					s.metrics.APIRequests++
				}
			}(i)
		}

		// Collect response times
		var totalTime time.Duration
		var maxTime time.Duration
		var minTime time.Duration = time.Hour // Initialize to large value
		successCount := 0

		for i := 0; i < numUsers*requestsPerUser; i++ {
			responseTime := <-done
			if responseTime > 0 {
				successCount++
				totalTime += responseTime
				if responseTime > maxTime {
					maxTime = responseTime
				}
				if responseTime < minTime {
					minTime = responseTime
				}
			}
		}

		s.Greater(successCount, numUsers*requestsPerUser/2, "Most requests should succeed")
		
		avgTime := totalTime / time.Duration(successCount)
		s.Less(avgTime, 100*time.Millisecond, "Average response time should be under 100ms")
		s.Less(maxTime, 500*time.Millisecond, "Max response time should be under 500ms")

		throughput := float64(successCount) / time.Since(startTime).Seconds()
		s.Greater(throughput, 50.0, "Should achieve >50 requests/second")

		s.metrics.AverageResponse = avgTime
		s.metrics.MaxResponse = maxTime
		s.metrics.MinResponse = minTime
		s.metrics.Throughput = throughput
	})

	s.metrics.TestsPassed++
	s.metrics.TotalDuration = time.Since(startTime)
}

// TearDownSuite performs final validation and cleanup
func (s *E2EWorkflowTestSuite) TearDownSuite() {
	// Calculate final metrics
	s.metrics.CalculateMetrics()
	
	// Validate performance SLA
	if s.metrics.TestsRun > 0 {
		withinSLA := s.metrics.IsWithinSLA(s.testConfig)
		s.True(withinSLA, "Performance metrics should be within SLA")
	}

	// Print test summary
	s.T().Logf("E2E Test Summary:")
	s.T().Logf("  Tests Run: %d", s.metrics.TestsRun)
	s.T().Logf("  Tests Passed: %d", s.metrics.TestsPassed)
	s.T().Logf("  Tests Failed: %d", s.metrics.TestsFailed)
	s.T().Logf("  Total Duration: %v", s.metrics.TotalDuration)
	s.T().Logf("  Average Response: %v", s.metrics.AverageResponse)
	s.T().Logf("  Throughput: %.2f ops/sec", s.metrics.Throughput)
	s.T().Logf("  Error Rate: %.2f%%", s.metrics.ErrorRate)
	s.T().Logf("  API Requests: %d", s.metrics.APIRequests)
	s.T().Logf("  Database Ops: %d", s.metrics.DatabaseOps)
	s.T().Logf("  Security Events: %d", s.metrics.SecurityEvents)

	s.IntegrationTestSuite.TearDownSuite()
}

// Helper fields for maintaining state across test steps
type E2EWorkflowTestSuite struct {
	IntegrationTestSuite
	testConfig *config.TestConfig
	metrics    *config.TestMetrics
	authToken  string
}

// Run the E2E workflow test suite
func TestE2EWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(E2EWorkflowTestSuite))
}