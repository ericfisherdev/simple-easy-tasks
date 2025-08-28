package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
	"simple-easy-tasks/internal/services"
)

// GitHubHandler handles GitHub integration endpoints
type GitHubHandler struct {
	githubOAuthService   *services.GitHubOAuthService
	githubService        *services.GitHubService
	githubWebhookService *services.GitHubWebhookService
}

// NewGitHubHandler creates a new GitHub handler
func NewGitHubHandler(
	githubOAuthService *services.GitHubOAuthService,
	githubService *services.GitHubService,
	githubWebhookService *services.GitHubWebhookService,
) *GitHubHandler {
	return &GitHubHandler{
		githubOAuthService:   githubOAuthService,
		githubService:        githubService,
		githubWebhookService: githubWebhookService,
	}
}

// InitiateGitHubAuth starts the GitHub OAuth flow
func (h *GitHubHandler) InitiateGitHubAuth(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req struct {
		ProjectID *string `json:"project_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	authReq := &services.GitHubAuthRequest{
		UserID:    userID,
		ProjectID: req.ProjectID,
	}

	resp, err := h.githubOAuthService.InitiateAuth(c.Request.Context(), authReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to initiate GitHub authentication",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": resp.AuthURL,
		"state":    resp.State,
	})
}

// HandleGitHubCallback processes GitHub OAuth callback
func (h *GitHubHandler) HandleGitHubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing code or state parameter",
		})
		return
	}

	callbackReq := &services.GitHubCallbackRequest{
		Code:  code,
		State: state,
	}

	resp, err := h.githubOAuthService.HandleCallback(c.Request.Context(), callbackReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "OAuth callback failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
		"emails":       resp.Emails,
		"project_id":   resp.ProjectID,
	})
}

// GetUserRepositories gets GitHub repositories accessible to the user
func (h *GitHubHandler) GetUserRepositories(c *gin.Context) {
	accessToken := c.GetHeader("X-GitHub-Token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "GitHub access token is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "30"))

	if perPage > 100 {
		perPage = 100
	}

	repos, err := h.githubOAuthService.GetUserRepositories(c.Request.Context(), accessToken, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get repositories",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repos,
		"page":         page,
		"per_page":     perPage,
	})
}

// CreateIntegration creates a new GitHub integration
func (h *GitHubHandler) CreateIntegration(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req struct {
		AccessToken string `json:"access_token" binding:"required"`
		ProjectID   string `json:"project_id" binding:"required"`
		RepoOwner   string `json:"repo_owner" binding:"required"`
		RepoName    string `json:"repo_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	integration, err := h.githubService.CreateIntegration(
		c.Request.Context(),
		req.AccessToken,
		req.ProjectID,
		userID,
		req.RepoOwner,
		req.RepoName,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create integration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, integration)
}

// GetIntegrationByProject gets GitHub integration for a project
func (h *GitHubHandler) GetIntegrationByProject(c *gin.Context) {
	projectID := c.Param("projectId")

	integration, err := h.githubService.GetIntegrationByProjectID(c.Request.Context(), projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No GitHub integration found for project",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get integration",
			"details": err.Error(),
		})
		return
	}

	// Don't return the access token
	integration.AccessToken = ""

	c.JSON(http.StatusOK, integration)
}

// SyncIssueToTask synchronizes a GitHub issue with a task
func (h *GitHubHandler) SyncIssueToTask(c *gin.Context) {
	integrationID := c.Param("integrationId")

	var req struct {
		IssueNumber int    `json:"issue_number" binding:"required"`
		TaskID      string `json:"task_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	err := h.githubService.SyncIssueToTask(
		c.Request.Context(),
		integrationID,
		req.IssueNumber,
		req.TaskID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to sync issue to task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Issue synchronized successfully",
	})
}

// CreateIssueFromTask creates a GitHub issue from a task
func (h *GitHubHandler) CreateIssueFromTask(c *gin.Context) {
	integrationID := c.Param("integrationId")
	taskID := c.Param("taskId")

	// Get task details - this would need to be implemented
	task, err := h.getTaskByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}

	issue, err := h.githubService.CreateIssueFromTask(
		c.Request.Context(),
		integrationID,
		taskID,
		task,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create issue from task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"issue":   issue,
		"message": "Issue created successfully",
	})
}

// CreateBranchForTask creates a branch for a task
func (h *GitHubHandler) CreateBranchForTask(c *gin.Context) {
	integrationID := c.Param("integrationId")
	taskID := c.Param("taskId")

	// Get task details
	task, err := h.getTaskByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}

	err = h.githubService.CreateBranchForTask(
		c.Request.Context(),
		integrationID,
		taskID,
		task,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create branch for task",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Branch created successfully",
	})
}

// GetTaskCommits gets commits linked to a task
func (h *GitHubHandler) GetTaskCommits(c *gin.Context) {
	taskID := c.Param("taskId")

	commits, err := h.githubService.GetCommitsByTaskID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get task commits",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"commits": commits,
	})
}

// GetTaskPullRequests gets pull requests linked to a task
func (h *GitHubHandler) GetTaskPullRequests(c *gin.Context) {
	taskID := c.Param("taskId")

	pr, err := h.githubService.GetPRsByTaskID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get task pull requests",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pull_request": pr,
	})
}

// UpdateIntegrationSettings updates GitHub integration settings
func (h *GitHubHandler) UpdateIntegrationSettings(c *gin.Context) {
	integrationID := c.Param("integrationId")

	var req struct {
		Settings domain.GitHubSettings `json:"settings" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	integration, err := h.githubService.GetIntegrationByID(c.Request.Context(), integrationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Integration not found",
		})
		return
	}

	integration.Settings = req.Settings
	integration.UpdatedAt = time.Now()

	err = h.githubService.UpdateIntegration(c.Request.Context(), integration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update integration settings",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Integration settings updated successfully",
	})
}

// DeleteIntegration deletes a GitHub integration
func (h *GitHubHandler) DeleteIntegration(c *gin.Context) {
	integrationID := c.Param("integrationId")

	err := h.githubService.DeleteIntegration(c.Request.Context(), integrationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete integration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Integration deleted successfully",
	})
}

// HandleWebhook processes GitHub webhooks
func (h *GitHubHandler) HandleWebhook(c *gin.Context) {
	// Delegate to webhook service
	h.githubWebhookService.HandleWebhook(c.Writer, c.Request)
}

// Helper methods

func (h *GitHubHandler) getTaskByID(_ context.Context, taskID string) (*domain.Task, error) {
	// This would need to be implemented to get task from task service
	// For now, return a placeholder
	return &domain.Task{
		ID:          taskID,
		Title:       "Sample Task",
		Description: "Sample task description",
		Status:      "todo",
		Priority:    "medium",
	}, nil
}

func getUserIDFromContext(c *gin.Context) string {
	// Extract user ID from JWT token or session
	// This would be implemented based on your auth middleware
	if userID, exists := c.Get("user_id"); exists {
		if str, ok := userID.(string); ok {
			return str
		}
	}
	return ""
}

// RegisterGitHubRoutes registers GitHub integration routes
func RegisterGitHubRoutes(r *gin.RouterGroup, handler *GitHubHandler, authMiddleware gin.HandlerFunc) {
	github := r.Group("/github")

	// OAuth routes
	github.POST("/auth", authMiddleware, handler.InitiateGitHubAuth)
	github.GET("/callback", handler.HandleGitHubCallback)

	// Repository routes
	github.GET("/repositories", authMiddleware, handler.GetUserRepositories)

	// Integration routes
	github.POST("/integrations", authMiddleware, handler.CreateIntegration)
	github.GET("/integrations/project/:projectId", authMiddleware, handler.GetIntegrationByProject)
	github.PUT("/integrations/:integrationId/settings", authMiddleware, handler.UpdateIntegrationSettings)
	github.DELETE("/integrations/:integrationId", authMiddleware, handler.DeleteIntegration)

	// Synchronization routes
	github.POST("/integrations/:integrationId/sync-issue", authMiddleware, handler.SyncIssueToTask)
	github.POST("/integrations/:integrationId/tasks/:taskId/create-issue", authMiddleware, handler.CreateIssueFromTask)
	github.POST("/integrations/:integrationId/tasks/:taskId/create-branch", authMiddleware, handler.CreateBranchForTask)

	// Task linking routes
	github.GET("/tasks/:taskId/commits", authMiddleware, handler.GetTaskCommits)
	github.GET("/tasks/:taskId/pull-requests", authMiddleware, handler.GetTaskPullRequests)

	// Webhook route (no auth required)
	github.POST("/webhook", handler.HandleWebhook)
}
