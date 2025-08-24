package repository

import (
	"context"
	"simple-easy-tasks/internal/domain"
)

// UserRepository defines the interface for user data operations.
// Following Interface Segregation Principle.
type UserRepository interface {
	// Create creates a new user.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by email address.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetByUsername retrieves a user by username.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *domain.User) error

	// Delete deletes a user by ID.
	Delete(ctx context.Context, id string) error

	// List retrieves users with pagination.
	List(ctx context.Context, offset, limit int) ([]*domain.User, error)

	// Count returns the total number of users.
	Count(ctx context.Context) (int, error)

	// ExistsByEmail checks if a user exists with the given email.
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername checks if a user exists with the given username.
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// UserQueryRepository defines read-only operations for user queries.
// Following Interface Segregation Principle.
type UserQueryRepository interface {
	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by email address.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetByUsername retrieves a user by username.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// List retrieves users with pagination.
	List(ctx context.Context, offset, limit int) ([]*domain.User, error)

	// Count returns the total number of users.
	Count(ctx context.Context) (int, error)

	// ExistsByEmail checks if a user exists with the given email.
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername checks if a user exists with the given username.
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// UserCommandRepository defines write operations for users.
// Following Interface Segregation Principle.
type UserCommandRepository interface {
	// Create creates a new user.
	Create(ctx context.Context, user *domain.User) error

	// Update updates an existing user.
	Update(ctx context.Context, user *domain.User) error

	// Delete deletes a user by ID.
	Delete(ctx context.Context, id string) error
}
