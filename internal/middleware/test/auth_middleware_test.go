package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tiris-backend/internal/middleware"
	"tiris-backend/pkg/auth"
	"tiris-backend/test/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions

func setupAuthTest() (*gin.Engine, *auth.JWTManager) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create JWT manager for testing
	secretKey := "test-secret-key-for-auth-middleware-testing-only"
	refreshSecretKey := "test-refresh-secret-key-for-auth-middleware-testing-only"
	jwtManager := auth.NewJWTManager(secretKey, refreshSecretKey, time.Hour, 24*time.Hour)
	
	// Setup router
	router := gin.New()
	
	return router, jwtManager
}

func createTestToken(jwtManager *auth.JWTManager, userID uuid.UUID, username, email, role string) string {
	tokenPair, err := jwtManager.GenerateTokenPair(userID, username, email, role)
	if err != nil {
		panic(err)
	}
	return tokenPair.AccessToken
}

func createErrorResponse() map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    "",
			"message": "",
		},
	}
}

// Test AuthMiddleware
func TestAuthMiddleware(t *testing.T) {
	router, jwtManager := setupAuthTest()
	
	// Setup middleware and test endpoint
	router.Use(middleware.AuthMiddleware(jwtManager))
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		username, _ := middleware.GetUsername(c)
		email, _ := middleware.GetEmail(c)
		role, _ := middleware.GetRole(c)
		
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"email":    email,
			"role":     role,
		})
	})
	
	t.Run("successful_authentication", func(t *testing.T) {
		userID := uuid.New()
		token := createTestToken(jwtManager, userID, "testuser", "test@example.com", "user")
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, userID.String(), response["user_id"])
		assert.Equal(t, "testuser", response["username"])
		assert.Equal(t, "test@example.com", response["email"])
		assert.Equal(t, "user", response["role"])
	})
	
	t.Run("missing_authorization_header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "AUTH_REQUIRED", errorObj["code"])
		assert.Equal(t, "Authorization header is required", errorObj["message"])
	})
	
	t.Run("invalid_authorization_header_format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_TOKEN", errorObj["code"])
		assert.Equal(t, "Invalid authorization header format", errorObj["message"])
	})
	
	t.Run("expired_token", func(t *testing.T) {
		// Create JWT manager with very short expiry
		shortJWTManager := auth.NewJWTManager("test-secret", "test-refresh-secret", time.Nanosecond, time.Nanosecond)
		userID := uuid.New()
		
		tokenPair, err := shortJWTManager.GenerateTokenPair(userID, "testuser", "test@example.com", "user")
		require.NoError(t, err)
		
		// Wait for token to expire
		time.Sleep(2 * time.Millisecond)
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_TOKEN", errorObj["code"])
		assert.Equal(t, "Invalid or expired token", errorObj["message"])
	})
	
	t.Run("invalid_token_signature", func(t *testing.T) {
		// Create token with different JWT manager (different secret)
		otherJWTManager := auth.NewJWTManager("different-secret", "different-refresh-secret", time.Hour, 24*time.Hour)
		userID := uuid.New()
		token := createTestToken(otherJWTManager, userID, "testuser", "test@example.com", "user")
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_TOKEN", errorObj["code"])
	})
}

// Test OptionalAuthMiddleware
func TestOptionalAuthMiddleware(t *testing.T) {
	router, jwtManager := setupAuthTest()
	
	// Setup middleware and test endpoint
	router.Use(middleware.OptionalAuthMiddleware(jwtManager))
	router.GET("/optional", func(c *gin.Context) {
		authenticated := middleware.IsAuthenticated(c)
		var userID *uuid.UUID
		if id, exists := middleware.GetUserID(c); exists {
			userID = &id
		}
		
		c.JSON(http.StatusOK, gin.H{
			"authenticated": authenticated,
			"user_id":       userID,
		})
	})
	
	t.Run("authenticated_request", func(t *testing.T) {
		userID := uuid.New()
		token := createTestToken(jwtManager, userID, "testuser", "test@example.com", "user")
		
		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response["authenticated"].(bool))
		assert.Equal(t, userID.String(), response["user_id"])
	})
	
	t.Run("unauthenticated_request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/optional", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["authenticated"].(bool))
		assert.Nil(t, response["user_id"])
	})
	
	t.Run("invalid_token_continues", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["authenticated"].(bool))
		assert.Nil(t, response["user_id"])
	})
}

// Test AdminMiddleware
func TestAdminMiddleware(t *testing.T) {
	router, jwtManager := setupAuthTest()
	
	// Setup auth and admin middleware
	router.Use(middleware.OptionalAuthMiddleware(jwtManager))
	router.Use(middleware.AdminMiddleware())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})
	
	t.Run("admin_user_access", func(t *testing.T) {
		userID := uuid.New()
		token := createTestToken(jwtManager, userID, "admin", "admin@example.com", "admin")
		
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "admin access granted", response["message"])
	})
	
	t.Run("non_admin_user_denied", func(t *testing.T) {
		userID := uuid.New()
		token := createTestToken(jwtManager, userID, "user", "user@example.com", "user")
		
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "ACCESS_DENIED", errorObj["code"])
		assert.Equal(t, "Admin access required", errorObj["message"])
	})
	
	t.Run("unauthenticated_user_denied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "AUTH_REQUIRED", errorObj["code"])
		assert.Equal(t, "Authentication required", errorObj["message"])
	})
}

// Test RequireUserOwnership middleware
func TestRequireUserOwnership(t *testing.T) {
	router, jwtManager := setupAuthTest()
	
	// Setup auth and ownership middleware
	router.Use(middleware.OptionalAuthMiddleware(jwtManager))
	router.Use(middleware.RequireUserOwnership())
	router.GET("/users/:user_id/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})
	
	t.Run("user_accesses_own_resource", func(t *testing.T) {
		userID := uuid.New()
		token := createTestToken(jwtManager, userID, "user", "user@example.com", "user")
		
		req := httptest.NewRequest("GET", "/users/"+userID.String()+"/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "access granted", response["message"])
	})
	
	t.Run("admin_accesses_any_resource", func(t *testing.T) {
		adminID := uuid.New()
		otherUserID := uuid.New()
		token := createTestToken(jwtManager, adminID, "admin", "admin@example.com", "admin")
		
		req := httptest.NewRequest("GET", "/users/"+otherUserID.String()+"/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("user_denied_other_user_resource", func(t *testing.T) {
		userID := uuid.New()
		otherUserID := uuid.New()
		token := createTestToken(jwtManager, userID, "user", "user@example.com", "user")
		
		req := httptest.NewRequest("GET", "/users/"+otherUserID.String()+"/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "ACCESS_DENIED", errorObj["code"])
		assert.Equal(t, "Access denied to this resource", errorObj["message"])
	})
	
	t.Run("invalid_user_id_format", func(t *testing.T) {
		userID := uuid.New()
		token := createTestToken(jwtManager, userID, "user", "user@example.com", "user")
		
		req := httptest.NewRequest("GET", "/users/invalid-uuid/data", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_USER_ID", errorObj["code"])
		assert.Equal(t, "Invalid user ID format", errorObj["message"])
	})
	
	t.Run("unauthenticated_access_denied", func(t *testing.T) {
		userID := uuid.New()
		
		req := httptest.NewRequest("GET", "/users/"+userID.String()+"/data", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "AUTH_REQUIRED", errorObj["code"])
	})
}

// Test CORS middleware
func TestCORSMiddleware(t *testing.T) {
	router := gin.New()
	
	allowedOrigins := []string{"https://example.com", "https://test.com"}
	router.Use(middleware.CORSMiddleware(allowedOrigins))
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	t.Run("allowed_origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	})
	
	t.Run("disallowed_origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
	
	t.Run("wildcard_origin", func(t *testing.T) {
		wildcardRouter := gin.New()
		wildcardRouter.Use(middleware.CORSMiddleware([]string{"*"}))
		wildcardRouter.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://any-origin.com")
		
		w := httptest.NewRecorder()
		wildcardRouter.ServeHTTP(w, req)
		
		assert.Equal(t, "https://any-origin.com", w.Header().Get("Access-Control-Allow-Origin"))
	})
	
	t.Run("options_request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// Test RequestIDMiddleware
func TestRequestIDMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		requestID, _ := c.Get("request_id")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})
	
	t.Run("generates_request_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Check response header
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		
		// Validate it's a UUID format
		_, err := uuid.Parse(requestID)
		assert.NoError(t, err)
		
		// Check response body
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, requestID, response["request_id"])
	})
	
	t.Run("preserves_existing_request_id", func(t *testing.T) {
		existingID := "existing-request-id-123"
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, existingID, response["request_id"])
	})
}

// Test SecurityHeadersMiddleware
func TestSecurityHeadersMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(middleware.SecurityHeadersMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	t.Run("sets_security_headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Check all security headers
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
		assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	})
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	router, jwtManager := setupAuthTest()
	
	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"
	role := "user"
	
	router.Use(middleware.OptionalAuthMiddleware(jwtManager))
	router.GET("/test", func(c *gin.Context) {
		extractedUserID, userIDExists := middleware.GetUserID(c)
		extractedUsername, usernameExists := middleware.GetUsername(c)
		extractedEmail, emailExists := middleware.GetEmail(c)
		extractedRole, roleExists := middleware.GetRole(c)
		extractedClaims, claimsExist := middleware.GetClaims(c)
		authenticated := middleware.IsAuthenticated(c)
		
		var claimsUserID interface{}
		if claimsExist && extractedClaims != nil {
			claimsUserID = extractedClaims.UserID.String()
		} else {
			claimsUserID = nil
		}
		
		c.JSON(http.StatusOK, gin.H{
			"user_id":         extractedUserID,
			"user_id_exists":  userIDExists,
			"username":        extractedUsername,
			"username_exists": usernameExists,
			"email":           extractedEmail,
			"email_exists":    emailExists,
			"role":            extractedRole,
			"role_exists":     roleExists,
			"claims_exist":    claimsExist,
			"claims_user_id":  claimsUserID,
			"authenticated":   authenticated,
		})
	})
	
	t.Run("helper_functions_with_auth", func(t *testing.T) {
		token := createTestToken(jwtManager, userID, username, email, role)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, userID.String(), response["user_id"])
		assert.True(t, response["user_id_exists"].(bool))
		assert.Equal(t, username, response["username"])
		assert.True(t, response["username_exists"].(bool))
		assert.Equal(t, email, response["email"])
		assert.True(t, response["email_exists"].(bool))
		assert.Equal(t, role, response["role"])
		assert.True(t, response["role_exists"].(bool))
		assert.True(t, response["claims_exist"].(bool))
		assert.Equal(t, userID.String(), response["claims_user_id"])
		assert.True(t, response["authenticated"].(bool))
	})
	
	t.Run("helper_functions_without_auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "00000000-0000-0000-0000-000000000000", response["user_id"])
		assert.False(t, response["user_id_exists"].(bool))
		assert.Equal(t, "", response["username"])
		assert.False(t, response["username_exists"].(bool))
		assert.Equal(t, "", response["email"])
		assert.False(t, response["email_exists"].(bool))
		assert.Equal(t, "", response["role"])
		assert.False(t, response["role_exists"].(bool))
		assert.False(t, response["claims_exist"].(bool))
		assert.Nil(t, response["claims_user_id"])
		assert.False(t, response["authenticated"].(bool))
	})
}