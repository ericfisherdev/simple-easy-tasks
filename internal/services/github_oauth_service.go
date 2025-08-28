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

	"simple-easy-tasks/internal/domain"
)

// GitHubOAuthService handles GitHub OAuth2 authentication
type GitHubOAuthService struct {
	config      *oauth2.Config
	stateStore  GitHubOAuthStateRepository
	userService UserService
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
		config:      config,
		stateStore:  stateStore,
		userService: userService,
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
	AccessToken string              `json:"access_token"`
	User        *github.User        `json:"user"`
	Emails      []*github.UserEmail `json:"emails"`
	ProjectID   *string             `json:"project_id,omitempty"`
}

// InitiateAuth starts the GitHub OAuth flow
func (s *GitHubOAuthService) InitiateAuth(ctx context.Context, req *GitHubAuthRequest) (*GitHubAuthResponse, error) {
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
func (s *GitHubOAuthService) HandleCallback(ctx context.Context, req *GitHubCallbackRequest) (*GitHubCallbackResponse, error) {
	// Verify state
	storedState, err := s.stateStore.GetByState(ctx, req.State)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired state: %w", err)
	}

	// Check expiration
	if time.Now().After(storedState.ExpiresAt) {
		if deleteErr := s.stateStore.DeleteByState(ctx, req.State); deleteErr != nil {
			// TODO: Add proper logging for state cleanup error
			// State cleanup failure is not critical for OAuth flow
			_ = deleteErr // Acknowledge error without action
		}
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

	// Clean up state
	if err := s.stateStore.DeleteByState(ctx, req.State); err != nil {
		// TODO: Add proper logging for state cleanup error
		// State cleanup failure is not critical for OAuth flow
		_ = err // Acknowledge error without action
	}

	return &GitHubCallbackResponse{
		AccessToken: token.AccessToken,
		User:        user,
		Emails:      emails,
		ProjectID:   storedState.ProjectID,
	}, nil
}

// RefreshToken refreshes an OAuth token if needed
func (s *GitHubOAuthService) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := s.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
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
