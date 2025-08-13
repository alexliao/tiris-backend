package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tiris-backend/internal/services"
	"tiris-backend/test/fixtures"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// AuthServiceInterface for testing
type AuthServiceInterface interface {
	InitiateLogin(ctx context.Context, req *services.LoginRequest) (*services.LoginResponse, error)
	HandleCallback(ctx context.Context, req *services.CallbackRequest, expectedState string) (*services.AuthResponse, error)
	RefreshToken(ctx context.Context, req *services.RefreshRequest) (*services.AuthResponse, error)
	Logout(ctx context.Context, userID uuid.UUID) error
}

// MockAuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) InitiateLogin(ctx context.Context, req *services.LoginRequest) (*services.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LoginResponse), args.Error(1)
}

func (m *MockAuthService) HandleCallback(ctx context.Context, req *services.CallbackRequest, expectedState string) (*services.AuthResponse, error) {
	args := m.Called(ctx, req, expectedState)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, req *services.RefreshRequest) (*services.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// TestAuthHandler wraps the handler for testing
type TestAuthHandler struct {
	authService AuthServiceInterface
}

func (h *TestAuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	response, err := h.authService.InitiateLogin(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "LOGIN_FAILED",
				Message: "Failed to initiate login",
				Details: err.Error(),
			},
		})
		return
	}

	c.SetCookie("oauth_state", response.State, 600, "/", "", false, true)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
	})
}

func (h *TestAuthHandler) Callback(c *gin.Context) {
	var req services.CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	expectedState, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_STATE",
				Message: "Missing or invalid state parameter",
				Details: "OAuth state not found in session",
			},
		})
		return
	}

	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	response, err := h.authService.HandleCallback(c.Request.Context(), &req, expectedState)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "OAUTH_CALLBACK_FAILED",
				Message: "OAuth callback failed",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
	})
}

func (h *TestAuthHandler) Refresh(c *gin.Context) {
	var req services.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "TOKEN_REFRESH_FAILED",
				Message: "Failed to refresh token",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
	})
}

func (h *TestAuthHandler) Logout(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "AUTH_REQUIRED",
				Message: "Authentication required",
			},
		})
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "AUTH_REQUIRED",
				Message: "Authentication required",
			},
		})
		return
	}

	err := h.authService.Logout(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "LOGOUT_FAILED",
				Message: "Failed to logout",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: gin.H{
			"message": "Logged out successfully",
		},
	})
}

type AuthHandlerTestSuite struct {
	suite.Suite
	authHandler     *TestAuthHandler
	mockAuthService *MockAuthService
	router          *gin.Engine
}

func (suite *AuthHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.mockAuthService = &MockAuthService{}
	suite.authHandler = &TestAuthHandler{authService: suite.mockAuthService}
	
	suite.router = gin.New()
	suite.router.POST("/auth/login", suite.authHandler.Login)
	suite.router.POST("/auth/callback", suite.authHandler.Callback)
	suite.router.POST("/auth/refresh", suite.authHandler.Refresh)
	suite.router.POST("/auth/logout", suite.authHandler.Logout)
}

func (suite *AuthHandlerTestSuite) TestLogin_Success() {
	// Arrange
	req := services.LoginRequest{
		Provider:    "google",
		RedirectURL: "https://example.com/callback",
	}
	
	response := &services.LoginResponse{
		AuthURL: "https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=callback&response_type=code&state=test-state",
		State:   "test-state",
	}
	
	suite.mockAuthService.On("InitiateLogin", mock.Anything, &req).Return(response, nil)
	
	reqBody, _ := json.Marshal(req)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var successResp SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &successResp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), successResp.Success)
	
	// Check if state cookie is set
	cookies := w.Result().Cookies()
	var stateCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "oauth_state" {
			stateCookie = cookie
			break
		}
	}
	assert.NotNil(suite.T(), stateCookie)
	assert.Equal(suite.T(), "test-state", stateCookie.Value)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestLogin_InvalidRequest() {
	// Arrange
	invalidReq := `{"invalid": "json"}`
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(invalidReq))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "INVALID_REQUEST", errorResp.Error.Code)
}

func (suite *AuthHandlerTestSuite) TestLogin_ServiceError() {
	// Arrange
	req := services.LoginRequest{
		Provider:    "google",
		RedirectURL: "https://example.com/callback",
	}
	
	suite.mockAuthService.On("InitiateLogin", mock.Anything, &req).Return(nil, fmt.Errorf("service error"))
	
	reqBody, _ := json.Marshal(req)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "LOGIN_FAILED", errorResp.Error.Code)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestCallback_Success() {
	// Arrange
	req := services.CallbackRequest{
		Provider: "google",
		Code:     "auth-code",
		State:    "test-state",
	}
	
	response := &services.AuthResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		User: &services.UserInfo{
			ID:       fixtures.UserFixtures.ValidUser.ID,
			Username: "testuser",
			Email:    "test@example.com",
		},
	}
	
	suite.mockAuthService.On("HandleCallback", mock.Anything, &req, "test-state").Return(response, nil)
	
	reqBody, _ := json.Marshal(req)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/callback", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var successResp SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &successResp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), successResp.Success)
	
	// Check that state cookie is cleared
	cookies := w.Result().Cookies()
	var stateCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "oauth_state" {
			stateCookie = cookie
			break
		}
	}
	assert.NotNil(suite.T(), stateCookie)
	assert.Equal(suite.T(), "", stateCookie.Value)
	assert.Equal(suite.T(), -1, stateCookie.MaxAge)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestCallback_InvalidRequest() {
	// Arrange
	invalidReq := `{"invalid": "json"}`
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/callback", bytes.NewBufferString(invalidReq))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "INVALID_REQUEST", errorResp.Error.Code)
}

func (suite *AuthHandlerTestSuite) TestCallback_MissingState() {
	// Arrange
	req := services.CallbackRequest{
		Provider: "google",
		Code:     "auth-code",
		State:    "test-state",
	}
	
	reqBody, _ := json.Marshal(req)
	
	// Act (no state cookie)
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/callback", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "INVALID_STATE", errorResp.Error.Code)
}

func (suite *AuthHandlerTestSuite) TestCallback_ServiceError() {
	// Arrange
	req := services.CallbackRequest{
		Provider: "google",
		Code:     "auth-code",
		State:    "test-state",
	}
	
	suite.mockAuthService.On("HandleCallback", mock.Anything, &req, "test-state").Return(nil, fmt.Errorf("callback error"))
	
	reqBody, _ := json.Marshal(req)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/callback", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "OAUTH_CALLBACK_FAILED", errorResp.Error.Code)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestRefresh_Success() {
	// Arrange
	req := services.RefreshRequest{
		RefreshToken: "refresh-token",
	}
	
	response := &services.AuthResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresIn:    3600,
		User: &services.UserInfo{
			ID:       fixtures.UserFixtures.ValidUser.ID,
			Username: "testuser",
			Email:    "test@example.com",
		},
	}
	
	suite.mockAuthService.On("RefreshToken", mock.Anything, &req).Return(response, nil)
	
	reqBody, _ := json.Marshal(req)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var successResp SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &successResp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), successResp.Success)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestRefresh_InvalidRequest() {
	// Arrange
	invalidReq := `{"invalid": "json"}`
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBufferString(invalidReq))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "INVALID_REQUEST", errorResp.Error.Code)
}

func (suite *AuthHandlerTestSuite) TestRefresh_ServiceError() {
	// Arrange
	req := services.RefreshRequest{
		RefreshToken: "refresh-token",
	}
	
	suite.mockAuthService.On("RefreshToken", mock.Anything, &req).Return(nil, fmt.Errorf("token error"))
	
	reqBody, _ := json.Marshal(req)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, request)
	
	// Assert
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "TOKEN_REFRESH_FAILED", errorResp.Error.Code)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestLogout_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	suite.mockAuthService.On("Logout", mock.Anything, userID).Return(nil)
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/logout", nil)
	
	// Set user in context (simulate auth middleware)
	c, _ := gin.CreateTestContext(w)
	c.Request = request
	c.Set("user_id", userID)
	suite.authHandler.Logout(c)
	
	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var successResp SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &successResp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), successResp.Success)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func (suite *AuthHandlerTestSuite) TestLogout_Unauthorized() {
	// Act (no user in context)
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/logout", nil)
	
	c, _ := gin.CreateTestContext(w)
	c.Request = request
	// Don't set user in context
	suite.authHandler.Logout(c)
	
	// Assert
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "AUTH_REQUIRED", errorResp.Error.Code)
}

func (suite *AuthHandlerTestSuite) TestLogout_ServiceError() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	suite.mockAuthService.On("Logout", mock.Anything, userID).Return(fmt.Errorf("logout error"))
	
	// Act
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/auth/logout", nil)
	
	c, _ := gin.CreateTestContext(w)
	c.Request = request
	c.Set("user_id", userID)
	suite.authHandler.Logout(c)
	
	// Assert
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), errorResp.Success)
	assert.Equal(suite.T(), "LOGOUT_FAILED", errorResp.Error.Code)
	
	suite.mockAuthService.AssertExpectations(suite.T())
}

func TestAuthHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}