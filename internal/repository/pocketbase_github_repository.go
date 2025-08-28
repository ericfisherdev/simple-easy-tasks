package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// PocketBaseGitHubIntegrationRepository implements GitHubIntegrationRepository using PocketBase
type PocketBaseGitHubIntegrationRepository struct {
	app core.App
}

// NewPocketBaseGitHubIntegrationRepository creates a new GitHub integration repository instance
func NewPocketBaseGitHubIntegrationRepository(app core.App) *PocketBaseGitHubIntegrationRepository {
	return &PocketBaseGitHubIntegrationRepository{app: app}
}

// escapeFilterValue escapes single quotes in filter values to prevent injection
func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

// Create creates a new GitHub integration in PocketBase
func (r *PocketBaseGitHubIntegrationRepository) Create(_ context.Context, integration *domain.GitHubIntegration) error {
	collection, err := r.app.FindCollectionByNameOrId("github_integrations")
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Id = integration.ID
	record.Set("project_id", integration.ProjectID)
	record.Set("user_id", integration.UserID)
	record.Set("repo_owner", integration.RepoOwner)
	record.Set("repo_name", integration.RepoName)
	record.Set("repo_id", integration.RepoID)

	if integration.InstallID != nil {
		record.Set("install_id", *integration.InstallID)
	}

	// Store encrypted token instead of plaintext
	// TODO: Replace with actual encryption service
	if integration.AccessToken != "" {
		record.Set("access_token_encrypted", fmt.Sprintf("ENCRYPTED:%s", integration.AccessToken))
		record.Set("token_type", "bearer")
		record.Set("key_version", "v1")
	}

	settingsJSON, err := json.Marshal(integration.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	record.Set("settings", string(settingsJSON))

	if !integration.CreatedAt.IsZero() {
		record.Set("created", integration.CreatedAt)
	}
	if !integration.UpdatedAt.IsZero() {
		record.Set("updated", integration.UpdatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}

	integration.ID = record.Id
	return nil
}

// GetByID retrieves a GitHub integration by ID from PocketBase
func (r *PocketBaseGitHubIntegrationRepository) GetByID(
	_ context.Context,
	id string,
) (*domain.GitHubIntegration, error) {
	record, err := r.app.FindRecordById("github_integrations", id)
	if err != nil {
		return nil, fmt.Errorf("%w: GitHub integration with ID %s: %v", ErrNotFound, id, err)
	}

	return r.recordToIntegration(record)
}

// GetByProjectID retrieves a GitHub integration by project ID from PocketBase
func (r *PocketBaseGitHubIntegrationRepository) GetByProjectID(
	_ context.Context,
	projectID string,
) (*domain.GitHubIntegration, error) {
	escapedProjectID := escapeFilterValue(projectID)
	filter := fmt.Sprintf("project_id = '%s'", escapedProjectID)
	record, err := r.app.FindFirstRecordByFilter("github_integrations", filter)
	if err != nil {
		return nil, fmt.Errorf("%w: GitHub integration for project %s: %v", ErrNotFound, projectID, err)
	}

	return r.recordToIntegration(record)
}

// GetByRepoFullName retrieves a GitHub integration by repository owner and name from PocketBase
func (r *PocketBaseGitHubIntegrationRepository) GetByRepoFullName(
	_ context.Context,
	owner, name string,
) (*domain.GitHubIntegration, error) {
	escapedOwner := escapeFilterValue(owner)
	escapedName := escapeFilterValue(name)
	filter := fmt.Sprintf("repo_owner = '%s' && repo_name = '%s'", escapedOwner, escapedName)
	record, err := r.app.FindFirstRecordByFilter("github_integrations", filter)
	if err != nil {
		return nil, fmt.Errorf("%w: GitHub integration for repository %s/%s: %v", ErrNotFound, owner, name, err)
	}

	return r.recordToIntegration(record)
}

// Update updates a GitHub integration in PocketBase
func (r *PocketBaseGitHubIntegrationRepository) Update(_ context.Context, integration *domain.GitHubIntegration) error {
	record, err := r.app.FindRecordById("github_integrations", integration.ID)
	if err != nil {
		return fmt.Errorf("GitHub integration not found: %w", err)
	}

	record.Set("project_id", integration.ProjectID)
	record.Set("user_id", integration.UserID)
	record.Set("repo_owner", integration.RepoOwner)
	record.Set("repo_name", integration.RepoName)
	record.Set("repo_id", integration.RepoID)

	if integration.InstallID != nil {
		record.Set("install_id", *integration.InstallID)
	}

	// Store encrypted token instead of plaintext
	// TODO: Replace with actual encryption service
	if integration.AccessToken != "" {
		record.Set("access_token_encrypted", fmt.Sprintf("ENCRYPTED:%s", integration.AccessToken))
		record.Set("token_type", "bearer")
		record.Set("key_version", "v1")
	}

	settingsJSON, err := json.Marshal(integration.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	record.Set("settings", string(settingsJSON))

	if !integration.UpdatedAt.IsZero() {
		record.Set("updated", integration.UpdatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update integration: %w", err)
	}

	return nil
}

// Delete deletes a GitHub integration from PocketBase
func (r *PocketBaseGitHubIntegrationRepository) Delete(_ context.Context, id string) error {
	record, err := r.app.FindRecordById("github_integrations", id)
	if err != nil {
		return fmt.Errorf("GitHub integration not found: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	return nil
}

// List retrieves GitHub integrations for a user from PocketBase
func (r *PocketBaseGitHubIntegrationRepository) List(
	_ context.Context,
	userID string,
) ([]*domain.GitHubIntegration, error) {
	filter := fmt.Sprintf("user_id = '%s'", userID)
	return listRecordsByFilter(r.app, "github_integrations", filter, "-created", 100, 0, r.recordToIntegration)
}

// recordToIntegration converts a PocketBase record to a GitHubIntegration
func (r *PocketBaseGitHubIntegrationRepository) recordToIntegration(
	record *core.Record,
) (*domain.GitHubIntegration, error) {
	var settings domain.GitHubSettings
	if settingsStr := record.GetString("settings"); settingsStr != "" {
		if err := json.Unmarshal([]byte(settingsStr), &settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	// Decrypt access token
	var accessToken string
	if encryptedToken := record.GetString("access_token_encrypted"); encryptedToken != "" {
		// TODO: Replace with actual decryption service
		if strings.HasPrefix(encryptedToken, "ENCRYPTED:") {
			accessToken = encryptedToken[10:] // Remove "ENCRYPTED:" prefix
		} else if strings.HasPrefix(encryptedToken, "NEEDS_ENCRYPTION:") {
			accessToken = encryptedToken[18:] // Remove "NEEDS_ENCRYPTION:" prefix
		}
	} else {
		// Fallback to deprecated field for backward compatibility during migration
		accessToken = record.GetString("access_token_deprecated")
	}

	integration := &domain.GitHubIntegration{
		ID:          record.Id,
		ProjectID:   record.GetString("project_id"),
		UserID:      record.GetString("user_id"),
		RepoOwner:   record.GetString("repo_owner"),
		RepoName:    record.GetString("repo_name"),
		RepoID:      int64(record.GetInt("repo_id")),
		AccessToken: accessToken,
		Settings:    settings,
		CreatedAt:   record.GetDateTime("created").Time(),
		UpdatedAt:   record.GetDateTime("updated").Time(),
	}

	if installID := record.GetInt("install_id"); installID != 0 {
		id := int64(installID)
		integration.InstallID = &id
	}

	return integration, nil
}

// Helper function to reduce code duplication for list operations
func listRecordsByFilter[T any](
	app core.App,
	collection, filter, sort string,
	limit, offset int,
	converter func(*core.Record) (*T, error),
) ([]*T, error) {
	records, err := app.FindRecordsByFilter(collection, filter, sort, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find records: %w", err)
	}

	results := make([]*T, len(records))
	for i, record := range records {
		result, err := converter(record)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

// Simplified implementations for other repositories following the same pattern

// PocketBaseGitHubOAuthStateRepository implements GitHubOAuthStateRepository using PocketBase
type PocketBaseGitHubOAuthStateRepository struct {
	app core.App
}

// NewPocketBaseGitHubOAuthStateRepository creates a new GitHub OAuth state repository instance
func NewPocketBaseGitHubOAuthStateRepository(app core.App) *PocketBaseGitHubOAuthStateRepository {
	return &PocketBaseGitHubOAuthStateRepository{app: app}
}

// Create creates a new GitHub OAuth state in PocketBase
func (r *PocketBaseGitHubOAuthStateRepository) Create(_ context.Context, state *domain.GitHubOAuthState) error {
	collection, err := r.app.FindCollectionByNameOrId("github_oauth_states")
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Id = state.ID
	record.Set("state", state.State)
	record.Set("user_id", state.UserID)

	if state.ProjectID != nil {
		record.Set("project_id", *state.ProjectID)
	}

	record.Set("expires_at", state.ExpiresAt)

	if !state.CreatedAt.IsZero() {
		record.Set("created", state.CreatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save OAuth state: %w", err)
	}

	state.ID = record.Id
	return nil
}

// GetByState retrieves a GitHub OAuth state by state value from PocketBase
func (r *PocketBaseGitHubOAuthStateRepository) GetByState(_ context.Context, state string) (*domain.GitHubOAuthState, error) {
	escapedState := escapeFilterValue(state)
	filter := fmt.Sprintf("state = '%s'", escapedState)
	record, err := r.app.FindFirstRecordByFilter("github_oauth_states", filter)
	if err != nil {
		return nil, fmt.Errorf("GitHub OAuth state not found: %w", err)
	}

	return r.recordToOAuthState(record)
}

// DeleteByState deletes a GitHub OAuth state by state value from PocketBase
func (r *PocketBaseGitHubOAuthStateRepository) DeleteByState(_ context.Context, state string) error {
	escapedState := escapeFilterValue(state)
	filter := fmt.Sprintf("state = '%s'", escapedState)
	record, err := r.app.FindFirstRecordByFilter("github_oauth_states", filter)
	if err != nil {
		return fmt.Errorf("GitHub OAuth state not found: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete OAuth state: %w", err)
	}

	return nil
}

// CleanupExpired removes expired GitHub OAuth states from PocketBase
func (r *PocketBaseGitHubOAuthStateRepository) CleanupExpired(_ context.Context) error {
	filter := fmt.Sprintf("expires_at <= '%s'", time.Now().Format(time.RFC3339))
	records, err := r.app.FindRecordsByFilter("github_oauth_states", filter, "", 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to find expired states: %w", err)
	}

	for _, record := range records {
		if err := r.app.Delete(record); err != nil {
			return fmt.Errorf("failed to delete expired state: %w", err)
		}
	}

	return nil
}

func (r *PocketBaseGitHubOAuthStateRepository) recordToOAuthState(record *core.Record) (*domain.GitHubOAuthState, error) {
	state := &domain.GitHubOAuthState{
		ID:        record.Id,
		State:     record.GetString("state"),
		UserID:    record.GetString("user_id"),
		ExpiresAt: record.GetDateTime("expires_at").Time(),
		CreatedAt: record.GetDateTime("created").Time(),
	}

	if projectID := record.GetString("project_id"); projectID != "" {
		state.ProjectID = &projectID
	}

	return state, nil
}

// Minimal implementations for other repositories (can be expanded similarly)

// PocketBaseGitHubIssueMappingRepository implements GitHubIssueMappingRepository using PocketBase
type PocketBaseGitHubIssueMappingRepository struct {
	app core.App
}

// NewPocketBaseGitHubIssueMappingRepository creates a new GitHub issue mapping repository instance
func NewPocketBaseGitHubIssueMappingRepository(app core.App) *PocketBaseGitHubIssueMappingRepository {
	return &PocketBaseGitHubIssueMappingRepository{app: app}
}

// Create creates a new GitHub issue mapping in PocketBase
func (r *PocketBaseGitHubIssueMappingRepository) Create(_ context.Context, mapping *domain.GitHubIssueMapping) error {
	collection, err := r.app.FindCollectionByNameOrId("github_issue_mappings")
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Id = mapping.ID
	record.Set("integration_id", mapping.IntegrationID)
	record.Set("task_id", mapping.TaskID)
	record.Set("issue_number", mapping.IssueNumber)
	record.Set("issue_id", mapping.IssueID)
	record.Set("sync_direction", mapping.SyncDirection)

	if mapping.LastSyncedAt != nil {
		record.Set("last_synced_at", *mapping.LastSyncedAt)
	}

	if !mapping.CreatedAt.IsZero() {
		record.Set("created", mapping.CreatedAt)
	}
	if !mapping.UpdatedAt.IsZero() {
		record.Set("updated", mapping.UpdatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save issue mapping: %w", err)
	}

	mapping.ID = record.Id
	return nil
}

// GetByTaskID retrieves a GitHub issue mapping by task ID from PocketBase
func (r *PocketBaseGitHubIssueMappingRepository) GetByTaskID(_ context.Context, taskID string) (*domain.GitHubIssueMapping, error) {
	escapedTaskID := escapeFilterValue(taskID)
	filter := fmt.Sprintf("task_id = '%s'", escapedTaskID)
	record, err := r.app.FindFirstRecordByFilter("github_issue_mappings", filter)
	if err != nil {
		return nil, fmt.Errorf("GitHub issue mapping not found: %w", err)
	}

	return r.recordToIssueMapping(record)
}

// GetByIssueNumber retrieves a GitHub issue mapping by issue number from PocketBase
func (r *PocketBaseGitHubIssueMappingRepository) GetByIssueNumber(_ context.Context, integrationID string, issueNumber int) (*domain.GitHubIssueMapping, error) {
	escapedIntegrationID := escapeFilterValue(integrationID)
	filter := fmt.Sprintf("integration_id = '%s' && issue_number = %d", escapedIntegrationID, issueNumber)
	record, err := r.app.FindFirstRecordByFilter("github_issue_mappings", filter)
	if err != nil {
		return nil, fmt.Errorf("GitHub issue mapping not found: %w", err)
	}

	return r.recordToIssueMapping(record)
}

// Update updates a GitHub issue mapping in PocketBase
func (r *PocketBaseGitHubIssueMappingRepository) Update(_ context.Context, mapping *domain.GitHubIssueMapping) error {
	record, err := r.app.FindRecordById("github_issue_mappings", mapping.ID)
	if err != nil {
		return fmt.Errorf("GitHub issue mapping not found: %w", err)
	}

	record.Set("integration_id", mapping.IntegrationID)
	record.Set("task_id", mapping.TaskID)
	record.Set("issue_number", mapping.IssueNumber)
	record.Set("issue_id", mapping.IssueID)
	record.Set("sync_direction", mapping.SyncDirection)

	if mapping.LastSyncedAt != nil {
		record.Set("last_synced_at", *mapping.LastSyncedAt)
	}

	if !mapping.UpdatedAt.IsZero() {
		record.Set("updated", mapping.UpdatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update issue mapping: %w", err)
	}

	return nil
}

// Delete deletes a GitHub issue mapping from PocketBase
func (r *PocketBaseGitHubIssueMappingRepository) Delete(_ context.Context, id string) error {
	record, err := r.app.FindRecordById("github_issue_mappings", id)
	if err != nil {
		return fmt.Errorf("GitHub issue mapping not found: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete issue mapping: %w", err)
	}

	return nil
}

// ListByIntegration retrieves GitHub issue mappings by integration ID from PocketBase
func (r *PocketBaseGitHubIssueMappingRepository) ListByIntegration(_ context.Context, integrationID string) ([]*domain.GitHubIssueMapping, error) {
	filter := fmt.Sprintf("integration_id = '%s'", integrationID)
	return listRecordsByFilter(r.app, "github_issue_mappings", filter, "-created", 100, 0, r.recordToIssueMapping)
}

func (r *PocketBaseGitHubIssueMappingRepository) recordToIssueMapping(record *core.Record) (*domain.GitHubIssueMapping, error) {
	mapping := &domain.GitHubIssueMapping{
		ID:            record.Id,
		IntegrationID: record.GetString("integration_id"),
		TaskID:        record.GetString("task_id"),
		IssueNumber:   record.GetInt("issue_number"),
		IssueID:       int64(record.GetInt("issue_id")),
		SyncDirection: record.GetString("sync_direction"),
		CreatedAt:     record.GetDateTime("created").Time(),
		UpdatedAt:     record.GetDateTime("updated").Time(),
	}

	if lastSynced := record.GetDateTime("last_synced_at"); !lastSynced.IsZero() {
		t := lastSynced.Time()
		mapping.LastSyncedAt = &t
	}

	return mapping, nil
}

// PocketBaseGitHubCommitLinkRepository implements GitHubCommitLinkRepository using PocketBase
type PocketBaseGitHubCommitLinkRepository struct {
	app core.App
}

// NewPocketBaseGitHubCommitLinkRepository creates a new GitHub commit link repository instance
func NewPocketBaseGitHubCommitLinkRepository(app core.App) *PocketBaseGitHubCommitLinkRepository {
	return &PocketBaseGitHubCommitLinkRepository{app: app}
}

// Create creates a new GitHub commit link in PocketBase
func (r *PocketBaseGitHubCommitLinkRepository) Create(_ context.Context, link *domain.GitHubCommitLink) error {
	collection, err := r.app.FindCollectionByNameOrId("github_commit_links")
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Id = link.ID
	record.Set("integration_id", link.IntegrationID)
	record.Set("task_id", link.TaskID)
	record.Set("commit_sha", link.CommitSHA)
	record.Set("commit_message", link.CommitMessage)
	record.Set("commit_url", link.CommitURL)
	record.Set("author_login", link.AuthorLogin)

	if !link.CreatedAt.IsZero() {
		record.Set("created", link.CreatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save commit link: %w", err)
	}

	link.ID = record.Id
	return nil
}

// GetByTaskID retrieves GitHub commit links by task ID from PocketBase
func (r *PocketBaseGitHubCommitLinkRepository) GetByTaskID(_ context.Context, taskID string) ([]*domain.GitHubCommitLink, error) {
	escapedTaskID := escapeFilterValue(taskID)
	filter := fmt.Sprintf("task_id = '%s'", escapedTaskID)
	return listRecordsByFilter(r.app, "github_commit_links", filter, "-created", 100, 0, r.recordToCommitLink)
}

// GetByCommitSHA retrieves a GitHub commit link by commit SHA from PocketBase
func (r *PocketBaseGitHubCommitLinkRepository) GetByCommitSHA(_ context.Context, integrationID, commitSHA string) (*domain.GitHubCommitLink, error) {
	escapedIntegrationID := escapeFilterValue(integrationID)
	escapedCommitSHA := escapeFilterValue(commitSHA)
	filter := fmt.Sprintf("integration_id = '%s' && commit_sha = '%s'", escapedIntegrationID, escapedCommitSHA)
	record, err := r.app.FindFirstRecordByFilter("github_commit_links", filter)
	if err != nil {
		return nil, fmt.Errorf("GitHub commit link not found: %w", err)
	}

	return r.recordToCommitLink(record)
}

// ListByIntegration retrieves GitHub commit links by integration ID from PocketBase
func (r *PocketBaseGitHubCommitLinkRepository) ListByIntegration(_ context.Context, integrationID string) ([]*domain.GitHubCommitLink, error) {
	filter := fmt.Sprintf("integration_id = '%s'", integrationID)
	return listRecordsByFilter(r.app, "github_commit_links", filter, "-created", 100, 0, r.recordToCommitLink)
}

// Delete deletes a GitHub commit link from PocketBase
func (r *PocketBaseGitHubCommitLinkRepository) Delete(_ context.Context, id string) error {
	record, err := r.app.FindRecordById("github_commit_links", id)
	if err != nil {
		return fmt.Errorf("GitHub commit link not found: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete commit link: %w", err)
	}

	return nil
}

func (r *PocketBaseGitHubCommitLinkRepository) recordToCommitLink(record *core.Record) (*domain.GitHubCommitLink, error) {
	link := &domain.GitHubCommitLink{
		ID:            record.Id,
		IntegrationID: record.GetString("integration_id"),
		TaskID:        record.GetString("task_id"),
		CommitSHA:     record.GetString("commit_sha"),
		CommitMessage: record.GetString("commit_message"),
		CommitURL:     record.GetString("commit_url"),
		AuthorLogin:   record.GetString("author_login"),
		CreatedAt:     record.GetDateTime("created").Time(),
	}

	return link, nil
}

// PocketBaseGitHubPRMappingRepository implements GitHubPRMappingRepository using PocketBase
type PocketBaseGitHubPRMappingRepository struct {
	app core.App
}

// NewPocketBaseGitHubPRMappingRepository creates a new GitHub PR mapping repository instance
func NewPocketBaseGitHubPRMappingRepository(app core.App) *PocketBaseGitHubPRMappingRepository {
	return &PocketBaseGitHubPRMappingRepository{app: app}
}

// Create creates a new GitHub PR mapping in PocketBase
func (r *PocketBaseGitHubPRMappingRepository) Create(_ context.Context, mapping *domain.GitHubPRMapping) error {
	collection, err := r.app.FindCollectionByNameOrId("github_pr_mappings")
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Id = mapping.ID
	record.Set("integration_id", mapping.IntegrationID)
	record.Set("task_id", mapping.TaskID)
	record.Set("pr_number", mapping.PRNumber)
	record.Set("pr_id", mapping.PRID)
	record.Set("pr_status", mapping.PRStatus)
	record.Set("branch_name", mapping.BranchName)

	if mapping.MergedAt != nil {
		record.Set("merged_at", *mapping.MergedAt)
	}

	if !mapping.CreatedAt.IsZero() {
		record.Set("created", mapping.CreatedAt)
	}
	if !mapping.UpdatedAt.IsZero() {
		record.Set("updated", mapping.UpdatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save PR mapping: %w", err)
	}

	mapping.ID = record.Id
	return nil
}

// GetByTaskID retrieves a GitHub PR mapping by task ID from PocketBase
func (r *PocketBaseGitHubPRMappingRepository) GetByTaskID(_ context.Context, taskID string) (*domain.GitHubPRMapping, error) {
	escapedTaskID := escapeFilterValue(taskID)
	filter := fmt.Sprintf("task_id = '%s'", escapedTaskID)
	record, err := r.app.FindFirstRecordByFilter("github_pr_mappings", filter)
	if err != nil {
		return nil, fmt.Errorf("GitHub PR mapping not found: %w", err)
	}

	return r.recordToPRMapping(record)
}

// GetByPRNumber retrieves a GitHub PR mapping by PR number from PocketBase
func (r *PocketBaseGitHubPRMappingRepository) GetByPRNumber(_ context.Context, integrationID string, prNumber int) (*domain.GitHubPRMapping, error) {
	escapedIntegrationID := escapeFilterValue(integrationID)
	filter := fmt.Sprintf("integration_id = '%s' && pr_number = %d", escapedIntegrationID, prNumber)
	record, err := r.app.FindFirstRecordByFilter("github_pr_mappings", filter)
	if err != nil {
		return nil, fmt.Errorf("GitHub PR mapping not found: %w", err)
	}

	return r.recordToPRMapping(record)
}

// Update updates a GitHub PR mapping in PocketBase
func (r *PocketBaseGitHubPRMappingRepository) Update(_ context.Context, mapping *domain.GitHubPRMapping) error {
	record, err := r.app.FindRecordById("github_pr_mappings", mapping.ID)
	if err != nil {
		return fmt.Errorf("GitHub PR mapping not found: %w", err)
	}

	record.Set("integration_id", mapping.IntegrationID)
	record.Set("task_id", mapping.TaskID)
	record.Set("pr_number", mapping.PRNumber)
	record.Set("pr_id", mapping.PRID)
	record.Set("pr_status", mapping.PRStatus)
	record.Set("branch_name", mapping.BranchName)

	if mapping.MergedAt != nil {
		record.Set("merged_at", *mapping.MergedAt)
	}

	if !mapping.UpdatedAt.IsZero() {
		record.Set("updated", mapping.UpdatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update PR mapping: %w", err)
	}

	return nil
}

// Delete deletes a GitHub PR mapping from PocketBase
func (r *PocketBaseGitHubPRMappingRepository) Delete(_ context.Context, id string) error {
	record, err := r.app.FindRecordById("github_pr_mappings", id)
	if err != nil {
		return fmt.Errorf("GitHub PR mapping not found: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete PR mapping: %w", err)
	}

	return nil
}

// ListByIntegration retrieves GitHub PR mappings by integration ID from PocketBase
func (r *PocketBaseGitHubPRMappingRepository) ListByIntegration(_ context.Context, integrationID string) ([]*domain.GitHubPRMapping, error) {
	filter := fmt.Sprintf("integration_id = '%s'", integrationID)
	return listRecordsByFilter(r.app, "github_pr_mappings", filter, "-created", 100, 0, r.recordToPRMapping)
}

func (r *PocketBaseGitHubPRMappingRepository) recordToPRMapping(record *core.Record) (*domain.GitHubPRMapping, error) {
	mapping := &domain.GitHubPRMapping{
		ID:            record.Id,
		IntegrationID: record.GetString("integration_id"),
		TaskID:        record.GetString("task_id"),
		PRNumber:      record.GetInt("pr_number"),
		PRID:          int64(record.GetInt("pr_id")),
		PRStatus:      record.GetString("pr_status"),
		BranchName:    record.GetString("branch_name"),
		CreatedAt:     record.GetDateTime("created").Time(),
		UpdatedAt:     record.GetDateTime("updated").Time(),
	}

	if mergedAt := record.GetDateTime("merged_at"); !mergedAt.IsZero() {
		t := mergedAt.Time()
		mapping.MergedAt = &t
	}

	return mapping, nil
}

// PocketBaseGitHubWebhookEventRepository implements GitHubWebhookEventRepository using PocketBase
type PocketBaseGitHubWebhookEventRepository struct {
	app core.App
}

// NewPocketBaseGitHubWebhookEventRepository creates a new GitHub webhook event repository instance
func NewPocketBaseGitHubWebhookEventRepository(app core.App) *PocketBaseGitHubWebhookEventRepository {
	return &PocketBaseGitHubWebhookEventRepository{app: app}
}

// Create creates a new GitHub webhook event in PocketBase
func (r *PocketBaseGitHubWebhookEventRepository) Create(_ context.Context, event *domain.GitHubWebhookEvent) error {
	collection, err := r.app.FindCollectionByNameOrId("github_webhook_events")
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Id = event.ID
	record.Set("integration_id", event.IntegrationID)
	record.Set("event_type", event.EventType)
	record.Set("action", event.Action)
	record.Set("payload", event.Payload)

	if event.ProcessedAt != nil {
		record.Set("processed_at", *event.ProcessedAt)
	}

	if event.ProcessingError != nil {
		record.Set("processing_error", *event.ProcessingError)
	}

	if !event.CreatedAt.IsZero() {
		record.Set("created", event.CreatedAt)
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save webhook event: %w", err)
	}

	event.ID = record.Id
	return nil
}

// GetByID retrieves a GitHub webhook event by ID from PocketBase
func (r *PocketBaseGitHubWebhookEventRepository) GetByID(_ context.Context, id string) (*domain.GitHubWebhookEvent, error) {
	record, err := r.app.FindRecordById("github_webhook_events", id)
	if err != nil {
		return nil, fmt.Errorf("GitHub webhook event not found: %w", err)
	}

	return r.recordToWebhookEvent(record)
}

// MarkProcessed marks a GitHub webhook event as processed in PocketBase
func (r *PocketBaseGitHubWebhookEventRepository) MarkProcessed(_ context.Context, id string, processedAt time.Time) error {
	record, err := r.app.FindRecordById("github_webhook_events", id)
	if err != nil {
		return fmt.Errorf("GitHub webhook event not found: %w", err)
	}

	record.Set("processed_at", processedAt)
	record.Set("processing_error", nil) // Clear any previous error

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to mark webhook event as processed: %w", err)
	}

	return nil
}

// MarkError marks a GitHub webhook event with an error message in PocketBase
func (r *PocketBaseGitHubWebhookEventRepository) MarkError(_ context.Context, id string, errorMsg string) error {
	record, err := r.app.FindRecordById("github_webhook_events", id)
	if err != nil {
		return fmt.Errorf("GitHub webhook event not found: %w", err)
	}

	record.Set("processing_error", errorMsg)
	record.Set("processed_at", time.Now())

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to mark webhook event error: %w", err)
	}

	return nil
}

// ListUnprocessed retrieves unprocessed GitHub webhook events from PocketBase
func (r *PocketBaseGitHubWebhookEventRepository) ListUnprocessed(_ context.Context, limit int) ([]*domain.GitHubWebhookEvent, error) {
	records, err := r.app.FindRecordsByFilter("github_webhook_events", "processed_at IS NULL", "-created", limit, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find unprocessed webhook events: %w", err)
	}

	events := make([]*domain.GitHubWebhookEvent, len(records))
	for i, record := range records {
		event, err := r.recordToWebhookEvent(record)
		if err != nil {
			return nil, err
		}
		events[i] = event
	}

	return events, nil
}

// CleanupOld removes old GitHub webhook events from PocketBase
func (r *PocketBaseGitHubWebhookEventRepository) CleanupOld(_ context.Context, olderThan time.Time) error {
	filter := fmt.Sprintf("created <= '%s'", olderThan.Format(time.RFC3339))
	records, err := r.app.FindRecordsByFilter("github_webhook_events", filter, "", 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to find old webhook events: %w", err)
	}

	for _, record := range records {
		if err := r.app.Delete(record); err != nil {
			return fmt.Errorf("failed to delete old webhook event: %w", err)
		}
	}

	return nil
}

func (r *PocketBaseGitHubWebhookEventRepository) recordToWebhookEvent(record *core.Record) (*domain.GitHubWebhookEvent, error) {
	// Get payload as raw JSON bytes
	var payloadBytes []byte
	if payload := record.Get("payload"); payload != nil {
		switch p := payload.(type) {
		case []byte:
			payloadBytes = p
		case string:
			// Backward compatibility for string payloads
			payloadBytes = []byte(p)
		case map[string]interface{}:
			// Handle JSON object
			var err error
			payloadBytes, err = json.Marshal(p)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
		default:
			// Try to marshal any other type
			var err error
			payloadBytes, err = json.Marshal(p)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload of type %T: %w", p, err)
			}
		}
	}

	event := &domain.GitHubWebhookEvent{
		ID:            record.Id,
		IntegrationID: record.GetString("integration_id"),
		EventType:     record.GetString("event_type"),
		Action:        record.GetString("action"),
		Payload:       payloadBytes,
		CreatedAt:     record.GetDateTime("created").Time(),
	}

	if processedAt := record.GetDateTime("processed_at"); !processedAt.IsZero() {
		t := processedAt.Time()
		event.ProcessedAt = &t
	}

	if errorMsg := record.GetString("processing_error"); errorMsg != "" {
		event.ProcessingError = &errorMsg
	}

	return event, nil
}
