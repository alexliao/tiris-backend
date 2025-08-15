package middleware

import (
	"strconv"
	"time"

	"tiris-backend/pkg/monitoring"

	"github.com/gin-gonic/gin"
)

// MonitoringMiddleware creates middleware for request monitoring
func MonitoringMiddleware(metricsCollector *monitoring.MetricsCollector, logger *monitoring.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get user ID if available
		userID := ""
		if uid, exists := c.Get("user_id"); exists {
			userID = uid.(string)
		}

		// Get request ID if available
		requestID := ""
		if rid, exists := c.Get("request_id"); exists {
			requestID = rid.(string)
		}

		// Record metrics
		if metricsCollector != nil {
			metricsCollector.RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
		}

		// Log request
		if logger != nil {
			logger.LogHTTP(c.Request.Method, c.FullPath(), c.Writer.Status(), duration, userID, requestID)
		}
	})
}

// SecurityMonitoringMiddleware creates middleware for security event monitoring
func SecurityMonitoringMiddleware(metricsCollector *monitoring.MetricsCollector, logger *monitoring.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Check for suspicious patterns
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		// Rate limiting violations (if rate limiter sets a header)
		if rateLimitExceeded := c.GetHeader("X-Rate-Limit-Exceeded"); rateLimitExceeded == "true" {
			if metricsCollector != nil {
				metricsCollector.RecordSecurityEvent("rate_limit_exceeded", "warning")
			}
			if logger != nil {
				logger.LogSecurity("rate_limit_exceeded", "Rate limit exceeded", "", clientIP, map[string]interface{}{
					"user_agent": userAgent,
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
				})
			}
		}

		// Suspicious user agents
		if len(userAgent) == 0 {
			if metricsCollector != nil {
				metricsCollector.RecordSecurityEvent("missing_user_agent", "low")
			}
			if logger != nil {
				logger.LogSecurity("missing_user_agent", "Request without user agent", "", clientIP, map[string]interface{}{
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
				})
			}
		}

		c.Next()

		// Check response status for security events
		status := c.Writer.Status()
		if status == 401 || status == 403 {
			userID := ""
			if uid, exists := c.Get("user_id"); exists {
				userID = uid.(string)
			}

			eventType := "auth_failure"
			if status == 403 {
				eventType = "access_denied"
			}

			if metricsCollector != nil {
				metricsCollector.RecordSecurityEvent(eventType, "medium")
			}
			if logger != nil {
				logger.LogSecurity(eventType, "Authentication/authorization failure", userID, clientIP, map[string]interface{}{
					"status_code": status,
					"path":        c.Request.URL.Path,
					"method":      c.Request.Method,
					"user_agent":  userAgent,
				})
			}
		}
	})
}

// DatabaseMonitoringMiddleware creates middleware for database operation monitoring
func DatabaseMonitoringMiddleware(metricsCollector *monitoring.MetricsCollector) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Store start time for database operations
		c.Set("db_monitor_start", time.Now())
		c.Set("db_metrics_collector", metricsCollector)

		c.Next()
	})
}

// BusinessMetricsMiddleware creates middleware for business metrics tracking
func BusinessMetricsMiddleware(metricsCollector *monitoring.MetricsCollector) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Record business metrics based on successful operations
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			path := c.FullPath()
			method := c.Request.Method

			// User operations
			if method == "POST" && path == "/api/v1/auth/register" {
				metricsCollector.RecordBusinessEvent("user_registration")
			} else if method == "POST" && path == "/api/v1/auth/login" {
				metricsCollector.RecordBusinessEvent("user_login")
			}

			// Exchange operations
			if method == "POST" && path == "/api/v1/exchanges" {
				metricsCollector.RecordBusinessEvent("exchange_created")
			} else if method == "DELETE" && path == "/api/v1/exchanges/:id" {
				metricsCollector.RecordBusinessEvent("exchange_deleted")
			}

			// Sub-account operations
			if method == "POST" && path == "/api/v1/subaccounts" {
				metricsCollector.RecordBusinessEvent("subaccount_created")
			} else if method == "DELETE" && path == "/api/v1/subaccounts/:id" {
				metricsCollector.RecordBusinessEvent("subaccount_deleted")
			}

			// Transaction operations
			if method == "POST" && path == "/api/v1/transactions" {
				metricsCollector.RecordBusinessEvent("transaction_created")
			}

			// Trading log operations
			if method == "POST" && path == "/api/v1/trading-logs" {
				metricsCollector.RecordBusinessEvent("trading_log_created")
			}
		}
	})
}