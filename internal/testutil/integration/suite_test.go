//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"simple-easy-tasks/internal/domain"
)

func TestSetupDatabaseTest(t *testing.T) {
	suite := SetupDatabaseTest(t)

	// Verify all components are initialized
	assert.NotNil(t, suite.DB, "Database should be initialized")
	assert.NotNil(t, suite.Factory, "Factory should be initialized")
	assert.NotNil(t, suite.Repos, "Repositories should be initialized")
	assert.NotNil(t, suite.Assert, "Assert helper should be initialized")
	assert.NotNil(t, suite.ctx, "Context should be initialized")

	// Verify repositories are initialized
	assert.NotNil(t, suite.Repos.Users, "UserRepository should be initialized")
	assert.NotNil(t, suite.Repos.Projects, "ProjectRepository should be initialized")
	assert.NotNil(t, suite.Repos.Tasks, "TaskRepository should be initialized")
	assert.NotNil(t, suite.Repos.Comments, "CommentRepository should be initialized")

	// Verify database starts empty
	suite.Assert.DatabaseIsEmpty()
}

func TestDatabaseTestSuite_Reset(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	// Create some test data
	user := suite.Factory.CreateUser()
	project := suite.Factory.CreateProject(user)

	err := suite.Repos.Users.Create(ctx, user)
	require.NoError(t, err)

	err = suite.Repos.Projects.Create(ctx, project)
	require.NoError(t, err)

	// Verify data exists
	suite.Assert.UserExists(user.ID)
	suite.Assert.ProjectExists(project.ID)
	suite.Assert.DatabaseIsNotEmpty()

	// Reset database
	err = suite.Reset()
	require.NoError(t, err)

	// Verify data is gone
	suite.Assert.UserNotExists(user.ID)
	suite.Assert.ProjectNotExists(project.ID)
	suite.Assert.DatabaseIsEmpty()
}

func TestAssertDatabaseState_UserAssertions(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	// Test user existence assertions with non-existent user
	nonExistentID := "non-existent-user"
	suite.Assert.UserNotExists(nonExistentID)

	// Create and save a user
	user := suite.Factory.CreateUser()
	err := suite.Repos.Users.Create(ctx, user)
	require.NoError(t, err)

	// Test user existence assertions
	suite.Assert.UserExists(user.ID)
	suite.Assert.UserCount(1)

	// Create another user
	user2 := suite.Factory.CreateUser()
	err = suite.Repos.Users.Create(ctx, user2)
	require.NoError(t, err)

	suite.Assert.UserExists(user2.ID)
	suite.Assert.UserCount(2)
}

func TestAssertDatabaseState_ProjectAssertions(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	// Skip test if project repository not implemented
	if isRepositoryStub(suite.Repos.Projects) {
		t.Skip("Project repository not yet implemented")
	}

	// Create owner user
	owner := suite.Factory.CreateUser()
	err := suite.Repos.Users.Create(ctx, owner)
	require.NoError(t, err)

	// Test project existence assertions with non-existent project
	nonExistentID := "non-existent-project"
	suite.Assert.ProjectNotExists(nonExistentID)

	// Create and save a project
	project := suite.Factory.CreateProject(owner)
	err = suite.Repos.Projects.Create(ctx, project)
	require.NoError(t, err)

	// Test project assertions
	suite.Assert.ProjectExists(project.ID)
	suite.Assert.ProjectHasOwner(project.ID, owner.ID)
	suite.Assert.ProjectCount(1)
}

func TestAssertDatabaseState_TaskAssertions(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	// Skip test if task repository not implemented
	if isRepositoryStub(suite.Repos.Tasks) {
		t.Skip("Task repository not yet implemented")
	}

	// Create user and project
	user := suite.Factory.CreateUser()
	err := suite.Repos.Users.Create(ctx, user)
	require.NoError(t, err)

	project := suite.Factory.CreateProject(user)
	err = suite.Repos.Projects.Create(ctx, project)
	require.NoError(t, err)

	// Test task existence assertions with non-existent task
	nonExistentID := "non-existent-task"
	suite.Assert.TaskNotExists(nonExistentID)

	// Create and save a task without assignee
	task := suite.Factory.CreateTask(project, user)
	err = suite.Repos.Tasks.Create(ctx, task)
	require.NoError(t, err)

	// Test task assertions
	suite.Assert.TaskExists(task.ID)
	suite.Assert.TaskHasNoAssignee(task.ID)
	suite.Assert.TaskHasNoParent(task.ID)
	suite.Assert.TaskHasStatus(task.ID, domain.StatusBacklog)
	suite.Assert.TaskCount(1)
	suite.Assert.TaskCountByProject(project.ID, 1)

	// Create assignee
	assignee := suite.Factory.CreateUser()
	err = suite.Repos.Users.Create(ctx, assignee)
	require.NoError(t, err)

	// Create task with assignee
	taskWithAssignee := suite.Factory.CreateTask(project, user,
		WithTaskAssignee(assignee.ID),
		WithTaskStatus(domain.StatusDeveloping))
	err = suite.Repos.Tasks.Create(ctx, taskWithAssignee)
	require.NoError(t, err)

	suite.Assert.TaskHasAssignee(taskWithAssignee.ID, assignee.ID)
	suite.Assert.TaskHasStatus(taskWithAssignee.ID, domain.StatusDeveloping)
	suite.Assert.TaskCount(2)

	// Create parent-child task relationship
	parentTask := suite.Factory.CreateTask(project, user)
	err = suite.Repos.Tasks.Create(ctx, parentTask)
	require.NoError(t, err)

	childTask := suite.Factory.CreateTask(project, user,
		WithTaskParent(parentTask.ID))
	err = suite.Repos.Tasks.Create(ctx, childTask)
	require.NoError(t, err)

	suite.Assert.TaskHasParent(childTask.ID, parentTask.ID)
	suite.Assert.TaskHasNoParent(parentTask.ID)
}

func TestAssertDatabaseState_CommentAssertions(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	// Skip test if repositories not implemented
	if isRepositoryStub(suite.Repos.Tasks) || isRepositoryStub(suite.Repos.Comments) {
		t.Skip("Task or Comment repository not yet implemented")
	}

	// Create user, project, and task
	user := suite.Factory.CreateUser()
	err := suite.Repos.Users.Create(ctx, user)
	require.NoError(t, err)

	project := suite.Factory.CreateProject(user)
	err = suite.Repos.Projects.Create(ctx, project)
	require.NoError(t, err)

	task := suite.Factory.CreateTask(project, user)
	err = suite.Repos.Tasks.Create(ctx, task)
	require.NoError(t, err)

	// Test comment existence assertions with non-existent comment
	nonExistentID := "non-existent-comment"
	suite.Assert.CommentNotExists(nonExistentID)

	// Create and save a comment
	comment := suite.Factory.CreateComment(task, user)
	err = suite.Repos.Comments.Create(ctx, comment)
	require.NoError(t, err)

	// Test comment assertions
	suite.Assert.CommentExists(comment.ID)
	suite.Assert.CommentHasNoParent(comment.ID)
	suite.Assert.CommentCount(1)
	suite.Assert.CommentCountByTask(task.ID, 1)

	// Create reply comment
	reply := suite.Factory.CreateComment(task, user,
		WithCommentParent(comment.ID))
	err = suite.Repos.Comments.Create(ctx, reply)
	require.NoError(t, err)

	suite.Assert.CommentHasParent(reply.ID, comment.ID)
	suite.Assert.CommentCount(2)
	suite.Assert.CommentCountByTask(task.ID, 2)
}

func TestAssertDatabaseState_ConstraintViolations(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	t.Run("unique constraint violation - duplicate user email", func(t *testing.T) {
		// Create first user
		user1 := suite.Factory.CreateUser(WithUserEmail("duplicate@example.com"))
		err := suite.Repos.Users.Create(ctx, user1)
		require.NoError(t, err)

		// Try to create second user with same email
		user2 := suite.Factory.CreateUser(WithUserEmail("duplicate@example.com"))
		err = suite.Repos.Users.Create(ctx, user2)

		// Should violate unique constraint
		suite.Assert.UniqueConstraintViolated(err)
	})

	t.Run("unique constraint violation - duplicate username", func(t *testing.T) {
		// Create first user
		user1 := suite.Factory.CreateUser(WithUserUsername("duplicateuser"))
		err := suite.Repos.Users.Create(ctx, user1)
		require.NoError(t, err)

		// Try to create second user with same username
		user2 := suite.Factory.CreateUser(WithUserUsername("duplicateuser"))
		err = suite.Repos.Users.Create(ctx, user2)

		// Should violate unique constraint
		suite.Assert.UniqueConstraintViolated(err)
	})

	t.Run("foreign key constraint violation - task with non-existent project", func(t *testing.T) {
		// Skip if task repository not implemented
		if isRepositoryStub(suite.Repos.Tasks) {
			t.Skip("Task repository not yet implemented")
		}

		// Create user
		user := suite.Factory.CreateUser()
		err := suite.Repos.Users.Create(ctx, user)
		require.NoError(t, err)

		// Try to create task with non-existent project
		task := suite.Factory.CreateTask(
			&domain.Project{ID: "non-existent-project"},
			user,
		)
		err = suite.Repos.Tasks.Create(ctx, task)

		// Should violate foreign key constraint
		suite.Assert.ForeignKeyConstraintViolated(err)
	})
}

func TestAssertDatabaseState_RecordCounts(t *testing.T) {
	suite := SetupDatabaseTest(t)
	ctx := suite.Context()

	// Skip test if repositories not implemented
	if isRepositoryStub(suite.Repos.Tasks) || isRepositoryStub(suite.Repos.Comments) {
		t.Skip("Task or Comment repository not yet implemented")
	}

	// Start with empty database
	suite.Assert.DatabaseIsEmpty()
	suite.Assert.RecordCount("users", 0)
	suite.Assert.RecordCount("projects", 0)
	suite.Assert.RecordCount("tasks", 0)
	suite.Assert.RecordCount("comments", 0)

	// Create test data structure
	user, project, task, comment := suite.Factory.CreateFullTaskStructure()

	// Save all entities
	err := suite.Repos.Users.Create(ctx, user)
	require.NoError(t, err)

	err = suite.Repos.Projects.Create(ctx, project)
	require.NoError(t, err)

	err = suite.Repos.Tasks.Create(ctx, task)
	require.NoError(t, err)

	err = suite.Repos.Comments.Create(ctx, comment)
	require.NoError(t, err)

	// Verify counts
	suite.Assert.DatabaseIsNotEmpty()
	suite.Assert.UserCount(1)
	suite.Assert.ProjectCount(1)
	suite.Assert.TaskCount(1)
	suite.Assert.CommentCount(1)

	// Create additional entities
	user2 := suite.Factory.CreateUser()
	err = suite.Repos.Users.Create(ctx, user2)
	require.NoError(t, err)

	task2 := suite.Factory.CreateTask(project, user2)
	err = suite.Repos.Tasks.Create(ctx, task2)
	require.NoError(t, err)

	// Verify updated counts
	suite.Assert.UserCount(2)
	suite.Assert.ProjectCount(1) // Still only one project
	suite.Assert.TaskCount(2)
	suite.Assert.CommentCount(1)
	suite.Assert.TaskCountByProject(project.ID, 2)
	suite.Assert.CommentCountByTask(task.ID, 1)
	suite.Assert.CommentCountByTask(task2.ID, 0)
}

func TestDatabaseTestSuite_IsolatedTests(t *testing.T) {
	// This test verifies that each test gets an isolated database

	t.Run("first test creates data", func(t *testing.T) {
		suite := SetupDatabaseTest(t)
		ctx := suite.Context()

		user := suite.Factory.CreateUser()
		err := suite.Repos.Users.Create(ctx, user)
		require.NoError(t, err)

		suite.Assert.UserCount(1)
	})

	t.Run("second test has clean database", func(t *testing.T) {
		suite := SetupDatabaseTest(t)

		// Should start with empty database
		suite.Assert.DatabaseIsEmpty()
		suite.Assert.UserCount(0)
	})

	t.Run("third test can create same data without conflicts", func(t *testing.T) {
		suite := SetupDatabaseTest(t)
		ctx := suite.Context()

		// Should be able to create user with same email as first test
		user := suite.Factory.CreateUser(WithUserEmail("test@example.com"))
		err := suite.Repos.Users.Create(ctx, user)
		require.NoError(t, err)

		suite.Assert.UserCount(1)
	})
}
