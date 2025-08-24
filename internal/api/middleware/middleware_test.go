package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		requestIDHeader string
		expectHeader    bool
	}{
		{
			name:         "generates new request ID when not provided",
			expectHeader: true,
		},
		{
			name:            "uses provided request ID",
			requestIDHeader: "test-request-id-123",
			expectHeader:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutil.NewTestRouter()
			router.Use(middleware.RequestIDMiddleware())
			router.GET("/test", func(c *gin.Context) {
				requestID := middleware.GetRequestID(c)
				c.JSON(http.StatusOK, gin.H{
					"request_id": requestID,
				})
			})

			helper := testutil.NewHTTPTestHelper(t, router)
			headers := make(map[string]string)
			if tt.requestIDHeader != "" {
				headers["X-Request-ID"] = tt.requestIDHeader
			}

			recorder := helper.GET("/test", headers)

			helper.AssertStatus(recorder, http.StatusOK)

			if tt.expectHeader {
				responseRequestID := recorder.Header().Get("X-Request-ID")
				if responseRequestID == "" {
					t.Error("Expected X-Request-ID header in response")
				}

				if tt.requestIDHeader != "" && responseRequestID != tt.requestIDHeader {
					t.Errorf("Expected request ID %s, got %s", tt.requestIDHeader, responseRequestID)
				}
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		method         string
		expectedOrigin string
		expectedStatus int
	}{
		{
			name:           "allows localhost origin",
			origin:         "http://localhost:3000",
			method:         "GET",
			expectedOrigin: "http://localhost:3000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "handles preflight request",
			origin:         "http://localhost:3000",
			method:         "OPTIONS",
			expectedOrigin: "http://localhost:3000",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "allows GET request",
			origin:         "http://localhost:8080",
			method:         "GET",
			expectedOrigin: "http://localhost:8080",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutil.NewTestRouter()
			router.Use(middleware.DefaultCORSMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})
			router.OPTIONS("/test", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			headers := map[string]string{
				"Origin": tt.origin,
			}

			var recorder *httptest.ResponseRecorder
			helper := testutil.NewHTTPTestHelper(t, router)

			if tt.method == "OPTIONS" {
				req, _ := http.NewRequestWithContext(context.Background(), "OPTIONS", "/test", nil)
				req.Header.Set("Origin", tt.origin)
				recorder = httptest.NewRecorder()
				router.ServeHTTP(recorder, req)
			} else {
				recorder = helper.GET("/test", headers)
			}

			helper.AssertStatus(recorder, tt.expectedStatus)

			if tt.expectedOrigin != "" {
				actualOrigin := recorder.Header().Get("Access-Control-Allow-Origin")
				if actualOrigin != tt.expectedOrigin {
					t.Errorf("Expected Access-Control-Allow-Origin %s, got %s", tt.expectedOrigin, actualOrigin)
				}
			}

			// Check for other CORS headers
			allowMethods := recorder.Header().Get("Access-Control-Allow-Methods")
			if allowMethods == "" {
				t.Error("Expected Access-Control-Allow-Methods header")
			}

			allowHeaders := recorder.Header().Get("Access-Control-Allow-Headers")
			if allowHeaders == "" {
				t.Error("Expected Access-Control-Allow-Headers header")
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		requestCount   int
		expectedStatus int
	}{
		{
			name:           "allows requests under limit",
			requestCount:   5,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blocks requests over limit",
			requestCount:   12, // More than 10 requests per minute
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutil.NewTestRouter()
			router.Use(middleware.DefaultRateLimitMiddleware(10)) // 10 requests per minute
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			helper := testutil.NewHTTPTestHelper(t, router)

			var lastStatus int
			for i := 0; i < tt.requestCount; i++ {
				recorder := helper.GET("/test", nil)
				lastStatus = recorder.Code

				// If we hit rate limit, break early
				if lastStatus == http.StatusTooManyRequests {
					break
				}
			}

			if tt.expectedStatus == http.StatusTooManyRequests && lastStatus != http.StatusTooManyRequests {
				t.Errorf("Expected rate limit to be hit, but got status %d", lastStatus)
			} else if tt.expectedStatus == http.StatusOK && lastStatus != http.StatusOK {
				t.Errorf("Expected requests to be allowed, but got status %d", lastStatus)
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		panicValue     interface{}
		expectedStatus int
	}{
		{
			name:           "handles panic with string",
			panicValue:     "test panic",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "handles panic with error",
			panicValue:     http.ErrAbortHandler,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := testutil.NewTestRouter()
			router.Use(middleware.DefaultRecoveryMiddleware())
			router.GET("/panic", func(_ *gin.Context) {
				panic(tt.panicValue)
			})

			helper := testutil.NewHTTPTestHelper(t, router)
			recorder := helper.GET("/panic", nil)

			helper.AssertStatus(recorder, tt.expectedStatus)

			// Check that response is valid JSON
			responseBody := recorder.Body.String()
			if !strings.Contains(responseBody, "success") {
				t.Error("Expected structured error response")
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	router := testutil.NewTestRouter()
	router.Use(middleware.DefaultLoggingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	helper := testutil.NewHTTPTestHelper(t, router)
	recorder := helper.GET("/test", nil)

	helper.AssertStatus(recorder, http.StatusOK)

	// The logging middleware should not affect the response
	expectedBody := map[string]interface{}{
		"message": "ok",
	}
	helper.AssertJSON(recorder, expectedBody)
}
