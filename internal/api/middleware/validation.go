// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"

	"simple-easy-tasks/internal/validation"

	"github.com/gin-gonic/gin"
)

// ValidationMiddleware returns a middleware that validates request bodies.
func ValidationMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()
	})
}

// ValidateJSON validates JSON request body against a struct.
func ValidateJSON[T any](c *gin.Context, target *T) bool {
	// First, try to bind JSON
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_JSON",
				"message": "Invalid JSON format or structure",
				"details": err.Error(),
			},
		})
		return false
	}

	// Then validate using our custom validator
	if err := validation.ValidateStruct(*target); err != nil {
		statusCode := http.StatusBadRequest
		// err is already a *domain.Error from ValidateStruct
		domainErr := err

		c.JSON(statusCode, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    domainErr.Type,
				"code":    domainErr.Code,
				"message": domainErr.Message,
				"details": domainErr.Details,
			},
		})
		return false
	}

	return true
}

// ValidateQuery validates query parameters against a struct.
func ValidateQuery[T any](c *gin.Context, target *T) bool {
	// First, try to bind query parameters
	if err := c.ShouldBindQuery(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_QUERY_PARAMS",
				"message": "Invalid query parameters",
				"details": err.Error(),
			},
		})
		return false
	}

	// Then validate using our custom validator
	if domainErr := validation.ValidateStruct(*target); domainErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    domainErr.Type,
				"code":    domainErr.Code,
				"message": domainErr.Message,
				"details": domainErr.Details,
			},
		})
		return false
	}

	return true
}

// ValidateURI validates URI parameters against a struct.
func ValidateURI[T any](c *gin.Context, target *T) bool {
	// First, try to bind URI parameters
	if err := c.ShouldBindUri(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_URI_PARAMS",
				"message": "Invalid URI parameters",
				"details": err.Error(),
			},
		})
		return false
	}

	// Then validate using our custom validator
	if domainErr := validation.ValidateStruct(*target); domainErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    domainErr.Type,
				"code":    domainErr.Code,
				"message": domainErr.Message,
				"details": domainErr.Details,
			},
		})
		return false
	}

	return true
}
