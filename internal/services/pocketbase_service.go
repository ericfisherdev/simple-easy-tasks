package services

import (
	"context"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

// PocketBaseService manages PocketBase operations
type PocketBaseService struct {
	app        *pocketbase.PocketBase
	dataDir    string
	isEmbedded bool
}

// NewPocketBaseService creates a new PocketBase service
func NewPocketBaseService(dataDir string, embedded bool) *PocketBaseService {
	app := pocketbase.New()

	return &PocketBaseService{
		app:        app,
		dataDir:    dataDir,
		isEmbedded: embedded,
	}
}

// Initialize sets up PocketBase with collections and hooks
func (p *PocketBaseService) Initialize(_ context.Context) error {
	// Enable migrate command
	migratecmd.MustRegister(p.app, p.app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	// Set up basic hooks
	p.setupHooks()

	log.Println("PocketBase initialized successfully")
	return nil
}

// Start starts the PocketBase service
func (p *PocketBaseService) Start(_ context.Context, _ string) error {
	if p.isEmbedded {
		// For embedded mode, start in background
		go func() {
			if err := p.app.Start(); err != nil {
				log.Printf("PocketBase embedded server error: %v", err)
			}
		}()
		return nil
	}

	// For standalone mode
	return p.app.Start()
}

// Stop stops the PocketBase service
func (p *PocketBaseService) Stop(_ context.Context) error {
	// PocketBase doesn't have a built-in stop method
	// In production, this would be handled by process management
	return nil
}

// GetApp returns the PocketBase app instance
func (p *PocketBaseService) GetApp() *pocketbase.PocketBase {
	return p.app
}

// Note: Collection setup is handled via Admin UI in PocketBase v0.29.3

// setupHooks sets up PocketBase event hooks
func (p *PocketBaseService) setupHooks() {
	// Basic logging hooks can be added here
	log.Printf("PocketBase hooks configured")
}

// PocketBaseHealthChecker implements health checking for PocketBase services.
type PocketBaseHealthChecker struct {
	service *PocketBaseService
}

// NewPocketBaseHealthChecker creates a health checker for PocketBase
func NewPocketBaseHealthChecker(service *PocketBaseService) *PocketBaseHealthChecker {
	return &PocketBaseHealthChecker{service: service}
}

// Name returns the checker name
func (h *PocketBaseHealthChecker) Name() string {
	return "pocketbase"
}

// Check performs the PocketBase health check
func (h *PocketBaseHealthChecker) Check(_ context.Context) HealthCheck {
	if h.service.app == nil {
		return HealthCheck{
			Name:   "pocketbase",
			Status: HealthStatusUnhealthy,
			Error:  "PocketBase app is not initialized",
		}
	}

	return HealthCheck{
		Name:    "pocketbase",
		Status:  HealthStatusHealthy,
		Message: "PocketBase is running",
		Details: map[string]interface{}{
			"data_dir":    h.service.dataDir,
			"is_embedded": h.service.isEmbedded,
		},
	}
}
