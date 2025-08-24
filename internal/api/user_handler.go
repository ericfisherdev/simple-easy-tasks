package api

import (
	"net/http"

	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user profile-related HTTP requests.
type UserHandler struct {
	userRepo repository.UserRepository
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// RegisterRoutes registers user profile routes with the router.
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	users := router.Group("/users")
	users.Use(authMiddleware.RequireAuth())
	{
		users.GET("/profile", h.GetProfile)
		users.PUT("/profile", h.UpdateProfile)
		users.POST("/avatar", h.UpdateAvatar)
		users.DELETE("/avatar", h.RemoveAvatar)
		users.PUT("/preferences", h.UpdatePreferences)

		// Admin-only endpoints
		adminUsers := users.Group("")
		adminUsers.Use(authMiddleware.RequireAdmin())
		{
			adminUsers.GET("", h.ListUsers)
			adminUsers.GET("/:id", h.GetUserByID)
			adminUsers.PUT("/:id/role", h.UpdateUserRole)
			adminUsers.DELETE("/:id", h.DeleteUser)
		}
	}
}

// GetProfile handles GET /api/users/profile requests.
func (h *UserHandler) GetProfile(c *gin.Context) {
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

// UpdateProfile handles PUT /api/users/profile requests.
func (h *UserHandler) UpdateProfile(c *gin.Context) {
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

	var req domain.UpdateUserRequest
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

	// Update fields if provided
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}
	if req.Preferences != nil {
		user.Preferences = *req.Preferences
	}

	// Update user
	err := h.userRepo.Update(c.Request.Context(), user)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// UpdateAvatar handles POST /api/users/avatar requests.
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
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

	var req struct {
		Avatar string `json:"avatar" binding:"required,url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_REQUEST",
				"message": "Invalid request format. Avatar must be a valid URL.",
				"details": err.Error(),
			},
		})
		return
	}

	// Update avatar
	user.Avatar = req.Avatar
	err := h.userRepo.Update(c.Request.Context(), user)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// RemoveAvatar handles DELETE /api/users/avatar requests.
func (h *UserHandler) RemoveAvatar(c *gin.Context) {
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

	// Remove avatar
	user.Avatar = ""
	err := h.userRepo.Update(c.Request.Context(), user)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// UpdatePreferences handles PUT /api/users/preferences requests.
func (h *UserHandler) UpdatePreferences(c *gin.Context) {
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

	var req domain.UserPreferences
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

	// Update preferences
	user.Preferences = req
	err := h.userRepo.Update(c.Request.Context(), user)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// ListUsers handles GET /api/users requests (admin only).
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse query parameters
	search := c.Query("search")

	limit := 50
	offset := 0

	// This is a placeholder implementation - the actual repository interface
	// would need to be extended to support listing users with pagination
	if search != "" {
		// Search by email
		user, err := h.userRepo.GetByEmail(c.Request.Context(), search)
		if err != nil {
			// If search fails, return empty list instead of error
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"users": []*domain.User{},
					"meta": gin.H{
						"total":  0,
						"limit":  limit,
						"offset": offset,
					},
				},
			})
			return
		}

		userList := []*domain.User{user}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"users": userList,
				"meta": gin.H{
					"total":  len(userList),
					"limit":  limit,
					"offset": offset,
				},
			},
		})
		return
	}

	// For now, without search, return empty list
	// In a real implementation, we'd call h.userRepo.List(ctx, offset, limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users": []*domain.User{},
			"meta": gin.H{
				"total":  0,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// GetUserByID handles GET /api/users/:id requests (admin only).
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_USER_ID",
				"message": "User ID is required",
			},
		})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// UpdateUserRole handles PUT /api/users/:id/role requests (admin only).
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_USER_ID",
				"message": "User ID is required",
			},
		})
		return
	}

	var req struct {
		Role domain.UserRole `json:"role" binding:"required"`
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

	// Validate role
	if req.Role != domain.AdminRole && req.Role != domain.RegularUserRole {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_ROLE",
				"message": "Role must be 'admin' or 'user'",
			},
		})
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Update role
	user.Role = req.Role
	err = h.userRepo.Update(c.Request.Context(), user)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
	})
}

// DeleteUser handles DELETE /api/users/:id requests (admin only).
func (h *UserHandler) DeleteUser(c *gin.Context) {
	currentUser, exists := middleware.GetUserFromContext(c)
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

	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_USER_ID",
				"message": "User ID is required",
			},
		})
		return
	}

	// Prevent self-deletion
	if userID == currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHORIZATION_ERROR",
				"code":    "CANNOT_DELETE_SELF",
				"message": "You cannot delete your own account",
			},
		})
		return
	}

	err := h.userRepo.Delete(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// handleError handles domain errors with appropriate HTTP status codes.
func (h *UserHandler) handleError(c *gin.Context, err error) {
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
