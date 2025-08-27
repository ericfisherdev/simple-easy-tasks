package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskEventType represents the type of task event that occurred
type TaskEventType string

// Task event type constants define the possible real-time event types
const (
	TaskCreated   TaskEventType = "task.created"   // TaskCreated indicates a new task was created
	TaskUpdated   TaskEventType = "task.updated"   // TaskUpdated indicates an existing task was modified
	TaskMoved     TaskEventType = "task.moved"     // TaskMoved indicates a task was moved between columns/statuses
	TaskAssigned  TaskEventType = "task.assigned"  // TaskAssigned indicates a task was assigned to someone
	TaskDeleted   TaskEventType = "task.deleted"   // TaskDeleted indicates a task was removed
	TaskCommented TaskEventType = "task.commented" // TaskCommented indicates a comment was added to a task
)

// IsValid checks if the TaskEventType is one of the allowed values
func (t TaskEventType) IsValid() bool {
	switch t {
	case TaskCreated, TaskUpdated, TaskMoved, TaskAssigned, TaskDeleted, TaskCommented:
		return true
	default:
		return false
	}
}

// String returns the string representation of the event type
func (t TaskEventType) String() string {
	return string(t)
}

// TaskEvent represents a real-time event related to task operations
// This structure is used for broadcasting changes to all connected clients
type TaskEvent struct {
	Type      TaskEventType   `json:"type"`       // Type specifies what kind of event occurred
	TaskID    string          `json:"task_id"`    // TaskID identifies the task that was affected
	ProjectID string          `json:"project_id"` // ProjectID identifies the project containing the task
	UserID    string          `json:"user_id"`    // UserID identifies who triggered the event
	Data      json.RawMessage `json:"data"`       // Data contains event-specific payload
	Timestamp time.Time       `json:"timestamp"`  // Timestamp when the event occurred
	EventID   string          `json:"event_id"`   // EventID provides unique identifier for the event
}

// NewTaskEvent creates a new TaskEvent with the specified details
func NewTaskEvent(eventType TaskEventType, taskID, projectID, userID string, data interface{}) (*TaskEvent, error) {
	if !eventType.IsValid() {
		return nil, NewValidationError("INVALID_EVENT_TYPE", "Invalid task event type", nil)
	}

	if taskID == "" {
		return nil, NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	if projectID == "" {
		return nil, NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	if userID == "" {
		return nil, NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	// Marshal data to JSON
	var jsonData json.RawMessage
	if data != nil {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return nil, NewInternalError("DATA_MARSHAL_FAILED", "Failed to marshal event data", err)
		}
		jsonData = dataBytes
	}

	return &TaskEvent{
		Type:      eventType,
		TaskID:    taskID,
		ProjectID: projectID,
		UserID:    userID,
		Data:      jsonData,
		Timestamp: time.Now().UTC(),
		EventID:   generateEventID(),
	}, nil
}

// Validate ensures the TaskEvent has all required fields
func (e *TaskEvent) Validate() error {
	if !e.Type.IsValid() {
		return NewValidationError("INVALID_EVENT_TYPE", "Invalid task event type", nil)
	}

	if e.TaskID == "" {
		return NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	if e.ProjectID == "" {
		return NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	if e.UserID == "" {
		return NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	if e.Timestamp.IsZero() {
		return NewValidationError("INVALID_TIMESTAMP", "Event timestamp cannot be zero", nil)
	}

	if e.EventID == "" {
		return NewValidationError("INVALID_EVENT_ID", "Event ID cannot be empty", nil)
	}

	return nil
}

// GetDataAs unmarshals the event data into the provided structure
func (e *TaskEvent) GetDataAs(target interface{}) error {
	if len(e.Data) == 0 {
		return NewValidationError("NO_EVENT_DATA", "Event has no data to unmarshal", nil)
	}

	if err := json.Unmarshal(e.Data, target); err != nil {
		return NewInternalError("DATA_UNMARSHAL_FAILED", "Failed to unmarshal event data", err)
	}

	return nil
}

// ToJSON converts the TaskEvent to JSON bytes for transmission
func (e *TaskEvent) ToJSON() ([]byte, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, NewInternalError("EVENT_MARSHAL_FAILED", "Failed to marshal task event", err)
	}
	return data, nil
}

// TaskEventData contains common data structures for different event types

// TaskCreatedData contains data for task creation events
type TaskCreatedData struct {
	Task *Task `json:"task"`
}

// TaskUpdatedData contains data for task update events
type TaskUpdatedData struct {
	Task      *Task                  `json:"task"`
	Changes   map[string]interface{} `json:"changes,omitempty"` // Fields that were changed
	OldValues map[string]interface{} `json:"old_values,omitempty"` // Previous values for changed fields
}

// TaskMovedData contains data for task move events (status/position changes)
type TaskMovedData struct {
	Task        *Task      `json:"task"`
	OldStatus   TaskStatus `json:"old_status"`
	NewStatus   TaskStatus `json:"new_status"`
	OldPosition int        `json:"old_position"`
	NewPosition int        `json:"new_position"`
}

// TaskAssignedData contains data for task assignment events
type TaskAssignedData struct {
	Task         *Task   `json:"task"`
	OldAssignee  *string `json:"old_assignee,omitempty"`
	NewAssignee  *string `json:"new_assignee,omitempty"`
	AssignedBy   string  `json:"assigned_by"`
}

// TaskDeletedData contains data for task deletion events
type TaskDeletedData struct {
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	DeletedBy string `json:"deleted_by"`
}

// TaskCommentedData contains data for task comment events
type TaskCommentedData struct {
	Task      *Task   `json:"task"`
	CommentID string  `json:"comment_id"`
	Comment   string  `json:"comment"`
	Author    string  `json:"author"`
}

// generateEventID creates a unique identifier for events
// In production, consider using more sophisticated ID generation
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// EventSubscription represents a client subscription to task events
type EventSubscription struct {
	ID           string            `json:"id"`            // Unique subscription identifier
	UserID       string            `json:"user_id"`       // User who created the subscription
	ProjectID    *string           `json:"project_id"`    // Optional project filter
	EventTypes   []TaskEventType   `json:"event_types"`   // Types of events to receive
	Filters      map[string]string `json:"filters"`       // Additional filters (assignee, status, etc.)
	CreatedAt    time.Time         `json:"created_at"`    // When subscription was created
	LastActivity time.Time         `json:"last_activity"` // Last time subscription received an event
	Active       bool              `json:"active"`        // Whether subscription is active
}

// NewEventSubscription creates a new event subscription
func NewEventSubscription(userID string, projectID *string, eventTypes []TaskEventType) *EventSubscription {
	return &EventSubscription{
		ID:           generateSubscriptionID(),
		UserID:       userID,
		ProjectID:    projectID,
		EventTypes:   eventTypes,
		Filters:      make(map[string]string),
		CreatedAt:    time.Now().UTC(),
		LastActivity: time.Now().UTC(),
		Active:       true,
	}
}

// Validate ensures the EventSubscription has valid configuration
func (s *EventSubscription) Validate() error {
	if s.ID == "" {
		return NewValidationError("INVALID_SUBSCRIPTION_ID", "Subscription ID cannot be empty", nil)
	}

	if s.UserID == "" {
		return NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	if len(s.EventTypes) == 0 {
		return NewValidationError("NO_EVENT_TYPES", "At least one event type must be specified", nil)
	}

	for _, eventType := range s.EventTypes {
		if !eventType.IsValid() {
			return NewValidationError("INVALID_EVENT_TYPE", fmt.Sprintf("Invalid event type: %s", eventType), nil)
		}
	}

	return nil
}

// MatchesEvent checks if a subscription should receive a specific event
func (s *EventSubscription) MatchesEvent(event *TaskEvent) bool {
	if !s.Active {
		return false
	}

	// Check if subscription is interested in this event type
	eventTypeMatch := false
	for _, eventType := range s.EventTypes {
		if eventType == event.Type {
			eventTypeMatch = true
			break
		}
	}
	if !eventTypeMatch {
		return false
	}

	// Check project filter
	if s.ProjectID != nil && *s.ProjectID != event.ProjectID {
		return false
	}

	// Apply additional filters
	for key, value := range s.Filters {
		if !s.matchesFilter(key, value, event) {
			return false
		}
	}

	return true
}

// matchesFilter checks if an event matches a specific filter criteria
func (s *EventSubscription) matchesFilter(key, value string, event *TaskEvent) bool {
	switch key {
	case "user_id":
		return event.UserID == value
	case "task_id":
		return event.TaskID == value
	// Add more filters as needed
	default:
		return true // Unknown filters are ignored
	}
}

// UpdateActivity updates the last activity timestamp
func (s *EventSubscription) UpdateActivity() {
	s.LastActivity = time.Now().UTC()
}

// generateSubscriptionID creates a unique identifier for subscriptions
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}