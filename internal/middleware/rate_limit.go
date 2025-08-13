package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a token bucket rate limiter
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// GetTokens returns the current number of available tokens
func (rl *RateLimiter) GetTokens() int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.tokens
}

// RateLimitStore manages rate limiters for different keys
type RateLimitStore struct {
	limiters   map[string]*RateLimiter
	mutex      sync.RWMutex
	maxTokens  int
	refillRate time.Duration
}

// NewRateLimitStore creates a new rate limit store
func NewRateLimitStore(maxTokens int, refillRate time.Duration) *RateLimitStore {
	return &RateLimitStore{
		limiters:   make(map[string]*RateLimiter),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

// GetLimiter gets or creates a rate limiter for a key
func (rls *RateLimitStore) GetLimiter(key string) *RateLimiter {
	rls.mutex.RLock()
	limiter, exists := rls.limiters[key]
	rls.mutex.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	rls.mutex.Lock()
	defer rls.mutex.Unlock()

	// Check again in case another goroutine created it
	if limiter, exists := rls.limiters[key]; exists {
		return limiter
	}

	limiter = NewRateLimiter(rls.maxTokens, rls.refillRate)
	rls.limiters[key] = limiter
	return limiter
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerHour int                       // Maximum requests per hour
	WindowDuration  time.Duration             // Time window for rate limiting
	KeyFunc         func(*gin.Context) string // Function to generate rate limit key
}

// DefaultKeyFunc generates a rate limit key based on IP address
func DefaultKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// UserKeyFunc generates a rate limit key based on user ID
func UserKeyFunc(c *gin.Context) string {
	userID, exists := GetUserID(c)
	if exists {
		return fmt.Sprintf("user:%s", userID.String())
	}
	return c.ClientIP()
}

// EndpointKeyFunc generates a rate limit key based on endpoint and user/IP
func EndpointKeyFunc(c *gin.Context) string {
	endpoint := c.Request.Method + ":" + c.FullPath()
	userID, exists := GetUserID(c)
	if exists {
		return fmt.Sprintf("%s:user:%s", endpoint, userID.String())
	}
	return fmt.Sprintf("%s:ip:%s", endpoint, c.ClientIP())
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	if config.KeyFunc == nil {
		config.KeyFunc = DefaultKeyFunc
	}

	if config.WindowDuration == 0 {
		config.WindowDuration = time.Hour
	}

	// Calculate refill rate based on requests per hour
	refillRate := config.WindowDuration / time.Duration(config.RequestsPerHour)
	store := NewRateLimitStore(config.RequestsPerHour, refillRate)

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		limiter := store.GetLimiter(key)

		if !limiter.Allow() {
			// Calculate reset time
			resetTime := time.Now().Add(refillRate).Unix()

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerHour))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
			c.Header("X-RateLimit-Window", strconv.Itoa(int(config.WindowDuration.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
				},
			})
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		remaining := limiter.GetTokens()
		resetTime := time.Now().Add(refillRate).Unix()

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerHour))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
		c.Header("X-RateLimit-Window", strconv.Itoa(int(config.WindowDuration.Seconds())))

		c.Next()
	}
}

// AuthRateLimitMiddleware applies specific rate limiting for authentication endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		RequestsPerHour: 60, // 60 requests per hour for auth endpoints
		WindowDuration:  time.Hour,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("auth:%s", c.ClientIP())
		},
	})
}

// APIRateLimitMiddleware applies general rate limiting for API endpoints
func APIRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		RequestsPerHour: 1000, // 1000 requests per hour for general API
		WindowDuration:  time.Hour,
		KeyFunc:         UserKeyFunc,
	})
}

// TradingRateLimitMiddleware applies specific rate limiting for trading endpoints
func TradingRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		RequestsPerHour: 600, // 600 requests per hour for trading operations
		WindowDuration:  time.Hour,
		KeyFunc: func(c *gin.Context) string {
			userID, exists := GetUserID(c)
			if exists {
				return fmt.Sprintf("trading:user:%s", userID.String())
			}
			return fmt.Sprintf("trading:ip:%s", c.ClientIP())
		},
	})
}
