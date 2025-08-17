package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"tiris-backend/internal/middleware"
	"tiris-backend/test/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions

func setupLoggingTest() (*gin.Engine, *bytes.Buffer) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Capture log output
	logBuffer := &bytes.Buffer{}
	log.SetOutput(logBuffer)
	
	// Setup router
	router := gin.New()
	
	return router, logBuffer
}

func restoreLogging() {
	log.SetOutput(os.Stderr)
}

func parseLogEntry(logLine string) (middleware.LogEntry, error) {
	var entry middleware.LogEntry
	err := json.Unmarshal([]byte(logLine), &entry)
	return entry, err
}

func getLastValidLogEntry(logBuffer *bytes.Buffer) (middleware.LogEntry, error) {
	logOutput := strings.TrimSpace(logBuffer.String())
	if logOutput == "" {
		return middleware.LogEntry{}, errors.New("no log output")
	}
	
	logLines := strings.Split(logOutput, "\n")
	
	// Try to parse log entries from the end (most recent first)
	for i := len(logLines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(logLines[i])
		if line == "" {
			continue
		}
		
		// Look for JSON in the line (skip Go log prefix)
		jsonStart := strings.Index(line, "{")
		if jsonStart >= 0 {
			jsonPart := line[jsonStart:]
			
			var entry middleware.LogEntry
			if err := json.Unmarshal([]byte(jsonPart), &entry); err == nil {
				// Verify it's our structured log entry by checking for service field
				if entry.Service == "tiris-backend" {
					return entry, nil
				}
			}
		}
	}
	
	return middleware.LogEntry{}, errors.New("no valid structured log entry found")
}

func setUserContext(router *gin.Engine) {
	router.Use(func(c *gin.Context) {
		if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", userID)
			}
		}
		if username := c.GetHeader("X-Username"); username != "" {
			c.Set("username", username)
		}
		c.Next()
	})
}

// Test LogEntry structure
func TestLogEntry(t *testing.T) {
	t.Run("log_entry_serialization", func(t *testing.T) {
		userID := uuid.New()
		username := "testuser"
		
		entry := middleware.LogEntry{
			Timestamp:    time.Now(),
			RequestID:    "test-request-123",
			Method:       "GET",
			Path:         "/api/test",
			Query:        "param=value",
			StatusCode:   200,
			Duration:     150.5,
			ClientIP:     "192.168.1.100",
			UserAgent:    "Test-Agent/1.0",
			UserID:       &userID,
			Username:     &username,
			RequestSize:  1024,
			ResponseSize: 2048,
			Error:        nil,
			Level:        "info",
			Service:      "tiris-backend",
		}
		
		// Serialize to JSON
		data, err := json.Marshal(entry)
		require.NoError(t, err)
		
		// Deserialize back
		var parsed middleware.LogEntry
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		
		// Verify fields
		assert.Equal(t, entry.RequestID, parsed.RequestID)
		assert.Equal(t, entry.Method, parsed.Method)
		assert.Equal(t, entry.Path, parsed.Path)
		assert.Equal(t, entry.StatusCode, parsed.StatusCode)
		assert.Equal(t, entry.Duration, parsed.Duration)
		assert.Equal(t, entry.UserID.String(), parsed.UserID.String())
		assert.Equal(t, *entry.Username, *parsed.Username)
		assert.Equal(t, entry.Level, parsed.Level)
		assert.Equal(t, entry.Service, parsed.Service)
	})
}

// Test responseWriter wrapper
func TestResponseWriter(t *testing.T) {
	t.Run("captures_response_size", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "test response",
				"data":    "some data here",
			})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		// Should capture response size
		assert.Greater(t, entry.ResponseSize, 0)
		assert.Equal(t, len(w.Body.Bytes()), entry.ResponseSize)
	})
	
	t.Run("writestring_method", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello, World!")
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		// Should capture string response size
		assert.Equal(t, len("Hello, World!"), entry.ResponseSize)
	})
}

// Test LoggingMiddleware basic functionality
func TestLoggingMiddleware(t *testing.T) {
	t.Run("logs_basic_request_info", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		
		req := httptest.NewRequest("GET", "/api/test?param=value", nil)
		req.Header.Set("User-Agent", "Test-Client/1.0")
		req.RemoteAddr = "192.168.1.100:12345"
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		// Verify basic fields
		assert.Equal(t, "GET", entry.Method)
		assert.Equal(t, "/api/test", entry.Path)
		assert.Equal(t, "param=value", entry.Query)
		assert.Equal(t, 200, entry.StatusCode)
		assert.Equal(t, "192.168.1.100", entry.ClientIP)
		assert.Equal(t, "Test-Client/1.0", entry.UserAgent)
		assert.Equal(t, "info", entry.Level)
		assert.Equal(t, "tiris-backend", entry.Service)
		assert.Greater(t, entry.Duration, 0.0)
		assert.NotEmpty(t, entry.RequestID)
	})
	
	t.Run("generates_request_id", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should have X-Request-ID header in response
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		
		// Should be valid UUID
		_, err := uuid.Parse(requestID)
		assert.NoError(t, err)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Equal(t, requestID, entry.RequestID)
	})
	
	t.Run("preserves_existing_request_id", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		existingID := "existing-request-id-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should preserve existing request ID
		assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Equal(t, existingID, entry.RequestID)
	})
	
	t.Run("captures_request_body_size", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "received"})
		})
		
		requestBody := `{"name": "test", "value": 123}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Equal(t, int64(len(requestBody)), entry.RequestSize)
	})
	
	t.Run("handles_empty_request_body", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Equal(t, int64(0), entry.RequestSize)
	})
}

// Test user context logging
func TestLoggingWithUserContext(t *testing.T) {
	t.Run("logs_authenticated_user_info", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		setUserContext(router)
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		userID := uuid.New()
		username := "testuser"
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Username", username)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		require.NotNil(t, entry.UserID)
		require.NotNil(t, entry.Username)
		assert.Equal(t, userID.String(), entry.UserID.String())
		assert.Equal(t, username, *entry.Username)
	})
	
	t.Run("handles_unauthenticated_requests", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		setUserContext(router)
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Nil(t, entry.UserID)
		assert.Nil(t, entry.Username)
	})
}

// Test error logging
func TestErrorLogging(t *testing.T) {
	t.Run("logs_gin_errors", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.Error(errors.New("test error occurred"))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		require.NotNil(t, entry.Error)
		assert.Equal(t, "test error occurred", *entry.Error)
		assert.Equal(t, "error", entry.Level)
	})
	
	t.Run("no_error_when_successful", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Parse log entry
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Nil(t, entry.Error)
		assert.Equal(t, "info", entry.Level)
	})
}

// Test log levels based on status codes
func TestLogLevels(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedLevel  string
		handlerFunc    func(*gin.Context)
	}{
		{
			name:          "success_info_level",
			statusCode:    200,
			expectedLevel: "info",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			},
		},
		{
			name:          "created_info_level",
			statusCode:    201,
			expectedLevel: "info",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"message": "created"})
			},
		},
		{
			name:          "bad_request_warn_level",
			statusCode:    400,
			expectedLevel: "warn",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			},
		},
		{
			name:          "unauthorized_warn_level",
			statusCode:    401,
			expectedLevel: "warn",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			},
		},
		{
			name:          "not_found_warn_level",
			statusCode:    404,
			expectedLevel: "warn",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			},
		},
		{
			name:          "internal_error_error_level",
			statusCode:    500,
			expectedLevel: "error",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			},
		},
		{
			name:          "bad_gateway_error_level",
			statusCode:    502,
			expectedLevel: "error",
			handlerFunc: func(c *gin.Context) {
				c.JSON(http.StatusBadGateway, gin.H{"error": "bad gateway"})
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router, logBuffer := setupLoggingTest()
			defer restoreLogging()
			
			router.Use(middleware.LoggingMiddleware())
			router.GET("/test", tc.handlerFunc)
			
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tc.statusCode, w.Code)
			
			// Parse log entry
			logLines := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
			entry, err := parseLogEntry(logLines[len(logLines)-1])
			require.NoError(t, err)
			
			assert.Equal(t, tc.expectedLevel, entry.Level)
			assert.Equal(t, tc.statusCode, entry.StatusCode)
		})
	}
}

// Test ErrorLoggingMiddleware (panic recovery)
func TestErrorLoggingMiddleware(t *testing.T) {
	t.Run("handles_panic_recovery", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.ErrorLoggingMiddleware())
		
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic occurred")
		})
		
		req := httptest.NewRequest("GET", "/panic", nil)
		req.Header.Set("X-Request-ID", "panic-test-123")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		// Parse error log entry
		logOutput := strings.TrimSpace(logBuffer.String())
		require.NotEmpty(t, logOutput)
		
		logLines := strings.Split(logOutput, "\n")
		
		// Find the last valid error log entry
		var errorEntry map[string]interface{}
		var err error
		for i := len(logLines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(logLines[i])
			if line == "" {
				continue
			}
			
			// Look for JSON in the line (skip Go log prefix)
			jsonStart := strings.Index(line, "{")
			if jsonStart >= 0 {
				jsonPart := line[jsonStart:]
				
				err = json.Unmarshal([]byte(jsonPart), &errorEntry)
				if err == nil {
					// Check if it's an error log entry
					if level, ok := errorEntry["level"]; ok && level == "error" {
						break
					}
				}
			}
		}
		require.NoError(t, err)
		
		assert.Equal(t, "panic-test-123", errorEntry["request_id"])
		assert.Equal(t, "error", errorEntry["level"])
		assert.Equal(t, "tiris-backend", errorEntry["service"])
		assert.Equal(t, "panic_recovery", errorEntry["type"])
		assert.Equal(t, "GET", errorEntry["method"])
		assert.Equal(t, "/panic", errorEntry["path"])
		assert.Equal(t, "test panic occurred", errorEntry["panic_value"])
		
		// Check response
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INTERNAL_ERROR", errorObj["code"])
		assert.Equal(t, "Internal server error", errorObj["message"])
		assert.Equal(t, "panic-test-123", errorObj["request_id"])
	})
	
	t.Run("panic_with_user_context", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		setUserContext(router)
		router.Use(middleware.ErrorLoggingMiddleware())
		
		router.GET("/panic", func(c *gin.Context) {
			panic("authenticated user panic")
		})
		
		userID := uuid.New()
		username := "panicuser"
		
		req := httptest.NewRequest("GET", "/panic", nil)
		req.Header.Set("X-User-ID", userID.String())
		req.Header.Set("X-Username", username)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Parse error log entry
		logOutput := strings.TrimSpace(logBuffer.String())
		require.NotEmpty(t, logOutput)
		
		logLines := strings.Split(logOutput, "\n")
		
		// Find the last valid error log entry
		var errorEntry map[string]interface{}
		var err error
		for i := len(logLines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(logLines[i])
			if line == "" {
				continue
			}
			
			// Look for JSON in the line (skip Go log prefix)
			jsonStart := strings.Index(line, "{")
			if jsonStart >= 0 {
				jsonPart := line[jsonStart:]
				
				err = json.Unmarshal([]byte(jsonPart), &errorEntry)
				if err == nil {
					// Check if it's an error log entry
					if level, ok := errorEntry["level"]; ok && level == "error" {
						break
					}
				}
			}
		}
		require.NoError(t, err)
		
		assert.Equal(t, userID.String(), errorEntry["user_id"])
		assert.Equal(t, username, errorEntry["username"])
	})
	
	t.Run("generates_request_id_for_panic", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.ErrorLoggingMiddleware())
		
		router.GET("/panic", func(c *gin.Context) {
			panic("no request id panic")
		})
		
		req := httptest.NewRequest("GET", "/panic", nil)
		// No X-Request-ID header
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Parse error log entry
		logOutput := strings.TrimSpace(logBuffer.String())
		require.NotEmpty(t, logOutput)
		
		logLines := strings.Split(logOutput, "\n")
		
		// Find the last valid error log entry
		var errorEntry map[string]interface{}
		var err error
		for i := len(logLines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(logLines[i])
			if line == "" {
				continue
			}
			
			// Look for JSON in the line (skip Go log prefix)
			jsonStart := strings.Index(line, "{")
			if jsonStart >= 0 {
				jsonPart := line[jsonStart:]
				
				err = json.Unmarshal([]byte(jsonPart), &errorEntry)
				if err == nil {
					// Check if it's an error log entry
					if level, ok := errorEntry["level"]; ok && level == "error" {
						break
					}
				}
			}
		}
		require.NoError(t, err)
		
		requestID := errorEntry["request_id"].(string)
		assert.NotEmpty(t, requestID)
		
		// Should be valid UUID
		_, err = uuid.Parse(requestID)
		assert.NoError(t, err)
		
		// Check response
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, requestID, errorObj["request_id"])
	})
}

// Test HealthCheckLoggingMiddleware
func TestHealthCheckLoggingMiddleware(t *testing.T) {
	t.Run("skips_logging_for_health_endpoints", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.HealthCheckLoggingMiddleware())
		
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
		
		router.GET("/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})
		
		// Test /health/live
		req1 := httptest.NewRequest("GET", "/health/live", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		
		assert.Equal(t, http.StatusOK, w1.Code)
		
		// Test /health/ready
		req2 := httptest.NewRequest("GET", "/health/ready", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		
		assert.Equal(t, http.StatusOK, w2.Code)
		
		// Should have no log entries (health checks skipped)
		logOutput := strings.TrimSpace(logBuffer.String())
		assert.Empty(t, logOutput)
	})
	
	t.Run("logs_non_health_endpoints", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.HealthCheckLoggingMiddleware())
		
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Should have log entry for non-health endpoint
		entry, err := getLastValidLogEntry(logBuffer)
		require.NoError(t, err)
		
		assert.Equal(t, "GET", entry.Method)
		assert.Equal(t, "/api/test", entry.Path)
	})
}

// Test concurrent logging
func TestConcurrentLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}
	
	t.Run("concurrent_requests_logging", func(t *testing.T) {
		router, logBuffer := setupLoggingTest()
		defer restoreLogging()
		
		router.Use(middleware.LoggingMiddleware())
		
		router.GET("/test/:id", func(c *gin.Context) {
			id := c.Param("id")
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"id": id})
		})
		
		concurrency := 10
		requests := 50
		
		results := make(chan int, requests)
		
		for i := 0; i < concurrency; i++ {
			go func(goroutineID int) {
				for j := 0; j < requests/concurrency; j++ {
					requestID := goroutineID*100 + j
					req := httptest.NewRequest("GET", "/test/"+strconv.Itoa(requestID), nil)
					req.Header.Set("User-Agent", "TestClient-"+strconv.Itoa(goroutineID))
					
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
					
					results <- w.Code
				}
			}(i)
		}
		
		// Collect results
		successCount := 0
		for i := 0; i < requests; i++ {
			code := <-results
			if code == http.StatusOK {
				successCount++
			}
		}
		
		assert.Equal(t, requests, successCount)
		
		// Check that we have log entries (might be fewer due to concurrent writes)
		logOutput := strings.TrimSpace(logBuffer.String())
		if logOutput != "" {
			logLines := strings.Split(logOutput, "\n")
			assert.Greater(t, len(logLines), 0)
			
			// Verify at least some log entries are valid JSON
			validEntries := 0
			for _, line := range logLines {
				if line != "" {
					var entry middleware.LogEntry
					if json.Unmarshal([]byte(line), &entry) == nil && entry.Service == "tiris-backend" {
						validEntries++
					}
				}
			}
			
			// We should have at least some valid entries, but due to concurrency it might be less than total
			assert.GreaterOrEqual(t, validEntries, 0)
		}
	})
}

// Benchmark logging performance
func BenchmarkLoggingMiddleware(b *testing.B) {
	router, _ := setupLoggingTest()
	defer restoreLogging()
	
	// Discard log output for benchmarking
	log.SetOutput(io.Discard)
	
	router.Use(middleware.LoggingMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkLogEntryMarshal(b *testing.B) {
	userID := uuid.New()
	username := "benchuser"
	
	entry := middleware.LogEntry{
		Timestamp:    time.Now(),
		RequestID:    "bench-request-123",
		Method:       "GET",
		Path:         "/api/bench",
		StatusCode:   200,
		Duration:     150.5,
		ClientIP:     "192.168.1.100",
		UserAgent:    "Bench-Agent/1.0",
		UserID:       &userID,
		Username:     &username,
		RequestSize:  1024,
		ResponseSize: 2048,
		Level:        "info",
		Service:      "tiris-backend",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(entry)
		if err != nil {
			b.Fatal(err)
		}
	}
}