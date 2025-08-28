// Package repository provides data access interfaces following SOLID principles.
package repository

import (
	"context"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// GitHubAuthSessionRepository defines the interface for GitHub auth session data access.
type GitHubAuthSessionRepository interface {
	// Create creates a new GitHub auth session
	Create(ctx context.Context, session *domain.GitHubAuthSession) error

	// GetByUserID retrieves a GitHub auth session by user ID
	GetByUserID(ctx context.Context, userID string) (*domain.GitHubAuthSession, error)

	// DeleteByUserID deletes a GitHub auth session by user ID
	DeleteByUserID(ctx context.Context, userID string) error

	// DeleteExpired deletes all expired GitHub auth sessions
	DeleteExpired(ctx context.Context) error
}
