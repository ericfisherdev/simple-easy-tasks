package api

//nolint:gofumpt
import (
	"net/http"
	"strconv"

	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
	"simple-easy-tasks/internal/services"

	"github.com/gin-gonic/gin"
)

// ProjectHandler handles project-related HTTP requests.
type ProjectHandler struct {
	projectService services.ProjectService
	projectRepo    repository.ProjectRepository
}

// NewProjectHandler creates a new project handler.
func NewProjectHandler(
	projectService services.ProjectService,
	projectRepo repository.ProjectRepository,
) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		projectRepo:    projectRepo,
	}
}

// RegisterRoutes registers project routes with the router.
func (h *ProjectHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	projects := router.Group("/projects")
	projects.Use(authMiddleware.RequireAuth())
	{
		projects.GET("", h.ListProjects)
		projects.POST("", h.CreateProject)
		projects.GET("/:id", h.GetProject)
		projects.PUT("/:id", authMiddleware.RequireOwnership(h.extractProjectOwnerID), h.UpdateProject)
		projects.DELETE("/:id", authMiddleware.RequireOwnership(h.extractProjectOwnerID), h.DeleteProject)
		projects.POST("/:id/members", authMiddleware.RequireOwnership(h.extractProjectOwnerID), h.AddMember)
		projects.DELETE("/:id/members/:memberID", authMiddleware.RequireOwnership(h.extractProjectOwnerID), h.RemoveMember)
	}
}

// ListProjects handles GET /api/projects requests.
func (h *ProjectHandler) ListProjects(c *gin.Context) {
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

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	status := c.Query("status")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get user's projects (owner + member)
	projects, err := h.projectService.ListUserProjects(c.Request.Context(), user.ID, offset, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Filter by status if provided
	if status != "" {
		filteredProjects := make([]*domain.Project, 0)
		for _, project := range projects {
			if string(project.Status) == status {
				filteredProjects = append(filteredProjects, project)
			}
		}
		projects = filteredProjects
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"projects": projects,
			"meta": gin.H{
				"total":  len(projects),
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// CreateProject handles POST /api/projects requests.
func (h *ProjectHandler) CreateProject(c *gin.Context) {
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

	var req domain.CreateProjectRequest
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

	// Create project
	project := &domain.Project{
		Title:       req.Title,
		Description: req.Description,
		Slug:        req.Slug,
		OwnerID:     user.ID,
		Color:       req.Color,
		Icon:        req.Icon,
		Status:      domain.ActiveProject,
		Settings: func() domain.ProjectSettings {
			if req.Settings != nil {
				return *req.Settings
			}
			return domain.ProjectSettings{
				CustomFields:   make(map[string]string),
				Notifications:  make(map[string]bool),
				IsPrivate:      false,
				AllowGuestView: true,
				EnableComments: true,
			}
		}(),
		MemberIDs: []string{},
	}

	err := h.projectRepo.Create(c.Request.Context(), project)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"project": project,
		},
	})
}

// GetProject handles GET /api/projects/:id requests.
func (h *ProjectHandler) GetProject(c *gin.Context) {
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

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PROJECT_ID",
				"message": "Project ID is required",
			},
		})
		return
	}

	project, err := h.projectRepo.GetByID(c.Request.Context(), projectID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access permissions
	if !project.HasAccess(user.ID) && !user.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "AUTHORIZATION_ERROR",
				"code":    "ACCESS_DENIED",
				"message": "You do not have access to this project",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"project": project,
		},
	})
}

// UpdateProject handles PUT /api/projects/:id requests.
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PROJECT_ID",
				"message": "Project ID is required",
			},
		})
		return
	}

	var req domain.UpdateProjectRequest
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

	// Get existing project
	project, err := h.projectRepo.GetByID(c.Request.Context(), projectID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Update fields if provided
	if req.Title != nil {
		project.Title = *req.Title
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Color != nil {
		project.Color = *req.Color
	}
	if req.Icon != nil {
		project.Icon = *req.Icon
	}
	if req.Settings != nil {
		project.Settings = *req.Settings
	}
	if req.Status != nil {
		project.Status = *req.Status
	}

	// Update project
	err = h.projectRepo.Update(c.Request.Context(), project)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"project": project,
		},
	})
}

// DeleteProject handles DELETE /api/projects/:id requests.
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PROJECT_ID",
				"message": "Project ID is required",
			},
		})
		return
	}

	err := h.projectRepo.Delete(c.Request.Context(), projectID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// AddMember handles POST /api/projects/:id/members requests.
func (h *ProjectHandler) AddMember(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PROJECT_ID",
				"message": "Project ID is required",
			},
		})
		return
	}

	var req struct {
		UserID string `json:"user_id" binding:"required"`
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

	// Get requester ID from context
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

	// Add member using service (includes validation)
	err := h.projectService.AddMember(c.Request.Context(), projectID, req.UserID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get updated project to return
	project, err := h.projectService.GetProject(c.Request.Context(), projectID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"project": project,
		},
	})
}

// RemoveMember handles DELETE /api/projects/:id/members/:memberID requests.
func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	projectID := c.Param("id")
	memberID := c.Param("memberID")

	if projectID == "" || memberID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and member ID are required",
			},
		})
		return
	}

	// Get project
	project, err := h.projectRepo.GetByID(c.Request.Context(), projectID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Remove member
	project.RemoveMember(memberID)

	// Update project
	err = h.projectRepo.Update(c.Request.Context(), project)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"project": project,
		},
	})
}

// extractProjectOwnerID extracts the project owner ID for ownership middleware.
func (h *ProjectHandler) extractProjectOwnerID(c *gin.Context) string {
	projectID := c.Param("id")
	if projectID == "" {
		return ""
	}

	project, err := h.projectRepo.GetByID(c.Request.Context(), projectID)
	if err != nil {
		return ""
	}

	return project.OwnerID
}

// handleError handles domain errors with appropriate HTTP status codes.
func (h *ProjectHandler) handleError(c *gin.Context, err error) {
	ErrorResponse(c, err)
}
