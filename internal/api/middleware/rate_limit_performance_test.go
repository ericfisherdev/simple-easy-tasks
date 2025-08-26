package middleware

import (
	"context"
	"fmt"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitManager_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory-heavy test in short mode")
	}
	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test LRU cache prevents unbounded memory growth
	config := RateLimitConfig{
		RequestsPerMinute: 100,
		CacheCapacity:     100, // Small capacity to test LRU behavior
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

	// Measure initial memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create many more limiters than cache capacity
	numRequests := 1000 // 10x cache capacity
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Test-Key", fmt.Sprintf("key-%d", i))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Give cleanup a chance to run
	time.Sleep(200 * time.Millisecond)

	// Measure final memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Check cache stats
	stats := manager.Stats()

	// Cache should not exceed capacity
	if stats.CacheSize > config.CacheCapacity {
		t.Errorf("Cache size %d exceeded capacity %d", stats.CacheSize, config.CacheCapacity)
	}

	// Cache usage should be within reasonable bounds
	if stats.CacheUsage > 1.0 {
		t.Errorf("Cache usage %.2f exceeds 100%%", stats.CacheUsage)
	}

	// Memory growth should be bounded (handle overflow from GC)
	var memoryGrowth uint64
	if m2.Alloc > m1.Alloc {
		memoryGrowth = m2.Alloc - m1.Alloc
	}

	t.Logf("Initial memory: %d bytes", m1.Alloc)
	t.Logf("Final memory: %d bytes", m2.Alloc)
	t.Logf("Memory growth: %d bytes", memoryGrowth)
	t.Logf("Cache stats: %+v", stats)

	// The growth should be reasonable (less than 10MB for this test)
	// Allow some overhead for Go runtime
	if memoryGrowth > 10*1024*1024 {
		t.Errorf("Excessive memory growth: %d bytes", memoryGrowth)
	}
}

func TestRateLimitManager_Concurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := RateLimitConfig{
		RequestsPerMinute: 1000,
		CacheCapacity:     100,
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

	// Test concurrent access with many goroutines
	numGoroutines := 50
	requestsPerGoroutine := 20
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Test-Key", fmt.Sprintf("routine-%d-req-%d", routineID, j))
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	totalRequests := numGoroutines * requestsPerGoroutine
	requestsPerSecond := float64(totalRequests) / duration.Seconds()

	t.Logf("Processed %d requests in %v (%.2f req/s)", totalRequests, duration, requestsPerSecond)

	// Should handle at least 1000 requests per second
	if requestsPerSecond < 1000 {
		t.Errorf("Performance too low: %.2f req/s", requestsPerSecond)
	}

	// Cache should still be within bounds
	stats := manager.Stats()
	if stats.CacheSize > config.CacheCapacity {
		t.Errorf("Cache size %d exceeded capacity %d after concurrent test", stats.CacheSize, config.CacheCapacity)
	}
}

func BenchmarkRateLimitManager_InMemory(b *testing.B) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	config := RateLimitConfig{
		RequestsPerMinute: 10000,
		CacheCapacity:     1000,
		KeyGenerator: func(_ *gin.Context) string {
			return "benchmark-key"
		},
	}

	middleware, manager := RateLimitMiddleware(ctx, config)
	defer manager.Shutdown()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
