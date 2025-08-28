package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// Mock implementations for testing
type mockProjectRepository struct {
	projects map[string]*domain.Project
}

func (m *mockProjectRepository) GetByID(_ context.Context, id string) (*domain.Project, error) {
	project, exists := m.projects[id]
	if !exists {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}
	return project, nil
}

type mockUserRepository struct {
	users map[string]*domain.User
}

func (m *mockUserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}
	return user, nil
}

func TestSubscriptionManager(t *testing.T) {
	// Set up mocks
	projectRepo := &mockProjectRepository{
		projects: map[string]*domain.Project{
			"project1": {
				ID:        "project1",
				Title:     "Test Project 1",
				OwnerID:   "user1",
				MemberIDs: []string{"user1", "user2"},
			},
			"project2": {
				ID:        "project2",
				Title:     "Test Project 2",
				OwnerID:   "user2",
				MemberIDs: []string{"user2"},
			},
		},
	}

	userRepo := &mockUserRepository{
		users: map[string]*domain.User{
			"user1": {
				ID:       "user1",
				Username: "testuser1",
				Email:    "user1@example.com",
			},
			"user2": {
				ID:       "user2",
				Username: "testuser2",
				Email:    "user2@example.com",
			},
			"user_delete_test": {
				ID:       "user_delete_test",
				Username: "testuser_delete",
				Email:    "delete@example.com",
			},
		},
	}

	// Create event broadcaster
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	broadcaster := NewEventBroadcaster(nil, EventBroadcasterConfig{
		MaxSubscriptionsPerUser: 5,
		SubscriptionTimeout:     time.Hour,
		EventQueueSize:          10,
		Logger:                  logger,
	})

	// Create subscription manager
	manager := NewSubscriptionManager(
		broadcaster,
		projectRepo,
		userRepo,
		SubscriptionManagerConfig{
			Logger:          logger,
			CleanupInterval: time.Minute,
		},
	)

	t.Run("CreateSubscription", func(t *testing.T) {
		ctx := context.Background()

		req := CreateSubscriptionRequest{
			UserID:     "user1",
			ProjectID:  stringPtr("project1"),
			EventTypes: []domain.TaskEventType{domain.TaskCreated, domain.TaskUpdated},
		}

		subscription, err := manager.CreateSubscription(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create subscription: %v", err)
		}

		if subscription.UserID != req.UserID {
			t.Errorf("Expected user ID %s, got %s", req.UserID, subscription.UserID)
		}

		if subscription.ProjectID == nil || *subscription.ProjectID != *req.ProjectID {
			t.Errorf("Expected project ID %s, got %v", *req.ProjectID, subscription.ProjectID)
		}

		if len(subscription.EventTypes) != len(req.EventTypes) {
			t.Errorf("Expected %d event types, got %d", len(req.EventTypes), len(subscription.EventTypes))
		}
	})

	t.Run("CreateSubscriptionWithoutProject", func(t *testing.T) {
		ctx := context.Background()

		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		subscription, err := manager.CreateSubscription(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create subscription: %v", err)
		}

		if subscription.ProjectID != nil {
			t.Errorf("Expected project ID to be nil, got %v", subscription.ProjectID)
		}
	})

	t.Run("CreateSubscriptionInvalidUser", func(t *testing.T) {
		ctx := context.Background()

		req := CreateSubscriptionRequest{
			UserID:     "nonexistent",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		_, err := manager.CreateSubscription(ctx, req)
		if err == nil {
			t.Error("Expected error for nonexistent user")
		}
	})

	t.Run("CreateSubscriptionInvalidProject", func(t *testing.T) {
		ctx := context.Background()

		req := CreateSubscriptionRequest{
			UserID:     "user1",
			ProjectID:  stringPtr("nonexistent"),
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		_, err := manager.CreateSubscription(ctx, req)
		if err == nil {
			t.Error("Expected error for nonexistent project")
		}
	})

	t.Run("CreateSubscriptionNoProjectAccess", func(t *testing.T) {
		ctx := context.Background()

		req := CreateSubscriptionRequest{
			UserID:     "user1",
			ProjectID:  stringPtr("project2"), // user1 doesn't have access to project2
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		_, err := manager.CreateSubscription(ctx, req)
		if err == nil {
			t.Error("Expected error for no project access")
		}
	})

	t.Run("ListUserSubscriptions", func(t *testing.T) {
		ctx := context.Background()

		// Create multiple subscriptions for user2
		req1 := CreateSubscriptionRequest{
			UserID:     "user2",
			ProjectID:  stringPtr("project2"),
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		req2 := CreateSubscriptionRequest{
			UserID:     "user2",
			EventTypes: []domain.TaskEventType{domain.TaskUpdated},
		}

		sub1, err := manager.CreateSubscription(ctx, req1)
		if err != nil {
			t.Fatalf("Failed to create subscription 1: %v", err)
		}

		sub2, err := manager.CreateSubscription(ctx, req2)
		if err != nil {
			t.Fatalf("Failed to create subscription 2: %v", err)
		}

		// List subscriptions
		subscriptions, err := manager.ListUserSubscriptions(ctx, "user2")
		if err != nil {
			t.Fatalf("Failed to list user subscriptions: %v", err)
		}

		if len(subscriptions) < 2 {
			t.Errorf("Expected at least 2 subscriptions, got %d", len(subscriptions))
		}

		// Verify subscription IDs are in the list
		found1, found2 := false, false
		for _, sub := range subscriptions {
			if sub.ID == sub1.ID {
				found1 = true
			}
			if sub.ID == sub2.ID {
				found2 = true
			}
		}

		if !found1 || !found2 {
			t.Error("Expected to find both created subscriptions in the list")
		}
	})

	t.Run("GetSubscription", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription
		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		created, err := manager.CreateSubscription(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create subscription: %v", err)
		}

		// Get subscription
		retrieved, err := manager.GetSubscription(ctx, created.ID, "user1")
		if err != nil {
			t.Fatalf("Failed to get subscription: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected subscription ID %s, got %s", created.ID, retrieved.ID)
		}
	})

	t.Run("GetSubscriptionAccessDenied", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription for user1
		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		created, err := manager.CreateSubscription(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create subscription: %v", err)
		}

		// Try to get subscription as user2 (should fail)
		_, err = manager.GetSubscription(ctx, created.ID, "user2")
		if err == nil {
			t.Error("Expected access denied error")
		}
	})

	t.Run("UpdateSubscription", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription
		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		created, err := manager.CreateSubscription(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create subscription: %v", err)
		}

		// Update subscription
		updateReq := UpdateSubscriptionRequest{
			EventTypes: &[]domain.TaskEventType{domain.TaskCreated, domain.TaskUpdated, domain.TaskMoved},
			Active:     boolPtr(false),
		}

		updated, err := manager.UpdateSubscription(ctx, created.ID, updateReq)
		if err != nil {
			t.Fatalf("Failed to update subscription: %v", err)
		}

		if len(updated.EventTypes) != 3 {
			t.Errorf("Expected 3 event types after update, got %d", len(updated.EventTypes))
		}

		if updated.Active {
			t.Error("Expected subscription to be inactive after update")
		}
	})

	t.Run("DeleteSubscription", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription - use a unique user to avoid subscription limit
		req := CreateSubscriptionRequest{
			UserID:     "user_delete_test",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		created, err := manager.CreateSubscription(ctx, req)
		if err != nil {
			t.Fatalf("Failed to create subscription: %v", err)
		}

		// Delete subscription
		err = manager.DeleteSubscription(ctx, created.ID, "user_delete_test")
		if err != nil {
			t.Fatalf("Failed to delete subscription: %v", err)
		}

		// Try to get subscription (should fail)
		_, err = manager.GetSubscription(ctx, created.ID, "user_delete_test")
		if err == nil {
			t.Error("Expected subscription to be deleted")
		}
	})

	t.Run("CleanupRoutine", func(_ *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		// Start cleanup routine
		manager.StartCleanupRoutine(ctx, time.Millisecond*100)

		// Wait for a few cleanup cycles
		time.Sleep(time.Millisecond * 300)

		// The cleanup routine should run without errors
		// This is mainly testing that the routine starts and runs
	})
}

func TestCreateSubscriptionRequestValidation(t *testing.T) {
	t.Run("ValidRequest", func(t *testing.T) {
		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		err := req.Validate()
		if err != nil {
			t.Errorf("Expected valid request to pass validation: %v", err)
		}
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		req := CreateSubscriptionRequest{
			UserID:     "",
			EventTypes: []domain.TaskEventType{domain.TaskCreated},
		}

		err := req.Validate()
		if err == nil {
			t.Error("Expected validation error for empty user ID")
		}
	})

	t.Run("NoEventTypes", func(t *testing.T) {
		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{},
		}

		err := req.Validate()
		if err == nil {
			t.Error("Expected validation error for no event types")
		}
	})

	t.Run("InvalidEventType", func(t *testing.T) {
		req := CreateSubscriptionRequest{
			UserID:     "user1",
			EventTypes: []domain.TaskEventType{"invalid.event"},
		}

		err := req.Validate()
		if err == nil {
			t.Error("Expected validation error for invalid event type")
		}
	})
}

func TestSubscriptionFilter(t *testing.T) {
	t.Run("BuildFilters", func(t *testing.T) {
		filters := NewSubscriptionFilter().
			ByTaskID("task123").
			ByUserID("user456").
			ByAssignee("assignee789").
			ByStatus(domain.StatusTodo).
			Build()

		expected := map[string]string{
			"task_id":     "task123",
			"user_id":     "user456",
			"assignee_id": "assignee789",
			"status":      "todo",
		}

		for key, expectedValue := range expected {
			if filters[key] != expectedValue {
				t.Errorf("Expected filter %s to be %s, got %s", key, expectedValue, filters[key])
			}
		}
	})

	t.Run("EmptyFilters", func(t *testing.T) {
		filters := NewSubscriptionFilter().Build()

		if len(filters) != 0 {
			t.Errorf("Expected empty filters, got %v", filters)
		}
	})
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
