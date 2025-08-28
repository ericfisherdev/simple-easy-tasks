package repository

import (
	"context"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// CommentRepository defines the interface for comment data access operations.
type CommentRepository interface {
	CommentQueryRepository
	CommentCommandRepository
}

// CommentQueryRepository defines query operations for comments.
type CommentQueryRepository interface {
	// GetByID retrieves a comment by its ID
	GetByID(ctx context.Context, id string) (*domain.Comment, error)

	// ListByTask retrieves comments for a specific task
	ListByTask(ctx context.Context, taskID string, offset, limit int) ([]*domain.Comment, error)

	// ListByAuthor retrieves comments by a specific author
	ListByAuthor(ctx context.Context, authorID string, offset, limit int) ([]*domain.Comment, error)

	// ListReplies retrieves replies to a specific comment
	ListReplies(ctx context.Context, parentID string) ([]*domain.Comment, error)

	// GetThread retrieves a comment thread (comment and its replies)
	GetThread(ctx context.Context, rootCommentID string) ([]*domain.Comment, error)

	// Count returns the total number of comments
	Count(ctx context.Context) (int, error)

	// CountByTask returns the number of comments for a task
	CountByTask(ctx context.Context, taskID string) (int, error)

	// CountByAuthor returns the number of comments by an author
	CountByAuthor(ctx context.Context, authorID string) (int, error)

	// ExistsByID checks if a comment exists by ID
	ExistsByID(ctx context.Context, id string) (bool, error)

	// Search searches comments by content
	Search(ctx context.Context, query string, taskID string, offset, limit int) ([]*domain.Comment, error)
}

// CommentCommandRepository defines command operations for comments.
type CommentCommandRepository interface {
	// Create creates a new comment
	Create(ctx context.Context, comment *domain.Comment) error

	// Update updates an existing comment
	Update(ctx context.Context, comment *domain.Comment) error

	// Delete deletes a comment by ID
	Delete(ctx context.Context, id string) error

	// BulkDelete deletes multiple comments
	BulkDelete(ctx context.Context, ids []string) error

	// DeleteByTask deletes all comments for a task
	DeleteByTask(ctx context.Context, taskID string) error

	// SoftDelete marks a comment as deleted without removing it
	SoftDelete(ctx context.Context, id string) error

	// Restore restores a soft-deleted comment
	Restore(ctx context.Context, id string) error
}
