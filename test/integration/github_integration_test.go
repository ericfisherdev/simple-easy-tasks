//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
	testutil "simple-easy-tasks/internal/testutil/integration"
)

func TestGitHubIntegrationRepository_Integration(t *testing.T) {
	// Setup test container with DI
	tc := NewTestContainer(t)
	defer tc.Cleanup()

	t.Run("GitHubIntegration_CRUD_Operations", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Create GitHub integration repository
		integrationRepo := repository.NewPocketBaseGitHubIntegrationRepository(tc.GetPocketBaseApp(t))

		// Create test user and project for the integration
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("github-integration@test.example.com"),
			testutil.WithUserUsername("githubuser"),
			testutil.WithUserName("GitHub User"),
		)
		err := tc.GetUserRepository(t).Create(context.Background(), user)
		require.NoError(t, err)

		project := suite.Factory.CreateProject(user,
			testutil.WithProjectTitle("GitHub Integration Test Project"),
			testutil.WithProjectDescription("Test project for GitHub integration"),
		)
		err = tc.GetProjectRepository(t).Create(context.Background(), project)
		require.NoError(t, err)

		// Test creating GitHub integration
		integration := &domain.GitHubIntegration{
			ProjectID:   project.ID,
			UserID:      user.ID,
			RepoOwner:   "testowner",
			RepoName:    "testrepo",
			RepoID:      12345,
			AccessToken: "mock_access_token",
			Settings:    domain.NewDefaultGitHubSettings(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = integrationRepo.Create(context.Background(), integration)
		require.NoError(t, err)
		assert.NotEmpty(t, integration.ID)

		// Test retrieving integration by project ID
		retrieved, err := integrationRepo.GetByProjectID(context.Background(), project.ID)
		require.NoError(t, err)
		assert.Equal(t, integration.RepoOwner, retrieved.RepoOwner)
		assert.Equal(t, integration.RepoName, retrieved.RepoName)
		assert.Equal(t, integration.Settings.AutoLinkCommits, retrieved.Settings.AutoLinkCommits)

		// Test retrieving by repo full name
		repoIntegration, err := integrationRepo.GetByRepoFullName(context.Background(), "testowner", "testrepo")
		require.NoError(t, err)
		assert.Equal(t, integration.ID, repoIntegration.ID)

		// Test updating integration
		integration.Settings.AutoLinkCommits = false
		integration.Settings.AutoCreateBranches = true
		integration.UpdatedAt = time.Now()

		err = integrationRepo.Update(context.Background(), integration)
		require.NoError(t, err)

		updated, err := integrationRepo.GetByID(context.Background(), integration.ID)
		require.NoError(t, err)
		assert.False(t, updated.Settings.AutoLinkCommits)
		assert.True(t, updated.Settings.AutoCreateBranches)

		// Test deleting integration
		err = integrationRepo.Delete(context.Background(), integration.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = integrationRepo.GetByID(context.Background(), integration.ID)
		assert.Error(t, err)
	})

	t.Run("GitHubOAuthState_CRUD_Operations", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Create OAuth state repository
		oauthStateRepo := repository.NewPocketBaseGitHubOAuthStateRepository(tc.GetPocketBaseApp(t))

		// Create test user
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("oauth-state@test.example.com"),
			testutil.WithUserUsername("oauthuser"),
			testutil.WithUserName("OAuth User"),
		)
		err := tc.GetUserRepository(t).Create(context.Background(), user)
		require.NoError(t, err)

		// Test creating OAuth state
		state := &domain.GitHubOAuthState{
			State:     "test-oauth-state",
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(1 * time.Hour),
			CreatedAt: time.Now(),
		}

		err = oauthStateRepo.Create(context.Background(), state)
		require.NoError(t, err)
		assert.NotEmpty(t, state.ID)

		// Test retrieving state
		retrieved, err := oauthStateRepo.GetByState(context.Background(), "test-oauth-state")
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.UserID)
		assert.True(t, retrieved.ExpiresAt.After(time.Now()))

		// Test deleting state
		err = oauthStateRepo.DeleteByState(context.Background(), "test-oauth-state")
		require.NoError(t, err)

		// Verify deletion
		_, err = oauthStateRepo.GetByState(context.Background(), "test-oauth-state")
		assert.Error(t, err)
	})

	t.Run("GitHubIssueMapping_CRUD_Operations", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Create repositories
		integrationRepo := repository.NewPocketBaseGitHubIntegrationRepository(tc.GetPocketBaseApp(t))
		issueMappingRepo := repository.NewPocketBaseGitHubIssueMappingRepository(tc.GetPocketBaseApp(t))

		// Create test data
		user := suite.Factory.CreateUser()
		err := tc.GetUserRepository(t).Create(context.Background(), user)
		require.NoError(t, err)

		project := suite.Factory.CreateProject(user)
		err = tc.GetProjectRepository(t).Create(context.Background(), project)
		require.NoError(t, err)

		task := suite.Factory.CreateTask(project, user)
		err = tc.GetTaskRepository(t).Create(context.Background(), task)
		require.NoError(t, err)

		// Create integration
		integration := &domain.GitHubIntegration{
			ProjectID:   project.ID,
			UserID:      user.ID,
			RepoOwner:   "testowner",
			RepoName:    "testrepo",
			RepoID:      12345,
			AccessToken: "mock_token",
			Settings:    domain.NewDefaultGitHubSettings(),
		}
		err = integrationRepo.Create(context.Background(), integration)
		require.NoError(t, err)

		// Test creating issue mapping
		mapping := &domain.GitHubIssueMapping{
			IntegrationID: integration.ID,
			TaskID:        task.ID,
			IssueNumber:   42,
			IssueID:       98765,
			SyncDirection: "both",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = issueMappingRepo.Create(context.Background(), mapping)
		require.NoError(t, err)
		assert.NotEmpty(t, mapping.ID)

		// Test retrieving by task ID
		taskMapping, err := issueMappingRepo.GetByTaskID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Equal(t, mapping.IssueNumber, taskMapping.IssueNumber)

		// Test retrieving by issue number
		issueMapping, err := issueMappingRepo.GetByIssueNumber(context.Background(), integration.ID, 42)
		require.NoError(t, err)
		assert.Equal(t, task.ID, issueMapping.TaskID)

		// Test updating mapping
		now := time.Now()
		mapping.LastSyncedAt = &now
		err = issueMappingRepo.Update(context.Background(), mapping)
		require.NoError(t, err)

		// Test listing by integration
		mappings, err := issueMappingRepo.ListByIntegration(context.Background(), integration.ID)
		require.NoError(t, err)
		assert.Len(t, mappings, 1)

		// Test deleting mapping
		err = issueMappingRepo.Delete(context.Background(), mapping.ID)
		require.NoError(t, err)

		_, err = issueMappingRepo.GetByTaskID(context.Background(), task.ID)
		assert.Error(t, err)
	})

	t.Run("GitHubCommitLink_CRUD_Operations", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Create repositories
		integrationRepo := repository.NewPocketBaseGitHubIntegrationRepository(tc.GetPocketBaseApp(t))
		commitLinkRepo := repository.NewPocketBaseGitHubCommitLinkRepository(tc.GetPocketBaseApp(t))

		// Create test data
		user := suite.Factory.CreateUser()
		err := tc.GetUserRepository(t).Create(context.Background(), user)
		require.NoError(t, err)

		project := suite.Factory.CreateProject(user)
		err = tc.GetProjectRepository(t).Create(context.Background(), project)
		require.NoError(t, err)

		task := suite.Factory.CreateTask(project, user)
		err = tc.GetTaskRepository(t).Create(context.Background(), task)
		require.NoError(t, err)

		// Create integration
		integration := &domain.GitHubIntegration{
			ProjectID:   project.ID,
			UserID:      user.ID,
			RepoOwner:   "testowner",
			RepoName:    "testrepo",
			RepoID:      12345,
			AccessToken: "mock_token",
			Settings:    domain.NewDefaultGitHubSettings(),
		}
		err = integrationRepo.Create(context.Background(), integration)
		require.NoError(t, err)

		// Test creating commit link
		commitLink := &domain.GitHubCommitLink{
			IntegrationID: integration.ID,
			TaskID:        task.ID,
			CommitSHA:     "abc123def456",
			CommitMessage: "fix: resolve issue #TASK-123",
			CommitURL:     "https://github.com/testowner/testrepo/commit/abc123def456",
			AuthorLogin:   "testuser",
			CreatedAt:     time.Now(),
		}

		err = commitLinkRepo.Create(context.Background(), commitLink)
		require.NoError(t, err)
		assert.NotEmpty(t, commitLink.ID)

		// Test retrieving by task ID
		links, err := commitLinkRepo.GetByTaskID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Len(t, links, 1)
		assert.Equal(t, "abc123def456", links[0].CommitSHA)

		// Test retrieving by commit SHA
		link, err := commitLinkRepo.GetByCommitSHA(context.Background(), integration.ID, "abc123def456")
		require.NoError(t, err)
		assert.Equal(t, task.ID, link.TaskID)

		// Test listing by integration
		allLinks, err := commitLinkRepo.ListByIntegration(context.Background(), integration.ID)
		require.NoError(t, err)
		assert.Len(t, allLinks, 1)

		// Test deleting commit link
		err = commitLinkRepo.Delete(context.Background(), commitLink.ID)
		require.NoError(t, err)

		links, err = commitLinkRepo.GetByTaskID(context.Background(), task.ID)
		require.NoError(t, err)
		assert.Len(t, links, 0)
	})
}