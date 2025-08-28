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

	"simple-easy-tasks/internal/domain"
)

// createUserWithPassword is a helper function that creates a user and sets password
// For now, only use fields that work with default PocketBase auth collections
func createUserWithPassword(email, name string) *domain.User {
	user := &domain.User{
		Email: email,
		Name:  name,
		// Don't set username, role, or preferences since they don't exist in default auth collection
	}
	// Set password for auth collection - ignore error since we know it's valid
	_ = user.SetPassword("testpassword123")
	return user
}

// TestUserRepository_BasicIntegration tests core user repository functionality
// with only the fields that are working properly in the current schema
func TestUserRepository_BasicIntegration(t *testing.T) {
	// Setup test container with DI
	tc := NewTestContainer(t)
	defer tc.Cleanup()

	// Get repository from DI container
	repo := tc.GetUserRepository(t)

	t.Run("Create_ValidUser_BasicFields_Success", func(t *testing.T) {
		// Arrange - Create a user with minimal working fields
		user := createUserWithPassword(
			"basic@test.example.com",
			"Basic Test User",
		)

		// Act
		err := repo.Create(context.Background(), user)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())

		// Verify the user can be retrieved and basic fields work
		retrieved, err := repo.GetByID(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Name, retrieved.Name)
		assert.Equal(t, user.ID, retrieved.ID)
	})

	t.Run("Create_DuplicateEmail_ConstraintViolation", func(t *testing.T) {
		// Arrange
		email := "duplicate.basic@test.example.com"
		user1 := createUserWithPassword(
			email,
			"User One",
		)
		user2 := createUserWithPassword(
			email, // Same email should fail
			"User Two",
		)

		// Act
		err1 := repo.Create(context.Background(), user1)
		err2 := repo.Create(context.Background(), user2)

		// Assert
		require.NoError(t, err1, "First user creation should succeed")
		require.Error(t, err2, "Second user with same email should fail")
		// Check for constraint violation in error message (might be different format)
		assert.Contains(t, err2.Error(), "email", "Error should mention email field")
	})

	t.Run("GetByID_ExistingUser_ReturnsUser", func(t *testing.T) {
		// Arrange
		user := createUserWithPassword(
			"getbyid.basic@test.example.com",
			"GetByID Test User",
		)
		require.NoError(t, repo.Create(context.Background(), user))

		// Act
		retrieved, err := repo.GetByID(context.Background(), user.ID)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Name, retrieved.Name)
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
		// Arrange
		email := "getbyemail.basic@test.example.com"
		user := createUserWithPassword(
			email,
			"GetByEmail Test User",
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
		_, err := repo.GetByEmail(context.Background(), "nonexistent.basic@test.example.com")

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

	t.Run("Update_ValidChanges_Success", func(t *testing.T) {
		// Arrange
		user := createUserWithPassword(
			"update.basic@test.example.com",
			"Original Name",
		)
		require.NoError(t, repo.Create(context.Background(), user))

		originalCreatedAt := user.CreatedAt
		originalID := user.ID

		// Modify user data
		user.Name = "Updated Name"

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
	})

	t.Run("Update_EmptyID_ReturnsError", func(t *testing.T) {
		// Arrange
		user := createUserWithPassword(
			"update.emptyid@test.example.com",
			"Test User",
		)
		user.ID = "" // Clear ID

		// Act
		err := repo.Update(context.Background(), user)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be empty for update")
	})

	t.Run("Update_NonExistentUser_ReturnsError", func(t *testing.T) {
		// Arrange
		user := createUserWithPassword(
			"update.nonexistent@test.example.com",
			"Test User",
		)
		user.ID = "nonexistent123" // Set after creation

		// Act
		err := repo.Update(context.Background(), user)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find user for update")
	})

	t.Run("Delete_ExistingUser_Success", func(t *testing.T) {
		// Arrange
		user := createUserWithPassword(
			"delete.basic@test.example.com",
			"Delete Test User",
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
		// Arrange - Clear database for isolation
		tc.ClearDatabase(t)
		expectedCount := 5
		for i := 0; i < expectedCount; i++ {
			user := createUserWithPassword(
				fmt.Sprintf("list%d@test.example.com", i),
				fmt.Sprintf("List User %d", i),
			)
			require.NoError(t, repo.Create(context.Background(), user))
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
		// Arrange - Clear database to ensure it's empty
		tc.ClearDatabase(t)

		// Act
		users, err := repo.List(context.Background(), 0, 10)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("Count_MultipleUsers_ReturnsCorrectCount", func(t *testing.T) {
		// Arrange - Clear database and create known number of users
		tc.ClearDatabase(t)
		expectedCount := 7
		for i := 0; i < expectedCount; i++ {
			user := createUserWithPassword(
				fmt.Sprintf("count%d@test.example.com", i),
				fmt.Sprintf("Count User %d", i),
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
		// Arrange - Clear database
		tc.ClearDatabase(t)

		// Act
		count, err := repo.Count(context.Background())

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("ExistsByEmail_ExistingUser_ReturnsTrue", func(t *testing.T) {
		// Arrange
		email := "exists.basic@test.example.com"
		user := createUserWithPassword(
			email,
			"Exists Test User",
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
		exists, err := repo.ExistsByEmail(context.Background(), "nonexistent.basic@test.example.com")

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

	t.Run("ConcurrentUserCreation_DifferentData_BothSucceed", func(t *testing.T) {
		// Arrange
		user1 := createUserWithPassword(
			"concurrent1.basic@test.example.com",
			"Concurrent User 1",
		)
		user2 := createUserWithPassword(
			"concurrent2.basic@test.example.com",
			"Concurrent User 2",
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

	t.Run("TimestampManagement_CreatedAndUpdated_WorkCorrectly", func(t *testing.T) {
		// Arrange
		user := createUserWithPassword(
			"timestamp.basic@test.example.com",
			"Timestamp Test User",
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
