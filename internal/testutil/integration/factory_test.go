//go:build integration
// +build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"simple-easy-tasks/internal/domain"
)

func TestNewTestDataFactory(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	assert.NotNil(t, factory)
	assert.Equal(t, testDB, factory.testDB)
	assert.Equal(t, int64(0), factory.counter)
}

func TestTestDataFactory_CreateUser(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	t.Run("creates user with default values", func(t *testing.T) {
		user := factory.CreateUser()

		require.NotNil(t, user)
		assert.NotEmpty(t, user.ID)
		assert.Contains(t, user.Email, "@test.example.com")
		assert.Contains(t, user.Username, "testuser_")
		assert.Contains(t, user.Name, "Test User")
		assert.Equal(t, domain.RegularUserRole, user.Role)
		assert.Equal(t, "light", user.Preferences.Theme)
		assert.Equal(t, "en", user.Preferences.Language)
		assert.Equal(t, "UTC", user.Preferences.Timezone)
		assert.NotEmpty(t, user.PasswordHash)
		assert.Equal(t, 1, user.TokenVersion)

		// Validate user
		err := user.Validate()
		assert.NoError(t, err)
	})

	t.Run("applies user overrides correctly", func(t *testing.T) {
		customEmail := "custom@example.com"
		customUsername := "customuser"
		customName := "Custom User"
		customRole := domain.AdminRole
		customID := "custom-id-123"

		user := factory.CreateUser(
			WithUserEmail(customEmail),
			WithUserUsername(customUsername),
			WithUserName(customName),
			WithUserRole(customRole),
			WithUserID(customID),
		)

		assert.Equal(t, customID, user.ID)
		assert.Equal(t, customEmail, user.Email)
		assert.Equal(t, customUsername, user.Username)
		assert.Equal(t, customName, user.Name)
		assert.Equal(t, customRole, user.Role)

		// Validate user
		err := user.Validate()
		assert.NoError(t, err)
	})

	t.Run("creates deterministic but unique users", func(t *testing.T) {
		user1 := factory.CreateUser()
		user2 := factory.CreateUser()

		// Should be different
		assert.NotEqual(t, user1.ID, user2.ID)
		assert.NotEqual(t, user1.Email, user2.Email)
		assert.NotEqual(t, user1.Username, user2.Username)

		// But should follow pattern (don't check exact numbers due to shared counter)
		assert.Contains(t, user1.Email, "@test.example.com")
		assert.Contains(t, user2.Email, "@test.example.com")
		assert.Contains(t, user1.Username, "testuser_")
		assert.Contains(t, user2.Username, "testuser_")
	})
}

func TestTestDataFactory_CreateProject(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	owner := factory.CreateUser()

	t.Run("creates project with default values", func(t *testing.T) {
		project := factory.CreateProject(owner)

		require.NotNil(t, project)
		assert.NotEmpty(t, project.ID)
		assert.Contains(t, project.Title, "Test Project")
		assert.Contains(t, project.Slug, "test-project-")
		assert.Contains(t, project.Description, "Test project description")
		assert.Equal(t, owner.ID, project.OwnerID)
		assert.Equal(t, domain.ActiveProject, project.Status)
		assert.False(t, project.Settings.IsPrivate)
		assert.True(t, project.Settings.AllowGuestView)
		assert.True(t, project.Settings.EnableComments)
		assert.Empty(t, project.MemberIDs)
		assert.Equal(t, "#3B82F6", project.Color)
		assert.Equal(t, "📝", project.Icon)

		// Validate project
		err := project.Validate()
		assert.NoError(t, err)
	})

	t.Run("applies project overrides correctly", func(t *testing.T) {
		customTitle := "Custom Project"
		customSlug := "custom-project"
		customDescription := "Custom project description"
		customMembers := []string{"member1", "member2"}
		customID := "custom-project-123"

		project := factory.CreateProject(owner,
			WithProjectTitle(customTitle),
			WithProjectSlug(customSlug),
			WithProjectDescription(customDescription),
			WithProjectMembers(customMembers),
			WithProjectStatus(domain.ArchivedProject),
			WithProjectID(customID),
		)

		assert.Equal(t, customID, project.ID)
		assert.Equal(t, customTitle, project.Title)
		assert.Equal(t, customSlug, project.Slug)
		assert.Equal(t, customDescription, project.Description)
		assert.Equal(t, customMembers, project.MemberIDs)
		assert.Equal(t, domain.ArchivedProject, project.Status)

		// Validate project
		err := project.Validate()
		assert.NoError(t, err)
	})
}

func TestTestDataFactory_CreateTask(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	owner := factory.CreateUser()
	project := factory.CreateProject(owner)
	reporter := factory.CreateUser()

	t.Run("creates task with default values", func(t *testing.T) {
		task := factory.CreateTask(project, reporter)

		require.NotNil(t, task)
		assert.NotEmpty(t, task.ID)
		assert.Contains(t, task.Title, "Test Task")
		assert.Contains(t, task.Description, "Test task description")
		assert.Equal(t, project.ID, task.ProjectID)
		assert.Equal(t, reporter.ID, task.ReporterID)
		assert.Equal(t, domain.StatusBacklog, task.Status)
		assert.Equal(t, domain.PriorityMedium, task.Priority)
		assert.Equal(t, 0, task.Progress)
		assert.Equal(t, 0.0, task.TimeSpent)
		assert.Nil(t, task.AssigneeID)
		assert.Nil(t, task.ParentTaskID)
		assert.Empty(t, task.Tags)
		assert.Empty(t, task.Dependencies)
		assert.Empty(t, task.Attachments)

		// Validate task
		err := task.Validate()
		assert.NoError(t, err)
	})

	t.Run("applies task overrides correctly", func(t *testing.T) {
		customTitle := "Custom Task"
		customDescription := "Custom task description"
		assignee := factory.CreateUser()
		dueDate := time.Now().AddDate(0, 0, 7) // One week from now
		customID := "custom-task-123"

		task := factory.CreateTask(project, reporter,
			WithTaskTitle(customTitle),
			WithTaskDescription(customDescription),
			WithTaskStatus(domain.StatusDeveloping),
			WithTaskPriority(domain.PriorityHigh),
			WithTaskAssignee(assignee.ID),
			WithTaskDueDate(dueDate),
			WithTaskProgress(50),
			WithTaskID(customID),
		)

		assert.Equal(t, customID, task.ID)
		assert.Equal(t, customTitle, task.Title)
		assert.Equal(t, customDescription, task.Description)
		assert.Equal(t, domain.StatusDeveloping, task.Status)
		assert.Equal(t, domain.PriorityHigh, task.Priority)
		assert.NotNil(t, task.AssigneeID)
		assert.Equal(t, assignee.ID, *task.AssigneeID)
		assert.NotNil(t, task.DueDate)
		assert.Equal(t, 50, task.Progress)

		// Validate task
		err := task.Validate()
		assert.NoError(t, err)
	})

	t.Run("creates task with parent relationship", func(t *testing.T) {
		parentTask := factory.CreateTask(project, reporter)
		childTask := factory.CreateTask(project, reporter,
			WithTaskParent(parentTask.ID),
		)

		assert.NotNil(t, childTask.ParentTaskID)
		assert.Equal(t, parentTask.ID, *childTask.ParentTaskID)

		// Validate both tasks
		err := parentTask.Validate()
		assert.NoError(t, err)
		err = childTask.Validate()
		assert.NoError(t, err)
	})
}

func TestTestDataFactory_CreateComment(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	owner := factory.CreateUser()
	project := factory.CreateProject(owner)
	task := factory.CreateTask(project, owner)
	author := factory.CreateUser()

	t.Run("creates comment with default values", func(t *testing.T) {
		comment := factory.CreateComment(task, author)

		require.NotNil(t, comment)
		assert.NotEmpty(t, comment.ID)
		assert.Contains(t, comment.Content, "Test comment content")
		assert.Equal(t, task.ID, comment.TaskID)
		assert.Equal(t, author.ID, comment.AuthorID)
		assert.Equal(t, domain.CommentTypeRegular, comment.Type)
		assert.False(t, comment.IsEdited)
		assert.Nil(t, comment.ParentCommentID)
		assert.Empty(t, comment.Attachments)

		// Validate comment
		err := comment.Validate()
		assert.NoError(t, err)
	})

	t.Run("applies comment overrides correctly", func(t *testing.T) {
		customContent := "Custom comment content"
		parentComment := factory.CreateComment(task, author)
		customID := "custom-comment-123"

		comment := factory.CreateComment(task, author,
			WithCommentContent(customContent),
			WithCommentType(domain.CommentTypeSystemMessage),
			WithCommentParent(parentComment.ID),
			WithCommentID(customID),
		)

		assert.Equal(t, customID, comment.ID)
		assert.Equal(t, customContent, comment.Content)
		assert.Equal(t, domain.CommentTypeSystemMessage, comment.Type)
		assert.NotNil(t, comment.ParentCommentID)
		assert.Equal(t, parentComment.ID, *comment.ParentCommentID)

		// Validate comment
		err := comment.Validate()
		assert.NoError(t, err)
	})
}

func TestTestDataFactory_CreateUserWithProject(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	user, project := factory.CreateUserWithProject(
		[]UserOverride{WithUserName("Test User")},
		[]ProjectOverride{WithProjectTitle("Test Project")},
	)

	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "Test Project", project.Title)
	assert.Equal(t, user.ID, project.OwnerID)

	// Validate both
	err := user.Validate()
	assert.NoError(t, err)
	err = project.Validate()
	assert.NoError(t, err)
}

func TestTestDataFactory_CreateFullTaskStructure(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	user, project, task, comment := factory.CreateFullTaskStructure()

	// Check relationships
	assert.Equal(t, user.ID, project.OwnerID)
	assert.Equal(t, project.ID, task.ProjectID)
	assert.Equal(t, user.ID, task.ReporterID)
	assert.Equal(t, task.ID, comment.TaskID)
	assert.Equal(t, user.ID, comment.AuthorID)

	// Validate all entities
	err := user.Validate()
	assert.NoError(t, err)
	err = project.Validate()
	assert.NoError(t, err)
	err = task.Validate()
	assert.NoError(t, err)
	err = comment.Validate()
	assert.NoError(t, err)
}

func TestTestDataFactory_CreateTaskHierarchy(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	owner := factory.CreateUser()
	project := factory.CreateProject(owner)

	parentTask, childTask := factory.CreateTaskHierarchy(project, owner)

	assert.Contains(t, parentTask.Title, "Parent Task")
	assert.Contains(t, childTask.Title, "Child Task")
	assert.NotNil(t, childTask.ParentTaskID)
	assert.Equal(t, parentTask.ID, *childTask.ParentTaskID)

	// Validate both tasks
	err := parentTask.Validate()
	assert.NoError(t, err)
	err = childTask.Validate()
	assert.NoError(t, err)
}

func TestTestDataFactory_CreateCommentThread(t *testing.T) {
	testDB := NewTestDatabase(t)
	factory := NewTestDataFactory(testDB)

	owner := factory.CreateUser()
	project := factory.CreateProject(owner)
	task := factory.CreateTask(project, owner)

	parentComment, replies := factory.CreateCommentThread(task, owner, 3)

	assert.Contains(t, parentComment.Content, "Parent comment for thread")
	assert.Len(t, replies, 3)

	for i, reply := range replies {
		assert.Contains(t, reply.Content, fmt.Sprintf("Reply %d", i+1))
		assert.NotNil(t, reply.ParentCommentID)
		assert.Equal(t, parentComment.ID, *reply.ParentCommentID)

		// Validate reply
		err := reply.Validate()
		assert.NoError(t, err)
	}

	// Validate parent comment
	err := parentComment.Validate()
	assert.NoError(t, err)
}

func TestTestDataFactory_Deterministic(t *testing.T) {
	// Test that factory creates deterministic but unique data
	testDB1 := NewTestDatabase(t)
	factory1 := NewTestDataFactory(testDB1)

	testDB2 := NewTestDatabase(t)
	factory2 := NewTestDataFactory(testDB2)

	// Create users with both factories
	user1_factory1 := factory1.CreateUser()
	user2_factory1 := factory1.CreateUser()

	user1_factory2 := factory2.CreateUser()

	// Within same factory, data should be different
	assert.NotEqual(t, user1_factory1.ID, user2_factory1.ID)
	assert.NotEqual(t, user1_factory1.Email, user2_factory1.Email)

	// Between factories, pattern should be consistent but IDs different
	assert.NotEqual(t, user1_factory1.ID, user1_factory2.ID)
	assert.Contains(t, user1_factory1.Email, "@test.example.com")
	assert.Contains(t, user1_factory2.Email, "@test.example.com")
	assert.Contains(t, user1_factory1.Username, "testuser_")
	assert.Contains(t, user1_factory2.Username, "testuser_")
}
