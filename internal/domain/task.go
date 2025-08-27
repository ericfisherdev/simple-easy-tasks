package domain

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
)

// TaskStatus represents the current status of a task in the workflow
type TaskStatus string

// Task status constants define the possible states in the kanban workflow
const (
	StatusBacklog    TaskStatus = "backlog"    // StatusBacklog indicates task is in backlog
	StatusTodo       TaskStatus = "todo"       // StatusTodo indicates task is ready to start
	StatusDeveloping TaskStatus = "developing" // StatusDeveloping indicates task is in progress
	StatusReview     TaskStatus = "review"     // StatusReview indicates task is under review
	StatusComplete   TaskStatus = "complete"   // StatusComplete indicates task is finished
)

// IsValid checks if the TaskStatus is one of the allowed values
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusBacklog, StatusTodo, StatusDeveloping, StatusReview, StatusComplete:
		return true
	default:
		return false
	}
}

// TaskPriority represents the importance level of a task
type TaskPriority string

// Task priority constants define the possible importance levels
const (
	PriorityCritical TaskPriority = "critical" // PriorityCritical indicates highest importance
	PriorityHigh     TaskPriority = "high"     // PriorityHigh indicates high importance
	PriorityMedium   TaskPriority = "medium"   // PriorityMedium indicates normal importance
	PriorityLow      TaskPriority = "low"      // PriorityLow indicates lowest importance
)

// IsValid checks if the TaskPriority is one of the allowed values
func (p TaskPriority) IsValid() bool {
	switch p {
	case PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow:
		return true
	default:
		return false
	}
}

// Task validation constraints
const (
	TitleMaxLen    = 200 // TitleMaxLen defines the maximum length for task titles
	ProgressMin    = 0   // ProgressMin defines the minimum progress value
	ProgressMax    = 100 // ProgressMax defines the maximum progress value
	ProgressFull   = 100 // ProgressFull indicates task completion
	MinEffortHours = 0.0 // MinEffortHours defines minimum effort estimate
	MinTimeSpent   = 0.0 // MinTimeSpent defines minimum time spent
)

// Task represents a work item in the project management system
type Task struct {
	UpdatedAt      time.Time       `json:"updated_at" db:"updated"`
	CreatedAt      time.Time       `json:"created_at" db:"created"`
	EffortEstimate *float64        `json:"effort_estimate,omitempty" db:"effort_estimate"`
	AssigneeID     *string         `json:"assignee_id,omitempty" db:"assignee"`
	ParentTaskID   *string         `json:"parent_task_id,omitempty" db:"parent_task"`
	DueDate        *time.Time      `json:"due_date,omitempty" db:"due_date"`
	StartDate      *time.Time      `json:"start_date,omitempty" db:"start_date"`
	ArchivedAt     *time.Time      `json:"archived_at,omitempty" db:"archived_at"`
	Description    string          `json:"description" db:"description"`
	Priority       TaskPriority    `json:"priority" db:"priority"`
	Title          string          `json:"title" db:"title"`
	ID             string          `json:"id" db:"id"`
	ProjectID      string          `json:"project_id" db:"project"`
	ReporterID     string          `json:"reporter_id" db:"reporter"`
	Status         TaskStatus      `json:"status" db:"status"`
	Dependencies   []string        `json:"dependencies,omitempty" db:"-"`
	Tags           []string        `json:"tags,omitempty" db:"-"`
	Attachments    []string        `json:"attachments,omitempty" db:"-"`
	ColumnPosition json.RawMessage `json:"column_position,omitempty" db:"column_position"`
	GithubData     json.RawMessage `json:"github_data,omitempty" db:"github_data"`
	CustomFields   json.RawMessage `json:"custom_fields,omitempty" db:"custom_fields"`
	TimeSpent      float64         `json:"time_spent" db:"time_spent"`
	Progress       int             `json:"progress" db:"progress"`
	Position       int             `json:"position" db:"position"`
	Archived       bool            `json:"archived" db:"archived"`
}

// NewTask creates a new task with default values
func NewTask(title, description, projectID, reporterID string) *Task {
	now := time.Now().UTC()
	// Normalize inputs
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	projectID = strings.TrimSpace(projectID)
	reporterID = strings.TrimSpace(reporterID)
	return &Task{
		Title:       title,
		Description: description,
		ProjectID:   projectID,
		Status:      StatusBacklog,
		Priority:    PriorityMedium,
		ReporterID:  reporterID,
		TimeSpent:   0,
		Progress:    0,
		Position:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate performs comprehensive validation of the task
func (t *Task) Validate() error {
	if err := t.validateRequiredFields(); err != nil {
		return err
	}
	if err := t.validateFieldValues(); err != nil {
		return err
	}
	if err := t.validateDateRules(); err != nil {
		return err
	}
	return t.validateNumericFields()
}

func (t *Task) validateRequiredFields() error {
	if strings.TrimSpace(t.Title) == "" {
		return NewValidationError("title", "Title is required", nil)
	}
	if strings.TrimSpace(t.ProjectID) == "" {
		return NewValidationError("project_id", "Project ID is required", nil)
	}
	if strings.TrimSpace(t.ReporterID) == "" {
		return NewValidationError("reporter_id", "Reporter ID is required", nil)
	}
	return nil
}

func (t *Task) validateFieldValues() error {
	if len(t.Title) > TitleMaxLen {
		return NewValidationError("title", fmt.Sprintf("Title must not exceed %d characters", TitleMaxLen), nil)
	}
	if !t.Status.IsValid() {
		return NewValidationError("status", "Invalid task status", nil)
	}
	if !t.Priority.IsValid() {
		return NewValidationError("priority", "Invalid task priority", nil)
	}
	if t.Progress < ProgressMin || t.Progress > ProgressMax {
		msg := fmt.Sprintf("Progress must be between %d and %d", ProgressMin, ProgressMax)
		return NewValidationError("progress", msg, nil)
	}
	return nil
}

func (t *Task) validateDateRules() error {
	if t.DueDate != nil {
		now := time.Now().UTC()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if t.DueDate.Before(startOfDay) {
			return NewValidationError("due_date", "Due date cannot be in the past", nil)
		}
	}
	if t.StartDate != nil && t.DueDate != nil && t.StartDate.After(*t.DueDate) {
		return NewValidationError("dates", "Start date cannot be after due date", nil)
	}
	return nil
}

func (t *Task) validateNumericFields() error {
	if t.EffortEstimate != nil && *t.EffortEstimate < MinEffortHours {
		return NewValidationError("effort_estimate", "Effort estimate cannot be negative", nil)
	}
	if t.TimeSpent < MinTimeSpent {
		return NewValidationError("time_spent", "Time spent cannot be negative", nil)
	}
	if t.Position < 0 {
		return NewValidationError("position", "Position cannot be negative", nil)
	}
	return nil
}

// CanTransitionTo checks if the task can transition to the specified status
func (t *Task) CanTransitionTo(newStatus TaskStatus) bool {
	if !newStatus.IsValid() {
		return false
	}

	transitions := map[TaskStatus][]TaskStatus{
		StatusBacklog:    {StatusTodo, StatusDeveloping},
		StatusTodo:       {StatusBacklog, StatusDeveloping},
		StatusDeveloping: {StatusTodo, StatusReview, StatusBacklog},
		StatusReview:     {StatusDeveloping, StatusComplete, StatusTodo},
		StatusComplete:   {StatusReview, StatusTodo},
	}

	allowedStatuses, exists := transitions[t.Status]
	if !exists {
		return false
	}

	for _, status := range allowedStatuses {
		if status == newStatus {
			return true
		}
	}

	return false
}

// UpdateStatus transitions the task to a new status if allowed
func (t *Task) UpdateStatus(newStatus TaskStatus) error {
	if !newStatus.IsValid() {
		return NewValidationError("status", "Invalid status provided", map[string]interface{}{
			"provided_status": string(newStatus),
			"valid_statuses": []string{
				string(StatusBacklog), string(StatusTodo), string(StatusDeveloping),
				string(StatusReview), string(StatusComplete),
			},
		})
	}

	if !t.CanTransitionTo(newStatus) {
		transitions := map[TaskStatus][]TaskStatus{
			StatusBacklog:    {StatusTodo, StatusDeveloping},
			StatusTodo:       {StatusBacklog, StatusDeveloping},
			StatusDeveloping: {StatusTodo, StatusReview, StatusBacklog},
			StatusReview:     {StatusDeveloping, StatusComplete, StatusTodo},
			StatusComplete:   {StatusReview, StatusTodo},
		}

		allowedStatuses, exists := transitions[t.Status]
		var allowedStatusStrings []string
		if exists {
			for _, status := range allowedStatuses {
				allowedStatusStrings = append(allowedStatusStrings, string(status))
			}
		}

		conflictErr := NewConflictError("invalid_transition",
			fmt.Sprintf("Cannot transition from %s to %s", t.Status, newStatus))
		conflictErr.Details = map[string]interface{}{
			"current_status":      string(t.Status),
			"requested_status":    string(newStatus),
			"allowed_transitions": allowedStatusStrings,
		}
		return conflictErr
	}

	previousStatus := t.Status
	t.Status = newStatus
	t.UpdatedAt = time.Now().UTC()

	// Log successful transition for audit trail
	slog.Debug("Task status transition completed",
		"task_id", t.ID,
		"from_status", string(previousStatus),
		"to_status", string(newStatus),
		"timestamp", t.UpdatedAt,
	)

	return nil
}

// AssignTo assigns the task to a specific user
func (t *Task) AssignTo(userID string) {
	t.AssigneeID = &userID
	t.UpdatedAt = time.Now().UTC()
}

// Unassign removes the current assignee from the task
func (t *Task) Unassign() {
	t.AssigneeID = nil
	t.UpdatedAt = time.Now().UTC()
}

// UpdateProgress updates the task completion percentage
func (t *Task) UpdateProgress(progress int) error {
	if progress < ProgressMin || progress > ProgressMax {
		msg := fmt.Sprintf("Progress must be between %d and %d", ProgressMin, ProgressMax)
		return NewValidationError("progress", msg, nil)
	}
	t.Progress = progress
	t.UpdatedAt = time.Now().UTC()

	// If progress reaches 100%, attempt automatic status transition with proper error handling
	if progress == ProgressFull && t.Status != StatusComplete {
		if err := t.UpdateStatus(StatusComplete); err != nil {
			// Log the transition failure but don't fail the progress update
			// This maintains backward compatibility while providing visibility
			slog.Warn("Automatic status transition failed after progress completion",
				"task_id", t.ID,
				"current_status", string(t.Status),
				"attempted_status", string(StatusComplete),
				"error", err.Error(),
				"progress", progress,
			)

			// Optionally, we could add the failed transition to task metadata
			// for later analysis or manual intervention

			// Parse existing CustomFields or create empty map
			var existingData map[string]interface{}
			if len(t.CustomFields) > 0 {
				// Try to unmarshal existing data
				if unmarshalErr := json.Unmarshal(t.CustomFields, &existingData); unmarshalErr != nil {
					// If unmarshal fails, start with empty map
					existingData = make(map[string]interface{})
				}
			} else {
				existingData = make(map[string]interface{})
			}

			// Add transition failure metadata to existing data
			existingData["failed_auto_transition"] = map[string]interface{}{
				"timestamp":        time.Now().UTC(),
				"from_status":      string(t.Status),
				"to_status":        string(StatusComplete),
				"trigger":          "progress_completion",
				"error":            err.Error(),
				"progress_at_time": progress,
			}

			// Marshal the merged data back to CustomFields
			if mergedData, err := json.Marshal(existingData); err == nil {
				t.CustomFields = mergedData
			}
			// If marshal fails, preserve the original CustomFields
		}
	}
	return nil
}

// AddTimeSpent adds hours to the time spent on the task
func (t *Task) AddTimeSpent(hours float64) error {
	if hours < MinTimeSpent {
		return NewValidationError("time_spent", "Time spent cannot be negative", nil)
	}
	if math.IsNaN(hours) || math.IsInf(hours, 0) {
		return NewValidationError("time_spent", "Time spent must be a finite number", nil)
	}
	t.TimeSpent += hours
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// SetParentTask sets the parent task for subtask relationships
// Note: This only validates immediate self-parent cycles. For broader cycle detection
// and cross-project validation, additional checks should be performed at the service layer
// with repository access to traverse the full parent chain via DFS.
func (t *Task) SetParentTask(parentID string) error {
	if parentID == t.ID {
		return NewConflictError("circular_dependency",
			"Task cannot be its own parent")
	}
	// TODO: Implement broader cycle detection and cross-project validation
	// at the service layer with repository access
	t.ParentTaskID = &parentID
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// IsOverdue checks if the task is past its due date and not complete
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return t.DueDate.Before(time.Now()) && t.Status != StatusComplete
}

// Archive archives the task by setting archived flag and timestamp
func (t *Task) Archive() {
	now := time.Now().UTC()
	t.Archived = true
	t.ArchivedAt = &now
	t.UpdatedAt = now
}

// Unarchive unarchives the task by clearing archived flag and timestamp
func (t *Task) Unarchive() {
	t.Archived = false
	t.ArchivedAt = nil
	t.UpdatedAt = time.Now().UTC()
}

// IsArchived returns whether the task is archived
func (t *Task) IsArchived() bool {
	return t.Archived
}

// GetColumnPositionMap retrieves the column positions as a map
func (t *Task) GetColumnPositionMap() (map[string]int, error) {
	if len(t.ColumnPosition) == 0 {
		return make(map[string]int), nil
	}

	var positions map[string]int
	if err := json.Unmarshal(t.ColumnPosition, &positions); err != nil {
		return nil, err
	}
	if positions == nil {
		positions = make(map[string]int)
	}
	return positions, nil
}

// SetColumnPosition stores the column positions as JSON
func (t *Task) SetColumnPosition(positions map[string]int) error {
	data, err := json.Marshal(positions)
	if err != nil {
		return err
	}
	t.ColumnPosition = data
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// GetCustomFieldsMap retrieves the custom fields as a map
func (t *Task) GetCustomFieldsMap() (map[string]interface{}, error) {
	if len(t.CustomFields) == 0 {
		return make(map[string]interface{}), nil
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(t.CustomFields, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = make(map[string]interface{})
	}
	return fields, nil
}

// SetCustomFields stores the custom fields as JSON
func (t *Task) SetCustomFields(fields map[string]interface{}) error {
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	t.CustomFields = data
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// CreateTaskRequest represents the data needed to create a new task.
type CreateTaskRequest struct {
	DueDate     *time.Time             `json:"due_date,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Title       string                 `json:"title" binding:"required,min=1,max=200"`
	Description string                 `json:"description,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	AssigneeID  string                 `json:"assignee_id,omitempty"`
	Priority    TaskPriority           `json:"priority,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// Validate validates the create task request.
func (r *CreateTaskRequest) Validate() error {
	if err := ValidateRequired("title", r.Title, "INVALID_TITLE", "Task title is required"); err != nil {
		return err
	}

	if err := ValidateRequired("project_id", r.ProjectID, "INVALID_PROJECT_ID", "Project ID is required"); err != nil {
		return err
	}

	return nil
}

// UpdateTaskRequest represents the data that can be updated for a task.
type UpdateTaskRequest struct {
	Title       *string                `json:"title,omitempty"`
	Description *string                `json:"description,omitempty"`
	AssigneeID  *string                `json:"assignee_id,omitempty"`
	Status      *TaskStatus            `json:"status,omitempty"`
	Priority    *TaskPriority          `json:"priority,omitempty"`
	DueDate     *time.Time             `json:"due_date,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}
