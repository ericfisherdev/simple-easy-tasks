// Package middleware provides HTTP middleware functions.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	
	"simple-easy-tasks/internal/domain"
)

// RecoveryConfig holds configuration for the recovery middleware.
type RecoveryConfig struct {
	// HandleRecovery is a custom function to handle panic recovery.
	// If nil, the default handler will be used.
	HandleRecovery func(c *gin.Context, err interface{})
	// PrintStack determines whether to print stack trace to logs.
	PrintStack bool
	// IncludeStackInResponse determines whether to include stack trace in error response (dev only).
	IncludeStackInResponse bool
}

// RecoveryMiddleware returns a panic recovery middleware with custom configuration.
func RecoveryMiddleware(config RecoveryConfig) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if config.HandleRecovery != nil {
			config.HandleRecovery(c, recovered)
			return
		}

		// Default recovery handling
		requestID := GetRequestID(c)
		stack := debug.Stack()

		// Print stack trace if configured
		if config.PrintStack {
			fmt.Printf("[PANIC RECOVERY] Request ID: %s\nPanic: %v\nStack:\n%s\n", requestID, recovered, stack)
		}

		// Create a panic error and use SanitizedErrorResponse for proper error handling
		panicErr := domain.NewInternalError(
			"PANIC_RECOVERED", 
			fmt.Sprintf("Panic recovered: %v", recovered), 
			fmt.Errorf("panic: %v", recovered),
		)
		
		sanitizedErrorResponse(c, panicErr)
	})
}

// DefaultRecoveryMiddleware returns a recovery middleware with sensible defaults for production.
func DefaultRecoveryMiddleware() gin.HandlerFunc {
	return RecoveryMiddleware(RecoveryConfig{
		PrintStack:             true,
		IncludeStackInResponse: false,
	})
}

// DevelopmentRecoveryMiddleware returns a recovery middleware configured for development.
func DevelopmentRecoveryMiddleware() gin.HandlerFunc {
	return RecoveryMiddleware(RecoveryConfig{
		PrintStack:             true,
		IncludeStackInResponse: true,
	})
}

// ProductionRecoveryMiddleware returns a recovery middleware configured for production.
func ProductionRecoveryMiddleware() gin.HandlerFunc {
	return RecoveryMiddleware(RecoveryConfig{
		PrintStack:             true,
		IncludeStackInResponse: false,
		HandleRecovery: func(c *gin.Context, err interface{}) {
			requestID := GetRequestID(c)

			// Log panic for monitoring systems
			fmt.Printf("[PANIC] Request ID: %s, Path: %s, Method: %s, Panic: %v\n",
				requestID, c.Request.URL.Path, c.Request.Method, err)

			// Return clean error response
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"type":       "INTERNAL_ERROR",
					"code":       "SERVICE_UNAVAILABLE",
					"message":    "Service temporarily unavailable",
					"request_id": requestID,
				},
			})
		},
	})
}
