//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/config"
	"simple-easy-tasks/internal/container"
	"simple-easy-tasks/internal/repository"
	testutil "simple-easy-tasks/internal/testutil/integration"
)

// TestContainer provides a configured DI container for integration tests
type TestContainer struct {
	Container container.Container
	App       core.App
	cleanup   func()
}

// NewTestContainer creates a new test container with all services registered
func NewTestContainer(t *testing.T) *TestContainer {
	// Use the existing DatabaseTestSuite but extract what we need
	suite := testutil.SetupDatabaseTest(t)
	
	// Create DI container
	c := container.NewContainer()
	
	// Create config for testing using the standard constructor
	cfg := config.NewConfig()
	
	// Register services with the test PocketBase app
	err := container.RegisterServices(c, cfg, suite.DB.App())
	if err != nil {
		t.Fatalf("Failed to register services: %v", err)
	}
	
	return &TestContainer{
		Container: c,
		App:       suite.DB.App(),
		cleanup:   suite.Cleanup,
	}
}

// Cleanup cleans up the test container and database
func (tc *TestContainer) Cleanup() {
	if tc.cleanup != nil {
		tc.cleanup()
	}
}

// GetUserRepository returns the user repository from the DI container
func (tc *TestContainer) GetUserRepository(t *testing.T) repository.UserRepository {
	repo, err := tc.Container.Resolve(container.UserRepositoryService)
	if err != nil {
		t.Fatalf("Failed to resolve user repository: %v", err)
	}
	
	userRepo, ok := repo.(repository.UserRepository)
	if !ok {
		t.Fatalf("Failed to cast to UserRepository")
	}
	
	return userRepo
}

// GetTaskRepository returns the task repository from the DI container
func (tc *TestContainer) GetTaskRepository(t *testing.T) repository.TaskRepository {
	repo, err := tc.Container.Resolve(container.TaskRepositoryService)
	if err != nil {
		t.Fatalf("Failed to resolve task repository: %v", err)
	}
	
	taskRepo, ok := repo.(repository.TaskRepository)
	if !ok {
		t.Fatalf("Failed to cast to TaskRepository")
	}
	
	return taskRepo
}

// GetProjectRepository returns the project repository from the DI container
func (tc *TestContainer) GetProjectRepository(t *testing.T) repository.ProjectRepository {
	repo, err := tc.Container.Resolve(container.ProjectRepositoryService)
	if err != nil {
		t.Fatalf("Failed to resolve project repository: %v", err)
	}
	
	projectRepo, ok := repo.(repository.ProjectRepository)
	if !ok {
		t.Fatalf("Failed to cast to ProjectRepository")
	}
	
	return projectRepo
}

// GetCommentRepository returns the comment repository from the DI container
func (tc *TestContainer) GetCommentRepository(t *testing.T) repository.CommentRepository {
	repo, err := tc.Container.Resolve(container.CommentRepositoryService)
	if err != nil {
		t.Fatalf("Failed to resolve comment repository: %v", err)
	}
	
	commentRepo, ok := repo.(repository.CommentRepository)
	if !ok {
		t.Fatalf("Failed to cast to CommentRepository")
	}
	
	return commentRepo
}

// ClearDatabase clears all data from the test database
func (tc *TestContainer) ClearDatabase(t *testing.T) {
	collections, err := tc.App.FindAllCollections()
	if err != nil {
		t.Fatalf("Failed to get collections: %v", err)
	}

	for _, collection := range collections {
		if collection.IsView() {
			continue // Skip views
		}

		// Clear all records from the collection
		records, err := tc.App.FindRecordsByFilter(collection.Name, "", "", 0, 0)
		if err != nil {
			continue // Skip collections that don't exist or have issues
		}

		for _, record := range records {
			if err := tc.App.Delete(record); err != nil {
				t.Logf("Warning: Failed to delete record from %s: %v", collection.Name, err)
			}
		}
	}
}