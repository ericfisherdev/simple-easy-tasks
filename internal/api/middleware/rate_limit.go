// Package middleware provides HTTP middleware functions.
package middleware

import (
	"container/list"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

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

// LRUCache represents an LRU cache for rate limiters.
type LRUCache struct {
	items    map[string]*list.Element
	list     *list.List
	mu       sync.RWMutex
	capacity int
}

// LRUItem represents an item in the LRU cache.
type LRUItem struct {
	limiter *RateLimiter
	key     string
}

// NewLRUCache creates a new LRU cache with the specified capacity.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get retrieves a rate limiter from the cache or creates a new one.
func (c *LRUCache) Get(key string, factory func() *RateLimiter) *RateLimiter {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		// Move to front (most recently used)
		c.list.MoveToFront(elem)
		item, ok := elem.Value.(*LRUItem)
		if !ok {
			// This should never happen, but handle gracefully
			delete(c.items, key)
			c.list.Remove(elem)
			return factory()
		}
		return item.limiter
	}

	// Create new limiter
	limiter := factory()
	item := &LRUItem{key: key, limiter: limiter}

	// Add to front
	elem := c.list.PushFront(item)
	c.items[key] = elem

	// Remove oldest if over capacity
	if c.list.Len() > c.capacity {
		oldest := c.list.Back()
		if oldest != nil {
			c.removeElement(oldest)
		}
	}

	return limiter
}

// removeElement removes an element from the cache.
func (c *LRUCache) removeElement(elem *list.Element) {
	c.list.Remove(elem)
	if item, ok := elem.Value.(*LRUItem); ok {
		delete(c.items, item.key)
	}
}

// Len returns the current number of items in the cache.
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list.Len()
}

// Clear removes all items from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.list = list.New()
	c.items = make(map[string]*list.Element)
}

// RedisRateLimiter implements distributed rate limiting using Redis.
type RedisRateLimiter struct {
	client            *redis.Client
	keyPrefix         string
	requestsPerMinute int
	windowSize        time.Duration
}

// NewRedisRateLimiter creates a new Redis-based rate limiter.
func NewRedisRateLimiter(client *redis.Client, keyPrefix string, requestsPerMinute int) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:            client,
		keyPrefix:         keyPrefix,
		requestsPerMinute: requestsPerMinute,
		windowSize:        time.Minute,
	}
}

// Allow checks if a request should be allowed using Redis sliding window.
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s", rl.keyPrefix, key)
	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	pipe := rl.client.Pipeline()

	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.Unix(), 10))

	// Count current requests in window
	pipe.ZCard(ctx, redisKey)

	// Add current request
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.Unix()),
		Member: now.UnixNano(), // Use nanoseconds for uniqueness
	})

	// Set expiration
	pipe.Expire(ctx, redisKey, rl.windowSize+time.Minute)

	results, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis rate limiting error: %w", err)
	}

	// Get the count of requests (second command in pipeline)
	countCmd, ok := results[1].(*redis.IntCmd)
	if !ok {
		return false, fmt.Errorf("unexpected Redis command result type")
	}
	count, err := countCmd.Result()
	if err != nil {
		return false, fmt.Errorf("failed to get request count: %w", err)
	}

	return count < int64(rl.requestsPerMinute), nil
}

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	// KeyGenerator is a function that generates a key for rate limiting (e.g., IP address, user ID).
	KeyGenerator func(c *gin.Context) string
	// OnExceeded is called when rate limit is exceeded.
	OnExceeded func(c *gin.Context)
	// RedisAddr specifies the Redis server address (required if UseRedis is true).
	RedisAddr string
	// RedisPassword specifies the Redis password (optional).
	RedisPassword string
	// CleanupInterval specifies how often to clean up old limiters (default: 5 minutes).
	CleanupInterval time.Duration
	// MaxAge specifies the maximum age of an inactive limiter before cleanup (default: 10 minutes).
	MaxAge time.Duration
	// RequestsPerMinute specifies the maximum number of requests per minute.
	RequestsPerMinute int
	// CacheCapacity specifies the maximum number of rate limiters to keep in memory (default: 10000).
	// Using LRU cache prevents unbounded memory growth.
	CacheCapacity int
	// RedisDB specifies the Redis database number (default: 0).
	RedisDB int
	// UseRedis enables Redis-based distributed rate limiting for production environments.
	UseRedis bool
}

// RateLimitManager manages rate limiters and their lifecycle.
type RateLimitManager struct {
	cache            *LRUCache
	redisRateLimiter *RedisRateLimiter
	cleanupDone      chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	config           RateLimitConfig
	cleanupInterval  time.Duration
	maxAge           time.Duration
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
	if config.CacheCapacity == 0 {
		config.CacheCapacity = 10000
	}

	managerCtx, cancel := context.WithCancel(ctx)

	manager := &RateLimitManager{
		cache:           NewLRUCache(config.CacheCapacity),
		config:          config,
		ctx:             managerCtx,
		cancel:          cancel,
		cleanupDone:     make(chan struct{}),
		cleanupInterval: config.CleanupInterval,
		maxAge:          config.MaxAge,
	}

	// Setup Redis rate limiter if enabled
	if config.UseRedis {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})

		// Test Redis connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			panic(fmt.Sprintf("failed to connect to Redis: %v", err))
		}

		manager.redisRateLimiter = NewRedisRateLimiter(redisClient, "rate_limit", config.RequestsPerMinute)
	}

	// Start cleanup goroutine for age-based cleanup (only needed for in-memory cache)
	if !config.UseRedis {
		go manager.cleanup()
	} else {
		// Still need the cleanup goroutine for proper shutdown
		go func() {
			<-manager.ctx.Done()
			close(manager.cleanupDone)
		}()
	}

	return manager
}

// Allow checks if a request should be allowed for the given key.
func (rm *RateLimitManager) Allow(ctx context.Context, key string) (bool, error) {
	if rm.config.UseRedis && rm.redisRateLimiter != nil {
		return rm.redisRateLimiter.Allow(ctx, key)
	}

	// Fall back to in-memory rate limiting
	limiter := rm.GetLimiter(key)
	return limiter.Allow(), nil
}

// GetLimiter gets or creates a rate limiter for the given key (in-memory only).
func (rm *RateLimitManager) GetLimiter(key string) *RateLimiter {
	return rm.cache.Get(key, func() *RateLimiter {
		return NewRateLimiter(rm.config.RequestsPerMinute, time.Minute/time.Duration(rm.config.RequestsPerMinute))
	})
}

// cleanup runs periodically to clean up old rate limiters from the cache.
// With LRU cache, this is less critical but still useful for removing truly stale entries.
func (rm *RateLimitManager) cleanup() {
	defer close(rm.cleanupDone)

	ticker := time.NewTicker(rm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			// With LRU cache, memory is already bounded, so cleanup is less critical
			// We can still remove very old entries, but it's optional
			rm.cleanupOldEntries()
		}
	}
}

// cleanupOldEntries removes entries that haven't been used for a long time.
// This is optional with LRU cache but can help free memory sooner.
func (rm *RateLimitManager) cleanupOldEntries() {
	rm.cache.mu.Lock()
	defer rm.cache.mu.Unlock()

	now := time.Now()
	var toRemove []*list.Element

	// Walk from back (oldest) to front
	for elem := rm.cache.list.Back(); elem != nil; elem = elem.Prev() {
		item, ok := elem.Value.(*LRUItem)
		if !ok {
			toRemove = append(toRemove, elem)
			continue
		}
		item.limiter.mu.Lock()
		age := now.Sub(item.limiter.lastRefill)
		item.limiter.mu.Unlock()

		if age > rm.maxAge {
			toRemove = append(toRemove, elem)
		} else {
			// Since we're walking from oldest to newest, we can break here
			break
		}
	}

	// Remove the old entries
	for _, elem := range toRemove {
		rm.cache.removeElement(elem)
	}
}

// Shutdown gracefully shuts down the rate limit manager.
func (rm *RateLimitManager) Shutdown() {
	rm.cancel()
	<-rm.cleanupDone
}

// Stats returns statistics about the rate limiter cache.
func (rm *RateLimitManager) Stats() RateLimitStats {
	cacheLen := rm.cache.Len()
	return RateLimitStats{
		CacheSize:     cacheLen,
		CacheCapacity: rm.config.CacheCapacity,
		CacheUsage:    float64(cacheLen) / float64(rm.config.CacheCapacity),
	}
}

// RateLimitStats holds statistics about rate limiting.
type RateLimitStats struct {
	CacheSize     int     `json:"cache_size"`
	CacheCapacity int     `json:"cache_capacity"`
	CacheUsage    float64 `json:"cache_usage"`
}

// RateLimitMiddleware returns a rate limiting middleware.
// Important: The returned manager must be shut down gracefully to prevent goroutine leaks.
func RateLimitMiddleware(ctx context.Context, config RateLimitConfig) (gin.HandlerFunc, *RateLimitManager) {
	manager := NewRateLimitManager(ctx, config)

	middleware := gin.HandlerFunc(func(c *gin.Context) {
		key := config.KeyGenerator(c)

		allowed, err := manager.Allow(c.Request.Context(), key)
		if err != nil {
			// Log error and allow request (fail open for resilience)
			c.Header("X-RateLimit-Error", "true")
			c.Next()
			return
		}

		if !allowed {
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
