package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"simple-easy-tasks/internal/config"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
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
}

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// authService implements AuthService interface.
type authService struct {
	userRepo  repository.UserRepository
	config    config.SecurityConfig
	jwtSecret []byte
}

// NewAuthService creates a new authentication service.
func NewAuthService(userRepo repository.UserRepository, cfg config.SecurityConfig) AuthService {
	return &authService{
		userRepo:  userRepo,
		config:    cfg,
		jwtSecret: []byte(cfg.GetJWTSecret()),
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
	if err := user.CheckPassword(req.Password); err != nil {
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

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.NewAuthenticationError("USER_NOT_FOUND", "User not found")
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// Logout invalidates tokens (placeholder for future blacklist implementation).
func (s *authService) Logout(ctx context.Context, userID string) error {
	// TODO: Implement token blacklist or invalidation mechanism
	// For now, we rely on token expiration
	return nil
}

// generateTokenPair creates both access and refresh tokens.
func (s *authService) generateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.config.GetJWTExpiration())
	refreshExpiry := now.Add(s.config.GetRefreshTokenExpiration())

	// Create access token claims
	accessClaims := &TokenClaims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     string(user.Role),
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
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     string(user.Role),
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
