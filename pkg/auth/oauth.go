package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuthProvider represents different OAuth providers
type OAuthProvider string

const (
	ProviderGoogle OAuthProvider = "google"
	ProviderWeChat OAuthProvider = "wechat"
)

// OAuthUser represents user information from OAuth provider
type OAuthUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Provider string `json:"provider"`
}

// GoogleUser represents Google OAuth user response
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// WeChatUser represents WeChat OAuth user response
type WeChatUser struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid"`
}

// WeChatTokenResponse represents WeChat access token response
type WeChatTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrorCode    int    `json:"errcode,omitempty"`
	ErrorMsg     string `json:"errmsg,omitempty"`
}

// OAuthConfig represents OAuth configuration
type OAuthConfig struct {
	Google GoogleOAuthConfig
	WeChat WeChatOAuthConfig
}

// GoogleOAuthConfig represents Google OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	config       *oauth2.Config
}

// WeChatOAuthConfig represents WeChat OAuth configuration
type WeChatOAuthConfig struct {
	AppID       string
	AppSecret   string
	RedirectURL string
}

// OAuthManager manages OAuth operations
type OAuthManager struct {
	google GoogleOAuthConfig
	wechat WeChatOAuthConfig
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(config OAuthConfig) *OAuthManager {
	manager := &OAuthManager{
		google: config.Google,
		wechat: config.WeChat,
	}

	// Initialize Google OAuth config
	manager.google.config = &oauth2.Config{
		ClientID:     config.Google.ClientID,
		ClientSecret: config.Google.ClientSecret,
		RedirectURL:  config.Google.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return manager
}

// GetAuthURL generates OAuth authorization URL
func (om *OAuthManager) GetAuthURL(provider OAuthProvider, state string) (string, error) {
	switch provider {
	case ProviderGoogle:
		return om.google.config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
	case ProviderWeChat:
		return om.getWeChatAuthURL(state), nil
	default:
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// ExchangeCodeForToken exchanges authorization code for access token
func (om *OAuthManager) ExchangeCodeForToken(provider OAuthProvider, code string) (*oauth2.Token, error) {
	ctx := context.Background()

	switch provider {
	case ProviderGoogle:
		return om.google.config.Exchange(ctx, code)
	case ProviderWeChat:
		return om.exchangeWeChatCode(code)
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// GetUserInfo retrieves user information using access token
func (om *OAuthManager) GetUserInfo(provider OAuthProvider, token *oauth2.Token) (*OAuthUser, error) {
	switch provider {
	case ProviderGoogle:
		return om.getGoogleUserInfo(token)
	case ProviderWeChat:
		return om.getWeChatUserInfo(token)
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// getGoogleUserInfo retrieves Google user information
func (om *OAuthManager) getGoogleUserInfo(token *oauth2.Token) (*OAuthUser, error) {
	ctx := context.Background()
	client := om.google.config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get Google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Google user info response: %w", err)
	}

	var googleUser GoogleUser
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse Google user info: %w", err)
	}

	// Validate required fields
	if googleUser.ID == "" || googleUser.Email == "" {
		return nil, errors.New("incomplete Google user information")
	}

	return &OAuthUser{
		ID:       googleUser.ID,
		Email:    googleUser.Email,
		Name:     googleUser.Name,
		Avatar:   googleUser.Picture,
		Provider: string(ProviderGoogle),
	}, nil
}

// getWeChatAuthURL generates WeChat OAuth authorization URL
func (om *OAuthManager) getWeChatAuthURL(state string) string {
	baseURL := "https://open.weixin.qq.com/connect/oauth2/authorize"
	params := url.Values{
		"appid":         {om.wechat.AppID},
		"redirect_uri":  {om.wechat.RedirectURL},
		"response_type": {"code"},
		"scope":         {"snsapi_userinfo"},
		"state":         {state},
	}

	return fmt.Sprintf("%s?%s#wechat_redirect", baseURL, params.Encode())
}

// exchangeWeChatCode exchanges WeChat authorization code for access token
func (om *OAuthManager) exchangeWeChatCode(code string) (*oauth2.Token, error) {
	tokenURL := "https://api.weixin.qq.com/sns/oauth2/access_token"
	params := url.Values{
		"appid":      {om.wechat.AppID},
		"secret":     {om.wechat.AppSecret},
		"code":       {code},
		"grant_type": {"authorization_code"},
	}

	resp, err := http.Get(fmt.Sprintf("%s?%s", tokenURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to exchange WeChat code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WeChat API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read WeChat token response: %w", err)
	}

	var tokenResp WeChatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse WeChat token response: %w", err)
	}

	// Check for errors
	if tokenResp.ErrorCode != 0 {
		return nil, fmt.Errorf("WeChat API error %d: %s", tokenResp.ErrorCode, tokenResp.ErrorMsg)
	}

	// Convert to oauth2.Token
	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	// Store OpenID in token extra data
	token = token.WithExtra(map[string]interface{}{
		"openid": tokenResp.OpenID,
	})

	return token, nil
}

// getWeChatUserInfo retrieves WeChat user information
func (om *OAuthManager) getWeChatUserInfo(token *oauth2.Token) (*OAuthUser, error) {
	// Get OpenID from token
	openID, ok := token.Extra("openid").(string)
	if !ok || openID == "" {
		return nil, errors.New("missing OpenID in WeChat token")
	}

	userInfoURL := "https://api.weixin.qq.com/sns/userinfo"
	params := url.Values{
		"access_token": {token.AccessToken},
		"openid":       {openID},
		"lang":         {"zh_CN"},
	}

	resp, err := http.Get(fmt.Sprintf("%s?%s", userInfoURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to get WeChat user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WeChat API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read WeChat user info response: %w", err)
	}

	var wechatUser WeChatUser
	if err := json.Unmarshal(body, &wechatUser); err != nil {
		return nil, fmt.Errorf("failed to parse WeChat user info: %w", err)
	}

	// Validate required fields
	if wechatUser.OpenID == "" {
		return nil, errors.New("incomplete WeChat user information")
	}

	// Generate email since WeChat doesn't provide email
	email := fmt.Sprintf("%s@wechat.local", wechatUser.OpenID)

	return &OAuthUser{
		ID:       wechatUser.OpenID,
		Email:    email,
		Name:     wechatUser.Nickname,
		Avatar:   wechatUser.HeadImgURL,
		Provider: string(ProviderWeChat),
	}, nil
}

// ValidateState validates OAuth state parameter to prevent CSRF attacks
func ValidateState(expectedState, actualState string) error {
	if expectedState == "" {
		return errors.New("expected state is empty")
	}
	if actualState == "" {
		return errors.New("actual state is empty")
	}
	if expectedState != actualState {
		return errors.New("state parameter mismatch")
	}
	return nil
}

// GenerateState generates a random state parameter for OAuth
func GenerateState() string {
	// Generate a random UUID as state
	return strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), "-", "")
}

// RefreshWeChatToken refreshes WeChat access token
func (om *OAuthManager) RefreshWeChatToken(refreshToken string) (*oauth2.Token, error) {
	refreshURL := "https://api.weixin.qq.com/sns/oauth2/refresh_token"
	params := url.Values{
		"appid":         {om.wechat.AppID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	resp, err := http.Get(fmt.Sprintf("%s?%s", refreshURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh WeChat token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WeChat API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read WeChat refresh response: %w", err)
	}

	var tokenResp WeChatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse WeChat refresh response: %w", err)
	}

	// Check for errors
	if tokenResp.ErrorCode != 0 {
		return nil, fmt.Errorf("WeChat API error %d: %s", tokenResp.ErrorCode, tokenResp.ErrorMsg)
	}

	// Convert to oauth2.Token
	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	// Store OpenID in token extra data
	token = token.WithExtra(map[string]interface{}{
		"openid": tokenResp.OpenID,
	})

	return token, nil
}