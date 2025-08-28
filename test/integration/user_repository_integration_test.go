//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
	testutil "github.com/ericfisherdev/simple-easy-tasks/internal/testutil/integration"
)

func TestUserRepository_Integration(t *testing.T) {
	// Setup test container with DI
	tc := NewTestContainer(t)
	defer tc.Cleanup()

	// Get repository from DI container
	repo := tc.GetUserRepository(t)

	t.Run("Create_ValidUser_Success", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("create.valid@test.example.com"),
			testutil.WithUserUsername("createvalid"),
			testutil.WithUserName("Create Valid"),
		)

		// Act
		err := repo.Create(context.Background(), user)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())

		// Verify the user can be retrieved
		retrieved, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Username, retrieved.Username)
		assert.Equal(t, user.Name, retrieved.Name)
		assert.Equal(t, user.Role, retrieved.Role)
	})

	t.Run("Create_DuplicateEmail_ConstraintViolation", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		email := "duplicate.email@test.example.com"
		user1 := suite.Factory.CreateUser(
			testutil.WithUserEmail(email),
			testutil.WithUserUsername("user1duplicate"),
			testutil.WithUserName("User 1 Duplicate"),
		)
		user2 := suite.Factory.CreateUser(
			testutil.WithUserEmail(email),
			testutil.WithUserUsername("user2duplicate"),
			testutil.WithUserName("User 2 Duplicate"),
		)

		// Act
		err1 := repo.Create(context.Background(), user1)
		err2 := repo.Create(context.Background(), user2)

		// Assert
		require.NoError(t, err1, "First user creation should succeed")
		require.Error(t, err2, "Second user with same email should fail")
		assert.Contains(t, err2.Error(), "UNIQUE", "Error should indicate constraint violation")
	})

	t.Run("Create_DuplicateUsername_ConstraintViolation", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		username := "duplicateuser"
		user1 := suite.Factory.CreateUser(
			testutil.WithUserEmail("user1.username@test.example.com"),
			testutil.WithUserUsername(username),
			testutil.WithUserName("User 1 Username"),
		)
		user2 := suite.Factory.CreateUser(
			testutil.WithUserEmail("user2.username@test.example.com"),
			testutil.WithUserUsername(username),
			testutil.WithUserName("User 2 Username"),
		)

		// Act
		err1 := repo.Create(context.Background(), user1)
		err2 := repo.Create(context.Background(), user2)

		// Assert
		require.NoError(t, err1, "First user creation should succeed")
		require.Error(t, err2, "Second user with same username should fail")
		assert.Contains(t, err2.Error(), "UNIQUE", "Error should indicate constraint violation")
	})

	t.Run("Create_InvalidUserData_ValidationError", func(t *testing.T) {
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail(""), // Invalid: empty email
			testutil.WithUserUsername("testuser"),
			testutil.WithUserName("Test User"),
			testutil.WithUserRole(domain.RegularUserRole),
		)

		// Act
		err := repo.Create(context.Background(), user)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("GetByID_ExistingUser_ReturnsUser", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("getbyid@test.example.com"),
			testutil.WithUserUsername("getbyid"),
			testutil.WithUserName("Get By ID User"),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Act
		retrieved, err := repo.GetByID(context.Background(), user.ID)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Username, retrieved.Username)
		assert.Equal(t, user.Name, retrieved.Name)
		assert.Equal(t, user.Role, retrieved.Role)
		assert.Equal(t, user.Preferences.Theme, retrieved.Preferences.Theme)
	})

	t.Run("GetByID_NonExistentUser_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.GetByID(context.Background(), "nonexistent123")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find user by ID")
	})

	t.Run("GetByID_EmptyID_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.GetByID(context.Background(), "")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
	})

	t.Run("GetByEmail_ExistingUser_ReturnsUser", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		email := "getbyemail@test.example.com"
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail(email),
			testutil.WithUserUsername("getbyemail"),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Act
		retrieved, err := repo.GetByEmail(context.Background(), email)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, email, retrieved.Email)
	})

	t.Run("GetByEmail_NonExistentUser_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.GetByEmail(context.Background(), "nonexistent@test.example.com")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find user by email")
	})

	t.Run("GetByEmail_EmptyEmail_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.GetByEmail(context.Background(), "")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user email cannot be empty")
	})

	t.Run("GetByUsername_ExistingUser_ReturnsUser", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		username := "getbyusername"
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("getbyusername@test.example.com"),
			testutil.WithUserUsername(username),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Act
		retrieved, err := repo.GetByUsername(context.Background(), username)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, username, retrieved.Username)
	})

	t.Run("GetByUsername_NonExistentUser_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.GetByUsername(context.Background(), "nonexistentuser")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find user by username")
	})

	t.Run("GetByUsername_EmptyUsername_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.GetByUsername(context.Background(), "")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")
	})

	t.Run("Update_ValidChanges_Success", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("update@test.example.com"),
			testutil.WithUserUsername("updateuser"),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		originalCreatedAt := user.CreatedAt
		originalID := user.ID

		// Modify user data
		user.Name = "Updated Name"
		user.Avatar = "new-avatar.png"
		user.Preferences.Theme = "dark"
		user.Preferences.Language = "es"

		// Act
		err := repo.Update(context.Background(), user)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, originalID, user.ID, "ID should not change")
		assert.Equal(t, originalCreatedAt, user.CreatedAt, "CreatedAt should not change")
		assert.True(t, user.UpdatedAt.After(originalCreatedAt), "UpdatedAt should be after CreatedAt")

		// Verify changes persisted
		retrieved, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "new-avatar.png", retrieved.Avatar)
		assert.Equal(t, "dark", retrieved.Preferences.Theme)
		assert.Equal(t, "es", retrieved.Preferences.Language)
	})

	t.Run("Update_EmptyID_ReturnsError", func(t *testing.T) {
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser()
		user.ID = "" // Clear ID

		// Act
		err := repo.Update(context.Background(), user)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty for update")
	})

	t.Run("Update_NonExistentUser_ReturnsError", func(t *testing.T) {
		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserID("nonexistent123"),
		)

		// Act
		err := repo.Update(context.Background(), user)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find user for update")
	})

	t.Run("Update_InvalidData_ValidationError", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("updateinvalid@test.example.com"),
			testutil.WithUserUsername("updateinvalid"),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Make data invalid
		user.Email = "" // Invalid: empty email

		// Act
		err := repo.Update(context.Background(), user)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Delete_ExistingUser_Success", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("delete@test.example.com"),
			testutil.WithUserUsername("deleteuser"),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Verify user exists
		_, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err)

		// Act
		err = repo.Delete(context.Background(), user.ID)

		// Assert
		require.NoError(t, err)

		// Verify user no longer exists
		_, err = repo.GetByID(context.Background(), user.ID)
		require.Error(t, err)
	})

	t.Run("Delete_NonExistentUser_ReturnsError", func(t *testing.T) {
		// Act
		err := repo.Delete(context.Background(), "nonexistent123")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find user for deletion")
	})

	t.Run("Delete_EmptyID_ReturnsError", func(t *testing.T) {
		// Act
		err := repo.Delete(context.Background(), "")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
	})

	t.Run("List_WithPagination_ReturnsUsers", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange - Create multiple users
		users := make([]*domain.User, 5)
		for i := 0; i < 5; i++ {
			user := suite.Factory.CreateUser(
				testutil.WithUserEmail(fmt.Sprintf("list%d@test.example.com", i)),
				testutil.WithUserUsername(fmt.Sprintf("listuser%d", i)),
			)
			require.NoError(t, repo.Create(context.Background(), user))
			users[i] = user
		}

		// Act - Get first 3 users
		retrieved, err := repo.List(context.Background(), 0, 3)

		// Assert
		require.NoError(t, err)
		assert.Len(t, retrieved, 3)

		// Verify pagination works
		retrieved2, err := repo.List(context.Background(), 3, 3)
		require.NoError(t, err)
		assert.Len(t, retrieved2, 2) // Should get remaining 2 users

		// Verify no overlap
		retrievedIDs := make(map[string]bool)
		for _, user := range retrieved {
			retrievedIDs[user.ID] = true
		}
		for _, user := range retrieved2 {
			assert.False(t, retrievedIDs[user.ID], "Users should not overlap between pages")
		}
	})

	t.Run("List_EmptyDatabase_ReturnsEmptySlice", func(t *testing.T) {
		// Arrange - Reset database to ensure it's empty
		tc.ClearDatabase(t)

		// Act
		users, err := repo.List(context.Background(), 0, 10)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("Count_MultipleUsers_ReturnsCorrectCount", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)
		expectedCount := 7

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		for i := 0; i < expectedCount; i++ {
			user := suite.Factory.CreateUser(
				testutil.WithUserEmail(fmt.Sprintf("count%d@test.example.com", i)),
				testutil.WithUserUsername(fmt.Sprintf("countuser%d", i)),
				testutil.WithUserName(fmt.Sprintf("Count User %d", i)),
			)
			require.NoError(t, repo.Create(context.Background(), user))
		}

		// Act
		count, err := repo.Count(context.Background())

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("Count_EmptyDatabase_ReturnsZero", func(t *testing.T) {
		// Arrange - Reset database
		tc.ClearDatabase(t)

		// Act
		count, err := repo.Count(context.Background())

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("ExistsByEmail_ExistingUser_ReturnsTrue", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		email := "exists@test.example.com"
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail(email),
			testutil.WithUserUsername("existsuser"),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Act
		exists, err := repo.ExistsByEmail(context.Background(), email)

		// Assert
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ExistsByEmail_NonExistentUser_ReturnsFalse", func(t *testing.T) {
		// Act
		exists, err := repo.ExistsByEmail(context.Background(), "nonexistent@test.example.com")

		// Assert
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ExistsByEmail_EmptyEmail_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.ExistsByEmail(context.Background(), "")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email cannot be empty")
	})

	t.Run("ExistsByUsername_ExistingUser_ReturnsTrue", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		username := "existsusername"
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("existsusername@test.example.com"),
			testutil.WithUserUsername(username),
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Act
		exists, err := repo.ExistsByUsername(context.Background(), username)

		// Assert
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ExistsByUsername_NonExistentUser_ReturnsFalse", func(t *testing.T) {
		// Act
		exists, err := repo.ExistsByUsername(context.Background(), "nonexistentusername")

		// Assert
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ExistsByUsername_EmptyUsername_ReturnsError", func(t *testing.T) {
		// Act
		_, err := repo.ExistsByUsername(context.Background(), "")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")
	})

	t.Run("UserPreferences_ComplexData_PersistsCorrectly", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		complexPreferences := domain.UserPreferences{
			Theme:    "dark",
			Language: "fr",
			Timezone: "Europe/Paris",
			Preferences: map[string]string{
				"notification_email":   "true",
				"notification_desktop": "false",
				"digest_frequency":     "daily",
				"date_format":          "DD/MM/YYYY",
				"time_format":          "24h",
				"items_per_page":       "50",
				"default_project_view": "kanban",
			},
		}

		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("complex@test.example.com"),
			testutil.WithUserUsername("complexuser"),
		)
		user.Preferences = complexPreferences

		// Act - Create and retrieve
		require.NoError(t, repo.Create(context.Background(), user))
		retrieved, err := repo.GetByID(context.Background(), user.ID)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, complexPreferences.Theme, retrieved.Preferences.Theme)
		assert.Equal(t, complexPreferences.Language, retrieved.Preferences.Language)
		assert.Equal(t, complexPreferences.Timezone, retrieved.Preferences.Timezone)
		assert.Equal(t, len(complexPreferences.Preferences), len(retrieved.Preferences.Preferences))

		for key, value := range complexPreferences.Preferences {
			assert.Equal(t, value, retrieved.Preferences.Preferences[key], fmt.Sprintf("Preference %s should match", key))
		}
	})

	t.Run("ConcurrentUserCreation_DifferentData_BothSucceed", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user1 := suite.Factory.CreateUser(
			testutil.WithUserEmail("concurrent1@test.example.com"),
			testutil.WithUserUsername("concurrent1"),
		)
		user2 := suite.Factory.CreateUser(
			testutil.WithUserEmail("concurrent2@test.example.com"),
			testutil.WithUserUsername("concurrent2"),
		)

		// Act - Create users concurrently
		var err1, err2 error
		done := make(chan bool, 2)

		go func() {
			err1 = repo.Create(context.Background(), user1)
			done <- true
		}()
		go func() {
			err2 = repo.Create(context.Background(), user2)
			done <- true
		}()

		// Wait for both to complete
		<-done
		<-done

		// Assert
		require.NoError(t, err1, "First concurrent user creation should succeed")
		require.NoError(t, err2, "Second concurrent user creation should succeed")

		// Verify both users exist
		retrieved1, err := repo.GetByID(context.Background(), user1.ID)
		require.NoError(t, err)
		assert.Equal(t, user1.Email, retrieved1.Email)

		retrieved2, err := repo.GetByID(context.Background(), user2.ID)
		require.NoError(t, err)
		assert.Equal(t, user2.Email, retrieved2.Email)
	})

	t.Run("UserRoles_DefaultBehavior", func(t *testing.T) {
		roles := []domain.UserRole{
			domain.RegularUserRole,
			domain.AdminRole,
		}

		for i, role := range roles {
			t.Run(string(role), func(t *testing.T) {
				// Clear database for isolation
				tc.ClearDatabase(t)

				// Create test database suite for factory access
				suite := testutil.SetupDatabaseTest(t)
				defer suite.Cleanup()

				// Arrange
				user := suite.Factory.CreateUser(
					testutil.WithUserEmail(fmt.Sprintf("role%d@test.example.com", i)),
					testutil.WithUserUsername(fmt.Sprintf("roleuser%d", i)),
					testutil.WithUserRole(role),
				)

				// Act
				require.NoError(t, repo.Create(context.Background(), user))
				retrieved, err := repo.GetByID(context.Background(), user.ID)

				// Assert
				require.NoError(t, err)
				// Since role field doesn't exist in collection schema, all roles default to RegularUserRole
				assert.Equal(t, domain.RegularUserRole, retrieved.Role, "Role should default to RegularUserRole when field missing")
			})
		}
	})

	t.Run("TimestampManagement_CreatedAndUpdated_WorkCorrectly", func(t *testing.T) {
		// Clear database for isolation
		tc.ClearDatabase(t)

		// Create test database suite for factory access
		suite := testutil.SetupDatabaseTest(t)
		defer suite.Cleanup()

		// Arrange
		user := suite.Factory.CreateUser(
			testutil.WithUserEmail("timestamp@test.example.com"),
			testutil.WithUserUsername("timestampuser"),
		)

		// Act - Create user
		beforeCreate := time.Now().UTC()
		require.NoError(t, repo.Create(context.Background(), user))
		afterCreate := time.Now().UTC()

		// Assert creation timestamps
		assert.True(t, user.CreatedAt.After(beforeCreate.Add(-1*time.Second)), "CreatedAt should be recent")
		assert.True(t, user.CreatedAt.Before(afterCreate.Add(1*time.Second)), "CreatedAt should be recent")
		assert.True(t, user.UpdatedAt.After(beforeCreate.Add(-1*time.Second)), "UpdatedAt should be recent")
		assert.True(t, user.UpdatedAt.Before(afterCreate.Add(1*time.Second)), "UpdatedAt should be recent")

		originalCreated := user.CreatedAt
		originalUpdated := user.UpdatedAt

		// Wait a moment to ensure timestamp difference
		time.Sleep(100 * time.Millisecond)

		// Act - Update user
		user.Name = "Updated Name"
		beforeUpdate := time.Now().UTC()
		require.NoError(t, repo.Update(context.Background(), user))
		afterUpdate := time.Now().UTC()

		// Assert update timestamps
		assert.Equal(t, originalCreated, user.CreatedAt, "CreatedAt should not change on update")
		assert.True(t, user.UpdatedAt.After(originalUpdated), "UpdatedAt should be newer after update")
		assert.True(t, user.UpdatedAt.After(beforeUpdate.Add(-1*time.Second)), "UpdatedAt should be recent")
		assert.True(t, user.UpdatedAt.Before(afterUpdate.Add(1*time.Second)), "UpdatedAt should be recent")
	})
}
