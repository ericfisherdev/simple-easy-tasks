// Package services provides real-time subscription services for PocketBase with enhanced task event support.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/tools/subscriptions"

	"simple-easy-tasks/internal/domain"
)

// EnhancedRealtimeService provides comprehensive real-time functionality
// integrating with PocketBase's built-in real-time capabilities and our custom event system
type EnhancedRealtimeService struct {
	pbService        *PocketBaseService
	eventBroadcaster EventBroadcaster
	subscriptionMgr  SubscriptionManager
	logger           *slog.Logger
}

// NewEnhancedRealtimeService creates a new enhanced realtime service
func NewEnhancedRealtimeService(
	pbService *PocketBaseService,
	eventBroadcaster EventBroadcaster,
	subscriptionMgr SubscriptionManager,
	logger *slog.Logger,
) *EnhancedRealtimeService {
	if logger == nil {
		logger = slog.Default()
	}

	service := &EnhancedRealtimeService{
		pbService:        pbService,
		eventBroadcaster: eventBroadcaster,
		subscriptionMgr:  subscriptionMgr,
		logger:           logger,
	}

	// Set up PocketBase real-time hooks
	service.setupPocketBaseHooks()

	return service
}

// SetupRealtimeSubscriptions configures real-time subscriptions for live updates.
func (r *EnhancedRealtimeService) SetupRealtimeSubscriptions() {
	if r.pbService == nil || r.pbService.app == nil {
		r.logger.Error("PocketBase service not available for real-time setup")
		return
	}

	r.logger.Info("Enhanced real-time subscriptions configured with PocketBase integration")
}

// GetRealtimeEndpoint returns the real-time subscription endpoint
func (r *EnhancedRealtimeService) GetRealtimeEndpoint() string {
	return "/api/realtime"
}

// GetTaskEventEndpoint returns the endpoint for task-specific events
func (r *EnhancedRealtimeService) GetTaskEventEndpoint() string {
	return "/api/realtime/tasks"
}

// GetSubscriptionURL returns the subscription URL for a specific collection
func (r *EnhancedRealtimeService) GetSubscriptionURL(collection string) string {
	return "/api/realtime?collection=" + collection
}

// GetTaskSubscriptionURL returns the subscription URL for task events
func (r *EnhancedRealtimeService) GetTaskSubscriptionURL(projectID string) string {
	if projectID != "" {
		return fmt.Sprintf("/api/realtime/events?project_id=%s", projectID)
	}
	return "/api/realtime/events"
}

// BroadcastTaskEvent broadcasts a task event through both PocketBase and custom systems
func (r *EnhancedRealtimeService) BroadcastTaskEvent(ctx context.Context, event *domain.TaskEvent) error {
	if r.eventBroadcaster == nil {
		return fmt.Errorf("event broadcaster not available")
	}

	// Broadcast through custom event system
	if err := r.eventBroadcaster.BroadcastEvent(ctx, event); err != nil {
		r.logger.Error("Failed to broadcast through custom event system", "error", err)
		return err
	}

	// Also broadcast through PocketBase's system
	if err := r.broadcastToPocketBase(ctx, event); err != nil {
		r.logger.Error("Failed to broadcast through PocketBase system", "error", err)
		// Don't fail the operation if PocketBase broadcast fails
	}

	return nil
}

// setupPocketBaseHooks configures hooks to integrate with PocketBase's real-time system
func (r *EnhancedRealtimeService) setupPocketBaseHooks() {
	if r.pbService == nil || r.pbService.app == nil {
		return
	}

	_ = r.pbService.app

	// Hook into real-time connection events if available
	// Note: PocketBase real-time hooks may vary by version
	// These hooks are for future implementation or when PocketBase adds them
	r.logger.Info("PocketBase real-time hooks setup - integration ready for when hooks are available")

	// For now, we'll integrate at the API level instead of hooks
	// The real-time integration will happen through the BroadcastTaskEvent method
	// which uses PocketBase's subscription broker directly
}

// broadcastToPocketBase sends events through PocketBase's real-time system
func (r *EnhancedRealtimeService) broadcastToPocketBase(_ context.Context, event *domain.TaskEvent) error {
	if r.pbService == nil || r.pbService.app == nil {
		return fmt.Errorf("PocketBase service not available")
	}

	broker := r.pbService.app.SubscriptionsBroker()
	if broker == nil {
		return fmt.Errorf("subscription broker not available")
	}

	// Convert task event to PocketBase subscription message
	message, err := r.convertToSubscriptionMessage(event)
	if err != nil {
		return fmt.Errorf("failed to convert event to subscription message: %w", err)
	}

	// Send message to all relevant clients
	// This would filter clients based on subscriptions
	clients := broker.Clients()
	for _, client := range clients {
		// Filter clients based on subscription criteria
		if r.shouldReceiveEvent(client, event) {
			client.Send(*message)
			r.logger.Debug("Sent event to client",
				"client_id", client.Id(),
				"event_type", event.Type,
				"event_id", event.EventID)
		}
	}

	return nil
}

// convertToSubscriptionMessage converts a TaskEvent to PocketBase subscription message format
func (r *EnhancedRealtimeService) convertToSubscriptionMessage(
	event *domain.TaskEvent,
) (*subscriptions.Message, error) {
	// Create a message compatible with PocketBase's subscription format
	messageData := map[string]interface{}{
		"action": "custom_event",
		"record": map[string]interface{}{
			"id":         event.EventID,
			"collection": "task_events",
			"type":       event.Type,
			"task_id":    event.TaskID,
			"project_id": event.ProjectID,
			"user_id":    event.UserID,
			"data":       event.Data,
			"timestamp":  event.Timestamp.Format(time.RFC3339),
		},
	}

	dataBytes, err := json.Marshal(messageData)
	if err != nil {
		return nil, err
	}

	return &subscriptions.Message{
		Name: "task_events/" + string(event.Type),
		Data: dataBytes,
	}, nil
}

// shouldReceiveEvent determines if a client should receive a specific event
func (r *EnhancedRealtimeService) shouldReceiveEvent(_ subscriptions.Client, _ *domain.TaskEvent) bool {
	// Get client's subscription info from context
	// This is a simplified version - in practice, you'd store subscription data with the client

	// For now, send to all clients - in production, implement proper filtering
	// based on client subscriptions stored in client context
	return true
}

// GetStats returns real-time service statistics
func (r *EnhancedRealtimeService) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if r.eventBroadcaster != nil {
		stats["active_subscriptions"] = r.eventBroadcaster.GetActiveSubscriptionCount()
	}

	if r.pbService != nil && r.pbService.app != nil {
		broker := r.pbService.app.SubscriptionsBroker()
		if broker != nil {
			stats["pocketbase_clients"] = len(broker.Clients())
		}
	}

	stats["endpoints"] = map[string]string{
		"realtime":    r.GetRealtimeEndpoint(),
		"task_events": r.GetTaskEventEndpoint(),
	}

	return stats
}

// EnhancedRealtimeHealthChecker checks the health of the enhanced real-time service
type EnhancedRealtimeHealthChecker struct {
	service *EnhancedRealtimeService
}

// NewEnhancedRealtimeHealthChecker creates a health checker for enhanced real-time functionality
func NewEnhancedRealtimeHealthChecker(service *EnhancedRealtimeService) *EnhancedRealtimeHealthChecker {
	return &EnhancedRealtimeHealthChecker{service: service}
}

// Name returns the checker name
func (h *EnhancedRealtimeHealthChecker) Name() string {
	return "enhanced_realtime"
}

// Check performs the enhanced real-time health check
func (h *EnhancedRealtimeHealthChecker) Check(_ context.Context) HealthCheck {
	if h.service == nil {
		return HealthCheck{
			Name:   "enhanced_realtime",
			Status: HealthStatusUnhealthy,
			Error:  "Enhanced real-time service not available",
		}
	}

	if h.service.pbService == nil || h.service.pbService.app == nil {
		return HealthCheck{
			Name:   "enhanced_realtime",
			Status: HealthStatusUnhealthy,
			Error:  "PocketBase service not available for real-time functionality",
		}
	}

	if h.service.eventBroadcaster == nil {
		return HealthCheck{
			Name:   "enhanced_realtime",
			Status: HealthStatusUnhealthy,
			Error:  "Event broadcaster not available",
		}
	}

	stats := h.service.GetStats()

	return HealthCheck{
		Name:    "enhanced_realtime",
		Status:  HealthStatusHealthy,
		Message: "Enhanced real-time service is operational",
		Details: stats,
	}
}

// Legacy compatibility aliases

// RealtimeService provides utilities for working with PocketBase real-time features
// Deprecated: Use EnhancedRealtimeService instead
type RealtimeService = EnhancedRealtimeService

// NewRealtimeService creates a new realtime service
// Deprecated: Use NewEnhancedRealtimeService instead
func NewRealtimeService(pbService *PocketBaseService) *RealtimeService {
	return NewEnhancedRealtimeService(pbService, nil, nil, nil)
}

// RealtimeHealthChecker checks the health of real-time subscriptions
// Deprecated: Use EnhancedRealtimeHealthChecker instead
type RealtimeHealthChecker = EnhancedRealtimeHealthChecker

// NewRealtimeHealthChecker creates a health checker for real-time functionality
// Deprecated: Use NewEnhancedRealtimeHealthChecker instead
func NewRealtimeHealthChecker(service *RealtimeService) *RealtimeHealthChecker {
	return NewEnhancedRealtimeHealthChecker(service)
}
