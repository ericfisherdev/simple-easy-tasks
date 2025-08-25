package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitManager_Lifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := RateLimitConfig{
		RequestsPerMinute: 10,
		CleanupInterval:   100 * time.Millisecond,
		MaxAge:            200 * time.Millisecond,
		KeyGenerator: func(c *gin.Context) string {
			return c.ClientIP()
		},
	}

	manager := NewRateLimitManager(ctx, config)

	// Test that limiters are created
	limiter1 := manager.GetLimiter("test-key-1")
	limiter2 := manager.GetLimiter("test-key-2")

	assert.NotNil(t, limiter1)
	assert.NotNil(t, limiter2)
	assert.Equal(t, 2, manager.cache.Len())

	// Wait for cleanup to potentially trigger
	time.Sleep(300 * time.Millisecond)

	// Limiters should be cleaned up due to inactivity
	count := manager.cache.Len()

	assert.Equal(t, 0, count, "Inactive limiters should be cleaned up")

	// Test graceful shutdown
	manager.Shutdown()

	// Verify cleanup goroutine is stopped
	select {
	case <-manager.cleanupDone:
		// Success - cleanup goroutine finished
	case <-time.After(1 * time.Second):
		t.Fatal("Cleanup goroutine did not finish within timeout")
	}
}

func TestRateLimitMiddleware_MemoryLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := RateLimitConfig{
		RequestsPerMinute: 100,
		CleanupInterval:   50 * time.Millisecond,
		MaxAge:            100 * time.Millisecond,
		KeyGenerator: func(c *gin.Context) string {
			return c.GetHeader("X-Test-Key")
		},
	}

	middleware, manager := RateLimitMiddleware(ctx, config)
	defer manager.Shutdown()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Create many different limiters by using different keys
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Test-Key", fmt.Sprintf("key-%d", i))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}

	// Verify limiters were created
	initialCount := manager.cache.Len()
	assert.Equal(t, 50, initialCount)

	// Wait for cleanup cycles
	time.Sleep(200 * time.Millisecond)

	// Verify limiters were cleaned up
	finalCount := manager.cache.Len()

	assert.Equal(t, 0, finalCount, "All inactive limiters should be cleaned up")
}

func TestRateLimitMiddleware_RateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	config := RateLimitConfig{
		RequestsPerMinute: 2, // Very low limit for testing
		KeyGenerator: func(_ *gin.Context) string {
			return "test-key"
		},
	}

	middleware, manager := RateLimitMiddleware(ctx, config)
	defer manager.Shutdown()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// Second request should succeed
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code)

	// Third request should be rate limited
	req3 := httptest.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
}

func TestRateLimitManager_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	config := RateLimitConfig{
		RequestsPerMinute: 10,
		CleanupInterval:   50 * time.Millisecond,
		MaxAge:            100 * time.Millisecond,
		KeyGenerator: func(_ *gin.Context) string {
			return "test"
		},
	}

	manager := NewRateLimitManager(ctx, config)

	// Cancel context to trigger cleanup goroutine exit
	cancel()

	// Verify cleanup goroutine exits within reasonable time
	select {
	case <-manager.cleanupDone:
		// Success - cleanup goroutine finished due to context cancellation
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Cleanup goroutine did not exit after context cancellation")
	}
}

func TestRateLimitMiddleware_CustomOnExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	customHandlerCalled := false

	config := RateLimitConfig{
		RequestsPerMinute: 1,
		KeyGenerator: func(_ *gin.Context) string {
			return "test-key"
		},
		OnExceeded: func(c *gin.Context) {
			customHandlerCalled = true
			c.JSON(http.StatusTooManyRequests, gin.H{
				"custom": "rate limit exceeded",
			})
		},
	}

	middleware, manager := RateLimitMiddleware(ctx, config)
	defer manager.Shutdown()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// Second request should trigger custom handler
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	assert.True(t, customHandlerCalled)
	assert.Contains(t, w2.Body.String(), "custom")
}
