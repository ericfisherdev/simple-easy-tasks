package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"simple-easy-tasks/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSanitizedErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		error              error
		expectedStatusCode int
		expectedErrorType  string
		expectedCode       string
		shouldHaveDetails  bool
		shouldHaveCorrID   bool
	}{
		{
			name:               "validation error should be sanitized",
			error:              domain.NewValidationError("INVALID_FIELD", "Field validation failed", map[string]interface{}{"field": "email"}),
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorType:  "VALIDATION_ERROR",
			expectedCode:       "INVALID_FIELD",
			shouldHaveDetails:  true,
			shouldHaveCorrID:   true,
		},
		{
			name:               "authentication error should be sanitized",
			error:              domain.NewAuthenticationError("INVALID_CREDENTIALS", "Invalid username or password"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorType:  "AUTHENTICATION_ERROR",
			expectedCode:       "INVALID_CREDENTIALS",
			shouldHaveDetails:  false,
			shouldHaveCorrID:   true,
		},
		{
			name:               "unknown error should be sanitized",
			error:              assert.AnError,
			expectedStatusCode: http.StatusInternalServerError,
			expectedErrorType:  "INTERNAL_ERROR",
			expectedCode:       "SYSTEM_ERROR",
			shouldHaveDetails:  false,
			shouldHaveCorrID:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a mock request so gin context works properly
			req, _ := http.NewRequest("GET", "/test", nil)
			c.Request = req

			SanitizedErrorResponse(c, tt.error)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Check that correlation ID is present in response
			if tt.shouldHaveCorrID {
				corrID := c.GetString("correlation_id")
				assert.NotEmpty(t, corrID)
				assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
			}

			// Verify response structure contains error information but no sensitive details
			assert.Contains(t, w.Body.String(), `"success":false`)
			assert.Contains(t, w.Body.String(), `"correlation_id"`)
			assert.Contains(t, w.Body.String(), tt.expectedErrorType)
			assert.Contains(t, w.Body.String(), tt.expectedCode)

			// Ensure no raw error details are exposed
			assert.NotContains(t, w.Body.String(), "stack trace")
			assert.NotContains(t, w.Body.String(), "internal error")
		})
	}
}

func TestErrorSanitization_SensitiveFields(t *testing.T) {
	sensitiveFields := []string{
		"password", "token", "secret", "key", "authorization",
		"cookie", "session", "private_key", "access_token",
		"refresh_token", "jwt", "api_key", "credit_card",
		"ssn", "social_security",
	}

	for _, field := range sensitiveFields {
		assert.True(t, isSensitiveField(field), "Field %s should be marked as sensitive", field)
	}

	// Test non-sensitive fields
	nonSensitiveFields := []string{
		"username", "email", "id", "name", "field", "value",
	}

	for _, field := range nonSensitiveFields {
		assert.False(t, isSensitiveField(field), "Field %s should not be marked as sensitive", field)
	}
}
