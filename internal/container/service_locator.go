package container

import (
	"context"
	"sync"
)

// ServiceLocator provides a global access point to the DI container
type ServiceLocator struct {
	container Container
	mu        sync.RWMutex
}

var (
	instance *ServiceLocator
	once     sync.Once
)

// GetInstance returns the singleton instance of ServiceLocator
func GetInstance() *ServiceLocator {
	once.Do(func() {
		instance = &ServiceLocator{
			container: NewContainer(),
		}
	})
	return instance
}

// SetContainer sets the underlying container
func (sl *ServiceLocator) SetContainer(container Container) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.container = container
}

// GetContainer returns the underlying container
func (sl *ServiceLocator) GetContainer() Container {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container
}

// Register registers a factory for a named service
func (sl *ServiceLocator) Register(name string, factory Factory) error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container.Register(name, factory)
}

// RegisterSingleton registers a singleton factory for a named service
func (sl *ServiceLocator) RegisterSingleton(name string, factory Factory) error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container.RegisterSingleton(name, factory)
}

// Resolve resolves a dependency by name
func (sl *ServiceLocator) Resolve(name string) (interface{}, error) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container.Resolve(name)
}

// ResolveWithContext resolves a dependency by name with context
func (sl *ServiceLocator) ResolveWithContext(ctx context.Context, name string) (interface{}, error) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container.ResolveWithContext(ctx, name)
}

// Has checks if a service is registered
func (sl *ServiceLocator) Has(name string) bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container.Has(name)
}

// Helper functions for common patterns

// MustResolve resolves a dependency and panics if it fails
func (sl *ServiceLocator) MustResolve(name string) interface{} {
	service, err := sl.Resolve(name)
	if err != nil {
		panic("failed to resolve service " + name + ": " + err.Error())
	}
	return service
}

// MustResolveWithContext resolves a dependency with context and panics if it fails
func (sl *ServiceLocator) MustResolveWithContext(ctx context.Context, name string) interface{} {
	service, err := sl.ResolveWithContext(ctx, name)
	if err != nil {
		panic("failed to resolve service " + name + ": " + err.Error())
	}
	return service
}
