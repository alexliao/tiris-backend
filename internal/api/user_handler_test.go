package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tiris-backend/internal/services"
	"tiris-backend/test/fixtures"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockUserServiceFixed for testing with correct interface
type MockUserServiceFixed struct {
	mock.Mock
}

func (m *MockUserServiceFixed) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*services.UserResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserResponse), args.Error(1)
}

func (m *MockUserServiceFixed) UpdateCurrentUser(ctx context.Context, userID uuid.UUID, req *services.UpdateUserRequest) (*services.UserResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserResponse), args.Error(1)
}

func (m *MockUserServiceFixed) DisableUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserServiceFixed) ListUsers(ctx context.Context, limit, offset int) ([]*services.UserResponse, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*services.UserResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserServiceFixed) GetUserByID(ctx context.Context, userID uuid.UUID) (*services.UserResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserResponse), args.Error(1)
}

func (m *MockUserServiceFixed) GetUserStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// UserHandlerTestSuiteFixed defines the test suite with corrected interface
type UserHandlerTestSuiteFixed struct {
	suite.Suite
	handler     *UserHandler
	mockService *MockUserServiceFixed
	router      *gin.Engine
}

func (suite *UserHandlerTestSuiteFixed) SetupTest() {
	gin.SetMode(gin.TestMode)
	
	suite.mockService = new(MockUserServiceFixed)
	// We need to properly type-cast the mock to match the UserHandler expectation
	suite.handler = &UserHandler{
		userService: suite.mockService,
	}
	suite.router = gin.New()
}

func (suite *UserHandlerTestSuiteFixed) TestGetCurrentUser_Success() {
	userID := fixtures.UserFixtures.ValidUser.ID
	expectedResponse := &services.UserResponse{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("GetCurrentUser", mock.AnythingOfType("*gin.Context"), userID).Return(expectedResponse, nil)

	req := httptest.NewRequest("GET", "/api/v1/users/current", nil)
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID)
	
	suite.handler.GetCurrentUser(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuiteFixed) TestGetCurrentUser_Unauthorized() {
	req := httptest.NewRequest("GET", "/api/v1/users/current", nil)
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	
	suite.handler.GetCurrentUser(c)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *UserHandlerTestSuiteFixed) TestUpdateCurrentUser_Success() {
	userID := fixtures.UserFixtures.ValidUser.ID
	updateReq := &services.UpdateUserRequest{
		Username: stringPtr("newusername"),
	}
	
	expectedResponse := &services.UserResponse{
		ID:        userID,
		Username:  "newusername",
		Email:     "test@example.com",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("UpdateCurrentUser", mock.AnythingOfType("*gin.Context"), userID, updateReq).Return(expectedResponse, nil)

	reqBody, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api/v1/users/current", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID)
	
	suite.handler.UpdateCurrentUser(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuiteFixed) TestGetUserStats_Success() {
	userID := fixtures.UserFixtures.ValidUser.ID
	expectedStats := map[string]interface{}{
		"total_exchanges":    2,
		"total_sub_accounts": 5,
		"total_transactions": 100,
		"last_login_at":      fixtures.FixedTime.Format(time.RFC3339),
	}

	suite.mockService.On("GetUserStats", mock.AnythingOfType("*gin.Context"), userID).Return(expectedStats, nil)

	req := httptest.NewRequest("GET", "/api/v1/users/stats", nil)
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID)
	
	suite.handler.GetUserStats(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuiteFixed) TestListUsers_Success() {
	expectedUsers := []*services.UserResponse{
		{
			ID:        fixtures.UserFixtures.ValidUser.ID,
			Username:  "testuser1",
			Email:     "test1@example.com",
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}
	totalCount := int64(1)

	suite.mockService.On("ListUsers", mock.AnythingOfType("*gin.Context"), 10, 0).Return(expectedUsers, totalCount, nil)

	req := httptest.NewRequest("GET", "/api/v1/users?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	
	suite.handler.ListUsers(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuiteFixed) TestGetUserByID_Success() {
	userID := fixtures.UserFixtures.ValidUser.ID
	expectedResponse := &services.UserResponse{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	suite.mockService.On("GetUserByID", mock.AnythingOfType("*gin.Context"), userID).Return(expectedResponse, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", userID), nil)
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: userID.String()}}
	
	suite.handler.GetUserByID(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserHandlerTestSuiteFixed) TestDisableUser_Success() {
	userID := fixtures.UserFixtures.ValidUser.ID

	suite.mockService.On("DisableUser", mock.AnythingOfType("*gin.Context"), userID).Return(nil)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%s", userID), nil)
	w := httptest.NewRecorder()
	
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: userID.String()}}
	
	suite.handler.DisableUser(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	suite.mockService.AssertExpectations(suite.T())
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

func TestUserHandlerTestSuiteFixed(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuiteFixed))
}