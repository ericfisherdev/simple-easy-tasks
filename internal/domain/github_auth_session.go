// Package domain provides core business entities following SOLID principles.
package domain

import (
	"time"
)

// GitHubAuthSession represents a temporary GitHub auth session stored server-side
type GitHubAuthSession struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	AccessToken string    `json:"-"` // Never serialize the token
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Validate validates the GitHubAuthSession
func (g *GitHubAuthSession) Validate() error {
	if g.UserID == "" {
		return NewValidationError("user_id", "User ID is required", nil)
	}
	if g.AccessToken == "" {
		return NewValidationError("access_token", "Access token is required", nil)
	}
	// Allow expired sessions to be created (they will be handled by the repository)
	return nil
}

// IsExpired checks if the session has expired
func (g *GitHubAuthSession) IsExpired() bool {
	return time.Now().After(g.ExpiresAt)
}
