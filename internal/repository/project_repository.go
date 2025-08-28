package repository

//nolint:gofumpt
import (
	"context"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// ProjectRepository defines the interface for project data operations.
// Following Interface Segregation Principle.
type ProjectRepository interface {
	// Create creates a new project.
	Create(ctx context.Context, project *domain.Project) error

	// GetByID retrieves a project by ID.
	GetByID(ctx context.Context, id string) (*domain.Project, error)

	// GetBySlug retrieves a project by slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)

	// Update updates an existing project.
	Update(ctx context.Context, project *domain.Project) error

	// Delete deletes a project by ID.
	Delete(ctx context.Context, id string) error

	// ListByOwner retrieves projects owned by a specific user.
	ListByOwner(ctx context.Context, ownerID string, offset, limit int) ([]*domain.Project, error)

	// ListByMember retrieves projects where a user is a member.
	ListByMember(ctx context.Context, memberID string, offset, limit int) ([]*domain.Project, error)

	// List retrieves projects with pagination.
	List(ctx context.Context, offset, limit int) ([]*domain.Project, error)

	// Count returns the total number of projects.
	Count(ctx context.Context) (int, error)

	// ExistsBySlug checks if a project exists with the given slug.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// GetMemberProjects retrieves all projects where user has access.
	GetMemberProjects(ctx context.Context, userID string, offset, limit int) ([]*domain.Project, error)
}

// ProjectQueryRepository defines read-only operations for project queries.
// Following Interface Segregation Principle.
type ProjectQueryRepository interface {
	// GetByID retrieves a project by ID.
	GetByID(ctx context.Context, id string) (*domain.Project, error)

	// GetBySlug retrieves a project by slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Project, error)

	// ListByOwner retrieves projects owned by a specific user.
	ListByOwner(ctx context.Context, ownerID string, offset, limit int) ([]*domain.Project, error)

	// ListByMember retrieves projects where a user is a member.
	ListByMember(ctx context.Context, memberID string, offset, limit int) ([]*domain.Project, error)

	// List retrieves projects with pagination.
	List(ctx context.Context, offset, limit int) ([]*domain.Project, error)

	// Count returns the total number of projects.
	Count(ctx context.Context) (int, error)

	// ExistsBySlug checks if a project exists with the given slug.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// GetMemberProjects retrieves all projects where user has access.
	GetMemberProjects(ctx context.Context, userID string, offset, limit int) ([]*domain.Project, error)
}

// ProjectCommandRepository defines write operations for projects.
// Following Interface Segregation Principle.
type ProjectCommandRepository interface {
	// Create creates a new project.
	Create(ctx context.Context, project *domain.Project) error

	// Update updates an existing project.
	Update(ctx context.Context, project *domain.Project) error

	// Delete deletes a project by ID.
	Delete(ctx context.Context, id string) error
}
