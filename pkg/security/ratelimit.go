package security

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimitRule defines rate limiting rules
type RateLimitRule struct {
	Name        string        `json:"name"`
	Limit       int           `json:"limit"`       // Number of requests
	Window      time.Duration `json:"window"`      // Time window
	Burst       int           `json:"burst"`       // Burst allowance
	Description string        `json:"description"`
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed       bool          `json:"allowed"`
	Remaining     int           `json:"remaining"`
	ResetTime     time.Time     `json:"reset_time"`
	RetryAfter    time.Duration `json:"retry_after,omitempty"`
	RuleName      string        `json:"rule_name"`
	CurrentUsage  int           `json:"current_usage"`
}

// RateLimiter implements Redis-based rate limiting with multiple strategies
type RateLimiter struct {
	redis  *redis.Client
	prefix string
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, keyPrefix string) *RateLimiter {
	if keyPrefix == "" {
		keyPrefix = "tiris:ratelimit"
	}
	
	return &RateLimiter{
		redis:  redisClient,
		prefix: keyPrefix,
	}
}

// DefaultRules returns default rate limiting rules for different scenarios
func DefaultRules() map[string]RateLimitRule {
	return map[string]RateLimitRule{
		"auth_login": {
			Name:        "auth_login",
			Limit:       5,
			Window:      15 * time.Minute,
			Burst:       2,
			Description: "Login attempt rate limit",
		},
		"auth_register": {
			Name:        "auth_register",
			Limit:       3,
			Window:      time.Hour,
			Burst:       1,
			Description: "Registration attempt rate limit",
		},
		"api_general": {
			Name:        "api_general",
			Limit:       1000,
			Window:      time.Hour,
			Burst:       100,
			Description: "General API usage limit",
		},
		"api_heavy": {
			Name:        "api_heavy",
			Limit:       100,
			Window:      time.Hour,
			Burst:       10,
			Description: "Heavy API operations limit",
		},
		"password_reset": {
			Name:        "password_reset",
			Limit:       3,
			Window:      time.Hour,
			Burst:       1,
			Description: "Password reset request limit",
		},
		"webhook": {
			Name:        "webhook",
			Limit:       10000,
			Window:      time.Hour,
			Burst:       1000,
			Description: "Webhook delivery limit",
		},
	}
}

// CheckRateLimit checks if a request should be allowed based on rate limiting rules
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, identifier, ruleName string, rule RateLimitRule) (*RateLimitResult, error) {
	key := fmt.Sprintf("%s:%s:%s", rl.prefix, ruleName, identifier)
	
	// Use sliding window log algorithm for more accurate rate limiting
	return rl.slidingWindowCounter(ctx, key, rule)
}

// CheckMultipleRules checks multiple rate limiting rules and returns the most restrictive result
func (rl *RateLimiter) CheckMultipleRules(ctx context.Context, identifier string, rules map[string]RateLimitRule) (*RateLimitResult, error) {
	var mostRestrictive *RateLimitResult
	
	for ruleName, rule := range rules {
		result, err := rl.CheckRateLimit(ctx, identifier, ruleName, rule)
		if err != nil {
			return nil, fmt.Errorf("failed to check rule %s: %v", ruleName, err)
		}
		
		if !result.Allowed {
			return result, nil // Return immediately if any rule blocks the request
		}
		
		if mostRestrictive == nil || result.Remaining < mostRestrictive.Remaining {
			mostRestrictive = result
		}
	}
	
	return mostRestrictive, nil
}

// slidingWindowCounter implements sliding window counter algorithm
func (rl *RateLimiter) slidingWindowCounter(ctx context.Context, key string, rule RateLimitRule) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-rule.Window)
	
	pipe := rl.redis.Pipeline()
	
	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%.3f", float64(windowStart.UnixNano())/1e9))
	
	// Count current entries
	countCmd := pipe.ZCard(ctx, key)
	
	// Add current request timestamp
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()) / 1e9,
		Member: fmt.Sprintf("%d_%d", now.UnixNano(), rand.Intn(10000)),
	})
	
	// Set expiration
	pipe.Expire(ctx, key, rule.Window+time.Minute)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("redis pipeline execution failed: %v", err)
	}
	
	currentCount := int(countCmd.Val())
	
	// Calculate remaining and reset time
	remaining := rule.Limit - currentCount
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := now.Add(rule.Window)
	
	// Check if request is allowed
	allowed := currentCount <= rule.Limit
	
	// Handle burst allowance
	if !allowed && currentCount <= rule.Limit+rule.Burst {
		// Check if burst is available
		burstKey := key + ":burst"
		burstCount, err := rl.redis.Get(ctx, burstKey).Int()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("failed to get burst count: %v", err)
		}
		
		if burstCount < rule.Burst {
			allowed = true
			// Increment burst counter
			pipe := rl.redis.Pipeline()
			pipe.Incr(ctx, burstKey)
			pipe.Expire(ctx, burstKey, rule.Window)
			_, err := pipe.Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update burst counter: %v", err)
			}
		}
	}
	
	result := &RateLimitResult{
		Allowed:      allowed,
		Remaining:    remaining,
		ResetTime:    resetTime,
		RuleName:     rule.Name,
		CurrentUsage: currentCount,
	}
	
	if !allowed {
		// Calculate retry after based on oldest entry
		oldestEntries, err := rl.redis.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err == nil && len(oldestEntries) > 0 {
			oldestTime := time.Unix(int64(oldestEntries[0].Score), 0)
			result.RetryAfter = oldestTime.Add(rule.Window).Sub(now)
			if result.RetryAfter < 0 {
				result.RetryAfter = time.Second
			}
		}
	}
	
	return result, nil
}

// ResetRateLimit resets the rate limit for a specific identifier and rule
func (rl *RateLimiter) ResetRateLimit(ctx context.Context, identifier, ruleName string) error {
	key := fmt.Sprintf("%s:%s:%s", rl.prefix, ruleName, identifier)
	burstKey := key + ":burst"
	
	pipe := rl.redis.Pipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, burstKey)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetRateLimitStatus returns the current status for a specific identifier and rule
func (rl *RateLimiter) GetRateLimitStatus(ctx context.Context, identifier, ruleName string, rule RateLimitRule) (*RateLimitResult, error) {
	key := fmt.Sprintf("%s:%s:%s", rl.prefix, ruleName, identifier)
	
	now := time.Now()
	windowStart := now.Add(-rule.Window)
	
	// Count current entries without adding new ones
	count, err := rl.redis.ZCount(ctx, key, fmt.Sprintf("%.3f", float64(windowStart.UnixNano())/1e9), "+inf").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to count entries: %v", err)
	}
	
	currentCount := int(count)
	remaining := rule.Limit - currentCount
	if remaining < 0 {
		remaining = 0
	}
	
	return &RateLimitResult{
		Allowed:      currentCount <= rule.Limit,
		Remaining:    remaining,
		ResetTime:    now.Add(rule.Window),
		RuleName:     rule.Name,
		CurrentUsage: currentCount,
	}, nil
}

// CleanupExpiredEntries removes expired rate limit entries (should be called periodically)
func (rl *RateLimiter) CleanupExpiredEntries(ctx context.Context) error {
	pattern := rl.prefix + ":*"
	
	iter := rl.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		
		// Skip burst keys
		if strings.HasSuffix(key, ":burst") {
			continue
		}
		
		// Remove entries older than 24 hours
		cutoff := time.Now().Add(-24 * time.Hour)
		_, err := rl.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%.3f", float64(cutoff.UnixNano())/1e9)).Result()
		if err != nil {
			return fmt.Errorf("failed to cleanup key %s: %v", key, err)
		}
	}
	
	return iter.Err()
}