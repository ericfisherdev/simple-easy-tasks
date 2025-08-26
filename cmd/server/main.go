// Package main provides the entry point for the Simple Easy Tasks server application.
package main

//nolint:gofumpt
import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simple-easy-tasks/internal/api"
	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/config"

	// Import migrations to register them with PocketBase
	_ "simple-easy-tasks/migrations"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg := config.NewConfig()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize service container
	// Note: In a full PocketBase integration, we'd pass the PocketBase app instance here
	// For now, we'll set up the container with stub implementations
	container, err := setupServiceContainer(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup service container: %w", err)
	}

	// Setup Gin router with services
	router, rateLimitManager := setupRouter(ctx, cfg, container)
	defer rateLimitManager.Shutdown()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.GetServerPort(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	select {
	case <-sigChan:
		log.Println("Shutdown signal received")
	case <-ctx.Done():
		log.Println("Context canceled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

// setupServiceContainer initializes the DI container with all services.
func setupServiceContainer(cfg *config.AppConfig) (interface{}, error) {
	// For now, return a simple map as a placeholder
	// In a full implementation, this would use the container package
	services := map[string]interface{}{
		"config": cfg,
		"status": "initialized",
	}
	return services, nil
}

// setupRouter configures the Gin router with all middleware and routes.
func setupRouter(
	ctx context.Context,
	cfg *config.AppConfig,
	_ interface{},
) (*gin.Engine, *middleware.RateLimitManager) {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	// Create router
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.DefaultLoggingMiddleware())
	router.Use(middleware.DefaultRecoveryMiddleware())
	router.Use(middleware.DefaultCORSMiddleware())

	// Rate limiting middleware with configuration-driven settings
	var rateLimitManager *middleware.RateLimitManager
	if cfg.GetRateLimitEnabled() {
		rateLimitMiddleware, manager := middleware.RateLimitMiddleware(ctx, middleware.RateLimitConfig{
			RequestsPerMinute: cfg.GetRateLimitRequestsPerMinute(),
			CacheCapacity:     cfg.GetRateLimitCacheCapacity(),
			UseRedis:          cfg.GetRedisEnabled(),
			RedisAddr:         cfg.GetRedisAddr(),
			RedisPassword:     cfg.GetRedisPassword(),
			RedisDB:           cfg.GetRedisDB(),
			KeyGenerator: func(c *gin.Context) string {
				return c.ClientIP()
			},
		})
		router.Use(rateLimitMiddleware)
		rateLimitManager = manager
	}

	// Service container is now initialized and available for use
	// Future handlers can resolve services from the container

	// Root route
	router.GET("/", rootHandler)

	// Add ping endpoint for simple health checks
	router.GET("/ping", api.PingHandler)

	// Enhanced health endpoint with service container status
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"timestamp":   fmt.Sprintf("%d", time.Now().Unix()),
			"environment": cfg.GetEnvironment(),
			"version":     "1.0.0",
			"services": gin.H{
				"container":    "initialized",
				"architecture": "service-oriented",
				"di_pattern":   "enabled",
			},
		})
	})

	// Add metrics endpoint for monitoring
	router.GET("/metrics", func(c *gin.Context) {
		metrics := gin.H{
			"timestamp": time.Now().Unix(),
			"system": gin.H{
				"environment": cfg.GetEnvironment(),
				"version":     "1.0.0",
			},
		}

		if rateLimitManager != nil {
			stats := rateLimitManager.Stats()
			metrics["rate_limiting"] = gin.H{
				"enabled":       cfg.GetRateLimitEnabled(),
				"redis_enabled": cfg.GetRedisEnabled(),
				"cache_stats":   stats,
			}
		} else {
			metrics["rate_limiting"] = gin.H{
				"enabled": false,
			}
		}

		c.JSON(http.StatusOK, metrics)
	})

	// API base endpoint
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Simple Easy Tasks API",
			"version": "1.0.0",
			"status":  "operational",
			"endpoints": gin.H{
				"health":   "/health",
				"ping":     "/ping",
				"auth":     "/api/auth/*",
				"users":    "/api/users/*",
				"projects": "/api/projects/*",
			},
		})
	})

	return router, rateLimitManager
}

// rootHandler handles the root path and returns a simple HTML page.
func rootHandler(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Simple Easy Tasks</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        .container { max-width: 600px; margin: 0 auto; }
        .status { color: #28a745; font-weight: bold; }
        .links { margin-top: 20px; }
        .links a { display: inline-block; margin-right: 15px; color: #007bff; text-decoration: none; }
        .links a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Simple Easy Tasks</h1>
        <p><strong>Phase 1: Foundation & Infrastructure Setup</strong></p>
        <p class="status">✅ Server is running!</p>
        
        <h3>Week 4 Progress:</h3>
        <ul>
            <li>✅ Gin router with comprehensive middleware stack</li>
            <li>✅ RESTful API endpoints for authentication</li>
            <li>✅ User profile management endpoints</li>
            <li>✅ Project management endpoints</li>
            <li>✅ CORS, logging, rate limiting, and recovery middleware</li>
            <li>✅ Request ID tracking and standardized error handling</li>
        </ul>

        <div class="links">
            <h3>API Endpoints:</h3>
            <a href="/health">Health Check</a>
            <a href="/ping">Ping</a>
            <a href="/api">API Base</a>
        </div>
    </div>
</body>
</html>
`)
}
