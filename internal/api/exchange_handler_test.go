package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tiris-backend/internal/services"
	"tiris-backend/test/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ExchangeHandlerTestSuite struct {
	suite.Suite
	handler        *ExchangeHandler
	mockService    *mocks.MockExchangeService
	router         *gin.Engine
	testUserID     uuid.UUID
	testExchangeID uuid.UUID
	authMiddleware gin.HandlerFunc
}

func (suite *ExchangeHandlerTestSuite) SetupTest() {
	// Create test UUIDs
	suite.testUserID = uuid.New()
	suite.testExchangeID = uuid.New()

	// Create mock service
	suite.mockService = &mocks.MockExchangeService{}

	// Create handler
	suite.handler = NewExchangeHandler(suite.mockService)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test router
	suite.router = gin.New()

	// Create auth middleware for tests
	suite.authMiddleware = func(c *gin.Context) {
		c.Set("user_id", suite.testUserID)
		c.Next()
	}

	// Set up routes
	v1 := suite.router.Group("/api/v1")
	{
		exchanges := v1.Group("/exchanges")
		exchanges.Use(suite.authMiddleware)
		{
			exchanges.POST("", suite.handler.CreateExchange)
			exchanges.GET("", suite.handler.GetUserExchanges)
			exchanges.GET("/:id", suite.handler.GetExchange)
			exchanges.PUT("/:id", suite.handler.UpdateExchange)
			exchanges.DELETE("/:id", suite.handler.DeleteExchange)
		}

		admin := v1.Group("/admin/exchanges")
		admin.Use(suite.authMiddleware)
		{
			admin.GET("", suite.handler.ListExchanges)
			admin.GET("/:id", suite.handler.GetExchangeByID)
		}
	}
}

func (suite *ExchangeHandlerTestSuite) TestCreateExchange_Success() {
	// Arrange
	req := services.CreateExchangeRequest{
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
	}

	expectedExchange := &services.ExchangeResponse{
		ID:        suite.testExchangeID,
		UserID:    suite.testUserID,
		Name:      req.Name,
		Type:      req.Type,
		APIKey:    "test***key", // Masked
		Status:    "active",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("CreateExchange", mock.Anything, suite.testUserID, &req).
		Return(expectedExchange, nil)

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/exchanges", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestCreateExchange_InvalidRequest() {
	// Arrange
	invalidReq := map[string]interface{}{
		"name": "", // Invalid empty name
	}

	body, _ := json.Marshal(invalidReq)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/exchanges", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INVALID_REQUEST", response.Error.Code)
}

func (suite *ExchangeHandlerTestSuite) TestCreateExchange_DuplicateName() {
	// Arrange
	req := services.CreateExchangeRequest{
		Name:      "Duplicate Exchange",
		Type:      "binance",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
	}

	suite.mockService.On("CreateExchange", mock.Anything, suite.testUserID, &req).
		Return(nil, fmt.Errorf("exchange name already exists"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/exchanges", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "EXCHANGE_NAME_EXISTS", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestCreateExchange_LimitReached() {
	// Arrange
	req := services.CreateExchangeRequest{
		Name:      "Limit Test Exchange",
		Type:      "binance",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
	}

	suite.mockService.On("CreateExchange", mock.Anything, suite.testUserID, &req).
		Return(nil, fmt.Errorf("maximum number of exchanges reached (10)"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/exchanges", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "EXCHANGE_LIMIT_REACHED", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestCreateExchange_Unauthorized() {
	// Arrange - router without auth middleware
	router := gin.New()
	router.POST("/exchanges", suite.handler.CreateExchange)

	req := services.CreateExchangeRequest{
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
	}

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/exchanges", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "AUTH_REQUIRED", response.Error.Code)
}

func (suite *ExchangeHandlerTestSuite) TestGetUserExchanges_Success() {
	// Arrange
	exchanges := []*services.ExchangeResponse{
		{
			ID:        suite.testExchangeID,
			UserID:    suite.testUserID,
			Name:      "Exchange 1",
			Type:      "binance",
			Status:    "active",
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID:        uuid.New(),
			UserID:    suite.testUserID,
			Name:      "Exchange 2",
			Type:      "coinbase",
			Status:    "active",
			CreatedAt: time.Now().Format(time.RFC3339),
		},
	}

	suite.mockService.On("GetUserExchanges", mock.Anything, suite.testUserID).
		Return(exchanges, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/exchanges", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	data := response.Data.(map[string]interface{})
	exchangesData := data["exchanges"].([]interface{})
	assert.Len(suite.T(), exchangesData, 2)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestGetExchange_Success() {
	// Arrange
	expectedExchange := &services.ExchangeResponse{
		ID:        suite.testExchangeID,
		UserID:    suite.testUserID,
		Name:      "Test Exchange",
		Type:      "binance",
		Status:    "active",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("GetExchange", mock.Anything, suite.testUserID, suite.testExchangeID).
		Return(expectedExchange, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/exchanges/%s", suite.testExchangeID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestGetExchange_InvalidID() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/exchanges/invalid-uuid", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INVALID_EXCHANGE_ID", response.Error.Code)
}

func (suite *ExchangeHandlerTestSuite) TestGetExchange_NotFound() {
	// Arrange
	suite.mockService.On("GetExchange", mock.Anything, suite.testUserID, suite.testExchangeID).
		Return(nil, fmt.Errorf("exchange not found"))

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/exchanges/%s", suite.testExchangeID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "EXCHANGE_NOT_FOUND", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestUpdateExchange_Success() {
	// Arrange
	name := "Updated Exchange Name"
	apiKey := "new_api_key"
	apiSecret := "new_api_secret"
	req := services.UpdateExchangeRequest{
		Name:      &name,
		APIKey:    &apiKey,
		APISecret: &apiSecret,
	}

	expectedExchange := &services.ExchangeResponse{
		ID:        suite.testExchangeID,
		UserID:    suite.testUserID,
		Name:      *req.Name,
		Type:      "binance",
		Status:    "active",
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("UpdateExchange", mock.Anything, suite.testUserID, suite.testExchangeID, &req).
		Return(expectedExchange, nil)

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/exchanges/%s", suite.testExchangeID), bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestUpdateExchange_NotFound() {
	// Arrange
	name := "Updated Exchange Name"
	req := services.UpdateExchangeRequest{
		Name: &name,
	}

	suite.mockService.On("UpdateExchange", mock.Anything, suite.testUserID, suite.testExchangeID, &req).
		Return(nil, fmt.Errorf("exchange not found"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/exchanges/%s", suite.testExchangeID), bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "EXCHANGE_NOT_FOUND", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestDeleteExchange_Success() {
	// Arrange
	suite.mockService.On("DeleteExchange", mock.Anything, suite.testUserID, suite.testExchangeID).
		Return(nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/exchanges/%s", suite.testExchangeID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	data := response.Data.(map[string]interface{})
	assert.Equal(suite.T(), "Exchange deleted successfully", data["message"])

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestDeleteExchange_HasSubAccounts() {
	// Arrange
	suite.mockService.On("DeleteExchange", mock.Anything, suite.testUserID, suite.testExchangeID).
		Return(fmt.Errorf("cannot delete exchange with existing sub-accounts"))

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/exchanges/%s", suite.testExchangeID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "EXCHANGE_HAS_SUBACCOUNTS", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestListExchanges_Success() {
	// Arrange
	exchanges := []*services.ExchangeResponse{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Name:      "Admin Exchange 1",
			Type:      "binance",
			Status:    "active",
			CreatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Name:      "Admin Exchange 2",
			Type:      "coinbase",
			Status:    "active",
			CreatedAt: time.Now().Format(time.RFC3339),
		},
	}

	suite.mockService.On("ListExchanges", mock.Anything, 100, 0).
		Return(exchanges, int64(50), nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/admin/exchanges", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Pagination)
	assert.Equal(suite.T(), int64(50), response.Pagination.Total)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestListExchanges_WithPagination() {
	// Arrange
	exchanges := []*services.ExchangeResponse{
		{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Name:   "Exchange 1",
			Type:   "binance",
		},
	}

	suite.mockService.On("ListExchanges", mock.Anything, 10, 20).
		Return(exchanges, int64(100), nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/admin/exchanges?limit=10&offset=20", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), int64(100), response.Pagination.Total)
	assert.Equal(suite.T(), 10, response.Pagination.Limit)
	assert.Equal(suite.T(), 20, response.Pagination.Offset)
	assert.True(suite.T(), response.Pagination.HasMore)
	assert.Equal(suite.T(), 30, *response.Pagination.NextOffset)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestGetExchangeByID_Success() {
	// Arrange
	expectedExchange := &services.ExchangeResponse{
		ID:        suite.testExchangeID,
		UserID:    uuid.New(),
		Name:      "Admin Exchange",
		Type:      "binance",
		Status:    "active",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("GetExchangeByID", mock.Anything, suite.testExchangeID).
		Return(expectedExchange, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/admin/exchanges/%s", suite.testExchangeID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ExchangeHandlerTestSuite) TestCreateExchange_ServiceError() {
	// Arrange
	req := services.CreateExchangeRequest{
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
	}

	suite.mockService.On("CreateExchange", mock.Anything, suite.testUserID, &req).
		Return(nil, fmt.Errorf("database connection failed"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/exchanges", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "EXCHANGE_CREATE_FAILED", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func TestExchangeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ExchangeHandlerTestSuite))
}