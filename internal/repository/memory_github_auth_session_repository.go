// Package repository provides data access interfaces following SOLID principles.
package repository

import (
	"context"
	"fmt"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"sync"
	"time"
)

// memoryGitHubAuthSessionRepository provides an in-memory implementation of GitHubAuthSessionRepository.
type memoryGitHubAuthSessionRepository struct {
	sessions map[string]*domain.GitHubAuthSession
	mutex    sync.RWMutex
}

// NewMemoryGitHubAuthSessionRepository creates a new in-memory GitHub auth session repository.
func NewMemoryGitHubAuthSessionRepository() GitHubAuthSessionRepository {
	return &memoryGitHubAuthSessionRepository{
		sessions: make(map[string]*domain.GitHubAuthSession),
	}
}

// Create creates a new GitHub auth session
func (r *memoryGitHubAuthSessionRepository) Create(ctx context.Context, session *domain.GitHubAuthSession) error {
	if err := session.Validate(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Remove any existing session for this user
	delete(r.sessions, session.UserID)

	// Store the new session
	r.sessions[session.UserID] = session

	return nil
}

// GetByUserID retrieves a GitHub auth session by user ID
func (r *memoryGitHubAuthSessionRepository) GetByUserID(ctx context.Context, userID string) (*domain.GitHubAuthSession, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	session, exists := r.sessions[userID]
	if !exists {
		return nil, fmt.Errorf("GitHub auth session not found for user %s", userID)
	}

	if session.IsExpired() {
		// Clean up expired session
		r.mutex.RUnlock()
		r.mutex.Lock()
		delete(r.sessions, userID)
		r.mutex.Unlock()
		r.mutex.RLock()
		return nil, fmt.Errorf("GitHub auth session expired for user %s", userID)
	}

	return session, nil
}

// DeleteByUserID deletes a GitHub auth session by user ID
func (r *memoryGitHubAuthSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.sessions, userID)
	return nil
}

// DeleteExpired deletes all expired GitHub auth sessions
func (r *memoryGitHubAuthSessionRepository) DeleteExpired(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	for userID, session := range r.sessions {
		if session.ExpiresAt.Before(now) {
			delete(r.sessions, userID)
		}
	}

	return nil
}
