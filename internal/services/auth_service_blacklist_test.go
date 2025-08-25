package services

import (
	"context"
	"testing"
	"time"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/testutil"
)

// Test config implementation
type testConfig struct {
	jwtSecret         string
	jwtExpiration     time.Duration
	refreshExpiration time.Duration
}

func (t *testConfig) GetJWTSecret() string {
	return t.jwtSecret
}

func (t *testConfig) GetJWTExpiration() time.Duration {
	return t.jwtExpiration
}

func (t *testConfig) GetRefreshTokenExpiration() time.Duration {
	if t.refreshExpiration == 0 {
		return time.Hour * 24 * 7 // 7 days
	}
	return t.refreshExpiration
}

// Mock implementation of TokenBlacklistRepository for testing
type mockTokenBlacklistRepository struct {
	blacklistedTokens map[string]*domain.BlacklistedToken
}

func newMockTokenBlacklistRepository() *mockTokenBlacklistRepository {
	return &mockTokenBlacklistRepository{
		blacklistedTokens: make(map[string]*domain.BlacklistedToken),
	}
}

func (m *mockTokenBlacklistRepository) BlacklistToken(_ context.Context, token *domain.BlacklistedToken) error {
	m.blacklistedTokens[token.TokenID] = token
	return nil
}

func (m *mockTokenBlacklistRepository) IsTokenBlacklisted(_ context.Context, tokenID string) (bool, error) {
	_, exists := m.blacklistedTokens[tokenID]
	return exists, nil
}

func (m *mockTokenBlacklistRepository) CleanupExpiredTokens(_ context.Context) error {
	now := time.Now()
	for tokenID, token := range m.blacklistedTokens {
		if token.ExpiresAt.Before(now) {
			delete(m.blacklistedTokens, tokenID)
		}
	}
	return nil
}

func (m *mockTokenBlacklistRepository) BlacklistAllUserTokens(
	_ context.Context,
	userID string,
	expiryTime time.Time,
) error {
	m.blacklistedTokens["USER_ALL_TOKENS_"+userID] = &domain.BlacklistedToken{
		TokenID:   "USER_ALL_TOKENS_" + userID,
		UserID:    userID,
		ExpiresAt: expiryTime,
		CreatedAt: time.Now(),
	}
	return nil
}

// Mock implementation of PasswordResetTokenRepository for testing
type mockPasswordResetTokenRepository struct {
	tokens map[string]*domain.PasswordResetToken
}

func newMockPasswordResetTokenRepository() *mockPasswordResetTokenRepository {
	return &mockPasswordResetTokenRepository{
		tokens: make(map[string]*domain.PasswordResetToken),
	}
}

func (m *mockPasswordResetTokenRepository) Create(_ context.Context, token *domain.PasswordResetToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockPasswordResetTokenRepository) GetByToken(
	_ context.Context,
	tokenValue string,
) (*domain.PasswordResetToken, error) {
	token, exists := m.tokens[tokenValue]
	if !exists {
		return nil, domain.NewNotFoundError("TOKEN_NOT_FOUND", "Password reset token not found")
	}
	return token, nil
}

func (m *mockPasswordResetTokenRepository) Update(_ context.Context, token *domain.PasswordResetToken) error {
	if _, exists := m.tokens[token.Token]; !exists {
		return domain.NewNotFoundError("TOKEN_NOT_FOUND", "Password reset token not found")
	}
	m.tokens[token.Token] = token
	return nil
}

func (m *mockPasswordResetTokenRepository) Delete(_ context.Context, tokenID string) error {
	for tokenValue, token := range m.tokens {
		if token.ID == tokenID {
			delete(m.tokens, tokenValue)
			return nil
		}
	}
	return nil
}

func (m *mockPasswordResetTokenRepository) CleanupExpiredTokens(_ context.Context) error {
	now := time.Now()
	for tokenValue, token := range m.tokens {
		if token.ExpiresAt.Before(now) {
			delete(m.tokens, tokenValue)
		}
	}
	return nil
}

func (m *mockPasswordResetTokenRepository) InvalidateUserTokens(_ context.Context, userID string) error {
	for _, token := range m.tokens {
		if token.UserID == userID && token.IsValid() {
			token.MarkAsUsed()
		}
	}
	return nil
}

func TestAuthService_Logout(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, newMockPasswordResetTokenRepository(), cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:           "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Role:         domain.RegularUserRole,
		TokenVersion: 1,
	}
	userRepo.AddUser(user)

	// Generate a token pair
	tokenPair, err := authService.generateTokenPair(user)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Test logout
	err = authService.Logout(context.Background(), tokenPair.AccessToken)
	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}

	// Verify token was blacklisted
	claims, err := authService.parseToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	isBlacklisted, err := blacklistRepo.IsTokenBlacklisted(context.Background(), claims.ID)
	if err != nil {
		t.Errorf("Failed to check blacklist: %v", err)
	}
	if !isBlacklisted {
		t.Error("Token should be blacklisted after logout")
	}
}

func TestAuthService_ValidateToken_BlacklistedToken(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, newMockPasswordResetTokenRepository(), cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:           "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Role:         domain.RegularUserRole,
		TokenVersion: 1,
	}
	userRepo.AddUser(user)

	// Generate a token pair
	tokenPair, err := authService.generateTokenPair(user)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Blacklist the token
	err = authService.Logout(context.Background(), tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to logout: %v", err)
	}

	// Try to validate the blacklisted token
	_, err = authService.ValidateToken(context.Background(), tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected validation to fail for blacklisted token")
	}

	if domainErr, ok := err.(*domain.Error); ok {
		if domainErr.Code != "TOKEN_BLACKLISTED" {
			t.Errorf("Expected TOKEN_BLACKLISTED error, got: %s", domainErr.Code)
		}
	} else {
		t.Errorf("Expected domain error, got: %v", err)
	}
}

func TestAuthService_ValidateToken_OutdatedTokenVersion(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, newMockPasswordResetTokenRepository(), cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:           "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Role:         domain.RegularUserRole,
		TokenVersion: 1,
	}
	userRepo.AddUser(user)

	// Generate a token pair
	tokenPair, err := authService.generateTokenPair(user)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Increment user token version (simulating token invalidation)
	user.IncrementTokenVersion()

	// Try to validate the token with old version
	_, err = authService.ValidateToken(context.Background(), tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected validation to fail for outdated token version")
	}

	if domainErr, ok := err.(*domain.Error); ok {
		if domainErr.Code != "TOKEN_OUTDATED" {
			t.Errorf("Expected TOKEN_OUTDATED error, got: %s", domainErr.Code)
		}
	} else {
		t.Errorf("Expected domain error, got: %v", err)
	}
}

func TestAuthService_InvalidateAllUserTokens(t *testing.T) {
	// Setup
	userRepo := testutil.NewMockUserRepository()
	blacklistRepo := newMockTokenBlacklistRepository()
	cfg := &testConfig{jwtSecret: "test-secret-that-is-32-characters-long", jwtExpiration: time.Hour}
	authService := NewAuthService(userRepo, blacklistRepo, newMockPasswordResetTokenRepository(), cfg).(*authService)

	// Create a test user
	user := &domain.User{
		ID:           "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Role:         domain.RegularUserRole,
		TokenVersion: 1,
	}
	userRepo.AddUser(user)

	// Generate a token pair
	tokenPair, err := authService.generateTokenPair(user)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Invalidate all user tokens
	err = authService.InvalidateAllUserTokens(context.Background(), user.ID)
	if err != nil {
		t.Errorf("Failed to invalidate all user tokens: %v", err)
	}

	// Check that user token version was incremented
	updatedUser, err := userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}
	if updatedUser.TokenVersion != 2 {
		t.Errorf("Expected token version 2, got %d", updatedUser.TokenVersion)
	}

	// Check that blacklist entry was created
	isBlacklisted, err := blacklistRepo.IsTokenBlacklisted(context.Background(), "USER_ALL_TOKENS_"+user.ID)
	if err != nil {
		t.Errorf("Failed to check blacklist: %v", err)
	}
	if !isBlacklisted {
		t.Error("Expected user token blacklist entry to exist")
	}

	// Try to validate the old token
	_, err = authService.ValidateToken(context.Background(), tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected validation to fail for invalidated token")
	}
}
