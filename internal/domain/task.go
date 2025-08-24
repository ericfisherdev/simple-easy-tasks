package domain

import (
	"encoding/json"
	"fmt"
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

// Task represents a work item in the project management system
type Task struct {
	UpdatedAt      time.Time       `json:"updated_at" db:"updated"`
	CreatedAt      time.Time       `json:"created_at" db:"created"`
	EffortEstimate *float64        `json:"effort_estimate" db:"effort_estimate"`
	AssigneeID     *string         `json:"assignee_id" db:"assignee"`
	ParentTaskID   *string         `json:"parent_task_id" db:"parent_task"`
	DueDate        *time.Time      `json:"due_date" db:"due_date"`
	StartDate      *time.Time      `json:"start_date" db:"start_date"`
	Description    string          `json:"description" db:"description"`
	Priority       TaskPriority    `json:"priority" db:"priority"`
	Title          string          `json:"title" db:"title"`
	ID             string          `json:"id" db:"id"`
	ProjectID      string          `json:"project_id" db:"project"`
	ReporterID     string          `json:"reporter_id" db:"reporter"`
	Status         TaskStatus      `json:"status" db:"status"`
	Dependencies   []string        `json:"dependencies" db:"-"`
	Tags           []string        `json:"tags" db:"-"`
	Attachments    []string        `json:"attachments" db:"-"`
	ColumnPosition json.RawMessage `json:"column_position" db:"column_position"`
	GithubData     json.RawMessage `json:"github_data" db:"github_data"`
	CustomFields   json.RawMessage `json:"custom_fields" db:"custom_fields"`
	TimeSpent      float64         `json:"time_spent" db:"time_spent"`
	Progress       int             `json:"progress" db:"progress"`
	Position       int             `json:"position" db:"position"`
}

// NewTask creates a new task with default values
func NewTask(title, description, projectID, reporterID string) *Task {
	now := time.Now()
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
	if t.Title == "" {
		return NewValidationError("title", "Title is required", nil)
	}
	if t.ProjectID == "" {
		return NewValidationError("project_id", "Project ID is required", nil)
	}
	if t.ReporterID == "" {
		return NewValidationError("reporter_id", "Reporter ID is required", nil)
	}
	return nil
}

func (t *Task) validateFieldValues() error {
	if len(t.Title) > 200 {
		return NewValidationError("title", "Title must not exceed 200 characters", nil)
	}
	if !t.Status.IsValid() {
		return NewValidationError("status", "Invalid task status", nil)
	}
	if !t.Priority.IsValid() {
		return NewValidationError("priority", "Invalid task priority", nil)
	}
	if t.Progress < 0 || t.Progress > 100 {
		return NewValidationError("progress", "Progress must be between 0 and 100", nil)
	}
	return nil
}

func (t *Task) validateDateRules() error {
	if t.DueDate != nil && t.DueDate.Before(time.Now().Truncate(24*time.Hour)) {
		return NewValidationError("due_date", "Due date cannot be in the past", nil)
	}
	if t.StartDate != nil && t.DueDate != nil && t.StartDate.After(*t.DueDate) {
		return NewValidationError("dates", "Start date cannot be after due date", nil)
	}
	return nil
}

func (t *Task) validateNumericFields() error {
	if t.EffortEstimate != nil && *t.EffortEstimate < 0 {
		return NewValidationError("effort_estimate", "Effort estimate cannot be negative", nil)
	}
	if t.TimeSpent < 0 {
		return NewValidationError("time_spent", "Time spent cannot be negative", nil)
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
	if !t.CanTransitionTo(newStatus) {
		return NewConflictError("invalid_transition",
			fmt.Sprintf("Cannot transition from %s to %s", t.Status, newStatus))
	}
	t.Status = newStatus
	t.UpdatedAt = time.Now()
	return nil
}

// AssignTo assigns the task to a specific user
func (t *Task) AssignTo(userID string) {
	t.AssigneeID = &userID
	t.UpdatedAt = time.Now()
}

// Unassign removes the current assignee from the task
func (t *Task) Unassign() {
	t.AssigneeID = nil
	t.UpdatedAt = time.Now()
}

// UpdateProgress updates the task completion percentage
func (t *Task) UpdateProgress(progress int) error {
	if progress < 0 || progress > 100 {
		return NewValidationError("progress", "Progress must be between 0 and 100", nil)
	}
	t.Progress = progress
	t.UpdatedAt = time.Now()

	if progress == 100 && t.Status != StatusComplete {
		return t.UpdateStatus(StatusComplete)
	}
	return nil
}

// AddTimeSpent adds hours to the time spent on the task
func (t *Task) AddTimeSpent(hours float64) error {
	if hours < 0 {
		return NewValidationError("time_spent", "Time spent cannot be negative", nil)
	}
	t.TimeSpent += hours
	t.UpdatedAt = time.Now()
	return nil
}

// SetParentTask sets the parent task for subtask relationships
func (t *Task) SetParentTask(parentID string) error {
	if parentID == t.ID {
		return NewConflictError("circular_dependency",
			"Task cannot be its own parent")
	}
	t.ParentTaskID = &parentID
	t.UpdatedAt = time.Now()
	return nil
}

// IsOverdue checks if the task is past its due date and not complete
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return t.DueDate.Before(time.Now()) && t.Status != StatusComplete
}

// GetColumnPositionMap retrieves the column positions as a map
func (t *Task) GetColumnPositionMap() (map[string]int, error) {
	if t.ColumnPosition == nil {
		return make(map[string]int), nil
	}

	var positions map[string]int
	if err := json.Unmarshal(t.ColumnPosition, &positions); err != nil {
		return nil, err
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
	t.UpdatedAt = time.Now()
	return nil
}

// GetCustomFieldsMap retrieves the custom fields as a map
func (t *Task) GetCustomFieldsMap() (map[string]interface{}, error) {
	if t.CustomFields == nil {
		return make(map[string]interface{}), nil
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(t.CustomFields, &fields); err != nil {
		return nil, err
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
	t.UpdatedAt = time.Now()
	return nil
}
