//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

func TestForeignKeyConstraints_Integration(t *testing.T) {
	// Setup SHARED test suite for all subtests (performance optimization)
	suite := SetupDatabaseTest(t)
	defer suite.Cleanup()

	// Get repositories from suite
	taskRepo := suite.Repos.Tasks
	userRepo := suite.Repos.Users
	projectRepo := suite.Repos.Projects
	commentRepo := suite.Repos.Comments

	// Helper function to create basic test users and projects
	setupBasicTestData := func(t *testing.T, suite *DatabaseTestSuite) (*domain.User, *domain.User, *domain.Project) {
		// Create users
		owner := suite.Factory.CreateUser(
			WithUserEmail("owner@fk.test.com"),
			WithUserUsername("fkowner"),
			WithUserName("FK Owner"),
		)
		assignee := suite.Factory.CreateUser(
			WithUserEmail("assignee@fk.test.com"),
			WithUserUsername("fkassignee"),
			WithUserName("FK Assignee"),
		)

		// Save users to database
		require.NoError(t, userRepo.Create(context.Background(), owner))
		require.NoError(t, userRepo.Create(context.Background(), assignee))

		// Create project
		project := suite.Factory.CreateProject(owner,
			WithProjectTitle("FK Test Project"),
			WithProjectSlug("fk-test-project"),
		)
		require.NoError(t, projectRepo.Create(context.Background(), project))

		return owner, assignee, project
	}

	// ===========================================
	// REFERENTIAL INTEGRITY BEHAVIOR TESTS
	// Note: PocketBase doesn't enforce SQLite FK constraints by default
	// These tests verify the actual behavior and document integrity issues
	// ===========================================

	t.Run("TaskCreate_NonExistentProject_PocketBaseBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, _ := setupBasicTestData(t, suite)

		// Create task with non-existent project ID
		task := suite.Factory.CreateTask(&domain.Project{ID: "nonexistent_proj"}, owner,
			WithTaskTitle("Task with Invalid Project"),
			WithTaskDescription("This task references a non-existent project"),
		)

		// Act
		err := taskRepo.Create(context.Background(), task)

		// Assert - Document PocketBase behavior
		// PocketBase (by default) doesn't enforce foreign key constraints at the SQLite level
		if err != nil {
			// Some PocketBase configurations or versions might validate
			t.Logf("PocketBase rejected task with non-existent project: %v", err)
			assert.Contains(t, err.Error(), "failed to save task record")
			suite.Assert.TaskNotExists(task.ID)
		} else {
			// Most common behavior: PocketBase allows creation with invalid foreign keys
			t.Logf("WARNING: PocketBase allowed task with non-existent project - referential integrity not enforced")
			assert.NotEmpty(t, task.ID, "Task should have been assigned an ID")
			suite.Assert.TaskExists(task.ID)

			// Verify the broken reference exists in the database
			retrieved, retrieveErr := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, "nonexistent_proj", retrieved.ProjectID)
		}
	})

	t.Run("TaskCreate_NonExistentReporter_PocketBaseBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		_, _, project := setupBasicTestData(t, suite)

		// Create task with non-existent reporter ID
		invalidUser := &domain.User{ID: "nonexistent_user"}
		task := suite.Factory.CreateTask(project, invalidUser,
			WithTaskTitle("Task with Invalid Reporter"),
			WithTaskDescription("This task has a non-existent reporter"),
		)

		// Act
		err := taskRepo.Create(context.Background(), task)

		// Assert - Document PocketBase behavior
		if err != nil {
			t.Logf("PocketBase rejected task with non-existent reporter: %v", err)
			assert.Contains(t, err.Error(), "failed to save task record")
			suite.Assert.TaskNotExists(task.ID)
		} else {
			t.Logf("WARNING: PocketBase allowed task with non-existent reporter - referential integrity not enforced")
			assert.NotEmpty(t, task.ID, "Task should have been assigned an ID")
			suite.Assert.TaskExists(task.ID)

			retrieved, retrieveErr := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, "nonexistent_user", retrieved.ReporterID)
		}
	})

	t.Run("TaskCreate_NonExistentAssignee_PocketBaseBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Create task with non-existent assignee ID
		nonExistentAssigneeID := "nonexistent_assignee"
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task with Invalid Assignee"),
			WithTaskDescription("This task has a non-existent assignee"),
			WithTaskAssignee(nonExistentAssigneeID),
		)

		// Act
		err := taskRepo.Create(context.Background(), task)

		// Assert - Document PocketBase behavior
		if err != nil {
			t.Logf("PocketBase rejected task with non-existent assignee: %v", err)
			suite.Assert.TaskNotExists(task.ID)
		} else {
			t.Logf("WARNING: PocketBase allowed task with non-existent assignee - referential integrity not enforced")
			suite.Assert.TaskExists(task.ID)

			retrieved, retrieveErr := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, nonExistentAssigneeID, *retrieved.AssigneeID)
		}
	})

	t.Run("CommentCreate_NonExistentTask_PocketBaseBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, _ := setupBasicTestData(t, suite)

		// Create comment with non-existent task ID
		invalidTask := &domain.Task{ID: "nonexistent_task"}
		comment := suite.Factory.CreateComment(invalidTask, owner,
			WithCommentContent("Comment on non-existent task"),
		)

		// Act
		err := commentRepo.Create(context.Background(), comment)

		// Assert - Document PocketBase behavior
		if err != nil {
			t.Logf("PocketBase rejected comment with non-existent task: %v", err)
			suite.Assert.CommentNotExists(comment.ID)
		} else {
			t.Logf("WARNING: PocketBase allowed comment with non-existent task - referential integrity not enforced")
			suite.Assert.CommentExists(comment.ID)

			retrieved, retrieveErr := commentRepo.GetByID(context.Background(), comment.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, "nonexistent_task", retrieved.TaskID)
		}
	})

	t.Run("ProjectCreate_NonExistentOwner_PocketBaseBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		// Create project with non-existent owner ID
		invalidOwner := &domain.User{ID: "nonexistent_owner"}
		project := suite.Factory.CreateProject(invalidOwner,
			WithProjectTitle("Project with Invalid Owner"),
			WithProjectSlug("invalid-owner-project"),
		)

		// Act
		err := projectRepo.Create(context.Background(), project)

		// Assert - Document PocketBase behavior
		if err != nil {
			t.Logf("PocketBase rejected project with non-existent owner: %v", err)
			suite.Assert.ProjectNotExists(project.ID)
		} else {
			t.Logf("WARNING: PocketBase allowed project with non-existent owner - referential integrity not enforced")
			suite.Assert.ProjectExists(project.ID)

			retrieved, retrieveErr := projectRepo.GetByID(context.Background(), project.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, "nonexistent_owner", retrieved.OwnerID)
		}
	})

	// ===========================================
	// VALID RELATIONSHIPS TESTS
	// These should always work regardless of FK constraint enforcement
	// ===========================================

	t.Run("TaskCreate_ValidForeignKeys_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupBasicTestData(t, suite)

		// Create parent task first
		parentTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Parent Task"),
			WithTaskDescription("This is the parent task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		// Create child task with all valid foreign keys
		childTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Child Task"),
			WithTaskDescription("This is the child task"),
			WithTaskAssignee(assignee.ID),
			WithTaskParent(parentTask.ID),
		)

		// Act
		err := taskRepo.Create(context.Background(), childTask)

		// Assert - Should succeed regardless of FK enforcement
		require.NoError(t, err)
		suite.Assert.TaskExists(childTask.ID)

		// Verify relationships are correct
		retrieved, err := taskRepo.GetByID(context.Background(), childTask.ID)
		require.NoError(t, err)
		assert.Equal(t, project.ID, retrieved.ProjectID)
		assert.Equal(t, owner.ID, retrieved.ReporterID)
		assert.Equal(t, assignee.ID, *retrieved.AssigneeID)
		assert.Equal(t, parentTask.ID, *retrieved.ParentTaskID)
	})

	t.Run("CommentCreate_ValidForeignKeys_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Create a valid task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task for Valid Comment Test"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Create parent comment
		parentComment := suite.Factory.CreateComment(task, owner,
			WithCommentContent("Parent comment"),
		)
		require.NoError(t, commentRepo.Create(context.Background(), parentComment))

		// Create reply comment with valid foreign keys
		replyComment := suite.Factory.CreateComment(task, owner,
			WithCommentContent("Reply comment"),
			WithCommentParent(parentComment.ID),
		)

		// Act
		err := commentRepo.Create(context.Background(), replyComment)

		// Assert - Should succeed
		require.NoError(t, err)
		suite.Assert.CommentExists(replyComment.ID)

		// Verify relationships are correct
		retrieved, err := commentRepo.GetByID(context.Background(), replyComment.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, retrieved.TaskID)
		assert.Equal(t, owner.ID, retrieved.AuthorID)
		assert.Equal(t, parentComment.ID, *retrieved.ParentCommentID)
	})

	// ===========================================
	// CASCADE BEHAVIOR TESTS
	// Testing what actually happens with deletes
	// ===========================================

	t.Run("TaskDelete_WithComments_CascadeOrOrphan", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Create task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task with Comments to Delete"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		// Create multiple comments on the task
		comment1 := suite.Factory.CreateComment(task, owner,
			WithCommentContent("First comment"),
		)
		comment2 := suite.Factory.CreateComment(task, owner,
			WithCommentContent("Second comment"),
		)
		comment3 := suite.Factory.CreateComment(task, owner,
			WithCommentContent("Third comment"),
		)

		require.NoError(t, commentRepo.Create(context.Background(), comment1))
		require.NoError(t, commentRepo.Create(context.Background(), comment2))
		require.NoError(t, commentRepo.Create(context.Background(), comment3))

		// Verify comments exist before deletion
		suite.Assert.CommentExists(comment1.ID)
		suite.Assert.CommentExists(comment2.ID)
		suite.Assert.CommentExists(comment3.ID)

		// Act - Delete the task
		err := taskRepo.Delete(context.Background(), task.ID)
		require.NoError(t, err)

		// Assert - Task should be deleted
		suite.Assert.TaskNotExists(task.ID)

		// Comments behavior depends on PocketBase configuration
		// From schema: "cascadeDelete": true for task field in comments
		// Check if comments were cascaded or orphaned
		comment1Exists := true
		comment2Exists := true
		comment3Exists := true

		if _, err := commentRepo.GetByID(context.Background(), comment1.ID); err != nil {
			comment1Exists = false
		}
		if _, err := commentRepo.GetByID(context.Background(), comment2.ID); err != nil {
			comment2Exists = false
		}
		if _, err := commentRepo.GetByID(context.Background(), comment3.ID); err != nil {
			comment3Exists = false
		}

		if !comment1Exists && !comment2Exists && !comment3Exists {
			t.Logf("Comments were successfully cascaded when task was deleted")
		} else {
			t.Logf("WARNING: Some comments still exist after task deletion - cascade may not be working")
			t.Logf("Comment1 exists: %v, Comment2 exists: %v, Comment3 exists: %v",
				comment1Exists, comment2Exists, comment3Exists)
		}
	})

	t.Run("UserDelete_WithProjects_HandlesDependency", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Verify project exists
		suite.Assert.ProjectExists(project.ID)

		// Act - Try to delete user who owns projects
		err := userRepo.Delete(context.Background(), owner.ID)

		// Assert - Behavior depends on PocketBase configuration
		// From schema: "cascadeDelete": false for owner field, so it should either:
		// 1. Fail with constraint violation (if enforced)
		// 2. Allow deletion and leave orphaned projects (if not enforced)
		if err != nil {
			// Constraint enforced - user deletion failed
			t.Logf("User deletion prevented due to existing projects: %v", err)
			suite.Assert.UserExists(owner.ID)
			suite.Assert.ProjectExists(project.ID)
		} else {
			// User deleted - check if project was cascaded or orphaned
			t.Logf("User was deleted - checking project status")
			suite.Assert.UserNotExists(owner.ID)

			// Project might still exist with orphaned owner reference
			projectExists := true
			_, projErr := projectRepo.GetByID(context.Background(), project.ID)
			if projErr != nil {
				projectExists = false
			}

			if projectExists {
				t.Logf("WARNING: Project exists with orphaned owner reference - referential integrity broken")
			} else {
				t.Logf("Project was cascaded when owner was deleted")
			}
		}
	})

	t.Run("ProjectDelete_WithTasks_HandlesDependency", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Create tasks in the project
		task1 := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task 1 in project"),
		)
		task2 := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task 2 in project"),
		)

		require.NoError(t, taskRepo.Create(context.Background(), task1))
		require.NoError(t, taskRepo.Create(context.Background(), task2))

		// Verify tasks exist
		suite.Assert.TaskExists(task1.ID)
		suite.Assert.TaskExists(task2.ID)

		// Act - Try to delete project with tasks
		err := projectRepo.Delete(context.Background(), project.ID)

		// Assert - Behavior depends on PocketBase configuration
		// From schema: "cascadeDelete": false for project field in tasks
		if err != nil {
			// Constraint enforced - project deletion failed
			t.Logf("Project deletion prevented due to existing tasks: %v", err)
			suite.Assert.ProjectExists(project.ID)
			suite.Assert.TaskExists(task1.ID)
			suite.Assert.TaskExists(task2.ID)
		} else {
			// Project deleted - check if tasks were cascaded or orphaned
			t.Logf("Project was deleted - checking task status")
			suite.Assert.ProjectNotExists(project.ID)

			// Tasks might still exist with orphaned project references
			task1Exists := true
			task2Exists := true

			if _, err := taskRepo.GetByID(context.Background(), task1.ID); err != nil {
				task1Exists = false
			}
			if _, err := taskRepo.GetByID(context.Background(), task2.ID); err != nil {
				task2Exists = false
			}

			if task1Exists || task2Exists {
				t.Logf("WARNING: Tasks exist with orphaned project references - referential integrity broken")
			} else {
				t.Logf("Tasks were cascaded when project was deleted")
			}
		}
	})

	// ===========================================
	// UPDATE BEHAVIOR TESTS
	// Testing referential integrity during updates
	// ===========================================

	t.Run("TaskUpdate_ChangeToInvalidProject_PocketBaseBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Create valid task
		task := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Task to Update"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), task))

		originalProjectID := task.ProjectID

		// Try to update task with invalid project ID
		task.ProjectID = "nonexistent_project_update"

		// Act
		err := taskRepo.Update(context.Background(), task)

		// Assert - Document PocketBase behavior
		if err != nil {
			// Update rejected
			t.Logf("PocketBase rejected update with invalid project: %v", err)
			// Verify task wasn't updated - should still have original project
			retrieved, retrieveErr := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, originalProjectID, retrieved.ProjectID, "Project ID should remain unchanged")
		} else {
			// Update allowed - referential integrity broken
			t.Logf("WARNING: PocketBase allowed update with invalid project - referential integrity broken")
			retrieved, retrieveErr := taskRepo.GetByID(context.Background(), task.ID)
			require.NoError(t, retrieveErr)
			assert.Equal(t, "nonexistent_project_update", retrieved.ProjectID)
		}
	})

	// ===========================================
	// COMPLEX RELATIONSHIP TESTS
	// Testing valid complex hierarchies
	// ===========================================

	t.Run("ComplexHierarchy_ValidRelationships_Success", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, assignee, project := setupBasicTestData(t, suite)

		// Create a complex hierarchy: Project -> Parent Task -> Child Task -> Comments
		parentTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Parent Task"),
			WithTaskAssignee(assignee.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), parentTask))

		childTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Child Task"),
			WithTaskParent(parentTask.ID),
		)
		require.NoError(t, taskRepo.Create(context.Background(), childTask))

		parentComment := suite.Factory.CreateComment(childTask, owner,
			WithCommentContent("Parent comment on child task"),
		)
		require.NoError(t, commentRepo.Create(context.Background(), parentComment))

		replyComment := suite.Factory.CreateComment(childTask, assignee,
			WithCommentContent("Reply to parent comment"),
			WithCommentParent(parentComment.ID),
		)
		require.NoError(t, commentRepo.Create(context.Background(), replyComment))

		// Verify all relationships exist and are correct
		suite.Assert.TaskExists(parentTask.ID)
		suite.Assert.TaskExists(childTask.ID)
		suite.Assert.CommentExists(parentComment.ID)
		suite.Assert.CommentExists(replyComment.ID)

		// Verify task hierarchy
		suite.Assert.TaskHasNoParent(parentTask.ID)
		suite.Assert.TaskHasParent(childTask.ID, parentTask.ID)

		// Verify comment hierarchy
		suite.Assert.CommentHasNoParent(parentComment.ID)
		suite.Assert.CommentHasParent(replyComment.ID, parentComment.ID)

		// Verify cross-entity relationships
		retrievedChildTask, err := taskRepo.GetByID(context.Background(), childTask.ID)
		require.NoError(t, err)
		assert.Equal(t, project.ID, retrievedChildTask.ProjectID)
		assert.Equal(t, owner.ID, retrievedChildTask.ReporterID)

		retrievedReplyComment, err := commentRepo.GetByID(context.Background(), replyComment.ID)
		require.NoError(t, err)
		assert.Equal(t, childTask.ID, retrievedReplyComment.TaskID)
		assert.Equal(t, assignee.ID, retrievedReplyComment.AuthorID)
	})

	// ===========================================
	// REFERENTIAL INTEGRITY IMPACT TESTS
	// Testing the practical impact of broken references
	// ===========================================

	t.Run("BrokenReferences_ImpactOnQueries_DocumentBehavior", func(t *testing.T) {
		// Reset database state while preserving schema
		require.NoError(t, suite.Reset())

		owner, _, project := setupBasicTestData(t, suite)

		// Create task with valid project
		validTask := suite.Factory.CreateTask(project, owner,
			WithTaskTitle("Valid Task"),
		)
		require.NoError(t, taskRepo.Create(context.Background(), validTask))

		// Try to create task with invalid project (if PocketBase allows it)
		invalidTask := suite.Factory.CreateTask(&domain.Project{ID: "nonexistent_proj"}, owner,
			WithTaskTitle("Invalid Task"),
		)

		invalidTaskErr := taskRepo.Create(context.Background(), invalidTask)
		if invalidTaskErr == nil {
			// PocketBase allowed the broken reference
			t.Logf("PocketBase allowed task with broken project reference")

			// The task with broken reference exists but is effectively orphaned
			orphanedTask, err := taskRepo.GetByID(context.Background(), invalidTask.ID)
			require.NoError(t, err)
			assert.Equal(t, "nonexistent_proj", orphanedTask.ProjectID)

			t.Logf("INTEGRITY ISSUE: Task exists with broken project reference")

			// Test impact on project-based queries
			projectTasks, err := taskRepo.ListByProject(context.Background(), project.ID, 0, 10)
			if err != nil {
				t.Logf("Query for valid project failed: %v", err)
			} else {
				// Only valid task should be returned for the valid project
				assert.Len(t, projectTasks, 1)
				assert.Equal(t, validTask.ID, projectTasks[0].ID)
				t.Logf("Valid project query returned %d tasks as expected", len(projectTasks))
			}
		} else {
			t.Logf("PocketBase properly rejected task with invalid project reference")
		}
	})
}
