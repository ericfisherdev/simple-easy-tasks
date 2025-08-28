//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ericfisherdev/simple-easy-tasks/internal/config"
	"github.com/ericfisherdev/simple-easy-tasks/internal/container"
	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	"github.com/ericfisherdev/simple-easy-tasks/internal/repository"
	testutil "github.com/ericfisherdev/simple-easy-tasks/internal/testutil/integration"
)

// TestFIRSTPrinciples demonstrates that tests can now run individually
// This test can be run with: go test -v ./test/integration/ -run TestFIRSTPrinciples
func TestFIRSTPrinciples(t *testing.T) {
	// Setup - use testutil directly for this demonstration
	suite := testutil.SetupDatabaseTest(t)
	defer suite.Cleanup()

	// Create DI container - demonstrating dependency injection
	c := container.NewContainer()
	cfg := config.NewConfig()

	// Register services
	err := container.RegisterServices(c, cfg, suite.DB.App())
	require.NoError(t, err)

	// Resolve repository from DI container
	repoInterface, err := c.Resolve(container.UserRepositoryService)
	require.NoError(t, err)

	repo, ok := repoInterface.(repository.UserRepository)
	require.True(t, ok)

	// Test demonstrates FIRST principles:
	t.Run("Fast", func(t *testing.T) {
		// This test runs quickly in isolation
		user := &domain.User{
			Email: "fast@test.example.com",
			Name:  "Fast Test User",
		}
		require.NoError(t, user.SetPassword("password123"))

		err := repo.Create(context.Background(), user)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
	})

	t.Run("Isolated", func(t *testing.T) {
		// Clear database to ensure isolation
		suite.DB.Reset()

		// This test runs in isolation from other tests
		user := &domain.User{
			Email: "isolated@test.example.com",
			Name:  "Isolated Test User",
		}
		require.NoError(t, user.SetPassword("password123"))

		err := repo.Create(context.Background(), user)
		require.NoError(t, err)

		// Verify isolation by checking count
		count, err := repo.Count(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, count, "Should only have 1 user in isolated test")
	})

	t.Run("Repeatable", func(t *testing.T) {
		// Clear database to ensure repeatability
		suite.DB.Reset()

		// This test produces the same result every time
		for i := 0; i < 3; i++ {
			user := &domain.User{
				Email: "repeatable@test.example.com",
				Name:  "Repeatable Test User",
			}
			require.NoError(t, user.SetPassword("password123"))

			// Should fail after first iteration due to unique constraint
			err := repo.Create(context.Background(), user)
			if i == 0 {
				require.NoError(t, err, "First creation should succeed")
			} else {
				require.Error(t, err, "Subsequent creations should fail due to unique constraint")
			}
		}
	})

	t.Run("SelfVerifying", func(t *testing.T) {
		// Clear database
		suite.DB.Reset()

		// Test verifies its own success/failure automatically
		user := &domain.User{
			Email: "selfverifying@test.example.com",
			Name:  "Self Verifying Test User",
		}
		require.NoError(t, user.SetPassword("password123"))

		// Create user
		err := repo.Create(context.Background(), user)
		require.NoError(t, err)

		// Verify creation by retrieval
		retrieved, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Name, retrieved.Name)

		// Test automatically verifies success - no manual verification needed
	})

	t.Run("Timely", func(t *testing.T) {
		// This test is written alongside the production code it tests
		// It validates the repository interface and DI container integration
		// which were developed together, demonstrating timely testing

		// Verify that all repository services are properly registered
		services := []string{
			container.UserRepositoryService,
			container.ProjectRepositoryService,
			container.TaskRepositoryService,
			container.CommentRepositoryService,
		}

		for _, serviceName := range services {
			_, err := c.Resolve(serviceName)
			assert.NoError(t, err, "Service %s should be registered", serviceName)
		}
	})
}
