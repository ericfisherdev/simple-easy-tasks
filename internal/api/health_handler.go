// Package api provides HTTP handlers and API endpoints.
package api

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"simple-easy-tasks/internal/services"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	healthService *services.HealthService
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(healthService *services.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

// RegisterRoutes registers health check routes.
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	health := router.Group("/health")
	{
		// Comprehensive health check (all dependencies)
		health.GET("", h.HealthCheck)
		health.GET("/", h.HealthCheck)

		// Liveness probe - is the application alive?
		health.GET("/live", h.Liveness)
		health.GET("/liveness", h.Liveness)

		// Readiness probe - is the application ready to serve traffic?
		health.GET("/ready", h.Readiness)
		health.GET("/readiness", h.Readiness)

		// Detailed health information
		health.GET("/detailed", h.DetailedHealth)
		health.GET("/info", h.SystemInfo)
	}
}

// HealthCheck performs a comprehensive health check
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response := h.healthService.Check(ctx)
	h.renderHealthResponse(c, response)
}

// Liveness returns the liveness status
func (h *HealthHandler) Liveness(c *gin.Context) {
	response := h.healthService.Liveness()
	c.JSON(http.StatusOK, gin.H{
		"status":      "alive",
		"timestamp":   response.Timestamp,
		"version":     response.Version,
		"uptime":      response.Uptime.String(),
		"environment": response.Environment,
	})
}

// Readiness returns the readiness status
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := h.healthService.Readiness(ctx)
	h.renderHealthResponse(c, response)
}

// DetailedHealth returns detailed health information including system metrics
func (h *HealthHandler) DetailedHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response := h.healthService.Check(ctx)
	status := h.mapHealthStatusToHTTP(response.Status)

	c.JSON(status, response)
}

// SystemInfo returns system information only
func (h *HealthHandler) SystemInfo(c *gin.Context) {
	response := h.healthService.Liveness()
	c.JSON(http.StatusOK, gin.H{
		"version":     response.Version,
		"uptime":      response.Uptime.String(),
		"environment": response.Environment,
		"system":      response.System,
		"timestamp":   response.Timestamp,
	})
}

// renderHealthResponse renders a health response with proper status mapping
func (h *HealthHandler) renderHealthResponse(c *gin.Context, response services.HealthResponse) {
	status := h.mapHealthStatusToHTTP(response.Status)
	c.JSON(status, gin.H{
		"status":      string(response.Status),
		"timestamp":   response.Timestamp,
		"version":     response.Version,
		"uptime":      response.Uptime.String(),
		"environment": response.Environment,
		"checks":      response.Checks,
	})
}

// mapHealthStatusToHTTP maps health status to HTTP status code
func (h *HealthHandler) mapHealthStatusToHTTP(status services.HealthStatus) int {
	switch status {
	case services.HealthStatusHealthy:
		return http.StatusOK
	case services.HealthStatusDegraded:
		return http.StatusOK // Still OK but with warnings
	case services.HealthStatusUnhealthy:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// HealthMiddleware is a middleware that adds health check headers
func HealthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add health-related headers
		c.Header("X-Health-Check", "available")
		c.Header("X-Health-Endpoints", "/health, /health/live, /health/ready")
		c.Next()
	}
}

// HealthMetricsMiddleware tracks request metrics for health monitoring
func HealthMetricsMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format for health monitoring
		return fmt.Sprintf("[HEALTH] %v | %3d | %13v | %15s | %-7s %#v\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	})
}

// PingHandler provides a simple ping endpoint
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"time":    time.Now().Unix(),
	})
}

// StatusHandler provides application status
func StatusHandler(c *gin.Context) {
	// Get some basic stats
	stats := gin.H{
		"status":    "running",
		"timestamp": time.Now().Unix(),
		"version":   c.GetString("app_version"),
	}

	// Add request ID if available
	if requestID := c.GetString("request_id"); requestID != "" {
		stats["request_id"] = requestID
	}

	c.JSON(http.StatusOK, stats)
}

// VersionHandler returns version information
func VersionHandler(c *gin.Context) {
	version := c.GetString("app_version")
	if version == "" {
		version = "unknown"
	}

	buildTime := c.GetString("build_time")
	commitHash := c.GetString("commit_hash")

	c.JSON(http.StatusOK, gin.H{
		"version":     version,
		"build_time":  buildTime,
		"commit_hash": commitHash,
		"go_version":  runtime.Version(),
		"timestamp":   time.Now().Unix(),
	})
}
