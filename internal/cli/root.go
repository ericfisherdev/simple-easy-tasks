// Package cli provides command-line interface functionality for Simple Easy Tasks.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	applicationName = "set-cli"
	version         = "1.0.0"
)

var (
	cfgFile      string
	outputFormat string
	verbose      bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   applicationName,
	Short: "Simple Easy Tasks CLI - Task management from the command line",
	Long: `set-cli is a command-line interface for the Simple Easy Tasks application.

It provides powerful task and project management capabilities directly from your terminal,
including authentication, task creation, project management, and integration with Git workflows.`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.set-cli.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml, csv)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".set-cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".set-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Read config if available
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
		}
	}
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	if cfgFile != "" {
		// Convert to absolute path
		absPath, err := filepath.Abs(cfgFile)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for config file: %w", err)
		}
		return absPath, nil
	}

	// Try user home directory first
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".set-cli.yaml"), nil
	}

	// Fallback to user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine config directory: both UserHomeDir and UserConfigDir failed")
	}

	return filepath.Join(configDir, ".set-cli.yaml"), nil
}
