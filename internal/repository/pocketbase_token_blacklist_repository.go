package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/domain"
)

type pocketbaseTokenBlacklistRepository struct {
	app core.App
}

// NewPocketBaseTokenBlacklistRepository creates a new PocketBase token blacklist repository.
func NewPocketBaseTokenBlacklistRepository(app core.App) domain.TokenBlacklistRepository {
	return &pocketbaseTokenBlacklistRepository{app: app}
}

// BlacklistToken adds a token to the blacklist.
func (r *pocketbaseTokenBlacklistRepository) BlacklistToken(_ context.Context, token *domain.BlacklistedToken) error {
	collection, err := r.app.FindCollectionByNameOrId("blacklisted_tokens")
	if err != nil {
		return domain.NewInternalError("COLLECTION_NOT_FOUND", "Blacklisted tokens collection not found", err)
	}

	record := core.NewRecord(collection)
	record.Set("token_id", token.TokenID)
	record.Set("user_id", token.UserID)
	record.Set("expires_at", token.ExpiresAt)

	if err := r.app.Save(record); err != nil {
		return domain.NewInternalError("BLACKLIST_SAVE_FAILED", "Failed to save blacklisted token", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted.
func (r *pocketbaseTokenBlacklistRepository) IsTokenBlacklisted(_ context.Context, tokenID string) (bool, error) {
	const sqlNoRowsError = "sql: no rows in result set"

	_, err := r.app.FindFirstRecordByFilter(
		"blacklisted_tokens",
		"token_id = {:tokenID} AND expires_at > {:now}",
		dbx.Params{
			"tokenID": tokenID,
			"now":     time.Now().Format("2006-01-02 15:04:05.000Z"),
		},
	)
	if err != nil {
		if err.Error() == sqlNoRowsError {
			return false, nil
		}
		return false, domain.NewInternalError("BLACKLIST_CHECK_FAILED", "Failed to check token blacklist", err)
	}

	return true, nil
}

// CleanupExpiredTokens removes expired blacklisted tokens.
func (r *pocketbaseTokenBlacklistRepository) CleanupExpiredTokens(_ context.Context) error {
	records, err := r.app.FindRecordsByFilter(
		"blacklisted_tokens",
		"expires_at <= {:now}",
		"",
		0,
		0,
		dbx.Params{
			"now": time.Now().Format("2006-01-02 15:04:05.000Z"),
		},
	)
	if err != nil {
		return domain.NewInternalError("CLEANUP_QUERY_FAILED", "Failed to query expired tokens", err)
	}

	for _, record := range records {
		if err := r.app.Delete(record); err != nil {
			// Log error but continue cleanup
			continue
		}
	}

	return nil
}

// BlacklistAllUserTokens blacklists all tokens for a specific user.
func (r *pocketbaseTokenBlacklistRepository) BlacklistAllUserTokens(
	_ context.Context,
	userID string,
	expiryTime time.Time,
) error {
	// Create a single blacklist entry for all user tokens with a special token_id
	collection, err := r.app.FindCollectionByNameOrId("blacklisted_tokens")
	if err != nil {
		return domain.NewInternalError("COLLECTION_NOT_FOUND", "Blacklisted tokens collection not found", err)
	}

	record := core.NewRecord(collection)
	record.Set("token_id", fmt.Sprintf("USER_ALL_TOKENS_%s", userID))
	record.Set("user_id", userID)
	record.Set("expires_at", expiryTime)

	if err := r.app.Save(record); err != nil {
		return domain.NewInternalError("BLACKLIST_SAVE_FAILED", "Failed to save user token blacklist", err)
	}

	return nil
}
