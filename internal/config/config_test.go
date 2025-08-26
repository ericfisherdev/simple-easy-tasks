package config

import (
	"os"
	"strings"
	"testing"
)

func TestGetJWTSecret_Production_RequiresEnvVar(t *testing.T) {
	// Save original environment
	original := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", original)

	// Clear JWT_SECRET
	os.Unsetenv("JWT_SECRET")

	// Test that production panics without JWT_SECRET
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when JWT_SECRET is not set in production")
		}
	}()

	getJWTSecret(EnvProduction)
}

func TestGetJWTSecret_Development_GeneratesSecure(t *testing.T) {
	// Save original environment
	original := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", original)

	// Clear JWT_SECRET
	os.Unsetenv("JWT_SECRET")

	secret := getJWTSecret(EnvDevelopment)

	if len(secret) < 32 {
		t.Errorf("Generated secret too short: %d characters", len(secret))
	}

	// Should not be the old default
	if secret == "simple-easy-tasks-development-jwt-secret-key-32chars-minimum-length-required" {
		t.Error("Generated secret should not be the old default")
	}
}

func TestIsDefaultSecret(t *testing.T) {
	tests := []struct {
		secret   string
		expected bool
	}{
		{"simple-easy-tasks-development-jwt-secret-key-32chars-minimum-length-required", true},
		{"secret", true},
		{"jwt-secret", true},
		// New .env.example patterns
		{"your-super-secret-jwt-key-with-at-least-32-characters", true},
		{"your-super-secret-password-reset-key-with-at-least-32-characters", true},
		{"your-super-secret", true},
		{"super-secret", true},
		{"super-secret-key", true},
		{"super-secret-jwt-key", true},
		{"changeme", true},
		{"changeme123", true},
		{"placeholder", true},
		{"example-secret", true},
		{"example-key", true},
		{"sample-secret", true},
		{"sample-key", true},
		// Non-default secrets
		{"random-secure-secret-that-is-not-default", false},
		{"", false},
		{"actual-secure-random-key-with-32-chars-or-more", false},
	}

	for _, test := range tests {
		result := isDefaultSecret(test.secret)
		if result != test.expected {
			t.Errorf("isDefaultSecret(%q) = %v, expected %v", test.secret, result, test.expected)
		}
	}
}

func TestConfig_Validate_RejectsDefaultInProduction(t *testing.T) {
	config := &AppConfig{
		serverPort:  "8080",
		jwtSecret:   "simple-easy-tasks-development-jwt-secret-key-32chars-minimum-length-required",
		environment: EnvProduction,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for default secret in production")
	}

	if !strings.Contains(err.Error(), "default JWT secrets") {
		t.Errorf("Expected error about default secrets, got: %v", err)
	}
}

func TestGenerateSecureJWTSecret(t *testing.T) {
	secret1 := generateSecureJWTSecret()
	secret2 := generateSecureJWTSecret()

	// Should be different each time
	if secret1 == secret2 {
		t.Error("Generated secrets should be unique")
	}

	// Should be at least 32 characters when base64 encoded
	if len(secret1) < 32 {
		t.Errorf("Generated secret too short: %d characters", len(secret1))
	}

	// Should not be a default secret
	if isDefaultSecret(secret1) {
		t.Error("Generated secret should not match any default secret")
	}
}
