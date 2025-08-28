package container

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/ericfisherdev/simple-easy-tasks/internal/config"
)

// InitializeServices initializes the service container with all dependencies
func InitializeServices(cfg config.Config, app core.App) (Container, error) {
	container := NewContainer()

	// Register all services
	if err := RegisterServices(container, cfg, app); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return container, nil
}

// GetServiceLocatorContainer returns a container from the global service locator
func GetServiceLocatorContainer() Container {
	return GetInstance().GetContainer()
}

// SetupGlobalContainer initializes the global service locator with services
func SetupGlobalContainer(cfg config.Config, app core.App) error {
	container, err := InitializeServices(cfg, app)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	GetInstance().SetContainer(container)
	return nil
}
