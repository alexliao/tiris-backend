package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"tiris-backend/pkg/security"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecurityMiddleware provides comprehensive security features
type SecurityMiddleware struct {
	rateLimiter   *security.RateLimiter
	auditLogger   *security.AuditLogger
	apiKeyManager *security.APIKeyManager
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(db *gorm.DB, redisClient *redis.Client, masterKey, signingKey string) (*SecurityMiddleware, error) {
	apiKeyManager, err := security.NewAPIKeyManager(masterKey, signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key manager: %v", err)
	}

	return &SecurityMiddleware{
		rateLimiter:   security.NewRateLimiter(redisClient, "tiris:security:ratelimit"),
		auditLogger:   security.NewAuditLogger(db),
		apiKeyManager: apiKeyManager,
	}, nil
}

// APIKeyAuthMiddleware authenticates requests using API keys
func (sm *SecurityMiddleware) APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			sm.logAuthFailure(c, security.ActionAPIKeyUsed, "missing API key", startTime)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "API_KEY_REQUIRED",
					"message": "API key is required",
				},
			})
			c.Abort()
			return
		}

		// Validate API key format
		if err := sm.apiKeyManager.ValidateAPIKey(apiKey); err != nil {
			sm.logAuthFailure(c, security.ActionAPIKeyUsed, fmt.Sprintf("invalid API key format: %v", err), startTime)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_API_KEY",
					"message": "Invalid API key format",
				},
			})
			c.Abort()
			return
		}

		// Extract prefix to determine permissions
		prefix, err := sm.apiKeyManager.ExtractPrefix(apiKey)
		if err != nil {
			sm.logAuthFailure(c, security.ActionAPIKeyUsed, fmt.Sprintf("failed to extract prefix: %v", err), startTime)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_API_KEY",
					"message": "Invalid API key",
				},
			})
			c.Abort()
			return
		}

		// Set API key context
		c.Set("api_key", apiKey)
		c.Set("api_key_prefix", string(prefix))
		c.Set("api_key_hash", sm.apiKeyManager.HashAPIKey(apiKey))
		c.Set("auth_type", "api_key")

		// Log successful API key usage
		sm.logAuthSuccess(c, security.ActionAPIKeyUsed, nil, startTime)

		c.Next()
	}
}

// RateLimitMiddleware applies rate limiting based on configurable rules
func (sm *SecurityMiddleware) RateLimitMiddleware(ruleName string) gin.HandlerFunc {
	rules := security.DefaultRules()
	rule, exists := rules[ruleName]
	if !exists {
		rule = rules["api_general"] // Fallback to general rule
	}

	return func(c *gin.Context) {
		// Determine identifier for rate limiting
		identifier := sm.getRateLimitIdentifier(c)
		
		// Check rate limit
		result, err := sm.rateLimiter.CheckRateLimit(c.Request.Context(), identifier, ruleName, rule)
		if err != nil {
			// Log error but don't block the request
			sm.auditLogger.LogSecurityEvent(
				c.Request.Context(),
				security.ActionSystemError,
				nil,
				getClientIP(c.Request),
				map[string]interface{}{
					"rate_limit_error": err.Error(),
					"rule_name":        ruleName,
				},
				err,
			)
		} else {
			// Set rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rule.Limit))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime.Unix()))

			if !result.Allowed {
				// Log rate limit hit
				sm.auditLogger.LogSecurityEvent(
					c.Request.Context(),
					security.ActionRateLimitHit,
					getUserIDFromContext(c),
					getClientIP(c.Request),
					map[string]interface{}{
						"rule_name":     ruleName,
						"current_usage": result.CurrentUsage,
						"limit":         rule.Limit,
						"window":        rule.Window.String(),
					},
					nil,
				)

				c.Header("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "Rate limit exceeded",
						"retry_after": result.RetryAfter.Seconds(),
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// AuditMiddleware logs all requests for security auditing
func (sm *SecurityMiddleware) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Continue with request processing
		c.Next()
		
		// Log the request after completion
		duration := time.Since(startTime)
		success := c.Writer.Status() < 400
		
		action := sm.getAuditAction(c)
		userID := getUserIDFromContext(c)
		
		// Don't log health checks and metrics to reduce noise
		if strings.HasPrefix(c.Request.URL.Path, "/health") || 
		   strings.HasPrefix(c.Request.URL.Path, "/metrics") {
			return
		}

		sm.auditLogger.LogHTTPRequest(
			c.Request.Context(),
			c.Request,
			action,
			userID,
			success,
			duration,
			getLastError(c),
		)
	}
}

// CSRFMiddleware provides CSRF protection
func (sm *SecurityMiddleware) CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for GET, HEAD, OPTIONS, and API key authentication
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip for API key authentication
		if _, exists := c.Get("api_key"); exists {
			c.Next()
			return
		}

		// Check for CSRF token
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("_csrf_token")
		}

		if token == "" {
			sm.auditLogger.LogSecurityEvent(
				c.Request.Context(),
				security.ActionSecurityAlert,
				getUserIDFromContext(c),
				getClientIP(c.Request),
				map[string]interface{}{
					"alert_type": "missing_csrf_token",
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
				},
				nil,
			)

			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CSRF_TOKEN_REQUIRED",
					"message": "CSRF token is required",
				},
			})
			c.Abort()
			return
		}

		// TODO: Implement actual CSRF token validation
		// For now, just check that the token exists
		c.Next()
	}
}

// SecurityHeadersMiddleware adds comprehensive security headers
func (sm *SecurityMiddleware) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Strict Transport Security (HSTS)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		
		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'"
		c.Header("Content-Security-Policy", csp)
		
		// Permissions Policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// Server identification
		c.Header("Server", "Tiris-Backend/1.0")

		c.Next()
	}
}

// ThreatDetectionMiddleware detects and responds to potential threats
func (sm *SecurityMiddleware) ThreatDetectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c.Request)
		
		// Check for suspicious patterns
		threats := sm.detectThreats(c)
		
		for _, threat := range threats {
			sm.auditLogger.LogSecurityEvent(
				c.Request.Context(),
				security.ActionSecurityAlert,
				getUserIDFromContext(c),
				clientIP,
				map[string]interface{}{
					"threat_type":        threat.Type,
					"threat_severity":    threat.Severity,
					"threat_description": threat.Description,
					"user_agent":         c.Request.UserAgent(),
					"path":               c.Request.URL.Path,
				},
				nil,
			)

			// Block high-severity threats
			if threat.Severity == "critical" {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "THREAT_DETECTED",
						"message": "Request blocked due to security policy",
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// Helper methods

func (sm *SecurityMiddleware) getRateLimitIdentifier(c *gin.Context) string {
	// Prefer user ID if available
	if userID := getUserIDFromContext(c); userID != nil {
		return fmt.Sprintf("user:%s", userID.String())
	}

	// Fall back to IP address
	return fmt.Sprintf("ip:%s", getClientIP(c.Request))
}

func (sm *SecurityMiddleware) getAuditAction(c *gin.Context) security.AuditAction {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Map common paths to audit actions
	switch {
	case strings.Contains(path, "/auth/login"):
		return security.ActionLogin
	case strings.Contains(path, "/auth/logout"):
		return security.ActionLogout
	case strings.Contains(path, "/users") && method == "POST":
		return security.ActionUserCreate
	case strings.Contains(path, "/users") && method == "PUT":
		return security.ActionUserUpdate
	case strings.Contains(path, "/users") && method == "DELETE":
		return security.ActionUserDelete
	case strings.Contains(path, "/users") && method == "GET":
		return security.ActionUserView
	case strings.Contains(path, "/exchanges") && method == "POST":
		return security.ActionExchangeCreate
	case strings.Contains(path, "/exchanges") && method == "PUT":
		return security.ActionExchangeUpdate
	case strings.Contains(path, "/exchanges") && method == "DELETE":
		return security.ActionExchangeDelete
	case strings.Contains(path, "/exchanges") && method == "GET":
		return security.ActionExchangeView
	case strings.Contains(path, "/transactions") && method == "GET":
		return security.ActionTransactionView
	default:
		return security.ActionSystemAccess
	}
}

func (sm *SecurityMiddleware) logAuthSuccess(c *gin.Context, action security.AuditAction, userID *uuid.UUID, startTime time.Time) {
	duration := time.Since(startTime)
	sm.auditLogger.LogHTTPRequest(c.Request.Context(), c.Request, action, userID, true, duration, nil)
}

func (sm *SecurityMiddleware) logAuthFailure(c *gin.Context, action security.AuditAction, reason string, startTime time.Time) {
	details := map[string]interface{}{
		"failure_reason": reason,
		"duration_ms":    time.Since(startTime).Milliseconds(),
	}
	
	sm.auditLogger.LogSecurityEvent(
		c.Request.Context(),
		action,
		getUserIDFromContext(c),
		getClientIP(c.Request),
		details,
		fmt.Errorf("authentication failed: %s", reason),
	)
}

type Threat struct {
	Type        string
	Severity    string
	Description string
}

func (sm *SecurityMiddleware) detectThreats(c *gin.Context) []Threat {
	var threats []Threat
	
	userAgent := c.Request.UserAgent()
	path := c.Request.URL.Path
	
	// Detect common attack patterns
	if strings.Contains(strings.ToLower(userAgent), "sqlmap") ||
		strings.Contains(strings.ToLower(userAgent), "nikto") ||
		strings.Contains(strings.ToLower(userAgent), "nessus") {
		threats = append(threats, Threat{
			Type:        "malicious_user_agent",
			Severity:    "critical",
			Description: "Detected scanning/attack tool in user agent",
		})
	}
	
	// Detect SQL injection attempts
	suspiciousPatterns := []string{
		"union select", "drop table", "insert into", "delete from",
		"<script", "javascript:", "onerror=", "onload=",
	}
	
	queryString := strings.ToLower(c.Request.URL.RawQuery)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(queryString, pattern) {
			threats = append(threats, Threat{
				Type:        "injection_attempt",
				Severity:    "high",
				Description: fmt.Sprintf("Suspicious pattern detected in query: %s", pattern),
			})
			break
		}
	}
	
	// Detect directory traversal attempts
	if strings.Contains(path, "..") || strings.Contains(path, "%2e%2e") {
		threats = append(threats, Threat{
			Type:        "directory_traversal",
			Severity:    "high",
			Description: "Directory traversal attempt detected",
		})
	}
	
	return threats
}

func getUserIDFromContext(c *gin.Context) *uuid.UUID {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return &id
		}
	}
	return nil
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	return strings.Split(r.RemoteAddr, ":")[0]
}

func getLastError(c *gin.Context) error {
	if errors := c.Errors.Last(); errors != nil {
		return errors.Err
	}
	return nil
}