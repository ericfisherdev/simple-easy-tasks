// Package repository provides data access interfaces following SOLID principles.
package repository

import (
	"context"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryGitHubAuthSessionRepository(t *testing.T) {
	t.Run("CreateAndGetAuthSession", func(t *testing.T) {
		repo := NewMemoryGitHubAuthSessionRepository()
		ctx := context.Background()

		session := &domain.GitHubAuthSession{
			ID:          "test-session-1",
			UserID:      "user123",
			AccessToken: "github_token_123",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := repo.Create(ctx, session)
		require.NoError(t, err)

		retrieved, err := repo.GetByUserID(ctx, "user123")
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.UserID, retrieved.UserID)
		assert.Equal(t, session.AccessToken, retrieved.AccessToken)
	})

	t.Run("GetNonExistentSession", func(t *testing.T) {
		repo := NewMemoryGitHubAuthSessionRepository()
		ctx := context.Background()

		_, err := repo.GetByUserID(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub auth session not found")
	})

	t.Run("ExpiredSessionHandling", func(t *testing.T) {
		repo := NewMemoryGitHubAuthSessionRepository()
		ctx := context.Background()

		expiredSession := &domain.GitHubAuthSession{
			ID:          "expired-session",
			UserID:      "user456",
			AccessToken: "expired_token",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			ExpiresAt:   time.Now().Add(-time.Hour), // Expired 1 hour ago
		}

		err := repo.Create(ctx, expiredSession)
		require.NoError(t, err)

		_, err = repo.GetByUserID(ctx, "user456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("DeleteByUserID", func(t *testing.T) {
		repo := NewMemoryGitHubAuthSessionRepository()
		ctx := context.Background()

		session := &domain.GitHubAuthSession{
			ID:          "delete-test",
			UserID:      "user789",
			AccessToken: "token_to_delete",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := repo.Create(ctx, session)
		require.NoError(t, err)

		err = repo.DeleteByUserID(ctx, "user789")
		require.NoError(t, err)

		_, err = repo.GetByUserID(ctx, "user789")
		assert.Error(t, err)
	})

	t.Run("DeleteExpiredSessions", func(t *testing.T) {
		repo := NewMemoryGitHubAuthSessionRepository()
		ctx := context.Background()

		// Create an expired session
		expiredSession := &domain.GitHubAuthSession{
			ID:          "expired",
			UserID:      "expired_user",
			AccessToken: "expired_token",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			ExpiresAt:   time.Now().Add(-time.Hour),
		}

		// Create a valid session
		validSession := &domain.GitHubAuthSession{
			ID:          "valid",
			UserID:      "valid_user",
			AccessToken: "valid_token",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := repo.Create(ctx, expiredSession)
		require.NoError(t, err)
		err = repo.Create(ctx, validSession)
		require.NoError(t, err)

		err = repo.DeleteExpired(ctx)
		require.NoError(t, err)

		// Expired session should be gone
		_, err = repo.GetByUserID(ctx, "expired_user")
		assert.Error(t, err)

		// Valid session should still exist
		retrieved, err := repo.GetByUserID(ctx, "valid_user")
		require.NoError(t, err)
		assert.Equal(t, "valid", retrieved.ID)
	})

	t.Run("ReplaceExistingSession", func(t *testing.T) {
		repo := NewMemoryGitHubAuthSessionRepository()
		ctx := context.Background()

		// Create first session
		session1 := &domain.GitHubAuthSession{
			ID:          "session-1",
			UserID:      "user999",
			AccessToken: "token_1",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := repo.Create(ctx, session1)
		require.NoError(t, err)

		// Create second session for same user
		session2 := &domain.GitHubAuthSession{
			ID:          "session-2",
			UserID:      "user999",
			AccessToken: "token_2",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err = repo.Create(ctx, session2)
		require.NoError(t, err)

		// Should get the second session (replaced the first)
		retrieved, err := repo.GetByUserID(ctx, "user999")
		require.NoError(t, err)
		assert.Equal(t, "session-2", retrieved.ID)
		assert.Equal(t, "token_2", retrieved.AccessToken)
	})
}
