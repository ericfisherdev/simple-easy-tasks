package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectSelectCmd)

	// Project create flags
	projectCreateCmd.Flags().StringP("description", "d", "", "Project description")
	projectCreateCmd.Flags().BoolP("select", "s", false, "Select as default project after creation")

	// Project show flags
	projectShowCmd.Flags().BoolP("tasks", "t", false, "Include task summary")
}

var projectCmd = &cobra.Command{
	Use:     "project",
	Short:   "Project management commands",
	Long:    `Manage projects in Simple Easy Tasks.`,
	Aliases: []string{"proj", "p"},
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all projects",
	Long:    `List all projects accessible to the current user.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		client := NewAPIClientFromProfile(profile)
		projects, err := client.GetProjects()
		if err != nil {
			return fmt.Errorf("failed to get projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("No projects found")
			return nil
		}

		// Render projects based on output format
		return RenderProjects(projects, profile.ProjectID, outputFormat)
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project",
	Long:  `Create a new project with the specified name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectName := args[0]
		description, _ := cmd.Flags().GetString("description")
		selectProject, _ := cmd.Flags().GetBool("select")

		req := &CreateProjectRequest{
			Title:       projectName,
			Description: description,
		}

		client := NewAPIClientFromProfile(profile)
		project, err := client.CreateProject(req)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		fmt.Printf("✓ Project '%s' created successfully\n", project.Title)
		fmt.Printf("  ID: %s\n", project.ID)

		// Select as default project if requested
		if selectProject {
			profile.ProjectID = project.ID
			if err := AddProfile(*profile); err != nil {
				fmt.Printf("Warning: Failed to set as default project: %v\n", err)
			} else {
				fmt.Printf("✓ Project selected as default\n")
			}
		}

		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show [project-id]",
	Short: "Show project details",
	Long:  `Show detailed information about a project.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		var projectID string
		if len(args) > 0 {
			projectID = args[0]
		} else {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		includeTasks, _ := cmd.Flags().GetBool("tasks")

		client := NewAPIClientFromProfile(profile)
		project, err := client.GetProject(projectID)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		return RenderProjectDetails(project, includeTasks, client, outputFormat)
	},
}

var projectSelectCmd = &cobra.Command{
	Use:     "select [project-id]",
	Short:   "Select a project as default",
	Long:    `Set the specified project as default for task operations.`,
	Aliases: []string{"switch", "use"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID := args[0]

		// Verify project exists
		client := NewAPIClientFromProfile(profile)
		project, err := client.GetProject(projectID)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		// Update profile
		profile.ProjectID = projectID
		if err := AddProfile(*profile); err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}

		fmt.Printf("✓ Project '%s' selected as default\n", project.Title)
		return nil
	},
}
