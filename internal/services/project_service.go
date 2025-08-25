package services

import (
	"context"
	"strings"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// ProjectService defines the interface for project-related business logic.
type ProjectService interface {
	// CreateProject creates a new project
	CreateProject(ctx context.Context, req domain.CreateProjectRequest, ownerID string) (*domain.Project, error)

	// GetProject gets a project by ID
	GetProject(ctx context.Context, projectID string, userID string) (*domain.Project, error)

	// GetProjectBySlug gets a project by slug
	GetProjectBySlug(ctx context.Context, slug string, userID string) (*domain.Project, error)

	// UpdateProject updates a project
	UpdateProject(ctx context.Context, projectID string, req domain.UpdateProjectRequest, userID string) (*domain.Project, error)

	// DeleteProject deletes a project
	DeleteProject(ctx context.Context, projectID string, userID string) error

	// ListUserProjects lists projects for a user
	ListUserProjects(ctx context.Context, userID string, offset, limit int) ([]*domain.Project, error)

	// AddMember adds a user to a project
	AddMember(ctx context.Context, projectID string, userID string, requesterID string) error

	// RemoveMember removes a user from a project
	RemoveMember(ctx context.Context, projectID string, userID string, requesterID string) error

	// ListMembers lists project members
	ListMembers(ctx context.Context, projectID string, userID string) ([]*domain.User, error)
}

// projectService implements ProjectService interface.
type projectService struct {
	projectRepo repository.ProjectRepository
	userRepo    repository.UserRepository
}

// NewProjectService creates a new project service.
func NewProjectService(projectRepo repository.ProjectRepository, userRepo repository.UserRepository) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		userRepo:    userRepo,
	}
}

// CreateProject creates a new project.
func (s *projectService) CreateProject(ctx context.Context, req domain.CreateProjectRequest, ownerID string) (*domain.Project, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if owner exists
	_, err := s.userRepo.GetByID(ctx, ownerID)
	if err != nil {
		return nil, domain.NewNotFoundError("OWNER_NOT_FOUND", "Project owner not found")
	}

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = s.generateSlug(req.Title)
	}

	// Check if slug exists
	exists, err := s.projectRepo.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, domain.NewInternalError("SLUG_CHECK_FAILED", "Failed to check slug availability", err)
	}
	if exists {
		return nil, domain.NewConflictError("SLUG_EXISTS", "Project slug already exists")
	}

	// Create project
	project := &domain.Project{
		Title:       req.Title,
		Description: req.Description,
		Slug:        slug,
		OwnerID:     ownerID,
		Status:      domain.ProjectStatusActive,
		Color:       req.Color,
		Icon:        req.Icon,
		Settings: domain.ProjectSettings{
			IsPublic:    req.IsPublic,
			AllowGuests: req.AllowGuests,
			DefaultRole: domain.ProjectRoleMember,
		},
		Members: []string{ownerID}, // Owner is automatically a member
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, domain.NewInternalError("PROJECT_CREATE_FAILED", "Failed to create project", err)
	}

	return project, nil
}

// GetProject gets a project by ID.
func (s *projectService) GetProject(ctx context.Context, projectID string, userID string) (*domain.Project, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if user has access
	if !project.HasAccess(userID) && !project.Settings.IsPublic {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	return project, nil
}

// GetProjectBySlug gets a project by slug.
func (s *projectService) GetProjectBySlug(ctx context.Context, slug string, userID string) (*domain.Project, error) {
	if slug == "" {
		return nil, domain.NewValidationError("INVALID_SLUG", "Project slug cannot be empty")
	}

	project, err := s.projectRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Check if user has access
	if !project.HasAccess(userID) && !project.Settings.IsPublic {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to this project")
	}

	return project, nil
}

// UpdateProject updates a project.
func (s *projectService) UpdateProject(ctx context.Context, projectID string, req domain.UpdateProjectRequest, userID string) (*domain.Project, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty")
	}

	// Get existing project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if user is owner or admin
	if !project.IsOwner(userID) {
		// Check if user is admin member
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || (!project.HasAccess(userID) || user.Role != domain.AdminRole) {
			return nil, domain.NewAuthorizationError("ACCESS_DENIED", "Only project owners can update projects")
		}
	}

	// Apply updates
	if req.Title != nil {
		project.Title = *req.Title
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Color != nil {
		project.Color = *req.Color
	}
	if req.Icon != nil {
		project.Icon = *req.Icon
	}
	if req.Status != nil {
		project.Status = *req.Status
	}
	if req.Settings != nil {
		project.Settings = *req.Settings
	}

	// Validate updated project
	if err := project.Validate(); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, domain.NewInternalError("PROJECT_UPDATE_FAILED", "Failed to update project", err)
	}

	return project, nil
}

// DeleteProject deletes a project.
func (s *projectService) DeleteProject(ctx context.Context, projectID string, userID string) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty")
	}

	// Get existing project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Check if user is owner
	if !project.IsOwner(userID) {
		return domain.NewAuthorizationError("ACCESS_DENIED", "Only project owners can delete projects")
	}

	// Delete from repository
	if err := s.projectRepo.Delete(ctx, projectID); err != nil {
		return domain.NewInternalError("PROJECT_DELETE_FAILED", "Failed to delete project", err)
	}

	return nil
}

// ListUserProjects lists projects for a user.
func (s *projectService) ListUserProjects(ctx context.Context, userID string, offset, limit int) ([]*domain.Project, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty")
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Get projects where user has access
	projects, err := s.projectRepo.GetMemberProjects(ctx, userID, offset, limit)
	if err != nil {
		return nil, domain.NewInternalError("PROJECT_LIST_FAILED", "Failed to list projects", err)
	}

	return projects, nil
}

// AddMember adds a user to a project.
func (s *projectService) AddMember(ctx context.Context, projectID string, userID string, requesterID string) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty")
	}
	if userID == "" {
		return domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty")
	}

	// Get project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Check if requester has permission to add members
	if !project.IsOwner(requesterID) && !project.IsAdmin(requesterID) {
		return domain.NewAuthorizationError("ACCESS_DENIED", "You don't have permission to add members")
	}

	// Check if user exists
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return domain.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	// Add member
	if err := project.AddMember(userID); err != nil {
		return err
	}

	// Update project
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return domain.NewInternalError("MEMBER_ADD_FAILED", "Failed to add member", err)
	}

	return nil
}

// RemoveMember removes a user from a project.
func (s *projectService) RemoveMember(ctx context.Context, projectID string, userID string, requesterID string) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty")
	}
	if userID == "" {
		return domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty")
	}

	// Get project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Check if requester has permission (owners and admins can remove others, users can remove themselves)
	canRemove := project.IsOwner(requesterID) || project.IsAdmin(requesterID) || requesterID == userID
	if !canRemove {
		return domain.NewAuthorizationError("ACCESS_DENIED", "You don't have permission to remove this member")
	}

	// Cannot remove owner
	if project.IsOwner(userID) {
		return domain.NewValidationError("CANNOT_REMOVE_OWNER", "Project owner cannot be removed")
	}

	// Remove member
	if err := project.RemoveMember(userID); err != nil {
		return err
	}

	// Update project
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return domain.NewInternalError("MEMBER_REMOVE_FAILED", "Failed to remove member", err)
	}

	return nil
}

// ListMembers lists project members.
func (s *projectService) ListMembers(ctx context.Context, projectID string, userID string) ([]*domain.User, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty")
	}

	// Get project
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to view members
	if !project.HasAccess(userID) && !project.Settings.IsPublic {
		return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You don't have access to view project members")
	}

	// Get member details
	members := make([]*domain.User, 0, len(project.Members))
	for _, memberID := range project.Members {
		user, err := s.userRepo.GetByID(ctx, memberID)
		if err != nil {
			// Skip members that no longer exist
			continue
		}
		// Remove sensitive information
		user.PasswordHash = ""
		members = append(members, user)
	}

	return members, nil
}

// generateSlug creates a URL-friendly slug from the title.
func (s *projectService) generateSlug(title string) string {
	// Simple slug generation - replace spaces with hyphens and lowercase
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters (basic implementation)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
