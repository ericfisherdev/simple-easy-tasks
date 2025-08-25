package repository

//nolint:gofumpt
import (
	"context"
	"fmt"
	"time"

	"simple-easy-tasks/internal/domain"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// pocketbaseUserRepository implements UserRepository using PocketBase.
type pocketbaseUserRepository struct {
	app core.App
}

// NewPocketBaseUserRepository creates a new PocketBase user repository.
func NewPocketBaseUserRepository(app core.App) UserRepository {
	return &pocketbaseUserRepository{
		app: app,
	}
}

// Create creates a new user in PocketBase.
func (r *pocketbaseUserRepository) Create(_ context.Context, user *domain.User) error {
	if err := user.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	collection, err := r.app.FindCollectionByNameOrId("users")
	if err != nil {
		return fmt.Errorf("failed to find users collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("email", user.Email)
	record.Set("username", user.Username)
	record.Set("name", user.Name)
	record.Set("avatar", user.Avatar)
	record.Set("role", string(user.Role))
	record.Set("preferences", user.Preferences)

	// Set password using the auth record's SetPassword method
	if user.PasswordHash != "" {
		// If we already have a hash, set it directly
		record.Set("password", user.PasswordHash)
	}

	if !user.CreatedAt.IsZero() {
		record.Set("created", user.CreatedAt)
	}
	if !user.UpdatedAt.IsZero() {
		record.Set("updated", user.UpdatedAt)
	}

	if user.ID != "" {
		record.Id = user.ID
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save user record: %w", err)
	}

	user.ID = record.Id
	if createdTime := record.GetDateTime("created"); !createdTime.IsZero() {
		user.CreatedAt = createdTime.Time()
	}
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		user.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// GetByID retrieves a user by ID from PocketBase.
func (r *pocketbaseUserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	record, err := r.app.FindRecordById("users", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by ID %s: %w", id, err)
	}

	return r.recordToUser(record)
}

// GetByEmail retrieves a user by email from PocketBase.
func (r *pocketbaseUserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, fmt.Errorf("user email cannot be empty")
	}

	record, err := r.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email %s: %w", email, err)
	}

	return r.recordToUser(record)
}

// GetByUsername retrieves a user by username from PocketBase.
func (r *pocketbaseUserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	record, err := r.app.FindFirstRecordByFilter("users", "username = {:username}", dbx.Params{"username": username})
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username %s: %w", username, err)
	}

	return r.recordToUser(record)
}

// Update updates an existing user in PocketBase.
func (r *pocketbaseUserRepository) Update(_ context.Context, user *domain.User) error {
	if err := user.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if user.ID == "" {
		return fmt.Errorf("user ID cannot be empty for update")
	}

	record, err := r.app.FindRecordById("users", user.ID)
	if err != nil {
		return fmt.Errorf("failed to find user for update: %w", err)
	}

	record.Set("email", user.Email)
	record.Set("username", user.Username)
	record.Set("name", user.Name)
	record.Set("avatar", user.Avatar)
	record.Set("role", string(user.Role))
	record.Set("preferences", user.Preferences)
	record.Set("updated", time.Now())

	// Only update password if it has changed
	if user.PasswordHash != "" {
		record.Set("password", user.PasswordHash)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update user record: %w", err)
	}

	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		user.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// Delete deletes a user by ID from PocketBase.
func (r *pocketbaseUserRepository) Delete(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	record, err := r.app.FindRecordById("users", id)
	if err != nil {
		return fmt.Errorf("failed to find user for deletion: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete user record: %w", err)
	}

	return nil
}

// List retrieves users with pagination from PocketBase.
func (r *pocketbaseUserRepository) List(_ context.Context, offset, limit int) ([]*domain.User, error) {
	records, err := r.app.FindRecordsByFilter("users", "", "-created", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return r.recordsToUsers(records)
}

// Count returns the total number of users in PocketBase.
func (r *pocketbaseUserRepository) Count(_ context.Context) (int, error) {
	total, err := r.app.CountRecords("users")
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return int(total), nil
}

// ExistsByEmail checks if a user exists with the given email.
func (r *pocketbaseUserRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	if email == "" {
		return false, fmt.Errorf("email cannot be empty")
	}

	_, err := r.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user existence by email: %w", err)
	}

	return true, nil
}

// ExistsByUsername checks if a user exists with the given username.
func (r *pocketbaseUserRepository) ExistsByUsername(_ context.Context, username string) (bool, error) {
	if username == "" {
		return false, fmt.Errorf("username cannot be empty")
	}

	_, err := r.app.FindFirstRecordByFilter("users", "username = {:username}", dbx.Params{"username": username})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user existence by username: %w", err)
	}

	return true, nil
}

// recordToUser converts a PocketBase record to a domain.User.
func (r *pocketbaseUserRepository) recordToUser(record *core.Record) (*domain.User, error) {
	var preferences domain.UserPreferences
	if err := record.UnmarshalJSONField("preferences", &preferences); err != nil {
		preferences = domain.UserPreferences{}
	}

	user := &domain.User{
		ID:           record.Id,
		Email:        record.GetString("email"),
		Username:     record.GetString("username"),
		Name:         record.GetString("name"),
		PasswordHash: record.GetString("password"),
		Avatar:       record.GetString("avatar"),
		Role:         domain.UserRole(record.GetString("role")),
		Preferences:  preferences,
		CreatedAt:    record.GetDateTime("created").Time(),
		UpdatedAt:    record.GetDateTime("updated").Time(),
	}

	return user, nil
}

// recordsToUsers converts PocketBase records to domain.User slice.
func (r *pocketbaseUserRepository) recordsToUsers(records []*core.Record) ([]*domain.User, error) {
	users := make([]*domain.User, len(records))
	for i, record := range records {
		user, err := r.recordToUser(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to user: %w", err)
		}
		users[i] = user
	}
	return users, nil
}
