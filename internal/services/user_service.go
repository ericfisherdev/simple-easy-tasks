package services

import (
	"context"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
)

// UserService defines the interface for user-related business logic.
type UserService interface {
	// GetProfile gets a user's profile by ID
	GetProfile(ctx context.Context, userID string) (*domain.User, error)

	// UpdateProfile updates a user's profile
	UpdateProfile(ctx context.Context, userID string, req domain.UpdateUserRequest) (*domain.User, error)

	// ListUsers lists users with pagination
	ListUsers(ctx context.Context, offset, limit int) ([]*domain.User, error)

	// GetUserByEmail gets a user by email (admin only)
	GetUserByEmail(ctx context.Context, email string, currentUserID string) (*domain.User, error)

	// GetUserByUsername gets a user by username
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)

	// ExistsByEmail checks if a user exists by email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername checks if a user exists by username
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// userService implements UserService interface.
type userService struct {
	userRepo    repository.UserRepository
	authService AuthService
}

// NewUserService creates a new user service.
func NewUserService(userRepo repository.UserRepository, authService AuthService) UserService {
	return &userService{
		userRepo:    userRepo,
		authService: authService,
	}
}

// GetProfile gets a user's profile by ID.
func (s *userService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Remove sensitive information
	user.PasswordHash = ""

	return user, nil
}

// UpdateProfile updates a user's profile.
func (s *userService) UpdateProfile(
	ctx context.Context,
	userID string,
	req domain.UpdateUserRequest,
) (*domain.User, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	// Get existing user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}
	if req.Preferences != nil {
		user.Preferences = *req.Preferences
	}

	// Validate updated user
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, domain.NewInternalError("USER_UPDATE_FAILED", "Failed to update user", err)
	}

	// Remove sensitive information
	user.PasswordHash = ""

	return user, nil
}

// ListUsers lists users with pagination.
func (s *userService) ListUsers(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // Default page size
	}

	users, err := s.userRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, domain.NewInternalError("USER_LIST_FAILED", "Failed to list users", err)
	}

	// Remove sensitive information from all users
	for _, user := range users {
		user.PasswordHash = ""
	}

	return users, nil
}

// GetUserByEmail gets a user by email (admin only).
func (s *userService) GetUserByEmail(ctx context.Context, email string, currentUserID string) (*domain.User, error) {
	if email == "" {
		return nil, domain.NewValidationError("INVALID_EMAIL", "Email cannot be empty", nil)
	}

	if currentUserID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "Current user ID cannot be empty", nil)
	}

	// Check admin permissions
	currentUser, err := s.userRepo.GetByID(ctx, currentUserID)
	if err != nil {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "Unable to verify admin permissions")
	}

	if currentUser.Role != domain.AdminRole {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "Only administrators can access user information by email")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Remove sensitive information
	user.PasswordHash = ""

	return user, nil
}

// GetUserByUsername gets a user by username.
func (s *userService) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	if username == "" {
		return nil, domain.NewValidationError("INVALID_USERNAME", "Username cannot be empty", nil)
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Remove sensitive information
	user.PasswordHash = ""

	return user, nil
}

// ExistsByEmail checks if a user exists by email.
func (s *userService) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, domain.NewValidationError("INVALID_EMAIL", "Email cannot be empty", nil)
	}

	return s.userRepo.ExistsByEmail(ctx, email)
}

// ExistsByUsername checks if a user exists by username.
func (s *userService) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if username == "" {
		return false, domain.NewValidationError("INVALID_USERNAME", "Username cannot be empty", nil)
	}

	return s.userRepo.ExistsByUsername(ctx, username)
}
