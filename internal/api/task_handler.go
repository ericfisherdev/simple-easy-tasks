package api

import (
	"net/http"
	"strconv"
	"time"

	"simple-easy-tasks/internal/api/middleware"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
	"simple-easy-tasks/internal/services"

	"github.com/gin-gonic/gin"
)

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	taskService services.TaskService
	taskRepo    repository.TaskRepository
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(
	taskService services.TaskService,
	taskRepo repository.TaskRepository,
) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		taskRepo:    taskRepo,
	}
}

// RegisterRoutes registers task routes with the router.
func (h *TaskHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	projects := router.Group("/projects")
	projects.Use(authMiddleware.RequireAuth())
	{
		// Nested tasks routes under projects
		tasks := projects.Group("/:projectId/tasks")
		{
			// Core CRUD operations
			tasks.GET("", h.ListTasks)
			tasks.POST("", h.CreateTask)
			tasks.GET("/:id", h.GetTask)
			tasks.PUT("/:id", h.UpdateTask)
			tasks.DELETE("/:id", h.DeleteTask)

			// Task movement and status operations
			tasks.POST("/:id/move", h.MoveTask)
			tasks.PUT("/:id/status", h.UpdateTaskStatus)
			tasks.PUT("/:id/position", h.UpdateTaskPosition)

			// Task assignment operations
			tasks.POST("/:id/assign", h.AssignTask)
			tasks.DELETE("/:id/assign", h.UnassignTask)

			// Advanced task operations
			tasks.POST("/:id/duplicate", h.DuplicateTask)
			tasks.GET("/:id/history", h.GetTaskHistory)
			tasks.POST("/:id/time-log", h.LogTimeSpent)

			// Task relationship operations
			tasks.POST("/:id/subtasks", h.CreateSubtask)
			tasks.GET("/:id/subtasks", h.ListSubtasks)
			tasks.POST("/:id/dependencies", h.AddDependency)
			tasks.DELETE("/:id/dependencies/:depId", h.RemoveDependency)
		}
	}
}

// ListTasks handles GET /api/projects/:projectId/tasks requests.
func (h *TaskHandler) ListTasks(c *gin.Context) {
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

	projectID := c.Param("projectId")
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

	// Parse query parameters for filtering
	filters := h.parseTaskFilters(c)

	// Use the advanced filtering service method
	tasks, err := h.taskService.GetProjectTasksFiltered(c.Request.Context(), projectID, filters, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get total count for pagination (if not using skipTotal)
	totalTasks := len(tasks)
	if filters.Limit > 0 {
		// For paginated requests, we might want exact count
		totalCount, err := h.taskRepo.CountByProject(c.Request.Context(), projectID)
		if err == nil {
			totalTasks = totalCount
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks": tasks,
			"meta": gin.H{
				"total":  totalTasks,
				"limit":  filters.Limit,
				"offset": filters.Offset,
				"count":  len(tasks),
			},
		},
	})
}

// CreateTask handles POST /api/projects/:projectId/tasks requests.
func (h *TaskHandler) CreateTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
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

	var req domain.CreateTaskRequest
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

	// Ensure project ID matches the URL parameter
	req.ProjectID = projectID

	// Create task
	task, err := h.taskService.CreateTask(c.Request.Context(), req, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
	})
}

// GetTask handles GET /api/projects/:projectId/tasks/:id requests.
func (h *TaskHandler) GetTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
	})
}

// UpdateTask handles PUT /api/projects/:projectId/tasks/:id requests.
func (h *TaskHandler) UpdateTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req domain.UpdateTaskRequest
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

	// Update task
	task, err := h.taskService.UpdateTask(c.Request.Context(), taskID, req, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
	})
}

// DeleteTask handles DELETE /api/projects/:projectId/tasks/:id requests.
func (h *TaskHandler) DeleteTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	// Verify task belongs to project before deletion
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	err = h.taskService.DeleteTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Task deleted successfully",
	})
}

// MoveTask handles POST /api/projects/:projectId/tasks/:id/move requests.
func (h *TaskHandler) MoveTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req struct {
		NewStatus   domain.TaskStatus `json:"new_status" binding:"required"`
		NewPosition int               `json:"new_position" binding:"min=0"`
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

	// Create move request
	moveReq := services.MoveTaskRequest{
		TaskID:      taskID,
		ProjectID:   projectID,
		NewStatus:   req.NewStatus,
		NewPosition: req.NewPosition,
	}

	err := h.taskService.MoveTask(c.Request.Context(), moveReq, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get updated task to return
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
		"message": "Task moved successfully",
	})
}

// UpdateTaskStatus handles PUT /api/projects/:projectId/tasks/:id/status requests.
func (h *TaskHandler) UpdateTaskStatus(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req struct {
		Status domain.TaskStatus `json:"status" binding:"required"`
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

	// Validate status
	if !req.Status.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "INVALID_STATUS",
				"message": "Invalid task status",
			},
		})
		return
	}

	task, err := h.taskService.UpdateTaskStatus(c.Request.Context(), taskID, req.Status, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
		"message": "Task status updated successfully",
	})
}

// UpdateTaskPosition handles PUT /api/projects/:projectId/tasks/:id/position requests.
func (h *TaskHandler) UpdateTaskPosition(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req struct {
		Position int `json:"position" binding:"min=0"`
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

	// Get current task to maintain status during position update
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	// Use move task service to update position while keeping same status
	moveReq := services.MoveTaskRequest{
		TaskID:      taskID,
		ProjectID:   projectID,
		NewStatus:   task.Status,
		NewPosition: req.Position,
	}

	err = h.taskService.MoveTask(c.Request.Context(), moveReq, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get updated task to return
	updatedTask, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": updatedTask,
		},
		"message": "Task position updated successfully",
	})
}

// AssignTask handles POST /api/projects/:projectId/tasks/:id/assign requests.
func (h *TaskHandler) AssignTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req struct {
		AssigneeID string `json:"assignee_id" binding:"required"`
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

	task, err := h.taskService.AssignTask(c.Request.Context(), taskID, req.AssigneeID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
		"message": "Task assigned successfully",
	})
}

// UnassignTask handles DELETE /api/projects/:projectId/tasks/:id/assign requests.
func (h *TaskHandler) UnassignTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	task, err := h.taskService.UnassignTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
		"message": "Task unassigned successfully",
	})
}

// DuplicateTask handles POST /api/projects/:projectId/tasks/:id/duplicate requests.
func (h *TaskHandler) DuplicateTask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var options services.DuplicationOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		// Use default options if JSON is invalid or not provided
		options = services.DuplicationOptions{
			IncludeSubtasks:    false,
			IncludeComments:    false,
			IncludeAttachments: false,
			ResetProgress:      true,
			ResetTimeSpent:     true,
		}
	}

	task, err := h.taskService.DuplicateTask(c.Request.Context(), taskID, options, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"task": task,
		},
		"message": "Task duplicated successfully",
	})
}

// GetTaskHistory handles GET /api/projects/:projectId/tasks/:id/history requests.
func (h *TaskHandler) GetTaskHistory(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	// Verify user has access to the task
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	// Parse query parameters for history filtering
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Create history filter
	_ = domain.TaskHistoryFilter{
		TaskID: &taskID,
		Limit:  limit,
		Offset: offset,
	}

	// For now, we'll return a placeholder response
	// In a real implementation, you'd query from a task history repository
	history := []domain.TaskHistoryEntry{
		{
			ID:        "history-1",
			TaskID:    taskID,
			UserID:    task.ReporterID,
			Action:    domain.ActionCreated,
			CreatedAt: task.CreatedAt,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"history": history,
			"meta": gin.H{
				"total":  len(history),
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// LogTimeSpent handles POST /api/projects/:projectId/tasks/:id/time-log requests.
func (h *TaskHandler) LogTimeSpent(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req struct {
		Hours       float64 `json:"hours" binding:"required,min=0"`
		Description string  `json:"description,omitempty"`
		LoggedAt    *time.Time `json:"logged_at,omitempty"`
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

	// Verify user has access to the task
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify task belongs to the specified project
	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	// Add time spent to task
	err = task.AddTimeSpent(req.Hours)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Update the task in the repository
	err = h.taskRepo.Update(c.Request.Context(), task)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Create time log entry for history (placeholder implementation)
	timeLog := map[string]interface{}{
		"id":          "log-" + taskID + "-" + strconv.FormatInt(time.Now().Unix(), 10),
		"task_id":     taskID,
		"user_id":     user.ID,
		"hours":       req.Hours,
		"description": req.Description,
		"logged_at":   req.LoggedAt,
		"created_at":  time.Now(),
	}

	if req.LoggedAt == nil {
		now := time.Now()
		timeLog["logged_at"] = now
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"time_log": timeLog,
			"task":     task,
		},
		"message": "Time logged successfully",
	})
}

// parseTaskFilters parses query parameters into TaskFilters struct.
func (h *TaskHandler) parseTaskFilters(c *gin.Context) repository.TaskFilters {
	filters := repository.TaskFilters{}

	// Parse limit and offset
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filters.Limit = limit
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 20 // default
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Parse status filter
	if statusStr := c.Query("status"); statusStr != "" {
		statuses := []domain.TaskStatus{domain.TaskStatus(statusStr)}
		filters.Status = statuses
	}

	// Parse priority filter
	if priorityStr := c.Query("priority"); priorityStr != "" {
		priorities := []domain.TaskPriority{domain.TaskPriority(priorityStr)}
		filters.Priority = priorities
	}

	// Parse assignee filter
	if assigneeStr := c.Query("assignee"); assigneeStr != "" {
		filters.AssigneeID = &assigneeStr
	}

	// Parse reporter filter
	if reporterStr := c.Query("reporter"); reporterStr != "" {
		filters.ReporterID = &reporterStr
	}

	// Parse search filter
	if searchStr := c.Query("search"); searchStr != "" {
		filters.Search = searchStr
	}

	// Parse archived filter
	if archivedStr := c.Query("archived"); archivedStr != "" {
		if archivedStr == "true" {
			archived := true
			filters.Archived = &archived
		} else if archivedStr == "false" {
			archived := false
			filters.Archived = &archived
		}
	}

	// Parse due date filters
	if dueBeforeStr := c.Query("due_before"); dueBeforeStr != "" {
		if dueBefore, err := time.Parse(time.RFC3339, dueBeforeStr); err == nil {
			filters.DueBefore = &dueBefore
		}
	}

	if dueAfterStr := c.Query("due_after"); dueAfterStr != "" {
		if dueAfter, err := time.Parse(time.RFC3339, dueAfterStr); err == nil {
			filters.DueAfter = &dueAfter
		}
	}

	// Parse sort parameters
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters.SortBy = sortBy
	} else {
		filters.SortBy = repository.SortByUpdated // default
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		filters.SortOrder = sortOrder
	} else {
		filters.SortOrder = repository.SortOrderDesc // default
	}

	return filters
}

// CreateSubtask handles POST /api/projects/:projectId/tasks/:id/subtasks requests.
func (h *TaskHandler) CreateSubtask(c *gin.Context) {
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

	projectID := c.Param("projectId")
	parentTaskID := c.Param("id")

	if projectID == "" || parentTaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and parent task ID are required",
			},
		})
		return
	}

	var req domain.CreateTaskRequest
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

	// Create subtask
	subtask, err := h.taskService.CreateSubtask(c.Request.Context(), parentTaskID, req, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Verify subtask belongs to the specified project
	if subtask.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "PARENT_TASK_NOT_FOUND",
				"message": "Parent task not found in specified project",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"task": subtask,
		},
		"message": "Subtask created successfully",
	})
}

// ListSubtasks handles GET /api/projects/:projectId/tasks/:id/subtasks requests.
func (h *TaskHandler) ListSubtasks(c *gin.Context) {
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

	projectID := c.Param("projectId")
	parentTaskID := c.Param("id")

	if projectID == "" || parentTaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and parent task ID are required",
			},
		})
		return
	}

	// Verify parent task exists and belongs to project
	parentTask, err := h.taskService.GetTask(c.Request.Context(), parentTaskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if parentTask.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "PARENT_TASK_NOT_FOUND",
				"message": "Parent task not found in specified project",
			},
		})
		return
	}

	// Get subtasks
	subtasks, err := h.taskService.GetSubtasks(c.Request.Context(), parentTaskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subtasks": subtasks,
			"meta": gin.H{
				"count":        len(subtasks),
				"parent_task":  parentTask.ID,
				"parent_title": parentTask.Title,
			},
		},
	})
}

// AddDependency handles POST /api/projects/:projectId/tasks/:id/dependencies requests.
func (h *TaskHandler) AddDependency(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")

	if projectID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID and task ID are required",
			},
		})
		return
	}

	var req struct {
		DependencyID string `json:"dependency_id" binding:"required"`
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

	// Verify task belongs to project
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	// Add dependency
	err = h.taskService.AddDependency(c.Request.Context(), taskID, req.DependencyID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"task_id":       taskID,
			"dependency_id": req.DependencyID,
		},
		"message": "Dependency added successfully",
	})
}

// RemoveDependency handles DELETE /api/projects/:projectId/tasks/:id/dependencies/:depId requests.
func (h *TaskHandler) RemoveDependency(c *gin.Context) {
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

	projectID := c.Param("projectId")
	taskID := c.Param("id")
	dependencyID := c.Param("depId")

	if projectID == "" || taskID == "" || dependencyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "VALIDATION_ERROR",
				"code":    "MISSING_PARAMETERS",
				"message": "Project ID, task ID, and dependency ID are required",
			},
		})
		return
	}

	// Verify task belongs to project
	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if task.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"type":    "NOT_FOUND_ERROR",
				"code":    "TASK_NOT_FOUND",
				"message": "Task not found in specified project",
			},
		})
		return
	}

	// Remove dependency
	err = h.taskService.RemoveDependency(c.Request.Context(), taskID, dependencyID, user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task_id":       taskID,
			"dependency_id": dependencyID,
		},
		"message": "Dependency removed successfully",
	})
}

// handleError handles domain errors with appropriate HTTP status codes.
func (h *TaskHandler) handleError(c *gin.Context, err error) {
	ErrorResponse(c, err)
}