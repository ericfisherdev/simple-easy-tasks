// Package middleware provides HTTP middleware functions.
package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

// LoggingConfig holds configuration for the logging middleware.
type LoggingConfig struct {
	Output     io.Writer
	TimeFormat string
	SkipPaths  []string
}

// LoggingMiddleware returns a logging middleware with custom configuration.
func LoggingMiddleware(config LoggingConfig) gin.HandlerFunc {
	if config.Output == nil {
		config.Output = os.Stdout
	}

	if config.TimeFormat == "" {
		config.TimeFormat = "2006/01/02 - 15:04:05"
	}

	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Skip paths that shouldn't be logged
			for _, path := range config.SkipPaths {
				if param.Path == path {
					return ""
				}
			}

			// Get request ID if available
			requestID := ""
			if param.Keys != nil {
				if id, exists := param.Keys[string(RequestIDKey)]; exists {
					if idStr, ok := id.(string); ok {
						requestID = fmt.Sprintf(" | ReqID: %s", idStr)
					}
				}
			}

			// Custom log format with request ID
			return fmt.Sprintf("[API] %v | %3d | %13v | %15s | %-7s %#v%s\n%s",
				param.TimeStamp.Format(config.TimeFormat),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				requestID,
				param.ErrorMessage,
			)
		},
		Output:    config.Output,
		SkipPaths: config.SkipPaths,
	})
}

// DefaultLoggingMiddleware returns a logging middleware with sensible defaults.
func DefaultLoggingMiddleware() gin.HandlerFunc {
	return LoggingMiddleware(LoggingConfig{
		Output: os.Stdout,
		SkipPaths: []string{
			"/health",
			"/health/live",
			"/health/ready",
		},
		TimeFormat: "2006/01/02 - 15:04:05",
	})
}

// StructuredLoggingMiddleware returns a middleware that logs in structured JSON format.
func StructuredLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Get request ID if available
		requestID := ""
		if param.Keys != nil {
			if id, exists := param.Keys[string(RequestIDKey)]; exists {
				if idStr, ok := id.(string); ok {
					requestID = idStr
				}
			}
		}

		// Build structured log record using proper JSON marshaling
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

		// Marshal to JSON to ensure proper escaping
		b, _ := json.Marshal(rec)
		return string(b) + "\n"
	})
}
