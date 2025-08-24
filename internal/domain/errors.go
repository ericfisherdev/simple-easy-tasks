package domain

import "fmt"

// ErrorType represents the type of domain error.
type ErrorType string

const (
	// ValidationError represents validation failures.
	ValidationError ErrorType = "VALIDATION_ERROR"
	// NotFoundError represents resource not found.
	NotFoundError ErrorType = "NOT_FOUND_ERROR"
	// ConflictError represents resource conflicts.
	ConflictError ErrorType = "CONFLICT_ERROR"
	// AuthenticationError represents authentication failures.
	AuthenticationError ErrorType = "AUTHENTICATION_ERROR"
	// AuthorizationError represents authorization failures
	AuthorizationError ErrorType = "AUTHORIZATION_ERROR"
	// InternalError represents internal system errors
	InternalError ErrorType = "INTERNAL_ERROR"
	// ExternalServiceError represents external service failures
	ExternalServiceError ErrorType = "EXTERNAL_SERVICE_ERROR"
)

// Error represents a domain-specific error with additional context.
// This follows Go naming conventions to avoid stuttering with the package name.
type Error struct {
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
	Type    ErrorType              `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(code, message string, details map[string]interface{}) *Error {
	return &Error{
		Type:    ValidationError,
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(code, message string) *Error {
	return &Error{
		Type:    NotFoundError,
		Code:    code,
		Message: message,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(code, message string) *Error {
	return &Error{
		Type:    ConflictError,
		Code:    code,
		Message: message,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(code, message string) *Error {
	return &Error{
		Type:    AuthenticationError,
		Code:    code,
		Message: message,
	}
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(code, message string) *Error {
	return &Error{
		Type:    AuthorizationError,
		Code:    code,
		Message: message,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(code, message string, cause error) *Error {
	return &Error{
		Type:    InternalError,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewExternalServiceError creates a new external service error
func NewExternalServiceError(code, message string, cause error) *Error {
	return &Error{
		Type:    ExternalServiceError,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ValidateRequired validates that a required field is not empty
func ValidateRequired(field, value, errorCode, errorMessage string) *Error {
	if value == "" {
		return NewValidationError(errorCode, errorMessage, map[string]interface{}{
			"field": field,
		})
	}
	return nil
}

// ValidateEnum validates that a value is one of the allowed options
func ValidateEnum(field, value, errorCode, errorMessage string, allowedValues ...string) *Error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	return NewValidationError(errorCode, errorMessage, map[string]interface{}{
		"field": field,
		"value": value,
	})
}
