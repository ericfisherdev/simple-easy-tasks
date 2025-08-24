// Package services provides real-time subscription services for PocketBase.
package services

import (
	"context"
	"log"
)

// SetupRealtimeSubscriptions configures real-time subscriptions for live updates.
// Currently exported but not used - kept for future implementation.
func (p *PocketBaseService) SetupRealtimeSubscriptions() {
	// PocketBase provides real-time subscriptions via Server-Sent Events (SSE) out of the box
	// The API endpoint /api/realtime provides WebSocket-like functionality over SSE
	log.Printf("Real-time subscriptions configured - handled automatically by PocketBase")
}

// RealtimeService provides utilities for working with PocketBase real-time features
type RealtimeService struct {
	pbService *PocketBaseService
}

// NewRealtimeService creates a new realtime service
func NewRealtimeService(pbService *PocketBaseService) *RealtimeService {
	return &RealtimeService{
		pbService: pbService,
	}
}

// GetRealtimeEndpoint returns the real-time subscription endpoint
func (r *RealtimeService) GetRealtimeEndpoint() string {
	// PocketBase provides real-time subscriptions at /api/realtime
	return "/api/realtime"
}

// GetSubscriptionURL returns the subscription URL for a specific collection
func (r *RealtimeService) GetSubscriptionURL(collection string) string {
	return "/api/realtime?collection=" + collection
}

// RealtimeHealthChecker checks the health of real-time subscriptions
type RealtimeHealthChecker struct {
	service *RealtimeService
}

// NewRealtimeHealthChecker creates a health checker for real-time functionality
func NewRealtimeHealthChecker(service *RealtimeService) *RealtimeHealthChecker {
	return &RealtimeHealthChecker{service: service}
}

// Name returns the checker name
func (h *RealtimeHealthChecker) Name() string {
	return "realtime"
}

// Check performs the real-time health check
func (h *RealtimeHealthChecker) Check(_ context.Context) HealthCheck {
	if h.service.pbService == nil || h.service.pbService.app == nil {
		return HealthCheck{
			Name:   "realtime",
			Status: HealthStatusUnhealthy,
			Error:  "PocketBase service not available for real-time functionality",
		}
	}

	return HealthCheck{
		Name:    "realtime",
		Status:  HealthStatusHealthy,
		Message: "Real-time subscriptions are available",
		Details: map[string]interface{}{
			"endpoint":     h.service.GetRealtimeEndpoint(),
			"subscription": h.service.GetSubscriptionURL("*"),
		},
	}
}
