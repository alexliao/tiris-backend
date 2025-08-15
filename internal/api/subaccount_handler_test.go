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

type SubAccountHandlerTestSuite struct {
	suite.Suite
	handler           *SubAccountHandler
	mockService       *mocks.MockSubAccountService
	router            *gin.Engine
	testUserID        uuid.UUID
	testExchangeID    uuid.UUID
	testSubAccountID  uuid.UUID
	authMiddleware    gin.HandlerFunc
}

func (suite *SubAccountHandlerTestSuite) SetupTest() {
	// Create test UUIDs
	suite.testUserID = uuid.New()
	suite.testExchangeID = uuid.New()
	suite.testSubAccountID = uuid.New()

	// Create mock service
	suite.mockService = &mocks.MockSubAccountService{}

	// Create handler
	suite.handler = NewSubAccountHandler(suite.mockService)

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
		subAccounts := v1.Group("/sub-accounts")
		subAccounts.Use(suite.authMiddleware)
		{
			subAccounts.POST("", suite.handler.CreateSubAccount)
			subAccounts.GET("", suite.handler.GetUserSubAccounts)
			subAccounts.GET("/:id", suite.handler.GetSubAccount)
			subAccounts.PUT("/:id", suite.handler.UpdateSubAccount)
			subAccounts.PUT("/:id/balance", suite.handler.UpdateBalance)
			subAccounts.DELETE("/:id", suite.handler.DeleteSubAccount)
			subAccounts.GET("/symbol/:symbol", suite.handler.GetSubAccountsBySymbol)
		}
	}
}

func (suite *SubAccountHandlerTestSuite) TestCreateSubAccount_Success() {
	// Arrange
	req := services.CreateSubAccountRequest{
		ExchangeID: suite.testExchangeID,
		Name:       "Test Sub Account",
		Symbol:     "BTC",
	}

	expectedResponse := &services.SubAccountResponse{
		ID:         suite.testSubAccountID,
		UserID:     suite.testUserID,
		ExchangeID: suite.testExchangeID,
		Name:       req.Name,
		Symbol:     req.Symbol,
		Balance:    0.0,
		Info:       map[string]interface{}{},
		CreatedAt:  time.Now().Format(time.RFC3339),
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("CreateSubAccount", mock.Anything, suite.testUserID, &req).
		Return(expectedResponse, nil)

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/sub-accounts", bytes.NewBuffer(body))
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

func (suite *SubAccountHandlerTestSuite) TestCreateSubAccount_InvalidRequest() {
	// Arrange
	invalidReq := map[string]interface{}{
		"name": "", // Invalid empty name
	}

	body, _ := json.Marshal(invalidReq)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/sub-accounts", bytes.NewBuffer(body))
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

func (suite *SubAccountHandlerTestSuite) TestCreateSubAccount_ExchangeNotFound() {
	// Arrange
	req := services.CreateSubAccountRequest{
		ExchangeID: suite.testExchangeID,
		Name:       "Test Sub Account",
		Symbol:     "BTC",
	}

	suite.mockService.On("CreateSubAccount", mock.Anything, suite.testUserID, &req).
		Return(nil, fmt.Errorf("exchange not found"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/sub-accounts", bytes.NewBuffer(body))
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

func (suite *SubAccountHandlerTestSuite) TestCreateSubAccount_NameExists() {
	// Arrange
	req := services.CreateSubAccountRequest{
		ExchangeID: suite.testExchangeID,
		Name:       "Duplicate Sub Account",
		Symbol:     "BTC",
	}

	suite.mockService.On("CreateSubAccount", mock.Anything, suite.testUserID, &req).
		Return(nil, fmt.Errorf("sub-account name already exists for this exchange"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/sub-accounts", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SUBACCOUNT_NAME_EXISTS", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestCreateSubAccount_Unauthorized() {
	// Arrange - router without auth middleware
	router := gin.New()
	router.POST("/sub-accounts", suite.handler.CreateSubAccount)

	req := services.CreateSubAccountRequest{
		ExchangeID: suite.testExchangeID,
		Name:       "Test Sub Account",
		Symbol:     "BTC",
	}

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/sub-accounts", bytes.NewBuffer(body))
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

func (suite *SubAccountHandlerTestSuite) TestGetUserSubAccounts_Success() {
	// Arrange
	subAccounts := []*services.SubAccountResponse{
		{
			ID:         suite.testSubAccountID,
			UserID:     suite.testUserID,
			ExchangeID: suite.testExchangeID,
			Name:       "Sub Account 1",
			Symbol:     "BTC",
			Balance:    100.0,
			CreatedAt:  time.Now().Format(time.RFC3339),
		},
		{
			ID:         uuid.New(),
			UserID:     suite.testUserID,
			ExchangeID: suite.testExchangeID,
			Name:       "Sub Account 2",
			Symbol:     "ETH",
			Balance:    50.0,
			CreatedAt:  time.Now().Format(time.RFC3339),
		},
	}

	suite.mockService.On("GetUserSubAccounts", mock.Anything, suite.testUserID, (*uuid.UUID)(nil)).
		Return(subAccounts, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/sub-accounts", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	data := response.Data.(map[string]interface{})
	subAccountsData := data["sub_accounts"].([]interface{})
	assert.Len(suite.T(), subAccountsData, 2)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestGetUserSubAccounts_WithExchangeFilter() {
	// Arrange
	subAccounts := []*services.SubAccountResponse{
		{
			ID:         suite.testSubAccountID,
			UserID:     suite.testUserID,
			ExchangeID: suite.testExchangeID,
			Name:       "Sub Account 1",
			Symbol:     "BTC",
			Balance:    100.0,
			CreatedAt:  time.Now().Format(time.RFC3339),
		},
	}

	suite.mockService.On("GetUserSubAccounts", mock.Anything, suite.testUserID, &suite.testExchangeID).
		Return(subAccounts, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/sub-accounts?exchange_id=%s", suite.testExchangeID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestGetUserSubAccounts_InvalidExchangeID() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/sub-accounts?exchange_id=invalid-uuid", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INVALID_EXCHANGE_ID", response.Error.Code)
}

func (suite *SubAccountHandlerTestSuite) TestGetSubAccount_Success() {
	// Arrange
	expectedSubAccount := &services.SubAccountResponse{
		ID:         suite.testSubAccountID,
		UserID:     suite.testUserID,
		ExchangeID: suite.testExchangeID,
		Name:       "Test Sub Account",
		Symbol:     "BTC",
		Balance:    100.0,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("GetSubAccount", mock.Anything, suite.testUserID, suite.testSubAccountID).
		Return(expectedSubAccount, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/sub-accounts/%s", suite.testSubAccountID), nil)
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

func (suite *SubAccountHandlerTestSuite) TestGetSubAccount_InvalidID() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/sub-accounts/invalid-uuid", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INVALID_SUBACCOUNT_ID", response.Error.Code)
}

func (suite *SubAccountHandlerTestSuite) TestGetSubAccount_NotFound() {
	// Arrange
	suite.mockService.On("GetSubAccount", mock.Anything, suite.testUserID, suite.testSubAccountID).
		Return(nil, fmt.Errorf("sub-account not found"))

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/sub-accounts/%s", suite.testSubAccountID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SUBACCOUNT_NOT_FOUND", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestUpdateSubAccount_Success() {
	// Arrange
	name := "Updated Sub Account"
	req := services.UpdateSubAccountRequest{
		Name: &name,
	}

	expectedResponse := &services.SubAccountResponse{
		ID:         suite.testSubAccountID,
		UserID:     suite.testUserID,
		ExchangeID: suite.testExchangeID,
		Name:       *req.Name,
		Symbol:     "BTC",
		Balance:    100.0,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("UpdateSubAccount", mock.Anything, suite.testUserID, suite.testSubAccountID, &req).
		Return(expectedResponse, nil)

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/sub-accounts/%s", suite.testSubAccountID), bytes.NewBuffer(body))
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

func (suite *SubAccountHandlerTestSuite) TestUpdateSubAccount_NotFound() {
	// Arrange
	name := "Updated Sub Account"
	req := services.UpdateSubAccountRequest{
		Name: &name,
	}

	suite.mockService.On("UpdateSubAccount", mock.Anything, suite.testUserID, suite.testSubAccountID, &req).
		Return(nil, fmt.Errorf("sub-account not found"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/sub-accounts/%s", suite.testSubAccountID), bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SUBACCOUNT_NOT_FOUND", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestUpdateBalance_Success() {
	// Arrange
	req := services.UpdateBalanceRequest{
		Amount:    100.0,
		Direction: "credit",
		Reason:    "deposit",
	}

	expectedResponse := &services.SubAccountResponse{
		ID:         suite.testSubAccountID,
		UserID:     suite.testUserID,
		ExchangeID: suite.testExchangeID,
		Name:       "Test Sub Account",
		Symbol:     "BTC",
		Balance:    200.0, // Updated balance
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("UpdateBalance", mock.Anything, suite.testUserID, suite.testSubAccountID, &req).
		Return(expectedResponse, nil)

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/sub-accounts/%s/balance", suite.testSubAccountID), bytes.NewBuffer(body))
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

func (suite *SubAccountHandlerTestSuite) TestUpdateBalance_InsufficientBalance() {
	// Arrange
	req := services.UpdateBalanceRequest{
		Amount:    500.0,
		Direction: "debit",
		Reason:    "withdrawal",
	}

	suite.mockService.On("UpdateBalance", mock.Anything, suite.testUserID, suite.testSubAccountID, &req).
		Return(nil, fmt.Errorf("insufficient balance"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/sub-accounts/%s/balance", suite.testSubAccountID), bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INSUFFICIENT_BALANCE", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestDeleteSubAccount_Success() {
	// Arrange
	suite.mockService.On("DeleteSubAccount", mock.Anything, suite.testUserID, suite.testSubAccountID).
		Return(nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/sub-accounts/%s", suite.testSubAccountID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	data := response.Data.(map[string]interface{})
	assert.Equal(suite.T(), "Sub-account deleted successfully", data["message"])

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestDeleteSubAccount_HasBalance() {
	// Arrange
	suite.mockService.On("DeleteSubAccount", mock.Anything, suite.testUserID, suite.testSubAccountID).
		Return(fmt.Errorf("cannot delete sub-account with positive balance"))

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/sub-accounts/%s", suite.testSubAccountID), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SUBACCOUNT_HAS_BALANCE", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestGetSubAccountsBySymbol_Success() {
	// Arrange
	symbol := "BTC"
	subAccounts := []*services.SubAccountResponse{
		{
			ID:         suite.testSubAccountID,
			UserID:     suite.testUserID,
			ExchangeID: suite.testExchangeID,
			Name:       "BTC Sub Account 1",
			Symbol:     symbol,
			Balance:    100.0,
			CreatedAt:  time.Now().Format(time.RFC3339),
		},
		{
			ID:         uuid.New(),
			UserID:     suite.testUserID,
			ExchangeID: uuid.New(),
			Name:       "BTC Sub Account 2",
			Symbol:     symbol,
			Balance:    50.0,
			CreatedAt:  time.Now().Format(time.RFC3339),
		},
	}

	suite.mockService.On("GetSubAccountsBySymbol", mock.Anything, suite.testUserID, symbol).
		Return(subAccounts, nil)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/sub-accounts/symbol/%s", symbol), nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	data := response.Data.(map[string]interface{})
	subAccountsData := data["sub_accounts"].([]interface{})
	assert.Len(suite.T(), subAccountsData, 2)
	assert.Equal(suite.T(), symbol, data["symbol"])

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *SubAccountHandlerTestSuite) TestGetSubAccountsBySymbol_EmptySymbol() {
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/sub-accounts/symbol/", nil)
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code) // Gin returns 404 for empty path params
}

func (suite *SubAccountHandlerTestSuite) TestCreateSubAccount_ServiceError() {
	// Arrange
	req := services.CreateSubAccountRequest{
		ExchangeID: suite.testExchangeID,
		Name:       "Test Sub Account",
		Symbol:     "BTC",
	}

	suite.mockService.On("CreateSubAccount", mock.Anything, suite.testUserID, &req).
		Return(nil, fmt.Errorf("database connection failed"))

	body, _ := json.Marshal(req)

	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/api/v1/sub-accounts", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)

	// Assert
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "SUBACCOUNT_CREATE_FAILED", response.Error.Code)

	suite.mockService.AssertExpectations(suite.T())
}

func TestSubAccountHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SubAccountHandlerTestSuite))
}

