package domain

import (
	"context"
	"time"
)

// BlacklistedToken represents a blacklisted JWT token.
type BlacklistedToken struct {
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
}

// TokenBlacklistRepository defines the interface for token blacklist operations.
type TokenBlacklistRepository interface {
	// BlacklistToken adds a token to the blacklist
	BlacklistToken(ctx context.Context, token *BlacklistedToken) error

	// IsTokenBlacklisted checks if a token is blacklisted
	IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error)

	// CleanupExpiredTokens removes expired blacklisted tokens
	CleanupExpiredTokens(ctx context.Context) error

	// BlacklistAllUserTokens blacklists all tokens for a specific user
	BlacklistAllUserTokens(ctx context.Context, userID string, expiryTime time.Time) error
}
