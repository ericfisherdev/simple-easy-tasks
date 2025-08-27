// Package api provides shared utilities for API handlers.
//
// Error Handling:
// All handlers should use SanitizedErrorResponse for consistent error handling,
// security sanitization, and structured logging. This prevents internal message
// leakage and ensures proper error tracking across the application.
//
// Usage:
//
//	api.SanitizedErrorResponse(c, err)
//
// Avoid direct c.JSON calls with error payloads - use SanitizedErrorResponse instead.
package api

//nolint:gofumpt
import (
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	defaultSanitizer *ErrorSanitizer
	sanitizerOnce    sync.Once
)

// getDefaultSanitizer creates a singleton error sanitizer with structured logging
func getDefaultSanitizer() *ErrorSanitizer {
	sanitizerOnce.Do(func() {
		// Create structured logger for error handling
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
		defaultSanitizer = NewErrorSanitizer(logger)
	})
	return defaultSanitizer
}

// ErrorResponse handles domain errors with improved security and logging.
// Deprecated: Use SanitizedErrorResponse for better security.
// This function is kept for backward compatibility but should be replaced.
func ErrorResponse(c *gin.Context, err error) {
	// Use the new sanitized error response for better security
	getDefaultSanitizer().SanitizedErrorResponse(c, err)
}

// SanitizedErrorResponse handles errors with security-focused sanitization and structured logging
func SanitizedErrorResponse(c *gin.Context, err error) {
	getDefaultSanitizer().SanitizedErrorResponse(c, err)
}

// SuccessResponse returns a standardized success response.
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// CreatedResponse returns a standardized created response.
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}
