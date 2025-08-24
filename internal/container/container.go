// Package container provides dependency injection capabilities for the application.
package container

import (
	"context"
	"reflect"
	"sync"
)

// Container represents a dependency injection container
type Container interface {
	Register(name string, factory Factory) error
	RegisterSingleton(name string, factory Factory) error
	Resolve(name string) (interface{}, error)
	ResolveWithContext(ctx context.Context, name string) (interface{}, error)
	Has(name string) bool
}

// Factory is a function that creates an instance of a dependency
type Factory func(ctx context.Context, c Container) (interface{}, error)

// ServiceRegistration represents a registered service
type ServiceRegistration struct {
	Factory   Factory
	Singleton bool
	Instance  interface{}
	once      sync.Once
}

// DIContainer is the default implementation of Container
type DIContainer struct {
	mu       sync.RWMutex
	services map[string]*ServiceRegistration
}

// NewContainer creates a new dependency injection container
func NewContainer() *DIContainer {
	return &DIContainer{
		services: make(map[string]*ServiceRegistration),
	}
}

// Register registers a factory for a named service
func (c *DIContainer) Register(name string, factory Factory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = &ServiceRegistration{
		Factory:   factory,
		Singleton: false,
	}

	return nil
}

// RegisterSingleton registers a singleton factory for a named service
func (c *DIContainer) RegisterSingleton(name string, factory Factory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = &ServiceRegistration{
		Factory:   factory,
		Singleton: true,
	}

	return nil
}

// Resolve resolves a dependency by name
func (c *DIContainer) Resolve(name string) (interface{}, error) {
	return c.ResolveWithContext(context.Background(), name)
}

// ResolveWithContext resolves a dependency by name with context
func (c *DIContainer) ResolveWithContext(ctx context.Context, name string) (interface{}, error) {
	c.mu.RLock()
	registration, exists := c.services[name]
	c.mu.RUnlock()

	if !exists {
		return nil, NewDependencyError("SERVICE_NOT_FOUND", "service not registered: "+name)
	}

	if registration.Singleton {
		var err error
		registration.once.Do(func() {
			registration.Instance, err = registration.Factory(ctx, c)
		})
		if err != nil {
			return nil, err
		}
		return registration.Instance, nil
	}

	return registration.Factory(ctx, c)
}

// Has checks if a service is registered
func (c *DIContainer) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.services[name]
	return exists
}

// ResolveType resolves a dependency by type using reflection
func (c *DIContainer) ResolveType(t reflect.Type) (interface{}, error) {
	return c.ResolveTypeWithContext(context.Background(), t)
}

// ResolveTypeWithContext resolves a dependency by type using reflection with context
func (c *DIContainer) ResolveTypeWithContext(ctx context.Context, t reflect.Type) (interface{}, error) {
	typeName := t.String()
	return c.ResolveWithContext(ctx, typeName)
}

// RegisterType registers a factory for a service by type
func (c *DIContainer) RegisterType(serviceType reflect.Type, factory Factory) error {
	return c.Register(serviceType.String(), factory)
}

// RegisterSingletonType registers a singleton factory for a service by type
func (c *DIContainer) RegisterSingletonType(serviceType reflect.Type, factory Factory) error {
	return c.RegisterSingleton(serviceType.String(), factory)
}

// DependencyError represents a dependency injection error
type DependencyError struct {
	Code    string
	Message string
}

// Error implements the error interface
func (e *DependencyError) Error() string {
	return e.Code + ": " + e.Message
}

// NewDependencyError creates a new dependency error
func NewDependencyError(code, message string) *DependencyError {
	return &DependencyError{
		Code:    code,
		Message: message,
	}
}
