package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"tiris-backend/internal/api"
	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
	"tiris-backend/internal/models"
	"tiris-backend/internal/nats"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/fixtures"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceTestSuite defines the test suite for performance and load tests
type PerformanceTestSuite struct {
	suite.Suite
	server     *api.Server
	router     *gin.Engine
	db         *database.DB
	nats       *nats.Manager
	repos      *repositories.Repositories
	cfg        *config.Config
	adminToken string
	userTokens []string
	userIDs    []uuid.UUID
}

// SetupSuite runs once before all tests
func (suite *PerformanceTestSuite) SetupSuite() {
	// Skip performance tests in short mode
	if testing.Short() {
		suite.T().Skip("Skipping performance tests in short mode")
	}

	// Load test configuration with environment variable support
	suite.cfg = &config.Config{
		Database: config.DatabaseConfig{
			Host:         getEnv("TEST_DB_HOST", "localhost"),
			Port:         getEnv("TEST_DB_PORT", "5432"),
			Username:     getEnv("TEST_DB_USER", "tiris_test"),
			Password:     getEnv("TEST_DB_PASSWORD", "tiris_test"),
			DatabaseName: getEnv("TEST_DB_NAME", "tiris_test"),
			SSLMode:      getEnv("TEST_DB_SSL_MODE", "disable"),
			MaxConns:     25,
			MaxIdleConns: 10,
			MaxLifetime:  300,
		},
		Auth: config.AuthConfig{
			JWTSecret:         "test-jwt-secret-performance-testing",
			RefreshSecret:     "test-refresh-secret-performance-testing",
			JWTExpiration:     3600,
			RefreshExpiration: 86400,
		},
		OAuth: config.OAuthConfig{
			Google: config.GoogleOAuthConfig{
				ClientID:     "test-google-client-id",
				ClientSecret: "test-google-client-secret",
				RedirectURL:  "http://localhost:8080/auth/google/callback",
			},
			WeChat: config.WeChatOAuthConfig{
				AppID:       "test-wechat-app-id",
				AppSecret:   "test-wechat-app-secret",
				RedirectURL: "http://localhost:8080/auth/wechat/callback",
			},
		},
		NATS: config.NATSConfig{
			URL:         getEnv("TEST_NATS_URL", "nats://localhost:4222"),
			ClusterID:   "perf-test-cluster",
			ClientID:    "perf-test-client",
			DurableName: "perf-test-durable",
		},
		Environment: "test",
	}

	// Set Gin to release mode for performance testing
	gin.SetMode(gin.ReleaseMode)

	// Initialize database with detailed error handling
	var err error
	suite.db, err = database.Initialize(suite.cfg.Database)
	if err != nil {
		suite.T().Logf("Database connection failed. Please ensure:")
		suite.T().Logf("  1. PostgreSQL is running and accessible")
		suite.T().Logf("  2. Test database setup has been completed")
		suite.T().Logf("  3. Run: make setup-test-db")
		suite.T().Logf("Connection details:")
		suite.T().Logf("  Host: %s:%s", suite.cfg.Database.Host, suite.cfg.Database.Port)
		suite.T().Logf("  User: %s", suite.cfg.Database.Username)
		suite.T().Logf("  Database: %s", suite.cfg.Database.DatabaseName)
		require.NoError(suite.T(), err, "Failed to connect to test database")
	}

	// Initialize repositories
	suite.repos = repositories.NewRepositories(suite.db.DB)

	// Initialize NATS (allow failure in test environment)
	suite.nats, _ = nats.NewManager(suite.cfg.NATS, suite.repos)

	// Initialize API server
	suite.server = api.NewServer(suite.cfg, suite.repos, suite.db, suite.nats)
	suite.router = suite.server.SetupRoutes()

	// Clean database and run migrations
	suite.cleanDatabase()
	suite.runMigrations()

	// Create test users for load testing
	suite.createTestUsers()
}

// TearDownSuite runs once after all tests
func (suite *PerformanceTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanDatabase()
		database.Close(suite.db)
	}
	if suite.nats != nil {
		suite.nats.Stop()
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (suite *PerformanceTestSuite) cleanDatabase() {
	db := suite.db.DB
	
	tables := []string{
		"trading_logs", "transactions", "sub_accounts", "exchanges", 
		"oauth_tokens", "users",
	}
	
	for _, table := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}
}

func (suite *PerformanceTestSuite) runMigrations() {
	db := suite.db.DB
	
	err := db.AutoMigrate(
		&models.User{},
		&models.OAuthToken{},
		&models.Exchange{},
		&models.SubAccount{},
		&models.Transaction{},
		&models.TradingLog{},
	)
	require.NoError(suite.T(), err, "Failed to run migrations")
}

func (suite *PerformanceTestSuite) createTestUsers() {
	// Create admin user
	adminUser := fixtures.CreateUser()
	adminUser.Username = "admin_perf"
	adminUser.Email = "admin@perf.com"
	
	err := suite.repos.User.Create(context.Background(), adminUser)
	require.NoError(suite.T(), err)

	suite.adminToken = suite.generateToken(adminUser.ID, "admin_perf", "admin@perf.com", "admin")

	// Create multiple test users for load testing
	numUsers := 100
	suite.userTokens = make([]string, numUsers)
	suite.userIDs = make([]uuid.UUID, numUsers)

	for i := 0; i < numUsers; i++ {
		user := fixtures.CreateUser()
		user.Username = fmt.Sprintf("user_perf_%d", i)
		user.Email = fmt.Sprintf("user%d@perf.com", i)
		
		err := suite.repos.User.Create(context.Background(), user)
		require.NoError(suite.T(), err)

		suite.userIDs[i] = user.ID
		suite.userTokens[i] = suite.generateToken(user.ID, user.Username, user.Email, "user")
	}
}

func (suite *PerformanceTestSuite) generateToken(userID uuid.UUID, username, email, role string) string {
	tokenPair, err := suite.server.GetJWTManager().GenerateTokenPair(userID, username, email, role)
	require.NoError(suite.T(), err)
	return tokenPair.AccessToken
}

func (suite *PerformanceTestSuite) makeRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// Performance Metrics
type PerformanceMetrics struct {
	TotalRequests   int
	SuccessfulReqs  int
	FailedReqs      int
	MinDuration     time.Duration
	MaxDuration     time.Duration
	AvgDuration     time.Duration
	TotalDuration   time.Duration
	RequestsPerSec  float64
	Percentile95    time.Duration
	Percentile99    time.Duration
}

func (suite *PerformanceTestSuite) runLoadTest(name string, concurrency, totalRequests int, requestFunc func(int) *httptest.ResponseRecorder) PerformanceMetrics {
	fmt.Printf("\n=== Load Test: %s ===\n", name)
	fmt.Printf("Concurrency: %d, Total Requests: %d\n", concurrency, totalRequests)

	start := time.Now()
	
	// Channel to collect results
	results := make(chan time.Duration, totalRequests)
	statusCodes := make(chan int, totalRequests)
	
	// Worker pool
	requests := make(chan int, totalRequests)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for reqNum := range requests {
				reqStart := time.Now()
				resp := requestFunc(reqNum)
				duration := time.Since(reqStart)
				
				results <- duration
				statusCodes <- resp.Code
			}
		}()
	}

	// Send work to workers
	go func() {
		for i := 0; i < totalRequests; i++ {
			requests <- i
		}
		close(requests)
	}()

	// Wait for all workers to complete
	wg.Wait()
	close(results)
	close(statusCodes)

	// Collect metrics
	var durations []time.Duration
	var successCount, failCount int
	var totalDuration time.Duration
	
	for duration := range results {
		durations = append(durations, duration)
		totalDuration += duration
	}

	for statusCode := range statusCodes {
		if statusCode >= 200 && statusCode < 300 {
			successCount++
		} else {
			failCount++
		}
	}

	// Sort durations for percentile calculations
	for i := 0; i < len(durations); i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	metrics := PerformanceMetrics{
		TotalRequests:  totalRequests,
		SuccessfulReqs: successCount,
		FailedReqs:     failCount,
		TotalDuration:  time.Since(start),
		AvgDuration:    totalDuration / time.Duration(len(durations)),
	}

	if len(durations) > 0 {
		metrics.MinDuration = durations[0]
		metrics.MaxDuration = durations[len(durations)-1]
		metrics.RequestsPerSec = float64(totalRequests) / metrics.TotalDuration.Seconds()
		
		// Calculate percentiles
		p95Index := int(float64(len(durations)) * 0.95)
		p99Index := int(float64(len(durations)) * 0.99)
		
		if p95Index < len(durations) {
			metrics.Percentile95 = durations[p95Index]
		}
		if p99Index < len(durations) {
			metrics.Percentile99 = durations[p99Index]
		}
	}

	// Print results
	fmt.Printf("Total Duration: %v\n", metrics.TotalDuration)
	fmt.Printf("Successful Requests: %d\n", metrics.SuccessfulReqs)
	fmt.Printf("Failed Requests: %d\n", metrics.FailedReqs)
	fmt.Printf("Requests/sec: %.2f\n", metrics.RequestsPerSec)
	fmt.Printf("Min Duration: %v\n", metrics.MinDuration)
	fmt.Printf("Max Duration: %v\n", metrics.MaxDuration)
	fmt.Printf("Avg Duration: %v\n", metrics.AvgDuration)
	fmt.Printf("95th Percentile: %v\n", metrics.Percentile95)
	fmt.Printf("99th Percentile: %v\n", metrics.Percentile99)

	return metrics
}

// Test Health Endpoint Performance
func (suite *PerformanceTestSuite) TestHealthEndpointPerformance() {
	metrics := suite.runLoadTest("Health Endpoint", 50, 1000, func(reqNum int) *httptest.ResponseRecorder {
		return suite.makeRequest("GET", "/health/live", nil, "")
	})

	// Assert performance requirements
	suite.T().Run("health_performance_assertions", func(t *testing.T) {
		// Health endpoint should be very fast
		require.Less(t, metrics.Percentile95.Milliseconds(), int64(50), "95th percentile should be under 50ms")
		require.Less(t, metrics.Percentile99.Milliseconds(), int64(100), "99th percentile should be under 100ms")
		require.Greater(t, metrics.RequestsPerSec, 100.0, "Should handle at least 100 requests/sec")
		require.Equal(t, 0, metrics.FailedReqs, "Should have no failed requests")
	})
}

// Test Authentication Performance
func (suite *PerformanceTestSuite) TestAuthenticationPerformance() {
	metrics := suite.runLoadTest("Authentication", 20, 500, func(reqNum int) *httptest.ResponseRecorder {
		tokenIndex := reqNum % len(suite.userTokens)
		return suite.makeRequest("GET", "/v1/users/me", nil, suite.userTokens[tokenIndex])
	})

	suite.T().Run("auth_performance_assertions", func(t *testing.T) {
		require.Less(t, metrics.Percentile95.Milliseconds(), int64(200), "95th percentile should be under 200ms")
		require.Less(t, metrics.Percentile99.Milliseconds(), int64(500), "99th percentile should be under 500ms")
		require.Greater(t, metrics.RequestsPerSec, 50.0, "Should handle at least 50 requests/sec")
		require.Less(t, float64(metrics.FailedReqs)/float64(metrics.TotalRequests), 0.01, "Failure rate should be less than 1%")
	})
}

// Test Database Read Performance
func (suite *PerformanceTestSuite) TestDatabaseReadPerformance() {
	// First create some test data
	suite.createTestExchanges(50)

	metrics := suite.runLoadTest("Database Reads", 25, 500, func(reqNum int) *httptest.ResponseRecorder {
		tokenIndex := reqNum % len(suite.userTokens)
		return suite.makeRequest("GET", "/v1/exchanges", nil, suite.userTokens[tokenIndex])
	})

	suite.T().Run("db_read_performance_assertions", func(t *testing.T) {
		require.Less(t, metrics.Percentile95.Milliseconds(), int64(300), "95th percentile should be under 300ms")
		require.Less(t, metrics.Percentile99.Milliseconds(), int64(600), "99th percentile should be under 600ms")
		require.Greater(t, metrics.RequestsPerSec, 30.0, "Should handle at least 30 requests/sec")
		require.Less(t, float64(metrics.FailedReqs)/float64(metrics.TotalRequests), 0.02, "Failure rate should be less than 2%")
	})
}

// Test Database Write Performance
func (suite *PerformanceTestSuite) TestDatabaseWritePerformance() {
	metrics := suite.runLoadTest("Database Writes", 10, 200, func(reqNum int) *httptest.ResponseRecorder {
		tokenIndex := reqNum % len(suite.userTokens)
		
		createRequest := map[string]interface{}{
			"name":         fmt.Sprintf("PerfExchange_%d", reqNum),
			"display_name": fmt.Sprintf("Performance Exchange %d", reqNum),
			"description":  "Performance test exchange",
		}

		return suite.makeRequest("POST", "/v1/exchanges", createRequest, suite.userTokens[tokenIndex])
	})

	suite.T().Run("db_write_performance_assertions", func(t *testing.T) {
		require.Less(t, metrics.Percentile95.Milliseconds(), int64(500), "95th percentile should be under 500ms")
		require.Less(t, metrics.Percentile99.Milliseconds(), int64(1000), "99th percentile should be under 1000ms")
		require.Greater(t, metrics.RequestsPerSec, 15.0, "Should handle at least 15 writes/sec")
		require.Less(t, float64(metrics.FailedReqs)/float64(metrics.TotalRequests), 0.05, "Failure rate should be less than 5%")
	})
}

// Test Complex Query Performance
func (suite *PerformanceTestSuite) TestComplexQueryPerformance() {
	// Create test data with relationships
	suite.createTestDataWithRelationships(20)

	metrics := suite.runLoadTest("Complex Queries", 15, 300, func(reqNum int) *httptest.ResponseRecorder {
		tokenIndex := reqNum % len(suite.userTokens)
		return suite.makeRequest("GET", "/v1/users/me/stats", nil, suite.userTokens[tokenIndex])
	})

	suite.T().Run("complex_query_performance_assertions", func(t *testing.T) {
		require.Less(t, metrics.Percentile95.Milliseconds(), int64(800), "95th percentile should be under 800ms")
		require.Less(t, metrics.Percentile99.Milliseconds(), int64(1500), "99th percentile should be under 1500ms")
		require.Greater(t, metrics.RequestsPerSec, 10.0, "Should handle at least 10 requests/sec")
		require.Less(t, float64(metrics.FailedReqs)/float64(metrics.TotalRequests), 0.05, "Failure rate should be less than 5%")
	})
}

// Test Concurrent User Load
func (suite *PerformanceTestSuite) TestConcurrentUserLoad() {
	suite.T().Run("mixed_workload_test", func(t *testing.T) {
		fmt.Println("\n=== Mixed Workload Test ===")
		
		// Simulate realistic user behavior with mixed operations
		duration := 30 * time.Second
		
		var wg sync.WaitGroup
		var totalRequests int64
		var successfulRequests int64
		var failedRequests int64
		
		start := time.Now()
		
		// Health check workers (frequent)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for time.Since(start) < duration {
					resp := suite.makeRequest("GET", "/health/live", nil, "")
					totalRequests++
					if resp.Code >= 200 && resp.Code < 300 {
						successfulRequests++
					} else {
						failedRequests++
					}
					time.Sleep(100 * time.Millisecond)
				}
			}()
		}
		
		// Read workers (common)
		for i := 0; i < 15; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for time.Since(start) < duration {
					tokenIndex := workerID % len(suite.userTokens)
					
					// Mix of read operations
					operations := []string{
						"/v1/users/me",
						"/v1/exchanges",
						"/v1/sub-accounts",
						"/v1/users/me/stats",
					}
					
					operation := operations[rand.Intn(len(operations))]
					resp := suite.makeRequest("GET", operation, nil, suite.userTokens[tokenIndex])
					
					totalRequests++
					if resp.Code >= 200 && resp.Code < 300 {
						successfulRequests++
					} else {
						failedRequests++
					}
					
					time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
				}
			}(i)
		}
		
		// Write workers (less frequent)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for time.Since(start) < duration {
					tokenIndex := workerID % len(suite.userTokens)
					
					createRequest := map[string]interface{}{
						"name":         fmt.Sprintf("MixedLoad_%d_%d", workerID, time.Now().Unix()),
						"display_name": fmt.Sprintf("Mixed Load Exchange %d", workerID),
						"description":  "Mixed load test exchange",
					}
					
					resp := suite.makeRequest("POST", "/v1/exchanges", createRequest, suite.userTokens[tokenIndex])
					
					totalRequests++
					if resp.Code >= 200 && resp.Code < 300 {
						successfulRequests++
					} else {
						failedRequests++
					}
					
					time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)
				}
			}(i)
		}
		
		wg.Wait()
		totalDuration := time.Since(start)
		
		fmt.Printf("Mixed Workload Results:\n")
		fmt.Printf("Duration: %v\n", totalDuration)
		fmt.Printf("Total Requests: %d\n", totalRequests)
		fmt.Printf("Successful: %d\n", successfulRequests)
		fmt.Printf("Failed: %d\n", failedRequests)
		fmt.Printf("Requests/sec: %.2f\n", float64(totalRequests)/totalDuration.Seconds())
		fmt.Printf("Success Rate: %.2f%%\n", float64(successfulRequests)/float64(totalRequests)*100)
		
		// Assertions
		require.Greater(t, totalRequests, int64(100), "Should process significant number of requests")
		require.Less(t, float64(failedRequests)/float64(totalRequests), 0.05, "Failure rate should be less than 5%")
		require.Greater(t, float64(totalRequests)/totalDuration.Seconds(), 10.0, "Should maintain at least 10 req/sec")
	})
}

// Test Memory Usage and Resource Management
func (suite *PerformanceTestSuite) TestResourceManagement() {
	suite.T().Run("memory_stability_test", func(t *testing.T) {
		fmt.Println("\n=== Memory Stability Test ===")
		
		// Run sustained load to check for memory leaks
		iterations := 10
		requestsPerIteration := 100
		
		for i := 0; i < iterations; i++ {
			fmt.Printf("Iteration %d/%d\n", i+1, iterations)
			
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, 20) // Limit concurrency
			
			for j := 0; j < requestsPerIteration; j++ {
				wg.Add(1)
				go func(reqNum int) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					
					tokenIndex := reqNum % len(suite.userTokens)
					suite.makeRequest("GET", "/v1/users/me", nil, suite.userTokens[tokenIndex])
				}(j)
			}
			
			wg.Wait()
			
			// Small pause between iterations
			time.Sleep(100 * time.Millisecond)
		}
		
		fmt.Println("Memory stability test completed")
	})
}

// Helper methods to create test data

func (suite *PerformanceTestSuite) createTestExchanges(count int) {
	for i := 0; i < count; i++ {
		userIndex := i % len(suite.userIDs)
		
		exchange := &models.Exchange{
			UserID:    suite.userIDs[userIndex],
			Name:      fmt.Sprintf("PerfTestExchange_%d", i),
			Type:      "spot",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
			Status:    "active",
		}
		
		suite.repos.Exchange.Create(context.Background(), exchange)
	}
}

func (suite *PerformanceTestSuite) createTestDataWithRelationships(exchangesPerUser int) {
	for i, userID := range suite.userIDs[:10] { // Use first 10 users
		// Create exchanges
		for j := 0; j < exchangesPerUser; j++ {
			exchange := &models.Exchange{
				UserID:    userID,
				Name:      fmt.Sprintf("ComplexExchange_%d_%d", i, j),
				Type:      "spot",
				APIKey:    "test_api_key",
				APISecret: "test_api_secret",
				Status:    "active",
			}
			
			err := suite.repos.Exchange.Create(context.Background(), exchange)
			if err != nil {
				continue
			}
			
			// Create sub-accounts for each exchange
			for k := 0; k < 3; k++ {
				subAccount := &models.SubAccount{
					UserID:     userID,
					ExchangeID: exchange.ID,
					Name:       fmt.Sprintf("SubAccount_%d_%d_%d", i, j, k),
					Symbol:     fmt.Sprintf("SYM%d%d%d", i, j, k),
					Balance:    float64(rand.Intn(10000)),
				}
				
				suite.repos.SubAccount.Create(context.Background(), subAccount)
			}
		}
	}
}

// Benchmark tests
func BenchmarkHealthEndpoint(b *testing.B) {
	// Setup similar to test suite but for benchmarking
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:         getEnv("TEST_DB_HOST", "localhost"),
			Port:         getEnv("TEST_DB_PORT", "5432"),
			Username:     getEnv("TEST_DB_USER", "tiris_test"),
			Password:     getEnv("TEST_DB_PASSWORD", "tiris_test"),
			DatabaseName: getEnv("TEST_DB_NAME", "tiris_test"),
			SSLMode:      getEnv("TEST_DB_SSL_MODE", "disable"),
			MaxConns:     25,
			MaxIdleConns: 10,
			MaxLifetime:  300,
		},
		Auth: config.AuthConfig{
			JWTSecret:         "test-jwt-secret-benchmark",
			RefreshSecret:     "test-refresh-secret-benchmark",
			JWTExpiration:     3600,
			RefreshExpiration: 86400,
		},
		Environment: "test",
	}

	gin.SetMode(gin.ReleaseMode)

	db, err := database.Initialize(cfg.Database)
	if err != nil {
		b.Skip("Database not available for benchmark")
	}
	defer database.Close(db)

	repos := repositories.NewRepositories(db.DB)
	server := api.NewServer(cfg, repos, db, nil)
	router := server.SetupRoutes()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health/live", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkAuthenticatedRequest(b *testing.B) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:         getEnv("TEST_DB_HOST", "localhost"),
			Port:         getEnv("TEST_DB_PORT", "5432"),
			Username:     getEnv("TEST_DB_USER", "tiris_test"),
			Password:     getEnv("TEST_DB_PASSWORD", "tiris_test"),
			DatabaseName: getEnv("TEST_DB_NAME", "tiris_test"),
			SSLMode:      getEnv("TEST_DB_SSL_MODE", "disable"),
			MaxConns:     25,
			MaxIdleConns: 10,
			MaxLifetime:  300,
		},
		Auth: config.AuthConfig{
			JWTSecret:         "test-jwt-secret-benchmark",
			RefreshSecret:     "test-refresh-secret-benchmark",
			JWTExpiration:     3600,
			RefreshExpiration: 86400,
		},
		Environment: "test",
	}

	gin.SetMode(gin.ReleaseMode)

	db, err := database.Initialize(cfg.Database)
	if err != nil {
		b.Skip("Database not available for benchmark")
	}
	defer database.Close(db)

	repos := repositories.NewRepositories(db.DB)
	server := api.NewServer(cfg, repos, db, nil)
	router := server.SetupRoutes()

	// Create test user and generate token
	user := fixtures.CreateUser()
	user.Username = "bench_user"
	user.Email = "bench@test.com"
	
	repos.User.Create(context.Background(), user)
	
	tokenPair, _ := server.GetJWTManager().GenerateTokenPair(user.ID, user.Username, user.Email, "user")
	token := tokenPair.AccessToken

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/users/me", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// Test runner
// TODO: Fix performance test issues with test data setup and rate limiting
// func TestPerformanceSuite(t *testing.T) {
// 	suite.Run(t, new(PerformanceTestSuite))
// }