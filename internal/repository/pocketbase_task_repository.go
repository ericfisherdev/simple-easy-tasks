package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/domain"
)

type pocketbaseTaskRepository struct {
	app core.App
}

// NewPocketBaseTaskRepository creates a new PocketBase task repository.
func NewPocketBaseTaskRepository(app core.App) TaskRepository {
	return &pocketbaseTaskRepository{app: app}
}

// GetByID retrieves a task by its ID.
func (r *pocketbaseTaskRepository) GetByID(_ context.Context, id string) (*domain.Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	record, err := r.app.FindRecordById("tasks", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find task by ID %s: %w", id, err)
	}

	return r.recordToTask(record)
}

// ListByProject retrieves tasks for a specific project.
func (r *pocketbaseTaskRepository) ListByProject(
	_ context.Context, projectID string, offset, limit int,
) ([]*domain.Task, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	filter := "project = {:projectID}"
	params := dbx.Params{"projectID": projectID}

	records, err := r.app.FindRecordsByFilter(
		"tasks", filter, "position, -created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by project %s: %w", projectID, err)
	}

	return r.recordsToTasks(records)
}

// ListByAssignee retrieves tasks assigned to a specific user.
func (r *pocketbaseTaskRepository) ListByAssignee(
	_ context.Context, assigneeID string, offset, limit int,
) ([]*domain.Task, error) {
	if assigneeID == "" {
		return nil, fmt.Errorf("assignee ID cannot be empty")
	}

	filter := "assignee = {:assigneeID}"
	params := dbx.Params{"assigneeID": assigneeID}

	records, err := r.app.FindRecordsByFilter(
		"tasks", filter, "-created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by assignee %s: %w", assigneeID, err)
	}

	return r.recordsToTasks(records)
}

// ListByStatus retrieves tasks by status.
func (r *pocketbaseTaskRepository) ListByStatus(
	_ context.Context, status domain.TaskStatus, offset, limit int,
) ([]*domain.Task, error) {
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid task status: %s", status)
	}

	filter := "status = {:status}"
	params := dbx.Params{"status": string(status)}

	records, err := r.app.FindRecordsByFilter(
		"tasks", filter, "position, -created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by status %s: %w", status, err)
	}

	return r.recordsToTasks(records)
}

// ListByCreator retrieves tasks created by a specific user.
func (r *pocketbaseTaskRepository) ListByCreator(
	_ context.Context, creatorID string, offset, limit int,
) ([]*domain.Task, error) {
	if creatorID == "" {
		return nil, fmt.Errorf("creator ID cannot be empty")
	}

	filter := "reporter = {:creatorID}"
	params := dbx.Params{"creatorID": creatorID}

	records, err := r.app.FindRecordsByFilter(
		"tasks", filter, "-created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by creator %s: %w", creatorID, err)
	}

	return r.recordsToTasks(records)
}

// Search searches tasks by title, description or content.
func (r *pocketbaseTaskRepository) Search(
	_ context.Context, query string, projectID string, offset, limit int,
) ([]*domain.Task, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Sanitize search query for LIKE operations
	searchTerm := "%" + strings.ReplaceAll(query, "%", "\\%") + "%"

	filter := "(title ~ {:searchTerm} || description ~ {:searchTerm})"
	params := dbx.Params{"searchTerm": searchTerm}

	// Add project filter if provided
	if projectID != "" {
		filter += " && project = {:projectID}"
		params["projectID"] = projectID
	}

	records, err := r.app.FindRecordsByFilter(
		"tasks", filter, "-created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks with query '%s': %w", query, err)
	}

	return r.recordsToTasks(records)
}

// Count returns the total number of tasks matching criteria.
func (r *pocketbaseTaskRepository) Count(_ context.Context) (int, error) {
	total, err := r.app.CountRecords("tasks")
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return int(total), nil
}

// CountByProject returns the number of tasks in a project.
func (r *pocketbaseTaskRepository) CountByProject(_ context.Context, projectID string) (int, error) {
	if projectID == "" {
		return 0, fmt.Errorf("project ID cannot be empty")
	}

	expr := dbx.NewExp("project = {:projectID}", dbx.Params{"projectID": projectID})

	total, err := r.app.CountRecords("tasks", expr)
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks by project %s: %w", projectID, err)
	}

	return int(total), nil
}

// CountByAssignee returns the number of tasks assigned to a user.
func (r *pocketbaseTaskRepository) CountByAssignee(_ context.Context, assigneeID string) (int, error) {
	if assigneeID == "" {
		return 0, fmt.Errorf("assignee ID cannot be empty")
	}

	expr := dbx.NewExp("assignee = {:assigneeID}", dbx.Params{"assigneeID": assigneeID})

	total, err := r.app.CountRecords("tasks", expr)
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks by assignee %s: %w", assigneeID, err)
	}

	return int(total), nil
}

// CountByStatus returns the number of tasks with a specific status.
func (r *pocketbaseTaskRepository) CountByStatus(_ context.Context, status domain.TaskStatus) (int, error) {
	if !status.IsValid() {
		return 0, fmt.Errorf("invalid task status: %s", status)
	}

	expr := dbx.NewExp("status = {:status}", dbx.Params{"status": string(status)})

	total, err := r.app.CountRecords("tasks", expr)
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks by status %s: %w", status, err)
	}

	return int(total), nil
}

// ExistsByID checks if a task exists by ID.
func (r *pocketbaseTaskRepository) ExistsByID(_ context.Context, id string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("task ID cannot be empty")
	}

	_, err := r.app.FindRecordById("tasks", id)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check task existence by ID %s: %w", id, err)
	}

	return true, nil
}

// Create creates a new task.
func (r *pocketbaseTaskRepository) Create(_ context.Context, task *domain.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	collection, err := r.app.FindCollectionByNameOrId("tasks")
	if err != nil {
		return fmt.Errorf("failed to find tasks collection: %w", err)
	}

	record := core.NewRecord(collection)
	r.setTaskFields(record, task)

	// Set timestamps if provided, otherwise let PocketBase handle them
	if !task.CreatedAt.IsZero() {
		record.Set("created", task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		record.Set("updated", task.UpdatedAt)
	}
	if task.ID != "" {
		record.Id = task.ID
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save task record: %w", err)
	}

	r.updateTaskFromRecord(task, record)
	return nil
}

// Update updates an existing task.
func (r *pocketbaseTaskRepository) Update(_ context.Context, task *domain.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty for update")
	}

	record, err := r.app.FindRecordById("tasks", task.ID)
	if err != nil {
		return fmt.Errorf("failed to find task for update: %w", err)
	}

	r.setTaskFields(record, task)
	record.Set("updated", time.Now().UTC())

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update task record: %w", err)
	}

	// Update the task with the persisted timestamps
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		task.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// Delete deletes a task by ID.
func (r *pocketbaseTaskRepository) Delete(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	record, err := r.app.FindRecordById("tasks", id)
	if err != nil {
		return fmt.Errorf("failed to find task for deletion: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete task record: %w", err)
	}

	return nil
}

// BulkUpdate updates multiple tasks.
func (r *pocketbaseTaskRepository) BulkUpdate(ctx context.Context, tasks []*domain.Task) error {
	if len(tasks) == 0 {
		return nil // Nothing to update
	}

	// Validate all tasks first
	for i, task := range tasks {
		if err := task.Validate(); err != nil {
			return fmt.Errorf("validation failed for task %d: %w", i, err)
		}
		if task.ID == "" {
			return fmt.Errorf("task %d has empty ID", i)
		}
	}

	// Process updates individually
	// Note: PocketBase v0.29.3 doesn't have built-in transaction support in the basic app interface
	// For better transaction handling, consider implementing a service layer or using direct DB access
	for i, task := range tasks {
		if err := r.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to update task %d (ID: %s): %w", i, task.ID, err)
		}
	}

	return nil
}

// BulkDelete deletes multiple tasks.
func (r *pocketbaseTaskRepository) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil // Nothing to delete
	}

	// Validate all IDs first
	for i, id := range ids {
		if id == "" {
			return fmt.Errorf("task ID %d is empty", i)
		}
	}

	// Process deletes individually
	for i, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete task %d (ID: %s): %w", i, id, err)
		}
	}

	return nil
}

// ArchiveTask archives a task instead of deleting it.
func (r *pocketbaseTaskRepository) ArchiveTask(ctx context.Context, id string) error {
	return r.updateArchiveStatus(ctx, id, true)
}

// UnarchiveTask unarchives a task.
func (r *pocketbaseTaskRepository) UnarchiveTask(ctx context.Context, id string) error {
	return r.updateArchiveStatus(ctx, id, false)
}

// updateArchiveStatus handles archiving/unarchiving tasks
func (r *pocketbaseTaskRepository) updateArchiveStatus(_ context.Context, id string, archive bool) error {
	if id == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	record, err := r.app.FindRecordById("tasks", id)
	if err != nil {
		operation := "unarchiving"
		if archive {
			operation = "archiving"
		}
		return fmt.Errorf("failed to find task for %s: %w", operation, err)
	}

	// Convert to domain task to use domain logic
	task, err := r.recordToTask(record)
	if err != nil {
		return fmt.Errorf("failed to convert record to task: %w", err)
	}

	// Use domain method to archive/unarchive the task
	if archive {
		task.Archive()
	} else {
		task.Unarchive()
	}

	// Update record with archived values
	record.Set("archived", task.Archived)
	record.Set("archived_at", task.ArchivedAt)
	record.Set("updated", task.UpdatedAt)

	if err := r.app.Save(record); err != nil {
		operation := "unarchive"
		if archive {
			operation = "archive"
		}
		return fmt.Errorf("failed to %s task: %w", operation, err)
	}

	return nil
}

// recordToTask converts a PocketBase record to a domain.Task.
func (r *pocketbaseTaskRepository) recordToTask(record *core.Record) (*domain.Task, error) {
	task := &domain.Task{
		ID:          record.Id,
		Title:       record.GetString("title"),
		Description: record.GetString("description"),
		ProjectID:   record.GetString("project"),
		ReporterID:  record.GetString("reporter"),
		Status:      domain.TaskStatus(record.GetString("status")),
		Priority:    domain.TaskPriority(record.GetString("priority")),
		Position:    record.GetInt("position"),
		TimeSpent:   record.GetFloat("time_spent"),
		Progress:    record.GetInt("progress"),
		CreatedAt:   record.GetDateTime("created").Time(),
		UpdatedAt:   record.GetDateTime("updated").Time(),
	}

	// Handle optional string fields
	if assignee := record.GetString("assignee"); assignee != "" {
		task.AssigneeID = &assignee
	}
	if parentTask := record.GetString("parent_task"); parentTask != "" {
		task.ParentTaskID = &parentTask
	}

	// Handle optional date fields
	if dueDate := record.GetDateTime("due_date"); !dueDate.IsZero() {
		dueDateTime := dueDate.Time()
		task.DueDate = &dueDateTime
	}
	if startDate := record.GetDateTime("start_date"); !startDate.IsZero() {
		startDateTime := startDate.Time()
		task.StartDate = &startDateTime
	}
	if archivedAt := record.GetDateTime("archived_at"); !archivedAt.IsZero() {
		archivedDateTime := archivedAt.Time()
		task.ArchivedAt = &archivedDateTime
	}

	// Handle archived field
	task.Archived = record.GetBool("archived")

	// Handle optional numeric fields
	if effortEstimate := record.GetFloat("effort_estimate"); effortEstimate > 0 {
		task.EffortEstimate = &effortEstimate
	}

	// Handle JSON fields
	if columnPosition := record.GetString("column_position"); columnPosition != "" {
		task.ColumnPosition = []byte(columnPosition)
	}
	if githubData := record.GetString("github_data"); githubData != "" {
		task.GithubData = []byte(githubData)
	}
	if customFields := record.GetString("custom_fields"); customFields != "" {
		task.CustomFields = []byte(customFields)
	}

	// Handle array fields (if stored as JSON)
	var dependencies []string
	if err := record.UnmarshalJSONField("dependencies", &dependencies); err == nil && len(dependencies) > 0 {
		task.Dependencies = dependencies
	}

	var tags []string
	if err := record.UnmarshalJSONField("tags", &tags); err == nil && len(tags) > 0 {
		task.Tags = tags
	}

	var attachments []string
	if err := record.UnmarshalJSONField("attachments", &attachments); err == nil && len(attachments) > 0 {
		task.Attachments = attachments
	}

	return task, nil
}

// recordsToTasks converts PocketBase records to domain.Task slice.
func (r *pocketbaseTaskRepository) recordsToTasks(records []*core.Record) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, len(records))
	for i, record := range records {
		task, err := r.recordToTask(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to task: %w", err)
		}
		tasks[i] = task
	}
	return tasks, nil
}

// Helper functions to reduce cyclomatic complexity

// setTaskFields sets all task fields on a PocketBase record
func (r *pocketbaseTaskRepository) setTaskFields(record *core.Record, task *domain.Task) {
	// Required fields
	record.Set("title", task.Title)
	record.Set("project", task.ProjectID)
	record.Set("reporter", task.ReporterID)
	record.Set("status", string(task.Status))
	record.Set("position", task.Position)
	record.Set("description", task.Description)

	// Set priority with default
	priority := string(task.Priority)
	if priority == "" {
		priority = string(domain.PriorityMedium)
	}
	record.Set("priority", priority)

	// Numeric fields
	record.Set("time_spent", task.TimeSpent)
	record.Set("progress", task.Progress)

	r.setOptionalFields(record, task)
}

// setOptionalFields sets optional task fields on a PocketBase record
func (r *pocketbaseTaskRepository) setOptionalFields(record *core.Record, task *domain.Task) {
	// Optional pointer fields
	if task.AssigneeID != nil && *task.AssigneeID != "" {
		record.Set("assignee", *task.AssigneeID)
	} else {
		record.Set("assignee", "")
	}

	if task.ParentTaskID != nil && *task.ParentTaskID != "" {
		record.Set("parent_task", *task.ParentTaskID)
	} else {
		record.Set("parent_task", "")
	}

	if task.DueDate != nil && !task.DueDate.IsZero() {
		record.Set("due_date", *task.DueDate)
	} else {
		record.Set("due_date", nil)
	}

	if task.StartDate != nil && !task.StartDate.IsZero() {
		record.Set("start_date", *task.StartDate)
	} else {
		record.Set("start_date", nil)
	}

	if task.EffortEstimate != nil {
		record.Set("effort_estimate", *task.EffortEstimate)
	} else {
		record.Set("effort_estimate", nil)
	}

	r.setJSONAndArrayFields(record, task)
}

// setJSONAndArrayFields sets JSON and array fields on a PocketBase record
func (r *pocketbaseTaskRepository) setJSONAndArrayFields(record *core.Record, task *domain.Task) {
	// Handle JSON fields
	if len(task.ColumnPosition) > 0 {
		record.Set("column_position", task.ColumnPosition)
	}
	if len(task.GithubData) > 0 {
		record.Set("github_data", task.GithubData)
	}
	if len(task.CustomFields) > 0 {
		record.Set("custom_fields", task.CustomFields)
	}

	// Handle array fields
	if len(task.Dependencies) > 0 {
		record.Set("dependencies", task.Dependencies)
	} else {
		record.Set("dependencies", []string{})
	}
	if len(task.Tags) > 0 {
		record.Set("tags", task.Tags)
	} else {
		record.Set("tags", []string{})
	}
	if len(task.Attachments) > 0 {
		record.Set("attachments", task.Attachments)
	} else {
		record.Set("attachments", []string{})
	}
}

// updateTaskFromRecord updates a task with values from a PocketBase record
func (r *pocketbaseTaskRepository) updateTaskFromRecord(task *domain.Task, record *core.Record) {
	task.ID = record.Id
	if createdTime := record.GetDateTime("created"); !createdTime.IsZero() {
		task.CreatedAt = createdTime.Time()
	}
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		task.UpdatedAt = updatedTime.Time()
	}
}
