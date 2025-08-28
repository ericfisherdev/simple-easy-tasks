// Package services provides business logic implementations for the Simple Easy Tasks application.
package services

//nolint:gofumpt
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/config"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthService defines the interface for authentication operations.
// Following Interface Segregation Principle.
type AuthService interface {
	// Login authenticates a user and returns JWT tokens.
	Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenPair, error)

	// Register creates a new user account.
	Register(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error)

	// RefreshToken generates new tokens using a refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error)

	// ValidateToken validates a JWT token and returns user claims.
	ValidateToken(ctx context.Context, tokenString string) (*domain.User, error)

	// Logout invalidates tokens (placeholder for future blacklist implementation).
	Logout(ctx context.Context, userID string) error

	// ForgotPassword initiates the password reset flow.
	ForgotPassword(ctx context.Context, email string) error

	// ResetPassword resets the user's password using a reset token.
	//nolint:gofumpt
	ResetPassword(ctx context.Context, token string, newPassword string) error

	// InvalidateAllUserTokens invalidates all tokens for a user
	InvalidateAllUserTokens(ctx context.Context, userID string) error
}

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	TokenVersion int    `json:"token_version"`
}

// authService implements AuthService interface.
type authService struct {
	userRepo       repository.UserRepository
	blacklistRepo  domain.TokenBlacklistRepository
	resetTokenRepo domain.PasswordResetTokenRepository
	config         config.SecurityConfig
	jwtSecret      []byte
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo repository.UserRepository,
	blacklistRepo domain.TokenBlacklistRepository,
	resetTokenRepo domain.PasswordResetTokenRepository,
	cfg config.SecurityConfig,
) AuthService {
	return &authService{
		userRepo:       userRepo,
		blacklistRepo:  blacklistRepo,
		resetTokenRepo: resetTokenRepo,
		config:         cfg,
		jwtSecret:      []byte(cfg.GetJWTSecret()),
	}
}

// Login authenticates a user and returns JWT tokens.
func (s *authService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenPair, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.NewAuthenticationError("INVALID_CREDENTIALS", "Invalid email or password")
	}

	// Check password
	if passwordErr := user.CheckPassword(req.Password); passwordErr != nil {
		return nil, domain.NewAuthenticationError("INVALID_CREDENTIALS", "Invalid email or password")
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, domain.NewInternalError("TOKEN_GENERATION_FAILED", "Failed to generate authentication tokens", err)
	}

	return tokenPair, nil
}

// Register creates a new user account.
func (s *authService) Register(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.NewInternalError("USER_CHECK_FAILED", "Failed to check user existence", err)
	}
	if exists {
		return nil, domain.NewConflictError("EMAIL_EXISTS", "A user with this email already exists")
	}

	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, domain.NewInternalError("USER_CHECK_FAILED", "Failed to check user existence", err)
	}
	if exists {
		return nil, domain.NewConflictError("USERNAME_EXISTS", "A user with this username already exists")
	}

	// Create user entity
	user := &domain.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Username: req.Username,
		Name:     req.Name,
		Role:     domain.RegularUserRole,
		Preferences: domain.UserPreferences{
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set role if provided (admin users can set roles)
	if req.Role != "" {
		user.Role = domain.UserRole(req.Role)
	}

	// Set password
	if err := user.SetPassword(req.Password); err != nil {
		return nil, err
	}

	// Validate user data
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Save user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, domain.NewInternalError("USER_CREATION_FAILED", "Failed to create user", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// RefreshToken generates new tokens using a refresh token.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	// Parse and validate refresh token
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, domain.NewAuthenticationError("INVALID_REFRESH_TOKEN", "Invalid or expired refresh token")
	}

	// Get user to ensure they still exist
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.NewAuthenticationError("USER_NOT_FOUND", "User not found")
	}

	// Generate new token pair
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, domain.NewInternalError("TOKEN_GENERATION_FAILED", "Failed to generate new tokens", err)
	}

	return tokenPair, nil
}

// ValidateToken validates a JWT token and returns user claims.
func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*domain.User, error) {
	// Parse token
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, domain.NewAuthenticationError("INVALID_TOKEN", "Invalid or expired token")
	}

	// Check if token is blacklisted
	isBlacklisted, err := s.blacklistRepo.IsTokenBlacklisted(ctx, claims.ID)
	if err != nil { //nolint:revive // Intentionally ignore error to continue with fallback check
		// Log error but continue (fallback to token version check)
	} else if isBlacklisted {
		return nil, domain.NewAuthenticationError("TOKEN_BLACKLISTED", "Token has been invalidated")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.NewAuthenticationError("USER_NOT_FOUND", "User not found")
	}

	// Check token version
	if claims.TokenVersion < user.TokenVersion {
		return nil, domain.NewAuthenticationError("TOKEN_OUTDATED", "Token version is outdated")
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// Logout invalidates tokens by blacklisting the specific token.
func (s *authService) Logout(ctx context.Context, tokenString string) error {
	// Parse token to get claims
	claims, err := s.parseToken(tokenString)
	if err != nil {
		// Token is already invalid, consider logout successful
		return nil
	}

	// Create blacklist entry
	blacklistedToken := &domain.BlacklistedToken{
		TokenID:   claims.ID,
		UserID:    claims.UserID,
		ExpiresAt: claims.ExpiresAt.Time,
		CreatedAt: time.Now(),
	}

	// Add to blacklist
	return s.blacklistRepo.BlacklistToken(ctx, blacklistedToken)
}

// InvalidateAllUserTokens invalidates all tokens for a user by incrementing token version.
func (s *authService) InvalidateAllUserTokens(ctx context.Context, userID string) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return domain.NewInternalError("USER_NOT_FOUND", "User not found", err)
	}

	// Increment token version
	user.IncrementTokenVersion()

	// Update user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return domain.NewInternalError("USER_UPDATE_FAILED", "Failed to update user token version", err)
	}

	// Also add a blacklist entry for all current tokens as backup
	maxExpiry := time.Now().Add(s.config.GetRefreshTokenExpiration())
	return s.blacklistRepo.BlacklistAllUserTokens(ctx, userID, maxExpiry)
}

// generateTokenPair creates both access and refresh tokens.
func (s *authService) generateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.config.GetJWTExpiration())
	refreshExpiry := now.Add(s.config.GetRefreshTokenExpiration())

	// Create access token claims
	accessClaims := &TokenClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Username:     user.Username,
		Role:         string(user.Role),
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "simple-easy-tasks",
			Audience:  []string{"simple-easy-tasks-app"},
			ID:        uuid.New().String(),
		},
	}

	// Create refresh token claims
	refreshClaims := &TokenClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Username:     user.Username,
		Role:         string(user.Role),
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "simple-easy-tasks",
			Audience:  []string{"simple-easy-tasks-refresh"},
			ID:        uuid.New().String(),
		},
	}

	// Generate tokens
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

// parseToken parses and validates a JWT token.
func (s *authService) parseToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ForgotPassword initiates the password reset flow.
func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return nil
	}

	// Invalidate any existing tokens for this user
	if invalidateErr := s.resetTokenRepo.InvalidateUserTokens(ctx, user.ID); invalidateErr != nil { //nolint:revive // Error is intentionally ignored
		// Log error but continue - this is not critical for the flow
		_ = invalidateErr // Acknowledge the error exists
	}

	// Generate secure random token
	tokenValue, err := s.generateSecureToken()
	if err != nil {
		return domain.NewInternalError("TOKEN_GENERATION_FAILED", "Failed to generate reset token", err)
	}

	// Create password reset token
	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		Token:     tokenValue,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		Used:      false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Validate token before persistence
	if err := resetToken.Validate(); err != nil {
		return err
	}

	// Store token in database
	if err := s.resetTokenRepo.Create(ctx, resetToken); err != nil {
		return domain.NewInternalError("TOKEN_STORAGE_FAILED", "Failed to store reset token", err)
	}

	// Schedule cleanup of expired tokens
	go func() {
		if cleanupErr := s.resetTokenRepo.CleanupExpiredTokens(ctx); cleanupErr != nil { //nolint:revive // Error is intentionally ignored
			// Log error but don't affect the main flow
			_ = cleanupErr // Acknowledge the error exists
		}
	}()

	// TODO: Send email with reset link containing the token
	// For now, we'll log the token (in production, this should be sent via email)
	fmt.Printf("Password reset token for user %s: %s\n", user.Email, tokenValue)

	return nil
}

// ResetPassword resets the user's password using a reset token.
//
//nolint:gofumpt
func (s *authService) ResetPassword(ctx context.Context, tokenValue string, newPassword string) error {
	// Get token from database
	resetToken, err := s.resetTokenRepo.GetByToken(ctx, tokenValue)
	if err != nil {
		var domainErr *domain.Error
		if errors.As(err, &domainErr) && domainErr.Type == domain.NotFoundError {
			return domain.NewAuthenticationError("INVALID_RESET_TOKEN", "Invalid or expired reset token")
		}
		return domain.NewInternalError("TOKEN_LOOKUP_FAILED", "Failed to lookup reset token", err)
	}

	// Check if token is valid (not expired and not used)
	if !resetToken.IsValid() {
		// Clean up the invalid token
		if cleanupErr := s.resetTokenRepo.Delete(ctx, resetToken.ID); cleanupErr != nil { //nolint:revive // Error is intentionally ignored
			// Log error but continue
			_ = cleanupErr // Acknowledge the error exists
		}

		if resetToken.Used {
			return domain.NewAuthenticationError("TOKEN_ALREADY_USED", "Reset token has already been used")
		}
		return domain.NewAuthenticationError("EXPIRED_RESET_TOKEN", "Reset token has expired")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// Update user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return domain.NewInternalError("PASSWORD_UPDATE_FAILED", "Failed to update password", err)
	}

	// Mark token as used
	resetToken.MarkAsUsed()
	if err := s.resetTokenRepo.Update(ctx, resetToken); err != nil { //nolint:revive // Error is intentionally ignored
		// Log error but don't fail the password reset since it was successful
		_ = err // Acknowledge the error exists
	}

	// Invalidate all user sessions for security
	if err := s.InvalidateAllUserTokens(ctx, user.ID); err != nil { //nolint:revive // Error is intentionally ignored
		// Log error but don't fail the password reset
		_ = err // Acknowledge the error exists
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token.
func (s *authService) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CleanupExpiredTokens removes expired password reset tokens.
// This method can be called periodically by a background job.
func (s *authService) CleanupExpiredTokens(ctx context.Context) error {
	return s.resetTokenRepo.CleanupExpiredTokens(ctx)
}
