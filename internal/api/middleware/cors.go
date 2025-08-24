// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a CORS middleware with configurable options.
//
//nolint:gofumpt
func CORSMiddleware(allowedOrigins []string, allowedMethods []string, allowedHeaders []string, allowCredentials bool) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Set allowed origins
		if len(allowedOrigins) == 0 || contains(allowedOrigins, "*") {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if contains(allowedOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Set allowed methods
		if len(allowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
		} else {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		}

		// Set allowed headers
		if len(allowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		} else {
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID")
		}

		// Set credentials
		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set max age for preflight requests
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}

// DefaultCORSMiddleware returns a CORS middleware with sensible defaults for development.
func DefaultCORSMiddleware() gin.HandlerFunc {
	return CORSMiddleware(
		[]string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000", "http://127.0.0.1:8080"},
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		[]string{"Content-Type", "Authorization", "X-Requested-With", "X-Request-ID"},
		true,
	)
}

// ProductionCORSMiddleware returns a CORS middleware configured for production use.
func ProductionCORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return CORSMiddleware(
		allowedOrigins,
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		[]string{"Content-Type", "Authorization", "X-Request-ID"},
		true,
	)
}

// contains checks if a slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
