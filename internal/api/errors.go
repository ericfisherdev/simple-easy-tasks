package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"simple-easy-tasks/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorSanitizer provides safe error handling that prevents information disclosure
type ErrorSanitizer struct {
	logger *slog.Logger
}

// NewErrorSanitizer creates a new error sanitizer with structured logging
func NewErrorSanitizer(logger *slog.Logger) *ErrorSanitizer {
	if logger == nil {
		logger = slog.Default()
	}
	return &ErrorSanitizer{logger: logger}
}

// SanitizedErrorResponse provides safe error responses that prevent information disclosure
// while logging detailed errors server-side with correlation IDs
func (s *ErrorSanitizer) SanitizedErrorResponse(c *gin.Context, err error) {
	// Generate correlation ID for tracking
	correlationID := s.getOrCreateCorrelationID(c)

	// Extract domain error information if available
	domainErr, isDomainError := err.(*domain.Error)

	// Log detailed error server-side with correlation ID
	s.logErrorWithContext(c, err, correlationID, isDomainError, domainErr)

	// Return sanitized error to client
	statusCode, response := s.sanitizeErrorForClient(domainErr, isDomainError, correlationID)
	c.JSON(statusCode, response)
}

// getOrCreateCorrelationID gets existing correlation ID from context or creates new one
func (s *ErrorSanitizer) getOrCreateCorrelationID(c *gin.Context) string {
	// Check if correlation ID already exists in context
	if id, exists := c.Get("correlation_id"); exists {
		if strID, ok := id.(string); ok {
			return strID
		}
	}

	// Check request headers for existing correlation ID
	if id := c.GetHeader("X-Correlation-ID"); id != "" {
		c.Set("correlation_id", id)
		return id
	}

	// Generate new correlation ID
	correlationID := uuid.New().String()
	c.Set("correlation_id", correlationID)
	c.Header("X-Correlation-ID", correlationID)
	return correlationID
}

// logErrorWithContext logs detailed error information server-side
func (s *ErrorSanitizer) logErrorWithContext(c *gin.Context, err error, correlationID string, isDomainError bool, domainErr *domain.Error) {
	// Base log attributes
	attrs := []slog.Attr{
		slog.String("correlation_id", correlationID),
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("remote_addr", c.ClientIP()),
		slog.String("user_agent", c.Request.UserAgent()),
	}

	// Add user context if available
	if user, exists := c.Get("user"); exists {
		if userMap, ok := user.(map[string]interface{}); ok {
			if userID, ok := userMap["id"].(string); ok {
				attrs = append(attrs, slog.String("user_id", userID))
			}
		}
	}

	if isDomainError {
		// Log structured domain error
		attrs = append(attrs,
			slog.String("error_type", string(domainErr.Type)),
			slog.String("error_code", domainErr.Code),
			slog.String("error_message", domainErr.Message),
		)

		// Add cause chain if available
		if domainErr.Cause != nil {
			attrs = append(attrs, slog.String("underlying_error", domainErr.Cause.Error()))
		}

		// Add details if available (be careful not to log sensitive data)
		if len(domainErr.Details) > 0 {
			for key, value := range domainErr.Details {
				// Only log non-sensitive details
				if !isSensitiveField(key) {
					attrs = append(attrs, slog.Any(fmt.Sprintf("detail_%s", key), value))
				}
			}
		}

		// Convert []slog.Attr to []any for logging
		logArgs := make([]any, len(attrs))
		for i, attr := range attrs {
			logArgs[i] = attr
		}
		s.logger.ErrorContext(context.Background(), "Domain error occurred", logArgs...)
	} else {
		// Log unexpected system error
		attrs = append(attrs, slog.String("error", err.Error()))
		logArgs := make([]any, len(attrs))
		for i, attr := range attrs {
			logArgs[i] = attr
		}
		s.logger.ErrorContext(context.Background(), "Unexpected system error occurred", logArgs...)
	}
}

// sanitizeErrorForClient returns safe error response for client consumption
func (s *ErrorSanitizer) sanitizeErrorForClient(domainErr *domain.Error, isDomainError bool, correlationID string) (int, gin.H) {
	if isDomainError {
		statusCode := s.getStatusCodeForDomainError(domainErr.Type)

		// Return sanitized error message based on type
		response := gin.H{
			"success":        false,
			"correlation_id": correlationID,
			"error": map[string]interface{}{
				"type": domainErr.Type,
				"code": domainErr.Code,
			},
		}

		// Only include user-safe messages
		switch domainErr.Type {
		case domain.ValidationError:
			response["error"].(map[string]interface{})["message"] = "Invalid input provided"
			// Include field-level validation details if available and safe
			if domainErr.Details != nil {
				if field, ok := domainErr.Details["field"]; ok {
					response["error"].(map[string]interface{})["field"] = field
				}
			}
		case domain.NotFoundError:
			response["error"].(map[string]interface{})["message"] = "Requested resource not found"
		case domain.ConflictError:
			response["error"].(map[string]interface{})["message"] = "Resource conflict occurred"
		case domain.AuthenticationError:
			response["error"].(map[string]interface{})["message"] = "Authentication failed"
		case domain.AuthorizationError:
			response["error"].(map[string]interface{})["message"] = "Access denied"
		case domain.ExternalServiceError:
			response["error"].(map[string]interface{})["message"] = "External service temporarily unavailable"
		default:
			response["error"].(map[string]interface{})["message"] = "An error occurred while processing your request"
		}

		return statusCode, response
	}

	// For non-domain errors, return generic message
	return http.StatusInternalServerError, gin.H{
		"success":        false,
		"correlation_id": correlationID,
		"error": map[string]interface{}{
			"type":    "INTERNAL_ERROR",
			"code":    "SYSTEM_ERROR",
			"message": "An unexpected error occurred. Please try again later.",
		},
	}
}

// getStatusCodeForDomainError maps domain error types to HTTP status codes
func (s *ErrorSanitizer) getStatusCodeForDomainError(errorType domain.ErrorType) int {
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
	case domain.ExternalServiceError:
		return http.StatusBadGateway
	case domain.InternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// isSensitiveField checks if a field contains sensitive information that shouldn't be logged
func isSensitiveField(field string) bool {
	sensitiveFields := map[string]bool{
		"password":        true,
		"token":           true,
		"secret":          true,
		"key":             true,
		"authorization":   true,
		"cookie":          true,
		"session":         true,
		"private_key":     true,
		"access_token":    true,
		"refresh_token":   true,
		"jwt":             true,
		"api_key":         true,
		"credit_card":     true,
		"ssn":             true,
		"social_security": true,
	}
	return sensitiveFields[field]
}
