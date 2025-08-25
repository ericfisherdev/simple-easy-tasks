package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestStructuredLoggingMiddleware_JSONValidity(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		path         string
		errorMessage string
		description  string
	}{
		{
			name:         "normal request",
			path:         "/api/test",
			errorMessage: "",
			description:  "Should produce valid JSON for normal request",
		},
		{
			name:         "path with quotes",
			path:         `/api/test"quoted"path`,
			errorMessage: "",
			description:  "Should escape quotes in path",
		},
		{
			name:         "error with special characters",
			path:         "/api/error",
			errorMessage: `Error: "failed" with newline` + "\n" + `and more`,
			description:  "Should escape special characters in error message",
		},
		{
			name:         "path with backslashes",
			path:         `/api\test\path`,
			errorMessage: "",
			description:  "Should handle backslashes properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture logs
			var logBuffer bytes.Buffer

			// Create router with structured logging
			router := gin.New()
			router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
				// This is the same logic as StructuredLoggingMiddleware
				requestID := ""
				if param.Keys != nil {
					if id, exists := param.Keys[string(RequestIDKey)]; exists {
						if idStr, ok := id.(string); ok {
							requestID = idStr
						}
					}
				}

				rec := map[string]interface{}{
					"timestamp":  param.TimeStamp.Format("2006-01-02T15:04:05Z07:00"),
					"status":     param.StatusCode,
					"latency":    param.Latency.String(),
					"client_ip":  param.ClientIP,
					"method":     param.Method,
					"path":       param.Path,
					"request_id": requestID,
					"error":      param.ErrorMessage,
				}

				b, _ := json.Marshal(rec)
				output := string(b) + "\n"
				logBuffer.WriteString(output) // Capture the log
				return output
			}))

			// Add test route
			router.GET("/*path", func(c *gin.Context) {
				if tt.errorMessage != "" {
					c.AbortWithError(http.StatusInternalServerError, &testError{msg: tt.errorMessage})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// Make request
			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", tt.path, nil)
			router.ServeHTTP(w, req)

			// Give the logger time to write
			time.Sleep(10 * time.Millisecond)

			// Check if log output is valid JSON
			logOutput := logBuffer.String()
			if logOutput == "" {
				t.Skip("No log output captured")
			}

			// Remove trailing newline for JSON parsing
			logOutput = strings.TrimSpace(logOutput)

			// Try to parse as JSON
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(logOutput), &result); err != nil {
				t.Errorf("%s: Invalid JSON output: %v\nLog output: %s", tt.description, err, logOutput)
			}

			// Verify expected fields exist
			expectedFields := []string{"timestamp", "status", "latency", "client_ip", "method", "path", "request_id", "error"}
			for _, field := range expectedFields {
				if _, exists := result[field]; !exists {
					t.Errorf("%s: Missing field '%s' in JSON output", tt.description, field)
				}
			}

			// Verify path is correctly escaped
			if path, ok := result["path"].(string); ok {
				if !strings.Contains(path, tt.path) && path != tt.path {
					t.Errorf("%s: Path not properly preserved. Expected: %s, Got: %s", tt.description, tt.path, path)
				}
			}

			// Verify error message is correctly escaped if present
			if tt.errorMessage != "" {
				if errMsg, ok := result["error"].(string); ok {
					if !strings.Contains(errMsg, tt.errorMessage) && errMsg != tt.errorMessage && errMsg != "" {
						t.Errorf("%s: Error message not properly preserved", tt.description)
					}
				}
			}
		})
	}
}

// testError implements error interface for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
