package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskEventType(t *testing.T) {
	t.Run("ValidEventTypes", func(t *testing.T) {
		validTypes := []TaskEventType{
			TaskCreated,
			TaskUpdated,
			TaskMoved,
			TaskAssigned,
			TaskDeleted,
			TaskCommented,
		}

		for _, eventType := range validTypes {
			if !eventType.IsValid() {
				t.Errorf("Expected event type %s to be valid", eventType)
			}
		}
	})

	t.Run("InvalidEventType", func(t *testing.T) {
		invalidType := TaskEventType("invalid.event")
		if invalidType.IsValid() {
			t.Error("Expected invalid event type to be invalid")
		}
	})

	t.Run("StringMethod", func(t *testing.T) {
		eventType := TaskCreated
		if eventType.String() != string(TaskCreated) {
			t.Errorf("Expected string representation %s, got %s", TaskCreated, eventType.String())
		}
	})
}

func TestNewTaskEvent(t *testing.T) {
	t.Run("ValidEvent", func(t *testing.T) {
		eventData := &TaskCreatedData{
			Task: &Task{
				ID:        "task1",
				Title:     "Test Task",
				ProjectID: "project1",
			},
		}

		event, err := NewTaskEvent(TaskCreated, "task1", "project1", "user1", eventData)
		if err != nil {
			t.Fatalf("Failed to create task event: %v", err)
		}

		if event.Type != TaskCreated {
			t.Errorf("Expected event type %s, got %s", TaskCreated, event.Type)
		}

		if event.TaskID != "task1" {
			t.Errorf("Expected task ID 'task1', got %s", event.TaskID)
		}

		if event.ProjectID != "project1" {
			t.Errorf("Expected project ID 'project1', got %s", event.ProjectID)
		}

		if event.UserID != "user1" {
			t.Errorf("Expected user ID 'user1', got %s", event.UserID)
		}

		if event.EventID == "" {
			t.Error("Expected event ID to be generated")
		}

		if event.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}

		if len(event.Data) == 0 {
			t.Error("Expected event data to be marshaled")
		}
	})

	t.Run("InvalidEventType", func(t *testing.T) {
		_, err := NewTaskEvent("invalid.event", "task1", "project1", "user1", nil)
		if err == nil {
			t.Error("Expected error for invalid event type")
		}
	})

	t.Run("EmptyTaskID", func(t *testing.T) {
		_, err := NewTaskEvent(TaskCreated, "", "project1", "user1", nil)
		if err == nil {
			t.Error("Expected error for empty task ID")
		}
	})

	t.Run("EmptyProjectID", func(t *testing.T) {
		_, err := NewTaskEvent(TaskCreated, "task1", "", "user1", nil)
		if err == nil {
			t.Error("Expected error for empty project ID")
		}
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		_, err := NewTaskEvent(TaskCreated, "task1", "project1", "", nil)
		if err == nil {
			t.Error("Expected error for empty user ID")
		}
	})

	t.Run("NilData", func(t *testing.T) {
		event, err := NewTaskEvent(TaskCreated, "task1", "project1", "user1", nil)
		if err != nil {
			t.Fatalf("Failed to create task event with nil data: %v", err)
		}

		if len(event.Data) != 0 {
			t.Error("Expected event data to be empty for nil input")
		}
	})

	t.Run("DataMarshalError", func(t *testing.T) {
		// Create data that can't be marshaled (channel)
		invalidData := make(chan int)

		_, err := NewTaskEvent(TaskCreated, "task1", "project1", "user1", invalidData)
		if err == nil {
			t.Error("Expected error for unmarshalable data")
		}
	})
}

func TestTaskEventValidate(t *testing.T) {
	t.Run("ValidEvent", func(t *testing.T) {
		event := &TaskEvent{
			Type:      TaskCreated,
			TaskID:    "task1",
			ProjectID: "project1",
			UserID:    "user1",
			Timestamp: time.Now(),
			EventID:   "event1",
		}

		err := event.Validate()
		if err != nil {
			t.Errorf("Expected valid event to pass validation: %v", err)
		}
	})

	t.Run("InvalidEventType", func(t *testing.T) {
		event := &TaskEvent{
			Type:      "invalid.event",
			TaskID:    "task1",
			ProjectID: "project1",
			UserID:    "user1",
			Timestamp: time.Now(),
			EventID:   "event1",
		}

		err := event.Validate()
		if err == nil {
			t.Error("Expected validation error for invalid event type")
		}
	})

	t.Run("EmptyTaskID", func(t *testing.T) {
		event := &TaskEvent{
			Type:      TaskCreated,
			TaskID:    "",
			ProjectID: "project1",
			UserID:    "user1",
			Timestamp: time.Now(),
			EventID:   "event1",
		}

		err := event.Validate()
		if err == nil {
			t.Error("Expected validation error for empty task ID")
		}
	})

	t.Run("EmptyProjectID", func(t *testing.T) {
		event := &TaskEvent{
			Type:      TaskCreated,
			TaskID:    "task1",
			ProjectID: "",
			UserID:    "user1",
			Timestamp: time.Now(),
			EventID:   "event1",
		}

		err := event.Validate()
		if err == nil {
			t.Error("Expected validation error for empty project ID")
		}
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		event := &TaskEvent{
			Type:      TaskCreated,
			TaskID:    "task1",
			ProjectID: "project1",
			UserID:    "",
			Timestamp: time.Now(),
			EventID:   "event1",
		}

		err := event.Validate()
		if err == nil {
			t.Error("Expected validation error for empty user ID")
		}
	})

	t.Run("ZeroTimestamp", func(t *testing.T) {
		event := &TaskEvent{
			Type:      TaskCreated,
			TaskID:    "task1",
			ProjectID: "project1",
			UserID:    "user1",
			Timestamp: time.Time{},
			EventID:   "event1",
		}

		err := event.Validate()
		if err == nil {
			t.Error("Expected validation error for zero timestamp")
		}
	})

	t.Run("EmptyEventID", func(t *testing.T) {
		event := &TaskEvent{
			Type:      TaskCreated,
			TaskID:    "task1",
			ProjectID: "project1",
			UserID:    "user1",
			Timestamp: time.Now(),
			EventID:   "",
		}

		err := event.Validate()
		if err == nil {
			t.Error("Expected validation error for empty event ID")
		}
	})
}

func TestTaskEventGetDataAs(t *testing.T) {
	t.Run("SuccessfulUnmarshal", func(t *testing.T) {
		originalData := &TaskCreatedData{
			Task: &Task{
				ID:        "task1",
				Title:     "Test Task",
				ProjectID: "project1",
			},
		}

		event, err := NewTaskEvent(TaskCreated, "task1", "project1", "user1", originalData)
		if err != nil {
			t.Fatalf("Failed to create task event: %v", err)
		}

		var retrievedData TaskCreatedData
		err = event.GetDataAs(&retrievedData)
		if err != nil {
			t.Fatalf("Failed to unmarshal event data: %v", err)
		}

		if retrievedData.Task.ID != originalData.Task.ID {
			t.Errorf("Expected task ID %s, got %s", originalData.Task.ID, retrievedData.Task.ID)
		}

		if retrievedData.Task.Title != originalData.Task.Title {
			t.Errorf("Expected task title %s, got %s", originalData.Task.Title, retrievedData.Task.Title)
		}
	})

	t.Run("EmptyData", func(t *testing.T) {
		event := &TaskEvent{
			Data: json.RawMessage{},
		}

		var data TaskCreatedData
		err := event.GetDataAs(&data)
		if err == nil {
			t.Error("Expected error for empty event data")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		event := &TaskEvent{
			Data: json.RawMessage("invalid json"),
		}

		var data TaskCreatedData
		err := event.GetDataAs(&data)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestTaskEventToJSON(t *testing.T) {
	t.Run("ValidJSON", func(t *testing.T) {
		eventData := &TaskCreatedData{
			Task: &Task{
				ID:        "task1",
				Title:     "Test Task",
				ProjectID: "project1",
			},
		}

		event, err := NewTaskEvent(TaskCreated, "task1", "project1", "user1", eventData)
		if err != nil {
			t.Fatalf("Failed to create task event: %v", err)
		}

		jsonBytes, err := event.ToJSON()
		if err != nil {
			t.Fatalf("Failed to convert event to JSON: %v", err)
		}

		if len(jsonBytes) == 0 {
			t.Error("Expected non-empty JSON output")
		}

		// Verify it's valid JSON by unmarshaling it back
		var unmarshaled TaskEvent
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Errorf("Generated JSON is not valid: %v", err)
		}

		if unmarshaled.Type != event.Type {
			t.Errorf("Expected type %s, got %s", event.Type, unmarshaled.Type)
		}

		if unmarshaled.TaskID != event.TaskID {
			t.Errorf("Expected task ID %s, got %s", event.TaskID, unmarshaled.TaskID)
		}
	})
}

func TestEventSubscription(t *testing.T) {
	t.Run("NewEventSubscription", func(t *testing.T) {
		userID := "user1"
		projectID := "project1"
		eventTypes := []TaskEventType{TaskCreated, TaskUpdated}

		subscription := NewEventSubscription(userID, &projectID, eventTypes)

		if subscription.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, subscription.UserID)
		}

		if subscription.ProjectID == nil || *subscription.ProjectID != projectID {
			t.Errorf("Expected project ID %s, got %v", projectID, subscription.ProjectID)
		}

		if len(subscription.EventTypes) != len(eventTypes) {
			t.Errorf("Expected %d event types, got %d", len(eventTypes), len(subscription.EventTypes))
		}

		if subscription.ID == "" {
			t.Error("Expected subscription ID to be generated")
		}

		if !subscription.Active {
			t.Error("Expected subscription to be active by default")
		}
	})

	t.Run("ValidateSubscription", func(t *testing.T) {
		subscription := NewEventSubscription("user1", nil, []TaskEventType{TaskCreated})

		err := subscription.Validate()
		if err != nil {
			t.Errorf("Expected valid subscription to pass validation: %v", err)
		}
	})

	t.Run("ValidateEmptyID", func(t *testing.T) {
		subscription := &EventSubscription{
			ID:         "",
			UserID:     "user1",
			EventTypes: []TaskEventType{TaskCreated},
		}

		err := subscription.Validate()
		if err == nil {
			t.Error("Expected validation error for empty subscription ID")
		}
	})

	t.Run("ValidateEmptyUserID", func(t *testing.T) {
		subscription := &EventSubscription{
			ID:         "sub1",
			UserID:     "",
			EventTypes: []TaskEventType{TaskCreated},
		}

		err := subscription.Validate()
		if err == nil {
			t.Error("Expected validation error for empty user ID")
		}
	})

	t.Run("ValidateNoEventTypes", func(t *testing.T) {
		subscription := &EventSubscription{
			ID:         "sub1",
			UserID:     "user1",
			EventTypes: []TaskEventType{},
		}

		err := subscription.Validate()
		if err == nil {
			t.Error("Expected validation error for no event types")
		}
	})

	t.Run("ValidateInvalidEventType", func(t *testing.T) {
		subscription := &EventSubscription{
			ID:         "sub1",
			UserID:     "user1",
			EventTypes: []TaskEventType{"invalid.event"},
		}

		err := subscription.Validate()
		if err == nil {
			t.Error("Expected validation error for invalid event type")
		}
	})
}

func TestEventSubscriptionMatching(t *testing.T) {
	t.Run("MatchesEventType", func(t *testing.T) {
		subscription := NewEventSubscription(
			"user1",
			nil,
			[]TaskEventType{TaskCreated, TaskUpdated},
		)

		event, _ := NewTaskEvent(TaskCreated, "task1", "project1", "user1", nil)

		if !subscription.MatchesEvent(event) {
			t.Error("Expected subscription to match TaskCreated event")
		}

		event2, _ := NewTaskEvent(TaskDeleted, "task1", "project1", "user1", nil)

		if subscription.MatchesEvent(event2) {
			t.Error("Expected subscription not to match TaskDeleted event")
		}
	})

	t.Run("MatchesProject", func(t *testing.T) {
		projectID := "project1"
		subscription := NewEventSubscription(
			"user1",
			&projectID,
			[]TaskEventType{TaskCreated},
		)

		event1, _ := NewTaskEvent(TaskCreated, "task1", "project1", "user1", nil)
		if !subscription.MatchesEvent(event1) {
			t.Error("Expected subscription to match event from correct project")
		}

		event2, _ := NewTaskEvent(TaskCreated, "task1", "project2", "user1", nil)
		if subscription.MatchesEvent(event2) {
			t.Error("Expected subscription not to match event from different project")
		}
	})

	t.Run("InactiveSubscriptionDoesNotMatch", func(t *testing.T) {
		subscription := NewEventSubscription(
			"user1",
			nil,
			[]TaskEventType{TaskCreated},
		)

		subscription.Active = false

		event, _ := NewTaskEvent(TaskCreated, "task1", "project1", "user1", nil)

		if subscription.MatchesEvent(event) {
			t.Error("Expected inactive subscription not to match any events")
		}
	})

	t.Run("CustomFilters", func(t *testing.T) {
		subscription := NewEventSubscription(
			"user1",
			nil,
			[]TaskEventType{TaskCreated},
		)

		// Add custom filter
		subscription.Filters["user_id"] = "user2"

		event1, _ := NewTaskEvent(TaskCreated, "task1", "project1", "user2", nil)
		if !subscription.MatchesEvent(event1) {
			t.Error("Expected subscription to match event with correct user_id filter")
		}

		event2, _ := NewTaskEvent(TaskCreated, "task1", "project1", "user1", nil)
		if subscription.MatchesEvent(event2) {
			t.Error("Expected subscription not to match event with incorrect user_id filter")
		}
	})

	t.Run("UpdateActivity", func(t *testing.T) {
		subscription := NewEventSubscription(
			"user1",
			nil,
			[]TaskEventType{TaskCreated},
		)

		originalActivity := subscription.LastActivity

		// Wait a small amount to ensure timestamp difference
		time.Sleep(time.Millisecond)

		subscription.UpdateActivity()

		if !subscription.LastActivity.After(originalActivity) {
			t.Error("Expected last activity to be updated")
		}
	})
}

func TestEventDataStructures(t *testing.T) {
	t.Run("TaskCreatedData", func(t *testing.T) {
		task := &Task{
			ID:        "task1",
			Title:     "Test Task",
			ProjectID: "project1",
		}

		data := &TaskCreatedData{
			Task: task,
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("Failed to marshal TaskCreatedData: %v", err)
		}

		var unmarshaled TaskCreatedData
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal TaskCreatedData: %v", err)
		}

		if unmarshaled.Task.ID != task.ID {
			t.Errorf("Expected task ID %s, got %s", task.ID, unmarshaled.Task.ID)
		}
	})

	t.Run("TaskMovedData", func(t *testing.T) {
		task := &Task{
			ID:        "task1",
			Title:     "Test Task",
			ProjectID: "project1",
		}

		data := &TaskMovedData{
			Task:        task,
			OldStatus:   StatusTodo,
			NewStatus:   StatusDeveloping,
			OldPosition: 1,
			NewPosition: 3,
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("Failed to marshal TaskMovedData: %v", err)
		}

		var unmarshaled TaskMovedData
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal TaskMovedData: %v", err)
		}

		if unmarshaled.OldStatus != StatusTodo {
			t.Errorf("Expected old status %s, got %s", StatusTodo, unmarshaled.OldStatus)
		}

		if unmarshaled.NewStatus != StatusDeveloping {
			t.Errorf("Expected new status %s, got %s", StatusDeveloping, unmarshaled.NewStatus)
		}
	})

	t.Run("TaskAssignedData", func(t *testing.T) {
		task := &Task{ID: "task1", Title: "Test Task", ProjectID: "project1"}
		oldAssignee := "user1"
		newAssignee := "user2"

		data := &TaskAssignedData{
			Task:        task,
			OldAssignee: &oldAssignee,
			NewAssignee: &newAssignee,
			AssignedBy:  "user3",
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("Failed to marshal TaskAssignedData: %v", err)
		}

		var unmarshaled TaskAssignedData
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal TaskAssignedData: %v", err)
		}

		if unmarshaled.OldAssignee == nil || *unmarshaled.OldAssignee != oldAssignee {
			t.Errorf("Expected old assignee %s, got %v", oldAssignee, unmarshaled.OldAssignee)
		}

		if unmarshaled.NewAssignee == nil || *unmarshaled.NewAssignee != newAssignee {
			t.Errorf("Expected new assignee %s, got %v", newAssignee, unmarshaled.NewAssignee)
		}
	})
}