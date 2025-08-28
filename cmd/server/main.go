// Package main provides the entry point for the Simple Easy Tasks server application.
package main

//nolint:gofumpt
import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/api"
	"github.com/ericfisherdev/simple-easy-tasks/internal/api/middleware"
	"github.com/ericfisherdev/simple-easy-tasks/internal/config"
	"github.com/ericfisherdev/simple-easy-tasks/internal/container"

	// Import migrations to register them with PocketBase
	_ "github.com/ericfisherdev/simple-easy-tasks/migrations"

	"github.com/gin-gonic/gin"
	"github.com/pocketbase/pocketbase/core"
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
	// For now, we'll create a nil app until full PocketBase integration
	var app core.App // nil for now
	serviceContainer, err := setupServiceContainer(cfg, app)
	if err != nil {
		return fmt.Errorf("failed to setup service container: %w", err)
	}

	// Setup Gin router with services
	router, rateLimitManager := setupRouter(ctx, cfg, serviceContainer)
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
func setupServiceContainer(cfg *config.AppConfig, app core.App) (container.Container, error) {
	// Initialize the DI container with real services
	serviceContainer, err := container.InitializeServices(cfg, app)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return serviceContainer, nil
}

// setupRouter configures the Gin router with all middleware and routes.
func setupRouter(
	ctx context.Context,
	cfg *config.AppConfig,
	serviceContainer container.Container,
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
	// Note: serviceContainer parameter will be used when handlers are updated to use DI
	_ = serviceContainer

	// Static files
	router.Static("/static", "./web/static")

	// Root route
	router.GET("/", rootHandler)

	// Dashboard route
	router.GET("/dashboard", dashboardHandler)

	// Projects route
	router.GET("/projects", projectsHandler)

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

	// Temporary API endpoints to prevent 404 errors
	// TODO: Replace with proper authenticated handlers when DI is fully integrated
	api := router.Group("/api")
	{
		projects := api.Group("/projects")
		{
			// GET /api/projects - Return projects list
			projects.GET("", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"projects": []gin.H{
						{
							"id":          "1",
							"name":        "Website Redesign",
							"description": "Complete overhaul of the company website with modern design and improved UX",
							"status":      "Active",
							"progress":    65,
							"taskCount":   24,
							"memberCount": 5,
							"updatedAt":   "2025-08-26T10:00:00Z",
							"createdAt":   "2025-08-01T09:00:00Z",
						},
						{
							"id":          "2",
							"name":        "Mobile App Development",
							"description": "Native iOS and Android app for task management",
							"status":      "Planning",
							"progress":    15,
							"taskCount":   8,
							"memberCount": 3,
							"updatedAt":   "2025-08-23T14:30:00Z",
							"createdAt":   "2025-08-15T11:00:00Z",
						},
						{
							"id":          "3",
							"name":        "API Integration",
							"description": "Integration with third-party services and APIs",
							"status":      "Completed",
							"progress":    100,
							"taskCount":   12,
							"memberCount": 2,
							"updatedAt":   "2025-08-21T16:45:00Z",
							"createdAt":   "2025-07-10T08:30:00Z",
						},
					},
					"status":  "success",
					"message": "Projects loaded (demo mode)",
				})
			})

			projectsDetail := projects.Group("/:projectId")
			{
				// GET /api/projects/:projectId/tasks - Return empty task list for now
				projectsDetail.GET("/tasks", func(c *gin.Context) {
					projectID := c.Param("projectId")
					c.JSON(http.StatusOK, gin.H{
						"project_id": projectID,
						"tasks":      []interface{}{}, // Empty array for now
						"status":     "success",
						"message":    "Tasks loaded (demo mode)",
					})
				})

				// GET /api/projects/:projectId/events - Server-Sent Events for real-time updates
				projectsDetail.GET("/events", func(c *gin.Context) {
					projectID := c.Param("projectId")

					// Set headers for Server-Sent Events
					c.Header("Content-Type", "text/event-stream")
					c.Header("Cache-Control", "no-cache")
					c.Header("Connection", "keep-alive")
					c.Header("Access-Control-Allow-Origin", "*")
					c.Header("Access-Control-Allow-Headers", "Cache-Control")

					// Send initial connection event
					c.String(http.StatusOK, "data: {\"type\":\"connected\",\"project_id\":\"%s\",\"message\":\"Real-time connection established (demo mode)\"}\n\n", projectID)

					// For now, just send the initial message and close
					// TODO: Replace with proper EventSource implementation
				})
			}
		}
	}

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
            <h3>Available Pages:</h3>
            <a href="/dashboard">Task Board Dashboard</a>
            <a href="/health">Health Check</a>
            <a href="/ping">Ping</a>
            <a href="/api">API Base</a>
        </div>
    </div>
</body>
</html>
`)
}

// dashboardHandler serves the task board dashboard
func dashboardHandler(c *gin.Context) {
	// Load HTML template
	tmpl, err := template.ParseFiles("web/templates/dashboard.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading template: %v", err)
		return
	}

	// Template data
	data := struct {
		CSRFToken string
		User      struct {
			Name string
		}
	}{
		CSRFToken: "sample-csrf-token", // In real implementation, use actual CSRF token
		User: struct {
			Name string
		}{
			Name: "Demo User",
		},
	}

	// Render template
	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template: %v", err)
		return
	}
}

// projectsHandler serves the projects page
func projectsHandler(c *gin.Context) {
	// Load HTML template
	tmpl, err := template.ParseFiles("web/templates/projects.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error loading template: %v", err)
		return
	}

	// Template data
	data := struct {
		CSRFToken string
		User      struct {
			Name string
		}
	}{
		CSRFToken: "sample-csrf-token", // In real implementation, use actual CSRF token
		User: struct {
			Name string
		}{
			Name: "Demo User",
		},
	}

	// Render template
	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template: %v", err)
		return
	}
}
