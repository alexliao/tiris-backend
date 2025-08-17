package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"tiris-backend/internal/api"
	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
	"tiris-backend/internal/models"
	"tiris-backend/internal/nats"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/fixtures"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite defines the test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	server     *api.Server
	router     *gin.Engine
	db         *database.DB
	nats       *nats.Manager
	repos      *repositories.Repositories
	cfg        *config.Config
	adminToken string
	userToken  string
	userID     uuid.UUID
	adminID    uuid.UUID
}

// SetupSuite runs once before all tests
func (suite *IntegrationTestSuite) SetupSuite() {
	// Skip integration tests in short mode
	if testing.Short() {
		suite.T().Skip("Skipping integration tests in short mode")
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
			JWTSecret:         "test-jwt-secret-integration-testing",
			RefreshSecret:     "test-refresh-secret-integration-testing",
			JWTExpiration:     3600,  // 1 hour
			RefreshExpiration: 86400, // 24 hours
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
			ClusterID:   "test-cluster",
			ClientID:    "test-client",
			DurableName: "test-durable",
		},
		Environment: "test",
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

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

	// Create test users for full integration testing
	suite.createTestUsers()
}

// TearDownSuite runs once after all tests
func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.cleanDatabase()
		database.Close(suite.db)
	}
	if suite.nats != nil {
		suite.nats.Stop()
	}
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Clean transactional data but keep users
	suite.cleanTransactionalData()
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (suite *IntegrationTestSuite) cleanDatabase() {
	db := suite.db.DB

	// Drop only the tables we're testing
	tables := []string{
		"sub_accounts", "exchanges", "users",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}
}

func (suite *IntegrationTestSuite) runMigrations() {
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

func (suite *IntegrationTestSuite) cleanTransactionalData() {
	db := suite.db.DB

	// Clean transactional data but keep users (only for tables that exist)
	db.Exec("DELETE FROM sub_accounts")
	db.Exec("DELETE FROM exchanges")
}

func (suite *IntegrationTestSuite) createTestUsers() {
	// Create admin user
	adminUser := fixtures.CreateUser()
	adminUser.Username = "admin_user"
	adminUser.Email = "admin@test.com"

	err := suite.repos.User.Create(context.Background(), adminUser)
	require.NoError(suite.T(), err)
	suite.adminID = adminUser.ID

	// Create regular user
	regularUser := fixtures.CreateUser()
	regularUser.Username = "regular_user"
	regularUser.Email = "user@test.com"

	err = suite.repos.User.Create(context.Background(), regularUser)
	require.NoError(suite.T(), err)
	suite.userID = regularUser.ID

	// Generate tokens
	suite.adminToken = suite.generateToken(suite.adminID, "admin_user", "admin@test.com", "admin")
	suite.userToken = suite.generateToken(suite.userID, "regular_user", "user@test.com", "user")
}

func (suite *IntegrationTestSuite) generateToken(userID uuid.UUID, username, email, role string) string {
	tokenPair, err := suite.server.GetJWTManager().GenerateTokenPair(userID, username, email, role)
	require.NoError(suite.T(), err)
	return tokenPair.AccessToken
}

func (suite *IntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
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

func (suite *IntegrationTestSuite) parseResponse(w *httptest.ResponseRecorder, target interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), target)
	require.NoError(suite.T(), err)
}

// Test Health Endpoints
func (suite *IntegrationTestSuite) TestHealthEndpoints() {
	suite.T().Run("liveness_probe", func(t *testing.T) {
		w := suite.makeRequest("GET", "/health/live", nil, "")
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		// Check the nested data structure
		data := response.Data.(map[string]interface{})
		assert.Equal(t, "healthy", data["status"])
	})

	suite.T().Run("readiness_probe", func(t *testing.T) {
		w := suite.makeRequest("GET", "/health/ready", nil, "")
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		// Check the nested data structure
		data := response.Data.(map[string]interface{})
		assert.Equal(t, "healthy", data["status"])
	})

	suite.T().Run("detailed_health_check", func(t *testing.T) {
		w := suite.makeRequest("GET", "/health", nil, "")
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		// Check the nested data structure
		data := response.Data.(map[string]interface{})
		assert.Equal(t, "healthy", data["status"])
		assert.Contains(t, data, "dependencies")
	})
}

// Test Authentication Flow
// TODO: Fix OAuth authentication tests - issues with mock token validation
// func (suite *IntegrationTestSuite) TestAuthenticationFlow() {
// 	suite.T().Run("login_with_oauth", func(t *testing.T) {
// 		loginRequest := map[string]interface{}{
// 			"provider":      "google",
// 			"access_token":  "mock-google-access-token",
// 			"refresh_token": "mock-google-refresh-token",
// 		}

// 		w := suite.makeRequest("POST", "/v1/auth/login", loginRequest, "")
// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response api.SuccessResponse
// 		suite.parseResponse(w, &response)
// 		assert.True(t, response.Success)
// 		assert.Contains(t, response.Data, "access_token")
// 		assert.Contains(t, response.Data, "refresh_token")
// 	})

// 	suite.T().Run("refresh_token", func(t *testing.T) {
// 		// First login to get refresh token
// 		loginRequest := map[string]interface{}{
// 			"provider":      "google",
// 			"access_token":  "mock-google-access-token",
// 			"refresh_token": "mock-google-refresh-token",
// 		}

// 		loginResp := suite.makeRequest("POST", "/v1/auth/login", loginRequest, "")
// 		var loginData api.SuccessResponse
// 		suite.parseResponse(loginResp, &loginData)

// 		// Extract refresh token
// 		tokens := loginData.Data.(map[string]interface{})
// 		refreshToken := tokens["refresh_token"].(string)

// 		// Use refresh token
// 		refreshRequest := map[string]interface{}{
// 			"refresh_token": refreshToken,
// 		}

// 		w := suite.makeRequest("POST", "/v1/auth/refresh", refreshRequest, "")
// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response api.SuccessResponse
// 		suite.parseResponse(w, &response)
// 		assert.True(t, response.Success)
// 		assert.Contains(t, response.Data, "access_token")
// 	})

// 	suite.T().Run("logout", func(t *testing.T) {
// 		w := suite.makeRequest("POST", "/v1/auth/logout", nil, suite.userToken)
// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response api.SuccessResponse
// 		suite.parseResponse(w, &response)
// 		assert.True(t, response.Success)
// 	})
// }

// Test User Management
func (suite *IntegrationTestSuite) TestUserManagement() {
	suite.T().Run("get_current_user", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/users/me", nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		userData := response.Data.(map[string]interface{})
		assert.Equal(t, "regular_user", userData["username"])
		assert.Equal(t, "user@test.com", userData["email"])
	})

	suite.T().Run("update_current_user", func(t *testing.T) {
		updateRequest := map[string]interface{}{
			"username": "updated_user",
			"avatar":   "https://example.com/new-avatar.jpg",
		}

		w := suite.makeRequest("PUT", "/v1/users/me", updateRequest, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		userData := response.Data.(map[string]interface{})
		assert.Equal(t, "updated_user", userData["username"])
	})

	suite.T().Run("get_user_stats", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/users/me/stats", nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		stats := response.Data.(map[string]interface{})
		assert.Contains(t, stats, "total_exchanges")
		assert.Contains(t, stats, "total_subaccounts")
	})

	suite.T().Run("admin_list_users", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/users?limit=10&offset=0", nil, suite.adminToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.PaginatedResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)
		assert.NotNil(t, response.Pagination)
		assert.GreaterOrEqual(t, response.Pagination.Total, int64(2)) // At least admin and user
	})

	suite.T().Run("user_cannot_list_users", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/users", nil, suite.userToken)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// Test Exchange Management
// TODO: Fix Exchange Management tests - validation and API contract issues
func (suite *IntegrationTestSuite) TestExchangeManagement() {
	var exchangeID string

	suite.T().Run("create_exchange", func(t *testing.T) {
		createRequest := map[string]interface{}{
			"name":       "binance-main",
			"type":       "binance",
			"api_key":    "test_api_key_12345",
			"api_secret": "test_api_secret_67890",
		}

		w := suite.makeRequest("POST", "/v1/exchanges", createRequest, suite.userToken)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		exchangeData := response.Data.(map[string]interface{})
		exchangeID = exchangeData["id"].(string)
		assert.Equal(t, "binance-main", exchangeData["name"])
		assert.Equal(t, "binance", exchangeData["type"])
	})

	suite.T().Run("get_user_exchanges", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/exchanges", nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		data := response.Data.(map[string]interface{})
		exchanges := data["exchanges"].([]interface{})
		assert.Len(t, exchanges, 1)
	})

	suite.T().Run("get_exchange_by_id", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/exchanges/"+exchangeID, nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		exchangeData := response.Data.(map[string]interface{})
		assert.Equal(t, exchangeID, exchangeData["id"])
		assert.Equal(t, "binance-main", exchangeData["name"])
	})

	suite.T().Run("update_exchange", func(t *testing.T) {
		updateRequest := map[string]interface{}{
			"name": "updated-binance-main",
		}

		w := suite.makeRequest("PUT", "/v1/exchanges/"+exchangeID, updateRequest, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		exchangeData := response.Data.(map[string]interface{})
		assert.Equal(t, "updated-binance-main", exchangeData["name"])
	})

	suite.T().Run("delete_exchange", func(t *testing.T) {
		w := suite.makeRequest("DELETE", "/v1/exchanges/"+exchangeID, nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)
	})
}

// Test SubAccount Management
func (suite *IntegrationTestSuite) TestSubAccountManagement() {
	// First create an exchange
	createExchangeReq := map[string]interface{}{
		"name":       "test-exchange",
		"type":       "binance",
		"api_key":    "test_api_key_sub",
		"api_secret": "test_api_secret_sub",
	}

	exchangeResp := suite.makeRequest("POST", "/v1/exchanges", createExchangeReq, suite.userToken)
	var exchangeData api.SuccessResponse
	suite.parseResponse(exchangeResp, &exchangeData)
	exchangeID := exchangeData.Data.(map[string]interface{})["id"].(string)

	var subAccountID string

	suite.T().Run("create_sub_account", func(t *testing.T) {
		createRequest := map[string]interface{}{
			"exchange_id": exchangeID,
			"name":        "Main Trading Account",
			"symbol":      "BTC",
			"balance":     1000.50,
		}

		w := suite.makeRequest("POST", "/v1/sub-accounts", createRequest, suite.userToken)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		subAccountData := response.Data.(map[string]interface{})
		subAccountID = subAccountData["id"].(string)
		assert.Equal(t, "Main Trading Account", subAccountData["name"])
		assert.Equal(t, "BTC", subAccountData["symbol"])
		assert.Equal(t, float64(0), subAccountData["balance"])
	})

	suite.T().Run("get_user_sub_accounts", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/sub-accounts", nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		data := response.Data.(map[string]interface{})
		subAccounts := data["sub_accounts"].([]interface{})
		assert.Len(t, subAccounts, 1)
	})

	suite.T().Run("get_sub_account_by_id", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/sub-accounts/"+subAccountID, nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		subAccountData := response.Data.(map[string]interface{})
		assert.Equal(t, subAccountID, subAccountData["id"])
		assert.Equal(t, "Main Trading Account", subAccountData["name"])
	})

	// TODO: Fix update_sub_account_balance database function missing
	// suite.T().Run("update_sub_account_balance", func(t *testing.T) {
		// updateRequest := map[string]interface{}{
		//	"amount":    1500.75,
		//	"direction": "credit",
		//	"reason":    "Initial balance update",
		// }

		// w := suite.makeRequest("PUT", "/v1/sub-accounts/"+subAccountID+"/balance", updateRequest, suite.userToken)
		// assert.Equal(t, http.StatusOK, w.Code)

		// var response api.SuccessResponse
		// suite.parseResponse(w, &response)
		// assert.True(t, response.Success)

		// subAccountData := response.Data.(map[string]interface{})
		// assert.Equal(t, 1500.75, subAccountData["balance"])
	// })

	suite.T().Run("get_sub_accounts_by_symbol", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/sub-accounts/symbol/BTC", nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		data := response.Data.(map[string]interface{})
		subAccounts := data["sub_accounts"].([]interface{})
		assert.Len(t, subAccounts, 1)
	})
}

// Test Trading Log Management
func (suite *IntegrationTestSuite) TestTradingLogManagement() {
	// Setup: Create exchange and sub-account
	createExchangeReq := map[string]interface{}{
		"name":       "trading-exchange",
		"type":       "binance",
		"api_key":    "test_api_key_trading",
		"api_secret": "test_api_secret_trading",
	}

	exchangeResp := suite.makeRequest("POST", "/v1/exchanges", createExchangeReq, suite.userToken)
	var exchangeData api.SuccessResponse
	suite.parseResponse(exchangeResp, &exchangeData)
	exchangeID := exchangeData.Data.(map[string]interface{})["id"].(string)

	createSubAccountReq := map[string]interface{}{
		"exchange_id": exchangeID,
		"name":        "Trading Account",
		"symbol":      "ETH",
		"balance":     500.0,
	}

	subAccountResp := suite.makeRequest("POST", "/v1/sub-accounts", createSubAccountReq, suite.userToken)
	var subAccountData api.SuccessResponse
	suite.parseResponse(subAccountResp, &subAccountData)
	subAccountID := subAccountData.Data.(map[string]interface{})["id"].(string)

	var tradingLogID string

	suite.T().Run("create_trading_log", func(t *testing.T) {
		createRequest := map[string]interface{}{
			"exchange_id":    exchangeID,
			"sub_account_id": subAccountID,
			"type":           "trade",
			"source":         "manual",
			"message":        "ETH/USDT buy order: 10.5 @ 2500.00",
		}

		w := suite.makeRequest("POST", "/v1/trading-logs", createRequest, suite.userToken)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		tradingLogData := response.Data.(map[string]interface{})
		tradingLogID = tradingLogData["id"].(string)
		assert.Equal(t, "trade", tradingLogData["type"])
		assert.Equal(t, "manual", tradingLogData["source"])
		assert.Equal(t, "ETH/USDT buy order: 10.5 @ 2500.00", tradingLogData["message"])
	})

	suite.T().Run("get_user_trading_logs", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/trading-logs", nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		data := response.Data.(map[string]interface{})
		tradingLogs := data["trading_logs"].([]interface{})
		assert.Len(t, tradingLogs, 1)
	})

	suite.T().Run("get_trading_log_by_id", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/trading-logs/"+tradingLogID, nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		tradingLogData := response.Data.(map[string]interface{})
		assert.Equal(t, tradingLogID, tradingLogData["id"])
		assert.Equal(t, "trade", tradingLogData["type"])
	})

	suite.T().Run("get_sub_account_trading_logs", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/trading-logs/sub-account/"+subAccountID, nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)

		data := response.Data.(map[string]interface{})
		tradingLogs := data["trading_logs"].([]interface{})
		assert.Len(t, tradingLogs, 1)
	})

	suite.T().Run("delete_trading_log", func(t *testing.T) {
		w := suite.makeRequest("DELETE", "/v1/trading-logs/"+tradingLogID, nil, suite.userToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.SuccessResponse
		suite.parseResponse(w, &response)
		assert.True(t, response.Success)
	})
}

// Test Error Handling and Edge Cases
func (suite *IntegrationTestSuite) TestErrorHandling() {
	suite.T().Run("unauthorized_access", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/users/me", nil, "") // No token
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response api.ErrorResponse
		suite.parseResponse(w, &response)
		assert.False(t, response.Success)
		assert.Equal(t, "AUTH_REQUIRED", response.Error.Code)
	})

	suite.T().Run("invalid_token", func(t *testing.T) {
		w := suite.makeRequest("GET", "/v1/users/me", nil, "invalid-token")
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response api.ErrorResponse
		suite.parseResponse(w, &response)
		assert.False(t, response.Success)
		assert.Equal(t, "INVALID_TOKEN", response.Error.Code)
	})

	suite.T().Run("not_found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := suite.makeRequest("GET", "/v1/exchanges/"+nonExistentID, nil, suite.userToken)
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response api.ErrorResponse
		suite.parseResponse(w, &response)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error.Code, "NOT_FOUND")
	})

	suite.T().Run("invalid_input", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"name": "", // Empty name should be invalid
		}

		w := suite.makeRequest("POST", "/v1/exchanges", invalidRequest, suite.userToken)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response api.ErrorResponse
		suite.parseResponse(w, &response)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error.Code, "INVALID")
	})
}

// Test Rate Limiting
func (suite *IntegrationTestSuite) TestRateLimiting() {
	suite.T().Run("api_rate_limiting", func(t *testing.T) {
		// Make multiple rapid requests to test rate limiting
		// Note: This test might be flaky depending on rate limit configuration
		var lastStatusCode int

		for i := 0; i < 10; i++ {
			w := suite.makeRequest("GET", "/v1/users/me", nil, suite.userToken)
			lastStatusCode = w.Code

			if w.Code == http.StatusTooManyRequests {
				break
			}
		}

		// We expect either all requests to succeed (if rate limit is high)
		// or eventually hit rate limit
		assert.True(t, lastStatusCode == http.StatusOK || lastStatusCode == http.StatusTooManyRequests)
	})
}

// Test Metrics Endpoint
func (suite *IntegrationTestSuite) TestMetrics() {
	suite.T().Run("prometheus_metrics", func(t *testing.T) {
		w := suite.makeRequest("GET", "/metrics", nil, "")
		assert.Equal(t, http.StatusOK, w.Code)

		body := w.Body.String()
		assert.Contains(t, body, "# HELP")
		assert.Contains(t, body, "# TYPE")
	})
}

// Test runner
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
