//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ericfisherdev/simple-easy-tasks/internal/config"
	"github.com/ericfisherdev/simple-easy-tasks/internal/container"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
	"github.com/ericfisherdev/simple-easy-tasks/internal/services"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DatabaseTestSuite provides comprehensive testing infrastructure for database integration tests
type DatabaseTestSuite struct {
	DB        *TestDatabase
	Factory   *TestDataFactory
	Repos     *RepositorySet
	Services  *ServiceSet
	Container container.Container
	Assert    *AssertDatabaseState
	ctx       context.Context
	UseDI     bool
}

// RepositorySet holds all repository instances for testing
type RepositorySet struct {
	Users    repository.UserRepository
	Projects repository.ProjectRepository
	Tasks    repository.TaskRepository
	Comments repository.CommentRepository
}

// ServiceSet holds all service instances for testing
type ServiceSet struct {
	Auth                services.AuthService
	User                services.UserService
	Project             services.ProjectService
	Task                services.TaskService
	Comment             services.CommentService
	Health              services.HealthServiceInterface
	SubscriptionManager services.SubscriptionManager
	EventBroadcaster    services.EventBroadcaster
}

// TestSuiteOptions configures test suite behavior
type TestSuiteOptions struct {
	// UseDependencyInjection enables DI container and service layer testing
	UseDependencyInjection bool
}

// SetupDatabaseTest creates an isolated database test environment
// Maintains backward compatibility with existing tests
func SetupDatabaseTest(t *testing.T) *DatabaseTestSuite {
	return SetupDatabaseTestWithOptions(t, nil)
}

// SetupDatabaseTestWithOptions creates an isolated database test environment with configuration options
func SetupDatabaseTestWithOptions(t *testing.T, options *TestSuiteOptions) *DatabaseTestSuite {
	if options == nil {
		options = &TestSuiteOptions{UseDependencyInjection: false}
	}
	// Create test database
	testDB := NewTestDatabase(t)

	// Create data factory
	factory := NewTestDataFactory(testDB)

	var repos *RepositorySet
	var serviceSet *ServiceSet
	var diContainer container.Container

	if options.UseDependencyInjection {
		// Setup DI container and services
		diContainer = container.NewContainer()
		cfg := config.NewConfig()

		// Register all services with the DI container
		err := container.RegisterServices(diContainer, cfg, testDB.App())
		if err != nil {
			t.Fatalf("Failed to register services with DI container: %v", err)
		}

		// Resolve services from container
		serviceSet = &ServiceSet{}

		// Auth Service
		if authService, err := container.ResolveAuthService(diContainer); err != nil {
			t.Fatalf("Failed to resolve auth service: %v", err)
		} else {
			serviceSet.Auth = authService
		}

		// User Service
		if userService, err := container.ResolveUserService(diContainer); err != nil {
			t.Fatalf("Failed to resolve user service: %v", err)
		} else {
			serviceSet.User = userService
		}

		// Project Service
		if projectService, err := container.ResolveProjectService(diContainer); err != nil {
			t.Fatalf("Failed to resolve project service: %v", err)
		} else {
			serviceSet.Project = projectService
		}

		// Task Service
		if taskService, err := container.ResolveTaskService(diContainer); err != nil {
			t.Fatalf("Failed to resolve task service: %v", err)
		} else {
			serviceSet.Task = taskService
		}

		// Comment Service
		if commentService, err := container.ResolveCommentService(diContainer); err != nil {
			t.Fatalf("Failed to resolve comment service: %v", err)
		} else {
			serviceSet.Comment = commentService
		}

		// Health Service
		if healthService, err := container.ResolveHealthService(diContainer); err != nil {
			t.Fatalf("Failed to resolve health service: %v", err)
		} else {
			serviceSet.Health = healthService
		}

		// TODO: Add resolution for SubscriptionManager and EventBroadcaster when available in container
		// For now, these will be nil and tests should handle gracefully
		// serviceSet.SubscriptionManager = nil
		// serviceSet.EventBroadcaster = nil

		// Also resolve repositories from container for backward compatibility
		repos = &RepositorySet{}
		if userRepo, err := diContainer.Resolve(container.UserRepositoryService); err != nil {
			t.Fatalf("Failed to resolve user repository: %v", err)
		} else if typedRepo, ok := userRepo.(repository.UserRepository); !ok {
			t.Fatalf("Failed to cast user repository")
		} else {
			repos.Users = typedRepo
		}

		if projectRepo, err := diContainer.Resolve(container.ProjectRepositoryService); err != nil {
			t.Fatalf("Failed to resolve project repository: %v", err)
		} else if typedRepo, ok := projectRepo.(repository.ProjectRepository); !ok {
			t.Fatalf("Failed to cast project repository")
		} else {
			repos.Projects = typedRepo
		}

		if taskRepo, err := diContainer.Resolve(container.TaskRepositoryService); err != nil {
			t.Fatalf("Failed to resolve task repository: %v", err)
		} else if typedRepo, ok := taskRepo.(repository.TaskRepository); !ok {
			t.Fatalf("Failed to cast task repository")
		} else {
			repos.Tasks = typedRepo
		}

		if commentRepo, err := diContainer.Resolve(container.CommentRepositoryService); err != nil {
			t.Fatalf("Failed to resolve comment repository: %v", err)
		} else if typedRepo, ok := commentRepo.(repository.CommentRepository); !ok {
			t.Fatalf("Failed to cast comment repository")
		} else {
			repos.Comments = typedRepo
		}
	} else {
		// Legacy mode: Initialize repositories directly
		repos = &RepositorySet{
			Users:    repository.NewPocketBaseUserRepository(testDB.App()),
			Projects: repository.NewPocketBaseProjectRepository(testDB.App()),
			Tasks:    repository.NewPocketBaseTaskRepository(testDB.App()),
			Comments: repository.NewPocketBaseCommentRepository(testDB.App()),
		}
	}

	// Create assertion helper
	assertHelper := &AssertDatabaseState{
		t:  t,
		db: testDB,
	}

	// Create test suite
	suite := &DatabaseTestSuite{
		DB:        testDB,
		Factory:   factory,
		Repos:     repos,
		Services:  serviceSet,
		Container: diContainer,
		Assert:    assertHelper,
		ctx:       context.Background(),
		UseDI:     options.UseDependencyInjection,
	}

	// Register cleanup
	t.Cleanup(suite.Cleanup)

	return suite
}

// Context returns the test context
func (s *DatabaseTestSuite) Context() context.Context {
	return s.ctx
}

// Cleanup performs test cleanup
func (s *DatabaseTestSuite) Cleanup() {
	s.DB.Cleanup()
}

// Reset clears all test data while preserving schema
func (s *DatabaseTestSuite) Reset() error {
	return s.DB.Reset()
}

// HasServices returns true if this suite is configured with DI services
func (s *DatabaseTestSuite) HasServices() bool {
	return s.UseDI && s.Services != nil
}

// RequireServices ensures this suite has services configured, fails the test if not
func (s *DatabaseTestSuite) RequireServices(t *testing.T) {
	if !s.HasServices() {
		t.Fatal("Test requires DI services but suite was not configured with UseDependencyInjection=true")
	}
}

// RequireServicesT ensures this suite has services configured, fails the test if not (generic version)
func (s *DatabaseTestSuite) RequireServicesT(t TestingT) {
	if !s.HasServices() {
		t.Fatal("Test requires DI services but suite was not configured with UseDependencyInjection=true")
	}
}

// TestingT is an interface that both *testing.T and *testing.B implement
type TestingT interface {
	Helper()
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Skip(args ...interface{})
	Skipf(format string, args ...interface{})
}

// GetTaskService returns the task service, ensuring it's available
func (s *DatabaseTestSuite) GetTaskService(t TestingT) services.TaskService {
	s.RequireServicesT(t)
	return s.Services.Task
}

// GetTaskServiceT is the original method for backward compatibility
func (s *DatabaseTestSuite) GetTaskServiceT(t *testing.T) services.TaskService {
	s.RequireServices(t)
	return s.Services.Task
}

// GetProjectService returns the project service, ensuring it's available
func (s *DatabaseTestSuite) GetProjectService(t TestingT) services.ProjectService {
	s.RequireServicesT(t)
	return s.Services.Project
}

// GetUserService returns the user service, ensuring it's available
func (s *DatabaseTestSuite) GetUserService(t TestingT) services.UserService {
	s.RequireServicesT(t)
	return s.Services.User
}

// GetCommentService returns the comment service, ensuring it's available
func (s *DatabaseTestSuite) GetCommentService(t TestingT) services.CommentService {
	s.RequireServicesT(t)
	return s.Services.Comment
}

// GetAuthService returns the auth service, ensuring it's available
func (s *DatabaseTestSuite) GetAuthService(t TestingT) services.AuthService {
	s.RequireServicesT(t)
	return s.Services.Auth
}

// GetHealthService returns the health service, ensuring it's available
func (s *DatabaseTestSuite) GetHealthService(t TestingT) services.HealthServiceInterface {
	s.RequireServicesT(t)
	return s.Services.Health
}

// GetUserRepository returns the user repository, ensuring it's available
func (s *DatabaseTestSuite) GetUserRepository(t TestingT) repository.UserRepository {
	if s.Repos == nil || s.Repos.Users == nil {
		t.Fatal("User repository not available - ensure suite is configured properly")
	}
	return s.Repos.Users
}

// GetProjectRepository returns the project repository, ensuring it's available
func (s *DatabaseTestSuite) GetProjectRepository(t TestingT) repository.ProjectRepository {
	if s.Repos == nil || s.Repos.Projects == nil {
		t.Fatal("Project repository not available - ensure suite is configured properly")
	}
	return s.Repos.Projects
}

// GetTaskRepository returns the task repository, ensuring it's available
func (s *DatabaseTestSuite) GetTaskRepository(t TestingT) repository.TaskRepository {
	if s.Repos == nil || s.Repos.Tasks == nil {
		t.Fatal("Task repository not available - ensure suite is configured properly")
	}
	return s.Repos.Tasks
}

// GetCommentRepository returns the comment repository, ensuring it's available
func (s *DatabaseTestSuite) GetCommentRepository(t TestingT) repository.CommentRepository {
	if s.Repos == nil || s.Repos.Comments == nil {
		t.Fatal("Comment repository not available - ensure suite is configured properly")
	}
	return s.Repos.Comments
}

// GetSubscriptionManager returns the subscription manager, ensuring it's available
func (s *DatabaseTestSuite) GetSubscriptionManager(t TestingT) services.SubscriptionManager {
	s.RequireServicesT(t)
	if s.Services.SubscriptionManager == nil {
		t.Skip("SubscriptionManager not available - skipping test")
	}
	return s.Services.SubscriptionManager
}

// GetEventBroadcaster returns the event broadcaster, ensuring it's available
func (s *DatabaseTestSuite) GetEventBroadcaster(t TestingT) services.EventBroadcaster {
	s.RequireServicesT(t)
	if s.Services.EventBroadcaster == nil {
		t.Skip("EventBroadcaster not available - skipping test")
	}
	return s.Services.EventBroadcaster
}

// GetPocketBaseApp returns the PocketBase app instance
func (s *DatabaseTestSuite) GetPocketBaseApp(t TestingT) core.App {
	if s.DB == nil || s.DB.App() == nil {
		t.Fatal("PocketBase app not available - ensure suite is configured properly")
	}
	return s.DB.App()
}

// AssertDatabaseState provides database-specific assertions
type AssertDatabaseState struct {
	t  *testing.T
	db *TestDatabase
}

// UserExists asserts that a user exists in the database
func (a *AssertDatabaseState) UserExists(userID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("users").
		Where(dbx.HashExp{"id": userID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query user existence")
	assert.Equal(a.t, 1, count, "User should exist in database: %s", userID)
}

// UserNotExists asserts that a user does not exist in the database
func (a *AssertDatabaseState) UserNotExists(userID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("users").
		Where(dbx.HashExp{"id": userID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query user existence")
	assert.Equal(a.t, 0, count, "User should not exist in database: %s", userID)
}

// ProjectExists asserts that a project exists in the database
func (a *AssertDatabaseState) ProjectExists(projectID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("projects").
		Where(dbx.HashExp{"id": projectID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query project existence")
	assert.Equal(a.t, 1, count, "Project should exist in database: %s", projectID)
}

// ProjectNotExists asserts that a project does not exist in the database
func (a *AssertDatabaseState) ProjectNotExists(projectID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("projects").
		Where(dbx.HashExp{"id": projectID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query project existence")
	assert.Equal(a.t, 0, count, "Project should not exist in database: %s", projectID)
}

// TaskExists asserts that a task exists in the database
func (a *AssertDatabaseState) TaskExists(taskID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("tasks").
		Where(dbx.HashExp{"id": taskID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query task existence")
	assert.Equal(a.t, 1, count, "Task should exist in database: %s", taskID)
}

// TaskNotExists asserts that a task does not exist in the database
func (a *AssertDatabaseState) TaskNotExists(taskID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("tasks").
		Where(dbx.HashExp{"id": taskID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query task existence")
	assert.Equal(a.t, 0, count, "Task should not exist in database: %s", taskID)
}

// CommentExists asserts that a comment exists in the database
func (a *AssertDatabaseState) CommentExists(commentID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("comments").
		Where(dbx.HashExp{"id": commentID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query comment existence")
	assert.Equal(a.t, 1, count, "Comment should exist in database: %s", commentID)
}

// CommentNotExists asserts that a comment does not exist in the database
func (a *AssertDatabaseState) CommentNotExists(commentID string) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("comments").
		Where(dbx.HashExp{"id": commentID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to query comment existence")
	assert.Equal(a.t, 0, count, "Comment should not exist in database: %s", commentID)
}

// TaskHasParent asserts that a task has a specific parent task
func (a *AssertDatabaseState) TaskHasParent(taskID, parentID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("tasks")
	require.NoError(a.t, err, "Failed to find tasks collection")

	record, err := a.db.App().FindRecordById(collection, taskID, nil)
	require.NoError(a.t, err, "Failed to find task: %s", taskID)

	actualParentID := record.GetString("parent_task")
	assert.NotEmpty(a.t, actualParentID, "Task should have a parent: %s", taskID)
	assert.Equal(a.t, parentID, actualParentID, "Parent ID should match for task: %s", taskID)
}

// TaskHasNoParent asserts that a task has no parent task
func (a *AssertDatabaseState) TaskHasNoParent(taskID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("tasks")
	require.NoError(a.t, err, "Failed to find tasks collection")

	record, err := a.db.App().FindRecordById(collection, taskID, nil)
	require.NoError(a.t, err, "Failed to find task: %s", taskID)

	actualParentID := record.GetString("parent_task")
	assert.Empty(a.t, actualParentID, "Task should not have a parent: %s", taskID)
}

// CommentHasParent asserts that a comment has a specific parent comment
func (a *AssertDatabaseState) CommentHasParent(commentID, parentID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("comments")
	require.NoError(a.t, err, "Failed to find comments collection")

	record, err := a.db.App().FindRecordById(collection, commentID, nil)
	require.NoError(a.t, err, "Failed to find comment: %s", commentID)

	actualParentID := record.GetString("parent_comment")
	assert.NotEmpty(a.t, actualParentID, "Comment should have a parent: %s", commentID)
	assert.Equal(a.t, parentID, actualParentID, "Parent ID should match for comment: %s", commentID)
}

// CommentHasNoParent asserts that a comment has no parent comment
func (a *AssertDatabaseState) CommentHasNoParent(commentID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("comments")
	require.NoError(a.t, err, "Failed to find comments collection")

	record, err := a.db.App().FindRecordById(collection, commentID, nil)
	require.NoError(a.t, err, "Failed to find comment: %s", commentID)

	actualParentID := record.GetString("parent_comment")
	assert.Empty(a.t, actualParentID, "Comment should not have a parent: %s", commentID)
}

// ProjectHasOwner asserts that a project has a specific owner
func (a *AssertDatabaseState) ProjectHasOwner(projectID, ownerID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("projects")
	require.NoError(a.t, err, "Failed to find projects collection")

	record, err := a.db.App().FindRecordById(collection, projectID, nil)
	require.NoError(a.t, err, "Failed to find project: %s", projectID)

	// Debug: print all field values in the retrieved record
	fmt.Printf("DEBUG: Retrieved project record %s:\n", projectID)
	fmt.Printf("  Title: '%s'\n", record.GetString("title"))
	fmt.Printf("  Slug: '%s'\n", record.GetString("slug"))
	fmt.Printf("  Owner: '%s'\n", record.GetString("owner"))
	fmt.Printf("  Status: '%s'\n", record.GetString("status"))
	fmt.Printf("  Description: '%s'\n", record.GetString("description"))
	fmt.Printf("  Color: '%s'\n", record.GetString("color"))
	fmt.Printf("  Icon: '%s'\n", record.GetString("icon"))

	// Debug: Check what columns actually exist in the projects table
	// First, let's see what columns are in the table
	rows, err := a.db.App().DB().NewQuery("PRAGMA table_info(projects)").Rows()
	if err != nil {
		fmt.Printf("DEBUG: Failed to get table info: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Projects table columns:\n")
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var defaultValue interface{}
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err == nil {
				fmt.Printf("  %d: %s (%s)\n", cid, name, dataType)
			}
		}
	}

	actualOwnerID := record.GetString("owner")
	assert.Equal(a.t, ownerID, actualOwnerID, "Owner ID should match for project: %s", projectID)
}

// TaskHasAssignee asserts that a task has a specific assignee
func (a *AssertDatabaseState) TaskHasAssignee(taskID, assigneeID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("tasks")
	require.NoError(a.t, err, "Failed to find tasks collection")

	record, err := a.db.App().FindRecordById(collection, taskID, nil)
	require.NoError(a.t, err, "Failed to find task: %s", taskID)

	actualAssigneeID := record.GetString("assignee")
	assert.NotEmpty(a.t, actualAssigneeID, "Task should have an assignee: %s", taskID)
	assert.Equal(a.t, assigneeID, actualAssigneeID, "Assignee ID should match for task: %s", taskID)
}

// TaskHasNoAssignee asserts that a task has no assignee
func (a *AssertDatabaseState) TaskHasNoAssignee(taskID string) {
	collection, err := a.db.App().FindCollectionByNameOrId("tasks")
	require.NoError(a.t, err, "Failed to find tasks collection")

	record, err := a.db.App().FindRecordById(collection, taskID, nil)
	require.NoError(a.t, err, "Failed to find task: %s", taskID)

	actualAssigneeID := record.GetString("assignee")
	assert.Empty(a.t, actualAssigneeID, "Task should not have an assignee: %s", taskID)
}

// TaskHasStatus asserts that a task has a specific status
func (a *AssertDatabaseState) TaskHasStatus(taskID string, status domain.TaskStatus) {
	collection, err := a.db.App().FindCollectionByNameOrId("tasks")
	require.NoError(a.t, err, "Failed to find tasks collection")

	record, err := a.db.App().FindRecordById(collection, taskID, nil)
	require.NoError(a.t, err, "Failed to find task: %s", taskID)

	actualStatus := record.GetString("status")
	assert.Equal(a.t, string(status), actualStatus, "Status should match for task: %s", taskID)
}

// ConstraintViolated asserts that an error represents a database constraint violation
func (a *AssertDatabaseState) ConstraintViolated(err error, constraintType string) {
	assert.Error(a.t, err, "Expected constraint violation error")
	if err != nil {
		errStr := err.Error()

		// Check for various constraint violation patterns
		constraintViolationPatterns := []string{
			"constraint",
			"CONSTRAINT",
			"unique",
			"UNIQUE",
			"foreign key",
			"FOREIGN KEY",
			"not null",
			"NOT NULL",
			"check constraint",
			"CHECK",
		}

		hasConstraintError := false
		for _, pattern := range constraintViolationPatterns {
			if strings.Contains(errStr, pattern) {
				hasConstraintError = true
				break
			}
		}

		assert.True(a.t, hasConstraintError,
			"Error should indicate constraint violation, got: %s", errStr)

		// If specific constraint type is provided, check for it
		if constraintType != "" {
			assert.Contains(a.t, strings.ToLower(errStr), strings.ToLower(constraintType),
				"Error should contain constraint type '%s', got: %s", constraintType, errStr)
		}
	}
}

// UniqueConstraintViolated asserts that an error represents a unique constraint violation
func (a *AssertDatabaseState) UniqueConstraintViolated(err error) {
	a.ConstraintViolated(err, "unique")
}

// ForeignKeyConstraintViolated asserts that an error represents a foreign key constraint violation
func (a *AssertDatabaseState) ForeignKeyConstraintViolated(err error) {
	a.ConstraintViolated(err, "foreign key")
}

// NotNullConstraintViolated asserts that an error represents a not null constraint violation
func (a *AssertDatabaseState) NotNullConstraintViolated(err error) {
	a.ConstraintViolated(err, "not null")
}

// RecordCount asserts that a table has a specific number of records
func (a *AssertDatabaseState) RecordCount(tableName string, expectedCount int) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From(tableName).Row(&count)
	require.NoError(a.t, err, "Failed to count records in table: %s", tableName)
	assert.Equal(a.t, expectedCount, count, "Record count mismatch in table: %s", tableName)
}

// UserCount asserts that there are a specific number of users
func (a *AssertDatabaseState) UserCount(expectedCount int) {
	a.RecordCount("users", expectedCount)
}

// ProjectCount asserts that there are a specific number of projects
func (a *AssertDatabaseState) ProjectCount(expectedCount int) {
	a.RecordCount("projects", expectedCount)
}

// TaskCount asserts that there are a specific number of tasks
func (a *AssertDatabaseState) TaskCount(expectedCount int) {
	a.RecordCount("tasks", expectedCount)
}

// CommentCount asserts that there are a specific number of comments
func (a *AssertDatabaseState) CommentCount(expectedCount int) {
	a.RecordCount("comments", expectedCount)
}

// TaskCountByProject asserts that a project has a specific number of tasks
func (a *AssertDatabaseState) TaskCountByProject(projectID string, expectedCount int) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("tasks").
		Where(dbx.HashExp{"project": projectID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to count tasks for project: %s", projectID)
	assert.Equal(a.t, expectedCount, count, "Task count mismatch for project: %s", projectID)
}

// CommentCountByTask asserts that a task has a specific number of comments
func (a *AssertDatabaseState) CommentCountByTask(taskID string, expectedCount int) {
	var count int
	err := a.db.App().DB().Select("COUNT(*)").From("comments").
		Where(dbx.HashExp{"task": taskID}).
		Row(&count)
	require.NoError(a.t, err, "Failed to count comments for task: %s", taskID)
	assert.Equal(a.t, expectedCount, count, "Comment count mismatch for task: %s", taskID)
}

// DatabaseIsEmpty asserts that all main tables are empty
func (a *AssertDatabaseState) DatabaseIsEmpty() {
	a.UserCount(0)
	a.ProjectCount(0)
	a.TaskCount(0)
	a.CommentCount(0)
}

// DatabaseIsNotEmpty asserts that at least one main table has data
func (a *AssertDatabaseState) DatabaseIsNotEmpty() {
	var totalCount int

	tables := []string{"users", "projects", "tasks", "comments"}
	for _, table := range tables {
		var count int
		err := a.db.App().DB().Select("COUNT(*)").From(table).Row(&count)
		require.NoError(a.t, err, "Failed to count records in table: %s", table)
		totalCount += count
	}

	assert.Greater(a.t, totalCount, 0, "Database should not be empty")
}

// isRepositoryStub checks if a repository is a stub implementation by testing a simple method
func isRepositoryStub(repo interface{}) bool {
	// Test if repository returns NOT_IMPLEMENTED error
	switch r := repo.(type) {
	case repository.TaskRepository:
		_, err := r.Count(context.Background())
		return err != nil && strings.Contains(err.Error(), "NOT_IMPLEMENTED")
	case repository.CommentRepository:
		_, err := r.Count(context.Background())
		return err != nil && strings.Contains(err.Error(), "NOT_IMPLEMENTED")
	}
	return false
}

// SetupConcurrencyTestWithServices creates a concurrency test suite with DI services
// This is the recommended setup for new concurrency tests
func SetupConcurrencyTestWithServices(t *testing.T) *DatabaseTestSuite {
	return SetupDatabaseTestWithOptions(t, &TestSuiteOptions{
		UseDependencyInjection: true,
	})
}

// SetupServiceTest creates a test suite specifically configured for service-layer testing
func SetupServiceTest(t *testing.T) *DatabaseTestSuite {
	return SetupConcurrencyTestWithServices(t)
}
