package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp    time.Time  `json:"timestamp"`
	RequestID    string     `json:"request_id"`
	Method       string     `json:"method"`
	Path         string     `json:"path"`
	Query        string     `json:"query,omitempty"`
	StatusCode   int        `json:"status_code"`
	Duration     float64    `json:"duration_ms"`
	ClientIP     string     `json:"client_ip"`
	UserAgent    string     `json:"user_agent"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Username     *string    `json:"username,omitempty"`
	RequestSize  int64      `json:"request_size"`
	ResponseSize int        `json:"response_size"`
	Error        *string    `json:"error,omitempty"`
	Level        string     `json:"level"`
	Service      string     `json:"service"`
}

// responseWriter wrapper to capture response size
type responseWriter struct {
	gin.ResponseWriter
	size int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) WriteString(s string) (int, error) {
	size, err := rw.ResponseWriter.WriteString(s)
	rw.size += size
	return size, err
}

// LoggingMiddleware creates a structured logging middleware
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)

		// Wrap response writer to capture response size
		rw := &responseWriter{
			ResponseWriter: c.Writer,
			size:           0,
		}
		c.Writer = rw

		// Get request size
		var requestSize int64
		if c.Request.Body != nil {
			// Read body to calculate size
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestSize = int64(len(bodyBytes))
				// Restore body for further processing
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get user info if authenticated
		var userID *uuid.UUID
		var username *string
		if id, exists := GetUserID(c); exists {
			userID = &id
		}
		if name, exists := GetUsername(c); exists {
			username = &name
		}

		// Get error if any
		var errorMsg *string
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Error()
			errorMsg = &err
		}

		// Determine log level based on status code
		level := "info"
		statusCode := c.Writer.Status()
		if statusCode >= 400 && statusCode < 500 {
			level = "warn"
		} else if statusCode >= 500 {
			level = "error"
		}

		// Create log entry
		logEntry := LogEntry{
			Timestamp:    start,
			RequestID:    requestID,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Query:        c.Request.URL.RawQuery,
			StatusCode:   statusCode,
			Duration:     float64(duration.Nanoseconds()) / 1000000, // Convert to milliseconds
			ClientIP:     c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			UserID:       userID,
			Username:     username,
			RequestSize:  requestSize,
			ResponseSize: rw.size,
			Error:        errorMsg,
			Level:        level,
			Service:      "tiris-backend",
		}

		// Log the entry as JSON
		logJSON, err := json.Marshal(logEntry)
		if err != nil {
			log.Printf("Failed to marshal log entry: %v", err)
			return
		}

		log.Println(string(logJSON))
	}
}

// ErrorLoggingMiddleware logs errors with more detail
func ErrorLoggingMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Get user info if authenticated
		var userID *uuid.UUID
		var username *string
		if id, exists := GetUserID(c); exists {
			userID = &id
		}
		if name, exists := GetUsername(c); exists {
			username = &name
		}

		// Create detailed error log entry
		errorEntry := map[string]interface{}{
			"timestamp":   time.Now(),
			"request_id":  requestID,
			"level":       "error",
			"service":     "tiris-backend",
			"type":        "panic_recovery",
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"client_ip":   c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"user_id":     userID,
			"username":    username,
			"panic_value": recovered,
		}

		// Log the error
		errorJSON, err := json.Marshal(errorEntry)
		if err != nil {
			log.Printf("Failed to marshal error entry: %v", err)
		} else {
			log.Println(string(errorJSON))
		}

		// Return 500 error
		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":       "INTERNAL_ERROR",
				"message":    "Internal server error",
				"request_id": requestID,
			},
		})
	})
}

// HealthCheckLoggingMiddleware provides minimal logging for health checks
func HealthCheckLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip detailed logging for health check endpoints
		if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
			c.Next()
			return
		}

		// Use standard logging for other endpoints
		LoggingMiddleware()(c)
	}
}
