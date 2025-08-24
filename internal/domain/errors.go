package domain

import "fmt"

// ErrorType represents the type of domain error
type ErrorType string

const (
	// ValidationError represents validation failures
	ValidationError ErrorType = "VALIDATION_ERROR"
	// NotFoundError represents resource not found
	NotFoundError ErrorType = "NOT_FOUND_ERROR"
	// ConflictError represents resource conflicts
	ConflictError ErrorType = "CONFLICT_ERROR"
	// AuthenticationError represents authentication failures
	AuthenticationError ErrorType = "AUTHENTICATION_ERROR"
	// AuthorizationError represents authorization failures
	AuthorizationError ErrorType = "AUTHORIZATION_ERROR"
	// InternalError represents internal system errors
	InternalError ErrorType = "INTERNAL_ERROR"
	// ExternalServiceError represents external service failures
	ExternalServiceError ErrorType = "EXTERNAL_SERVICE_ERROR"
)

// DomainError represents a domain-specific error with additional context
type DomainError struct {
	Type    ErrorType              `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(code, message string, details map[string]interface{}) *DomainError {
	return &DomainError{
		Type:    ValidationError,
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(code, message string) *DomainError {
	return &DomainError{
		Type:    NotFoundError,
		Code:    code,
		Message: message,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(code, message string) *DomainError {
	return &DomainError{
		Type:    ConflictError,
		Code:    code,
		Message: message,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(code, message string) *DomainError {
	return &DomainError{
		Type:    AuthenticationError,
		Code:    code,
		Message: message,
	}
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(code, message string) *DomainError {
	return &DomainError{
		Type:    AuthorizationError,
		Code:    code,
		Message: message,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(code, message string, cause error) *DomainError {
	return &DomainError{
		Type:    InternalError,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewExternalServiceError creates a new external service error
func NewExternalServiceError(code, message string, cause error) *DomainError {
	return &DomainError{
		Type:    ExternalServiceError,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
