// Package middleware provides HTTP middleware functions.
package middleware

import (
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

// LoggingConfig holds configuration for the logging middleware.
//
//nolint:govet // fieldalignment: micro-optimization not critical for this config struct
type LoggingConfig struct {
	// SkipPaths is a slice of paths that should not be logged.
	SkipPaths []string
	// TimeFormat specifies the time format for logging.
	TimeFormat string
	// Output specifies the output destination. If nil, os.Stdout is used.
	Output io.Writer
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

		// JSON structured log
		return fmt.Sprintf(
			`{"timestamp":"%s","status":%d,"latency":"%s","client_ip":"%s",`+
				`"method":"%s","path":"%s","request_id":"%s","error":"%s"}`+"\n",
			param.TimeStamp.Format("2006-01-02T15:04:05Z07:00"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			requestID,
			param.ErrorMessage,
		)
	})
}
