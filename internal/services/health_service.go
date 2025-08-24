// Package services provides health monitoring and other service implementations.
package services

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is fully operational.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates the component is not operational.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded indicates the component has issues but is still functional.
	HealthStatusDegraded HealthStatus = "degraded"
)

// HealthCheck represents a single health check.
type HealthCheck struct {
	LastChecked time.Time              `json:"last_checked"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Name        string                 `json:"name"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Status      HealthStatus           `json:"status"`
	Duration    time.Duration          `json:"duration"`
}

// HealthResponse represents the overall health response.
type HealthResponse struct {
	Timestamp   time.Time              `json:"timestamp"`
	System      map[string]interface{} `json:"system"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment"`
	Status      HealthStatus           `json:"status"`
	Checks      []HealthCheck          `json:"checks"`
	Uptime      time.Duration          `json:"uptime"`
}

// HealthChecker defines the interface for health checkers.
type HealthChecker interface {
	Check(ctx context.Context) HealthCheck
	Name() string
}

// HealthService manages health checks for the application.
type HealthService struct {
	startTime time.Time
	version   string
	env       string
	checkers  []HealthChecker
}

// NewHealthService creates a new health service.
func NewHealthService(version, env string) *HealthService {
	return &HealthService{
		checkers:  make([]HealthChecker, 0),
		startTime: time.Now(),
		version:   version,
		env:       env,
	}
}

// RegisterChecker registers a health checker.
func (h *HealthService) RegisterChecker(checker HealthChecker) {
	h.checkers = append(h.checkers, checker)
}

// Check performs all health checks and returns the overall health status.
func (h *HealthService) Check(ctx context.Context) HealthResponse {
	checks := make([]HealthCheck, 0, len(h.checkers))
	overallStatus := HealthStatusHealthy

	// Perform all health checks
	for _, checker := range h.checkers {
		start := time.Now()
		check := checker.Check(ctx)
		check.Duration = time.Since(start)
		check.LastChecked = time.Now()

		checks = append(checks, check)

		// Update overall status
		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if check.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	return HealthResponse{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Version:     h.version,
		Uptime:      time.Since(h.startTime),
		Checks:      checks,
		System:      h.getSystemInfo(),
		Environment: h.env,
	}
}

// getSystemInfo returns system information.
func (h *HealthService) getSystemInfo() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"cpu_count":  runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc_bytes":   memStats.Alloc,
			"total_alloc":   memStats.TotalAlloc,
			"sys_bytes":     memStats.Sys,
			"heap_alloc":    memStats.HeapAlloc,
			"heap_sys":      memStats.HeapSys,
			"heap_idle":     memStats.HeapIdle,
			"heap_inuse":    memStats.HeapInuse,
			"heap_released": memStats.HeapReleased,
			"heap_objects":  memStats.HeapObjects,
			"gc_cycles":     memStats.NumGC,
			"last_gc":       time.Unix(0, int64(memStats.LastGC)), // #nosec G115 -- LastGC is always positive
		},
	}
}

// Liveness returns a simple liveness probe (application is running).
func (h *HealthService) Liveness() HealthResponse {
	return HealthResponse{
		Status:      HealthStatusHealthy,
		Timestamp:   time.Now(),
		Version:     h.version,
		Uptime:      time.Since(h.startTime),
		Environment: h.env,
		System: map[string]interface{}{
			"status": "alive",
		},
	}
}

// Readiness returns readiness probe (application is ready to serve traffic).
func (h *HealthService) Readiness(ctx context.Context) HealthResponse {
	// For readiness, we only check critical dependencies
	criticalCheckers := make([]HealthChecker, 0)
	for _, checker := range h.checkers {
		// Only include critical checkers (database, essential services)
		if isCriticalChecker(checker) {
			criticalCheckers = append(criticalCheckers, checker)
		}
	}

	checks := make([]HealthCheck, 0, len(criticalCheckers))
	overallStatus := HealthStatusHealthy

	for _, checker := range criticalCheckers {
		start := time.Now()
		check := checker.Check(ctx)
		check.Duration = time.Since(start)
		check.LastChecked = time.Now()

		checks = append(checks, check)

		if check.Status != HealthStatusHealthy {
			overallStatus = HealthStatusUnhealthy
		}
	}

	return HealthResponse{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Version:     h.version,
		Uptime:      time.Since(h.startTime),
		Checks:      checks,
		Environment: h.env,
	}
}

// isCriticalChecker determines if a checker is critical for readiness.
func isCriticalChecker(checker HealthChecker) bool {
	criticalCheckers := []string{
		"database",
		"pocketbase",
		"storage",
	}

	name := checker.Name()
	for _, critical := range criticalCheckers {
		if name == critical {
			return true
		}
	}
	return false
}

// HTTPHealthChecker checks HTTP endpoints.
type HTTPHealthChecker struct {
	name     string
	url      string
	timeout  time.Duration
	expected int
}

// NewHTTPHealthChecker creates a new HTTP health checker.
func NewHTTPHealthChecker(name, url string, timeout time.Duration, expectedStatus int) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		name:     name,
		url:      url,
		timeout:  timeout,
		expected: expectedStatus,
	}
}

// Name returns the checker name.
func (h *HTTPHealthChecker) Name() string {
	return h.name
}

// Check performs the HTTP health check
func (h *HTTPHealthChecker) Check(ctx context.Context) HealthCheck {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return HealthCheck{
			Name:   h.name,
			Status: HealthStatusUnhealthy,
			Error:  fmt.Sprintf("failed to create request: %v", err),
		}
	}

	client := &http.Client{Timeout: h.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return HealthCheck{
			Name:   h.name,
			Status: HealthStatusUnhealthy,
			Error:  fmt.Sprintf("request failed: %v", err),
		}
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			// Log error but don't fail the health check
			_ = cerr
		}
	}()

	if resp.StatusCode == h.expected {
		return HealthCheck{
			Name:    h.name,
			Status:  HealthStatusHealthy,
			Message: fmt.Sprintf("HTTP %d OK", resp.StatusCode),
		}
	}

	return HealthCheck{
		Name:   h.name,
		Status: HealthStatusUnhealthy,
		Error:  fmt.Sprintf("expected HTTP %d, got %d", h.expected, resp.StatusCode),
	}
}
