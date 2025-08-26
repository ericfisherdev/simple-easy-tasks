package services

import (
	"context"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// TaskService defines the interface for task-related business logic.
type TaskService interface {
	// CreateTask creates a new task
	CreateTask(ctx context.Context, req domain.CreateTaskRequest, userID string) (*domain.Task, error)

	// GetTask gets a task by ID
	GetTask(ctx context.Context, taskID string, userID string) (*domain.Task, error)

	// UpdateTask updates a task
	UpdateTask(ctx context.Context, taskID string, req domain.UpdateTaskRequest, userID string) (*domain.Task, error)

	// DeleteTask deletes a task
	DeleteTask(ctx context.Context, taskID string, userID string) error

	// ListProjectTasks lists tasks for a project
	ListProjectTasks(ctx context.Context, projectID string, userID string, offset, limit int) ([]*domain.Task, error)

	// ListUserTasks lists tasks assigned to a user
	ListUserTasks(ctx context.Context, userID string, offset, limit int) ([]*domain.Task, error)

	// AssignTask assigns a task to a user
	AssignTask(ctx context.Context, taskID string, assigneeID string, userID string) (*domain.Task, error)

	// UnassignTask removes assignment from a task
	UnassignTask(ctx context.Context, taskID string, userID string) (*domain.Task, error)

	// UpdateTaskStatus updates a task's status
	UpdateTaskStatus(ctx context.Context, taskID string, status domain.TaskStatus, userID string) (*domain.Task, error)
}

// taskService implements TaskService interface.
type taskService struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	userRepo    repository.UserRepository
}

// NewTaskService creates a new task service.
func NewTaskService(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
) TaskService {
	return &taskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
	}
}

// CreateTask creates a new task.
func (s *taskService) CreateTask(
	ctx context.Context,
	req domain.CreateTaskRequest,
	userID string,
) (*domain.Task, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if project exists and user has access
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	// Check if assignee exists and has access to project
	if req.AssigneeID != "" {
		_, assigneeErr := s.userRepo.GetByID(ctx, req.AssigneeID)
		if assigneeErr != nil {
			return nil, domain.NewNotFoundError("ASSIGNEE_NOT_FOUND", "Assignee not found")
		}

		if !project.HasAccess(req.AssigneeID) {
			return nil, domain.NewValidationError("ASSIGNEE_NO_ACCESS", "Assignee doesn't have access to this project", nil)
		}
	}

	// Create task with proper field mapping
	var assigneePtr *string
	if req.AssigneeID != "" {
		assigneePtr = &req.AssigneeID
	}

	// Get next position for task ordering (simple implementation)
	// In a production system, you might want more sophisticated positioning
	taskCount, err := s.taskRepo.CountByProject(ctx, req.ProjectID)
	if err != nil {
		return nil, domain.NewInternalError("TASK_COUNT_FAILED", "Failed to determine task position", err)
	}
	nextPosition := taskCount + 1

	task := &domain.Task{
		Title:       req.Title,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		ReporterID:  userID,
		AssigneeID:  assigneePtr,
		Status:      domain.StatusTodo,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		Tags:        req.Tags,
		Position:    nextPosition,
		Progress:    0,
		TimeSpent:   0.0,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, domain.NewInternalError("TASK_CREATE_FAILED", "Failed to create task", err)
	}

	return task, nil
}

// GetTask gets a task by ID.
func (s *taskService) GetTask(ctx context.Context, taskID string, userID string) (*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) && project.Settings.IsPrivate {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this task")
	}

	return task, nil
}

// UpdateTask updates a task.
func (s *taskService) UpdateTask(
	ctx context.Context,
	taskID string,
	req domain.UpdateTaskRequest,
	userID string,
) (*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	// Get existing task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to modify this task")
	}

	// Apply updates
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.AssigneeID != nil {
		if err := s.validateAssigneeAccess(ctx, *req.AssigneeID, project); err != nil {
			return nil, err
		}
		task.AssigneeID = req.AssigneeID
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.Tags != nil {
		task.Tags = req.Tags
	}
	// Metadata and LastModifiedBy fields don't exist in Task domain model
	// Remove these assignments

	// Validate updated task
	if err := task.Validate(); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, domain.NewInternalError("TASK_UPDATE_FAILED", "Failed to update task", err)
	}

	return task, nil
}

// DeleteTask deletes a task.
func (s *taskService) DeleteTask(ctx context.Context, taskID string, userID string) error {
	if taskID == "" {
		return domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	// Get existing task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Check if user has access to the project and can delete tasks
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	// Only project owners or task creator can delete tasks
	canDelete := project.IsOwner(userID) || task.ReporterID == userID
	if !canDelete {
		return domain.NewAuthorizationError("ACCESS_DENIED", "You don't have permission to delete this task")
	}

	// Delete from repository
	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return domain.NewInternalError("TASK_DELETE_FAILED", "Failed to delete task", err)
	}

	return nil
}

// ListProjectTasks lists tasks for a project.
func (s *taskService) ListProjectTasks(
	ctx context.Context,
	projectID string,
	userID string,
	offset, limit int,
) ([]*domain.Task, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) && project.Settings.IsPrivate {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tasks, err := s.taskRepo.ListByProject(ctx, projectID, offset, limit)
	if err != nil {
		return nil, domain.NewInternalError("TASK_LIST_FAILED", "Failed to list tasks", err)
	}

	return tasks, nil
}

// ListUserTasks lists tasks assigned to a user.
func (s *taskService) ListUserTasks(ctx context.Context, userID string, offset, limit int) ([]*domain.Task, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tasks, err := s.taskRepo.ListByAssignee(ctx, userID, offset, limit)
	if err != nil {
		return nil, domain.NewInternalError("TASK_LIST_FAILED", "Failed to list tasks", err)
	}

	return tasks, nil
}

// AssignTask assigns a task to a user.
func (s *taskService) AssignTask(
	ctx context.Context,
	taskID string,
	assigneeID string,
	userID string,
) (*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}
	if assigneeID == "" {
		return nil, domain.NewValidationError("INVALID_ASSIGNEE_ID", "Assignee ID cannot be empty", nil)
	}

	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to modify this task")
	}

	// Check if assignee exists and has access to project
	_, err = s.userRepo.GetByID(ctx, assigneeID)
	if err != nil {
		return nil, domain.NewNotFoundError("ASSIGNEE_NOT_FOUND", "Assignee not found")
	}

	if !project.HasAccess(assigneeID) {
		return nil, domain.NewValidationError("ASSIGNEE_NO_ACCESS", "Assignee doesn't have access to this project", nil)
	}

	// Assign task
	task.AssigneeID = &assigneeID

	// Update in repository
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, domain.NewInternalError("TASK_ASSIGN_FAILED", "Failed to assign task", err)
	}

	return task, nil
}

// validateAssigneeAccess validates that an assignee exists and has project access
func (s *taskService) validateAssigneeAccess(ctx context.Context, assigneeID string, project *domain.Project) error {
	// Check if assignee has access to project
	if assigneeID != "" {
		_, err := s.userRepo.GetByID(ctx, assigneeID)
		if err != nil {
			return domain.NewNotFoundError("ASSIGNEE_NOT_FOUND", "Assignee not found")
		}

		if !project.HasAccess(assigneeID) {
			return domain.NewValidationError("ASSIGNEE_NO_ACCESS", "Assignee doesn't have access to this project", nil)
		}
	}
	return nil
}

// validateTaskAccess validates task ID and user access
func (s *taskService) validateTaskAccess(ctx context.Context, taskID string, userID string) (*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to modify this task")
	}

	return task, nil
}

// UnassignTask removes assignment from a task.
func (s *taskService) UnassignTask(ctx context.Context, taskID string, userID string) (*domain.Task, error) {
	task, err := s.validateTaskAccess(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Unassign task
	task.AssigneeID = nil

	// Update in repository
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, domain.NewInternalError("TASK_UNASSIGN_FAILED", "Failed to unassign task", err)
	}

	return task, nil
}

// UpdateTaskStatus updates a task's status.
func (s *taskService) UpdateTaskStatus(
	ctx context.Context,
	taskID string,
	status domain.TaskStatus,
	userID string,
) (*domain.Task, error) {
	task, err := s.validateTaskAccess(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Update status
	task.Status = status

	// Update in repository
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, domain.NewInternalError("TASK_STATUS_UPDATE_FAILED", "Failed to update task status", err)
	}

	return task, nil
}
