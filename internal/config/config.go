// Package config provides application configuration management following SOLID principles.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Environment constants
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// Config defines the application configuration interface.
// Following Interface Segregation Principle.
type Config interface {
	GetServerPort() string
	GetDatabaseURL() string
	GetJWTSecret() string
	GetEnvironment() string
	GetLogLevel() string
	IsProduction() bool
}

// ServerConfig interface for server-specific configuration.
type ServerConfig interface {
	GetServerPort() string
	GetReadTimeout() time.Duration
	GetWriteTimeout() time.Duration
	GetIdleTimeout() time.Duration
}

// DatabaseConfig interface for database-specific configuration.
type DatabaseConfig interface {
	GetDatabaseURL() string
	GetMaxConnections() int
	GetConnectionTimeout() time.Duration
}

// SecurityConfig interface for security-related configuration.
type SecurityConfig interface {
	GetJWTSecret() string
	GetJWTExpiration() time.Duration
	GetRefreshTokenExpiration() time.Duration
	GetPasswordResetSecret() string
}

// RateLimitConfig interface for rate limiting configuration.
type RateLimitConfig interface {
	GetRateLimitEnabled() bool
	GetRateLimitRequestsPerMinute() int
	GetRateLimitCacheCapacity() int
	GetRedisEnabled() bool
	GetRedisAddr() string
	GetRedisPassword() string
	GetRedisDB() int
}

// GitHubConfig interface for GitHub integration configuration.
type GitHubConfig interface {
	GetGitHubClientID() string
	GetGitHubClientSecret() string
	GetGitHubRedirectURL() string
	GetGitHubWebhookSecret() string
}

// AppConfig implements all configuration interfaces.
type AppConfig struct {
	serverPort                 string
	databaseURL                string
	jwtSecret                  string
	passwordResetSecret        string
	environment                string
	logLevel                   string
	redisAddr                  string
	redisPassword              string
	githubClientID             string
	githubClientSecret         string
	githubRedirectURL          string
	githubWebhookSecret        string
	readTimeout                time.Duration
	writeTimeout               time.Duration
	idleTimeout                time.Duration
	connectionTimeout          time.Duration
	jwtExpiration              time.Duration
	refreshTokenExpiration     time.Duration
	maxConnections             int
	rateLimitRequestsPerMinute int
	rateLimitCacheCapacity     int
	redisDB                    int
	rateLimitEnabled           bool
	redisEnabled               bool
}

// NewConfig creates a new configuration instance with default values
// and overrides from environment variables.
func NewConfig() *AppConfig {
	environment := getEnvString("ENVIRONMENT", EnvDevelopment)
	jwtSecret := getJWTSecret(environment)

	return &AppConfig{
		serverPort:                 getEnvString("SERVER_PORT", "8080"),
		databaseURL:                getEnvString("DATABASE_URL", "pb_data/database.db"),
		jwtSecret:                  jwtSecret,
		passwordResetSecret:        getPasswordResetSecret(environment),
		environment:                environment,
		logLevel:                   getEnvString("LOG_LEVEL", "info"),
		githubClientID:             getEnvString("GITHUB_CLIENT_ID", ""),
		githubClientSecret:         getEnvString("GITHUB_CLIENT_SECRET", ""),
		githubRedirectURL:          getEnvString("GITHUB_REDIRECT_URL", "http://localhost:8090/api/v1/github/callback"),
		githubWebhookSecret:        getEnvString("GITHUB_WEBHOOK_SECRET", ""),
		readTimeout:                getEnvDuration("READ_TIMEOUT", "15s"),
		writeTimeout:               getEnvDuration("WRITE_TIMEOUT", "15s"),
		idleTimeout:                getEnvDuration("IDLE_TIMEOUT", "60s"),
		maxConnections:             getEnvInt("MAX_CONNECTIONS", 25),
		connectionTimeout:          getEnvDuration("CONNECTION_TIMEOUT", "30s"),
		jwtExpiration:              getEnvDuration("JWT_EXPIRATION", "24h"),
		refreshTokenExpiration:     getEnvDuration("REFRESH_TOKEN_EXPIRATION", "168h"), // 7 days
		rateLimitEnabled:           getEnvBool("RATE_LIMIT_ENABLED", true),
		rateLimitRequestsPerMinute: getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
		rateLimitCacheCapacity:     getEnvInt("RATE_LIMIT_CACHE_CAPACITY", 10000),
		redisEnabled:               getEnvBool("REDIS_ENABLED", false),
		redisAddr:                  getEnvString("REDIS_ADDR", "localhost:6379"),
		redisPassword:              getEnvString("REDIS_PASSWORD", ""),
		redisDB:                    getEnvInt("REDIS_DB", 0),
	}
}

// GetServerPort returns the server port configuration.
func (c *AppConfig) GetServerPort() string {
	return c.serverPort
}

// GetDatabaseURL returns the database URL configuration.
func (c *AppConfig) GetDatabaseURL() string {
	return c.databaseURL
}

// GetJWTSecret returns the JWT secret configuration.
func (c *AppConfig) GetJWTSecret() string {
	return c.jwtSecret
}

// GetEnvironment returns the application environment configuration.
func (c *AppConfig) GetEnvironment() string {
	return c.environment
}

// GetLogLevel returns the log level configuration.
func (c *AppConfig) GetLogLevel() string {
	return c.logLevel
}

// IsProduction returns true if the application is running in production environment.
func (c *AppConfig) IsProduction() bool {
	return c.environment == EnvProduction
}

// GetReadTimeout returns the server read timeout configuration.
func (c *AppConfig) GetReadTimeout() time.Duration {
	return c.readTimeout
}

// GetWriteTimeout returns the server write timeout configuration.
func (c *AppConfig) GetWriteTimeout() time.Duration {
	return c.writeTimeout
}

// GetIdleTimeout returns the server idle timeout configuration.
func (c *AppConfig) GetIdleTimeout() time.Duration {
	return c.idleTimeout
}

// GetMaxConnections returns the maximum database connections configuration.
func (c *AppConfig) GetMaxConnections() int {
	return c.maxConnections
}

// GetConnectionTimeout returns the database connection timeout configuration.
func (c *AppConfig) GetConnectionTimeout() time.Duration {
	return c.connectionTimeout
}

// GetJWTExpiration returns the JWT token expiration time configuration.
func (c *AppConfig) GetJWTExpiration() time.Duration {
	return c.jwtExpiration
}

// GetRefreshTokenExpiration returns the refresh token expiration time configuration.
func (c *AppConfig) GetRefreshTokenExpiration() time.Duration {
	return c.refreshTokenExpiration
}

// GetPasswordResetSecret returns the password reset token secret configuration.
func (c *AppConfig) GetPasswordResetSecret() string {
	return c.passwordResetSecret
}

// GetRateLimitEnabled returns whether rate limiting is enabled.
func (c *AppConfig) GetRateLimitEnabled() bool {
	return c.rateLimitEnabled
}

// GetRateLimitRequestsPerMinute returns the rate limit requests per minute.
func (c *AppConfig) GetRateLimitRequestsPerMinute() int {
	return c.rateLimitRequestsPerMinute
}

// GetRateLimitCacheCapacity returns the rate limiter cache capacity.
func (c *AppConfig) GetRateLimitCacheCapacity() int {
	return c.rateLimitCacheCapacity
}

// GetRedisEnabled returns whether Redis is enabled.
func (c *AppConfig) GetRedisEnabled() bool {
	return c.redisEnabled
}

// GetRedisAddr returns the Redis address.
func (c *AppConfig) GetRedisAddr() string {
	return c.redisAddr
}

// GetRedisPassword returns the Redis password.
func (c *AppConfig) GetRedisPassword() string {
	return c.redisPassword
}

// GetRedisDB returns the Redis database number.
func (c *AppConfig) GetRedisDB() int {
	return c.redisDB
}

// GetGitHubClientID returns the GitHub OAuth client ID.
func (c *AppConfig) GetGitHubClientID() string {
	return c.githubClientID
}

// GetGitHubClientSecret returns the GitHub OAuth client secret.
func (c *AppConfig) GetGitHubClientSecret() string {
	return c.githubClientSecret
}

// GetGitHubRedirectURL returns the GitHub OAuth redirect URL.
func (c *AppConfig) GetGitHubRedirectURL() string {
	return c.githubRedirectURL
}

// GetGitHubWebhookSecret returns the GitHub webhook secret.
func (c *AppConfig) GetGitHubWebhookSecret() string {
	return c.githubWebhookSecret
}

// Validate checks if the configuration is valid.
func (c *AppConfig) Validate() error {
	if err := c.validateBasicConfig(); err != nil {
		return err
	}
	if err := c.validateSecurityConfig(); err != nil {
		return err
	}
	if err := c.validateRedisConfig(); err != nil {
		return err
	}
	if err := c.validateRateLimitConfig(); err != nil {
		return err
	}
	return nil
}

// validateBasicConfig validates basic application configuration.
func (c *AppConfig) validateBasicConfig() error {
	if c.serverPort == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	if c.environment != EnvDevelopment && c.environment != EnvStaging && c.environment != EnvProduction {
		return fmt.Errorf("environment must be one of: %s, %s, %s", EnvDevelopment, EnvStaging, EnvProduction)
	}
	return nil
}

// validateSecurityConfig validates security-related configuration.
func (c *AppConfig) validateSecurityConfig() error {
	if err := c.validateJWTSecret(); err != nil {
		return err
	}
	return c.validatePasswordResetSecret()
}

// validateJWTSecret validates JWT secret configuration.
func (c *AppConfig) validateJWTSecret() error {
	if c.jwtSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}
	if len(c.jwtSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}
	if c.IsProduction() && isDefaultSecret(c.jwtSecret) {
		return fmt.Errorf("production environments cannot use default JWT secrets - set JWT_SECRET environment variable")
	}
	return nil
}

// validatePasswordResetSecret validates password reset secret configuration.
func (c *AppConfig) validatePasswordResetSecret() error {
	if c.passwordResetSecret == "" {
		return fmt.Errorf("password reset secret cannot be empty")
	}
	if len(c.passwordResetSecret) < 32 {
		return fmt.Errorf("password reset secret must be at least 32 characters long")
	}
	if c.IsProduction() && isDefaultSecret(c.passwordResetSecret) {
		return fmt.Errorf("production environments cannot use default password reset secrets - " +
			"set PASSWORD_RESET_SECRET environment variable")
	}
	return nil
}

// validateRedisConfig validates Redis configuration.
func (c *AppConfig) validateRedisConfig() error {
	if c.redisEnabled && c.redisAddr == "" {
		return fmt.Errorf("redis address cannot be empty when Redis is enabled")
	}
	return nil
}

// validateRateLimitConfig validates rate limiting configuration.
func (c *AppConfig) validateRateLimitConfig() error {
	if c.rateLimitRequestsPerMinute <= 0 {
		return fmt.Errorf("rate limit requests per minute must be positive")
	}
	if c.rateLimitCacheCapacity <= 0 {
		return fmt.Errorf("rate limit cache capacity must be positive")
	}
	return nil
}

// Helper functions for environment variable parsing.
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return time.Second
}

// getJWTSecret gets the JWT secret with proper security validation.
func getJWTSecret(environment string) string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}

	// In production, require JWT_SECRET to be explicitly set
	if environment == EnvProduction {
		panic("JWT_SECRET environment variable is required in production")
	}

	// For non-production environments, generate a cryptographically secure random secret
	return generateSecureJWTSecret()
}

// generateSecureJWTSecret generates a cryptographically secure random JWT secret.
func generateSecureJWTSecret() string {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate secure JWT secret: %v", err))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// getPasswordResetSecret gets the password reset secret with proper security validation.
func getPasswordResetSecret(environment string) string {
	if secret := os.Getenv("PASSWORD_RESET_SECRET"); secret != "" {
		return secret
	}

	// In production, require PASSWORD_RESET_SECRET to be explicitly set
	if environment == EnvProduction {
		panic("PASSWORD_RESET_SECRET environment variable is required in production")
	}

	// For non-production environments, generate a cryptographically secure random secret
	return generateSecurePasswordResetSecret()
}

// generateSecurePasswordResetSecret generates a cryptographically secure random password reset secret.
func generateSecurePasswordResetSecret() string {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate secure password reset secret: %v", err))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// isDefaultSecret checks if a secret is a known default/predictable value.
func isDefaultSecret(secret string) bool {
	defaultSecrets := []string{
		"simple-easy-tasks-development-jwt-secret-key-32chars-minimum-length-required",
		"secret",
		"jwt-secret",
		"your-secret-key",
		"default-secret",
		"development-secret",
		"test-secret",
		// Common .env.example patterns
		"your-super-secret-jwt-key-with-at-least-32-characters",
		"your-super-secret-password-reset-key-with-at-least-32-characters",
		"your-super-secret",
		"super-secret",
		"super-secret-key",
		"super-secret-jwt-key",
		"changeme",
		"changeme123",
		"placeholder",
		"example-secret",
		"example-key",
		"sample-secret",
		"sample-key",
	}

	for _, defaultSecret := range defaultSecrets {
		if secret == defaultSecret {
			return true
		}
	}

	return false
}
