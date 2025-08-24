package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type TaskStatus string

const (
	StatusBacklog    TaskStatus = "backlog"
	StatusTodo       TaskStatus = "todo"
	StatusDeveloping TaskStatus = "developing"
	StatusReview     TaskStatus = "review"
	StatusComplete   TaskStatus = "complete"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusBacklog, StatusTodo, StatusDeveloping, StatusReview, StatusComplete:
		return true
	default:
		return false
	}
}

type TaskPriority string

const (
	PriorityCritical TaskPriority = "critical"
	PriorityHigh     TaskPriority = "high"
	PriorityMedium   TaskPriority = "medium"
	PriorityLow      TaskPriority = "low"
)

func (p TaskPriority) IsValid() bool {
	switch p {
	case PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow:
		return true
	default:
		return false
	}
}

type Task struct {
	ID             string          `json:"id" db:"id"`
	Title          string          `json:"title" db:"title"`
	Description    string          `json:"description" db:"description"`
	ProjectID      string          `json:"project_id" db:"project"`
	Status         TaskStatus      `json:"status" db:"status"`
	Priority       TaskPriority    `json:"priority" db:"priority"`
	AssigneeID     *string         `json:"assignee_id" db:"assignee"`
	ReporterID     string          `json:"reporter_id" db:"reporter"`
	ParentTaskID   *string         `json:"parent_task_id" db:"parent_task"`
	Dependencies   []string        `json:"dependencies" db:"-"`
	Tags           []string        `json:"tags" db:"-"`
	DueDate        *time.Time      `json:"due_date" db:"due_date"`
	StartDate      *time.Time      `json:"start_date" db:"start_date"`
	EffortEstimate *float64        `json:"effort_estimate" db:"effort_estimate"`
	TimeSpent      float64         `json:"time_spent" db:"time_spent"`
	Progress       int             `json:"progress" db:"progress"`
	Position       int             `json:"position" db:"position"`
	ColumnPosition json.RawMessage `json:"column_position" db:"column_position"`
	GithubData     json.RawMessage `json:"github_data" db:"github_data"`
	CustomFields   json.RawMessage `json:"custom_fields" db:"custom_fields"`
	Attachments    []string        `json:"attachments" db:"-"`
	CreatedAt      time.Time       `json:"created_at" db:"created"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated"`
}

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

func (t *Task) Validate() error {
	if t.Title == "" {
		return NewValidationError("title", "Title is required", nil)
	}
	if len(t.Title) > 200 {
		return NewValidationError("title", "Title must not exceed 200 characters", nil)
	}
	if t.ProjectID == "" {
		return NewValidationError("project_id", "Project ID is required", nil)
	}
	if t.ReporterID == "" {
		return NewValidationError("reporter_id", "Reporter ID is required", nil)
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
	if t.DueDate != nil && t.DueDate.Before(time.Now().Truncate(24*time.Hour)) {
		return NewValidationError("due_date", "Due date cannot be in the past", nil)
	}
	if t.StartDate != nil && t.DueDate != nil && t.StartDate.After(*t.DueDate) {
		return NewValidationError("dates", "Start date cannot be after due date", nil)
	}
	if t.EffortEstimate != nil && *t.EffortEstimate < 0 {
		return NewValidationError("effort_estimate", "Effort estimate cannot be negative", nil)
	}
	if t.TimeSpent < 0 {
		return NewValidationError("time_spent", "Time spent cannot be negative", nil)
	}
	return nil
}

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

func (t *Task) UpdateStatus(newStatus TaskStatus) error {
	if !t.CanTransitionTo(newStatus) {
		return NewConflictError("invalid_transition",
			fmt.Sprintf("Cannot transition from %s to %s", t.Status, newStatus))
	}
	t.Status = newStatus
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Task) AssignTo(userID string) {
	t.AssigneeID = &userID
	t.UpdatedAt = time.Now()
}

func (t *Task) Unassign() {
	t.AssigneeID = nil
	t.UpdatedAt = time.Now()
}

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

func (t *Task) AddTimeSpent(hours float64) error {
	if hours < 0 {
		return NewValidationError("time_spent", "Time spent cannot be negative", nil)
	}
	t.TimeSpent += hours
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Task) SetParentTask(parentID string) error {
	if parentID == t.ID {
		return NewConflictError("circular_dependency",
			"Task cannot be its own parent")
	}
	t.ParentTaskID = &parentID
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return t.DueDate.Before(time.Now()) && t.Status != StatusComplete
}

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

func (t *Task) SetColumnPosition(positions map[string]int) error {
	data, err := json.Marshal(positions)
	if err != nil {
		return err
	}
	t.ColumnPosition = data
	t.UpdatedAt = time.Now()
	return nil
}

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

func (t *Task) SetCustomFields(fields map[string]interface{}) error {
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	t.CustomFields = data
	t.UpdatedAt = time.Now()
	return nil
}
