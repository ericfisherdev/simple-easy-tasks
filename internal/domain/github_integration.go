package domain

import (
	"time"
)

// GitHubIntegration represents a GitHub integration for a project
type GitHubIntegration struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	UserID      string         `json:"user_id"`
	RepoOwner   string         `json:"repo_owner"`
	RepoName    string         `json:"repo_name"`
	RepoID      int64          `json:"repo_id"`
	InstallID   *int64         `json:"install_id,omitempty"`
	AccessToken string         `json:"access_token"`
	Settings    GitHubSettings `json:"settings"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// GitHubSettings configures integration behavior
type GitHubSettings struct {
	AutoLinkCommits    bool     `json:"auto_link_commits"`
	AutoLinkPRs        bool     `json:"auto_link_prs"`
	AutoCreateBranches bool     `json:"auto_create_branches"`
	SyncLabels         bool     `json:"sync_labels"`
	WebhookEvents      []string `json:"webhook_events"`
}

// GitHubWebhookEvent represents a webhook event from GitHub
type GitHubWebhookEvent struct {
	ID              string     `json:"id"`
	IntegrationID   string     `json:"integration_id"`
	EventType       string     `json:"event_type"`
	Action          string     `json:"action"`
	Payload         string     `json:"payload"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ProcessingError *string    `json:"processing_error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// GitHubIssueMapping links tasks to GitHub issues
type GitHubIssueMapping struct {
	ID            string     `json:"id"`
	IntegrationID string     `json:"integration_id"`
	TaskID        string     `json:"task_id"`
	IssueNumber   int        `json:"issue_number"`
	IssueID       int64      `json:"issue_id"`
	SyncDirection string     `json:"sync_direction"` // "both", "to_github", "from_github"
	LastSyncedAt  *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// GitHubCommitLink links tasks to commits
type GitHubCommitLink struct {
	ID            string    `json:"id"`
	IntegrationID string    `json:"integration_id"`
	TaskID        string    `json:"task_id"`
	CommitSHA     string    `json:"commit_sha"`
	CommitMessage string    `json:"commit_message"`
	CommitURL     string    `json:"commit_url"`
	AuthorLogin   string    `json:"author_login"`
	CreatedAt     time.Time `json:"created_at"`
}

// GitHubPRMapping links tasks to pull requests
type GitHubPRMapping struct {
	ID            string     `json:"id"`
	IntegrationID string     `json:"integration_id"`
	TaskID        string     `json:"task_id"`
	PRNumber      int        `json:"pr_number"`
	PRID          int64      `json:"pr_id"`
	PRStatus      string     `json:"pr_status"` // "open", "closed", "merged"
	BranchName    string     `json:"branch_name"`
	MergedAt      *time.Time `json:"merged_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// GitHubOAuthState stores OAuth state for security
type GitHubOAuthState struct {
	ID        string    `json:"id"`
	State     string    `json:"state"`
	UserID    string    `json:"user_id"`
	ProjectID *string   `json:"project_id,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Validation methods

// Validate validates the GitHub integration fields
func (g *GitHubIntegration) Validate() error {
	if g.ProjectID == "" {
		return NewValidationError("project_id", "Project ID is required", nil)
	}
	if g.UserID == "" {
		return NewValidationError("user_id", "User ID is required", nil)
	}
	if g.RepoOwner == "" {
		return NewValidationError("repo_owner", "Repository owner is required", nil)
	}
	if g.RepoName == "" {
		return NewValidationError("repo_name", "Repository name is required", nil)
	}
	if g.AccessToken == "" {
		return NewValidationError("access_token", "Access token is required", nil)
	}
	return nil
}

// Validate validates the GitHub issue mapping fields
func (g *GitHubIssueMapping) Validate() error {
	if g.IntegrationID == "" {
		return NewValidationError("integration_id", "Integration ID is required", nil)
	}
	if g.TaskID == "" {
		return NewValidationError("task_id", "Task ID is required", nil)
	}
	if g.IssueNumber <= 0 {
		return NewValidationError("issue_number", "Issue number must be positive", nil)
	}
	if g.SyncDirection != "both" && g.SyncDirection != "to_github" && g.SyncDirection != "from_github" {
		return NewValidationError("sync_direction", "Sync direction must be 'both', 'to_github', or 'from_github'", nil)
	}
	return nil
}

// Validate validates the GitHub commit link fields
func (g *GitHubCommitLink) Validate() error {
	if g.IntegrationID == "" {
		return NewValidationError("integration_id", "Integration ID is required", nil)
	}
	if g.TaskID == "" {
		return NewValidationError("task_id", "Task ID is required", nil)
	}
	if g.CommitSHA == "" {
		return NewValidationError("commit_sha", "Commit SHA is required", nil)
	}
	return nil
}

// Helper methods

// GetRepositoryFullName returns the full repository name in owner/name format
func (g *GitHubIntegration) GetRepositoryFullName() string {
	return g.RepoOwner + "/" + g.RepoName
}

// IsWebhookEventEnabled checks if a webhook event type is enabled for this integration
func (g *GitHubIntegration) IsWebhookEventEnabled(eventType string) bool {
	for _, event := range g.Settings.WebhookEvents {
		if event == eventType {
			return true
		}
	}
	return false
}

// NewDefaultGitHubSettings returns the default GitHub integration settings
func NewDefaultGitHubSettings() GitHubSettings {
	return GitHubSettings{
		AutoLinkCommits:    true,
		AutoLinkPRs:        true,
		AutoCreateBranches: false,
		SyncLabels:         true,
		WebhookEvents: []string{
			"push",
			"pull_request",
			"issues",
			"issue_comment",
			"pull_request_review",
		},
	}
}
