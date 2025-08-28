package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
)

// GitHubOAuthService handles GitHub OAuth2 authentication
type GitHubOAuthService struct {
	config          *oauth2.Config
	stateStore      GitHubOAuthStateRepository
	authSessionRepo repository.GitHubAuthSessionRepository
	userService     UserService
}

// GitHubOAuthStateRepository defines the interface for storing OAuth states
type GitHubOAuthStateRepository interface {
	Create(ctx context.Context, state *domain.GitHubOAuthState) error
	GetByState(ctx context.Context, state string) (*domain.GitHubOAuthState, error)
	DeleteByState(ctx context.Context, state string) error
	CleanupExpired(ctx context.Context) error
}

// NewGitHubOAuthService creates a new GitHub OAuth service
func NewGitHubOAuthService(
	clientID, clientSecret, redirectURL string,
	stateStore GitHubOAuthStateRepository,
	authSessionRepo repository.GitHubAuthSessionRepository,
	userService UserService,
) *GitHubOAuthService {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email", "repo", "read:org"},
		Endpoint:     endpoints.GitHub,
	}

	return &GitHubOAuthService{
		config:          config,
		stateStore:      stateStore,
		authSessionRepo: authSessionRepo,
		userService:     userService,
	}
}

// GitHubAuthRequest contains data needed to initiate OAuth flow
type GitHubAuthRequest struct {
	UserID    string  `json:"user_id"`
	ProjectID *string `json:"project_id,omitempty"`
}

// GitHubAuthResponse contains the OAuth authorization URL
type GitHubAuthResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// GitHubCallbackRequest contains data from OAuth callback
type GitHubCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// GitHubCallbackResponse contains the result of OAuth callback processing
type GitHubCallbackResponse struct {
	User      *github.User        `json:"user"`
	Emails    []*github.UserEmail `json:"emails"`
	ProjectID *string             `json:"project_id,omitempty"`
}

// InitiateAuth starts the GitHub OAuth flow
func (s *GitHubOAuthService) InitiateAuth(ctx context.Context, req *GitHubAuthRequest) (*GitHubAuthResponse, error) {
	// Validate that UserID is non-empty
	if req.UserID == "" {
		return nil, fmt.Errorf("invalid argument: user ID cannot be empty")
	}

	// Generate secure random state
	state, err := s.generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state with expiration
	oauthState := &domain.GitHubOAuthState{
		ID:        generateID(),
		State:     state,
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10 minute expiration
		CreatedAt: time.Now(),
	}

	if err := s.stateStore.Create(ctx, oauthState); err != nil {
		return nil, fmt.Errorf("failed to store OAuth state: %w", err)
	}

	// Generate authorization URL
	authURL := s.config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return &GitHubAuthResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// HandleCallback processes the OAuth callback
func (s *GitHubOAuthService) HandleCallback(ctx context.Context, userID string, req *GitHubCallbackRequest) (*GitHubCallbackResponse, error) {
	// Verify state and delete immediately for single-use
	storedState, err := s.stateStore.GetByState(ctx, req.State)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired state: %w", err)
	}

	// Delete state immediately after retrieval for single-use behavior
	if deleteErr := s.stateStore.DeleteByState(ctx, req.State); deleteErr != nil {
		// Log but don't fail OAuth flow for cleanup errors
		_ = deleteErr
	}

	// Check expiration
	if time.Now().After(storedState.ExpiresAt) {
		return nil, fmt.Errorf("OAuth state has expired")
	}

	// Exchange code for token
	token, err := s.config.Exchange(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Create GitHub client
	client := github.NewClient(s.config.Client(ctx, token))

	// Get user information
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get user information: %w", err)
	}

	// Get user emails
	emails, _, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user emails: %w", err)
	}

	// Store the access token server-side with 1 hour expiration
	authSession := &domain.GitHubAuthSession{
		ID:          fmt.Sprintf("%s_%d", userID, time.Now().Unix()),
		UserID:      userID,
		AccessToken: token.AccessToken,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour), // 1 hour session
	}

	if err := s.authSessionRepo.Create(ctx, authSession); err != nil {
		return nil, fmt.Errorf("failed to store GitHub auth session: %w", err)
	}

	return &GitHubCallbackResponse{
		User:      user,
		Emails:    emails,
		ProjectID: storedState.ProjectID,
	}, nil
}

// GetUserToken retrieves the stored GitHub token for a user
func (s *GitHubOAuthService) GetUserToken(ctx context.Context, userID string) (string, error) {
	session, err := s.authSessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("no active GitHub session for user: %w", err)
	}

	if session.IsExpired() {
		// Clean up expired session
		_ = s.authSessionRepo.DeleteByUserID(ctx, userID)
		return "", fmt.Errorf("GitHub session expired for user %s", userID)
	}

	return session.AccessToken, nil
}

// RefreshToken is not supported for GitHub OAuth Apps as they don't issue refresh tokens
// GitHub OAuth Apps tokens do not expire, so refresh is not needed
// For GitHub App installations, use installation tokens instead
func (s *GitHubOAuthService) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("token refresh is not supported for GitHub OAuth Apps; re-authenticate via InitiateAuth/HandleCallback or use GitHub App installation tokens for expiring tokens")
}

// ValidateToken validates a GitHub access token
func (s *GitHubOAuthService) ValidateToken(ctx context.Context, accessToken string) (*github.User, error) {
	client := github.NewClient(http.DefaultClient)
	client = client.WithAuthToken(accessToken)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %w", err)
	}

	return user, nil
}

// GetUserRepositories gets repositories accessible to the user
func (s *GitHubOAuthService) GetUserRepositories(ctx context.Context, accessToken string, page, perPage int) ([]*github.Repository, error) {
	client := github.NewClient(http.DefaultClient)
	client = client.WithAuthToken(accessToken)

	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner,collaborator,organization_member",
		Sort:        "updated",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}

	repos, _, err := client.Repositories.ListByAuthenticatedUser(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	return repos, nil
}

// CleanupExpiredStates removes expired OAuth states
func (s *GitHubOAuthService) CleanupExpiredStates(ctx context.Context) error {
	return s.stateStore.CleanupExpired(ctx)
}

// generateState generates a cryptographically secure random state string
func (s *GitHubOAuthService) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateID generates a unique identifier
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based ID if crypto rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}
