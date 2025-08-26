// Package testutil provides testing utilities and mock implementations.
package testutil

//nolint:gofumpt
import (
	"context"
	"fmt"
	"strings"
	"sync"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// MockUserRepository implements UserRepository for testing.
type MockUserRepository struct {
	users map[string]*domain.User
	mu    sync.RWMutex
}

// NewMockUserRepository creates a new mock user repository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*domain.User),
	}
}

// Create creates a new user.
func (m *MockUserRepository) Create(_ context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing email or username
	for _, existingUser := range m.users {
		if existingUser.Email == user.Email {
			return domain.NewConflictError("EMAIL_EXISTS", "Email already exists")
		}
		if existingUser.Username == user.Username {
			return domain.NewConflictError("USERNAME_EXISTS", "Username already exists")
		}
	}

	m.users[user.ID] = user
	return nil
}

// GetByID retrieves a user by ID.
func (m *MockUserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}
	return user, nil
}

// GetByEmail retrieves a user by email.
func (m *MockUserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
}

// GetByUsername retrieves a user by username.
func (m *MockUserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
}

// Update updates an existing user.
func (m *MockUserRepository) Update(_ context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[user.ID]; !exists {
		return domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	m.users[user.ID] = user
	return nil
}

// Delete deletes a user by ID.
func (m *MockUserRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[id]; !exists {
		return domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	delete(m.users, id)
	return nil
}

// List retrieves users with pagination.
func (m *MockUserRepository) List(_ context.Context, offset, limit int) ([]*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*domain.User, 0)
	i := 0
	for _, user := range m.users {
		if i >= offset && len(users) < limit {
			users = append(users, user)
		}
		i++
	}
	return users, nil
}

// Count returns the total number of users.
func (m *MockUserRepository) Count(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.users), nil
}

// ExistsByEmail checks if a user exists with the given email.
func (m *MockUserRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, err := m.GetByEmail(context.Background(), email)
	if err != nil {
		if domainErr, ok := err.(*domain.Error); ok && domainErr.Type == domain.NotFoundError {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ExistsByUsername checks if a user exists with the given username.
func (m *MockUserRepository) ExistsByUsername(_ context.Context, username string) (bool, error) {
	_, err := m.GetByUsername(context.Background(), username)
	if err != nil {
		if domainErr, ok := err.(*domain.Error); ok && domainErr.Type == domain.NotFoundError {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// MockProjectRepository implements ProjectRepository for testing.
type MockProjectRepository struct {
	projects map[string]*domain.Project
	mu       sync.RWMutex
}

// NewMockProjectRepository creates a new mock project repository.
func NewMockProjectRepository() *MockProjectRepository {
	return &MockProjectRepository{
		projects: make(map[string]*domain.Project),
	}
}

// Create creates a new project.
func (m *MockProjectRepository) Create(_ context.Context, project *domain.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing slug
	for _, existingProject := range m.projects {
		if existingProject.Slug == project.Slug {
			return domain.NewConflictError("SLUG_EXISTS", "Project slug already exists")
		}
	}

	m.projects[project.ID] = project
	return nil
}

// GetByID retrieves a project by ID.
func (m *MockProjectRepository) GetByID(_ context.Context, id string) (*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	project, exists := m.projects[id]
	if !exists {
		return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}
	return project, nil
}

// GetBySlug retrieves a project by slug.
func (m *MockProjectRepository) GetBySlug(_ context.Context, slug string) (*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, project := range m.projects {
		if project.Slug == slug {
			return project, nil
		}
	}
	return nil, domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
}

// Update updates an existing project.
func (m *MockProjectRepository) Update(_ context.Context, project *domain.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.projects[project.ID]; !exists {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	m.projects[project.ID] = project
	return nil
}

// Delete deletes a project by ID.
func (m *MockProjectRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.projects[id]; !exists {
		return domain.NewNotFoundError("PROJECT_NOT_FOUND", "Project not found")
	}

	delete(m.projects, id)
	return nil
}

// ListByOwner retrieves projects owned by a specific user.
func (m *MockProjectRepository) ListByOwner(
	_ context.Context,
	ownerID string,
	offset,
	limit int,
) ([]*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*domain.Project, 0)
	i := 0
	for _, project := range m.projects {
		if project.OwnerID == ownerID {
			if i >= offset && len(projects) < limit {
				projects = append(projects, project)
			}
			i++
		}
	}
	return projects, nil
}

// ListByMember retrieves projects where a user is a member.
func (m *MockProjectRepository) ListByMember(
	_ context.Context,
	memberID string,
	offset,
	limit int,
) ([]*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*domain.Project, 0)
	i := 0
	for _, project := range m.projects {
		if project.IsMember(memberID) {
			if i >= offset && len(projects) < limit {
				projects = append(projects, project)
			}
			i++
		}
	}
	return projects, nil
}

// List retrieves projects with pagination.
func (m *MockProjectRepository) List(_ context.Context, offset, limit int) ([]*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*domain.Project, 0)
	i := 0
	for _, project := range m.projects {
		if i >= offset && len(projects) < limit {
			projects = append(projects, project)
		}
		i++
	}
	return projects, nil
}

// Count returns the total number of projects.
func (m *MockProjectRepository) Count(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.projects), nil
}

// ExistsBySlug checks if a project exists with the given slug.
func (m *MockProjectRepository) ExistsBySlug(_ context.Context, slug string) (bool, error) {
	_, err := m.GetBySlug(context.Background(), slug)
	if err != nil {
		if domainErr, ok := err.(*domain.Error); ok && domainErr.Type == domain.NotFoundError {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetMemberProjects retrieves all projects where user has access.
func (m *MockProjectRepository) GetMemberProjects(
	_ context.Context,
	userID string,
	offset,
	limit int,
) ([]*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*domain.Project, 0)
	i := 0
	for _, project := range m.projects {
		if project.HasAccess(userID) {
			if i >= offset && len(projects) < limit {
				projects = append(projects, project)
			}
			i++
		}
	}
	return projects, nil
}

// AddUser adds a user to mock repository for testing.
func (m *MockUserRepository) AddUser(user *domain.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

// AddProject adds a project to mock repository for testing.
func (m *MockProjectRepository) AddProject(project *domain.Project) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[project.ID] = project
}

// MockTaskRepository implements TaskRepository for testing enhanced functionality.
type MockTaskRepository struct {
	Tasks                      map[string]*domain.Task
	SubtasksByParent          map[string][]*domain.Task
	DependenciesByTask        map[string][]*domain.Task
	MoveCallLog               map[string]bool
	SubtasksDuplicated        map[string]bool
	ForceCreateError          bool
	ForceGetSubtasksError     bool
	ForceGetDependenciesError bool
	ForceMoveError            bool
	mu                        sync.RWMutex
}

// NewMockTaskRepository creates a new mock task repository.
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		Tasks:                make(map[string]*domain.Task),
		SubtasksByParent:     make(map[string][]*domain.Task),
		DependenciesByTask:   make(map[string][]*domain.Task),
		MoveCallLog:          make(map[string]bool),
		SubtasksDuplicated:   make(map[string]bool),
	}
}

// AddTask adds a task to mock repository for testing.
func (m *MockTaskRepository) AddTask(task *domain.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Tasks[task.ID] = task
}

// Create creates a new task.
func (m *MockTaskRepository) Create(_ context.Context, task *domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ForceCreateError {
		return domain.NewInternalError("MOCK_CREATE_ERROR", "Forced create error for testing", nil)
	}

	// Generate ID if not set
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", len(m.Tasks)+1)
	}

	m.Tasks[task.ID] = task
	return nil
}

// GetByID retrieves a task by ID.
func (m *MockTaskRepository) GetByID(_ context.Context, id string) (*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.Tasks[id]
	if !exists {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}
	return task, nil
}

// Update updates an existing task.
func (m *MockTaskRepository) Update(_ context.Context, task *domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Tasks[task.ID]; !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	m.Tasks[task.ID] = task
	return nil
}

// Delete deletes a task by ID.
func (m *MockTaskRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Tasks[id]; !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	delete(m.Tasks, id)
	return nil
}

// GetByProject retrieves tasks for a specific project with advanced filtering.
func (m *MockTaskRepository) GetByProject(_ context.Context, projectID string, filters repository.TaskFilters) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	for _, task := range m.Tasks {
		if task.ProjectID != projectID {
			continue
		}

		// Apply filters
		if len(filters.Status) > 0 {
			statusMatch := false
			for _, status := range filters.Status {
				if task.Status == status {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		if filters.AssigneeID != nil {
			if task.AssigneeID == nil || *task.AssigneeID != *filters.AssigneeID {
				continue
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ListByProject retrieves tasks for a specific project (legacy method).
func (m *MockTaskRepository) ListByProject(_ context.Context, projectID string, offset, limit int) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	for _, task := range m.Tasks {
		if task.ProjectID == projectID {
			tasks = append(tasks, task)
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(tasks) {
		return []*domain.Task{}, nil
	}
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end], nil
}

// ListByAssignee retrieves tasks assigned to a specific user.
func (m *MockTaskRepository) ListByAssignee(_ context.Context, assigneeID string, offset, limit int) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	for _, task := range m.Tasks {
		if task.AssigneeID != nil && *task.AssigneeID == assigneeID {
			tasks = append(tasks, task)
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(tasks) {
		return []*domain.Task{}, nil
	}
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end], nil
}

// ListByStatus retrieves tasks by status.
func (m *MockTaskRepository) ListByStatus(_ context.Context, status domain.TaskStatus, offset, limit int) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	for _, task := range m.Tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(tasks) {
		return []*domain.Task{}, nil
	}
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end], nil
}

// ListByCreator retrieves tasks created by a specific user.
func (m *MockTaskRepository) ListByCreator(_ context.Context, creatorID string, offset, limit int) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	for _, task := range m.Tasks {
		if task.ReporterID == creatorID {
			tasks = append(tasks, task)
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(tasks) {
		return []*domain.Task{}, nil
	}
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end], nil
}

// Search searches tasks by title, description or content.
func (m *MockTaskRepository) Search(_ context.Context, query string, projectID string, offset, limit int) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	queryLower := strings.ToLower(query)

	for _, task := range m.Tasks {
		if projectID != "" && task.ProjectID != projectID {
			continue
		}

		if strings.Contains(strings.ToLower(task.Title), queryLower) ||
			strings.Contains(strings.ToLower(task.Description), queryLower) {
			tasks = append(tasks, task)
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(tasks) {
		return []*domain.Task{}, nil
	}
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end], nil
}

// GetSubtasks retrieves subtasks for a parent task.
func (m *MockTaskRepository) GetSubtasks(_ context.Context, parentID string) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ForceGetSubtasksError {
		return nil, domain.NewInternalError("MOCK_SUBTASK_ERROR", "Forced subtask error for testing", nil)
	}

	subtasks, exists := m.SubtasksByParent[parentID]
	if !exists {
		return []*domain.Task{}, nil
	}

	// Track that subtasks were retrieved for duplication testing
	m.SubtasksDuplicated[parentID] = len(subtasks) > 0

	return subtasks, nil
}

// GetDependencies retrieves dependency tasks for a task.
func (m *MockTaskRepository) GetDependencies(_ context.Context, taskID string) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ForceGetDependenciesError {
		return nil, domain.NewInternalError("MOCK_DEPENDENCY_ERROR", "Forced dependency error for testing", nil)
	}

	dependencies, exists := m.DependenciesByTask[taskID]
	if !exists {
		return []*domain.Task{}, nil
	}

	return dependencies, nil
}

// GetTasksByFilter retrieves tasks using advanced filters.
func (m *MockTaskRepository) GetTasksByFilter(_ context.Context, filters repository.TaskFilters) ([]*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*domain.Task
	for _, task := range m.Tasks {
		// Apply all filters
		if len(filters.Status) > 0 {
			statusMatch := false
			for _, status := range filters.Status {
				if task.Status == status {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		if filters.AssigneeID != nil {
			if task.AssigneeID == nil || *task.AssigneeID != *filters.AssigneeID {
				continue
			}
		}

		if filters.ReporterID != nil && task.ReporterID != *filters.ReporterID {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Count returns the total number of tasks matching criteria.
func (m *MockTaskRepository) Count(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.Tasks), nil
}

// CountByProject returns the number of tasks in a project.
func (m *MockTaskRepository) CountByProject(_ context.Context, projectID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, task := range m.Tasks {
		if task.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

// CountByAssignee returns the number of tasks assigned to a user.
func (m *MockTaskRepository) CountByAssignee(_ context.Context, assigneeID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, task := range m.Tasks {
		if task.AssigneeID != nil && *task.AssigneeID == assigneeID {
			count++
		}
	}
	return count, nil
}

// CountByStatus returns the number of tasks with a specific status.
func (m *MockTaskRepository) CountByStatus(_ context.Context, status domain.TaskStatus) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, task := range m.Tasks {
		if task.Status == status {
			count++
		}
	}
	return count, nil
}

// ExistsByID checks if a task exists by ID.
func (m *MockTaskRepository) ExistsByID(_ context.Context, id string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.Tasks[id]
	return exists, nil
}

// Move moves a task to a new status and position.
func (m *MockTaskRepository) Move(_ context.Context, taskID string, newStatus domain.TaskStatus, position int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ForceMoveError {
		return domain.NewInternalError("MOCK_MOVE_ERROR", "Forced move error for testing", nil)
	}

	task, exists := m.Tasks[taskID]
	if !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.Status = newStatus
	task.Position = position
	m.MoveCallLog[taskID] = true

	return nil
}

// BulkUpdate updates multiple tasks.
func (m *MockTaskRepository) BulkUpdate(_ context.Context, tasks []*domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range tasks {
		if _, exists := m.Tasks[task.ID]; exists {
			m.Tasks[task.ID] = task
		}
	}
	return nil
}

// BulkDelete deletes multiple tasks.
func (m *MockTaskRepository) BulkDelete(_ context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range ids {
		delete(m.Tasks, id)
	}
	return nil
}

// BulkUpdateStatus updates multiple tasks with the same status.
func (m *MockTaskRepository) BulkUpdateStatus(_ context.Context, taskIDs []string, newStatus domain.TaskStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, taskID := range taskIDs {
		if task, exists := m.Tasks[taskID]; exists {
			task.Status = newStatus
		}
	}
	return nil
}

// ArchiveTask archives a task instead of deleting it.
func (m *MockTaskRepository) ArchiveTask(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.Tasks[id]
	if !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.Archive()
	return nil
}

// UnarchiveTask unarchives a task.
func (m *MockTaskRepository) UnarchiveTask(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.Tasks[id]
	if !exists {
		return domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	task.Unarchive()
	return nil
}

// Ensure interfaces are implemented
var (
	_ repository.UserRepository    = (*MockUserRepository)(nil)
	_ repository.ProjectRepository = (*MockProjectRepository)(nil)
	_ repository.TaskRepository    = (*MockTaskRepository)(nil)
)
