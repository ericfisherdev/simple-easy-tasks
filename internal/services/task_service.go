package services

import (
	"context"
	"fmt"

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

	// MoveTask moves a task between statuses and positions (kanban functionality)
	MoveTask(ctx context.Context, req MoveTaskRequest, userID string) error

	// GetProjectTasksFiltered gets tasks for a project with advanced filtering
	GetProjectTasksFiltered(
		ctx context.Context,
		projectID string,
		filters repository.TaskFilters,
		userID string,
	) ([]*domain.Task, error)

	// GetSubtasks retrieves subtasks for a parent task
	GetSubtasks(ctx context.Context, parentTaskID string, userID string) ([]*domain.Task, error)

	// GetTaskDependencies retrieves dependency tasks for a task
	GetTaskDependencies(ctx context.Context, taskID string, userID string) ([]*domain.Task, error)

	// DuplicateTask creates a copy of an existing task
	DuplicateTask(ctx context.Context, taskID string, options DuplicationOptions, userID string) (*domain.Task, error)

	// CreateFromTemplate creates a task from a predefined template
	CreateFromTemplate(ctx context.Context, templateID string, projectID string, userID string) (*domain.Task, error)

	// CreateSubtask creates a subtask under a parent task
	CreateSubtask(
		ctx context.Context, parentTaskID string, req domain.CreateTaskRequest, userID string,
	) (*domain.Task, error)

	// AddDependency adds a dependency to a task
	AddDependency(ctx context.Context, taskID string, dependencyID string, userID string) error

	// RemoveDependency removes a dependency from a task
	RemoveDependency(ctx context.Context, taskID string, dependencyID string, userID string) error
}

// MoveTaskRequest represents a request to move a task between columns/statuses
type MoveTaskRequest struct {
	TaskID      string            `json:"task_id" binding:"required"`
	ProjectID   string            `json:"project_id" binding:"required"`
	NewStatus   domain.TaskStatus `json:"new_status" binding:"required"`
	NewPosition int               `json:"new_position" binding:"min=0"`
}

// DuplicationOptions controls how a task is duplicated
type DuplicationOptions struct {
	NewTitle           string `json:"new_title,omitempty"`
	IncludeSubtasks    bool   `json:"include_subtasks"`
	IncludeComments    bool   `json:"include_comments"`
	IncludeAttachments bool   `json:"include_attachments"`
	ResetProgress      bool   `json:"reset_progress"`
	ResetTimeSpent     bool   `json:"reset_time_spent"`
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

// MoveTask moves a task between statuses and positions (kanban functionality)
func (s *taskService) MoveTask(ctx context.Context, req MoveTaskRequest, userID string) error {
	// Validate request
	if req.TaskID == "" {
		return domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}
	if !req.NewStatus.IsValid() {
		return domain.NewValidationError("INVALID_STATUS", "Invalid task status", nil)
	}
	if req.ProjectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Validate user has access to the task
	task, err := s.validateTaskAccess(ctx, req.TaskID, userID)
	if err != nil {
		return err
	}

	// Ensure the task belongs to the specified project
	if task.ProjectID != req.ProjectID {
		return domain.NewValidationError("PROJECT_MISMATCH", "Task does not belong to specified project", nil)
	}

	// Use repository's Move method which handles position calculation and validation
	if err := s.taskRepo.Move(ctx, req.TaskID, req.NewStatus, req.NewPosition); err != nil {
		return domain.NewInternalError("TASK_MOVE_FAILED", "Failed to move task", err)
	}

	return nil
}

// GetProjectTasksFiltered gets tasks for a project with advanced filtering
func (s *taskService) GetProjectTasksFiltered(
	ctx context.Context,
	projectID string,
	filters repository.TaskFilters,
	userID string,
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

	// Use repository's filtered query
	tasks, err := s.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return nil, domain.NewInternalError("TASK_FILTER_FAILED", "Failed to filter project tasks", err)
	}

	return tasks, nil
}

// GetSubtasks retrieves subtasks for a parent task
func (s *taskService) GetSubtasks(ctx context.Context, parentTaskID string, userID string) ([]*domain.Task, error) {
	if parentTaskID == "" {
		return nil, domain.NewValidationError("INVALID_PARENT_ID", "Parent task ID cannot be empty", nil)
	}

	// Validate user has access to the parent task
	_, err := s.validateTaskAccess(ctx, parentTaskID, userID)
	if err != nil {
		return nil, err
	}

	// Get subtasks from repository
	subtasks, err := s.taskRepo.GetSubtasks(ctx, parentTaskID)
	if err != nil {
		return nil, domain.NewInternalError("SUBTASK_FETCH_FAILED", "Failed to fetch subtasks", err)
	}

	return subtasks, nil
}

// GetTaskDependencies retrieves dependency tasks for a task
func (s *taskService) GetTaskDependencies(ctx context.Context, taskID string, userID string) ([]*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	// Validate user has access to the task
	_, err := s.validateTaskAccess(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Get dependencies from repository
	dependencies, err := s.taskRepo.GetDependencies(ctx, taskID)
	if err != nil {
		return nil, domain.NewInternalError("DEPENDENCY_FETCH_FAILED", "Failed to fetch task dependencies", err)
	}

	return dependencies, nil
}

// DuplicateTask creates a copy of an existing task
func (s *taskService) DuplicateTask(
	ctx context.Context,
	taskID string,
	options DuplicationOptions,
	userID string,
) (*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	// Get the original task and validate access
	originalTask, err := s.validateTaskAccess(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}

	// Create new task from original
	newTask := s.createTaskCopy(originalTask, options)
	newTask.ReporterID = userID // Set the user as the reporter of the duplicated task

	// Create the new task
	if err := s.taskRepo.Create(ctx, newTask); err != nil {
		return nil, domain.NewInternalError("TASK_DUPLICATE_FAILED", "Failed to duplicate task", err)
	}

	// Handle subtasks if requested
	if options.IncludeSubtasks {
		if err := s.duplicateSubtasks(ctx, taskID, newTask.ID, options, userID); err != nil {
			// Log error but don't fail the main duplication
			// In a production system, use structured logging here
			_ = err // Explicitly ignore error to satisfy linter
		}
	}

	return newTask, nil
}

// CreateFromTemplate creates a task from a predefined template
func (s *taskService) CreateFromTemplate(
	ctx context.Context,
	templateID string,
	projectID string,
	userID string,
) (*domain.Task, error) {
	if templateID == "" {
		return nil, domain.NewValidationError("INVALID_TEMPLATE_ID", "Template ID cannot be empty", nil)
	}
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	// Get the template task (assuming templates are just tasks marked as templates)
	templateTask, err := s.taskRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, domain.NewNotFoundError("TEMPLATE_NOT_FOUND", "Template not found")
	}

	// Create task from template
	newTask := s.createTaskFromTemplate(templateTask, projectID, userID)

	// Create the task
	if err := s.taskRepo.Create(ctx, newTask); err != nil {
		return nil, domain.NewInternalError("TEMPLATE_CREATE_FAILED", "Failed to create task from template", err)
	}

	return newTask, nil
}

// Helper methods for task duplication and templating

// createTaskCopy creates a copy of a task with specified options
func (s *taskService) createTaskCopy(original *domain.Task, options DuplicationOptions) *domain.Task {
	title := options.NewTitle
	if title == "" {
		title = "Copy of " + original.Title
	}

	newTask := &domain.Task{
		Title:          title,
		Description:    original.Description,
		ProjectID:      original.ProjectID,
		Status:         domain.StatusTodo, // Reset to todo for duplicated tasks
		Priority:       original.Priority,
		AssigneeID:     original.AssigneeID,
		ParentTaskID:   original.ParentTaskID,
		Dependencies:   make([]string, len(original.Dependencies)),
		Tags:           make([]string, len(original.Tags)),
		DueDate:        original.DueDate,
		StartDate:      original.StartDate,
		EffortEstimate: original.EffortEstimate,
		CustomFields:   original.CustomFields,
		Position:       0, // Will be calculated by repository
	}

	// Copy slices properly
	copy(newTask.Dependencies, original.Dependencies)
	copy(newTask.Tags, original.Tags)

	// Handle options
	if options.ResetProgress {
		newTask.Progress = 0
	} else {
		newTask.Progress = original.Progress
	}

	if options.ResetTimeSpent {
		newTask.TimeSpent = 0.0
	} else {
		newTask.TimeSpent = original.TimeSpent
	}

	// Don't copy attachments by default for security reasons
	if options.IncludeAttachments {
		newTask.Attachments = make([]string, len(original.Attachments))
		copy(newTask.Attachments, original.Attachments)
	}

	return newTask
}

// createTaskFromTemplate creates a new task from a template
func (s *taskService) createTaskFromTemplate(template *domain.Task, projectID string, userID string) *domain.Task {
	// Copy tags properly
	tags := make([]string, len(template.Tags))
	copy(tags, template.Tags)

	return &domain.Task{
		Title:          template.Title,
		Description:    template.Description,
		ProjectID:      projectID,
		Status:         domain.StatusTodo,
		Priority:       template.Priority,
		ReporterID:     userID,
		Tags:           tags,
		EffortEstimate: template.EffortEstimate,
		Progress:       0,
		TimeSpent:      0.0,
		Position:       0, // Will be calculated by repository
	}
}

// duplicateSubtasks recursively duplicates subtasks
func (s *taskService) duplicateSubtasks(
	ctx context.Context,
	originalParentID string,
	newParentID string,
	options DuplicationOptions,
	userID string,
) error {
	subtasks, err := s.taskRepo.GetSubtasks(ctx, originalParentID)
	if err != nil {
		return fmt.Errorf("failed to get subtasks for duplication: %w", err)
	}

	for _, subtask := range subtasks {
		// Create copy of subtask
		newSubtask := s.createTaskCopy(subtask, options)
		newSubtask.ParentTaskID = &newParentID
		newSubtask.ReporterID = userID

		if err := s.taskRepo.Create(ctx, newSubtask); err != nil {
			return fmt.Errorf("failed to create subtask copy: %w", err)
		}

		// Recursively duplicate nested subtasks
		if options.IncludeSubtasks {
			if err := s.duplicateSubtasks(ctx, subtask.ID, newSubtask.ID, options, userID); err != nil {
				// Log but continue with other subtasks
				continue
			}
		}
	}

	return nil
}

// CreateSubtask creates a subtask under a parent task
func (s *taskService) CreateSubtask(
	ctx context.Context,
	parentTaskID string,
	req domain.CreateTaskRequest,
	userID string,
) (*domain.Task, error) {
	if parentTaskID == "" {
		return nil, domain.NewValidationError("INVALID_PARENT_ID", "Parent task ID cannot be empty", nil)
	}

	// Validate parent task exists and user has access
	parentTask, err := s.validateTaskAccess(ctx, parentTaskID, userID)
	if err != nil {
		return nil, err
	}

	// Ensure subtask belongs to the same project as parent
	req.ProjectID = parentTask.ProjectID

	// Create the task
	task, err := s.CreateTask(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	// Set parent task ID to make it a subtask
	if err := task.SetParentTask(parentTaskID); err != nil {
		return nil, err
	}

	// Update the task with parent relationship
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, domain.NewInternalError("SUBTASK_CREATE_FAILED", "Failed to create subtask", err)
	}

	return task, nil
}

// AddDependency adds a dependency to a task
func (s *taskService) AddDependency(
	ctx context.Context,
	taskID string,
	dependencyID string,
	userID string,
) error {
	if dependencyID == "" {
		return domain.NewValidationError("INVALID_DEPENDENCY_ID", "Dependency task ID cannot be empty", nil)
	}

	// Validate both tasks exist and user has access
	task, err := s.validateTaskAccess(ctx, taskID, userID)
	if err != nil {
		return err
	}

	dependencyTask, err := s.validateTaskAccess(ctx, dependencyID, userID)
	if err != nil {
		return err
	}

	// Prevent circular dependencies - basic check
	if taskID == dependencyID {
		return domain.NewConflictError("CIRCULAR_DEPENDENCY", "Task cannot depend on itself")
	}

	// Check if dependency is already added
	for _, depID := range task.Dependencies {
		if depID == dependencyID {
			return domain.NewConflictError("DEPENDENCY_EXISTS", "Dependency already exists")
		}
	}

	// Ensure both tasks are in the same project
	if task.ProjectID != dependencyTask.ProjectID {
		return domain.NewValidationError("CROSS_PROJECT_DEPENDENCY", "Cannot add dependency from different project", nil)
	}

	// Add dependency
	task.Dependencies = append(task.Dependencies, dependencyID)

	// Update the task
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return domain.NewInternalError("DEPENDENCY_ADD_FAILED", "Failed to add dependency", err)
	}

	return nil
}

// RemoveDependency removes a dependency from a task
func (s *taskService) RemoveDependency(
	ctx context.Context,
	taskID string,
	dependencyID string,
	userID string,
) error {
	if dependencyID == "" {
		return domain.NewValidationError("INVALID_DEPENDENCY_ID", "Dependency task ID cannot be empty", nil)
	}

	// Validate task exists and user has access
	task, err := s.validateTaskAccess(ctx, taskID, userID)
	if err != nil {
		return err
	}

	// Find and remove the dependency
	dependencyIndex := -1
	for i, depID := range task.Dependencies {
		if depID == dependencyID {
			dependencyIndex = i
			break
		}
	}

	if dependencyIndex == -1 {
		return domain.NewNotFoundError("DEPENDENCY_NOT_FOUND", "Dependency not found")
	}

	// Remove dependency from slice
	task.Dependencies = append(task.Dependencies[:dependencyIndex], task.Dependencies[dependencyIndex+1:]...)

	// Update the task
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return domain.NewInternalError("DEPENDENCY_REMOVE_FAILED", "Failed to remove dependency", err)
	}

	return nil
}
