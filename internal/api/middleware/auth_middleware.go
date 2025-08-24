package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/services"
)

// UserContextKey is the key used to store user in request context.
const UserContextKey = "user"

// AuthMiddleware provides authentication middleware functionality.
type AuthMiddleware struct {
	authService services.AuthService
}

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(authService services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth middleware that requires valid JWT authentication.
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, err := m.extractUser(c)
		if err != nil {
			m.handleAuthError(c, err)
			return
		}

		// Store user in context for downstream handlers
		c.Set(UserContextKey, user)
		c.Next()
	})
}

// OptionalAuth middleware that extracts user if token is provided but doesn't require it.
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, _ := m.extractUser(c)
		if user != nil {
			c.Set(UserContextKey, user)
		}
		c.Next()
	})
}

// RequireRole middleware that requires a specific role.
func (m *AuthMiddleware) RequireRole(role domain.UserRole) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, err := m.extractUser(c)
		if err != nil {
			m.handleAuthError(c, err)
			return
		}

		if user.Role != role {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"type":    "AUTHORIZATION_ERROR",
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "Insufficient permissions to access this resource",
				},
			})
			c.Abort()
			return
		}

		c.Set(UserContextKey, user)
		c.Next()
	})
}

// RequireAdmin middleware that requires admin role.
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole(domain.AdminRole)
}

// extractUser extracts and validates user from request.
func (m *AuthMiddleware) extractUser(c *gin.Context) (*domain.User, error) {
	// Try to get token from Authorization header
	token := m.extractTokenFromHeader(c)
	if token == "" {
		// Try to get token from cookie
		token = m.extractTokenFromCookie(c)
	}

	if token == "" {
		return nil, domain.NewAuthenticationError("MISSING_TOKEN", "Authentication token required")
	}

	// Validate token and get user
	user, err := m.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// extractTokenFromHeader extracts JWT token from Authorization header.
func (m *AuthMiddleware) extractTokenFromHeader(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for Bearer token format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// extractTokenFromCookie extracts JWT token from cookie.
func (m *AuthMiddleware) extractTokenFromCookie(c *gin.Context) string {
	cookie, err := c.Cookie("access_token")
	if err != nil {
		return ""
	}
	return cookie
}

// handleAuthError handles authentication errors with consistent response format.
func (m *AuthMiddleware) handleAuthError(c *gin.Context, err error) {
	if domainErr, ok := err.(*domain.Error); ok {
		statusCode := http.StatusUnauthorized
		if domainErr.Type == domain.AuthorizationError {
			statusCode = http.StatusForbidden
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "INVALID_TOKEN",
				"message": "Invalid authentication token",
			},
		})
	}
	c.Abort()
}

// GetUserFromContext extracts the authenticated user from Gin context.
func GetUserFromContext(c *gin.Context) (*domain.User, bool) {
	if user, exists := c.Get(UserContextKey); exists {
		if u, ok := user.(*domain.User); ok {
			return u, true
		}
	}
	return nil, false
}

// GetUserFromRequestContext extracts the authenticated user from request context.
func GetUserFromRequestContext(ctx context.Context) (*domain.User, bool) {
	if user := ctx.Value(UserContextKey); user != nil {
		if u, ok := user.(*domain.User); ok {
			return u, true
		}
	}
	return nil, false
}

// RequireOwnership middleware that requires user to be the owner of a resource.
func (m *AuthMiddleware) RequireOwnership(extractOwnerID func(c *gin.Context) string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, exists := GetUserFromContext(c)
		if !exists {
			m.handleAuthError(c, domain.NewAuthenticationError("USER_NOT_FOUND", "User not found in context"))
			return
		}

		ownerID := extractOwnerID(c)
		if ownerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"type":    "VALIDATION_ERROR",
					"code":    "MISSING_OWNER_ID",
					"message": "Owner ID could not be determined",
				},
			})
			c.Abort()
			return
		}

		// Admin users can access any resource
		if user.IsAdmin() {
			c.Next()
			return
		}

		// Check ownership
		if user.ID != ownerID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"type":    "AUTHORIZATION_ERROR",
					"code":    "NOT_OWNER",
					"message": "You can only access resources you own",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	})
}
