// Package domain contains business entities and interfaces for the Simple Easy Tasks application.
package domain

import (
	"context"
	"time"
)

// PasswordResetToken represents a password reset token stored in the database.
type PasswordResetToken struct {
	ExpiresAt time.Time `json:"expires_at"` // When the token expires
	CreatedAt time.Time `json:"created_at"` // When the token was created
	UpdatedAt time.Time `json:"updated_at"` // When the token was last updated
	ID        string    `json:"id"`
	Token     string    `json:"token"`   // The reset token
	UserID    string    `json:"user_id"` // User the token belongs to
	Used      bool      `json:"used"`    // Whether the token has been used
}

// IsExpired checks if the token is expired.
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid checks if the token is valid (not expired and not used).
func (t *PasswordResetToken) IsValid() bool {
	return !t.Used && !t.IsExpired()
}

// MarkAsUsed marks the token as used.
func (t *PasswordResetToken) MarkAsUsed() {
	t.Used = true
	t.UpdatedAt = time.Now()
}

// PasswordResetTokenRepository defines the interface for password reset token operations.
type PasswordResetTokenRepository interface {
	// Create stores a new password reset token
	Create(ctx context.Context, token *PasswordResetToken) error

	// GetByToken retrieves a token by its token value
	GetByToken(ctx context.Context, token string) (*PasswordResetToken, error)

	// Update updates a password reset token
	Update(ctx context.Context, token *PasswordResetToken) error

	// Delete removes a password reset token
	Delete(ctx context.Context, tokenID string) error

	// CleanupExpiredTokens removes expired tokens
	CleanupExpiredTokens(ctx context.Context) error

	// InvalidateUserTokens marks all tokens for a user as used
	InvalidateUserTokens(ctx context.Context, userID string) error
}
