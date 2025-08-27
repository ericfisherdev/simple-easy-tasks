package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// Mock task service for testing
type mockTaskService struct {
	tasks       map[string]*domain.Task
	nextID      int
	createError error
	updateError error
	deleteError error
}

func (m *mockTaskService) CreateTask(ctx context.Context, req domain.CreateTaskRequest, userID string) (*domain.Task, error) {
	if m.createError != nil {
		return nil, m.createError
	}

	m.nextID++
	task := &domain.Task{
		ID:          generateTaskID(m.nextID),
		Title:       req.Title,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		ReporterID:  userID,
		Status:      domain.StatusTodo,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		Tags:        req.Tags,
		Position:    m.nextID,
		Progress:    0,
		TimeSpent:   0.0,
	}

	if req.AssigneeID != "" {
		task.AssigneeID = &req.AssigneeID
	}

	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockTaskService) GetTask(ctx context.Context, taskID string, userID string) (*domain.Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}
	return task, nil
}

func (m *mockTaskService) UpdateTask(ctx context.Context, taskID string, req domain.UpdateTaskRequest, userID string) (*domain.Task, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// Apply updates
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		task.AssigneeID = req.AssigneeID
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.Tags != nil {
		task.Tags = req.Tags
	}

	return task, nil
}

func (m *mockTaskService) DeleteTask(ctx context.Context, taskID string, userID string) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	_, exists := m.tasks[taskID]
	if !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	delete(m.tasks, taskID)
	return nil
}

func (m *mockTaskService) ListProjectTasks(ctx context.Context, projectID string, userID string, offset, limit int) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, task := range m.tasks {
		if task.ProjectID == projectID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *mockTaskService) ListUserTasks(ctx context.Context, userID string, offset, limit int) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, task := range m.tasks {
		if task.AssigneeID != nil && *task.AssigneeID == userID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *mockTaskService) AssignTask(ctx context.Context, taskID string, assigneeID string, userID string) (*domain.Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.AssigneeID = &assigneeID
	return task, nil
}

func (m *mockTaskService) UnassignTask(ctx context.Context, taskID string, userID string) (*domain.Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.AssigneeID = nil
	return task, nil
}

func (m *mockTaskService) UpdateTaskStatus(ctx context.Context, taskID string, status domain.TaskStatus, userID string) (*domain.Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.Status = status
	return task, nil
}

func (m *mockTaskService) MoveTask(ctx context.Context, req MoveTaskRequest, userID string) error {
	task, exists := m.tasks[req.TaskID]
	if !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.Status = req.NewStatus
	task.Position = req.NewPosition
	return nil
}

// Additional methods required by TaskService interface
func (m *mockTaskService) GetProjectTasksFiltered(ctx context.Context, projectID string, filters repository.TaskFilters, userID string) ([]*domain.Task, error) {
	return m.ListProjectTasks(ctx, projectID, userID, 0, 100)
}

func (m *mockTaskService) GetSubtasks(ctx context.Context, parentTaskID string, userID string) ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (m *mockTaskService) GetTaskDependencies(ctx context.Context, taskID string, userID string) ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (m *mockTaskService) DuplicateTask(ctx context.Context, taskID string, options DuplicationOptions, userID string) (*domain.Task, error) {
	return nil, nil
}

func (m *mockTaskService) CreateFromTemplate(ctx context.Context, templateID string, projectID string, userID string) (*domain.Task, error) {
	return nil, nil
}

func (m *mockTaskService) CreateSubtask(ctx context.Context, parentTaskID string, req domain.CreateTaskRequest, userID string) (*domain.Task, error) {
	return m.CreateTask(ctx, req, userID)
}

func (m *mockTaskService) AddDependency(ctx context.Context, taskID string, dependencyID string, userID string) error {
	return nil
}

func (m *mockTaskService) RemoveDependency(ctx context.Context, taskID string, dependencyID string, userID string) error {
	return nil
}

func generateTaskID(id int) string {
	return "task_" + string(rune('0'+id))
}

// Mock event broadcaster for testing
type mockEventBroadcaster struct {
	broadcastedEvents []*domain.TaskEvent
	broadcastError    error
}

func (m *mockEventBroadcaster) BroadcastEvent(ctx context.Context, event *domain.TaskEvent) error {
	if m.broadcastError != nil {
		return m.broadcastError
	}

	m.broadcastedEvents = append(m.broadcastedEvents, event)
	return nil
}

func (m *mockEventBroadcaster) Subscribe(ctx context.Context, subscription *domain.EventSubscription) error {
	return nil
}

func (m *mockEventBroadcaster) Unsubscribe(ctx context.Context, subscriptionID string) error {
	return nil
}

func (m *mockEventBroadcaster) GetSubscription(ctx context.Context, subscriptionID string) (*domain.EventSubscription, error) {
	return nil, nil
}

func (m *mockEventBroadcaster) GetUserSubscriptions(ctx context.Context, userID string) ([]*domain.EventSubscription, error) {
	return []*domain.EventSubscription{}, nil
}

func (m *mockEventBroadcaster) GetActiveSubscriptionCount() int {
	return 0
}

func (m *mockEventBroadcaster) Cleanup(ctx context.Context) error {
	return nil
}

func TestRealtimeTaskService(t *testing.T) {
	// Set up mocks
	baseService := &mockTaskService{
		tasks: make(map[string]*domain.Task),
	}

	eventBroadcaster := &mockEventBroadcaster{}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create realtime task service
	realtimeService := NewRealtimeTaskService(baseService, eventBroadcaster, logger)

	t.Run("CreateTaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		req := domain.CreateTaskRequest{
			Title:       "Test Task",
			Description: "Test Description",
			ProjectID:   "project1",
			Priority:    domain.PriorityMedium,
		}

		task, err := realtimeService.CreateTask(ctx, req, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		if task == nil {
			t.Fatal("Expected task to be created")
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskCreated {
			t.Errorf("Expected event type %s, got %s", domain.TaskCreated, event.Type)
		}

		if event.TaskID != task.ID {
			t.Errorf("Expected event task ID %s, got %s", task.ID, event.TaskID)
		}

		if event.ProjectID != req.ProjectID {
			t.Errorf("Expected event project ID %s, got %s", req.ProjectID, event.ProjectID)
		}
	})

	t.Run("UpdateTaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create initial task
		createReq := domain.CreateTaskRequest{
			Title:     "Original Title",
			ProjectID: "project1",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Update task
		newTitle := "Updated Title"
		updateReq := domain.UpdateTaskRequest{
			Title: &newTitle,
		}

		updatedTask, err := realtimeService.UpdateTask(ctx, task.ID, updateReq, "user1")
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		if updatedTask.Title != newTitle {
			t.Errorf("Expected title %s, got %s", newTitle, updatedTask.Title)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskUpdated {
			t.Errorf("Expected event type %s, got %s", domain.TaskUpdated, event.Type)
		}
	})

	t.Run("DeleteTaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create task to delete
		createReq := domain.CreateTaskRequest{
			Title:     "Task to Delete",
			ProjectID: "project1",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Delete task
		err = realtimeService.DeleteTask(ctx, task.ID, "user1")
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskDeleted {
			t.Errorf("Expected event type %s, got %s", domain.TaskDeleted, event.Type)
		}
	})

	t.Run("AssignTaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create task
		createReq := domain.CreateTaskRequest{
			Title:     "Task to Assign",
			ProjectID: "project1",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Assign task
		assignedTask, err := realtimeService.AssignTask(ctx, task.ID, "user2", "user1")
		if err != nil {
			t.Fatalf("Failed to assign task: %v", err)
		}

		if assignedTask.AssigneeID == nil || *assignedTask.AssigneeID != "user2" {
			t.Errorf("Expected assignee user2, got %v", assignedTask.AssigneeID)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskAssigned {
			t.Errorf("Expected event type %s, got %s", domain.TaskAssigned, event.Type)
		}
	})

	t.Run("UnassignTaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create and assign task
		createReq := domain.CreateTaskRequest{
			Title:      "Task to Unassign",
			ProjectID:  "project1",
			AssigneeID: "user2",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Unassign task
		unassignedTask, err := realtimeService.UnassignTask(ctx, task.ID, "user1")
		if err != nil {
			t.Fatalf("Failed to unassign task: %v", err)
		}

		if unassignedTask.AssigneeID != nil {
			t.Errorf("Expected assignee to be nil, got %v", unassignedTask.AssigneeID)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskAssigned {
			t.Errorf("Expected event type %s, got %s", domain.TaskAssigned, event.Type)
		}
	})

	t.Run("UpdateTaskStatusBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create task
		createReq := domain.CreateTaskRequest{
			Title:     "Task to Update Status",
			ProjectID: "project1",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Update status
		updatedTask, err := realtimeService.UpdateTaskStatus(ctx, task.ID, domain.StatusDeveloping, "user1")
		if err != nil {
			t.Fatalf("Failed to update task status: %v", err)
		}

		if updatedTask.Status != domain.StatusDeveloping {
			t.Errorf("Expected status %s, got %s", domain.StatusDeveloping, updatedTask.Status)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskUpdated {
			t.Errorf("Expected event type %s, got %s", domain.TaskUpdated, event.Type)
		}
	})

	t.Run("MoveTaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create task
		createReq := domain.CreateTaskRequest{
			Title:     "Task to Move",
			ProjectID: "project1",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Move task
		moveReq := MoveTaskRequest{
			TaskID:      task.ID,
			ProjectID:   "project1",
			NewStatus:   domain.StatusReview,
			NewPosition: 5,
		}

		err = realtimeService.MoveTask(ctx, moveReq, "user1")
		if err != nil {
			t.Fatalf("Failed to move task: %v", err)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskMoved {
			t.Errorf("Expected event type %s, got %s", domain.TaskMoved, event.Type)
		}
	})

	t.Run("CreateSubtaskBroadcastsEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create parent task
		createReq := domain.CreateTaskRequest{
			Title:     "Parent Task",
			ProjectID: "project1",
		}

		parentTask, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create parent task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Create subtask
		subtaskReq := domain.CreateTaskRequest{
			Title:       "Subtask",
			Description: "Subtask description",
			ProjectID:   "project1",
		}

		subtask, err := realtimeService.CreateSubtask(ctx, parentTask.ID, subtaskReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create subtask: %v", err)
		}

		if subtask == nil {
			t.Fatal("Expected subtask to be created")
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskCreated {
			t.Errorf("Expected event type %s, got %s", domain.TaskCreated, event.Type)
		}
	})

	t.Run("BroadcastTaskCommented", func(t *testing.T) {
		ctx := context.Background()

		// Create task
		createReq := domain.CreateTaskRequest{
			Title:     "Task for Comment",
			ProjectID: "project1",
		}

		task, err := realtimeService.CreateTask(ctx, createReq, "user1")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Clear previous events
		eventBroadcaster.broadcastedEvents = []*domain.TaskEvent{}

		// Broadcast comment event
		err = realtimeService.BroadcastTaskCommented(ctx, task, "comment1", "This is a test comment", "user2")
		if err != nil {
			t.Fatalf("Failed to broadcast comment event: %v", err)
		}

		// Verify event was broadcasted
		if len(eventBroadcaster.broadcastedEvents) != 1 {
			t.Errorf("Expected 1 broadcasted event, got %d", len(eventBroadcaster.broadcastedEvents))
		}

		event := eventBroadcaster.broadcastedEvents[0]
		if event.Type != domain.TaskCommented {
			t.Errorf("Expected event type %s, got %s", domain.TaskCommented, event.Type)
		}

		if event.UserID != "user2" {
			t.Errorf("Expected event user ID user2, got %s", event.UserID)
		}
	})

	t.Run("EventBroadcastFailureDoesNotFailOperation", func(t *testing.T) {
		// Set up broadcaster to fail
		failingBroadcaster := &mockEventBroadcaster{
			broadcastError: domain.NewInternalError("BROADCAST_FAILED", "Broadcast failed", nil),
		}

		failingRealtimeService := NewRealtimeTaskService(baseService, failingBroadcaster, logger)

		ctx := context.Background()

		req := domain.CreateTaskRequest{
			Title:     "Task with Failing Broadcast",
			ProjectID: "project1",
		}

		// Task creation should still succeed even if broadcast fails
		task, err := failingRealtimeService.CreateTask(ctx, req, "user1")
		if err != nil {
			t.Fatalf("Expected task creation to succeed despite broadcast failure: %v", err)
		}

		if task == nil {
			t.Fatal("Expected task to be created")
		}
	})

	t.Run("GetEventBroadcaster", func(t *testing.T) {
		broadcaster := realtimeService.GetEventBroadcaster()
		if broadcaster == nil {
			t.Fatal("Expected event broadcaster to be available")
		}

		if broadcaster != eventBroadcaster {
			t.Error("Expected to get the same event broadcaster instance")
		}
	})
}

func TestCalculateTaskChanges(t *testing.T) {
	baseService := &mockTaskService{tasks: make(map[string]*domain.Task)}
	eventBroadcaster := &mockEventBroadcaster{}
	logger := slog.Default()

	realtimeService := NewRealtimeTaskService(baseService, eventBroadcaster, logger).(*realtimeTaskService)

	t.Run("NoChanges", func(t *testing.T) {
		task1 := &domain.Task{
			ID:          "task1",
			Title:       "Same Title",
			Description: "Same Description",
			Status:      domain.StatusTodo,
			Priority:    domain.PriorityMedium,
			Position:    1,
			Progress:    50,
			TimeSpent:   10.5,
		}

		task2 := &domain.Task{
			ID:          "task1",
			Title:       "Same Title",
			Description: "Same Description",
			Status:      domain.StatusTodo,
			Priority:    domain.PriorityMedium,
			Position:    1,
			Progress:    50,
			TimeSpent:   10.5,
		}

		changes, oldValues := realtimeService.calculateTaskChanges(task1, task2)

		if len(changes) != 0 {
			t.Errorf("Expected no changes, got %v", changes)
		}

		if len(oldValues) != 0 {
			t.Errorf("Expected no old values, got %v", oldValues)
		}
	})

	t.Run("TitleChange", func(t *testing.T) {
		task1 := &domain.Task{
			ID:    "task1",
			Title: "Old Title",
		}

		task2 := &domain.Task{
			ID:    "task1",
			Title: "New Title",
		}

		changes, oldValues := realtimeService.calculateTaskChanges(task1, task2)

		if changes["title"] != "New Title" {
			t.Errorf("Expected title change to 'New Title', got %v", changes["title"])
		}

		if oldValues["title"] != "Old Title" {
			t.Errorf("Expected old title value 'Old Title', got %v", oldValues["title"])
		}
	})

	t.Run("MultipleChanges", func(t *testing.T) {
		assignee1 := "user1"
		assignee2 := "user2"

		task1 := &domain.Task{
			ID:          "task1",
			Title:       "Old Title",
			Status:      domain.StatusTodo,
			Priority:    domain.PriorityLow,
			AssigneeID:  &assignee1,
			Position:    1,
		}

		task2 := &domain.Task{
			ID:          "task1",
			Title:       "New Title",
			Status:      domain.StatusDeveloping,
			Priority:    domain.PriorityHigh,
			AssigneeID:  &assignee2,
			Position:    3,
		}

		changes, oldValues := realtimeService.calculateTaskChanges(task1, task2)

		expectedChanges := []string{"title", "status", "priority", "assignee_id", "position"}

		for _, field := range expectedChanges {
			if _, exists := changes[field]; !exists {
				t.Errorf("Expected change for field %s", field)
			}
			if _, exists := oldValues[field]; !exists {
				t.Errorf("Expected old value for field %s", field)
			}
		}
	})

	t.Run("AssigneeChanges", func(t *testing.T) {
		assignee := "user1"

		// Test nil to assigned
		task1 := &domain.Task{ID: "task1", AssigneeID: nil}
		task2 := &domain.Task{ID: "task1", AssigneeID: &assignee}

		changes, oldValues := realtimeService.calculateTaskChanges(task1, task2)

		if changes["assignee_id"] != &assignee {
			t.Errorf("Expected assignee_id change to %s, got %v", assignee, changes["assignee_id"])
		}

		if oldValues["assignee_id"] != nil {
			t.Errorf("Expected old assignee_id to be nil, got %v", oldValues["assignee_id"])
		}

		// Test assigned to nil
		task3 := &domain.Task{ID: "task1", AssigneeID: &assignee}
		task4 := &domain.Task{ID: "task1", AssigneeID: nil}

		changes, oldValues = realtimeService.calculateTaskChanges(task3, task4)

		if changes["assignee_id"] != nil {
			t.Errorf("Expected assignee_id change to nil, got %v", changes["assignee_id"])
		}

		if oldValues["assignee_id"] == nil || *oldValues["assignee_id"].(*string) != assignee {
			t.Errorf("Expected old assignee_id to be %s, got %v", assignee, oldValues["assignee_id"])
		}
	})

	t.Run("DateChanges", func(t *testing.T) {
		now := time.Now()
		later := now.Add(time.Hour)

		// Test nil to date
		task1 := &domain.Task{ID: "task1", DueDate: nil}
		task2 := &domain.Task{ID: "task1", DueDate: &now}

		changes, oldValues := realtimeService.calculateTaskChanges(task1, task2)

		if changes["due_date"] != &now {
			t.Errorf("Expected due_date change to %v, got %v", now, changes["due_date"])
		}

		if oldValues["due_date"] != nil {
			t.Errorf("Expected old due_date to be nil, got %v", oldValues["due_date"])
		}

		// Test date to different date
		task3 := &domain.Task{ID: "task1", DueDate: &now}
		task4 := &domain.Task{ID: "task1", DueDate: &later}

		changes, oldValues = realtimeService.calculateTaskChanges(task3, task4)

		if changes["due_date"] != &later {
			t.Errorf("Expected due_date change to %v, got %v", later, changes["due_date"])
		}

		if oldValues["due_date"] == nil || !oldValues["due_date"].(*time.Time).Equal(now) {
			t.Errorf("Expected old due_date to be %v, got %v", now, oldValues["due_date"])
		}
	})

	t.Run("TagsChanges", func(t *testing.T) {
		task1 := &domain.Task{
			ID:   "task1",
			Tags: []string{"tag1", "tag2"},
		}

		task2 := &domain.Task{
			ID:   "task1",
			Tags: []string{"tag1", "tag3"},
		}

		changes, oldValues := realtimeService.calculateTaskChanges(task1, task2)

		if len(changes) == 0 {
			t.Error("Expected tags change to be detected")
		}

		if changes["tags"] == nil {
			t.Error("Expected tags field in changes")
		}

		if oldValues["tags"] == nil {
			t.Error("Expected tags field in old values")
		}
	})
}