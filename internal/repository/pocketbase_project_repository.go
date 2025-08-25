package repository

//nolint:gofumpt
import (
	"context"
	"fmt"
	"time"

	"simple-easy-tasks/internal/domain"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// pocketbaseProjectRepository implements ProjectRepository using PocketBase.
type pocketbaseProjectRepository struct {
	app core.App
}

// NewPocketBaseProjectRepository creates a new PocketBase project repository.
func NewPocketBaseProjectRepository(app core.App) ProjectRepository {
	return &pocketbaseProjectRepository{
		app: app,
	}
}

// Create creates a new project in PocketBase.
func (r *pocketbaseProjectRepository) Create(_ context.Context, project *domain.Project) error {
	if err := project.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	collection, err := r.app.FindCollectionByNameOrId("projects")
	if err != nil {
		return fmt.Errorf("failed to find projects collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("title", project.Title)
	record.Set("description", project.Description)
	record.Set("slug", project.Slug)
	record.Set("owner", project.OwnerID)
	record.Set("color", project.Color)
	record.Set("icon", project.Icon)
	record.Set("status", string(project.Status))
	record.Set("settings", project.Settings)
	record.Set("members", project.MemberIDs)

	if !project.CreatedAt.IsZero() {
		record.Set("created", project.CreatedAt)
	}
	if !project.UpdatedAt.IsZero() {
		record.Set("updated", project.UpdatedAt)
	}

	if project.ID != "" {
		record.Id = project.ID
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save project record: %w", err)
	}

	project.ID = record.Id
	if createdTime := record.GetDateTime("created"); !createdTime.IsZero() {
		project.CreatedAt = createdTime.Time()
	}
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		project.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// GetByID retrieves a project by ID from PocketBase.
func (r *pocketbaseProjectRepository) GetByID(_ context.Context, id string) (*domain.Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	record, err := r.app.FindRecordById("projects", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find project by ID %s: %w", id, err)
	}

	return r.recordToProject(record)
}

// GetBySlug retrieves a project by slug from PocketBase.
func (r *pocketbaseProjectRepository) GetBySlug(_ context.Context, slug string) (*domain.Project, error) {
	if slug == "" {
		return nil, fmt.Errorf("project slug cannot be empty")
	}

	record, err := r.app.FindFirstRecordByFilter("projects", "slug = {:slug}", dbx.Params{"slug": slug})
	if err != nil {
		return nil, fmt.Errorf("failed to find project by slug %s: %w", slug, err)
	}

	return r.recordToProject(record)
}

// Update updates an existing project in PocketBase.
func (r *pocketbaseProjectRepository) Update(_ context.Context, project *domain.Project) error {
	if err := project.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if project.ID == "" {
		return fmt.Errorf("project ID cannot be empty for update")
	}

	record, err := r.app.FindRecordById("projects", project.ID)
	if err != nil {
		return fmt.Errorf("failed to find project for update: %w", err)
	}

	record.Set("title", project.Title)
	record.Set("description", project.Description)
	record.Set("slug", project.Slug)
	record.Set("owner", project.OwnerID)
	record.Set("color", project.Color)
	record.Set("icon", project.Icon)
	record.Set("status", string(project.Status))
	record.Set("settings", project.Settings)
	record.Set("members", project.MemberIDs)
	record.Set("updated", time.Now())

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update project record: %w", err)
	}

	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		project.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// Delete deletes a project by ID from PocketBase.
func (r *pocketbaseProjectRepository) Delete(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	record, err := r.app.FindRecordById("projects", id)
	if err != nil {
		return fmt.Errorf("failed to find project for deletion: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete project record: %w", err)
	}

	return nil
}

// ListByOwner retrieves projects owned by a specific user.
func (r *pocketbaseProjectRepository) ListByOwner(
	_ context.Context, ownerID string, offset, limit int,
) ([]*domain.Project, error) {
	if ownerID == "" {
		return nil, fmt.Errorf("owner ID cannot be empty")
	}

	records, err := r.app.FindRecordsByFilter(
		"projects", "owner = {:ownerID}", "-created", limit, offset, dbx.Params{"ownerID": ownerID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find projects by owner: %w", err)
	}

	return r.recordsToProjects(records)
}

// ListByMember retrieves projects where a user is a member.
func (r *pocketbaseProjectRepository) ListByMember(
	_ context.Context, memberID string, offset, limit int,
) ([]*domain.Project, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member ID cannot be empty")
	}

	records, err := r.app.FindRecordsByFilter(
		"projects", "members ~ {:memberID}", "-created", limit, offset, dbx.Params{"memberID": memberID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find projects by member: %w", err)
	}

	return r.recordsToProjects(records)
}

// List retrieves projects with pagination from PocketBase.
func (r *pocketbaseProjectRepository) List(_ context.Context, offset, limit int) ([]*domain.Project, error) {
	records, err := r.app.FindRecordsByFilter("projects", "", "-created", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return r.recordsToProjects(records)
}

// Count returns the total number of projects in PocketBase.
func (r *pocketbaseProjectRepository) Count(_ context.Context) (int, error) {
	total, err := r.app.CountRecords("projects")
	if err != nil {
		return 0, fmt.Errorf("failed to count projects: %w", err)
	}

	return int(total), nil
}

// ExistsBySlug checks if a project exists with the given slug.
func (r *pocketbaseProjectRepository) ExistsBySlug(_ context.Context, slug string) (bool, error) {
	if slug == "" {
		return false, fmt.Errorf("project slug cannot be empty")
	}

	_, err := r.app.FindFirstRecordByFilter("projects", "slug = {:slug}", dbx.Params{"slug": slug})
	if err != nil {
		if err.Error() == sqlNoRowsError {
			return false, nil
		}
		return false, fmt.Errorf("failed to check project existence by slug: %w", err)
	}

	return true, nil
}

// GetMemberProjects retrieves all projects where user has access.
func (r *pocketbaseProjectRepository) GetMemberProjects(
	_ context.Context, userID string, offset, limit int,
) ([]*domain.Project, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Find projects where user is owner OR member OR has guest access
	filter := "owner = {:userID} || members ~ {:userID} || " +
		"(settings.is_private = false && settings.allow_guest_view = true)"
	records, err := r.app.FindRecordsByFilter(
		"projects", filter, "-created", limit, offset, dbx.Params{"userID": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find member projects: %w", err)
	}

	return r.recordsToProjects(records)
}

// recordToProject converts a PocketBase record to a domain.Project.
func (r *pocketbaseProjectRepository) recordToProject(record *core.Record) (*domain.Project, error) {
	var settings domain.ProjectSettings
	if err := record.UnmarshalJSONField("settings", &settings); err != nil {
		settings = domain.ProjectSettings{}
	}

	var memberIDs []string
	if err := record.UnmarshalJSONField("members", &memberIDs); err != nil {
		memberIDs = []string{}
	}

	project := &domain.Project{
		ID:          record.Id,
		Title:       record.GetString("title"),
		Description: record.GetString("description"),
		Slug:        record.GetString("slug"),
		OwnerID:     record.GetString("owner"),
		Color:       record.GetString("color"),
		Icon:        record.GetString("icon"),
		Status:      domain.ProjectStatus(record.GetString("status")),
		Settings:    settings,
		MemberIDs:   memberIDs,
		CreatedAt:   record.GetDateTime("created").Time(),
		UpdatedAt:   record.GetDateTime("updated").Time(),
	}

	return project, nil
}

// recordsToProjects converts PocketBase records to domain.Project slice.
func (r *pocketbaseProjectRepository) recordsToProjects(records []*core.Record) ([]*domain.Project, error) {
	projects := make([]*domain.Project, len(records))
	for i, record := range records {
		project, err := r.recordToProject(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to project: %w", err)
		}
		projects[i] = project
	}
	return projects, nil
}
