package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gavinmcnair/tvproxy/pkg/database"
	"github.com/gavinmcnair/tvproxy/pkg/models"
)

func setupTestDB(t *testing.T) *database.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	db, err := database.New(context.Background(), dbPath, log)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestUserCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Username:     "testuser",
		PasswordHash: "$2a$10$fakehashfakehashfakehashfakehashfakehashfakehashfakeh",
		IsAdmin:      false,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	assert.NotZero(t, user.ID, "user ID should be set after creation")
	assert.Equal(t, "testuser", user.Username)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestUserCreateDuplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user1 := &models.User{
		Username:     "testuser",
		PasswordHash: "hash1",
		IsAdmin:      false,
	}
	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2 := &models.User{
		Username:     "testuser",
		PasswordHash: "hash2",
		IsAdmin:      false,
	}
	err = repo.Create(ctx, user2)
	assert.Error(t, err, "creating a user with duplicate username should fail due to UNIQUE constraint")
}

func TestUserGetByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Username:     "findme",
		PasswordHash: "somehash",
		IsAdmin:      true,
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByUsername(ctx, "findme")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "findme", found.Username)
	assert.Equal(t, "somehash", found.PasswordHash)
	assert.True(t, found.IsAdmin)
}

func TestUserGetByUsernameNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByUsername(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Username:     "testuser",
		PasswordHash: "somehash",
		IsAdmin:      false,
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "testuser", found.Username)
}

func TestUserGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Empty list
	users, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, users)

	// Create multiple users
	for _, name := range []string{"alice", "bob", "charlie"} {
		err := repo.Create(ctx, &models.User{
			Username:     name,
			PasswordHash: "hash_" + name,
			IsAdmin:      name == "alice",
		})
		require.NoError(t, err)
	}

	users, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// Verify ordered by ID
	assert.Equal(t, "alice", users[0].Username)
	assert.Equal(t, "bob", users[1].Username)
	assert.Equal(t, "charlie", users[2].Username)

	// Verify admin flag
	assert.True(t, users[0].IsAdmin)
	assert.False(t, users[1].IsAdmin)
	assert.False(t, users[2].IsAdmin)
}

func TestUserUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Username:     "testuser",
		PasswordHash: "originalhash",
		IsAdmin:      false,
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)
	originalUpdatedAt := user.UpdatedAt

	// Update the user
	user.Username = "updateduser"
	user.PasswordHash = "newhash"
	user.IsAdmin = true
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// The UpdatedAt should have been refreshed
	assert.True(t, user.UpdatedAt.After(originalUpdatedAt) || user.UpdatedAt.Equal(originalUpdatedAt))

	// Verify changes persisted
	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updateduser", found.Username)
	assert.Equal(t, "newhash", found.PasswordHash)
	assert.True(t, found.IsAdmin)
}

func TestUserDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Username:     "testuser",
		PasswordHash: "somehash",
		IsAdmin:      false,
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Verify the user exists
	_, err = repo.GetByID(ctx, user.ID)
	require.NoError(t, err)

	// Delete the user
	err = repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	// Verify the user is gone
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserDeleteNonExistent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Deleting a non-existent user should not error (DELETE WHERE id=? affects 0 rows)
	err := repo.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}
