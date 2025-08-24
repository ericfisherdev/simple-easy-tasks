package repository

//nolint:gofumpt
import (
	"context"
	"fmt"

	"simple-easy-tasks/internal/domain"

	"github.com/pocketbase/pocketbase"
)

// pocketbaseUserRepository implements UserRepository using PocketBase.
type pocketbaseUserRepository struct {
	app *pocketbase.PocketBase
}

// NewPocketBaseUserRepository creates a new PocketBase user repository.
func NewPocketBaseUserRepository(app *pocketbase.PocketBase) UserRepository {
	return &pocketbaseUserRepository{
		app: app,
	}
}

// Create creates a new user in PocketBase.
func (r *pocketbaseUserRepository) Create(_ context.Context, _ *domain.User) error {
	// TODO: Implement PocketBase user creation when DAO access is available
	return fmt.Errorf("PocketBase user creation not yet implemented")
}

// GetByID retrieves a user by ID from PocketBase.
func (r *pocketbaseUserRepository) GetByID(_ context.Context, _ string) (*domain.User, error) {
	// TODO: Implement PocketBase user retrieval when DAO access is available
	return nil, fmt.Errorf("PocketBase user retrieval not yet implemented")
}

// GetByEmail retrieves a user by email from PocketBase.
func (r *pocketbaseUserRepository) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	// TODO: Implement PocketBase user retrieval by email when DAO access is available
	return nil, fmt.Errorf("PocketBase user retrieval by email not yet implemented")
}

// GetByUsername retrieves a user by username from PocketBase.
func (r *pocketbaseUserRepository) GetByUsername(_ context.Context, _ string) (*domain.User, error) {
	// TODO: Implement PocketBase user retrieval by username when DAO access is available
	return nil, fmt.Errorf("PocketBase user retrieval by username not yet implemented")
}

// Update updates an existing user in PocketBase.
func (r *pocketbaseUserRepository) Update(_ context.Context, _ *domain.User) error {
	// TODO: Implement PocketBase user update when DAO access is available
	return fmt.Errorf("PocketBase user update not yet implemented")
}

// Delete deletes a user by ID from PocketBase.
func (r *pocketbaseUserRepository) Delete(_ context.Context, _ string) error {
	// TODO: Implement PocketBase user deletion when DAO access is available
	return fmt.Errorf("PocketBase user deletion not yet implemented")
}

// List retrieves users with pagination from PocketBase.
func (r *pocketbaseUserRepository) List(_ context.Context, _, _ int) ([]*domain.User, error) {
	// TODO: Implement PocketBase user listing when DAO access is available
	return nil, fmt.Errorf("PocketBase user listing not yet implemented")
}

// Count returns the total number of users in PocketBase.
func (r *pocketbaseUserRepository) Count(_ context.Context) (int, error) {
	// TODO: Implement PocketBase user count when DAO access is available
	return 0, fmt.Errorf("PocketBase user count not yet implemented")
}

// ExistsByEmail checks if a user exists with the given email.
func (r *pocketbaseUserRepository) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	// TODO: Implement PocketBase user existence check by email when DAO access is available
	return false, fmt.Errorf("PocketBase user existence check by email not yet implemented")
}

// ExistsByUsername checks if a user exists with the given username.
func (r *pocketbaseUserRepository) ExistsByUsername(_ context.Context, _ string) (bool, error) {
	// TODO: Implement PocketBase user existence check by username when DAO access is available
	return false, fmt.Errorf("PocketBase user existence check by username not yet implemented")
}
