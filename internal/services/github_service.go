package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"

	"simple-easy-tasks/internal/domain"
)

const (
	statusReview   = "review"
	statusComplete = "complete"
)

// GitHubService provides GitHub API integration with rate limiting
type GitHubService struct {
	integrationRepo  GitHubIntegrationRepository
	issueMappingRepo GitHubIssueMappingRepository
	commitLinkRepo   GitHubCommitLinkRepository
	prMappingRepo    GitHubPRMappingRepository
	rateLimiter      *GitHubRateLimiter
	webhookSecret    string
}

// GitHubIntegrationRepository defines methods for GitHub integration data persistence
type GitHubIntegrationRepository interface {
	Create(ctx context.Context, integration *domain.GitHubIntegration) error
	GetByID(ctx context.Context, id string) (*domain.GitHubIntegration, error)
	GetByProjectID(ctx context.Context, projectID string) (*domain.GitHubIntegration, error)
	GetByRepoFullName(ctx context.Context, owner, name string) (*domain.GitHubIntegration, error)
	Update(ctx context.Context, integration *domain.GitHubIntegration) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string) ([]*domain.GitHubIntegration, error)
}

type GitHubIssueMappingRepository interface {
	Create(ctx context.Context, mapping *domain.GitHubIssueMapping) error
	GetByTaskID(ctx context.Context, taskID string) (*domain.GitHubIssueMapping, error)
	GetByIssueNumber(ctx context.Context, integrationID string, issueNumber int) (*domain.GitHubIssueMapping, error)
	Update(ctx context.Context, mapping *domain.GitHubIssueMapping) error
	Delete(ctx context.Context, id string) error
	ListByIntegration(ctx context.Context, integrationID string) ([]*domain.GitHubIssueMapping, error)
}

// GitHubCommitLinkRepository defines methods for GitHub commit link data persistence
type GitHubCommitLinkRepository interface {
	Create(ctx context.Context, link *domain.GitHubCommitLink) error
	GetByTaskID(ctx context.Context, taskID string) ([]*domain.GitHubCommitLink, error)
	GetByCommitSHA(ctx context.Context, integrationID, commitSHA string) (*domain.GitHubCommitLink, error)
	ListByIntegration(ctx context.Context, integrationID string) ([]*domain.GitHubCommitLink, error)
	Delete(ctx context.Context, id string) error
}

type GitHubPRMappingRepository interface {
	Create(ctx context.Context, mapping *domain.GitHubPRMapping) error
	GetByTaskID(ctx context.Context, taskID string) (*domain.GitHubPRMapping, error)
	GetByPRNumber(ctx context.Context, integrationID string, prNumber int) (*domain.GitHubPRMapping, error)
	Update(ctx context.Context, mapping *domain.GitHubPRMapping) error
	Delete(ctx context.Context, id string) error
	ListByIntegration(ctx context.Context, integrationID string) ([]*domain.GitHubPRMapping, error)
}

// GitHubRateLimiter handles GitHub API rate limiting
type GitHubRateLimiter struct {
	remaining int
	resetTime time.Time
	lastCheck time.Time
}

// NewGitHubService creates a new GitHub service
func NewGitHubService(
	integrationRepo GitHubIntegrationRepository,
	issueMappingRepo GitHubIssueMappingRepository,
	commitLinkRepo GitHubCommitLinkRepository,
	prMappingRepo GitHubPRMappingRepository,
	webhookSecret string,
) *GitHubService {
	return &GitHubService{
		integrationRepo:  integrationRepo,
		issueMappingRepo: issueMappingRepo,
		commitLinkRepo:   commitLinkRepo,
		prMappingRepo:    prMappingRepo,
		rateLimiter:      &GitHubRateLimiter{remaining: 5000},
		webhookSecret:    webhookSecret,
	}
}

// CreateIntegration creates a new GitHub integration
func (s *GitHubService) CreateIntegration(ctx context.Context, accessToken string, projectID, userID string, repoOwner, repoName string) (*domain.GitHubIntegration, error) {
	// Create authenticated client
	client := s.createClient(accessToken)

	// Get repository information
	repo, _, err := client.Repositories.Get(ctx, repoOwner, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Create integration
	integration := &domain.GitHubIntegration{
		ID:          generateID(),
		ProjectID:   projectID,
		UserID:      userID,
		RepoOwner:   repoOwner,
		RepoName:    repoName,
		RepoID:      repo.GetID(),
		AccessToken: accessToken,
		Settings:    domain.NewDefaultGitHubSettings(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := integration.Validate(); err != nil {
		return nil, err
	}

	if err := s.integrationRepo.Create(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	return integration, nil
}

// SyncIssueToTask synchronizes a GitHub issue with a task
func (s *GitHubService) SyncIssueToTask(ctx context.Context, integrationID string, issueNumber int, taskID string) error {
	integration, err := s.integrationRepo.GetByID(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	client := s.createClient(integration.AccessToken)

	// Check rate limit
	if rateLimitErr := s.checkRateLimit(ctx, client); rateLimitErr != nil {
		return rateLimitErr
	}

	// Get issue from GitHub
	issue, _, err := client.Issues.Get(ctx, integration.RepoOwner, integration.RepoName, issueNumber)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Create or update mapping
	mapping, err := s.issueMappingRepo.GetByIssueNumber(ctx, integrationID, issueNumber)
	if err != nil {
		// Create new mapping
		mapping = &domain.GitHubIssueMapping{
			ID:            generateID(),
			IntegrationID: integrationID,
			TaskID:        taskID,
			IssueNumber:   issueNumber,
			IssueID:       issue.GetID(),
			SyncDirection: "both",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := mapping.Validate(); err != nil {
			return err
		}

		return s.issueMappingRepo.Create(ctx, mapping)
	}

	// Update existing mapping
	mapping.TaskID = taskID
	mapping.LastSyncedAt = &[]time.Time{time.Now()}[0]
	mapping.UpdatedAt = time.Now()

	return s.issueMappingRepo.Update(ctx, mapping)
}

// CreateIssueFromTask creates a GitHub issue from a task
func (s *GitHubService) CreateIssueFromTask(ctx context.Context, integrationID, taskID string, task *domain.Task) (*github.Issue, error) {
	integration, err := s.integrationRepo.GetByID(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration: %w", err)
	}

	client := s.createClient(integration.AccessToken)

	// Check rate limit
	if rateLimitErr := s.checkRateLimit(ctx, client); rateLimitErr != nil {
		return nil, rateLimitErr
	}

	// Create issue request
	issueRequest := &github.IssueRequest{
		Title: &task.Title,
		Body:  &task.Description,
	}

	// Add labels based on task priority and status
	labels := s.generateLabelsFromTask(task)
	if len(labels) > 0 {
		issueRequest.Labels = &labels
	}

	// Create issue
	issue, _, err := client.Issues.Create(ctx, integration.RepoOwner, integration.RepoName, issueRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	// Create mapping
	mapping := &domain.GitHubIssueMapping{
		ID:            generateID(),
		IntegrationID: integrationID,
		TaskID:        taskID,
		IssueNumber:   issue.GetNumber(),
		IssueID:       issue.GetID(),
		SyncDirection: "both",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.issueMappingRepo.Create(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to create issue mapping: %w", err)
	}

	return issue, nil
}

// LinkCommitToTask links a commit to a task based on commit message references
func (s *GitHubService) LinkCommitToTask(ctx context.Context, integrationID, taskID, commitSHA, commitMessage, commitURL, authorLogin string) error {
	// Check if link already exists
	existing, err := s.commitLinkRepo.GetByCommitSHA(ctx, integrationID, commitSHA)
	if err == nil && existing != nil {
		return nil // Link already exists
	}

	// Create commit link
	link := &domain.GitHubCommitLink{
		ID:            generateID(),
		IntegrationID: integrationID,
		TaskID:        taskID,
		CommitSHA:     commitSHA,
		CommitMessage: commitMessage,
		CommitURL:     commitURL,
		AuthorLogin:   authorLogin,
		CreatedAt:     time.Now(),
	}

	if err := link.Validate(); err != nil {
		return err
	}

	return s.commitLinkRepo.Create(ctx, link)
}

// ParseTaskReferencesFromCommit extracts task references from commit messages
func (s *GitHubService) ParseTaskReferencesFromCommit(commitMessage string) []string {
	var taskRefs []string

	// Common patterns for task references
	patterns := []string{
		"#TASK-", "#task-", "TASK-", "task-",
		"fixes #", "closes #", "resolves #",
	}

	message := strings.ToLower(commitMessage)
	for _, pattern := range patterns {
		if strings.Contains(message, pattern) {
			// Extract task IDs - simplified implementation
			// In production, use regex for more accurate extraction
			parts := strings.Split(message, pattern)
			for i := 1; i < len(parts); i++ {
				words := strings.Fields(parts[i])
				if len(words) > 0 {
					taskRefs = append(taskRefs, words[0])
				}
			}
		}
	}

	return taskRefs
}

// CreateBranchForTask creates a branch for a task
func (s *GitHubService) CreateBranchForTask(ctx context.Context, integrationID, _ string, task *domain.Task) error {
	integration, err := s.integrationRepo.GetByID(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	if !integration.Settings.AutoCreateBranches {
		return fmt.Errorf("auto branch creation is disabled")
	}

	client := s.createClient(integration.AccessToken)

	// Check rate limit
	if rateLimitErr := s.checkRateLimit(ctx, client); rateLimitErr != nil {
		return rateLimitErr
	}

	// Get default branch
	repo, _, err := client.Repositories.Get(ctx, integration.RepoOwner, integration.RepoName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	defaultBranch := repo.GetDefaultBranch()

	// Get reference for default branch
	ref, _, err := client.Git.GetRef(ctx, integration.RepoOwner, integration.RepoName, "refs/heads/"+defaultBranch)
	if err != nil {
		return fmt.Errorf("failed to get reference: %w", err)
	}

	// Generate branch name from task
	branchName := s.generateBranchName(task)

	// Create new branch
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: ref.Object.SHA},
	}

	_, _, err = client.Git.CreateRef(ctx, integration.RepoOwner, integration.RepoName, newRef)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

// Helper methods

func (s *GitHubService) createClient(accessToken string) *github.Client {
	client := github.NewClient(http.DefaultClient)
	return client.WithAuthToken(accessToken)
}

func (s *GitHubService) checkRateLimit(ctx context.Context, client *github.Client) error {
	if time.Since(s.rateLimiter.lastCheck) < 5*time.Minute {
		if s.rateLimiter.remaining < 100 && time.Now().Before(s.rateLimiter.resetTime) {
			return fmt.Errorf("rate limit exceeded, resets at %v", s.rateLimiter.resetTime)
		}
		return nil
	}

	// Check current rate limit
	rates, _, err := client.RateLimit.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to check rate limits: %w", err)
	}

	s.rateLimiter.remaining = rates.Core.Remaining
	s.rateLimiter.resetTime = rates.Core.Reset.Time
	s.rateLimiter.lastCheck = time.Now()

	if s.rateLimiter.remaining < 100 {
		return fmt.Errorf("rate limit low: %d remaining, resets at %v",
			s.rateLimiter.remaining, s.rateLimiter.resetTime)
	}

	return nil
}

func (s *GitHubService) generateLabelsFromTask(task *domain.Task) []string {
	var labels []string

	// Priority labels
	switch task.Priority {
	case "low":
		labels = append(labels, "priority: low")
	case "medium":
		labels = append(labels, "priority: medium")
	case "high":
		labels = append(labels, "priority: high")
	case "critical":
		labels = append(labels, "priority: critical")
	}

	// Status labels
	switch task.Status {
	case "todo":
		labels = append(labels, "status: todo")
	case "developing":
		labels = append(labels, "status: in progress")
	case statusReview:
		labels = append(labels, "status: review")
	case statusComplete:
		labels = append(labels, "status: done")
	}

	return labels
}

func (s *GitHubService) generateBranchName(task *domain.Task) string {
	// Sanitize title for branch name
	title := strings.ToLower(task.Title)
	title = strings.ReplaceAll(title, " ", "-")
	title = strings.ReplaceAll(title, "_", "-")

	// Remove special characters
	var cleaned strings.Builder
	for _, char := range title {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			cleaned.WriteRune(char)
		}
	}

	branchName := cleaned.String()
	if len(branchName) > 50 {
		branchName = branchName[:50]
	}

	// Add task ID if available
	if task.ID != "" {
		branchName = fmt.Sprintf("feature/%s-%s", task.ID[:8], branchName)
	} else {
		branchName = fmt.Sprintf("feature/%s", branchName)
	}

	return branchName
}

// Integration management methods

// GetIntegrationByProjectID retrieves a GitHub integration by project ID
func (s *GitHubService) GetIntegrationByProjectID(ctx context.Context, projectID string) (*domain.GitHubIntegration, error) {
	return s.integrationRepo.GetByProjectID(ctx, projectID)
}

// GetIntegrationByID retrieves a GitHub integration by ID
func (s *GitHubService) GetIntegrationByID(ctx context.Context, id string) (*domain.GitHubIntegration, error) {
	return s.integrationRepo.GetByID(ctx, id)
}

// UpdateIntegration updates a GitHub integration
func (s *GitHubService) UpdateIntegration(ctx context.Context, integration *domain.GitHubIntegration) error {
	return s.integrationRepo.Update(ctx, integration)
}

// DeleteIntegration deletes a GitHub integration
func (s *GitHubService) DeleteIntegration(ctx context.Context, id string) error {
	return s.integrationRepo.Delete(ctx, id)
}

// GetCommitsByTaskID retrieves commits linked to a task
func (s *GitHubService) GetCommitsByTaskID(ctx context.Context, taskID string) ([]*domain.GitHubCommitLink, error) {
	return s.commitLinkRepo.GetByTaskID(ctx, taskID)
}

// GetPRsByTaskID retrieves pull requests linked to a task
func (s *GitHubService) GetPRsByTaskID(ctx context.Context, taskID string) (*domain.GitHubPRMapping, error) {
	return s.prMappingRepo.GetByTaskID(ctx, taskID)
}
