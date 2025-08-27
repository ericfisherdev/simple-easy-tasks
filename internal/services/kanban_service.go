package services

import (
	"context"
	"time"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// KanbanBoard represents the complete kanban board state
type KanbanBoard struct {
	ProjectID string                    `json:"project_id"`
	Columns   map[domain.TaskStatus]*Column    `json:"columns"`
	Stats     *BoardStatistics         `json:"stats"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// Column represents a single column in the kanban board
type Column struct {
	Status TaskStatus  `json:"status"`
	Title  string      `json:"title"`
	Tasks  []*domain.Task     `json:"tasks"`
	Count  int         `json:"count"`
	WIP    *WIPLimits  `json:"wip_limits"`
}

// BoardStatistics provides analytics about the board state
type BoardStatistics struct {
	TotalTasks      int                        `json:"total_tasks"`
	TasksByStatus   map[domain.TaskStatus]int       `json:"tasks_by_status"`
	TasksByPriority map[domain.TaskPriority]int     `json:"tasks_by_priority"`
	OverdueTasks    int                        `json:"overdue_tasks"`
	UnassignedTasks int                        `json:"unassigned_tasks"`
}

// WIPLimits defines work-in-progress limits for a column
type WIPLimits struct {
	SoftLimit int  `json:"soft_limit"`
	HardLimit int  `json:"hard_limit"`
	Enabled   bool `json:"enabled"`
}

// TaskStatus is an alias to avoid import cycles
type TaskStatus = domain.TaskStatus

// KanbanService defines the interface for kanban board operations
type KanbanService interface {
	// GetBoard retrieves the complete kanban board for a project
	GetBoard(ctx context.Context, projectID string, userID string) (*KanbanBoard, error)
	
	// MoveTask moves a task between columns with position management
	MoveTask(ctx context.Context, req MoveTaskRequest, userID string) error
	
	// UpdateWIPLimits updates work-in-progress limits for a column
	UpdateWIPLimits(ctx context.Context, projectID string, status domain.TaskStatus, limits WIPLimits, userID string) error
	
	// GetBoardStatistics returns analytics for the board
	GetBoardStatistics(ctx context.Context, projectID string, userID string) (*BoardStatistics, error)
	
	// ValidateMove checks if a move is allowed based on WIP limits and business rules
	ValidateMove(ctx context.Context, req MoveTaskRequest, userID string) error
}

// kanbanService implements the KanbanService interface
type kanbanService struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	taskService TaskService
}

// NewKanbanService creates a new kanban service
func NewKanbanService(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	taskService TaskService,
) KanbanService {
	return &kanbanService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		taskService: taskService,
	}
}

// GetBoard retrieves the complete kanban board for a project
func (s *kanbanService) GetBoard(ctx context.Context, projectID string, userID string) (*KanbanBoard, error) {
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

	// Get all tasks for the project
	filters := repository.TaskFilters{
		SortBy:    "position",
		SortOrder: "asc",
		Limit:     1000, // Set a reasonable limit
	}
	
	tasks, err := s.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return nil, domain.NewInternalError("BOARD_LOAD_FAILED", "Failed to load board tasks", err)
	}

	// Organize tasks into columns
	columns := s.organizeTasks(tasks)
	
	// Calculate statistics
	stats := s.calculateStatistics(tasks)
	
	// Create board
	board := &KanbanBoard{
		ProjectID: projectID,
		Columns:   columns,
		Stats:     stats,
		UpdatedAt: time.Now().UTC(),
	}

	return board, nil
}

// organizeTasks groups tasks by status into columns
func (s *kanbanService) organizeTasks(tasks []*domain.Task) map[domain.TaskStatus]*Column {
	columns := make(map[domain.TaskStatus]*Column)
	
	// Initialize all columns
	statuses := []domain.TaskStatus{
		domain.StatusBacklog,
		domain.StatusTodo,
		domain.StatusDeveloping,
		domain.StatusReview,
		domain.StatusComplete,
	}
	
	titles := map[domain.TaskStatus]string{
		domain.StatusBacklog:    "Backlog",
		domain.StatusTodo:       "To Do",
		domain.StatusDeveloping: "In Progress",
		domain.StatusReview:     "Review",
		domain.StatusComplete:   "Complete",
	}
	
	for _, status := range statuses {
		columns[status] = &Column{
			Status: status,
			Title:  titles[status],
			Tasks:  []*domain.Task{},
			Count:  0,
			WIP:    &WIPLimits{Enabled: false}, // Default WIP limits off
		}
	}
	
	// Group tasks by status
	for _, task := range tasks {
		if task.IsArchived() {
			continue // Skip archived tasks
		}
		
		if column, exists := columns[task.Status]; exists {
			column.Tasks = append(column.Tasks, task)
			column.Count++
		}
	}
	
	return columns
}

// calculateStatistics computes board analytics
func (s *kanbanService) calculateStatistics(tasks []*domain.Task) *BoardStatistics {
	stats := &BoardStatistics{
		TasksByStatus:   make(map[domain.TaskStatus]int),
		TasksByPriority: make(map[domain.TaskPriority]int),
	}
	
	now := time.Now()
	
	for _, task := range tasks {
		if task.IsArchived() {
			continue // Skip archived tasks in statistics
		}
		
		stats.TotalTasks++
		stats.TasksByStatus[task.Status]++
		stats.TasksByPriority[task.Priority]++
		
		// Check if task is overdue
		if task.DueDate != nil && task.DueDate.Before(now) && task.Status != domain.StatusComplete {
			stats.OverdueTasks++
		}
		
		// Check if task is unassigned
		if task.AssigneeID == nil {
			stats.UnassignedTasks++
		}
	}
	
	return stats
}

// MoveTask moves a task between columns with position management
func (s *kanbanService) MoveTask(ctx context.Context, req MoveTaskRequest, userID string) error {
	// First validate the move
	if err := s.ValidateMove(ctx, req, userID); err != nil {
		return err
	}
	
	// Get the task to move
	task, err := s.taskRepo.GetByID(ctx, req.TaskID)
	if err != nil {
		return err
	}
	
	// Calculate new position if needed
	newPosition := req.NewPosition
	if newPosition == 0 {
		// Calculate next position for the target column
		filters := repository.TaskFilters{
			Status:    []domain.TaskStatus{req.NewStatus},
			SortBy:    "position",
			SortOrder: "desc",
			Limit:     1,
		}
		
		columnTasks, err := s.taskRepo.GetByProject(ctx, req.ProjectID, filters)
		if err != nil {
			return domain.NewInternalError("POSITION_CALC_FAILED", "Failed to calculate new position", err)
		}
		
		if len(columnTasks) > 0 {
			newPosition = columnTasks[0].Position + 1000 // Use large increments for easy reordering
		} else {
			newPosition = 1000 // First task in column
		}
	}
	
	// Update task status and position using business logic
	if err := task.UpdateStatus(req.NewStatus); err != nil {
		return err
	}
	
	// Update position
	task.Position = newPosition
	task.UpdatedAt = time.Now().UTC()
	
	// Save changes
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return domain.NewInternalError("TASK_MOVE_FAILED", "Failed to move task", err)
	}
	
	// Reorder other tasks in the target column if necessary
	if err := s.reorderColumnTasks(ctx, req.ProjectID, req.NewStatus); err != nil {
		// Log error but don't fail the move
		// In production, use structured logging here
		_ = err
	}
	
	return nil
}

// ValidateMove checks if a move is allowed based on WIP limits and business rules
func (s *kanbanService) ValidateMove(ctx context.Context, req MoveTaskRequest, userID string) error {
	// Basic validation
	if req.TaskID == "" {
		return domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}
	if !req.NewStatus.IsValid() {
		return domain.NewValidationError("INVALID_STATUS", "Invalid task status", nil)
	}
	if req.ProjectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Check if user has access to the task
	task, err := s.taskRepo.GetByID(ctx, req.TaskID)
	if err != nil {
		return err
	}

	// Ensure the task belongs to the specified project
	if task.ProjectID != req.ProjectID {
		return domain.NewValidationError("PROJECT_MISMATCH", "Task does not belong to specified project", nil)
	}

	// Check project access
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to modify this task")
	}

	// Check if status transition is allowed
	if !task.CanTransitionTo(req.NewStatus) {
		return domain.NewConflictError("INVALID_TRANSITION", 
			"Task cannot transition from " + string(task.Status) + " to " + string(req.NewStatus))
	}

	// TODO: Check WIP limits when they are implemented
	// if err := s.checkWIPLimits(ctx, req.ProjectID, req.NewStatus); err != nil {
	//     return err
	// }

	return nil
}

// reorderColumnTasks reorders tasks in a column to prevent position conflicts
func (s *kanbanService) reorderColumnTasks(ctx context.Context, projectID string, status domain.TaskStatus) error {
	// Get all tasks in the column ordered by position
	filters := repository.TaskFilters{
		Status:    []domain.TaskStatus{status},
		SortBy:    "position",
		SortOrder: "asc",
		Limit:     1000,
	}
	
	tasks, err := s.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return err
	}
	
	// Reassign positions with proper spacing
	for i, task := range tasks {
		newPosition := (i + 1) * 1000 // 1000, 2000, 3000, etc.
		if task.Position != newPosition {
			task.Position = newPosition
			task.UpdatedAt = time.Now().UTC()
			
			// Update in repository
			if err := s.taskRepo.Update(ctx, task); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// UpdateWIPLimits updates work-in-progress limits for a column
func (s *kanbanService) UpdateWIPLimits(ctx context.Context, projectID string, status domain.TaskStatus, limits WIPLimits, userID string) error {
	// Check if user has access to the project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	// Only project owners can modify WIP limits
	if !project.IsOwner(userID) {
		return domain.NewAuthorizationError("ACCESS_DENIED", "Only project owners can modify WIP limits")
	}

	// Validate limits
	if limits.Enabled {
		if limits.SoftLimit < 0 || limits.HardLimit < 0 {
			return domain.NewValidationError("INVALID_LIMITS", "WIP limits cannot be negative", nil)
		}
		if limits.SoftLimit > limits.HardLimit && limits.HardLimit > 0 {
			return domain.NewValidationError("INVALID_LIMITS", "Soft limit cannot exceed hard limit", nil)
		}
	}

	// TODO: Store WIP limits in project settings or dedicated table
	// For now, this is a placeholder implementation
	// In a real implementation, you'd store these in the database
	// associated with the project and column status
	
	return nil
}

// GetBoardStatistics returns analytics for the board
func (s *kanbanService) GetBoardStatistics(ctx context.Context, projectID string, userID string) (*BoardStatistics, error) {
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

	// Get all tasks for statistics
	filters := repository.TaskFilters{
		Limit: 10000, // Large limit to get all tasks
	}
	
	tasks, err := s.taskRepo.GetByProject(ctx, projectID, filters)
	if err != nil {
		return nil, domain.NewInternalError("STATS_LOAD_FAILED", "Failed to load board statistics", err)
	}

	// Calculate and return statistics
	stats := s.calculateStatistics(tasks)
	return stats, nil
}