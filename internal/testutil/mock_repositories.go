// Package testutil provides testing utilities and mock implementations.
package testutil

//nolint:gofumpt
import (
	"context"
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

// Ensure interfaces are implemented
var (
	_ repository.UserRepository    = (*MockUserRepository)(nil)
	_ repository.ProjectRepository = (*MockProjectRepository)(nil)
)
