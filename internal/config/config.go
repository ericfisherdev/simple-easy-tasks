// Package config provides application configuration management following SOLID principles.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
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
}

// AppConfig implements all configuration interfaces.
type AppConfig struct {
	serverPort             string
	databaseURL            string
	jwtSecret              string
	environment            string
	logLevel               string
	readTimeout            time.Duration
	writeTimeout           time.Duration
	idleTimeout            time.Duration
	maxConnections         int
	connectionTimeout      time.Duration
	jwtExpiration          time.Duration
	refreshTokenExpiration time.Duration
}

// NewConfig creates a new configuration instance with default values
// and overrides from environment variables.
func NewConfig() *AppConfig {
	return &AppConfig{
		serverPort:             getEnvString("SERVER_PORT", "8080"),
		databaseURL:            getEnvString("DATABASE_URL", "pb_data/database.db"),
		jwtSecret:              getEnvString("JWT_SECRET", generateDefaultJWTSecret()),
		environment:            getEnvString("ENVIRONMENT", "development"),
		logLevel:               getEnvString("LOG_LEVEL", "info"),
		readTimeout:            getEnvDuration("READ_TIMEOUT", "15s"),
		writeTimeout:           getEnvDuration("WRITE_TIMEOUT", "15s"),
		idleTimeout:            getEnvDuration("IDLE_TIMEOUT", "60s"),
		maxConnections:         getEnvInt("MAX_CONNECTIONS", 25),
		connectionTimeout:      getEnvDuration("CONNECTION_TIMEOUT", "30s"),
		jwtExpiration:          getEnvDuration("JWT_EXPIRATION", "24h"),
		refreshTokenExpiration: getEnvDuration("REFRESH_TOKEN_EXPIRATION", "168h"), // 7 days
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
	return c.environment == "production"
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

// Validate checks if the configuration is valid.
func (c *AppConfig) Validate() error {
	if c.serverPort == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	if c.jwtSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if len(c.jwtSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	if c.environment != "development" && c.environment != "staging" && c.environment != "production" {
		return fmt.Errorf("environment must be one of: development, staging, production")
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

// generateDefaultJWTSecret generates a default JWT secret for development.
func generateDefaultJWTSecret() string {
	return "simple-easy-tasks-development-jwt-secret-key-32chars-minimum-length-required"
}
