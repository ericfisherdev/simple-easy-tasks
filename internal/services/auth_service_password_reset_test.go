package services

import (
	"context"
	"testing"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/testutil"
)

func TestAuthService_ForgotPassword(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:       "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Role:     domain.RegularUserRole,
	}
	userRepo.AddUser(user)

	// Test forgot password
	err := authService.ForgotPassword(context.Background(), "test@example.com")
	if err != nil {
		t.Errorf("ForgotPassword failed: %v", err)
	}

	// Verify token was created
	if len(resetTokenRepo.tokens) != 1 {
		t.Errorf("Expected 1 reset token, got %d", len(resetTokenRepo.tokens))
	}

	// Get the created token
	var createdToken *domain.PasswordResetToken
	for _, token := range resetTokenRepo.tokens {
		createdToken = token
		break
	}

	if createdToken == nil {
		t.Fatal("No reset token was created")
	}

	// Verify token properties
	if createdToken.UserID != user.ID {
		t.Errorf("Expected UserID %s, got %s", user.ID, createdToken.UserID)
	}

	if createdToken.Used {
		t.Error("Token should not be marked as used initially")
	}

	if time.Until(createdToken.ExpiresAt) > time.Hour || time.Until(createdToken.ExpiresAt) < 59*time.Minute {
		t.Error("Token should expire in approximately 1 hour")
	}
}

func TestAuthService_ForgotPassword_NonExistentEmail(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Test forgot password with non-existent email
	err := authService.ForgotPassword(context.Background(), "nonexistent@example.com")
	// Should not return error for security (no email enumeration)
	if err != nil {
		t.Errorf("ForgotPassword should not return error for non-existent email: %v", err)
	}

	// Verify no token was created
	if len(resetTokenRepo.tokens) != 0 {
		t.Errorf("Expected 0 reset tokens, got %d", len(resetTokenRepo.tokens))
	}
}

func TestAuthService_ResetPassword(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:           "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Role:         domain.RegularUserRole,
		TokenVersion: 1,
	}
	err := user.SetPassword("oldpassword")
	if err != nil {
		t.Fatalf("Failed to set initial password: %v", err)
	}
	userRepo.AddUser(user)

	// Store the original token version for comparison
	originalTokenVersion := user.TokenVersion

	// Create a reset token
	resetToken := &domain.PasswordResetToken{
		ID:        "token123",
		Token:     "reset-token-value",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	resetTokenRepo.Create(context.Background(), resetToken)

	// Test password reset
	newPassword := "newpassword123"
	err = authService.ResetPassword(context.Background(), "reset-token-value", newPassword)
	if err != nil {
		t.Errorf("ResetPassword failed: %v", err)
	}

	// Verify password was changed
	updatedUser, err := userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	err = updatedUser.CheckPassword(newPassword)
	if err != nil {
		t.Error("New password should be valid")
	}

	err = updatedUser.CheckPassword("oldpassword")
	if err == nil {
		t.Error("Old password should no longer be valid")
	}

	// Verify token was marked as used
	updatedToken, err := resetTokenRepo.GetByToken(context.Background(), "reset-token-value")
	if err != nil {
		t.Fatalf("Failed to get updated token: %v", err)
	}

	if !updatedToken.Used {
		t.Error("Token should be marked as used after password reset")
	}

	// Verify user token version was incremented (all sessions invalidated)
	if updatedUser.TokenVersion <= originalTokenVersion {
		t.Errorf("User token version should be incremented after password reset. "+
			"Original: %d, Updated: %d", originalTokenVersion, updatedUser.TokenVersion)
	}
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Test password reset with invalid token
	err := authService.ResetPassword(context.Background(), "invalid-token", "newpassword")

	// Should return authentication error
	if err == nil {
		t.Error("ResetPassword should return error for invalid token")
	}

	authErr, ok := err.(*domain.Error)
	switch {
	case !ok:
		t.Errorf("Expected domain.Error, got %T", err)
	case authErr.Type != domain.AuthenticationError:
		t.Errorf("Expected AuthenticationError, got %s", authErr.Type)
	case authErr.Code != "INVALID_RESET_TOKEN":
		t.Errorf("Expected error code INVALID_RESET_TOKEN, got %s", authErr.Code)
	}
}

func TestAuthService_ResetPassword_ExpiredToken(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:       "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Role:     domain.RegularUserRole,
	}
	userRepo.AddUser(user)

	// Create an expired reset token
	resetToken := &domain.PasswordResetToken{
		ID:        "token123",
		Token:     "expired-token",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-time.Hour), // Expired 1 hour ago
		Used:      false,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	resetTokenRepo.Create(context.Background(), resetToken)

	// Test password reset with expired token
	err := authService.ResetPassword(context.Background(), "expired-token", "newpassword")

	// Should return authentication error
	if err == nil {
		t.Error("ResetPassword should return error for expired token")
	}

	authErr, ok := err.(*domain.Error)
	if !ok {
		t.Errorf("Expected AuthenticationError, got %T", err)
	} else if authErr.Code != "EXPIRED_RESET_TOKEN" {
		t.Errorf("Expected error code EXPIRED_RESET_TOKEN, got %s", authErr.Code)
	}
}

func TestAuthService_ResetPassword_UsedToken(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:       "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Role:     domain.RegularUserRole,
	}
	userRepo.AddUser(user)

	// Create a used reset token
	resetToken := &domain.PasswordResetToken{
		ID:        "token123",
		Token:     "used-token",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      true, // Already used
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	resetTokenRepo.Create(context.Background(), resetToken)

	// Test password reset with used token
	err := authService.ResetPassword(context.Background(), "used-token", "newpassword")

	// Should return authentication error
	if err == nil {
		t.Error("ResetPassword should return error for used token")
	}

	authErr, ok := err.(*domain.Error)
	if !ok {
		t.Errorf("Expected AuthenticationError, got %T", err)
	} else if authErr.Code != "TOKEN_ALREADY_USED" {
		t.Errorf("Expected error code TOKEN_ALREADY_USED, got %s", authErr.Code)
	}
}

func TestAuthService_CleanupExpiredTokens(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	resetTokenRepo := newMockPasswordResetTokenRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, resetTokenRepo, cfg).(*authService)

	// Create test tokens - some expired, some valid
	expiredToken1 := &domain.PasswordResetToken{
		ID:        "expired1",
		Token:     "expired-token-1",
		UserID:    "user1",
		ExpiresAt: time.Now().Add(-time.Hour),
		Used:      false,
	}
	expiredToken2 := &domain.PasswordResetToken{
		ID:        "expired2",
		Token:     "expired-token-2",
		UserID:    "user2",
		ExpiresAt: time.Now().Add(-time.Minute),
		Used:      false,
	}
	validToken := &domain.PasswordResetToken{
		ID:        "valid1",
		Token:     "valid-token",
		UserID:    "user3",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
	}

	resetTokenRepo.Create(context.Background(), expiredToken1)
	resetTokenRepo.Create(context.Background(), expiredToken2)
	resetTokenRepo.Create(context.Background(), validToken)

	// Verify initial state
	if len(resetTokenRepo.tokens) != 3 {
		t.Errorf("Expected 3 tokens initially, got %d", len(resetTokenRepo.tokens))
	}

	// Run cleanup
	err := authService.CleanupExpiredTokens(context.Background())
	if err != nil {
		t.Errorf("CleanupExpiredTokens failed: %v", err)
	}

	// Verify expired tokens were removed
	if len(resetTokenRepo.tokens) != 1 {
		t.Errorf("Expected 1 token after cleanup, got %d", len(resetTokenRepo.tokens))
	}

	// Verify the remaining token is the valid one
	if _, exists := resetTokenRepo.tokens["valid-token"]; !exists {
		t.Error("Valid token should still exist after cleanup")
	}
}
