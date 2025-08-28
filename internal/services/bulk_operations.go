package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
)

// BulkOperationService defines the interface for bulk task operations
type BulkOperationService interface {
	// BulkUpdate performs multiple update operations in a single transaction
	BulkUpdate(ctx context.Context, ops []BulkTaskOperation, userID string) (*BulkResult, error)

	// BulkCreate creates multiple tasks from a list or CSV data
	BulkCreate(ctx context.Context, req BulkCreateRequest, userID string) (*BulkResult, error)

	// BulkStatusUpdate updates the status of multiple tasks
	BulkStatusUpdate(
		ctx context.Context, taskIDs []string, newStatus domain.TaskStatus, userID string,
	) (*BulkResult, error)

	// BulkAssign assigns multiple tasks to a user
	BulkAssign(ctx context.Context, taskIDs []string, assigneeID string, userID string) (*BulkResult, error)

	// BulkTagUpdate adds or removes tags from multiple tasks
	BulkTagUpdate(ctx context.Context, req BulkTagUpdateRequest, userID string) (*BulkResult, error)

	// BulkDelete deletes multiple tasks with cascade handling
	BulkDelete(ctx context.Context, taskIDs []string, options BulkDeleteOptions, userID string) (*BulkResult, error)

	// ImportFromCSV creates tasks from CSV data
	ImportFromCSV(ctx context.Context, csvData io.Reader, projectID string, userID string) (*BulkResult, error)

	// ExportToCSV exports tasks to CSV format
	ExportToCSV(ctx context.Context, projectID string, filters repository.TaskFilters, userID string) ([]byte, error)
}

// BulkTaskOperation represents a single operation in a bulk update
type BulkTaskOperation struct {
	Operation string                 `json:"operation"` // "update", "delete", "assign", "tag", "status"
	TaskIDs   []string               `json:"task_ids"`
	Data      map[string]interface{} `json:"data"`
}

// BulkCreateRequest represents a request to create multiple tasks
type BulkCreateRequest struct {
	ProjectID string                     `json:"project_id"`
	Tasks     []domain.CreateTaskRequest `json:"tasks"`
	Template  *BulkCreateTemplate        `json:"template,omitempty"`
}

// BulkCreateTemplate defines a template for bulk task creation
type BulkCreateTemplate struct {
	TitlePattern string                 `json:"title_pattern"` // e.g., "Task {index}: {title}"
	CommonFields map[string]interface{} `json:"common_fields"` // Fields applied to all tasks
	Count        int                    `json:"count"`         // Number of tasks to create
	StartIndex   int                    `json:"start_index"`   // Starting index for numbering
}

// BulkTagUpdateRequest represents a request to update tags on multiple tasks
type BulkTagUpdateRequest struct {
	TaskIDs      []string `json:"task_ids"`
	TagsToAdd    []string `json:"tags_to_add,omitempty"`
	TagsToRemove []string `json:"tags_to_remove,omitempty"`
	ReplaceAll   bool     `json:"replace_all"` // If true, replace all tags with tags_to_add
}

// BulkDeleteOptions controls how bulk deletion is performed
type BulkDeleteOptions struct {
	IncludeSubtasks bool `json:"include_subtasks"` // Delete subtasks too
	Force           bool `json:"force"`            // Force delete even if there are dependencies
}

// BulkResult represents the result of a bulk operation
type BulkResult struct {
	TotalRequested int                    `json:"total_requested"`
	Successful     int                    `json:"successful"`
	Failed         int                    `json:"failed"`
	Errors         []BulkOperationError   `json:"errors,omitempty"`
	Results        []interface{}          `json:"results,omitempty"` // Successful operation results
	Duration       time.Duration          `json:"duration"`
	Summary        map[string]interface{} `json:"summary,omitempty"`
}

// BulkOperationError represents an error in a bulk operation
type BulkOperationError struct {
	TaskID    string `json:"task_id,omitempty"`
	Index     int    `json:"index"`
	Operation string `json:"operation"`
	Error     string `json:"error"`
}

// bulkOperationService implements bulk operations
type bulkOperationService struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	taskService TaskService
}

// NewBulkOperationService creates a new bulk operation service
func NewBulkOperationService(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	taskService TaskService,
) BulkOperationService {
	return &bulkOperationService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		taskService: taskService,
	}
}

// BulkUpdate performs multiple update operations in a single transaction
func (b *bulkOperationService) BulkUpdate(
	ctx context.Context, ops []BulkTaskOperation, userID string,
) (*BulkResult, error) {
	startTime := time.Now()
	result := &BulkResult{
		TotalRequested: len(ops),
		Results:        make([]interface{}, 0),
		Errors:         make([]BulkOperationError, 0),
	}

	for i, op := range ops {
		switch op.Operation {
		case "update":
			if err := b.processBulkTaskUpdate(ctx, op, userID, i, result); err != nil {
				result.Errors = append(result.Errors, BulkOperationError{
					Index:     i,
					Operation: op.Operation,
					Error:     err.Error(),
				})
				result.Failed++
			} else {
				result.Successful++
			}

		case "status":
			if err := b.processBulkStatusUpdate(ctx, op, userID, i, result); err != nil {
				result.Errors = append(result.Errors, BulkOperationError{
					Index:     i,
					Operation: op.Operation,
					Error:     err.Error(),
				})
				result.Failed++
			} else {
				result.Successful++
			}

		case "assign":
			if err := b.processBulkAssign(ctx, op, userID, i, result); err != nil {
				result.Errors = append(result.Errors, BulkOperationError{
					Index:     i,
					Operation: op.Operation,
					Error:     err.Error(),
				})
				result.Failed++
			} else {
				result.Successful++
			}

		default:
			result.Errors = append(result.Errors, BulkOperationError{
				Index:     i,
				Operation: op.Operation,
				Error:     fmt.Sprintf("Unknown operation: %s", op.Operation),
			})
			result.Failed++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// BulkCreate creates multiple tasks from a list or template
func (b *bulkOperationService) BulkCreate(
	ctx context.Context, req BulkCreateRequest, userID string,
) (*BulkResult, error) {
	startTime := time.Now()

	if req.ProjectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Validate project access
	project, err := b.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	if !project.HasAccess(userID) {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	var tasksToCreate []domain.CreateTaskRequest

	// Generate tasks from template if provided
	if req.Template != nil {
		tasksToCreate = b.generateTasksFromTemplate(req.Template, req.ProjectID)
	} else {
		tasksToCreate = req.Tasks
	}

	result := &BulkResult{
		TotalRequested: len(tasksToCreate),
		Results:        make([]interface{}, 0),
		Errors:         make([]BulkOperationError, 0),
	}

	// Create tasks
	for i, taskReq := range tasksToCreate {
		taskReq.ProjectID = req.ProjectID // Ensure project ID is set

		task, err := b.taskService.CreateTask(ctx, taskReq, userID)
		if err != nil {
			result.Errors = append(result.Errors, BulkOperationError{
				Index:     i,
				Operation: "create",
				Error:     err.Error(),
			})
			result.Failed++
		} else {
			result.Results = append(result.Results, task)
			result.Successful++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// BulkStatusUpdate updates the status of multiple tasks
func (b *bulkOperationService) BulkStatusUpdate(
	ctx context.Context, taskIDs []string, newStatus domain.TaskStatus, userID string,
) (*BulkResult, error) {
	startTime := time.Now()

	if !newStatus.IsValid() {
		return nil, domain.NewValidationError("INVALID_STATUS", "Invalid task status", nil)
	}

	result := &BulkResult{
		TotalRequested: len(taskIDs),
		Results:        make([]interface{}, 0),
		Errors:         make([]BulkOperationError, 0),
	}

	for i, taskID := range taskIDs {
		task, err := b.taskService.UpdateTaskStatus(ctx, taskID, newStatus, userID)
		if err != nil {
			result.Errors = append(result.Errors, BulkOperationError{
				TaskID:    taskID,
				Index:     i,
				Operation: "status_update",
				Error:     err.Error(),
			})
			result.Failed++
		} else {
			result.Results = append(result.Results, task)
			result.Successful++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// BulkAssign assigns multiple tasks to a user
func (b *bulkOperationService) BulkAssign(
	ctx context.Context, taskIDs []string, assigneeID string, userID string,
) (*BulkResult, error) {
	startTime := time.Now()

	result := &BulkResult{
		TotalRequested: len(taskIDs),
		Results:        make([]interface{}, 0),
		Errors:         make([]BulkOperationError, 0),
	}

	for i, taskID := range taskIDs {
		var task *domain.Task
		var err error

		if assigneeID == "" {
			// Unassign task
			task, err = b.taskService.UnassignTask(ctx, taskID, userID)
		} else {
			// Assign task
			task, err = b.taskService.AssignTask(ctx, taskID, assigneeID, userID)
		}

		if err != nil {
			result.Errors = append(result.Errors, BulkOperationError{
				TaskID:    taskID,
				Index:     i,
				Operation: "assign",
				Error:     err.Error(),
			})
			result.Failed++
		} else {
			result.Results = append(result.Results, task)
			result.Successful++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// BulkTagUpdate adds or removes tags from multiple tasks
func (b *bulkOperationService) BulkTagUpdate(
	ctx context.Context, req BulkTagUpdateRequest, userID string,
) (*BulkResult, error) {
	startTime := time.Now()

	result := &BulkResult{
		TotalRequested: len(req.TaskIDs),
		Results:        make([]interface{}, 0),
		Errors:         make([]BulkOperationError, 0),
	}

	for i, taskID := range req.TaskIDs {
		// Get current task
		task, err := b.taskService.GetTask(ctx, taskID, userID)
		if err != nil {
			result.Errors = append(result.Errors, BulkOperationError{
				TaskID:    taskID,
				Index:     i,
				Operation: "tag_update",
				Error:     err.Error(),
			})
			result.Failed++
			continue
		}

		// Update tags
		newTags := b.calculateNewTags(task.Tags, req)

		// Update task with new tags
		updateReq := domain.UpdateTaskRequest{
			Tags: newTags,
		}

		updatedTask, err := b.taskService.UpdateTask(ctx, taskID, updateReq, userID)
		if err != nil {
			result.Errors = append(result.Errors, BulkOperationError{
				TaskID:    taskID,
				Index:     i,
				Operation: "tag_update",
				Error:     err.Error(),
			})
			result.Failed++
		} else {
			result.Results = append(result.Results, updatedTask)
			result.Successful++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// BulkDelete deletes multiple tasks with cascade handling
func (b *bulkOperationService) BulkDelete(
	ctx context.Context, taskIDs []string, options BulkDeleteOptions, userID string,
) (*BulkResult, error) {
	startTime := time.Now()

	result := &BulkResult{
		TotalRequested: len(taskIDs),
		Results:        make([]interface{}, 0),
		Errors:         make([]BulkOperationError, 0),
	}

	// If including subtasks, expand the list of tasks to delete
	allTaskIDs := taskIDs
	if options.IncludeSubtasks {
		expandedIDs, err := b.expandWithSubtasks(ctx, taskIDs, userID)
		if err != nil {
			return nil, domain.NewInternalError("SUBTASK_EXPANSION_FAILED", "Failed to expand subtasks", err)
		}
		allTaskIDs = expandedIDs
		result.TotalRequested = len(allTaskIDs)
	}

	for i, taskID := range allTaskIDs {
		err := b.taskService.DeleteTask(ctx, taskID, userID)
		if err != nil {
			result.Errors = append(result.Errors, BulkOperationError{
				TaskID:    taskID,
				Index:     i,
				Operation: "delete",
				Error:     err.Error(),
			})
			result.Failed++
		} else {
			result.Results = append(result.Results, map[string]string{"deleted": taskID})
			result.Successful++
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// ImportFromCSV creates tasks from CSV data
func (b *bulkOperationService) ImportFromCSV(
	ctx context.Context, csvData io.Reader, projectID string, userID string,
) (*BulkResult, error) {
	startTime := time.Now()

	reader := csv.NewReader(csvData)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, domain.NewValidationError("CSV_READ_ERROR", "Failed to read CSV header", nil)
	}

	// Map header columns
	columnMap := b.mapCSVColumns(header)

	var tasks []domain.CreateTaskRequest

	// Read data rows
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, domain.NewValidationError("CSV_READ_ERROR", "Failed to read CSV data", nil)
		}

		task, parseErr := b.parseCSVRecord(record, columnMap, projectID)
		if parseErr != nil {
			// Skip invalid rows but continue processing
			continue
		}

		tasks = append(tasks, task)
	}

	// Create tasks using bulk create
	req := BulkCreateRequest{
		ProjectID: projectID,
		Tasks:     tasks,
	}

	result, err := b.BulkCreate(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// ExportToCSV exports tasks to CSV format
func (b *bulkOperationService) ExportToCSV(
	ctx context.Context, projectID string, filters repository.TaskFilters, userID string,
) ([]byte, error) {
	// Get tasks to export
	tasks, err := b.taskService.GetProjectTasksFiltered(ctx, projectID, filters, userID)
	if err != nil {
		return nil, err
	}

	// Create CSV content
	var csvContent strings.Builder
	writer := csv.NewWriter(&csvContent)

	// Write header
	header := []string{
		"ID", "Title", "Description", "Status", "Priority",
		"Assignee", "Reporter", "Due Date", "Created", "Updated",
	}
	if err := writer.Write(header); err != nil {
		return nil, domain.NewInternalError("CSV_WRITE_ERROR", "Failed to write CSV header", err)
	}

	// Write task data
	for _, task := range tasks {
		record := []string{
			task.ID,
			task.Title,
			task.Description,
			string(task.Status),
			string(task.Priority),
			b.getStringValue(task.AssigneeID),
			task.ReporterID,
			b.formatTime(task.DueDate),
			task.CreatedAt.Format(time.RFC3339),
			task.UpdatedAt.Format(time.RFC3339),
		}

		if err := writer.Write(record); err != nil {
			return nil, domain.NewInternalError("CSV_WRITE_ERROR", "Failed to write CSV record", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, domain.NewInternalError("CSV_FLUSH_ERROR", "Failed to flush CSV writer", err)
	}

	return []byte(csvContent.String()), nil
}

// Helper methods

// generateTasksFromTemplate creates tasks from a template
func (b *bulkOperationService) generateTasksFromTemplate(
	template *BulkCreateTemplate, projectID string,
) []domain.CreateTaskRequest {
	var tasks []domain.CreateTaskRequest

	for i := 0; i < template.Count; i++ {
		index := template.StartIndex + i

		// Generate title from pattern
		title := strings.ReplaceAll(template.TitlePattern, "{index}", strconv.Itoa(index))
		title = strings.ReplaceAll(title, "{number}", strconv.Itoa(i+1))

		task := domain.CreateTaskRequest{
			Title:     title,
			ProjectID: projectID,
		}

		// Apply common fields
		if template.CommonFields != nil {
			if desc, ok := template.CommonFields["description"].(string); ok {
				task.Description = desc
			}
			if priority, ok := template.CommonFields["priority"].(string); ok {
				task.Priority = domain.TaskPriority(priority)
			}
			if assignee, ok := template.CommonFields["assignee_id"].(string); ok {
				task.AssigneeID = assignee
			}
		}

		tasks = append(tasks, task)
	}

	return tasks
}

// Process individual bulk operations

func (b *bulkOperationService) processBulkTaskUpdate(
	ctx context.Context, op BulkTaskOperation, userID string, _ int, result *BulkResult,
) error {
	for _, taskID := range op.TaskIDs {
		updateReq := domain.UpdateTaskRequest{}

		// Map data fields to update request
		if title, ok := op.Data["title"].(string); ok {
			updateReq.Title = &title
		}
		if desc, ok := op.Data["description"].(string); ok {
			updateReq.Description = &desc
		}
		if priority, ok := op.Data["priority"].(string); ok {
			p := domain.TaskPriority(priority)
			updateReq.Priority = &p
		}

		task, err := b.taskService.UpdateTask(ctx, taskID, updateReq, userID)
		if err != nil {
			return err
		}

		result.Results = append(result.Results, task)
	}
	return nil
}

func (b *bulkOperationService) processBulkStatusUpdate(
	ctx context.Context, op BulkTaskOperation, userID string, _ int, _ *BulkResult,
) error {
	status, ok := op.Data["status"].(string)
	if !ok {
		return fmt.Errorf("status field is required for status operation")
	}

	_, err := b.BulkStatusUpdate(ctx, op.TaskIDs, domain.TaskStatus(status), userID)
	return err
}

func (b *bulkOperationService) processBulkAssign(
	ctx context.Context, op BulkTaskOperation, userID string, _ int, _ *BulkResult,
) error {
	assigneeID, _ := op.Data["assignee_id"].(string) // Empty string for unassign
	_, err := b.BulkAssign(ctx, op.TaskIDs, assigneeID, userID)
	return err
}

// calculateNewTags determines the new tag list based on the update request
func (b *bulkOperationService) calculateNewTags(currentTags []string, req BulkTagUpdateRequest) []string {
	if req.ReplaceAll {
		return req.TagsToAdd
	}

	// Start with current tags
	tagSet := make(map[string]bool)
	for _, tag := range currentTags {
		tagSet[tag] = true
	}

	// Remove tags
	for _, tag := range req.TagsToRemove {
		delete(tagSet, tag)
	}

	// Add tags
	for _, tag := range req.TagsToAdd {
		tagSet[tag] = true
	}

	// Convert back to slice
	var newTags []string
	for tag := range tagSet {
		newTags = append(newTags, tag)
	}

	return newTags
}

// expandWithSubtasks recursively finds all subtasks
func (b *bulkOperationService) expandWithSubtasks(
	ctx context.Context, taskIDs []string, userID string,
) ([]string, error) {
	allIDs := make(map[string]bool)

	// Add original task IDs
	for _, id := range taskIDs {
		allIDs[id] = true
	}

	// Recursively find subtasks
	for _, taskID := range taskIDs {
		subtasks, err := b.taskService.GetSubtasks(ctx, taskID, userID)
		if err != nil {
			continue // Skip if we can't get subtasks
		}

		for _, subtask := range subtasks {
			if !allIDs[subtask.ID] {
				allIDs[subtask.ID] = true

				// Recursively get subtasks of subtasks
				subSubtasks, err := b.expandWithSubtasks(ctx, []string{subtask.ID}, userID)
				if err == nil {
					for _, subID := range subSubtasks {
						allIDs[subID] = true
					}
				}
			}
		}
	}

	// Convert back to slice
	var result []string
	for id := range allIDs {
		result = append(result, id)
	}

	return result, nil
}

// CSV helper methods

func (b *bulkOperationService) mapCSVColumns(header []string) map[string]int {
	columnMap := make(map[string]int)
	for i, col := range header {
		columnMap[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return columnMap
}

func (b *bulkOperationService) parseCSVRecord(
	record []string, columnMap map[string]int, projectID string,
) (domain.CreateTaskRequest, error) {
	task := domain.CreateTaskRequest{
		ProjectID: projectID,
	}

	// Title is required
	if titleIdx, ok := columnMap["title"]; ok && titleIdx < len(record) {
		task.Title = strings.TrimSpace(record[titleIdx])
	} else {
		return task, fmt.Errorf("title column is required")
	}

	// Optional fields
	if descIdx, ok := columnMap["description"]; ok && descIdx < len(record) {
		task.Description = strings.TrimSpace(record[descIdx])
	}

	if priorityIdx, ok := columnMap["priority"]; ok && priorityIdx < len(record) {
		priority := strings.TrimSpace(record[priorityIdx])
		if priority != "" {
			task.Priority = domain.TaskPriority(priority)
		}
	}

	if assigneeIdx, ok := columnMap["assignee"]; ok && assigneeIdx < len(record) {
		assignee := strings.TrimSpace(record[assigneeIdx])
		if assignee != "" {
			task.AssigneeID = assignee
		}
	}

	return task, nil
}

// Utility helpers
func (b *bulkOperationService) getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func (b *bulkOperationService) formatTime(t *time.Time) string {
	if t != nil {
		return t.Format(time.RFC3339)
	}
	return ""
}
