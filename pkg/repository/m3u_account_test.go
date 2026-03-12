package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gavinmcnair/tvproxy/pkg/models"
)

func TestM3UAccountCRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewM3UAccountRepository(db)
	ctx := context.Background()

	// Create
	account := &models.M3UAccount{
		Name:            "Test IPTV",
		URL:             "http://example.com/playlist.m3u",
		Type:            "m3u",
		Username:        "",
		Password:        "",
		MaxStreams:      2,
		IsEnabled:       true,
		StreamCount:     0,
		RefreshInterval: 3600,
	}
	err := repo.Create(ctx, account)
	require.NoError(t, err)
	assert.NotZero(t, account.ID)
	assert.False(t, account.CreatedAt.IsZero())
	assert.False(t, account.UpdatedAt.IsZero())

	// Read
	fetched, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, account.ID, fetched.ID)
	assert.Equal(t, "Test IPTV", fetched.Name)
	assert.Equal(t, "http://example.com/playlist.m3u", fetched.URL)
	assert.Equal(t, "m3u", fetched.Type)
	assert.Equal(t, 2, fetched.MaxStreams)
	assert.True(t, fetched.IsEnabled)
	assert.Nil(t, fetched.LastRefreshed)
	assert.Equal(t, 0, fetched.StreamCount)
	assert.Equal(t, 3600, fetched.RefreshInterval)

	// Update
	account.Name = "Updated IPTV"
	account.URL = "http://example.com/updated.m3u"
	account.MaxStreams = 5
	account.IsEnabled = false
	account.RefreshInterval = 7200
	err = repo.Update(ctx, account)
	require.NoError(t, err)

	fetched, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated IPTV", fetched.Name)
	assert.Equal(t, "http://example.com/updated.m3u", fetched.URL)
	assert.Equal(t, 5, fetched.MaxStreams)
	assert.False(t, fetched.IsEnabled)
	assert.Equal(t, 7200, fetched.RefreshInterval)

	// List
	account2 := &models.M3UAccount{
		Name:       "Second Account",
		URL:        "http://example.com/second.m3u",
		Type:       "xtream",
		Username:   "user",
		Password:   "pass",
		MaxStreams: 1,
		IsEnabled:  true,
	}
	err = repo.Create(ctx, account2)
	require.NoError(t, err)

	accounts, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
	assert.Equal(t, "Updated IPTV", accounts[0].Name)
	assert.Equal(t, "Second Account", accounts[1].Name)
	assert.Equal(t, "xtream", accounts[1].Type)
	assert.Equal(t, "user", accounts[1].Username)
	assert.Equal(t, "pass", accounts[1].Password)

	// Delete
	err = repo.Delete(ctx, account.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, account.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "m3u account not found")

	// Verify only account2 remains
	accounts, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "Second Account", accounts[0].Name)
}

func TestM3UAccountGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewM3UAccountRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "m3u account not found")
}

func TestM3UAccountUpdateLastRefreshed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewM3UAccountRepository(db)
	ctx := context.Background()

	account := &models.M3UAccount{
		Name:       "Test Account",
		URL:        "http://example.com/playlist.m3u",
		Type:       "m3u",
		MaxStreams: 1,
		IsEnabled:  true,
	}
	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Initially, LastRefreshed should be nil
	fetched, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched.LastRefreshed)

	// Update last refreshed
	refreshTime := time.Now().Truncate(time.Second)
	err = repo.UpdateLastRefreshed(ctx, account.ID, refreshTime)
	require.NoError(t, err)

	// Verify
	fetched, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.LastRefreshed)
	assert.WithinDuration(t, refreshTime, *fetched.LastRefreshed, 2*time.Second)
}

func TestM3UAccountUpdateStreamCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewM3UAccountRepository(db)
	ctx := context.Background()

	account := &models.M3UAccount{
		Name:       "Test Account",
		URL:        "http://example.com/playlist.m3u",
		Type:       "m3u",
		MaxStreams: 1,
		IsEnabled:  true,
	}
	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Initially, stream count should be 0
	fetched, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, fetched.StreamCount)

	// Update stream count
	err = repo.UpdateStreamCount(ctx, account.ID, 42)
	require.NoError(t, err)

	// Verify
	fetched, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, 42, fetched.StreamCount)

	// Update again
	err = repo.UpdateStreamCount(ctx, account.ID, 100)
	require.NoError(t, err)

	fetched, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, 100, fetched.StreamCount)
}

func TestM3UAccountListEmpty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewM3UAccountRepository(db)
	ctx := context.Background()

	accounts, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestM3UAccountWithXtreamCredentials(t *testing.T) {
	db := setupTestDB(t)
	repo := NewM3UAccountRepository(db)
	ctx := context.Background()

	account := &models.M3UAccount{
		Name:       "Xtream Provider",
		URL:        "http://xtream.example.com",
		Type:       "xtream",
		Username:   "myuser",
		Password:   "mypass",
		MaxStreams: 3,
		IsEnabled:  true,
	}
	err := repo.Create(ctx, account)
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, "xtream", fetched.Type)
	assert.Equal(t, "myuser", fetched.Username)
	assert.Equal(t, "mypass", fetched.Password)
}
