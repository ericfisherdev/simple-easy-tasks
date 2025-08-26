package repository

import (
	"context"
	"time"

	"simple-easy-tasks/internal/domain"
)

// TaskRepository defines the interface for task data access operations.
type TaskRepository interface {
	TaskQueryRepository
	TaskCommandRepository
}

// TaskQueryRepository defines query operations for tasks.
type TaskQueryRepository interface {
	// GetByID retrieves a task by its ID
	GetByID(ctx context.Context, id string) (*domain.Task, error)

	// GetByProject retrieves tasks for a specific project with advanced filtering
	GetByProject(ctx context.Context, projectID string, filters TaskFilters) ([]*domain.Task, error)

	// ListByProject retrieves tasks for a specific project (legacy method)
	ListByProject(ctx context.Context, projectID string, offset, limit int) ([]*domain.Task, error)

	// ListByAssignee retrieves tasks assigned to a specific user
	ListByAssignee(ctx context.Context, assigneeID string, offset, limit int) ([]*domain.Task, error)

	// ListByStatus retrieves tasks by status
	ListByStatus(ctx context.Context, status domain.TaskStatus, offset, limit int) ([]*domain.Task, error)

	// ListByCreator retrieves tasks created by a specific user
	ListByCreator(ctx context.Context, creatorID string, offset, limit int) ([]*domain.Task, error)

	// Search searches tasks by title, description or content
	Search(ctx context.Context, query string, projectID string, offset, limit int) ([]*domain.Task, error)

	// GetSubtasks retrieves subtasks for a parent task
	GetSubtasks(ctx context.Context, parentID string) ([]*domain.Task, error)

	// GetDependencies retrieves dependency tasks for a task
	GetDependencies(ctx context.Context, taskID string) ([]*domain.Task, error)

	// GetTasksByFilter retrieves tasks using advanced filters
	GetTasksByFilter(ctx context.Context, filters TaskFilters) ([]*domain.Task, error)

	// Count returns the total number of tasks matching criteria
	Count(ctx context.Context) (int, error)

	// CountByProject returns the number of tasks in a project
	CountByProject(ctx context.Context, projectID string) (int, error)

	// CountByAssignee returns the number of tasks assigned to a user
	CountByAssignee(ctx context.Context, assigneeID string) (int, error)

	// CountByStatus returns the number of tasks with a specific status
	CountByStatus(ctx context.Context, status domain.TaskStatus) (int, error)

	// ExistsByID checks if a task exists by ID
	ExistsByID(ctx context.Context, id string) (bool, error)
}

// TaskCommandRepository defines command operations for tasks.
type TaskCommandRepository interface {
	// Create creates a new task
	Create(ctx context.Context, task *domain.Task) error

	// Update updates an existing task
	Update(ctx context.Context, task *domain.Task) error

	// Delete deletes a task by ID
	Delete(ctx context.Context, id string) error

	// Move moves a task to a new status and position
	Move(ctx context.Context, taskID string, newStatus domain.TaskStatus, position int) error

	// BulkUpdate updates multiple tasks
	BulkUpdate(ctx context.Context, tasks []*domain.Task) error

	// BulkDelete deletes multiple tasks
	BulkDelete(ctx context.Context, ids []string) error

	// BulkUpdateStatus updates multiple tasks with the same status
	BulkUpdateStatus(ctx context.Context, taskIDs []string, newStatus domain.TaskStatus) error

	// ArchiveTask archives a task instead of deleting it
	ArchiveTask(ctx context.Context, id string) error

	// UnarchiveTask unarchives a task
	UnarchiveTask(ctx context.Context, id string) error
}

// TaskFilters provides advanced filtering options for task queries
type TaskFilters struct {
	Status      []domain.TaskStatus      `json:"status,omitempty"`
	Priority    []domain.TaskPriority    `json:"priority,omitempty"`
	Tags        []string                 `json:"tags,omitempty"`
	Search      string                   `json:"search,omitempty"`
	SortBy      string                   `json:"sort_by,omitempty"`
	SortOrder   string                   `json:"sort_order,omitempty"`
	Limit       int                      `json:"limit,omitempty"`
	Offset      int                      `json:"offset,omitempty"`
	AssigneeID  *string                  `json:"assignee_id,omitempty"`
	ReporterID  *string                  `json:"reporter_id,omitempty"`
	DueBefore   *time.Time               `json:"due_before,omitempty"`
	DueAfter    *time.Time               `json:"due_after,omitempty"`
	Archived    *bool                    `json:"archived,omitempty"`
	HasParent   *bool                    `json:"has_parent,omitempty"`
	ParentID    *string                  `json:"parent_id,omitempty"`
}

// TaskUpdate represents a single task update operation for bulk updates
type TaskUpdate struct {
	Fields map[string]interface{} `json:"fields"`
	TaskID string                 `json:"task_id"`
}

// SortOrder constants for task sorting
const (
	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// Valid sort fields for tasks
const (
	SortByCreated    = "created"
	SortByUpdated    = "updated"
	SortByTitle      = "title"
	SortByStatus     = "status"
	SortByPriority   = "priority"
	SortByDueDate    = "due_date"
	SortByPosition   = "position"
	SortByProgress   = "progress"
)
