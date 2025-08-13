package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tiris-backend/internal/api"
	"tiris-backend/internal/middleware"
	"tiris-backend/pkg/security"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// APIIntegrationTestSuite tests the complete API functionality
type APIIntegrationTestSuite struct {
	IntegrationTestSuite
	Router *gin.Engine
	Server *httptest.Server
}

// SetupSuite initializes the API test environment
func (s *APIIntegrationTestSuite) SetupSuite() {
	s.IntegrationTestSuite.SetupSuite()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router with all middleware and handlers
	s.Router = s.setupRouter()

	// Create test server
	s.Server = httptest.NewServer(s.Router)
}

// TearDownSuite cleans up the API test environment
func (s *APIIntegrationTestSuite) TearDownSuite() {
	if s.Server != nil {
		s.Server.Close()
	}
	s.IntegrationTestSuite.TearDownSuite()
}

func (s *APIIntegrationTestSuite) setupRouter() *gin.Engine {
	router := gin.New()

	// Create security middleware
	securityMiddleware, err := middleware.NewSecurityMiddleware(
		s.DB,
		s.Redis,
		s.Config.Security.MasterKey,
		s.Config.Security.SigningKey,
	)
	s.Require().NoError(err, "Failed to create security middleware")

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(securityMiddleware.SecurityHeadersMiddleware())
	router.Use(securityMiddleware.AuditMiddleware())

	// Add CORS middleware
	corsOrigins := []string{"http://localhost:3000", "https://tiris.ai"}
	router.Use(middleware.CORSMiddleware(corsOrigins))

	// Create API handlers
	authHandler := api.NewAuthHandler(s.JWTManager, s.UserService)
	userHandler := api.NewUserHandler(s.UserService)
	exchangeHandler := api.NewExchangeHandler(s.ExchangeService)

	// Health check routes (no auth required)
	healthGroup := router.Group("/health")
	{
		healthGroup.GET("/live", api.NewHealthHandler().LivenessProbe)
		healthGroup.GET("/ready", api.NewHealthHandler().ReadinessProbe)
	}

	// Auth routes (no auth required)
	authGroup := router.Group("/api/v1/auth")
	authGroup.Use(securityMiddleware.RateLimitMiddleware("auth_login"))
	{
		authGroup.POST("/google", authHandler.GoogleAuth)
		authGroup.POST("/wechat", authHandler.WeChatAuth)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// Protected API routes
	apiGroup := router.Group("/api/v1")
	
	// JWT authentication for regular API
	apiGroup.Use(middleware.AuthMiddleware(s.JWTManager))
	apiGroup.Use(securityMiddleware.RateLimitMiddleware("api_general"))

	// User routes
	usersGroup := apiGroup.Group("/users")
	{
		usersGroup.GET("/profile", userHandler.GetProfile)
		usersGroup.PUT("/profile", userHandler.UpdateProfile)
		usersGroup.DELETE("/account", userHandler.DeleteAccount)
	}

	// Exchange routes
	exchangesGroup := apiGroup.Group("/exchanges")
	{
		exchangesGroup.GET("", exchangeHandler.GetExchanges)
		exchangesGroup.POST("", exchangeHandler.CreateExchange)
		exchangesGroup.GET("/:id", exchangeHandler.GetExchange)
		exchangesGroup.PUT("/:id", exchangeHandler.UpdateExchange)
		exchangesGroup.DELETE("/:id", exchangeHandler.DeleteExchange)
	}

	// API Key protected routes
	apiKeyGroup := router.Group("/api/v1/external")
	apiKeyGroup.Use(securityMiddleware.APIKeyAuthMiddleware())
	apiKeyGroup.Use(securityMiddleware.RateLimitMiddleware("api_external"))
	{
		apiKeyGroup.GET("/user/profile", userHandler.GetProfile)
		apiKeyGroup.GET("/exchanges", exchangeHandler.GetExchanges)
	}

	return router
}

// Test health endpoints
func (s *APIIntegrationTestSuite) TestHealthEndpoints() {
	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "liveness probe",
			endpoint:       "/health/live",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "readiness probe",
			endpoint:       "/health/ready",
			expectedStatus: http.StatusOK,
			expectedBody:   "Ready",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp, err := http.Get(s.Server.URL + tt.endpoint)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.Equal(tt.expectedStatus, resp.StatusCode)

			var body bytes.Buffer
			_, err = body.ReadFrom(resp.Body)
			s.Require().NoError(err)
			s.Contains(body.String(), tt.expectedBody)
		})
	}
}

// Test JWT authentication flow
func (s *APIIntegrationTestSuite) TestJWTAuthenticationFlow() {
	// Create a JWT token
	token, err := s.generateJWTToken(s.TestUser.ID, s.TestUser.Username, s.TestUser.Email)
	s.Require().NoError(err)

	// Test accessing protected endpoint without token
	resp, err := http.Get(s.Server.URL + "/api/v1/users/profile")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusUnauthorized, resp.StatusCode)

	// Test accessing protected endpoint with valid token
	req, err := http.NewRequest("GET", s.Server.URL+"/api/v1/users/profile", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Verify audit log entry was created
	auditEvent := s.waitForAuditEvent(security.ActionUserView, 2*time.Second)
	s.NotNil(auditEvent)
	s.Equal(security.ActionUserView, auditEvent.Action)
	s.Equal(s.TestUser.ID, *auditEvent.UserID)
	s.True(auditEvent.Success)
}

// Test API key authentication flow
func (s *APIIntegrationTestSuite) TestAPIKeyAuthenticationFlow() {
	// Create an API key
	apiKey := s.createUserAPIKey(s.TestUser.ID, "Test API Key")
	s.Require().NotNil(apiKey.PlaintextKey)

	// Test accessing API key endpoint without key
	resp, err := http.Get(s.Server.URL + "/api/v1/external/user/profile")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusUnauthorized, resp.StatusCode)

	// Test accessing API key endpoint with valid key
	req, err := http.NewRequest("GET", s.Server.URL+"/api/v1/external/user/profile", nil)
	s.Require().NoError(err)
	req.Header.Set("X-API-Key", *apiKey.PlaintextKey)

	client := &http.Client{}
	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Verify audit log entry was created
	auditEvent := s.waitForAuditEvent(security.ActionAPIKeyUsed, 2*time.Second)
	s.NotNil(auditEvent)
	s.Equal(security.ActionAPIKeyUsed, auditEvent.Action)
	s.True(auditEvent.Success)
}

// Test rate limiting functionality
func (s *APIIntegrationTestSuite) TestRateLimiting() {
	token, err := s.generateJWTToken(s.TestUser.ID, s.TestUser.Username, s.TestUser.Email)
	s.Require().NoError(err)

	// Get the rate limit for general API (1000 requests per hour)
	rules := security.DefaultRules()
	rule := rules["api_general"]

	client := &http.Client{}

	// Make requests up to the limit
	for i := 0; i < rule.Limit; i++ {
		req, err := http.NewRequest("GET", s.Server.URL+"/api/v1/users/profile", nil)
		s.Require().NoError(err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		s.Require().NoError(err)
		resp.Body.Close()

		if i < rule.Limit {
			s.Equal(http.StatusOK, resp.StatusCode, "Request %d should succeed", i)
		}
	}

	// The next request should be rate limited
	req, err := http.NewRequest("GET", s.Server.URL+"/api/v1/users/profile", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusTooManyRequests, resp.StatusCode, "Request should be rate limited")

	// Verify rate limit headers are present
	s.NotEmpty(resp.Header.Get("X-RateLimit-Limit"))
	s.NotEmpty(resp.Header.Get("X-RateLimit-Remaining"))
	s.NotEmpty(resp.Header.Get("Retry-After"))

	// Verify rate limit audit event
	auditEvent := s.waitForAuditEvent(security.ActionRateLimitHit, 2*time.Second)
	s.NotNil(auditEvent)
	s.Equal(security.ActionRateLimitHit, auditEvent.Action)
}

// Test security headers
func (s *APIIntegrationTestSuite) TestSecurityHeaders() {
	resp, err := http.Get(s.Server.URL + "/health/live")
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":   "nosniff",
		"X-Frame-Options":          "DENY",
		"X-XSS-Protection":         "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
		"Referrer-Policy":          "strict-origin-when-cross-origin",
		"Content-Security-Policy":  "default-src 'self'",
		"Server":                   "Tiris-Backend/1.0",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		s.NotEmpty(actualValue, "Header %s should be present", header)
		s.Contains(actualValue, expectedValue, "Header %s should contain %s", header, expectedValue)
	}
}

// Test CORS functionality
func (s *APIIntegrationTestSuite) TestCORS() {
	// Test preflight request
	req, err := http.NewRequest("OPTIONS", s.Server.URL+"/api/v1/users/profile", nil)
	s.Require().NoError(err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)
	s.Equal("http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	s.Equal("true", resp.Header.Get("Access-Control-Allow-Credentials"))
	s.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "GET")
}

// Test threat detection
func (s *APIIntegrationTestSuite) TestThreatDetection() {
	// Test with malicious user agent
	req, err := http.NewRequest("GET", s.Server.URL+"/health/live", nil)
	s.Require().NoError(err)
	req.Header.Set("User-Agent", "sqlmap/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should be blocked due to malicious user agent
	s.Equal(http.StatusForbidden, resp.StatusCode)

	// Verify security alert was logged
	auditEvent := s.waitForAuditEvent(security.ActionSecurityAlert, 2*time.Second)
	s.NotNil(auditEvent)
	s.Equal(security.ActionSecurityAlert, auditEvent.Action)
	s.Contains(auditEvent.Details["alert_type"], "malicious_user_agent")
}

// Test complete user workflow
func (s *APIIntegrationTestSuite) TestCompleteUserWorkflow() {
	token, err := s.generateJWTToken(s.TestUser.ID, s.TestUser.Username, s.TestUser.Email)
	s.Require().NoError(err)

	client := &http.Client{}

	// 1. Get user profile
	req, err := http.NewRequest("GET", s.Server.URL+"/api/v1/users/profile", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// 2. Update user profile
	updateData := map[string]interface{}{
		"username": "updated_username",
		"settings": map[string]interface{}{
			"theme": "dark",
		},
	}
	jsonData, _ := json.Marshal(updateData)

	req, err = http.NewRequest("PUT", s.Server.URL+"/api/v1/users/profile", bytes.NewBuffer(jsonData))
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// 3. Create an exchange
	exchangeData := map[string]interface{}{
		"name":       "Test Binance",
		"type":       "binance",
		"api_key":    "test_binance_key",
		"api_secret": "test_binance_secret",
	}
	jsonData, _ = json.Marshal(exchangeData)

	req, err = http.NewRequest("POST", s.Server.URL+"/api/v1/exchanges", bytes.NewBuffer(jsonData))
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)

	// Parse response to get exchange ID
	var exchangeResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&exchangeResp)
	s.Require().NoError(err)

	data, ok := exchangeResp["data"].(map[string]interface{})
	s.Require().True(ok)
	exchangeID := data["id"].(string)

	// 4. Get exchanges list
	req, err = http.NewRequest("GET", s.Server.URL+"/api/v1/exchanges", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// 5. Get specific exchange
	req, err = http.NewRequest("GET", s.Server.URL+"/api/v1/exchanges/"+exchangeID, nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Verify all audit events were created
	time.Sleep(500 * time.Millisecond) // Allow time for async audit logging

	var auditCount int64
	err = s.DB.Model(&security.AuditEvent{}).
		Where("user_id = ?", s.TestUser.ID).
		Count(&auditCount).Error
	s.Require().NoError(err)
	s.Greater(auditCount, int64(4), "Should have multiple audit events")
}

// Test error handling and edge cases
func (s *APIIntegrationTestSuite) TestErrorHandling() {
	token, err := s.generateJWTToken(s.TestUser.ID, s.TestUser.Username, s.TestUser.Email)
	s.Require().NoError(err)

	client := &http.Client{}

	// Test invalid JSON
	invalidJSON := `{"invalid": json}`
	req, err := http.NewRequest("PUT", s.Server.URL+"/api/v1/users/profile", strings.NewReader(invalidJSON))
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)

	// Test accessing non-existent resource
	req, err = http.NewRequest("GET", s.Server.URL+"/api/v1/exchanges/"+uuid.New().String(), nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNotFound, resp.StatusCode)

	// Test method not allowed
	req, err = http.NewRequest("PATCH", s.Server.URL+"/api/v1/users/profile", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusMethodNotAllowed, resp.StatusCode)
}

// Run the test suite
func TestAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}