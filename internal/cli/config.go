package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the CLI configuration
type Config struct {
	DefaultProfile string             `json:"default_profile" yaml:"default_profile"`
	Profiles       map[string]Profile `json:"profiles" yaml:"profiles"`
}

// Profile represents a configuration profile for different environments
type Profile struct {
	Name      string `json:"name" yaml:"name"`
	ServerURL string `json:"server_url" yaml:"server_url"`
	Token     string `json:"token" yaml:"token"`
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// validateConfigPath validates that the config path is safe
func validateConfigPath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid config path: path traversal not allowed")
	}

	// Ensure it's an absolute path or within user home
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("invalid config path: must be absolute path")
	}

	return nil
}

// LoadConfig loads the configuration from file
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	config := &Config{
		Profiles: make(map[string]Profile),
	}

	// Validate config path for security
	if validateErr := validateConfigPath(configPath); validateErr != nil {
		return nil, fmt.Errorf("config path validation failed: %w", validateErr)
	}

	// If config file doesn't exist, return empty config
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		return config, nil
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // Path is validated by validateConfigPath
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Validate config path for security
	if validateErr := validateConfigPath(configPath); validateErr != nil {
		return fmt.Errorf("config path validation failed: %w", validateErr)
	}

	// Create config directory if it doesn't exist
	if mkdirErr := os.MkdirAll(filepath.Dir(configPath), 0750); mkdirErr != nil {
		return fmt.Errorf("failed to create config directory: %w", mkdirErr)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentProfile returns the current active profile
func GetCurrentProfile() (*Profile, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	profileName := config.DefaultProfile
	if profileName == "" {
		profileName = "default"
	}

	// Check for environment override
	if envProfile := viper.GetString("profile"); envProfile != "" {
		profileName = envProfile
	}

	profile, exists := config.Profiles[profileName]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", profileName)
	}

	return &profile, nil
}

// SetCurrentProfile sets the default profile
func SetCurrentProfile(profileName string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if _, exists := config.Profiles[profileName]; !exists {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	config.DefaultProfile = profileName
	return SaveConfig(config)
}

// AddProfile adds a new profile to the configuration
func AddProfile(profile Profile) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if config.Profiles == nil {
		config.Profiles = make(map[string]Profile)
	}

	config.Profiles[profile.Name] = profile

	// Set as default if it's the first profile
	if config.DefaultProfile == "" {
		config.DefaultProfile = profile.Name
	}

	return SaveConfig(config)
}

// RemoveProfile removes a profile from the configuration
func RemoveProfile(profileName string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if _, exists := config.Profiles[profileName]; !exists {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	delete(config.Profiles, profileName)

	// Clear default if removing default profile
	if config.DefaultProfile == profileName {
		config.DefaultProfile = ""
		// Set first available profile as default
		for name := range config.Profiles {
			config.DefaultProfile = name
			break
		}
	}

	return SaveConfig(config)
}

// ListProfiles returns all available profiles
func ListProfiles() ([]Profile, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(config.Profiles))
	for _, profile := range config.Profiles {
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// ValidateProfile validates a profile configuration
func ValidateProfile(profile *Profile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	if profile.ServerURL == "" {
		return fmt.Errorf("server URL is required")
	}

	if profile.Token == "" {
		return fmt.Errorf("authentication token is required")
	}

	return nil
}

// GetConfigAsJSON returns configuration as JSON string for debugging
func GetConfigAsJSON() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	// Mask sensitive information
	maskedConfig := *config
	maskedConfig.Profiles = make(map[string]Profile)

	for name, profile := range config.Profiles {
		maskedProfile := profile
		if maskedProfile.Token != "" {
			maskedProfile.Token = "***masked***"
		}
		maskedConfig.Profiles[name] = maskedProfile
	}

	data, err := json.MarshalIndent(maskedConfig, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
