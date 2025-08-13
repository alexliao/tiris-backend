package auth

import (
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// JWTManagerInterface defines the interface for JWT operations
type JWTManagerInterface interface {
	GenerateToken(userID uuid.UUID, username, email, role string) (string, error)
	GenerateTokenPair(userID uuid.UUID, username, email, role string) (*TokenPair, error)
	ValidateToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(tokenString string) (uuid.UUID, error)
	RefreshToken(refreshToken, username, email, role string) (string, error)
}

// OAuthManagerInterface defines the interface for OAuth operations
type OAuthManagerInterface interface {
	GetAuthURL(provider OAuthProvider, state string) (string, error)
	ExchangeCodeForToken(provider OAuthProvider, code string) (*oauth2.Token, error)
	GetUserInfo(provider OAuthProvider, token *oauth2.Token) (*OAuthUser, error)
}
