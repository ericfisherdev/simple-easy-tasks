package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSanitizedErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		err                error
		name               string
		expectedErrorType  string
		expectedCode       string
		expectedStatusCode int
		shouldHaveDetails  bool
		shouldHaveCorrID   bool
	}{
		{
			name: "validation error should be sanitized",
			err: domain.NewValidationError(
				"INVALID_FIELD",
				"Field validation failed",
				map[string]interface{}{"field": "email"},
			),
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorType:  "VALIDATION_ERROR",
			expectedCode:       "INVALID_FIELD",
			shouldHaveDetails:  true,
			shouldHaveCorrID:   true,
		},
		{
			name:               "authentication error should be sanitized",
			err:                domain.NewAuthenticationError("INVALID_CREDENTIALS", "Invalid username or password"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorType:  "AUTHENTICATION_ERROR",
			expectedCode:       "INVALID_CREDENTIALS",
			shouldHaveDetails:  false,
			shouldHaveCorrID:   true,
		},
		{
			name:               "unknown error should be sanitized",
			err:                assert.AnError,
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
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			c.Request = req

			SanitizedErrorResponse(c, tt.err)

			// Content-Type should be JSON
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Check that correlation ID is present in response
			if tt.shouldHaveCorrID {
				corrID := c.GetString("correlation_id")
				assert.NotEmpty(t, corrID)
				assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
			}

			// Parse JSON response for type-safe assertions
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Response should be valid JSON")

			// Assert basic response structure
			success, exists := response["success"].(bool)
			assert.True(t, exists, "success field should exist and be boolean")
			assert.False(t, success, "success should be false")

			// Assert correlation_id exists and is non-empty
			correlationID, exists := response["correlation_id"].(string)
			assert.True(t, exists, "correlation_id should exist and be string")
			assert.NotEmpty(t, correlationID, "correlation_id should not be empty")

			// Assert error structure
			errorMap, exists := response["error"].(map[string]interface{})
			assert.True(t, exists, "error field should exist and be an object")

			// Assert error.type and error.code
			assert.Equal(t, tt.expectedErrorType, errorMap["type"], "error.type should match expected")
			assert.Equal(t, tt.expectedCode, errorMap["code"], "error.code should match expected")

			// Assert error.details is absent or nil (not exposed to clients)
			details, hasDetails := errorMap["details"]
			if hasDetails {
				assert.Nil(t, details, "error.details should be nil if present")
			}

			// Ensure no raw error details are exposed
			assert.NotContains(t, w.Body.String(), "stack trace")
			// Removed the "internal error" check as requested
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
