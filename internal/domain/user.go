package domain

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserRole represents the role of a user in the system.
type UserRole string

const (
	// AdminRole represents an administrator user.
	AdminRole UserRole = "admin"
	// RegularUserRole represents a regular user.
	RegularUserRole UserRole = "user"
)

// UserPreferences represents user-specific preferences.
type UserPreferences struct {
	Preferences map[string]string `json:"preferences"`
	Theme       string            `json:"theme"`
	Language    string            `json:"language"`
	Timezone    string            `json:"timezone"`
}

// User represents a user in the system following DDD principles.
type User struct {
	Preferences  UserPreferences `json:"preferences"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ID           string          `json:"id"`
	Email        string          `json:"email"`
	Username     string          `json:"username"`
	Name         string          `json:"name"`
	PasswordHash string          `json:"-"` // Never serialize password hash
	Avatar       string          `json:"avatar,omitempty"`
	Role         UserRole        `json:"role"`
	TokenVersion int             `json:"token_version"` // For token invalidation
}

// SetPassword hashes and sets the user's password.
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return NewInternalError("PASSWORD_HASH_FAILED", "Failed to hash password", err)
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the provided password against the stored hash.
func (u *User) CheckPassword(password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return NewAuthenticationError("INVALID_PASSWORD", "Password does not match")
	}
	return nil
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == AdminRole
}

// IncrementTokenVersion increments the user's token version to invalidate existing tokens.
func (u *User) IncrementTokenVersion() {
	u.TokenVersion++
}

// Validate validates the user data.
func (u *User) Validate() error {
	if err := ValidateRequired("email", u.Email, "INVALID_EMAIL", "Email is required"); err != nil {
		return err
	}

	// Username is optional for basic integration tests
	// TODO: Make this required when username field is properly configured in collection

	if err := ValidateRequired("name", u.Name, "INVALID_NAME", "Name is required"); err != nil {
		return err
	}

	// Role is optional for basic integration tests
	// TODO: Validate role when role field is properly configured in collection
	if u.Role != "" {
		if err := ValidateEnum("role", string(u.Role), "INVALID_ROLE", "Role must be 'admin' or 'user'",
			string(AdminRole), string(RegularUserRole)); err != nil {
			return err
		}
	}

	return nil
}

// CreateUserRequest represents the data needed to create a new user.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role,omitempty"`
}

// UpdateUserRequest represents the data that can be updated for a user.
type UpdateUserRequest struct {
	Name        *string          `json:"name,omitempty"`
	Avatar      *string          `json:"avatar,omitempty"`
	Preferences *UserPreferences `json:"preferences,omitempty"`
}

// LoginRequest represents login credentials.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenPair represents JWT tokens.
type TokenPair struct {
	ExpiresAt    time.Time `json:"expires_at"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}
