package cli

import (
	"fmt"
	"strings"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskMoveCmd)
	taskCmd.AddCommand(taskAssignCmd)
	taskCmd.AddCommand(taskCommentCmd)
	taskCmd.AddCommand(taskCloseCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskDeleteCmd)

	// Task list flags
	taskListCmd.Flags().StringSliceP("status", "s", nil, "Filter by status (todo, developing, review, complete)")
	taskListCmd.Flags().StringP("assignee", "a", "", "Filter by assignee (@me for current user)")
	taskListCmd.Flags().StringSliceP("priority", "p", nil, "Filter by priority (low, medium, high, critical)")
	taskListCmd.Flags().StringSliceP("tags", "t", nil, "Filter by tags")
	taskListCmd.Flags().StringP("search", "q", "", "Search in title and description")
	taskListCmd.Flags().IntP("limit", "l", 0, "Limit number of results")
	taskListCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")

	// Task create flags
	taskCreateCmd.Flags().StringP("description", "d", "", "Task description")
	taskCreateCmd.Flags().StringP("priority", "p", "medium", "Task priority (low, medium, high, critical)")
	taskCreateCmd.Flags().StringP("assignee", "a", "", "Assignee ID")
	taskCreateCmd.Flags().StringP("status", "s", "todo", "Initial status")
	taskCreateCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")

	// Task move flags
	taskMoveCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")

	// Task assign flags
	taskAssignCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")

	// Task comment flags
	taskCommentCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")

	// Task close/reopen/delete flags
	taskCloseCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")
	taskReopenCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")
	taskDeleteCmd.Flags().StringP("project", "", "", "Project ID (overrides default)")
}

var taskCmd = &cobra.Command{
	Use:     "task",
	Short:   "Task management commands",
	Long:    `Manage tasks within projects.`,
	Aliases: []string{"t"},
}

var taskListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List tasks",
	Long:    `List tasks in the current or specified project with optional filtering.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		// Build filter options
		options := &TaskListOptions{}
		options.Status, _ = cmd.Flags().GetStringSlice("status")
		options.Assignee, _ = cmd.Flags().GetString("assignee")
		options.Priority, _ = cmd.Flags().GetStringSlice("priority")
		options.Tags, _ = cmd.Flags().GetStringSlice("tags")
		options.Search, _ = cmd.Flags().GetString("search")
		options.Limit, _ = cmd.Flags().GetInt("limit")

		client := NewAPIClientFromProfile(profile)
		tasks, err := client.GetTasks(projectID, options)
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		// Render tasks based on output format
		return RenderTasks(tasks, outputFormat)
	},
}

var taskCreateCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new task",
	Long:  `Create a new task with the specified title.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		title := args[0]
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetString("priority")
		assigneeID, _ := cmd.Flags().GetString("assignee")
		status, _ := cmd.Flags().GetString("status")

		req := &CreateTaskRequest{
			Title:       title,
			Description: description,
			Priority:    priority,
			AssigneeID:  assigneeID,
			Status:      status,
		}

		client := NewAPIClientFromProfile(profile)
		task, err := client.CreateTask(projectID, req)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Printf("✓ Task '%s' created successfully\n", task.Title)
		fmt.Printf("  ID: %s\n", task.ID)
		fmt.Printf("  Status: %s\n", task.Status)
		fmt.Printf("  Priority: %s\n", task.Priority)

		return nil
	},
}

var taskShowCmd = &cobra.Command{
	Use:   "show [task-id]",
	Short: "Show task details",
	Long:  `Show detailed information about a task.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID := profile.ProjectID
		if projectID == "" {
			return fmt.Errorf("no default project set")
		}

		taskID := args[0]

		client := NewAPIClientFromProfile(profile)
		tasks, err := client.GetTasks(projectID, nil)
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		// Find the specific task
		var task *struct {
			ID          string              `json:"id"`
			Title       string              `json:"title"`
			Description string              `json:"description"`
			Status      domain.TaskStatus   `json:"status"`
			Priority    domain.TaskPriority `json:"priority"`
		}

		for _, t := range tasks {
			if t.ID == taskID {
				task = &struct {
					ID          string              `json:"id"`
					Title       string              `json:"title"`
					Description string              `json:"description"`
					Status      domain.TaskStatus   `json:"status"`
					Priority    domain.TaskPriority `json:"priority"`
				}{
					ID:          t.ID,
					Title:       t.Title,
					Description: t.Description,
					Status:      t.Status,
					Priority:    t.Priority,
				}
				break
			}
		}

		if task == nil {
			return fmt.Errorf("task with ID '%s' not found", taskID)
		}

		return RenderTaskDetails(task, outputFormat)
	},
}

var taskEditCmd = &cobra.Command{
	Use:   "edit [task-id]",
	Short: "Edit a task",
	Long:  `Edit task properties using command-line flags or interactive editor.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// For now, we'll implement this as a placeholder
		// In a full implementation, this would open an editor or use flags
		fmt.Println("Task editing not yet implemented")
		fmt.Println("Use individual commands like 'task move', 'task assign', etc.")
		return nil
	},
}

var taskMoveCmd = &cobra.Command{
	Use:   "move [task-id] [status]",
	Short: "Move task to a different status",
	Long:  `Move a task to a different status column (todo, developing, review, complete).`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		taskID := args[0]
		newStatus := args[1]

		// Validate status
		validStatuses := []string{"todo", "developing", "review", "complete"}
		isValid := false
		for _, status := range validStatuses {
			if status == newStatus {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid status '%s'. Valid statuses: %s", newStatus, strings.Join(validStatuses, ", "))
		}

		req := &UpdateTaskRequest{
			Status: &newStatus,
		}

		client := NewAPIClientFromProfile(profile)
		task, err := client.UpdateTask(projectID, taskID, req)
		if err != nil {
			return fmt.Errorf("failed to move task: %w", err)
		}

		fmt.Printf("✓ Task '%s' moved to %s\n", task.Title, newStatus)
		return nil
	},
}

var taskAssignCmd = &cobra.Command{
	Use:   "assign [task-id] [assignee]",
	Short: "Assign task to a user",
	Long:  `Assign a task to a specific user.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		taskID := args[0]
		assigneeID := args[1]

		req := &UpdateTaskRequest{
			AssigneeID: &assigneeID,
		}

		client := NewAPIClientFromProfile(profile)
		task, err := client.UpdateTask(projectID, taskID, req)
		if err != nil {
			return fmt.Errorf("failed to assign task: %w", err)
		}

		fmt.Printf("✓ Task '%s' assigned to %s\n", task.Title, assigneeID)
		return nil
	},
}

var taskCommentCmd = &cobra.Command{
	Use:   "comment [task-id] [comment]",
	Short: "Add comment to task",
	Long:  `Add a comment to a task.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// For now, this is a placeholder as we need comment API endpoints
		taskID := args[0]
		comment := args[1]

		fmt.Printf("Comment added to task %s: %s\n", taskID, comment)
		fmt.Println("Note: Comment functionality requires API endpoint implementation")
		return nil
	},
}

var taskCloseCmd = &cobra.Command{
	Use:   "close [task-id]",
	Short: "Mark task as complete",
	Long:  `Mark a task as complete.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		taskID := args[0]
		status := "complete"

		req := &UpdateTaskRequest{
			Status: &status,
		}

		client := NewAPIClientFromProfile(profile)
		task, err := client.UpdateTask(projectID, taskID, req)
		if err != nil {
			return fmt.Errorf("failed to close task: %w", err)
		}

		fmt.Printf("✓ Task '%s' marked as complete\n", task.Title)
		return nil
	},
}

var taskReopenCmd = &cobra.Command{
	Use:   "reopen [task-id]",
	Short: "Reopen a completed task",
	Long:  `Reopen a completed task by moving it back to todo status.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		taskID := args[0]
		status := "todo"

		req := &UpdateTaskRequest{
			Status: &status,
		}

		client := NewAPIClientFromProfile(profile)
		task, err := client.UpdateTask(projectID, taskID, req)
		if err != nil {
			return fmt.Errorf("failed to reopen task: %w", err)
		}

		fmt.Printf("✓ Task '%s' reopened\n", task.Title)
		return nil
	},
}

var taskDeleteCmd = &cobra.Command{
	Use:     "delete [task-id]",
	Short:   "Delete a task",
	Long:    `Delete a task permanently. This action cannot be undone.`,
	Aliases: []string{"remove", "rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := GetCurrentProfile()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			projectID = profile.ProjectID
			if projectID == "" {
				return fmt.Errorf("no project specified and no default project set")
			}
		}

		taskID := args[0]

		// Confirmation prompt
		fmt.Printf("Are you sure you want to delete task %s? [y/N]: ", taskID)
		var response string
		_, _ = fmt.Scanln(&response)

		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Task deletion canceled")
			return nil
		}

		client := NewAPIClientFromProfile(profile)
		if err := client.DeleteTask(projectID, taskID); err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}

		fmt.Printf("✓ Task %s deleted\n", taskID)
		return nil
	},
}
