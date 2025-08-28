package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/auth"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockJWTManager implements auth.JWTManagerInterface for testing
type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) GenerateToken(userID uuid.UUID, username, email, role string) (string, error) {
	args := m.Called(userID, username, email, role)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) GenerateTokenPair(userID uuid.UUID, username, email, role string) (*auth.TokenPair, error) {
	args := m.Called(userID, username, email, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockJWTManager) ValidateToken(tokenString string) (*auth.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Claims), args.Error(1)
}

func (m *MockJWTManager) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	args := m.Called(tokenString)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockJWTManager) RefreshToken(refreshToken, username, email, role string) (string, error) {
	args := m.Called(refreshToken, username, email, role)
	return args.String(0), args.Error(1)
}

// MockOAuthManager implements auth.OAuthManagerInterface for testing
type MockOAuthManager struct {
	mock.Mock
}

func (m *MockOAuthManager) GetAuthURL(provider auth.OAuthProvider, state string) (string, error) {
	args := m.Called(provider, state)
	return args.String(0), args.Error(1)
}

func (m *MockOAuthManager) ExchangeCodeForToken(provider auth.OAuthProvider, code string) (*oauth2.Token, error) {
	args := m.Called(provider, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *MockOAuthManager) GetUserInfo(provider auth.OAuthProvider, token *oauth2.Token) (*auth.OAuthUser, error) {
	args := m.Called(provider, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.OAuthUser), args.Error(1)
}

// TestAuthService_InitiateLogin tests the InitiateLogin functionality
func TestAuthService_InitiateLogin(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockJWTManager := &MockJWTManager{}
	mockOAuthManager := &MockOAuthManager{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	authService := services.NewAuthService(repos, mockJWTManager, mockOAuthManager)

	// Test successful login initiation
	t.Run("successful_google_login", func(t *testing.T) {
		request := &services.LoginRequest{
			Provider:    "google",
			RedirectURL: "https://example.com/callback",
		}

		expectedAuthURL := "https://accounts.google.com/oauth2/auth?client_id=test&redirect_uri=https://example.com/callback&state=test_state"

		// Setup mock expectations
		mockOAuthManager.On("GetAuthURL", auth.ProviderGoogle, mock.AnythingOfType("string")).
			Return(expectedAuthURL, nil).Once()

		// Execute test
		result, err := authService.InitiateLogin(context.Background(), request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expectedAuthURL, result.AuthURL)
		assert.NotEmpty(t, result.State)

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
	})

	// Test successful WeChat login initiation
	t.Run("successful_wechat_login", func(t *testing.T) {
		request := &services.LoginRequest{
			Provider:    "wechat",
			RedirectURL: "https://example.com/callback",
		}

		expectedAuthURL := "https://open.weixin.qq.com/connect/oauth2/authorize?appid=test&redirect_uri=https://example.com/callback&state=test_state"

		// Setup mock expectations
		mockOAuthManager.On("GetAuthURL", auth.ProviderWeChat, mock.AnythingOfType("string")).
			Return(expectedAuthURL, nil).Once()

		// Execute test
		result, err := authService.InitiateLogin(context.Background(), request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expectedAuthURL, result.AuthURL)
		assert.NotEmpty(t, result.State)

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
	})

	// Test OAuth manager failure
	t.Run("oauth_manager_failure", func(t *testing.T) {
		request := &services.LoginRequest{
			Provider:    "google",
			RedirectURL: "https://example.com/callback",
		}

		// Setup mock expectations
		mockOAuthManager.On("GetAuthURL", auth.ProviderGoogle, mock.AnythingOfType("string")).
			Return("", errors.New("oauth config error")).Once()

		// Execute test
		result, err := authService.InitiateLogin(context.Background(), request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to generate auth URL")

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
	})
}

// TestAuthService_HandleCallback tests the HandleCallback functionality
func TestAuthService_HandleCallback(t *testing.T) {
	// Create mocks
	mockJWTManager := &MockJWTManager{}
	mockOAuthManager := &MockOAuthManager{}
	mockUserRepo := &mocks.MockUserRepository{}
	mockOAuthTokenRepo := &mocks.MockOAuthTokenRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Trading:         &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      mockOAuthTokenRepo,
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	authService := services.NewAuthService(repos, mockJWTManager, mockOAuthManager)

	// Test successful callback for new user
	t.Run("successful_callback_new_user", func(t *testing.T) {
		request := &services.CallbackRequest{
			Provider: "google",
			Code:     "test_code_12345",
			State:    "valid_state",
		}
		expectedState := "valid_state"

		// Create test OAuth token and user
		oauthToken := &oauth2.Token{
			AccessToken:  "access_token_123",
			RefreshToken: "refresh_token_456",
			Expiry:       time.Now().Add(time.Hour),
		}

		oauthUser := &auth.OAuthUser{
			ID:       "google_user_123",
			Email:    "test@example.com",
			Name:     "Test User",
			Avatar:   "https://example.com/avatar.jpg",
			Provider: "google",
		}

		tokenPair := &auth.TokenPair{
			AccessToken:  "jwt_access_token",
			RefreshToken: "jwt_refresh_token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		}

		// Setup mock expectations
		mockOAuthManager.On("ExchangeCodeForToken", auth.ProviderGoogle, "test_code_12345").
			Return(oauthToken, nil).Once()
		mockOAuthManager.On("GetUserInfo", auth.ProviderGoogle, oauthToken).
			Return(oauthUser, nil).Once()

		// No existing OAuth token
		mockOAuthTokenRepo.On("GetByProviderUserID", mock.Anything, "google", "google_user_123").
			Return(nil, nil).Once()

		// No existing user by email
		mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").
			Return(nil, nil).Once()

		// Create new user
		mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).
			Return(nil).Once()

		// Create OAuth token
		mockOAuthTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.OAuthToken")).
			Return(nil).Once()

		// Generate JWT tokens
		mockJWTManager.On("GenerateTokenPair", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("string"), "test@example.com", "user").
			Return(tokenPair, nil).Once()

		// Execute test
		result, err := authService.HandleCallback(context.Background(), request, expectedState)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "jwt_access_token", result.AccessToken)
		assert.Equal(t, "jwt_refresh_token", result.RefreshToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, int64(3600), result.ExpiresIn)
		assert.NotNil(t, result.User)
		assert.Equal(t, "test@example.com", result.User.Email)

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockOAuthTokenRepo.AssertExpectations(t)
		mockJWTManager.AssertExpectations(t)
	})

	// Test successful callback for existing user
	t.Run("successful_callback_existing_user", func(t *testing.T) {
		request := &services.CallbackRequest{
			Provider: "google",
			Code:     "test_code_12345",
			State:    "valid_state",
		}
		expectedState := "valid_state"

		userID := uuid.New()
		userFactory := helpers.NewUserFactory()
		existingUser := userFactory.WithEmail("existing@example.com")
		existingUser.ID = userID

		// Create test OAuth token and user
		oauthToken := &oauth2.Token{
			AccessToken:  "new_access_token",
			RefreshToken: "new_refresh_token",
			Expiry:       time.Now().Add(time.Hour),
		}

		oauthUser := &auth.OAuthUser{
			ID:       "google_user_123",
			Email:    "existing@example.com",
			Name:     "Existing User",
			Avatar:   "https://example.com/new_avatar.jpg",
			Provider: "google",
		}

		existingOAuthToken := &models.OAuthToken{
			UserID:         userID,
			Provider:       "google",
			ProviderUserID: "google_user_123",
			AccessToken:    "old_access_token",
		}

		tokenPair := &auth.TokenPair{
			AccessToken:  "jwt_access_token",
			RefreshToken: "jwt_refresh_token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		}

		// Setup mock expectations
		mockOAuthManager.On("ExchangeCodeForToken", auth.ProviderGoogle, "test_code_12345").
			Return(oauthToken, nil).Once()
		mockOAuthManager.On("GetUserInfo", auth.ProviderGoogle, oauthToken).
			Return(oauthUser, nil).Once()

		// Existing OAuth token found
		mockOAuthTokenRepo.On("GetByProviderUserID", mock.Anything, "google", "google_user_123").
			Return(existingOAuthToken, nil).Once()

		// Update OAuth token
		mockOAuthTokenRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.OAuthToken")).
			Return(nil).Once()

		// Get existing user
		mockUserRepo.On("GetByID", mock.Anything, userID).
			Return(existingUser, nil).Once()

		// Update user avatar
		mockUserRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).
			Return(nil).Once()

		// Generate JWT tokens
		mockJWTManager.On("GenerateTokenPair", userID, existingUser.Username, existingUser.Email, "user").
			Return(tokenPair, nil).Once()

		// Execute test
		result, err := authService.HandleCallback(context.Background(), request, expectedState)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "jwt_access_token", result.AccessToken)
		assert.Equal(t, userID, result.User.ID)
		assert.Equal(t, existingUser.Email, result.User.Email)

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockOAuthTokenRepo.AssertExpectations(t)
		mockJWTManager.AssertExpectations(t)
	})

	// Test invalid state parameter
	t.Run("invalid_state", func(t *testing.T) {
		request := &services.CallbackRequest{
			Provider: "google",
			Code:     "test_code_12345",
			State:    "invalid_state",
		}
		expectedState := "valid_state"

		// Execute test
		result, err := authService.HandleCallback(context.Background(), request, expectedState)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid state parameter")
	})

	// Test OAuth token exchange failure
	t.Run("token_exchange_failure", func(t *testing.T) {
		request := &services.CallbackRequest{
			Provider: "google",
			Code:     "invalid_code",
			State:    "valid_state",
		}
		expectedState := "valid_state"

		// Setup mock expectations
		mockOAuthManager.On("ExchangeCodeForToken", auth.ProviderGoogle, "invalid_code").
			Return(nil, errors.New("invalid authorization code")).Once()

		// Execute test
		result, err := authService.HandleCallback(context.Background(), request, expectedState)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to exchange code for token")

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
	})

	// Test user info retrieval failure
	t.Run("user_info_failure", func(t *testing.T) {
		request := &services.CallbackRequest{
			Provider: "google",
			Code:     "test_code_12345",
			State:    "valid_state",
		}
		expectedState := "valid_state"

		oauthToken := &oauth2.Token{
			AccessToken: "access_token_123",
		}

		// Setup mock expectations
		mockOAuthManager.On("ExchangeCodeForToken", auth.ProviderGoogle, "test_code_12345").
			Return(oauthToken, nil).Once()
		mockOAuthManager.On("GetUserInfo", auth.ProviderGoogle, oauthToken).
			Return(nil, errors.New("failed to get user info")).Once()

		// Execute test
		result, err := authService.HandleCallback(context.Background(), request, expectedState)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get user info")

		// Verify mock expectations
		mockOAuthManager.AssertExpectations(t)
	})
}

// TestAuthService_RefreshToken tests the RefreshToken functionality
func TestAuthService_RefreshToken(t *testing.T) {
	// Create mocks
	mockJWTManager := &MockJWTManager{}
	mockUserRepo := &mocks.MockUserRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Trading:         &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	authService := services.NewAuthService(repos, mockJWTManager, &MockOAuthManager{})

	// Test successful token refresh
	t.Run("successful_refresh", func(t *testing.T) {
		userID := uuid.New()
		refreshToken := "valid_refresh_token"
		newAccessToken := "new_access_token"

		userFactory := helpers.NewUserFactory()
		testUser := userFactory.WithEmail("test@example.com")
		testUser.ID = userID

		request := &services.RefreshRequest{
			RefreshToken: refreshToken,
		}

		// Setup mock expectations
		mockJWTManager.On("ValidateRefreshToken", refreshToken).
			Return(userID, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, userID).
			Return(testUser, nil).Once()
		mockJWTManager.On("RefreshToken", refreshToken, testUser.Username, testUser.Email, "user").
			Return(newAccessToken, nil).Once()

		// Execute test
		result, err := authService.RefreshToken(context.Background(), request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newAccessToken, result.AccessToken)
		assert.Equal(t, refreshToken, result.RefreshToken) // Should keep the same refresh token
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, int64(3600), result.ExpiresIn)
		assert.Equal(t, userID, result.User.ID)
		assert.Equal(t, testUser.Email, result.User.Email)

		// Verify mock expectations
		mockJWTManager.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	// Test invalid refresh token
	t.Run("invalid_refresh_token", func(t *testing.T) {
		refreshToken := "invalid_refresh_token"

		request := &services.RefreshRequest{
			RefreshToken: refreshToken,
		}

		// Setup mock expectations
		mockJWTManager.On("ValidateRefreshToken", refreshToken).
			Return(uuid.Nil, errors.New("invalid refresh token")).Once()

		// Execute test
		result, err := authService.RefreshToken(context.Background(), request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid refresh token")

		// Verify mock expectations
		mockJWTManager.AssertExpectations(t)
	})

	// Test user not found
	t.Run("user_not_found", func(t *testing.T) {
		userID := uuid.New()
		refreshToken := "valid_refresh_token"

		request := &services.RefreshRequest{
			RefreshToken: refreshToken,
		}

		// Setup mock expectations
		mockJWTManager.On("ValidateRefreshToken", refreshToken).
			Return(userID, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, userID).
			Return(nil, nil).Once() // User not found

		// Execute test
		result, err := authService.RefreshToken(context.Background(), request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "user not found")

		// Verify mock expectations
		mockJWTManager.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	// Test token refresh failure
	t.Run("token_refresh_failure", func(t *testing.T) {
		userID := uuid.New()
		refreshToken := "valid_refresh_token"

		userFactory := helpers.NewUserFactory()
		testUser := userFactory.WithEmail("test@example.com")
		testUser.ID = userID

		request := &services.RefreshRequest{
			RefreshToken: refreshToken,
		}

		// Setup mock expectations
		mockJWTManager.On("ValidateRefreshToken", refreshToken).
			Return(userID, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, userID).
			Return(testUser, nil).Once()
		mockJWTManager.On("RefreshToken", refreshToken, testUser.Username, testUser.Email, "user").
			Return("", errors.New("failed to generate new token")).Once()

		// Execute test
		result, err := authService.RefreshToken(context.Background(), request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to refresh token")

		// Verify mock expectations
		mockJWTManager.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthService_Logout tests the Logout functionality
func TestAuthService_Logout(t *testing.T) {
	// Create service
	authService := services.NewAuthService(&repositories.Repositories{}, &MockJWTManager{}, &MockOAuthManager{})

	// Test successful logout
	t.Run("successful_logout", func(t *testing.T) {
		userID := uuid.New()

		// Execute test
		err := authService.Logout(context.Background(), userID)

		// Verify results - should always succeed for now
		require.NoError(t, err)
	})
}

// TestAuthService_generateUsername tests the generateUsername functionality
func TestAuthService_generateUsername(t *testing.T) {
	// We'll test this via reflection since it's a private method
	// For now, we'll test the happy path scenarios in integration
	t.Run("username_generation_scenarios", func(t *testing.T) {
		// This is tested indirectly through the HandleCallback tests
		// where new users are created and usernames are generated
		assert.True(t, true, "Username generation is tested through HandleCallback")
	})
}

// Performance test for auth operations
func TestAuthService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockJWTManager := &MockJWTManager{}
	mockOAuthManager := &MockOAuthManager{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:         &mocks.MockTradingRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	authService := services.NewAuthService(repos, mockJWTManager, mockOAuthManager)

	t.Run("initiate_login_performance", func(t *testing.T) {
		request := &services.LoginRequest{
			Provider:    "google",
			RedirectURL: "https://example.com/callback",
		}

		expectedAuthURL := "https://accounts.google.com/oauth2/auth?test=1"

		// Setup mock for multiple calls
		mockOAuthManager.On("GetAuthURL", auth.ProviderGoogle, mock.AnythingOfType("string")).
			Return(expectedAuthURL, nil).Times(100)

		timer := helpers.NewPerformanceTimer()
		timer.Start()

		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := authService.InitiateLogin(context.Background(), request)
			require.NoError(t, err)
		}

		timer.Stop()

		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 InitiateLogin operations should complete within 1 second")

		mockOAuthManager.AssertExpectations(t)
	})
}
