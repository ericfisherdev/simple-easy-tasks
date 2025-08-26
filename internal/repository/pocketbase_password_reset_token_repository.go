// Package repository provides data access implementations for the Simple Easy Tasks application.
package repository

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/domain"
)

type pocketbasePasswordResetTokenRepository struct {
	app    core.App
	secret string
}

// NewPocketBasePasswordResetTokenRepository creates a new PocketBase password reset token repository.
func NewPocketBasePasswordResetTokenRepository(app core.App, secret string) domain.PasswordResetTokenRepository {
	return &pocketbasePasswordResetTokenRepository{
		app:    app,
		secret: secret,
	}
}

// hashResetToken computes HMAC-SHA256 hash of the raw token using the repository's secret.
func (r *pocketbasePasswordResetTokenRepository) hashResetToken(rawToken string) string {
	h := hmac.New(sha256.New, []byte(r.secret))
	h.Write([]byte(rawToken))
	return hex.EncodeToString(h.Sum(nil))
}

// Create stores a new password reset token.
func (r *pocketbasePasswordResetTokenRepository) Create(_ context.Context, token *domain.PasswordResetToken) error {
	collection, err := r.app.FindCollectionByNameOrId("password_reset_tokens")
	if err != nil {
		return domain.NewInternalError("COLLECTION_NOT_FOUND", "Password reset tokens collection not found", err)
	}

	record := core.NewRecord(collection)
	if token.ID == "" {
		token.ID = uuid.New().String()
	}

	record.Set("id", token.ID)
	record.Set("token", r.hashResetToken(token.Token))
	record.Set("user_id", token.UserID)
	record.Set("expires_at", token.ExpiresAt)
	record.Set("used", token.Used)

	if err := r.app.Save(record); err != nil {
		return domain.NewInternalError("TOKEN_SAVE_FAILED", "Failed to save password reset token", err)
	}

	// Update the token with the saved record's timestamps
	if createdTime := record.GetDateTime("created"); !createdTime.IsZero() {
		token.CreatedAt = createdTime.Time()
	}
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		token.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// GetByToken retrieves a token by its token value.
func (r *pocketbasePasswordResetTokenRepository) GetByToken(
	_ context.Context, tokenValue string,
) (*domain.PasswordResetToken, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"password_reset_tokens",
		"token = {:token}",
		dbx.Params{"token": r.hashResetToken(tokenValue)},
	)
	if err != nil {
		if IsNotFound(err) {
			return nil, domain.NewNotFoundError("TOKEN_NOT_FOUND", "Password reset token not found")
		}
		return nil, domain.NewInternalError("TOKEN_QUERY_FAILED", "Failed to query password reset token", err)
	}

	return r.recordToToken(record), nil
}

// Update updates a password reset token.
func (r *pocketbasePasswordResetTokenRepository) Update(_ context.Context, token *domain.PasswordResetToken) error {
	record, err := r.app.FindRecordById("password_reset_tokens", token.ID)
	if err != nil {
		return domain.NewNotFoundError("TOKEN_NOT_FOUND", "Password reset token not found")
	}

	// Only update fields that can actually change - token hash should never be updated
	record.Set("user_id", token.UserID)
	record.Set("expires_at", token.ExpiresAt)
	record.Set("used", token.Used)

	if err := r.app.Save(record); err != nil {
		return domain.NewInternalError("TOKEN_UPDATE_FAILED", "Failed to update password reset token", err)
	}

	// Update the token with the saved record's timestamps
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		token.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// Delete removes a password reset token.
func (r *pocketbasePasswordResetTokenRepository) Delete(_ context.Context, tokenID string) error {
	record, err := r.app.FindRecordById("password_reset_tokens", tokenID)
	if err != nil {
		if IsNotFound(err) {
			return nil // Already deleted
		}
		return domain.NewInternalError("TOKEN_QUERY_FAILED", "Failed to find password reset token", err)
	}

	if err := r.app.Delete(record); err != nil {
		return domain.NewInternalError("TOKEN_DELETE_FAILED", "Failed to delete password reset token", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens.
func (r *pocketbasePasswordResetTokenRepository) CleanupExpiredTokens(_ context.Context) error {
	records, err := r.app.FindRecordsByFilter(
		"password_reset_tokens",
		"expires_at <= {:now}",
		"",
		0,
		0,
		dbx.Params{
			"now": time.Now().UTC(),
		},
	)
	if err != nil {
		return domain.NewInternalError("CLEANUP_QUERY_FAILED", "Failed to query expired password reset tokens", err)
	}

	for _, record := range records {
		if err := r.app.Delete(record); err != nil {
			// Log error but continue cleanup
			continue
		}
	}

	return nil
}

// InvalidateUserTokens marks all tokens for a user as used.
func (r *pocketbasePasswordResetTokenRepository) InvalidateUserTokens(_ context.Context, userID string) error {
	records, err := r.app.FindRecordsByFilter(
		"password_reset_tokens",
		"user_id = {:userID} AND used = false AND expires_at > {:now}",
		"",
		0,
		0,
		dbx.Params{
			"userID": userID,
			"now":    time.Now().UTC(),
		},
	)
	if err != nil {
		return domain.NewInternalError("INVALIDATE_QUERY_FAILED", "Failed to query user password reset tokens", err)
	}

	for _, record := range records {
		record.Set("used", true)
		if err := r.app.Save(record); err != nil {
			// Log error but continue invalidation
			continue
		}
	}

	return nil
}

// recordToToken converts a PocketBase record to a domain PasswordResetToken.
func (r *pocketbasePasswordResetTokenRepository) recordToToken(record *core.Record) *domain.PasswordResetToken {
	return &domain.PasswordResetToken{
		ExpiresAt: record.GetDateTime("expires_at").Time(),
		CreatedAt: record.GetDateTime("created").Time(),
		UpdatedAt: record.GetDateTime("updated").Time(),
		ID:        record.Id,
		Token:     "", // Token hash is stored in DB, but we don't need the raw value for domain operations
		UserID:    record.GetString("user_id"),
		Used:      record.GetBool("used"),
	}
}
