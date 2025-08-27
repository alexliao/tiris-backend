package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"tiris-backend/internal/middleware"
	"tiris-backend/test/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test helper functions

func setupRateLimitTest() *gin.Engine {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Setup router
	router := gin.New()
	
	return router
}

func createTestRateLimitConfig(requestsPerHour int, keyFunc func(*gin.Context) string) middleware.RateLimitConfig {
	config := middleware.RateLimitConfig{
		RequestsPerHour: requestsPerHour,
		WindowDuration:  time.Hour,
		KeyFunc:         keyFunc,
	}
	
	if keyFunc == nil {
		config.KeyFunc = middleware.DefaultKeyFunc
	}
	
	return config
}

// Test RateLimiter core functionality
func TestRateLimiter(t *testing.T) {
	t.Run("basic_token_bucket", func(t *testing.T) {
		limiter := middleware.NewRateLimiter(5, time.Second)
		
		// Should allow 5 requests initially
		for i := 0; i < 5; i++ {
			assert.True(t, limiter.Allow(), "Request %d should be allowed", i+1)
		}
		
		// 6th request should be denied
		assert.False(t, limiter.Allow(), "6th request should be denied")
		
		// Check token count
		assert.Equal(t, 0, limiter.GetTokens())
	})
	
	t.Run("token_refill", func(t *testing.T) {
		limiter := middleware.NewRateLimiter(2, 100*time.Millisecond)
		
		// Use all tokens
		assert.True(t, limiter.Allow())
		assert.True(t, limiter.Allow())
		assert.False(t, limiter.Allow())
		
		// Wait for refill
		time.Sleep(250 * time.Millisecond) // Should refill 2 tokens
		
		// Should have tokens again
		assert.True(t, limiter.Allow())
		assert.True(t, limiter.Allow())
		assert.False(t, limiter.Allow())
	})
	
	t.Run("token_cap", func(t *testing.T) {
		limiter := middleware.NewRateLimiter(3, 10*time.Millisecond)
		
		// Wait for potential overflow
		time.Sleep(100 * time.Millisecond)
		
		// Should still only have max tokens
		assert.LessOrEqual(t, limiter.GetTokens(), 3)
		
		// Should only allow max requests
		allowedCount := 0
		for i := 0; i < 10; i++ {
			if limiter.Allow() {
				allowedCount++
			}
		}
		assert.Equal(t, 3, allowedCount)
	})
}

// Test RateLimitStore functionality
func TestRateLimitStore(t *testing.T) {
	t.Run("create_limiters_per_key", func(t *testing.T) {
		store := middleware.NewRateLimitStore(5, time.Second)
		
		// Get limiters for different keys
		limiter1 := store.GetLimiter("key1")
		limiter2 := store.GetLimiter("key2")
		limiter3 := store.GetLimiter("key1") // Same as limiter1
		
		// Should be different instances for different keys
		assert.NotEqual(t, limiter1, limiter2)
		
		// Should be same instance for same key
		assert.Equal(t, limiter1, limiter3)
	})
	
	t.Run("independent_rate_limiting", func(t *testing.T) {
		store := middleware.NewRateLimitStore(2, time.Second)
		
		limiter1 := store.GetLimiter("user1")
		limiter2 := store.GetLimiter("user2")
		
		// Use all tokens for user1
		assert.True(t, limiter1.Allow())
		assert.True(t, limiter1.Allow())
		assert.False(t, limiter1.Allow())
		
		// User2 should still have tokens
		assert.True(t, limiter2.Allow())
		assert.True(t, limiter2.Allow())
		assert.False(t, limiter2.Allow())
	})
	
	t.Run("concurrent_access", func(t *testing.T) {
		store := middleware.NewRateLimitStore(100, time.Millisecond)
		
		var wg sync.WaitGroup
		results := make([]bool, 200)
		
		// 200 concurrent requests for same key
		for i := 0; i < 200; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				limiter := store.GetLimiter("concurrent-test")
				results[idx] = limiter.Allow()
			}(i)
		}
		
		wg.Wait()
		
		// Count allowed requests
		allowedCount := 0
		for _, allowed := range results {
			if allowed {
				allowedCount++
			}
		}
		
		// Should allow at most 100 requests
		assert.LessOrEqual(t, allowedCount, 100)
		assert.Greater(t, allowedCount, 0)
	})
}

// Test key generation functions
func TestKeyFunctions(t *testing.T) {
	router := setupRateLimitTest()
	
	t.Run("default_key_func", func(t *testing.T) {
		router.GET("/test", func(c *gin.Context) {
			key := middleware.DefaultKeyFunc(c)
			c.JSON(http.StatusOK, gin.H{"key": key})
		})
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		// Key should be based on IP
		assert.Contains(t, w.Body.String(), "192.168.1.100")
	})
	
	t.Run("user_key_func", func(t *testing.T) {
		userID := uuid.New()
		
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		
		router.GET("/test-user", func(c *gin.Context) {
			key := middleware.UserKeyFunc(c)
			c.JSON(http.StatusOK, gin.H{"key": key})
		})
		
		req := httptest.NewRequest("GET", "/test-user", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), fmt.Sprintf("user:%s", userID.String()))
	})
	
	t.Run("endpoint_key_func", func(t *testing.T) {
		userID := uuid.New()
		
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		
		router.GET("/api/trades/:id", func(c *gin.Context) {
			key := middleware.EndpointKeyFunc(c)
			c.JSON(http.StatusOK, gin.H{"key": key})
		})
		
		req := httptest.NewRequest("GET", "/api/trades/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		expected := fmt.Sprintf("GET:/api/trades/:id:user:%s", userID.String())
		assert.Contains(t, w.Body.String(), expected)
	})
}

// Test RateLimitMiddleware
func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows_requests_within_limit", func(t *testing.T) {
		router := setupRateLimitTest()
		
		config := createTestRateLimitConfig(5, nil) // 5 requests per hour
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		// Should allow first 5 requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should be allowed", i+1)
			
			// Check rate limit headers
			assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
			assert.Equal(t, "3600", w.Header().Get("X-RateLimit-Window"))
			
			remaining := w.Header().Get("X-RateLimit-Remaining")
			expectedRemaining := strconv.Itoa(4 - i)
			assert.Equal(t, expectedRemaining, remaining)
		}
	})
	
	t.Run("denies_requests_over_limit", func(t *testing.T) {
		router := setupRateLimitTest()
		
		config := createTestRateLimitConfig(3, nil) // 3 requests per hour
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		clientIP := "192.168.1.200:54321"
		
		// Use up the limit
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = clientIP
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
		}
		
		// Next request should be denied
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = clientIP
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		
		// Check rate limit headers
		assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
		
		// Check error response
		assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w.Body.String(), "Rate limit exceeded")
	})
	
	t.Run("independent_limits_per_client", func(t *testing.T) {
		router := setupRateLimitTest()
		
		config := createTestRateLimitConfig(2, nil) // 2 requests per hour
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		// Client 1 uses up limit
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
		}
		
		// Client 1 should be denied
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.1:12345"
		
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		
		assert.Equal(t, http.StatusTooManyRequests, w1.Code)
		
		// Client 2 should still be allowed
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.2:12345"
		
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		
		assert.Equal(t, http.StatusOK, w2.Code)
	})
	
	t.Run("custom_key_function", func(t *testing.T) {
		router := setupRateLimitTest()
		
		// Use custom key function that always returns same key
		config := createTestRateLimitConfig(1, func(c *gin.Context) string {
			return "global-limit"
		})
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		// First request from any IP should be allowed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "1.1.1.1:12345"
		
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		
		assert.Equal(t, http.StatusOK, w1.Code)
		
		// Second request from different IP should be denied (same key)
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "2.2.2.2:12345"
		
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	})
}

// Test predefined middleware configurations
func TestPredefinedRateLimitMiddleware(t *testing.T) {
	t.Run("auth_rate_limit_middleware", func(t *testing.T) {
		router := setupRateLimitTest()
		router.Use(middleware.AuthRateLimitMiddleware())
		
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "logged in"})
		})
		
		clientIP := "192.168.1.100:12345"
		
		// Test that auth endpoints have appropriate limits
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = clientIP
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Check rate limit headers indicate auth limits (60 per hour)
		assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "3600", w.Header().Get("X-RateLimit-Window"))
	})
	
	t.Run("api_rate_limit_middleware", func(t *testing.T) {
		router := setupRateLimitTest()
		
		// Simulate authenticated user
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		
		router.Use(middleware.APIRateLimitMiddleware())
		
		router.GET("/api/data", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "api data"})
		})
		
		req := httptest.NewRequest("GET", "/api/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Check rate limit headers indicate API limits (1000 per hour)
		assert.Equal(t, "1000", w.Header().Get("X-RateLimit-Limit"))
	})
	
	t.Run("trading_rate_limit_middleware", func(t *testing.T) {
		router := setupRateLimitTest()
		
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		
		router.Use(middleware.TradingRateLimitMiddleware())
		
		router.POST("/api/trades", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "trade executed"})
		})
		
		req := httptest.NewRequest("POST", "/api/trades", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Check rate limit headers indicate trading limits (600 per hour)
		assert.Equal(t, "600", w.Header().Get("X-RateLimit-Limit"))
	})
}

// Test rate limit recovery
func TestRateLimitRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping recovery test in short mode")
	}
	
	t.Run("tokens_refill_over_time", func(t *testing.T) {
		router := setupRateLimitTest()
		
		// Very short refill time for testing
		config := middleware.RateLimitConfig{
			RequestsPerHour: 2,
			WindowDuration:  200 * time.Millisecond, // Very fast refill for testing
			KeyFunc:         middleware.DefaultKeyFunc,
		}
		
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		clientIP := "192.168.1.100:12345"
		
		// Use up the limit
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = clientIP
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
		}
		
		// Should be denied now
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = clientIP
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		
		// Wait for refill
		time.Sleep(300 * time.Millisecond)
		
		// Should be allowed again
		req = httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = clientIP
		
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Performance tests for rate limiting
func TestRateLimitPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	t.Run("concurrent_rate_limiting", func(t *testing.T) {
		router := setupRateLimitTest()
		
		config := createTestRateLimitConfig(1000, middleware.DefaultKeyFunc)
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		// 100 concurrent requests from different IPs
		concurrency := 100
		results := make(chan int, concurrency)
		
		var wg sync.WaitGroup
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				// Use different IP for each goroutine
				clientIP := fmt.Sprintf("192.168.1.%d:12345", id%254+1)
				
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = clientIP
				
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				
				results <- w.Code
			}(i)
		}
		
		wg.Wait()
		close(results)
		
		// Count successful requests
		successCount := 0
		for code := range results {
			if code == http.StatusOK {
				successCount++
			}
		}
		
		// All should succeed since different IPs
		assert.Equal(t, concurrency, successCount)
	})
	
	t.Run("high_frequency_requests", func(t *testing.T) {
		router := setupRateLimitTest()
		
		config := createTestRateLimitConfig(50, middleware.DefaultKeyFunc)
		router.Use(middleware.RateLimitMiddleware(config))
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		clientIP := "192.168.1.100:12345"
		
		// Send 100 requests rapidly from same IP
		successCount := 0
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = clientIP
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code == http.StatusOK {
				successCount++
			}
		}
		
		// Should allow exactly 50 requests
		assert.Equal(t, 50, successCount)
	})
}

// Benchmark rate limiting performance
func BenchmarkRateLimitMiddleware(b *testing.B) {
	router := setupRateLimitTest()
	
	config := createTestRateLimitConfig(10000, middleware.DefaultKeyFunc)
	router.Use(middleware.RateLimitMiddleware(config))
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	limiter := middleware.NewRateLimiter(10000, time.Nanosecond)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

// Test environment variable configuration
func TestEnvironmentVariableConfiguration(t *testing.T) {
	// Save original environment variables
	originalRateLimitEnabled := os.Getenv("RATE_LIMIT_ENABLED")
	originalAPIRateLimit := os.Getenv("API_RATE_LIMIT_PER_HOUR")
	originalAuthRateLimit := os.Getenv("AUTH_RATE_LIMIT_PER_HOUR")
	originalTradingRateLimit := os.Getenv("TRADING_RATE_LIMIT_PER_HOUR")
	
	// Cleanup function to restore original values
	cleanup := func() {
		if originalRateLimitEnabled == "" {
			os.Unsetenv("RATE_LIMIT_ENABLED")
		} else {
			os.Setenv("RATE_LIMIT_ENABLED", originalRateLimitEnabled)
		}
		if originalAPIRateLimit == "" {
			os.Unsetenv("API_RATE_LIMIT_PER_HOUR")
		} else {
			os.Setenv("API_RATE_LIMIT_PER_HOUR", originalAPIRateLimit)
		}
		if originalAuthRateLimit == "" {
			os.Unsetenv("AUTH_RATE_LIMIT_PER_HOUR")
		} else {
			os.Setenv("AUTH_RATE_LIMIT_PER_HOUR", originalAuthRateLimit)
		}
		if originalTradingRateLimit == "" {
			os.Unsetenv("TRADING_RATE_LIMIT_PER_HOUR")
		} else {
			os.Setenv("TRADING_RATE_LIMIT_PER_HOUR", originalTradingRateLimit)
		}
	}
	defer cleanup()

	t.Run("rate_limit_disabled_globally", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "false")
		
		router := setupRateLimitTest()
		router.Use(middleware.AuthRateLimitMiddleware())
		router.Use(middleware.APIRateLimitMiddleware())
		router.Use(middleware.TradingRateLimitMiddleware())
		
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		// Should allow unlimited requests when rate limiting is disabled
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should be allowed when rate limiting is disabled", i+1)
			
			// Should not have rate limit headers when disabled
			assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
			assert.Empty(t, w.Header().Get("X-RateLimit-Remaining"))
		}
	})

	t.Run("custom_api_rate_limit_from_env", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "true")
		os.Setenv("API_RATE_LIMIT_PER_HOUR", "5") // Custom limit of 5
		
		router := setupRateLimitTest()
		
		// Simulate authenticated user
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		
		router.Use(middleware.APIRateLimitMiddleware())
		
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		// Should allow exactly 5 requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should be allowed", i+1)
			assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
		}
		
		// 6th request should be denied
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("custom_auth_rate_limit_from_env", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "true")
		os.Setenv("AUTH_RATE_LIMIT_PER_HOUR", "3") // Custom limit of 3
		
		router := setupRateLimitTest()
		router.Use(middleware.AuthRateLimitMiddleware())
		
		router.POST("/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "logged in"})
		})
		
		clientIP := "192.168.1.100:12345"
		
		// Should allow exactly 3 requests
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = clientIP
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should be allowed", i+1)
			assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
		}
		
		// 4th request should be denied
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = clientIP
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
	})

	t.Run("custom_trading_rate_limit_from_env", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "true")
		os.Setenv("TRADING_RATE_LIMIT_PER_HOUR", "2") // Custom limit of 2
		
		router := setupRateLimitTest()
		
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		
		router.Use(middleware.TradingRateLimitMiddleware())
		
		router.POST("/trading-logs", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "trade logged"})
		})
		
		// Should allow exactly 2 requests
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", "/trading-logs", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should be allowed", i+1)
			assert.Equal(t, "2", w.Header().Get("X-RateLimit-Limit"))
		}
		
		// 3rd request should be denied
		req := httptest.NewRequest("POST", "/trading-logs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "2", w.Header().Get("X-RateLimit-Limit"))
	})

	t.Run("invalid_env_values_use_defaults", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "true")
		os.Setenv("API_RATE_LIMIT_PER_HOUR", "invalid")     // Invalid value
		os.Setenv("AUTH_RATE_LIMIT_PER_HOUR", "-5")         // Invalid negative value
		os.Setenv("TRADING_RATE_LIMIT_PER_HOUR", "0")       // Invalid zero value
		
		router := setupRateLimitTest()
		
		// API middleware should use default (1000)
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		router.Use(middleware.APIRateLimitMiddleware())
		
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1000", w.Header().Get("X-RateLimit-Limit")) // Should use default
		
		// Auth middleware should use default (60)
		router2 := setupRateLimitTest()
		router2.Use(middleware.AuthRateLimitMiddleware())
		
		router2.POST("/auth/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req2 := httptest.NewRequest("POST", "/auth/test", nil)
		req2.RemoteAddr = "192.168.1.100:12345"
		
		w2 := httptest.NewRecorder()
		router2.ServeHTTP(w2, req2)
		
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, "60", w2.Header().Get("X-RateLimit-Limit")) // Should use default
		
		// Trading middleware should use default (600)
		router3 := setupRateLimitTest()
		
		router3.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		router3.Use(middleware.TradingRateLimitMiddleware())
		
		router3.POST("/trading/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req3 := httptest.NewRequest("POST", "/trading/test", nil)
		w3 := httptest.NewRecorder()
		router3.ServeHTTP(w3, req3)
		
		assert.Equal(t, http.StatusOK, w3.Code)
		assert.Equal(t, "600", w3.Header().Get("X-RateLimit-Limit")) // Should use default
	})

	t.Run("empty_env_values_use_defaults", func(t *testing.T) {
		os.Setenv("RATE_LIMIT_ENABLED", "true")
		os.Unsetenv("API_RATE_LIMIT_PER_HOUR")    // Empty/unset
		os.Unsetenv("AUTH_RATE_LIMIT_PER_HOUR")   // Empty/unset
		os.Unsetenv("TRADING_RATE_LIMIT_PER_HOUR") // Empty/unset
		
		router := setupRateLimitTest()
		
		// Test API middleware uses default
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})
		router.Use(middleware.APIRateLimitMiddleware())
		
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1000", w.Header().Get("X-RateLimit-Limit")) // Default value
	})
}