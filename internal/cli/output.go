package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"
)

const (
	formatJSON = "json"
	formatYAML = "yaml"
	formatYML  = "yml"
)

// RenderProjects renders a list of projects in the specified format
func RenderProjects(projects []domain.Project, defaultProjectID, format string) error {
	switch strings.ToLower(format) {
	case formatJSON:
		return renderProjectsJSON(projects)
	case formatYAML, formatYML:
		return renderProjectsYAML(projects)
	case "csv":
		return renderProjectsCSV(projects)
	default:
		return renderProjectsTable(projects, defaultProjectID)
	}
}

// RenderProjectDetails renders detailed project information
func RenderProjectDetails(project *domain.Project, includeTasks bool, client *APIClient, format string) error {
	switch strings.ToLower(format) {
	case formatJSON:
		return renderProjectDetailsJSON(project, includeTasks, client)
	case formatYAML, formatYML:
		return renderProjectDetailsYAML(project, includeTasks, client)
	default:
		return renderProjectDetailsTable(project, includeTasks, client)
	}
}

// RenderTasks renders a list of tasks in the specified format
func RenderTasks(tasks []domain.Task, format string) error {
	switch strings.ToLower(format) {
	case formatJSON:
		return renderTasksJSON(tasks)
	case formatYAML, formatYML:
		return renderTasksYAML(tasks)
	case "csv":
		return renderTasksCSV(tasks)
	default:
		return renderTasksTable(tasks)
	}
}

// RenderTaskDetails renders detailed task information
func RenderTaskDetails(task interface{}, format string) error {
	switch strings.ToLower(format) {
	case formatJSON:
		return renderTaskDetailsJSON(task)
	case formatYAML, formatYML:
		return renderTaskDetailsYAML(task)
	default:
		return renderTaskDetailsTable(task)
	}
}

// Table rendering functions
func renderProjectsTable(projects []domain.Project, defaultProjectID string) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Description", "Created", "Default"})

	for _, project := range projects {
		isDefault := ""
		if project.ID == defaultProjectID {
			isDefault = "*"
		}

		createdAt := ""
		if !project.CreatedAt.IsZero() {
			createdAt = project.CreatedAt.Format("2006-01-02")
		}

		description := project.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		t.AppendRow(table.Row{
			project.ID,
			project.Title,
			description,
			createdAt,
			isDefault,
		})
	}

	t.SetStyle(table.StyleLight)
	t.Render()
	return nil
}

func renderTasksTable(tasks []domain.Task) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Title", "Status", "Priority", "Assignee", "Created"})

	for _, task := range tasks {
		createdAt := ""
		if !task.CreatedAt.IsZero() {
			createdAt = task.CreatedAt.Format("2006-01-02")
		}

		title := task.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		assignee := "Unassigned"
		if task.AssigneeID != nil && *task.AssigneeID != "" {
			assignee = *task.AssigneeID
		}

		// Colorize status
		status := string(task.Status)
		switch strings.ToLower(status) {
		case "complete":
			status = "✓ " + status
		case "developing":
			status = "🔧 " + status
		case "review":
			status = "👁 " + status
		case "todo":
			status = "📋 " + status
		}

		t.AppendRow(table.Row{
			task.ID,
			title,
			status,
			string(task.Priority),
			assignee,
			createdAt,
		})
	}

	t.SetStyle(table.StyleLight)
	t.Render()
	return nil
}

func renderProjectDetailsTable(project *domain.Project, includeTasks bool, client *APIClient) error {
	fmt.Printf("Project: %s\n", project.Title)
	fmt.Printf("ID: %s\n", project.ID)
	fmt.Printf("Description: %s\n", project.Description)

	if !project.CreatedAt.IsZero() {
		fmt.Printf("Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if !project.UpdatedAt.IsZero() {
		fmt.Printf("Updated: %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	if includeTasks {
		fmt.Printf("\nTasks:\n")
		tasks, err := client.GetTasks(project.ID, nil)
		switch {
		case err != nil:
			fmt.Printf("Error loading tasks: %v\n", err)
		case len(tasks) == 0:
			fmt.Printf("No tasks found\n")
		default:
			return renderTasksTable(tasks)
		}
	}

	return nil
}

func renderTaskDetailsTable(task interface{}) error {
	// Use reflection or type assertion to handle different task types
	switch t := task.(type) {
	case *domain.Task:
		fmt.Printf("Task: %s\n", t.Title)
		fmt.Printf("ID: %s\n", t.ID)
		fmt.Printf("Description: %s\n", t.Description)
		fmt.Printf("Status: %s\n", t.Status)
		fmt.Printf("Priority: %s\n", t.Priority)
		if !t.CreatedAt.IsZero() {
			fmt.Printf("Created: %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	default:
		// Handle generic struct
		data, _ := json.MarshalIndent(task, "", "  ")
		fmt.Printf("%s\n", data)
	}

	return nil
}

// JSON rendering functions
func renderProjectsJSON(projects []domain.Project) error {
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func renderProjectDetailsJSON(project *domain.Project, includeTasks bool, client *APIClient) error {
	result := map[string]interface{}{
		"project": project,
	}

	if includeTasks {
		tasks, err := client.GetTasks(project.ID, nil)
		if err != nil {
			result["tasks_error"] = err.Error()
		} else {
			result["tasks"] = tasks
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func renderTasksJSON(tasks []domain.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func renderTaskDetailsJSON(task interface{}) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

// YAML rendering functions
func renderProjectsYAML(projects []domain.Project) error {
	data, err := yaml.Marshal(projects)
	if err != nil {
		return err
	}
	fmt.Printf("%s", data)
	return nil
}

func renderProjectDetailsYAML(project *domain.Project, includeTasks bool, client *APIClient) error {
	result := map[string]interface{}{
		"project": project,
	}

	if includeTasks {
		tasks, err := client.GetTasks(project.ID, nil)
		if err != nil {
			result["tasks_error"] = err.Error()
		} else {
			result["tasks"] = tasks
		}
	}

	data, err := yaml.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Printf("%s", data)
	return nil
}

func renderTasksYAML(tasks []domain.Task) error {
	data, err := yaml.Marshal(tasks)
	if err != nil {
		return err
	}
	fmt.Printf("%s", data)
	return nil
}

func renderTaskDetailsYAML(task interface{}) error {
	data, err := yaml.Marshal(task)
	if err != nil {
		return err
	}
	fmt.Printf("%s", data)
	return nil
}

// CSV rendering functions
func renderProjectsCSV(projects []domain.Project) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Header
	_ = writer.Write([]string{"ID", "Title", "Description", "Created", "Updated"})

	// Data
	for _, project := range projects {
		createdAt := ""
		if !project.CreatedAt.IsZero() {
			createdAt = project.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !project.UpdatedAt.IsZero() {
			updatedAt = project.UpdatedAt.Format(time.RFC3339)
		}

		_ = writer.Write([]string{
			project.ID,
			project.Title,
			project.Description,
			createdAt,
			updatedAt,
		})
	}

	return nil
}

func renderTasksCSV(tasks []domain.Task) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Header
	_ = writer.Write([]string{"ID", "Title", "Description", "Status", "Priority", "AssigneeID", "Created", "Updated"})

	// Data
	for _, task := range tasks {
		createdAt := ""
		if !task.CreatedAt.IsZero() {
			createdAt = task.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !task.UpdatedAt.IsZero() {
			updatedAt = task.UpdatedAt.Format(time.RFC3339)
		}

		assigneeID := ""
		if task.AssigneeID != nil {
			assigneeID = *task.AssigneeID
		}

		_ = writer.Write([]string{
			task.ID,
			task.Title,
			task.Description,
			string(task.Status),
			string(task.Priority),
			assigneeID,
			createdAt,
			updatedAt,
		})
	}

	return nil
}

// Utility functions

// Success prints a success message with a checkmark
func Success(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

// Warning prints a warning message
func Warning(format string, args ...interface{}) {
	fmt.Printf("⚠ "+format+"\n", args...)
}

// Error prints an error message
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

// Info prints an informational message
func Info(format string, args ...interface{}) {
	fmt.Printf("ℹ "+format+"\n", args...)
}
