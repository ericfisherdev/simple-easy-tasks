package cli

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(profileCmd)

	// Profile subcommands
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileSelectCmd)
	profileCmd.AddCommand(profileShowCmd)

	// Login command flags
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	loginCmd.Flags().StringP("password", "p", "", "Password (not recommended, use interactive prompt)")
	loginCmd.Flags().StringP("server", "s", "http://localhost:8090", "Server URL")
	loginCmd.Flags().StringP("profile", "", "default", "Profile name")

	// Profile create flags
	profileCreateCmd.Flags().StringP("server", "s", "", "Server URL")
	profileCreateCmd.Flags().StringP("token", "t", "", "API Token")
	_ = profileCreateCmd.MarkFlagRequired("server")
	_ = profileCreateCmd.MarkFlagRequired("token")
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication and user profiles for the Simple Easy Tasks API.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Simple Easy Tasks API",
	Long: `Authenticate with the Simple Easy Tasks API using email and password.

This command will prompt for credentials if not provided via flags.
The authentication token will be stored securely for future use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		serverURL, _ := cmd.Flags().GetString("server")
		profileName, _ := cmd.Flags().GetString("profile")

		// Prompt for email if not provided
		if email == "" {
			fmt.Print("Email: ")
			_, _ = fmt.Scanln(&email)
		}

		// Prompt for password if not provided
		if password == "" {
			fmt.Print("Password: ")
			bytePassword, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = string(bytePassword)
			fmt.Println() // New line after password input
		}

		// Validate inputs
		if email == "" {
			return fmt.Errorf("email is required")
		}
		if password == "" {
			return fmt.Errorf("password is required")
		}

		// Create API client
		client := NewAPIClient(serverURL, "")

		// Attempt login
		fmt.Printf("Authenticating with %s...\n", serverURL)
		loginResp, err := client.Login(email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Create profile
		profile := Profile{
			Name:      profileName,
			ServerURL: serverURL,
			Token:     loginResp.Token,
		}

		if err := AddProfile(profile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("✓ Successfully authenticated as %s\n", loginResp.User.Email)
		fmt.Printf("✓ Profile '%s' created and set as default\n", profileName)

		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout [profile]",
	Short: "Logout and remove authentication token",
	Long: `Remove the authentication token for the specified profile.
If no profile is specified, removes the current default profile.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		} else {
			// Get current profile
			config, err := LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			profileName = config.DefaultProfile
		}

		if profileName == "" {
			return fmt.Errorf("no profile specified and no default profile set")
		}

		if err := RemoveProfile(profileName); err != nil {
			return fmt.Errorf("failed to remove profile: %w", err)
		}

		fmt.Printf("✓ Profile '%s' removed\n", profileName)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display current authentication status and active profile information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			fmt.Println("Status: Not authenticated")
			fmt.Printf("Error: %s\n", err.Error())
			return nil
		}

		// Test connection
		client := NewAPIClientFromProfile(profile)
		if err := client.TestConnection(); err != nil {
			fmt.Printf("Status: Authentication token exists but connection failed\n")
			fmt.Printf("Profile: %s\n", profile.Name)
			fmt.Printf("Server: %s\n", profile.ServerURL)
			fmt.Printf("Error: %s\n", err.Error())
			return nil
		}

		fmt.Printf("Status: ✓ Authenticated\n")
		fmt.Printf("Profile: %s\n", profile.Name)
		fmt.Printf("Server: %s\n", profile.ServerURL)

		return nil
	},
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage authentication profiles",
	Long:  `Manage multiple authentication profiles for different environments.`,
}

var profileListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all profiles",
	Long:    `List all configured authentication profiles.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles configured")
			return nil
		}

		config, err := LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("Available profiles:")
		for _, profile := range profiles {
			prefix := "  "
			if profile.Name == config.DefaultProfile {
				prefix = "* "
			}

			fmt.Printf("%s%s\n", prefix, profile.Name)
			fmt.Printf("    Server: %s\n", profile.ServerURL)
			if profile.ProjectID != "" {
				fmt.Printf("    Project: %s\n", profile.ProjectID)
			}
		}

		return nil
	},
}

var profileCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new profile",
	Long:  `Create a new authentication profile with specified credentials.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]
		serverURL, _ := cmd.Flags().GetString("server")
		token, _ := cmd.Flags().GetString("token")

		profile := Profile{
			Name:      profileName,
			ServerURL: serverURL,
			Token:     token,
		}

		if err := ValidateProfile(&profile); err != nil {
			return fmt.Errorf("invalid profile: %w", err)
		}

		// Test connection
		client := NewAPIClientFromProfile(&profile)
		if err := client.TestConnection(); err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}

		if err := AddProfile(profile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}

		fmt.Printf("✓ Profile '%s' created successfully\n", profileName)
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Short:   "Delete a profile",
	Long:    `Delete an authentication profile.`,
	Aliases: []string{"remove", "rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		if err := RemoveProfile(profileName); err != nil {
			return fmt.Errorf("failed to delete profile: %w", err)
		}

		fmt.Printf("✓ Profile '%s' deleted\n", profileName)
		return nil
	},
}

var profileSelectCmd = &cobra.Command{
	Use:     "select [name]",
	Short:   "Select a profile as default",
	Long:    `Set the specified profile as the default for all operations.`,
	Aliases: []string{"switch", "use"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		if err := SetCurrentProfile(profileName); err != nil {
			return fmt.Errorf("failed to select profile: %w", err)
		}

		fmt.Printf("✓ Profile '%s' selected as default\n", profileName)
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show profile details",
	Long:  `Display detailed information about a profile.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		}

		var profile *Profile
		var err error

		if profileName == "" {
			// Show current profile
			profile, err = GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}
		} else {
			// Show specific profile
			config, err := LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			p, exists := config.Profiles[profileName]
			if !exists {
				return fmt.Errorf("profile '%s' not found", profileName)
			}
			profile = &p
		}

		fmt.Printf("Profile: %s\n", profile.Name)
		fmt.Printf("Server: %s\n", profile.ServerURL)
		if profile.ProjectID != "" {
			fmt.Printf("Project: %s\n", profile.ProjectID)
		}

		// Mask token
		if profile.Token != "" {
			fmt.Printf("Token: %s...%s\n",
				profile.Token[:8],
				strings.Repeat("*", len(profile.Token)-16)+profile.Token[len(profile.Token)-8:])
		} else {
			fmt.Printf("Token: Not set\n")
		}

		return nil
	},
}
