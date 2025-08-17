package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"tiris-backend/internal/api"
	"tiris-backend/internal/services"
	"tiris-backend/test/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// UserServiceInterface defines the interface for user service operations
type UserServiceInterface interface {
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*services.UserResponse, error)
	UpdateCurrentUser(ctx context.Context, userID uuid.UUID, req *services.UpdateUserRequest) (*services.UserResponse, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error)
	ListUsers(ctx context.Context, limit, offset int) ([]*services.UserResponse, int64, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*services.UserResponse, error)
	DisableUser(ctx context.Context, userID uuid.UUID) error
}

// MockUserService is a mock implementation of UserServiceInterface for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*services.UserResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserResponse), args.Error(1)
}

func (m *MockUserService) UpdateCurrentUser(ctx context.Context, userID uuid.UUID, req *services.UpdateUserRequest) (*services.UserResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserResponse), args.Error(1)
}

func (m *MockUserService) GetUserStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockUserService) ListUsers(ctx context.Context, limit, offset int) ([]*services.UserResponse, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*services.UserResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*services.UserResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserResponse), args.Error(1)
}

func (m *MockUserService) DisableUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// TestableUserHandler wraps the UserHandler to allow interface-based testing
type TestableUserHandler struct {
	userService UserServiceInterface
}

func NewTestableUserHandler(userService UserServiceInterface) *TestableUserHandler {
	return &TestableUserHandler{
		userService: userService,
	}
}

// Copy all the handler methods but use the interface
func (h *TestableUserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, api.CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	user, err := h.userService.GetCurrentUser(c.Request.Context(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.CreateErrorResponse(
			"USER_GET_FAILED",
			"Failed to get user profile",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, api.CreateSuccessResponse(user, getTraceID(c)))
}

func (h *TestableUserHandler) UpdateCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, api.CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	user, err := h.userService.UpdateCurrentUser(c.Request.Context(), userUUID, &req)
	if err != nil {
		if err.Error() == "username already taken" {
			c.JSON(http.StatusConflict, api.CreateErrorResponse(
				"USERNAME_TAKEN",
				"Username is already taken",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, api.CreateErrorResponse(
			"USER_UPDATE_FAILED",
			"Failed to update user profile",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, api.CreateSuccessResponse(user, getTraceID(c)))
}

func (h *TestableUserHandler) GetUserStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, api.CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, api.CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	stats, err := h.userService.GetUserStats(c.Request.Context(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.CreateErrorResponse(
			"STATS_GET_FAILED",
			"Failed to get user statistics",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, api.CreateSuccessResponse(stats, getTraceID(c)))
}

func (h *TestableUserHandler) ListUsers(c *gin.Context) {
	// Parse pagination parameters (same logic as original)
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	users, total, err := h.userService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.CreateErrorResponse(
			"USERS_LIST_FAILED",
			"Failed to list users",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	// Create pagination metadata
	hasMore := int64(offset+limit) < total
	var nextOffset *int
	if hasMore {
		next := offset + limit
		nextOffset = &next
	}

	pagination := &api.PaginationMetadata{
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}

	response := map[string]interface{}{
		"users": users,
	}

	c.JSON(http.StatusOK, api.CreatePaginatedResponse(response, pagination, getTraceID(c)))
}

func (h *TestableUserHandler) GetUserByID(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.CreateErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, api.CreateErrorResponse(
				"USER_NOT_FOUND",
				"User not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, api.CreateErrorResponse(
			"USER_GET_FAILED",
			"Failed to get user",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, api.CreateSuccessResponse(user, getTraceID(c)))
}

func (h *TestableUserHandler) DisableUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.CreateErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	err = h.userService.DisableUser(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, api.CreateErrorResponse(
				"USER_NOT_FOUND",
				"User not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, api.CreateErrorResponse(
			"USER_DISABLE_FAILED",
			"Failed to disable user",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"message": "User account disabled successfully",
	}

	c.JSON(http.StatusOK, api.CreateSuccessResponse(response, getTraceID(c)))
}

// Helper function to get trace ID
func getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("request_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// Test helper functions

func setupUserHandlerTest() (*gin.Engine, *MockUserService, *TestableUserHandler) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create mock service
	mockUserService := &MockUserService{}
	
	// Create handler
	userHandler := NewTestableUserHandler(mockUserService)
	
	// Setup Gin router
	router := gin.New()
	
	// Add middleware to set user ID in context (simulating authentication)
	router.Use(func(c *gin.Context) {
		if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", userID)
			}
		}
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			c.Set("request_id", requestID)
		}
		c.Next()
	})
	
	return router, mockUserService, userHandler
}

func createTestUserResponse() *services.UserResponse {
	return &services.UserResponse{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Avatar:   stringPtr("https://example.com/avatar.jpg"),
		Settings: map[string]interface{}{"theme": "dark"},
		Info:     map[string]interface{}{"timezone": "UTC"},
	}
}

func stringPtr(s string) *string {
	return &s
}

// Test GetCurrentUser endpoint
func TestUserHandler_GetCurrentUser(t *testing.T) {
	router, mockService, handler := setupUserHandlerTest()
	
	// Setup route
	router.GET("/users/me", handler.GetCurrentUser)
	
	t.Run("successful_get_current_user", func(t *testing.T) {
		userID := uuid.New()
		testUser := createTestUserResponse()
		testUser.ID = userID
		
		// Setup mock expectations
		mockService.On("GetCurrentUser", mock.Anything, userID).Return(testUser, nil)
		
		// Create request
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Request-ID", "test-request-123")
		
		// Create response recorder
		w := httptest.NewRecorder()
		
		// Execute request
		router.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
		assert.Equal(t, "test-request-123", response.Metadata.TraceID)
		
		// Verify mock was called
		mockService.AssertExpectations(t)
	})
	
	t.Run("unauthorized_no_user_id", func(t *testing.T) {
		// Create request without user ID
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-Request-ID", "test-request-456")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify unauthorized response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response.Success)
		assert.Equal(t, "AUTH_REQUIRED", response.Error.Code)
		assert.Equal(t, "Authentication required", response.Error.Message)
		assert.Equal(t, "test-request-456", response.Metadata.TraceID)
	})
	
	t.Run("service_error", func(t *testing.T) {
		userID := uuid.New()
		
		// Setup mock to return error
		mockService.On("GetCurrentUser", mock.Anything, userID).Return(nil, errors.New("database connection failed"))
		
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Request-ID", "test-request-789")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify error response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response.Success)
		assert.Equal(t, "USER_GET_FAILED", response.Error.Code)
		assert.Equal(t, "Failed to get user profile", response.Error.Message)
		
		mockService.AssertExpectations(t)
	})
}

// Test UpdateCurrentUser endpoint
func TestUserHandler_UpdateCurrentUser(t *testing.T) {
	router, mockService, handler := setupUserHandlerTest()
	
	// Setup route
	router.PUT("/users/me", handler.UpdateCurrentUser)
	
	t.Run("successful_update", func(t *testing.T) {
		userID := uuid.New()
		testUser := createTestUserResponse()
		testUser.ID = userID
		testUser.Username = "updateduser"
		
		updateReq := &services.UpdateUserRequest{
			Username: stringPtr("updateduser"),
			Avatar:   stringPtr("https://example.com/new-avatar.jpg"),
		}
		
		// Setup mock expectations
		mockService.On("UpdateCurrentUser", mock.Anything, userID, mock.MatchedBy(func(req *services.UpdateUserRequest) bool {
			return req.Username != nil && *req.Username == "updateduser"
		})).Return(testUser, nil)
		
		// Create request body
		reqBody, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/users/me", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Request-ID", "test-update-123")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		mockService.AssertExpectations(t)
	})
	
	t.Run("invalid_json", func(t *testing.T) {
		userID := uuid.New()
		
		// Create request with invalid JSON
		req := httptest.NewRequest("PUT", "/users/me", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Request-ID", "test-invalid-json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify bad request response
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response.Success)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})
	
	t.Run("username_taken", func(t *testing.T) {
		userID := uuid.New()
		
		updateReq := &services.UpdateUserRequest{
			Username: stringPtr("takenusername"),
		}
		
		// Setup mock to return username taken error
		mockService.On("UpdateCurrentUser", mock.Anything, userID, mock.Anything).Return(nil, errors.New("username already taken"))
		
		reqBody, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/users/me", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify conflict response
		assert.Equal(t, http.StatusConflict, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response.Success)
		assert.Equal(t, "USERNAME_TAKEN", response.Error.Code)
		
		mockService.AssertExpectations(t)
	})
}

// Test GetUserStats endpoint
func TestUserHandler_GetUserStats(t *testing.T) {
	router, mockService, handler := setupUserHandlerTest()
	
	// Setup route
	router.GET("/users/me/stats", handler.GetUserStats)
	
	t.Run("successful_get_stats", func(t *testing.T) {
		userID := uuid.New()
		testStats := map[string]interface{}{
			"total_exchanges":   3,
			"total_subaccounts": 15,
			"total_balance_usd": 25000.50,
			"active_trades":     8,
		}
		
		// Setup mock expectations
		mockService.On("GetUserStats", mock.Anything, userID).Return(testStats, nil)
		
		req := httptest.NewRequest("GET", "/users/me/stats", nil)
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Request-ID", "test-stats-123")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		assert.NotNil(t, response.Data)
		
		// Verify stats data
		statsData, ok := response.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(3), statsData["total_exchanges"]) // JSON numbers are float64
		
		mockService.AssertExpectations(t)
	})
	
	t.Run("service_error", func(t *testing.T) {
		userID := uuid.New()
		
		mockService.On("GetUserStats", mock.Anything, userID).Return(nil, errors.New("stats calculation failed"))
		
		req := httptest.NewRequest("GET", "/users/me/stats", nil)
		req.Header.Set("X-User-ID", userID.String())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "STATS_GET_FAILED", response.Error.Code)
		mockService.AssertExpectations(t)
	})
}

// Test ListUsers endpoint (admin functionality)
func TestUserHandler_ListUsers(t *testing.T) {
	router, mockService, handler := setupUserHandlerTest()
	
	// Setup route
	router.GET("/users", handler.ListUsers)
	
	t.Run("successful_list_users", func(t *testing.T) {
		testUsers := []*services.UserResponse{
			createTestUserResponse(),
			createTestUserResponse(),
		}
		testUsers[0].Username = "user1"
		testUsers[1].Username = "user2"
		
		// Setup mock expectations
		mockService.On("ListUsers", mock.Anything, 100, 0).Return(testUsers, int64(2), nil)
		
		req := httptest.NewRequest("GET", "/users", nil)
		req.Header.Set("X-Request-ID", "test-list-123")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.PaginatedResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		assert.NotNil(t, response.Pagination)
		assert.Equal(t, int64(2), response.Pagination.Total)
		assert.Equal(t, 100, response.Pagination.Limit)
		assert.Equal(t, 0, response.Pagination.Offset)
		assert.False(t, response.Pagination.HasMore)
		
		mockService.AssertExpectations(t)
	})
	
	t.Run("pagination_with_custom_params", func(t *testing.T) {
		testUsers := []*services.UserResponse{
			createTestUserResponse(),
		}
		
		mockService.On("ListUsers", mock.Anything, 10, 20).Return(testUsers, int64(100), nil)
		
		req := httptest.NewRequest("GET", "/users?limit=10&offset=20", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.PaginatedResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Pagination.HasMore)
		assert.NotNil(t, response.Pagination.NextOffset)
		assert.Equal(t, 30, *response.Pagination.NextOffset)
		
		mockService.AssertExpectations(t)
	})
	
	t.Run("invalid_pagination_params", func(t *testing.T) {
		// Should use defaults for invalid params
		testUsers := []*services.UserResponse{}
		
		mockService.On("ListUsers", mock.Anything, 100, 0).Return(testUsers, int64(0), nil)
		
		req := httptest.NewRequest("GET", "/users?limit=invalid&offset=-5", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}

// Test GetUserByID endpoint (admin functionality)
func TestUserHandler_GetUserByID(t *testing.T) {
	router, mockService, handler := setupUserHandlerTest()
	
	// Setup route
	router.GET("/users/:id", handler.GetUserByID)
	
	t.Run("successful_get_user_by_id", func(t *testing.T) {
		userID := uuid.New()
		testUser := createTestUserResponse()
		testUser.ID = userID
		
		mockService.On("GetUserByID", mock.Anything, userID).Return(testUser, nil)
		
		req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
		req.Header.Set("X-Request-ID", "test-get-user-123")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		mockService.AssertExpectations(t)
	})
	
	t.Run("invalid_user_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/invalid-uuid", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "INVALID_USER_ID", response.Error.Code)
	})
	
	t.Run("user_not_found", func(t *testing.T) {
		userID := uuid.New()
		
		mockService.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("user not found"))
		
		req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "USER_NOT_FOUND", response.Error.Code)
		mockService.AssertExpectations(t)
	})
}

// Test DisableUser endpoint (admin functionality)
func TestUserHandler_DisableUser(t *testing.T) {
	router, mockService, handler := setupUserHandlerTest()
	
	// Setup route
	router.PUT("/users/:id/disable", handler.DisableUser)
	
	t.Run("successful_disable_user", func(t *testing.T) {
		userID := uuid.New()
		
		mockService.On("DisableUser", mock.Anything, userID).Return(nil)
		
		req := httptest.NewRequest("PUT", "/users/"+userID.String()+"/disable", nil)
		req.Header.Set("X-Request-ID", "test-disable-123")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response api.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		
		// Verify success message
		data, ok := response.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "User account disabled successfully", data["message"])
		
		mockService.AssertExpectations(t)
	})
	
	t.Run("user_not_found_disable", func(t *testing.T) {
		userID := uuid.New()
		
		mockService.On("DisableUser", mock.Anything, userID).Return(errors.New("user not found"))
		
		req := httptest.NewRequest("PUT", "/users/"+userID.String()+"/disable", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "USER_NOT_FOUND", response.Error.Code)
		mockService.AssertExpectations(t)
	})
	
	t.Run("invalid_user_id_disable", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/users/invalid-uuid/disable", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "INVALID_USER_ID", response.Error.Code)
	})
}

// Performance tests for user handler endpoints
func TestUserHandler_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	router, mockService, handler := setupUserHandlerTest()
	router.GET("/users/me", handler.GetCurrentUser)
	
	t.Run("concurrent_get_current_user", func(t *testing.T) {
		userID := uuid.New()
		testUser := createTestUserResponse()
		
		// Setup mock to handle multiple concurrent calls
		mockService.On("GetCurrentUser", mock.Anything, userID).Return(testUser, nil).Times(100)
		
		// Run 100 concurrent requests
		concurrency := 10
		requests := 100
		
		results := make(chan int, requests)
		
		for i := 0; i < concurrency; i++ {
			go func() {
				for j := 0; j < requests/concurrency; j++ {
					req := httptest.NewRequest("GET", "/users/me", nil)
					req.Header.Set("X-User-ID", userID.String())
					
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
					
					results <- w.Code
				}
			}()
		}
		
		// Collect results
		for i := 0; i < requests; i++ {
			code := <-results
			assert.Equal(t, http.StatusOK, code)
		}
		
		mockService.AssertExpectations(t)
	})
}