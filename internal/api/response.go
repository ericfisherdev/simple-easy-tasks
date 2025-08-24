// Package api provides shared utilities for API handlers.
package api

//nolint:gofumpt
import (
	"net/http"

	"simple-easy-tasks/internal/domain"

	"github.com/gin-gonic/gin"
)

// ErrorResponse handles domain errors consistently across all handlers.
func ErrorResponse(c *gin.Context, err error) {
	if domainErr, ok := err.(*domain.Error); ok {
		statusCode := http.StatusInternalServerError

		switch domainErr.Type {
		case domain.ValidationError:
			statusCode = http.StatusBadRequest
		case domain.NotFoundError:
			statusCode = http.StatusNotFound
		case domain.ConflictError:
			statusCode = http.StatusConflict
		case domain.AuthenticationError:
			statusCode = http.StatusUnauthorized
		case domain.AuthorizationError:
			statusCode = http.StatusForbidden
		case domain.InternalError:
			statusCode = http.StatusInternalServerError
		case domain.ExternalServiceError:
			statusCode = http.StatusBadGateway
		}

		c.JSON(statusCode, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    domainErr.Type,
				"code":    domainErr.Code,
				"message": domainErr.Message,
				"details": domainErr.Details,
			},
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "INTERNAL_ERROR",
				"code":    "UNKNOWN_ERROR",
				"message": "An unexpected error occurred",
			},
		})
	}
}

// SuccessResponse returns a standardized success response.
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// CreatedResponse returns a standardized created response.
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}
