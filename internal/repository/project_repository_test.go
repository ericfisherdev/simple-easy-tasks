package repository

//nolint:gofumpt
import (
	"context"
	"errors"
	"testing"
	"time"

	"simple-easy-tasks/internal/domain"
)

// MockProjectRepository is a mock implementation of ProjectRepository for testing.
type MockProjectRepository struct {
	CreateFunc            func(ctx context.Context, project *domain.Project) error
	GetByIDFunc           func(ctx context.Context, id string) (*domain.Project, error)
	GetBySlugFunc         func(ctx context.Context, slug string) (*domain.Project, error)
	UpdateFunc            func(ctx context.Context, project *domain.Project) error
	DeleteFunc            func(ctx context.Context, id string) error
	ListFunc              func(ctx context.Context, offset, limit int) ([]*domain.Project, error)
	ListByOwnerFunc       func(ctx context.Context, ownerID string, offset, limit int) ([]*domain.Project, error)
	ListByMemberFunc      func(ctx context.Context, memberID string, offset, limit int) ([]*domain.Project, error)
	CountFunc             func(ctx context.Context) (int, error)
	ExistsBySlugFunc      func(ctx context.Context, slug string) (bool, error)
	GetMemberProjectsFunc func(ctx context.Context, userID string, offset, limit int) ([]*domain.Project, error)
}

func (m *MockProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockProjectRepository) GetBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	if m.GetBySlugFunc != nil {
		return m.GetBySlugFunc(ctx, slug)
	}
	return nil, errors.New("not implemented")
}

func (m *MockProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, project)
	}
	return nil
}

func (m *MockProjectRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockProjectRepository) List(ctx context.Context, offset, limit int) ([]*domain.Project, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, offset, limit)
	}
	return nil, nil
}

func (m *MockProjectRepository) ListByOwner(
	ctx context.Context, ownerID string, offset, limit int,
) ([]*domain.Project, error) {
	if m.ListByOwnerFunc != nil {
		return m.ListByOwnerFunc(ctx, ownerID, offset, limit)
	}
	return nil, nil
}

func (m *MockProjectRepository) ListByMember(
	ctx context.Context, memberID string, offset, limit int,
) ([]*domain.Project, error) {
	if m.ListByMemberFunc != nil {
		return m.ListByMemberFunc(ctx, memberID, offset, limit)
	}
	return nil, nil
}

func (m *MockProjectRepository) Count(ctx context.Context) (int, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

func (m *MockProjectRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.ExistsBySlugFunc != nil {
		return m.ExistsBySlugFunc(ctx, slug)
	}
	return false, nil
}

func (m *MockProjectRepository) GetMemberProjects(
	ctx context.Context, userID string, offset, limit int,
) ([]*domain.Project, error) {
	if m.GetMemberProjectsFunc != nil {
		return m.GetMemberProjectsFunc(ctx, userID, offset, limit)
	}
	return nil, nil
}

// TestProjectRepositoryInterface ensures MockProjectRepository implements ProjectRepository
func TestProjectRepositoryInterface(_ *testing.T) {
	var _ ProjectRepository = (*MockProjectRepository)(nil)
}

// TestMockProjectRepository_Create tests the Create method
func TestMockProjectRepository_Create(t *testing.T) {
	tests := []struct {
		project *domain.Project
		mock    func() *MockProjectRepository
		name    string
		wantErr bool
	}{
		{
			name: "successful create",
			project: &domain.Project{
				ID:          "proj-123",
				Title:       "Test Project",
				Description: "A test project",
				Slug:        "test-project",
				OwnerID:     "user-123",
				Owner: &domain.User{
					ID:   "user-123",
					Name: "Test User",
				},
				Status:    domain.ActiveProject,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					CreateFunc: func(_ context.Context, project *domain.Project) error {
						if project.Slug != "test-project" {
							t.Errorf("expected slug %s, got %s", "test-project", project.Slug)
						}
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "duplicate slug error",
			project: &domain.Project{
				ID:   "proj-456",
				Slug: "existing-project",
			},
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					CreateFunc: func(_ context.Context, _ *domain.Project) error {
						return errors.New("project with slug already exists")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			err := repo.Create(context.Background(), tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMockProjectRepository_GetBySlug tests the GetBySlug method
func TestMockProjectRepository_GetBySlug(t *testing.T) {
	tests := []struct {
		mock        func() *MockProjectRepository
		name        string
		slug        string
		wantProject bool
		wantErr     bool
	}{
		{
			name: "project found",
			slug: "test-project",
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					GetBySlugFunc: func(_ context.Context, slug string) (*domain.Project, error) {
						return &domain.Project{
							ID:    "proj-123",
							Title: "Test Project",
							Slug:  slug,
						}, nil
					},
				}
			},
			wantProject: true,
			wantErr:     false,
		},
		{
			name: "project not found",
			slug: "nonexistent-project",
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					GetBySlugFunc: func(_ context.Context, _ string) (*domain.Project, error) {
						return nil, errors.New("project not found")
					},
				}
			},
			wantProject: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			project, err := repo.GetBySlug(context.Background(), tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBySlug() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantProject && project == nil {
				t.Error("GetBySlug() expected project, got nil")
			}
			if !tt.wantProject && project != nil {
				t.Error("GetBySlug() expected nil project, got project")
			}
		})
	}
}

// TestMockProjectRepository_GetMemberProjects tests the GetMemberProjects method
func TestMockProjectRepository_GetMemberProjects(t *testing.T) {
	tests := []struct {
		mock         func() *MockProjectRepository
		name         string
		userID       string
		offset       int
		limit        int
		wantProjects bool
		wantErr      bool
	}{
		{
			name:   "user has projects",
			userID: "user-123",
			offset: 0,
			limit:  10,
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					GetMemberProjectsFunc: func(_ context.Context, _ string, _, _ int) ([]*domain.Project, error) {
						return []*domain.Project{
							{
								ID:    "proj-1",
								Title: "Project 1",
							},
							{
								ID:    "proj-2",
								Title: "Project 2",
							},
						}, nil
					},
				}
			},
			wantProjects: true,
			wantErr:      false,
		},
		{
			name:   "user has no projects",
			userID: "user-456",
			offset: 0,
			limit:  10,
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					GetMemberProjectsFunc: func(_ context.Context, _ string, _, _ int) ([]*domain.Project, error) {
						return []*domain.Project{}, nil
					},
				}
			},
			wantProjects: false,
			wantErr:      false,
		},
		{
			name:   "database error",
			userID: "user-789",
			offset: 0,
			limit:  10,
			mock: func() *MockProjectRepository {
				return &MockProjectRepository{
					GetMemberProjectsFunc: func(_ context.Context, _ string, _, _ int) ([]*domain.Project, error) {
						return nil, errors.New("database connection failed")
					},
				}
			},
			wantProjects: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			projects, err := repo.GetMemberProjects(context.Background(), tt.userID, tt.offset, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMemberProjects() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantProjects && len(projects) == 0 {
				t.Error("GetMemberProjects() expected projects, got none")
			}
			if !tt.wantProjects && len(projects) > 0 {
				t.Error("GetMemberProjects() expected no projects, got some")
			}
		})
	}
}
