// Package middleware provides HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"simple-easy-tasks/internal/domain"
)

// RateLimiter represents a simple rate limiter using token bucket algorithm.
type RateLimiter struct {
	lastRefill time.Time
	mu         sync.Mutex
	refill     time.Duration
	tokens     int
	capacity   int
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(capacity int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		lastRefill: time.Now(),
		refill:     refillRate,
		tokens:     capacity,
		capacity:   capacity,
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
		rl.tokens = minInt(rl.capacity, rl.tokens+tokensToAdd)
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
	// KeyGenerator is a function that generates a key for rate limiting (e.g., IP address, user ID).
	KeyGenerator func(c *gin.Context) string
	// OnExceeded is called when rate limit is exceeded.
	OnExceeded func(c *gin.Context)
	// RequestsPerMinute specifies the maximum number of requests per minute.
	RequestsPerMinute int
	// CleanupInterval specifies how often to clean up old limiters (default: 5 minutes).
	CleanupInterval time.Duration
	// MaxAge specifies the maximum age of an inactive limiter before cleanup (default: 10 minutes).
	MaxAge time.Duration
}

// RateLimitManager manages rate limiters and their lifecycle.
type RateLimitManager struct {
	limiters        map[string]*RateLimiter
	cleanupDone     chan struct{}
	ctx             context.Context
	cancel          context.CancelFunc
	config          RateLimitConfig
	mu              sync.RWMutex
	cleanupInterval time.Duration
	maxAge          time.Duration
}

// NewRateLimitManager creates a new rate limit manager.
func NewRateLimitManager(ctx context.Context, config RateLimitConfig) *RateLimitManager {
	// Set defaults
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.MaxAge == 0 {
		config.MaxAge = 10 * time.Minute
	}

	managerCtx, cancel := context.WithCancel(ctx)

	manager := &RateLimitManager{
		limiters:        make(map[string]*RateLimiter),
		config:          config,
		ctx:             managerCtx,
		cancel:          cancel,
		cleanupDone:     make(chan struct{}),
		cleanupInterval: config.CleanupInterval,
		maxAge:          config.MaxAge,
	}

	// Start cleanup goroutine
	go manager.cleanup()

	return manager
}

// GetLimiter gets or creates a rate limiter for the given key.
func (rm *RateLimitManager) GetLimiter(key string) *RateLimiter {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	limiter, exists := rm.limiters[key]
	if !exists {
		limiter = NewRateLimiter(rm.config.RequestsPerMinute, time.Minute/time.Duration(rm.config.RequestsPerMinute))
		rm.limiters[key] = limiter
	}

	return limiter
}

// cleanup runs periodically to remove old rate limiters.
func (rm *RateLimitManager) cleanup() {
	defer close(rm.cleanupDone)

	ticker := time.NewTicker(rm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.Lock()
			now := time.Now()
			for key, limiter := range rm.limiters {
				limiter.mu.Lock()
				if now.Sub(limiter.lastRefill) > rm.maxAge {
					delete(rm.limiters, key)
				}
				limiter.mu.Unlock()
			}
			rm.mu.Unlock()
		}
	}
}

// Shutdown gracefully shuts down the rate limit manager.
func (rm *RateLimitManager) Shutdown() {
	rm.cancel()
	<-rm.cleanupDone
}

// RateLimitMiddleware returns a rate limiting middleware.
// Important: The returned manager must be shut down gracefully to prevent goroutine leaks.
func RateLimitMiddleware(ctx context.Context, config RateLimitConfig) (gin.HandlerFunc, *RateLimitManager) {
	manager := NewRateLimitManager(ctx, config)

	middleware := gin.HandlerFunc(func(c *gin.Context) {
		key := config.KeyGenerator(c)
		limiter := manager.GetLimiter(key)

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

	return middleware, manager
}

// DefaultRateLimitMiddleware returns a rate limiting middleware with IP-based limiting.
// Note: This creates a rate limiter without proper cleanup. Use RateLimitMiddleware directly for production.
func DefaultRateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	ctx := context.Background() // NOTE: This doesn't support graceful shutdown
	middleware, _ := RateLimitMiddleware(ctx, RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		KeyGenerator: func(c *gin.Context) string {
			return c.ClientIP()
		},
	})
	return middleware
}

// UserBasedRateLimitMiddleware returns a rate limiting middleware based on authenticated user.
// Note: This creates a rate limiter without proper cleanup. Use RateLimitMiddleware directly for production.
func UserBasedRateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	ctx := context.Background() // NOTE: This doesn't support graceful shutdown
	middleware, _ := RateLimitMiddleware(ctx, RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		KeyGenerator: func(c *gin.Context) string {
			// Try to get user from context first
			// Note: GetUserFromContext is defined in auth_middleware.go
			if userVal, exists := c.Get("user"); exists {
				if user, ok := userVal.(*domain.User); ok {
					return "user:" + user.ID
				}
			}
			// Fall back to IP address for unauthenticated requests
			return "ip:" + c.ClientIP()
		},
	})
	return middleware
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
