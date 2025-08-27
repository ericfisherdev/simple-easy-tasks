package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"simple-easy-tasks/internal/domain"
)

func TestEventBroadcaster(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Test configuration
	config := EventBroadcasterConfig{
		MaxSubscriptionsPerUser: 5,
		SubscriptionTimeout:     time.Minute,
		EventQueueSize:          10,
		Logger:                  logger,
	}

	// Create event broadcaster
	broadcaster := NewEventBroadcaster(nil, config)

	t.Run("CreateAndRetrieveSubscription", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription
		subscription := domain.NewEventSubscription(
			"user1",
			stringPtr("project1"),
			[]domain.TaskEventType{domain.TaskCreated, domain.TaskUpdated},
		)

		err := broadcaster.Subscribe(ctx, subscription)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Retrieve subscription
		retrieved, err := broadcaster.GetSubscription(ctx, subscription.ID)
		if err != nil {
			t.Fatalf("Failed to get subscription: %v", err)
		}

		if retrieved.ID != subscription.ID {
			t.Errorf("Expected subscription ID %s, got %s", subscription.ID, retrieved.ID)
		}

		if retrieved.UserID != subscription.UserID {
			t.Errorf("Expected user ID %s, got %s", subscription.UserID, retrieved.UserID)
		}
	})

	t.Run("ListUserSubscriptions", func(t *testing.T) {
		ctx := context.Background()
		userID := "user2"

		// Create multiple subscriptions
		subscription1 := domain.NewEventSubscription(userID, nil, []domain.TaskEventType{domain.TaskCreated})
		subscription2 := domain.NewEventSubscription(
			userID, stringPtr("project1"), []domain.TaskEventType{domain.TaskUpdated},
		)

		err := broadcaster.Subscribe(ctx, subscription1)
		if err != nil {
			t.Fatalf("Failed to subscribe 1: %v", err)
		}

		err = broadcaster.Subscribe(ctx, subscription2)
		if err != nil {
			t.Fatalf("Failed to subscribe 2: %v", err)
		}

		// List subscriptions
		subscriptions, err := broadcaster.GetUserSubscriptions(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to get user subscriptions: %v", err)
		}

		if len(subscriptions) != 2 {
			t.Errorf("Expected 2 subscriptions, got %d", len(subscriptions))
		}
	})

	t.Run("BroadcastEvent", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription
		subscription := domain.NewEventSubscription(
			"user3",
			stringPtr("project1"),
			[]domain.TaskEventType{domain.TaskCreated},
		)

		err := broadcaster.Subscribe(ctx, subscription)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Store initial activity time
		initialActivity := subscription.LastActivity

		// Add delay to ensure timestamp difference (local operations are very fast)
		time.Sleep(10 * time.Millisecond)

		// Create test event
		eventData := &domain.TaskCreatedData{
			Task: &domain.Task{
				ID:        "task1",
				Title:     "Test Task",
				ProjectID: "project1",
			},
		}

		event, err := domain.NewTaskEvent(
			domain.TaskCreated,
			"task1",
			"project1",
			"user3",
			eventData,
		)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}

		// Broadcast event
		err = broadcaster.BroadcastEvent(ctx, event)
		if err != nil {
			t.Fatalf("Failed to broadcast event: %v", err)
		}

		// Verify subscription activity was updated
		retrieved, err := broadcaster.GetSubscription(ctx, subscription.ID)
		if err != nil {
			t.Fatalf("Failed to get subscription: %v", err)
		}

		if !retrieved.LastActivity.After(initialActivity) {
			t.Error("Expected subscription activity to be updated")
		}
	})

	t.Run("SubscriptionLimitEnforcement", func(t *testing.T) {
		ctx := context.Background()
		userID := "user4"

		// Create subscriptions up to the limit
		for i := 0; i < config.MaxSubscriptionsPerUser; i++ {
			subscription := domain.NewEventSubscription(
				userID,
				nil,
				[]domain.TaskEventType{domain.TaskCreated},
			)

			err := broadcaster.Subscribe(ctx, subscription)
			if err != nil {
				t.Fatalf("Failed to subscribe %d: %v", i, err)
			}
		}

		// Try to create one more subscription (should fail)
		extraSubscription := domain.NewEventSubscription(
			userID,
			nil,
			[]domain.TaskEventType{domain.TaskCreated},
		)

		err := broadcaster.Subscribe(ctx, extraSubscription)
		if err == nil {
			t.Error("Expected subscription to fail due to limit")
		}
	})

	t.Run("UnsubscribeSubscription", func(t *testing.T) {
		ctx := context.Background()

		// Create subscription
		subscription := domain.NewEventSubscription(
			"user5",
			nil,
			[]domain.TaskEventType{domain.TaskCreated},
		)

		err := broadcaster.Subscribe(ctx, subscription)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Unsubscribe
		err = broadcaster.Unsubscribe(ctx, subscription.ID)
		if err != nil {
			t.Fatalf("Failed to unsubscribe: %v", err)
		}

		// Try to retrieve (should fail)
		_, err = broadcaster.GetSubscription(ctx, subscription.ID)
		if err == nil {
			t.Error("Expected subscription to be deleted")
		}
	})

	t.Run("EventFiltering", func(t *testing.T) {
		ctx := context.Background()

		// Create subscriptions with different filters
		subscription1 := domain.NewEventSubscription(
			"user6",
			stringPtr("project1"),
			[]domain.TaskEventType{domain.TaskCreated},
		)

		subscription2 := domain.NewEventSubscription(
			"user6",
			stringPtr("project2"),
			[]domain.TaskEventType{domain.TaskCreated},
		)

		// Record the original activity times before subscribing
		originalActivity1 := subscription1.LastActivity
		originalActivity2 := subscription2.LastActivity

		err := broadcaster.Subscribe(ctx, subscription1)
		if err != nil {
			t.Fatalf("Failed to subscribe 1: %v", err)
		}

		err = broadcaster.Subscribe(ctx, subscription2)
		if err != nil {
			t.Fatalf("Failed to subscribe 2: %v", err)
		}

		// Create event for project1 (should match subscription1 only)
		eventData := &domain.TaskCreatedData{
			Task: &domain.Task{
				ID:        "task1",
				Title:     "Test Task",
				ProjectID: "project1",
			},
		}

		event, err := domain.NewTaskEvent(
			domain.TaskCreated,
			"task1",
			"project1",
			"user6",
			eventData,
		)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}

		// Add delay to ensure timestamp difference (local operations are very fast)
		time.Sleep(10 * time.Millisecond)

		// Broadcast event
		err = broadcaster.BroadcastEvent(ctx, event)
		if err != nil {
			t.Fatalf("Failed to broadcast event: %v", err)
		}

		// Check that only subscription1 was updated
		sub1After, _ := broadcaster.GetSubscription(ctx, subscription1.ID)
		sub2After, _ := broadcaster.GetSubscription(ctx, subscription2.ID)

		// Subscription1 should have activity after the original time (it was updated)
		if !sub1After.LastActivity.After(originalActivity1) {
			t.Errorf("Expected subscription1 activity to be updated. Original: %s, After broadcast: %s",
				originalActivity1.Format(time.RFC3339Nano),
				sub1After.LastActivity.Format(time.RFC3339Nano))
		}

		// Subscription2 should still have the original activity (it was not updated)
		if !sub2After.LastActivity.Equal(originalActivity2) {
			t.Errorf("Expected subscription2 activity to remain unchanged. Original: %s, After broadcast: %s",
				originalActivity2.Format(time.RFC3339Nano),
				sub2After.LastActivity.Format(time.RFC3339Nano))
		}
	})

	t.Run("GetActiveSubscriptionCount", func(t *testing.T) {
		ctx := context.Background()

		initialCount := broadcaster.GetActiveSubscriptionCount()

		// Create a few subscriptions
		for i := 0; i < 3; i++ {
			subscription := domain.NewEventSubscription(
				"user7",
				nil,
				[]domain.TaskEventType{domain.TaskCreated},
			)

			err := broadcaster.Subscribe(ctx, subscription)
			if err != nil {
				t.Fatalf("Failed to subscribe %d: %v", i, err)
			}
		}

		finalCount := broadcaster.GetActiveSubscriptionCount()
		expectedCount := initialCount + 3

		if finalCount != expectedCount {
			t.Errorf("Expected %d active subscriptions, got %d", expectedCount, finalCount)
		}
	})

	t.Run("CleanupExpiredSubscriptions", func(t *testing.T) {
		ctx := context.Background()

		// Create broadcaster with very short timeout for testing
		shortConfig := config
		shortConfig.SubscriptionTimeout = time.Millisecond * 10

		shortBroadcaster := NewEventBroadcaster(nil, shortConfig)

		// Create subscription
		subscription := domain.NewEventSubscription(
			"user8",
			nil,
			[]domain.TaskEventType{domain.TaskCreated},
		)

		err := shortBroadcaster.Subscribe(ctx, subscription)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Wait for subscription to expire
		time.Sleep(time.Millisecond * 20)

		// Run cleanup
		err = shortBroadcaster.Cleanup(ctx)
		if err != nil {
			t.Fatalf("Failed to cleanup: %v", err)
		}

		// Try to retrieve subscription (should fail)
		_, err = shortBroadcaster.GetSubscription(ctx, subscription.ID)
		if err == nil {
			t.Error("Expected subscription to be cleaned up")
		}
	})
}

func TestEventBroadcasterValidation(t *testing.T) {
	broadcaster := NewEventBroadcaster(nil, EventBroadcasterConfig{})
	ctx := context.Background()

	t.Run("NilEventValidation", func(t *testing.T) {
		err := broadcaster.BroadcastEvent(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil event")
		}
	})

	t.Run("NilSubscriptionValidation", func(t *testing.T) {
		err := broadcaster.Subscribe(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil subscription")
		}
	})

	t.Run("EmptySubscriptionIDValidation", func(t *testing.T) {
		err := broadcaster.Unsubscribe(ctx, "")
		if err == nil {
			t.Error("Expected error for empty subscription ID")
		}
	})

	t.Run("EmptyUserIDValidation", func(t *testing.T) {
		_, err := broadcaster.GetUserSubscriptions(ctx, "")
		if err == nil {
			t.Error("Expected error for empty user ID")
		}
	})
}

func TestEventHandlers(t *testing.T) {
	config := EventBroadcasterConfig{
		MaxSubscriptionsPerUser: 5,
		SubscriptionTimeout:     time.Minute,
		EventQueueSize:          10,
	}

	broadcaster := NewEventBroadcaster(nil, config).(*eventBroadcaster)
	ctx := context.Background()

	t.Run("CustomEventHandler", func(t *testing.T) {
		handlerCalled := false
		var receivedEvent *domain.TaskEvent
		var receivedSubscription *domain.EventSubscription

		// Add custom event handler
		eventHandler := func(event *domain.TaskEvent, subscription *domain.EventSubscription) error {
			handlerCalled = true
			receivedEvent = event
			receivedSubscription = subscription
			return nil
		}

		broadcaster.AddEventHandler(eventHandler)

		// Create subscription
		subscription := domain.NewEventSubscription(
			"user1",
			nil,
			[]domain.TaskEventType{domain.TaskCreated},
		)

		err := broadcaster.Subscribe(ctx, subscription)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Create and broadcast event
		eventData := &domain.TaskCreatedData{
			Task: &domain.Task{
				ID:        "task1",
				Title:     "Test Task",
				ProjectID: "project1",
			},
		}

		event, err := domain.NewTaskEvent(
			domain.TaskCreated,
			"task1",
			"project1",
			"user1",
			eventData,
		)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}

		err = broadcaster.BroadcastEvent(ctx, event)
		if err != nil {
			t.Fatalf("Failed to broadcast event: %v", err)
		}

		// Verify handler was called
		if !handlerCalled {
			t.Error("Expected event handler to be called")
		}

		if receivedEvent == nil || receivedEvent.EventID != event.EventID {
			t.Error("Expected handler to receive the correct event")
		}

		if receivedSubscription == nil || receivedSubscription.ID != subscription.ID {
			t.Error("Expected handler to receive the correct subscription")
		}
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
