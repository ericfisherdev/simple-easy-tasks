package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type TaskHistoryAction string

const (
	ActionCreated   TaskHistoryAction = "created"
	ActionUpdated   TaskHistoryAction = "updated"
	ActionMoved     TaskHistoryAction = "moved"
	ActionAssigned  TaskHistoryAction = "assigned"
	ActionCommented TaskHistoryAction = "commented"
	ActionDeleted   TaskHistoryAction = "deleted"
)

func (a TaskHistoryAction) IsValid() bool {
	switch a {
	case ActionCreated, ActionUpdated, ActionMoved, ActionAssigned, ActionCommented, ActionDeleted:
		return true
	default:
		return false
	}
}

type TaskHistoryEntry struct {
	ID        string            `json:"id" db:"id"`
	TaskID    string            `json:"task_id" db:"task"`
	UserID    string            `json:"user_id" db:"user"`
	Action    TaskHistoryAction `json:"action" db:"action"`
	FieldName *string           `json:"field_name" db:"field_name"`
	OldValue  json.RawMessage   `json:"old_value" db:"old_value"`
	NewValue  json.RawMessage   `json:"new_value" db:"new_value"`
	Metadata  json.RawMessage   `json:"metadata" db:"metadata"`
	CreatedAt time.Time         `json:"created_at" db:"created"`
}

func NewTaskHistoryEntry(taskID, userID string, action TaskHistoryAction) *TaskHistoryEntry {
	return &TaskHistoryEntry{
		TaskID:    taskID,
		UserID:    userID,
		Action:    action,
		CreatedAt: time.Now(),
	}
}

func (h *TaskHistoryEntry) Validate() error {
	if h.TaskID == "" {
		return NewValidationError("task_id", "Task ID is required", nil)
	}
	if h.UserID == "" {
		return NewValidationError("user_id", "User ID is required", nil)
	}
	if !h.Action.IsValid() {
		return NewValidationError("action", "Invalid task history action", nil)
	}
	return nil
}

func (h *TaskHistoryEntry) SetFieldChange(fieldName string, oldValue, newValue interface{}) error {
	h.FieldName = &fieldName

	oldJSON, err := json.Marshal(oldValue)
	if err != nil {
		return err
	}
	h.OldValue = oldJSON

	newJSON, err := json.Marshal(newValue)
	if err != nil {
		return err
	}
	h.NewValue = newJSON

	return nil
}

func (h *TaskHistoryEntry) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	h.Metadata = data
	return nil
}

func (h *TaskHistoryEntry) GetOldValue() (interface{}, error) {
	if h.OldValue == nil {
		return nil, nil
	}
	var value interface{}
	if err := json.Unmarshal(h.OldValue, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (h *TaskHistoryEntry) GetNewValue() (interface{}, error) {
	if h.NewValue == nil {
		return nil, nil
	}
	var value interface{}
	if err := json.Unmarshal(h.NewValue, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (h *TaskHistoryEntry) GetMetadata() (map[string]interface{}, error) {
	if h.Metadata == nil {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(h.Metadata, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

type TaskHistoryFilter struct {
	TaskID    *string             `json:"task_id"`
	UserID    *string             `json:"user_id"`
	Actions   []TaskHistoryAction `json:"actions"`
	FieldName *string             `json:"field_name"`
	StartDate *time.Time          `json:"start_date"`
	EndDate   *time.Time          `json:"end_date"`
	Limit     int                 `json:"limit"`
	Offset    int                 `json:"offset"`
}

func (f *TaskHistoryFilter) Validate() error {
	if f.StartDate != nil && f.EndDate != nil && f.StartDate.After(*f.EndDate) {
		return NewValidationError("dates", "Start date cannot be after end date", nil)
	}
	if f.Limit < 0 {
		return NewValidationError("limit", "Limit cannot be negative", nil)
	}
	if f.Offset < 0 {
		return NewValidationError("offset", "Offset cannot be negative", nil)
	}
	for _, action := range f.Actions {
		if !action.IsValid() {
			return NewValidationError("actions", fmt.Sprintf("Invalid action in filter: %s", action), nil)
		}
	}
	return nil
}
