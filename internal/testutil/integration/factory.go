//go:build integration
// +build integration

package integration

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"simple-easy-tasks/internal/domain"
)

// TestDataFactory provides consistent test data creation with deterministic patterns
type TestDataFactory struct {
	testDB  *TestDatabase
	counter int64
}

// NewTestDataFactory creates a new test data factory
func NewTestDataFactory(testDB *TestDatabase) *TestDataFactory {
	// Use current nanoseconds to ensure different factory instances start with different counters
	// This maintains deterministic behavior within the same factory while avoiding conflicts between factories
	initialCounter := time.Now().UnixNano() % 1000000
	return &TestDataFactory{
		testDB:  testDB,
		counter: initialCounter,
	}
}

// nextID generates a deterministic ID based on counter
func (f *TestDataFactory) nextID() int64 {
	return atomic.AddInt64(&f.counter, 1)
}

// nextUUID generates a deterministic UUID for testing
func (f *TestDataFactory) nextUUID() string {
	return uuid.New().String()
}

// nextRecordID generates a record ID for PocketBase collections (exactly 15 chars)
func (f *TestDataFactory) nextRecordID() string {
	id := f.nextID()
	// PocketBase record IDs must be exactly 15 characters
	return fmt.Sprintf("record%09d", id)
}

// UserOverride allows customization of user creation
type UserOverride func(*domain.User)

// WithUserEmail sets a custom email
func WithUserEmail(email string) UserOverride {
	return func(u *domain.User) {
		u.Email = email
	}
}

// WithUserUsername sets a custom username
func WithUserUsername(username string) UserOverride {
	return func(u *domain.User) {
		u.Username = username
	}
}

// WithUserName sets a custom name
func WithUserName(name string) UserOverride {
	return func(u *domain.User) {
		u.Name = name
	}
}

// WithUserRole sets a custom role
func WithUserRole(role domain.UserRole) UserOverride {
	return func(u *domain.User) {
		u.Role = role
	}
}

// WithUserID sets a custom ID
func WithUserID(id string) UserOverride {
	return func(u *domain.User) {
		u.ID = id
	}
}

// CreateUser creates a test user with optional overrides
func (f *TestDataFactory) CreateUser(overrides ...UserOverride) *domain.User {
	id := f.nextID()
	now := time.Now().UTC()

	user := &domain.User{
		ID:       f.nextRecordID(),
		Email:    fmt.Sprintf("user_%d@test.example.com", id),
		Username: fmt.Sprintf("testuser_%d", id),
		Name:     fmt.Sprintf("Test User %d", id),
		Role:     domain.RegularUserRole,
		Preferences: domain.UserPreferences{
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
			Preferences: map[string]string{
				"notifications": "true",
				"email_digest":  "weekly",
			},
		},
		Avatar:       "",
		TokenVersion: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Set a default password hash for testing
	_ = user.SetPassword("testpassword123")

	// Apply overrides
	for _, override := range overrides {
		override(user)
	}

	return user
}

// ProjectOverride allows customization of project creation
type ProjectOverride func(*domain.Project)

// WithProjectTitle sets a custom title
func WithProjectTitle(title string) ProjectOverride {
	return func(p *domain.Project) {
		p.Title = title
	}
}

// WithProjectSlug sets a custom slug
func WithProjectSlug(slug string) ProjectOverride {
	return func(p *domain.Project) {
		p.Slug = slug
	}
}

// WithProjectDescription sets a custom description
func WithProjectDescription(description string) ProjectOverride {
	return func(p *domain.Project) {
		p.Description = description
	}
}

// WithProjectOwner sets a custom owner
func WithProjectOwner(ownerID string) ProjectOverride {
	return func(p *domain.Project) {
		p.OwnerID = ownerID
	}
}

// WithProjectMembers sets custom member IDs
func WithProjectMembers(memberIDs []string) ProjectOverride {
	return func(p *domain.Project) {
		p.MemberIDs = memberIDs
	}
}

// WithProjectStatus sets a custom status
func WithProjectStatus(status domain.ProjectStatus) ProjectOverride {
	return func(p *domain.Project) {
		p.Status = status
	}
}

// WithProjectID sets a custom ID
func WithProjectID(id string) ProjectOverride {
	return func(p *domain.Project) {
		p.ID = id
	}
}

// CreateProject creates a test project with the given owner and optional overrides
func (f *TestDataFactory) CreateProject(owner *domain.User, overrides ...ProjectOverride) *domain.Project {
	id := f.nextID()
	now := time.Now().UTC()

	project := &domain.Project{
		ID:          f.nextRecordID(),
		Title:       fmt.Sprintf("Test Project %d", id),
		Slug:        fmt.Sprintf("test-project-%d", id),
		Description: fmt.Sprintf("Test project description for project %d", id),
		OwnerID:     owner.ID,
		Status:      domain.ActiveProject,
		Settings: domain.ProjectSettings{
			CustomFields:   make(map[string]string),
			Notifications:  make(map[string]bool),
			IsPrivate:      false,
			AllowGuestView: true,
			EnableComments: true,
		},
		MemberIDs: []string{},
		Color:     "#3B82F6", // Default blue color
		Icon:      "📝",       // Default icon
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Apply overrides
	for _, override := range overrides {
		override(project)
	}

	return project
}

// TaskOverride allows customization of task creation
type TaskOverride func(*domain.Task)

// WithTaskTitle sets a custom title
func WithTaskTitle(title string) TaskOverride {
	return func(t *domain.Task) {
		t.Title = title
	}
}

// WithTaskDescription sets a custom description
func WithTaskDescription(description string) TaskOverride {
	return func(t *domain.Task) {
		t.Description = description
	}
}

// WithTaskStatus sets a custom status
func WithTaskStatus(status domain.TaskStatus) TaskOverride {
	return func(t *domain.Task) {
		t.Status = status
	}
}

// WithTaskPriority sets a custom priority
func WithTaskPriority(priority domain.TaskPriority) TaskOverride {
	return func(t *domain.Task) {
		t.Priority = priority
	}
}

// WithTaskAssignee sets a custom assignee
func WithTaskAssignee(assigneeID string) TaskOverride {
	return func(t *domain.Task) {
		t.AssigneeID = &assigneeID
	}
}

// WithTaskParent sets a parent task
func WithTaskParent(parentID string) TaskOverride {
	return func(t *domain.Task) {
		t.ParentTaskID = &parentID
	}
}

// WithTaskDueDate sets a custom due date
func WithTaskDueDate(dueDate time.Time) TaskOverride {
	return func(t *domain.Task) {
		t.DueDate = &dueDate
	}
}

// WithTaskProgress sets custom progress
func WithTaskProgress(progress int) TaskOverride {
	return func(t *domain.Task) {
		t.Progress = progress
	}
}

// WithTaskID sets a custom ID
func WithTaskID(id string) TaskOverride {
	return func(t *domain.Task) {
		t.ID = id
	}
}

// CreateTask creates a test task with the given project and reporter, with optional overrides
func (f *TestDataFactory) CreateTask(project *domain.Project, reporter *domain.User, overrides ...TaskOverride) *domain.Task {
	id := f.nextID()
	now := time.Now().UTC()

	task := &domain.Task{
		ID:           f.nextRecordID(),
		Title:        fmt.Sprintf("Test Task %d", id),
		Description:  fmt.Sprintf("Test task description for task %d", id),
		ProjectID:    project.ID,
		ReporterID:   reporter.ID,
		Status:       domain.StatusBacklog,
		Priority:     domain.PriorityMedium,
		Progress:     0,
		TimeSpent:    0.0,
		Position:     int(id), // Use counter as position for ordering
		Tags:         []string{},
		Dependencies: []string{},
		Attachments:  []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Apply overrides
	for _, override := range overrides {
		override(task)
	}

	return task
}

// CommentOverride allows customization of comment creation
type CommentOverride func(*domain.Comment)

// WithCommentContent sets a custom content
func WithCommentContent(content string) CommentOverride {
	return func(c *domain.Comment) {
		c.Content = content
	}
}

// WithCommentType sets a custom type
func WithCommentType(commentType domain.CommentType) CommentOverride {
	return func(c *domain.Comment) {
		c.Type = commentType
	}
}

// WithCommentParent sets a parent comment for threading
func WithCommentParent(parentID string) CommentOverride {
	return func(c *domain.Comment) {
		c.ParentCommentID = &parentID
	}
}

// WithCommentID sets a custom ID
func WithCommentID(id string) CommentOverride {
	return func(c *domain.Comment) {
		c.ID = id
	}
}

// CreateComment creates a test comment with the given task and author, with optional overrides
func (f *TestDataFactory) CreateComment(task *domain.Task, author *domain.User, overrides ...CommentOverride) *domain.Comment {
	id := f.nextID()
	now := time.Now().UTC()

	comment := &domain.Comment{
		ID:          f.nextRecordID(),
		Content:     fmt.Sprintf("Test comment content for comment %d", id),
		TaskID:      task.ID,
		AuthorID:    author.ID,
		Type:        domain.CommentTypeRegular,
		IsEdited:    false,
		Attachments: []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Apply overrides
	for _, override := range overrides {
		override(comment)
	}

	return comment
}

// CreateUserWithProject creates a user and their project in one step
func (f *TestDataFactory) CreateUserWithProject(userOverrides []UserOverride, projectOverrides []ProjectOverride) (*domain.User, *domain.Project) {
	user := f.CreateUser(userOverrides...)
	project := f.CreateProject(user, projectOverrides...)
	return user, project
}

// CreateFullTaskStructure creates a complete task structure: user, project, task, and comment
func (f *TestDataFactory) CreateFullTaskStructure() (*domain.User, *domain.Project, *domain.Task, *domain.Comment) {
	user := f.CreateUser()
	project := f.CreateProject(user)
	task := f.CreateTask(project, user)
	comment := f.CreateComment(task, user)
	return user, project, task, comment
}

// CreateTaskHierarchy creates a parent task with a child task
func (f *TestDataFactory) CreateTaskHierarchy(project *domain.Project, reporter *domain.User) (*domain.Task, *domain.Task) {
	parentTask := f.CreateTask(project, reporter,
		WithTaskTitle("Parent Task"),
		WithTaskDescription("Parent task description"))

	childTask := f.CreateTask(project, reporter,
		WithTaskTitle("Child Task"),
		WithTaskDescription("Child task description"),
		WithTaskParent(parentTask.ID))

	return parentTask, childTask
}

// CreateCommentThread creates a comment with replies for threading tests
func (f *TestDataFactory) CreateCommentThread(task *domain.Task, author *domain.User, replyCount int) (*domain.Comment, []*domain.Comment) {
	parentComment := f.CreateComment(task, author,
		WithCommentContent("Parent comment for thread"))

	var replies []*domain.Comment
	for i := 0; i < replyCount; i++ {
		reply := f.CreateComment(task, author,
			WithCommentContent(fmt.Sprintf("Reply %d to parent comment", i+1)),
			WithCommentParent(parentComment.ID))
		replies = append(replies, reply)
	}

	return parentComment, replies
}
