package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/pkg/auth"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// AuthService handles authentication business logic
type AuthService struct {
	repos        *repositories.Repositories
	jwtManager   auth.JWTManagerInterface
	oauthManager auth.OAuthManagerInterface
}

// NewAuthService creates a new authentication service
func NewAuthService(repos *repositories.Repositories, jwtManager auth.JWTManagerInterface, oauthManager auth.OAuthManagerInterface) *AuthService {
	return &AuthService{
		repos:        repos,
		jwtManager:   jwtManager,
		oauthManager: oauthManager,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Provider    string `json:"provider" binding:"required,oneof=google wechat"`
	RedirectURL string `json:"redirect_uri" binding:"required,url"`
}

// LoginResponse represents a login response with auth URL
type LoginResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// CallbackRequest represents an OAuth callback request
type CallbackRequest struct {
	Provider string `json:"provider" binding:"required,oneof=google wechat"`
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
}

// AuthResponse represents an authentication response with tokens
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

// UserInfo represents user information in auth response
type UserInfo struct {
	ID       uuid.UUID              `json:"id"`
	Username string                 `json:"username"`
	Email    string                 `json:"email"`
	Avatar   *string                `json:"avatar,omitempty"`
	Info     map[string]interface{} `json:"info"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// InitiateLogin initiates OAuth login flow
func (s *AuthService) InitiateLogin(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Generate state for CSRF protection
	state := auth.GenerateState()

	// Get OAuth authorization URL
	authURL, err := s.oauthManager.GetAuthURL(auth.OAuthProvider(req.Provider), state)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth URL: %w", err)
	}

	return &LoginResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// HandleCallback handles OAuth callback and creates/updates user
func (s *AuthService) HandleCallback(ctx context.Context, req *CallbackRequest, expectedState string) (*AuthResponse, error) {
	// Validate state to prevent CSRF attacks
	if err := auth.ValidateState(expectedState, req.State); err != nil {
		return nil, fmt.Errorf("invalid state parameter: %w", err)
	}

	// Exchange code for token
	token, err := s.oauthManager.ExchangeCodeForToken(auth.OAuthProvider(req.Provider), req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from OAuth provider
	oauthUser, err := s.oauthManager.GetUserInfo(auth.OAuthProvider(req.Provider), token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, oauthUser, token)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Generate JWT tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username, user.Email, "user")
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Convert user info
	var avatar *string
	if user.Avatar != nil && *user.Avatar != "" {
		avatar = user.Avatar
	}

	var userInfoMap map[string]interface{}
	if len(user.Info) > 0 {
		userInfoMap = user.Info
	} else {
		userInfoMap = make(map[string]interface{})
	}

	userInfo := &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   avatar,
		Info:     userInfoMap,
	}

	return &AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         userInfo,
	}, nil
}

// RefreshToken refreshes an access token using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, req *RefreshRequest) (*AuthResponse, error) {
	// Validate refresh token and get user ID
	userID, err := s.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user from database
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new access token
	accessToken, err := s.jwtManager.RefreshToken(req.RefreshToken, user.Username, user.Email, "user")
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Convert user info
	var avatar *string
	if user.Avatar != nil && *user.Avatar != "" {
		avatar = user.Avatar
	}

	var userInfoMap map[string]interface{}
	if len(user.Info) > 0 {
		userInfoMap = user.Info
	} else {
		userInfoMap = make(map[string]interface{})
	}

	userInfo := &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   avatar,
		Info:     userInfoMap,
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken, // Keep the same refresh token
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		User:         userInfo,
	}, nil
}

// Logout invalidates user session
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	// In a more sophisticated implementation, we might maintain a blacklist of tokens
	// For now, we just return success since JWT tokens are stateless
	// The client should discard the tokens
	return nil
}

// findOrCreateUser finds existing user or creates new one from OAuth info
func (s *AuthService) findOrCreateUser(ctx context.Context, oauthUser *auth.OAuthUser, token *oauth2.Token) (*models.User, error) {
	// Check if OAuth token already exists
	existingToken, err := s.repos.OAuthToken.GetByProviderUserID(ctx, oauthUser.Provider, oauthUser.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing OAuth token: %w", err)
	}

	if existingToken != nil {
		// User exists, update OAuth token and return user
		existingToken.AccessToken = token.AccessToken
		if token.RefreshToken != "" {
			existingToken.RefreshToken = &token.RefreshToken
		}
		if !token.Expiry.IsZero() {
			existingToken.ExpiresAt = &token.Expiry
		}
		existingToken.UpdatedAt = time.Now()

		// Update OAuth token info
		infoMap := map[string]interface{}{
			"last_login": time.Now(),
			"provider_data": map[string]interface{}{
				"name":   oauthUser.Name,
				"avatar": oauthUser.Avatar,
			},
		}
		existingToken.Info = models.JSON(infoMap)

		if err := s.repos.OAuthToken.Update(ctx, existingToken); err != nil {
			return nil, fmt.Errorf("failed to update OAuth token: %w", err)
		}

		// Get and return the user
		user, err := s.repos.User.GetByID(ctx, existingToken.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing user: %w", err)
		}

		// Update user avatar if provided
		if oauthUser.Avatar != "" && (user.Avatar == nil || *user.Avatar != oauthUser.Avatar) {
			user.Avatar = &oauthUser.Avatar
			if err := s.repos.User.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to update user avatar: %w", err)
			}
		}

		return user, nil
	}

	// Check if user exists by email
	existingUser, err := s.repos.User.GetByEmail(ctx, oauthUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	var user *models.User
	if existingUser != nil {
		// User exists with this email, link OAuth account
		user = existingUser
	} else {
		// Create new user
		user = &models.User{
			Username: s.generateUsername(oauthUser.Name, oauthUser.Email),
			Email:    oauthUser.Email,
			Avatar:   &oauthUser.Avatar,
			Settings: models.JSON{},
			Info: func() models.JSON {
				infoMap := map[string]interface{}{
					"oauth_provider": oauthUser.Provider,
					"created_via":    "oauth",
					"first_login":    time.Now(),
				}
				return models.JSON(infoMap)
			}(),
		}

		if err := s.repos.User.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Create OAuth token record
	oauthToken := &models.OAuthToken{
		UserID:         user.ID,
		Provider:       oauthUser.Provider,
		ProviderUserID: oauthUser.ID,
		AccessToken:    token.AccessToken,
		RefreshToken:   &token.RefreshToken,
		ExpiresAt:      &token.Expiry,
		Info: func() models.JSON {
			infoMap := map[string]interface{}{
				"provider_data": map[string]interface{}{
					"name":   oauthUser.Name,
					"avatar": oauthUser.Avatar,
				},
				"first_auth": time.Now(),
			}
			return models.JSON(infoMap)
		}(),
	}

	if err := s.repos.OAuthToken.Create(ctx, oauthToken); err != nil {
		return nil, fmt.Errorf("failed to create OAuth token: %w", err)
	}

	return user, nil
}

// generateUsername generates a unique username from name and email
func (s *AuthService) generateUsername(name, email string) string {
	// Start with name if available
	if name != "" {
		// Clean name: remove spaces, convert to lowercase
		username := ""
		for _, char := range name {
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
				username += string(char)
			}
		}
		if len(username) >= 3 {
			return username[:min(len(username), 20)]
		}
	}

	// Fall back to email prefix
	if email != "" {
		atIndex := strings.Index(email, "@")
		if atIndex > 0 {
			parts := email[:atIndex]
			if len(parts) >= 3 {
				return parts[:min(len(parts), 20)]
			}
		}
	}

	// Last resort: generate random username
	return fmt.Sprintf("user%d", time.Now().Unix())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
