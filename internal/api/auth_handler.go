package api

//nolint:gofumpt
import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/services"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRoutes registers authentication routes with the router.
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", authMiddleware.RequireAuth(), h.Logout)
		auth.GET("/me", authMiddleware.RequireAuth(), h.GetProfile)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
	}
}

// Login handles user login requests.
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Authenticate user
	tokenPair, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Set secure HTTP-only cookies
	h.setAuthCookies(c, tokenPair)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt,
		},
	})
}

// Register handles user registration requests.
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Create user
	user, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// RefreshToken handles token refresh requests.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get refresh token from request body or cookie
	var reqData struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&reqData); err != nil {
		// Try to get from cookie
		if cookie, err := c.Cookie("refresh_token"); err == nil {
			reqData.RefreshToken = cookie
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"type":    "VALIDATION_ERROR",
					"code":    "MISSING_REFRESH_TOKEN",
					"message": "Refresh token is required",
				},
			})
			return
		}
	}

	// Generate new tokens
	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), reqData.RefreshToken)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Set secure HTTP-only cookies
	h.setAuthCookies(c, tokenPair)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt,
		},
	})
}

// Logout handles user logout requests.
func (h *AuthHandler) Logout(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	// Logout user (placeholder for token blacklist)
	if err := h.authService.Logout(c.Request.Context(), user.ID); err != nil {
		h.handleError(c, err)
		return
	}

	// Clear cookies
	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully logged out",
	})
}

// GetProfile handles get user profile requests.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHENTICATION_ERROR",
				"code":    "USER_NOT_FOUND",
				"message": "User not found in context",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// ForgotPassword handles password reset request initiation.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Initiate password reset
	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		h.handleError(c, err)
		return
	}

	// Always return success to avoid email enumeration
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset with token.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// Reset password
	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password has been successfully reset",
	})
}

// setAuthCookies sets secure HTTP-only cookies for authentication.
func (h *AuthHandler) setAuthCookies(c *gin.Context, tokenPair *domain.TokenPair) {
	// Access token cookie (shorter expiry)
	c.SetCookie(
		"access_token",
		tokenPair.AccessToken,
		int(time.Until(tokenPair.ExpiresAt).Seconds()),
		"/",
		"",
		true, // Secure
		true, // HttpOnly
	)

	// Refresh token cookie (longer expiry, 7 days)
	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int((7 * 24 * time.Hour).Seconds()),
		"/",
		"",
		true, // Secure
		true, // HttpOnly
	)
}

// clearAuthCookies clears authentication cookies.
func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", true, true)
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
}

// handleError handles domain errors with appropriate HTTP status codes.
func (h *AuthHandler) handleError(c *gin.Context, err error) {
	ErrorResponse(c, err)
}
