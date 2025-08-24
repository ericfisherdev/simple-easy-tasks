// Package domain provides domain-specific error handling and business logic types.
package domain

import (
	"errors"
	"log"
	"net/http"
)

// ErrorHandler handles domain errors and converts them to appropriate HTTP responses.
type ErrorHandler interface {
	HandleError(err error) (statusCode int, response interface{})
	LogError(err error)
}

// DefaultErrorHandler is the default implementation of ErrorHandler.
type DefaultErrorHandler struct {
	logger *log.Logger
}

// NewDefaultErrorHandler creates a new default error handler.
func NewDefaultErrorHandler(logger *log.Logger) *DefaultErrorHandler {
	return &DefaultErrorHandler{
		logger: logger,
	}
}

// APIError represents an API error response
type APIError struct {
	Type    string                 `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HandleError converts domain errors to HTTP status codes and API error responses
func (h *DefaultErrorHandler) HandleError(err error) (statusCode int, response interface{}) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return h.handleDomainError(domainErr)
	}

	// Handle non-domain errors
	h.LogError(err)
	return http.StatusInternalServerError, APIError{
		Type:    string(InternalError),
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
	}
}

// handleDomainError handles specific domain errors
func (h *DefaultErrorHandler) handleDomainError(err *DomainError) (statusCode int, response interface{}) {
	h.LogError(err)

	apiError := APIError{
		Type:    string(err.Type),
		Code:    err.Code,
		Message: err.Message,
		Details: err.Details,
	}

	switch err.Type {
	case ValidationError:
		return http.StatusBadRequest, apiError
	case NotFoundError:
		return http.StatusNotFound, apiError
	case ConflictError:
		return http.StatusConflict, apiError
	case AuthenticationError:
		return http.StatusUnauthorized, apiError
	case AuthorizationError:
		return http.StatusForbidden, apiError
	case ExternalServiceError:
		return http.StatusBadGateway, apiError
	default:
		return http.StatusInternalServerError, apiError
	}
}

// LogError logs the error with appropriate level based on error type
func (h *DefaultErrorHandler) LogError(err error) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Type {
		case ValidationError, NotFoundError, ConflictError:
			// Client errors - log at info level
			h.logger.Printf("Client error: %v", err)
		case AuthenticationError, AuthorizationError:
			// Auth errors - log at warning level
			h.logger.Printf("Auth error: %v", err)
		default:
			// Server errors - log at error level
			h.logger.Printf("Server error: %v", err)
		}
	} else {
		h.logger.Printf("Unexpected error: %v", err)
	}
}
