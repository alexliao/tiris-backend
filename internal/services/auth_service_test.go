package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/pkg/auth"
	"tiris-backend/test/fixtures"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
	"gorm.io/datatypes"
)

type AuthServiceTestSuite struct {
	suite.Suite
	mockRepos        *repositories.Repositories
	mockJWTManager   *mocks.MockJWTManager
	mockOAuthManager *mocks.MockOAuthManager
	authService      *AuthService
	ctx              context.Context
}

func (suite *AuthServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()

	// Create mock repositories
	suite.mockRepos = &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock managers
	suite.mockJWTManager = &mocks.MockJWTManager{}
	suite.mockOAuthManager = &mocks.MockOAuthManager{}

	// Create service with mocked dependencies
	suite.authService = NewAuthService(suite.mockRepos, suite.mockJWTManager, suite.mockOAuthManager)
}

func (suite *AuthServiceTestSuite) TestInitiateLogin_Success() {
	// Arrange
	req := &LoginRequest{
		Provider:    "google",
		RedirectURL: "https://example.com/callback",
	}
	expectedAuthURL := "https://accounts.google.com/oauth/authorize?state=test-state"

	suite.mockOAuthManager.On("GetAuthURL", auth.OAuthProvider("google"), mock.AnythingOfType("string")).Return(expectedAuthURL, nil)

	// Act
	result, err := suite.authService.InitiateLogin(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), expectedAuthURL, result.AuthURL)
	assert.NotEmpty(suite.T(), result.State)

	suite.mockOAuthManager.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestInitiateLogin_GetAuthURLError() {
	// Arrange
	req := &LoginRequest{
		Provider:    "google",
		RedirectURL: "https://example.com/callback",
	}
	expectedError := errors.New("oauth config error")

	suite.mockOAuthManager.On("GetAuthURL", auth.OAuthProvider("google"), mock.AnythingOfType("string")).Return("", expectedError)

	// Act
	result, err := suite.authService.InitiateLogin(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to generate auth URL")

	suite.mockOAuthManager.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestHandleCallback_Success_ExistingUser() {
	// Arrange
	req := &CallbackRequest{
		Provider: "google",
		Code:     "auth-code",
		State:    "valid-state",
	}
	expectedState := "valid-state"

	token := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	oauthUser := &auth.OAuthUser{
		ID:       "oauth-user-id",
		Email:    fixtures.UserFixtures.ValidUser.Email,
		Name:     "Test User",
		Avatar:   "https://example.com/avatar.png",
		Provider: "google",
	}

	existingOAuthToken := &models.OAuthToken{
		ID:             uuid.New(),
		UserID:         fixtures.UserFixtures.ValidUser.ID,
		Provider:       "google",
		ProviderUserID: "oauth-user-id",
		AccessToken:    "old-access-token",
	}

	tokenPair := &auth.TokenPair{
		AccessToken:  "jwt-access-token",
		RefreshToken: "jwt-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}

	mockOAuthTokenRepo := suite.mockRepos.OAuthToken.(*mocks.MockOAuthTokenRepository)
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)

	suite.mockOAuthManager.On("ExchangeCodeForToken", auth.OAuthProvider("google"), "auth-code").Return(token, nil)
	suite.mockOAuthManager.On("GetUserInfo", auth.OAuthProvider("google"), token).Return(oauthUser, nil)
	mockOAuthTokenRepo.On("GetByProviderUserID", suite.ctx, "google", "oauth-user-id").Return(existingOAuthToken, nil)
	mockOAuthTokenRepo.On("Update", suite.ctx, mock.AnythingOfType("*models.OAuthToken")).Return(nil)
	mockUserRepo.On("GetByID", suite.ctx, fixtures.UserFixtures.ValidUser.ID).Return(fixtures.UserFixtures.ValidUser, nil)
	mockUserRepo.On("Update", suite.ctx, mock.AnythingOfType("*models.User")).Return(nil).Maybe()
	suite.mockJWTManager.On("GenerateTokenPair", fixtures.UserFixtures.ValidUser.ID, fixtures.UserFixtures.ValidUser.Username, fixtures.UserFixtures.ValidUser.Email, "user").Return(tokenPair, nil)

	// Act
	result, err := suite.authService.HandleCallback(suite.ctx, req, expectedState)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), tokenPair.AccessToken, result.AccessToken)
	assert.Equal(suite.T(), tokenPair.RefreshToken, result.RefreshToken)
	assert.Equal(suite.T(), fixtures.UserFixtures.ValidUser.ID, result.User.ID)

	suite.mockOAuthManager.AssertExpectations(suite.T())
	mockOAuthTokenRepo.AssertExpectations(suite.T())
	mockUserRepo.AssertExpectations(suite.T())
	suite.mockJWTManager.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestHandleCallback_InvalidState() {
	// Arrange
	req := &CallbackRequest{
		Provider: "google",
		Code:     "auth-code",
		State:    "invalid-state",
	}
	expectedState := "valid-state"

	// Act
	result, err := suite.authService.HandleCallback(suite.ctx, req, expectedState)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "invalid state parameter")
}

func (suite *AuthServiceTestSuite) TestHandleCallback_ExchangeCodeError() {
	// Arrange
	req := &CallbackRequest{
		Provider: "google",
		Code:     "invalid-code",
		State:    "valid-state",
	}
	expectedState := "valid-state"
	expectedError := errors.New("invalid authorization code")

	suite.mockOAuthManager.On("ExchangeCodeForToken", auth.OAuthProvider("google"), "invalid-code").Return((*oauth2.Token)(nil), expectedError)

	// Act
	result, err := suite.authService.HandleCallback(suite.ctx, req, expectedState)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to exchange code for token")

	suite.mockOAuthManager.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestRefreshToken_Success() {
	// Arrange
	req := &RefreshRequest{
		RefreshToken: "valid-refresh-token",
	}
	userID := fixtures.UserFixtures.ValidUser.ID
	newAccessToken := "new-access-token"

	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)

	suite.mockJWTManager.On("ValidateRefreshToken", req.RefreshToken).Return(userID, nil)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return(fixtures.UserFixtures.ValidUser, nil)
	suite.mockJWTManager.On("RefreshToken", req.RefreshToken, fixtures.UserFixtures.ValidUser.Username, fixtures.UserFixtures.ValidUser.Email, "user").Return(newAccessToken, nil)

	// Act
	result, err := suite.authService.RefreshToken(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), newAccessToken, result.AccessToken)
	assert.Equal(suite.T(), req.RefreshToken, result.RefreshToken)
	assert.Equal(suite.T(), "Bearer", result.TokenType)
	assert.Equal(suite.T(), int64(3600), result.ExpiresIn)
	assert.Equal(suite.T(), fixtures.UserFixtures.ValidUser.ID, result.User.ID)

	suite.mockJWTManager.AssertExpectations(suite.T())
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestRefreshToken_InvalidToken() {
	// Arrange
	req := &RefreshRequest{
		RefreshToken: "invalid-refresh-token",
	}
	expectedError := errors.New("invalid token")

	suite.mockJWTManager.On("ValidateRefreshToken", req.RefreshToken).Return(uuid.Nil, expectedError)

	// Act
	result, err := suite.authService.RefreshToken(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "invalid refresh token")

	suite.mockJWTManager.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestRefreshToken_UserNotFound() {
	// Arrange
	req := &RefreshRequest{
		RefreshToken: "valid-refresh-token",
	}
	userID := uuid.New()

	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)

	suite.mockJWTManager.On("ValidateRefreshToken", req.RefreshToken).Return(userID, nil)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return((*models.User)(nil), repositories.ErrUserNotFound)

	// Act
	result, err := suite.authService.RefreshToken(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get user")

	suite.mockJWTManager.AssertExpectations(suite.T())
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogout_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID

	// Act
	err := suite.authService.Logout(suite.ctx, userID)

	// Assert
	assert.NoError(suite.T(), err)
}

func (suite *AuthServiceTestSuite) TestHandleCallback_Success_NewUser() {
	// Arrange
	req := &CallbackRequest{
		Provider: "google",
		Code:     "auth-code",
		State:    "valid-state",
	}
	expectedState := "valid-state"

	token := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	oauthUser := &auth.OAuthUser{
		ID:       "new-oauth-user-id",
		Email:    "newuser@example.com",
		Name:     "New User",
		Avatar:   "https://example.com/avatar.png",
		Provider: "google",
	}

	newUser := &models.User{
		ID:       uuid.New(),
		Username: "newuser",
		Email:    oauthUser.Email,
		Avatar:   &oauthUser.Avatar,
		Settings: datatypes.JSON{},
		Info:     datatypes.JSON{},
	}

	tokenPair := &auth.TokenPair{
		AccessToken:  "jwt-access-token",
		RefreshToken: "jwt-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}

	mockOAuthTokenRepo := suite.mockRepos.OAuthToken.(*mocks.MockOAuthTokenRepository)
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)

	suite.mockOAuthManager.On("ExchangeCodeForToken", auth.OAuthProvider("google"), "auth-code").Return(token, nil)
	suite.mockOAuthManager.On("GetUserInfo", auth.OAuthProvider("google"), token).Return(oauthUser, nil)
	mockOAuthTokenRepo.On("GetByProviderUserID", suite.ctx, "google", "new-oauth-user-id").Return((*models.OAuthToken)(nil), nil)
	mockUserRepo.On("GetByEmail", suite.ctx, oauthUser.Email).Return((*models.User)(nil), nil)
	mockUserRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.User")).Return(nil).Run(func(args mock.Arguments) {
		user := args.Get(1).(*models.User)
		user.ID = newUser.ID
	})
	mockOAuthTokenRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.OAuthToken")).Return(nil)
	suite.mockJWTManager.On("GenerateTokenPair", newUser.ID, mock.AnythingOfType("string"), oauthUser.Email, "user").Return(tokenPair, nil)

	// Act
	result, err := suite.authService.HandleCallback(suite.ctx, req, expectedState)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), tokenPair.AccessToken, result.AccessToken)
	assert.Equal(suite.T(), tokenPair.RefreshToken, result.RefreshToken)
	assert.Equal(suite.T(), newUser.ID, result.User.ID)
	assert.Equal(suite.T(), oauthUser.Email, result.User.Email)

	suite.mockOAuthManager.AssertExpectations(suite.T())
	mockOAuthTokenRepo.AssertExpectations(suite.T())
	mockUserRepo.AssertExpectations(suite.T())
	suite.mockJWTManager.AssertExpectations(suite.T())
}

func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
