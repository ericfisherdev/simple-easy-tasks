// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a simple rate limiter using token bucket algorithm.
type RateLimiter struct {
	tokens     int
	capacity   int
	refill     time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(capacity int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     capacity,
		capacity:   capacity,
		refill:     refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed based on rate limits.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Refill tokens based on elapsed time
	if now.Sub(rl.lastRefill) >= rl.refill {
		tokensToAdd := int(now.Sub(rl.lastRefill) / rl.refill)
		rl.tokens = min(rl.capacity, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	// RequestsPerMinute specifies the maximum number of requests per minute.
	RequestsPerMinute int
	// KeyGenerator is a function that generates a key for rate limiting (e.g., IP address, user ID).
	KeyGenerator func(c *gin.Context) string
	// OnExceeded is called when rate limit is exceeded.
	OnExceeded func(c *gin.Context)
}

// RateLimitMiddleware returns a rate limiting middleware.
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	// Store rate limiters per key
	limiters := make(map[string]*RateLimiter)
	var mu sync.RWMutex

	// Cleanup goroutine to prevent memory leaks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			// Remove old limiters (simple cleanup - could be improved with LRU)
			for key, limiter := range limiters {
				limiter.mu.Lock()
				if time.Since(limiter.lastRefill) > 10*time.Minute {
					delete(limiters, key)
				}
				limiter.mu.Unlock()
			}
			mu.Unlock()
		}
	}()

	return gin.HandlerFunc(func(c *gin.Context) {
		key := config.KeyGenerator(c)

		mu.Lock()
		limiter, exists := limiters[key]
		if !exists {
			limiter = NewRateLimiter(config.RequestsPerMinute, time.Minute/time.Duration(config.RequestsPerMinute))
			limiters[key] = limiter
		}
		mu.Unlock()

		if !limiter.Allow() {
			if config.OnExceeded != nil {
				config.OnExceeded(c)
			} else {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error": map[string]interface{}{
						"type":    "RATE_LIMIT_ERROR",
						"code":    "TOO_MANY_REQUESTS",
						"message": "Rate limit exceeded. Please try again later.",
					},
				})
			}
			c.Abort()
			return
		}

		c.Next()
	})
}

// DefaultRateLimitMiddleware returns a rate limiting middleware with IP-based limiting.
func DefaultRateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		KeyGenerator: func(c *gin.Context) string {
			return c.ClientIP()
		},
	})
}

// UserBasedRateLimitMiddleware returns a rate limiting middleware based on authenticated user.
func UserBasedRateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		KeyGenerator: func(c *gin.Context) string {
			// Try to get user from context first
			if user, exists := GetUserFromContext(c); exists {
				return "user:" + user.ID
			}
			// Fall back to IP address for unauthenticated requests
			return "ip:" + c.ClientIP()
		},
	})
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
