//go:build integration
// +build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// DatabaseTestSuite provides comprehensive testing infrastructure for database integration tests
type DatabaseTestSuite struct {
	DB      *TestDatabase
	Factory *TestDataFactory
	Repos   *RepositorySet
	Assert  *AssertDatabaseState
	ctx     context.Context
}

// RepositorySet holds all repository instances for testing
type RepositorySet struct {
	Users    repository.UserRepository
	Projects repository.ProjectRepository
	Tasks    repository.TaskRepository
	Comments repository.CommentRepository
}

// SetupDatabaseTest creates an isolated database test environment
func SetupDatabaseTest(t *testing.T) *DatabaseTestSuite {
	// Create test database
	testDB := NewTestDatabase(t)

	// Create data factory
	factory := NewTestDataFactory(testDB)

	// Initialize repositories with the test database
	repos := &RepositorySet{
		Users:    repository.NewPocketBaseUserRepository(testDB.App()),
		Projects: repository.NewPocketBaseProjectRepository(testDB.App()),
		Tasks:    repository.NewPocketBaseTaskRepository(testDB.App()),
		Comments: repository.NewPocketBaseCommentRepository(testDB.App()),
	}

	// Create assertion helper
	assertHelper := &AssertDatabaseState{
		t:  t,
		db: testDB,
	}

	// Create test suite
	suite := &DatabaseTestSuite{
		DB:      testDB,
		Factory: factory,
		Repos:   repos,
		Assert:  assertHelper,
		ctx:     context.Background(),
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
