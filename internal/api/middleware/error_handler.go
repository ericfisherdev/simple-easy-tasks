// Package middleware provides HTTP middleware functions.
package middleware

//nolint:gofumpt
import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"simple-easy-tasks/internal/domain"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerConfig holds configuration for error handling middleware.
type ErrorHandlerConfig struct {
	// CustomErrorHandler allows for custom error handling logic
	CustomErrorHandler func(c *gin.Context, err error)
	// IncludeStackTrace determines whether to include stack traces in error responses
	IncludeStackTrace bool
	// LogErrors determines whether errors should be logged
	LogErrors bool
}

// ErrorHandlerMiddleware returns a middleware that handles errors consistently.
func ErrorHandlerMiddleware(config ErrorHandlerConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Process any errors that occurred during request handling
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			if config.CustomErrorHandler != nil {
				config.CustomErrorHandler(c, err.Err)
				return
			}

			handleError(c, err.Err, config)
		}
	})
}

// DefaultErrorHandlerMiddleware returns an error handler with sensible defaults.
func DefaultErrorHandlerMiddleware() gin.HandlerFunc {
	return ErrorHandlerMiddleware(ErrorHandlerConfig{
		IncludeStackTrace: false,
		LogErrors:         true,
	})
}

// DevelopmentErrorHandlerMiddleware returns an error handler for development.
func DevelopmentErrorHandlerMiddleware() gin.HandlerFunc {
	return ErrorHandlerMiddleware(ErrorHandlerConfig{
		IncludeStackTrace: true,
		LogErrors:         true,
	})
}

// handleError processes and responds to errors.
func handleError(c *gin.Context, err error, config ErrorHandlerConfig) {
	requestID := GetRequestID(c)

	if config.LogErrors {
		logError(err, requestID, c.Request.Method, c.Request.URL.Path)
	}

	// Check if response has already been written
	if c.Writer.Written() {
		return
	}

	// Handle domain errors
	if domainErr, ok := err.(*domain.Error); ok {
		handleDomainError(c, domainErr, config)
		return
	}

	// Handle JSON parsing errors
	if jsonErr, ok := err.(*json.SyntaxError); ok {
		handleJSONError(c, jsonErr, requestID)
		return
	}

	// Handle validation errors from Gin binding
	if strings.Contains(err.Error(), "binding") || strings.Contains(err.Error(), "validation") {
		handleBindingError(c, err, requestID)
		return
	}

	// Handle generic errors
	handleGenericError(c, err, requestID, config)
}

// handleDomainError handles domain-specific errors.
func handleDomainError(c *gin.Context, domainErr *domain.Error, _ ErrorHandlerConfig) {
	statusCode := mapDomainErrorToHTTPStatus(domainErr.Type)
	requestID := GetRequestID(c)

	errorResponse := gin.H{
		"success": false,
		"error": gin.H{
			"type":       string(domainErr.Type),
			"code":       domainErr.Code,
			"message":    domainErr.Message,
			"request_id": requestID,
		},
	}

	if domainErr.Details != nil {
		if errorMap, ok := errorResponse["error"].(gin.H); ok {
			errorMap["details"] = domainErr.Details
		}
	}

	// Stack trace handling removed - not available in current domain.Error implementation
	// if config.IncludeStackTrace {
	//     errorResponse["error"].(gin.H)["stack_trace"] = "Stack trace not implemented"
	// }

	c.JSON(statusCode, errorResponse)
}

// handleJSONError handles JSON parsing errors.
func handleJSONError(c *gin.Context, jsonErr *json.SyntaxError, requestID string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error": gin.H{
			"type":       "VALIDATION_ERROR",
			"code":       "INVALID_JSON",
			"message":    "Invalid JSON format in request body",
			"details":    fmt.Sprintf("JSON syntax error at offset %d", jsonErr.Offset),
			"request_id": requestID,
		},
	})
}

// handleBindingError handles Gin binding/validation errors.
func handleBindingError(c *gin.Context, err error, requestID string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error": gin.H{
			"type":       "VALIDATION_ERROR",
			"code":       "BINDING_ERROR",
			"message":    "Request validation failed",
			"details":    err.Error(),
			"request_id": requestID,
		},
	})
}

// handleGenericError handles generic/unknown errors.
func handleGenericError(c *gin.Context, err error, requestID string, config ErrorHandlerConfig) {
	errorResponse := gin.H{
		"success": false,
		"error": gin.H{
			"type":       "INTERNAL_ERROR",
			"code":       "UNEXPECTED_ERROR",
			"message":    "An unexpected error occurred",
			"request_id": requestID,
		},
	}

	if config.IncludeStackTrace {
		if errorMap, ok := errorResponse["error"].(gin.H); ok {
			errorMap["details"] = err.Error()
			errorMap["stack_trace"] = string(debug.Stack())
		}
	}

	c.JSON(http.StatusInternalServerError, errorResponse)
}

// mapDomainErrorToHTTPStatus maps domain error types to HTTP status codes.
func mapDomainErrorToHTTPStatus(errorType domain.ErrorType) int {
	switch errorType {
	case domain.ValidationError:
		return http.StatusBadRequest
	case domain.NotFoundError:
		return http.StatusNotFound
	case domain.ConflictError:
		return http.StatusConflict
	case domain.AuthenticationError:
		return http.StatusUnauthorized
	case domain.AuthorizationError:
		return http.StatusForbidden
	case domain.InternalError:
		return http.StatusInternalServerError
	case domain.ExternalServiceError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// logError logs error details for debugging and monitoring.
func logError(err error, requestID, method, path string) {
	fmt.Printf("[ERROR] Request ID: %s, Method: %s, Path: %s, Error: %v\n",
		requestID, method, path, err)
}

// AbortWithError aborts the request with a domain error.
func AbortWithError(c *gin.Context, err *domain.Error) {
	_ = c.Error(err) // Error is logged by the error handler middleware
	c.Abort()
}

// AbortWithValidationError aborts the request with a validation error.
func AbortWithValidationError(c *gin.Context, code, message string, details map[string]interface{}) {
	err := domain.NewValidationError(code, message, details)
	AbortWithError(c, err)
}

// AbortWithNotFoundError aborts the request with a not found error.
func AbortWithNotFoundError(c *gin.Context, code, message string) {
	err := domain.NewNotFoundError(code, message)
	AbortWithError(c, err)
}

// AbortWithConflictError aborts the request with a conflict error.
func AbortWithConflictError(c *gin.Context, code, message string) {
	err := domain.NewConflictError(code, message)
	AbortWithError(c, err)
}

// AbortWithAuthenticationError aborts the request with an authentication error.
func AbortWithAuthenticationError(c *gin.Context, code, message string) {
	err := domain.NewAuthenticationError(code, message)
	AbortWithError(c, err)
}

// AbortWithAuthorizationError aborts the request with an authorization error.
func AbortWithAuthorizationError(c *gin.Context, code, message string) {
	err := domain.NewAuthorizationError(code, message)
	AbortWithError(c, err)
}

// AbortWithInternalError aborts the request with an internal error.
func AbortWithInternalError(c *gin.Context, code, message string, cause error) {
	err := domain.NewInternalError(code, message, cause)
	AbortWithError(c, err)
}
