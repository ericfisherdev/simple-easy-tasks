package repository

//nolint:gofumpt
import (
	"context"
	"errors"
	"testing"
	"time"

	"simple-easy-tasks/internal/domain"
)

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	CreateFunc           func(ctx context.Context, user *domain.User) error
	GetByIDFunc          func(ctx context.Context, id string) (*domain.User, error)
	GetByEmailFunc       func(ctx context.Context, email string) (*domain.User, error)
	GetByUsernameFunc    func(ctx context.Context, username string) (*domain.User, error)
	UpdateFunc           func(ctx context.Context, user *domain.User) error
	DeleteFunc           func(ctx context.Context, id string) error
	ListFunc             func(ctx context.Context, offset, limit int) ([]*domain.User, error)
	CountFunc            func(ctx context.Context) (int, error)
	ExistsByEmailFunc    func(ctx context.Context, email string) (bool, error)
	ExistsByUsernameFunc func(ctx context.Context, username string) (bool, error)
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.GetByUsernameFunc != nil {
		return m.GetByUsernameFunc(ctx, username)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, offset, limit)
	}
	return nil, nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsByEmailFunc != nil {
		return m.ExistsByEmailFunc(ctx, email)
	}
	return false, nil
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.ExistsByUsernameFunc != nil {
		return m.ExistsByUsernameFunc(ctx, username)
	}
	return false, nil
}

// TestUserRepositoryInterface ensures MockUserRepository implements UserRepository
func TestUserRepositoryInterface(_ *testing.T) {
	var _ UserRepository = (*MockUserRepository)(nil)
}

// TestMockUserRepository_Create tests the Create method with various scenarios
func TestMockUserRepository_Create(t *testing.T) {
	tests := []struct {
		user    *domain.User
		mock    func() *MockUserRepository
		name    string
		wantErr bool
	}{
		{
			name: "successful create",
			user: &domain.User{
				ID:        "user-123",
				Email:     "test@example.com",
				Username:  "testuser",
				Name:      "Test User",
				Role:      domain.RegularUserRole,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					CreateFunc: func(_ context.Context, user *domain.User) error {
						if user.Email != "test@example.com" {
							t.Errorf("expected email %s, got %s", "test@example.com", user.Email)
						}
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "create with error",
			user: &domain.User{
				ID:    "user-456",
				Email: "duplicate@example.com",
			},
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					CreateFunc: func(_ context.Context, _ *domain.User) error {
						return errors.New("user already exists")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			err := repo.Create(context.Background(), tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMockUserRepository_GetByEmail tests the GetByEmail method
func TestMockUserRepository_GetByEmail(t *testing.T) {
	tests := []struct {
		mock     func() *MockUserRepository
		name     string
		email    string
		wantUser bool
		wantErr  bool
	}{
		{
			name:  "user found",
			email: "test@example.com",
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					GetByEmailFunc: func(_ context.Context, email string) (*domain.User, error) {
						return &domain.User{
							ID:       "user-123",
							Email:    email,
							Username: "testuser",
							Name:     "Test User",
						}, nil
					},
				}
			},
			wantUser: true,
			wantErr:  false,
		},
		{
			name:  "user not found",
			email: "notfound@example.com",
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					GetByEmailFunc: func(_ context.Context, _ string) (*domain.User, error) {
						return nil, errors.New("user not found")
					},
				}
			},
			wantUser: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			user, err := repo.GetByEmail(context.Background(), tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantUser && user == nil {
				t.Error("GetByEmail() expected user, got nil")
			}
			if !tt.wantUser && user != nil {
				t.Error("GetByEmail() expected nil user, got user")
			}
		})
	}
}

// TestMockUserRepository_Update tests the Update method
func TestMockUserRepository_Update(t *testing.T) {
	tests := []struct {
		user    *domain.User
		mock    func() *MockUserRepository
		name    string
		wantErr bool
	}{
		{
			name: "successful update",
			user: &domain.User{
				ID:       "user-123",
				Email:    "updated@example.com",
				Username: "updateduser",
				Name:     "Updated User",
			},
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					UpdateFunc: func(_ context.Context, user *domain.User) error {
						if user.ID != "user-123" {
							t.Errorf("expected user ID %s, got %s", "user-123", user.ID)
						}
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "update non-existent user",
			user: &domain.User{
				ID: "nonexistent",
			},
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					UpdateFunc: func(_ context.Context, _ *domain.User) error {
						return errors.New("user not found")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			err := repo.Update(context.Background(), tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMockUserRepository_ExistsByEmail tests the ExistsByEmail method
func TestMockUserRepository_ExistsByEmail(t *testing.T) {
	tests := []struct {
		mock       func() *MockUserRepository
		name       string
		email      string
		wantExists bool
		wantErr    bool
	}{
		{
			name:  "email exists",
			email: "existing@example.com",
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					ExistsByEmailFunc: func(_ context.Context, _ string) (bool, error) {
						return true, nil
					},
				}
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:  "email does not exist",
			email: "new@example.com",
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					ExistsByEmailFunc: func(_ context.Context, _ string) (bool, error) {
						return false, nil
					},
				}
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:  "database error",
			email: "error@example.com",
			mock: func() *MockUserRepository {
				return &MockUserRepository{
					ExistsByEmailFunc: func(_ context.Context, _ string) (bool, error) {
						return false, errors.New("database connection failed")
					},
				}
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			exists, err := repo.ExistsByEmail(context.Background(), tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExistsByEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if exists != tt.wantExists {
				t.Errorf("ExistsByEmail() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}
