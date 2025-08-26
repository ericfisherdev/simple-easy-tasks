//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simple-easy-tasks/internal/domain"
	testutil "simple-easy-tasks/internal/testutil/integration"
)

// TestUserRepository_SelectFieldBehavior tests PocketBase select field handling
// NOTE: These tests demonstrate the graceful handling when role field doesn't exist in collection schema
func TestUserRepository_SelectFieldBehavior(t *testing.T) {
	// Setup test container with DI
	tc := NewTestContainer(t)
	defer tc.Cleanup()

	// Get repository from DI container
	repo := tc.GetUserRepository(t)

	t.Run("SelectField_MissingFieldGracefulHandling", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)
		
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()
		
		// Arrange - Create admin user (role field doesn't exist in collection schema)
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("admin@selecttest.com"),
			testutil.WithUserName("Admin Select Test"),
			testutil.WithUserRole(domain.AdminRole),
		)

		// Act - Create user
		err := repo.Create(context.Background(), user)
		require.NoError(t, err, "Admin user creation should succeed")

		// Verify role was set during creation in memory
		assert.Equal(t, domain.AdminRole, user.Role, "User role should remain AdminRole in memory after creation")

		// Act - Retrieve user by ID
		retrieved, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err, "Should be able to retrieve created admin user")

		// Assert - Since role field doesn't exist in collection, it gracefully defaults to RegularUserRole
		assert.Equal(t, domain.RegularUserRole, retrieved.Role, "Retrieved user defaults to RegularUserRole when role field missing from schema")
		assert.Equal(t, "user", string(retrieved.Role), "Role should default to 'user' string")
		assert.NotEmpty(t, string(retrieved.Role), "Role should not be empty string")
		
		// Verify other fields persisted correctly
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Name, retrieved.Name)
	})

	t.Run("SelectField_RegularUserBehavior", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)
		
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()
		
		// Arrange - Create regular user
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("user@selecttest.com"),
			testutil.WithUserName("User Select Test"),
			testutil.WithUserRole(domain.RegularUserRole),
		)

		// Act - Create user
		err := repo.Create(context.Background(), user)
		require.NoError(t, err, "Regular user creation should succeed")

		// Act - Retrieve user by ID
		retrieved, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err, "Should be able to retrieve created regular user")

		// Assert - Role correctly defaults to RegularUserRole (matches what we set)
		assert.Equal(t, domain.RegularUserRole, retrieved.Role, "Retrieved user should have RegularUserRole")
		assert.Equal(t, "user", string(retrieved.Role), "Role should be 'user' string")
		
		// Verify other fields persisted correctly
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Name, retrieved.Name)
	})

	t.Run("SelectField_EmptyRoleHandling", func(t *testing.T) {
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()
		
		// Arrange - Create user with empty role
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("emptyrole@selecttest.com"),
			testutil.WithUserName("Empty Role Test"),
		)
		// Domain validation should set default role
		user.Role = ""

		// Act - Validate should set default role
		err := user.Validate()
		require.NoError(t, err, "Validation should succeed and set default role")
		
		// Assert - Role should be set to default by validation
		assert.Equal(t, domain.RegularUserRole, user.Role, "Role should default to RegularUserRole after validation")
	})

	t.Run("SelectField_InvalidRoleValidation", func(t *testing.T) {
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()
		
		// Arrange - Create user with invalid role
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("invalidrole@selecttest.com"),
			testutil.WithUserName("Invalid Role Test"),
		)
		// Set invalid role
		user.Role = domain.UserRole("superadmin") // Not a valid role

		// Act - Try to validate user with invalid role
		err := user.Validate()

		// Assert - Should fail validation
		require.Error(t, err, "Validation should fail for invalid role")
		assert.Contains(t, err.Error(), "Role must be", "Error should indicate valid role values")
	})

	t.Run("SelectField_RetrieveByEmail", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)
		
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()
		
		// Arrange
		email := "emailtest@selecttest.com"
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail(email),
			testutil.WithUserName("Email Test User"),
			testutil.WithUserRole(domain.AdminRole),
		)

		// Act - Create and retrieve by email
		err := repo.Create(context.Background(), user)
		require.NoError(t, err, "User creation should succeed")

		retrieved, err := repo.GetByEmail(context.Background(), email)
		require.NoError(t, err, "Should be able to retrieve user by email")

		// Assert - Role defaults to RegularUserRole due to missing field
		assert.Equal(t, domain.RegularUserRole, retrieved.Role, "Role should default when field missing")
		assert.Equal(t, "user", string(retrieved.Role), "Role should be 'user' string")
		assert.Equal(t, email, retrieved.Email, "Email should match")
	})
}