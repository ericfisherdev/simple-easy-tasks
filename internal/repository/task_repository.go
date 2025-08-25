package repository

import (
	"context"

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

	// ListByProject retrieves tasks for a specific project
	ListByProject(ctx context.Context, projectID string, offset, limit int) ([]*domain.Task, error)

	// ListByAssignee retrieves tasks assigned to a specific user
	ListByAssignee(ctx context.Context, assigneeID string, offset, limit int) ([]*domain.Task, error)

	// ListByStatus retrieves tasks by status
	ListByStatus(ctx context.Context, status domain.TaskStatus, offset, limit int) ([]*domain.Task, error)

	// ListByCreator retrieves tasks created by a specific user
	ListByCreator(ctx context.Context, creatorID string, offset, limit int) ([]*domain.Task, error)

	// Search searches tasks by title, description or content
	Search(ctx context.Context, query string, projectID string, offset, limit int) ([]*domain.Task, error)

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

	// BulkUpdate updates multiple tasks
	BulkUpdate(ctx context.Context, tasks []*domain.Task) error

	// BulkDelete deletes multiple tasks
	BulkDelete(ctx context.Context, ids []string) error

	// ArchiveTask archives a task instead of deleting it
	ArchiveTask(ctx context.Context, id string) error

	// UnarchiveTask unarchives a task
	UnarchiveTask(ctx context.Context, id string) error
}
