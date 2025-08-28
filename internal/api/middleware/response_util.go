// Package middleware provides HTTP middleware functions.
package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

var (
	middlewareLoggerOnce sync.Once
	middlewareLogger     *slog.Logger
)

// getMiddlewareLogger returns a singleton logger for middleware error handling
func getMiddlewareLogger() *slog.Logger {
	middlewareLoggerOnce.Do(func() {
		middlewareLogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	})
	return middlewareLogger
}

// sanitizedErrorResponse provides sanitized error response for middleware
// This provides basic error sanitization without creating import cycles
func sanitizedErrorResponse(c *gin.Context, err error) {
	logger := getMiddlewareLogger()

	// Log the error for debugging
	logger.Error("Middleware error occurred",
		"error", err.Error(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)

	// Create a basic sanitized response
	response := gin.H{
		"success": false,
		"error": gin.H{
			"type":    "INTERNAL_ERROR",
			"code":    "MIDDLEWARE_ERROR",
			"message": "An error occurred processing your request",
		},
	}

	// If it's a domain error, we can safely expose some details
	if domainErr, ok := err.(*domain.Error); ok {
		if errorInfo, ok := response["error"].(gin.H); ok {
			errorInfo["code"] = domainErr.Code
			errorInfo["type"] = string(domainErr.Type)
			// Only expose user-safe messages for domain errors
			if domainErr.Type == domain.ValidationError {
				errorInfo["message"] = domainErr.Message
			}
		}
	}

	statusCode := http.StatusInternalServerError
	if domainErr, ok := err.(*domain.Error); ok {
		switch domainErr.Type {
		case domain.ValidationError:
			statusCode = http.StatusBadRequest
		case domain.AuthenticationError:
			statusCode = http.StatusUnauthorized
		case domain.AuthorizationError:
			statusCode = http.StatusForbidden
		case domain.NotFoundError:
			statusCode = http.StatusNotFound
		}
	}

	c.JSON(statusCode, response)
}
