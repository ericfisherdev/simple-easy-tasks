package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvLoader handles loading environment variables from .env files.
type EnvLoader struct {
	loaded  map[string]string
	baseDir string
}

// NewEnvLoader creates a new environment loader.
func NewEnvLoader(baseDir string) *EnvLoader {
	return &EnvLoader{
		baseDir: baseDir,
		loaded:  make(map[string]string),
	}
}

// LoadEnvFiles loads environment variables from .env files in priority order.
func (l *EnvLoader) LoadEnvFiles(environment string) error {
	// Priority order (last one wins):
	// 1. .env.defaults (if exists)
	// 2. .env.{environment}
	// 3. .env.local
	// 4. .env

	envFiles := []string{
		".env.defaults",
		fmt.Sprintf(".env.%s", environment),
		".env.local",
		".env",
	}

	for _, filename := range envFiles {
		path := filepath.Join(l.baseDir, filename)
		if err := l.loadEnvFile(path); err != nil {
			// Only log error, don't fail - some files are optional
			if !os.IsNotExist(err) {
				fmt.Printf("Warning: Error loading %s: %v\n", filename, err)
			}
		}
	}

	// Set loaded environment variables
	for key, value := range l.loaded {
		if os.Getenv(key) == "" { // Only set if not already set
			if err := os.Setenv(key, value); err != nil {
				fmt.Printf("Warning: Failed to set environment variable %s: %v\n", key, err)
			}
		}
	}

	return nil
}

// loadEnvFile loads a single .env file.
func (l *EnvLoader) loadEnvFile(path string) error {
	file, err := os.Open(path) // #nosec G304 -- path is validated by caller
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Printf("Warning: Failed to close file %s: %v\n", path, cerr)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Warning: Invalid format in %s at line %d: %s\n", path, lineNum, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Expand environment variables in value
		value = os.ExpandEnv(value)

		l.loaded[key] = value
	}

	return scanner.Err()
}

// GetLoadedVars returns all loaded environment variables.
func (l *EnvLoader) GetLoadedVars() map[string]string {
	result := make(map[string]string)
	for k, v := range l.loaded {
		result[k] = v
	}
	return result
}

// AutoLoadEnv automatically loads environment files based on detected environment.
func AutoLoadEnv(baseDir string) error {
	loader := NewEnvLoader(baseDir)

	// Detect environment from ENV or ENVIRONMENT variables
	env := os.Getenv("ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "development" // default
	}

	return loader.LoadEnvFiles(env)
}
