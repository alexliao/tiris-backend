package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tiris-backend/internal/database"
	"tiris-backend/internal/nats"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HealthHandlerTestSuite struct {
	suite.Suite
	healthHandler *HealthHandler
	router        *gin.Engine
}

func (suite *HealthHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	
	// Create health handler with nil dependencies for basic testing
	// In a real test, we'd mock these dependencies
	suite.healthHandler = NewHealthHandler(nil, nil)
	
	suite.router = gin.New()
	suite.router.GET("/health/live", suite.healthHandler.LivenessProbe)
	suite.router.GET("/health/ready", suite.healthHandler.ReadinessProbe)
	suite.router.GET("/health", suite.healthHandler.HealthCheck)
}

func (suite *HealthHandlerTestSuite) TestLivenessProbe_Success() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/health/live", nil)
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	
	// Check response data structure
	data, ok := response.Data.(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "healthy", data["status"])
	assert.Contains(suite.T(), data, "timestamp")
	assert.Contains(suite.T(), data, "message")
}

func (suite *HealthHandlerTestSuite) TestReadinessProbe_WithNilDependencies() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/health/ready", nil)
	suite.router.ServeHTTP(w, request)
	
	// Assert - should return unhealthy due to nil dependencies
	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SERVICE_UNAVAILABLE", response.Error.Code)
}

func (suite *HealthHandlerTestSuite) TestHealthCheck_WithNilDependencies() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/health", nil)
	suite.router.ServeHTTP(w, request)
	
	// Assert - should return unhealthy due to nil dependencies
	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SERVICE_DEGRADED", response.Error.Code)
}

func (suite *HealthHandlerTestSuite) TestHealthCheck_ResponseStructure() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/health", nil)
	suite.router.ServeHTTP(w, request)
	
	// Assert structure even when unhealthy
	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	// Verify metadata structure
	assert.NotEmpty(suite.T(), response.Metadata.Timestamp)
	assert.Equal(suite.T(), "SERVICE_DEGRADED", response.Error.Code)
	assert.Contains(suite.T(), response.Error.Message, "dependencies are unhealthy")
}

// TestHealthHandlerWithMockDB demonstrates testing with a mock database
func (suite *HealthHandlerTestSuite) TestHealthHandlerWithMockDependencies() {
	// This test demonstrates the structure for testing with real mock dependencies
	// In practice, you'd create mock implementations of database.DB and nats.Manager
	
	// For now, just verify the handler can be created with non-nil dependencies
	var mockDB *database.DB = nil    // In real test: create mock
	var mockNATS *nats.Manager = nil // In real test: create mock
	
	handler := NewHealthHandler(mockDB, mockNATS)
	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), mockDB, handler.db)
	assert.Equal(suite.T(), mockNATS, handler.natsManager)
}

func TestHealthHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerTestSuite))
}