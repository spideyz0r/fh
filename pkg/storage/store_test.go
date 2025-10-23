package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)

	return db
}

func createTestEntry(t *testing.T, command string, timestamp int64) *HistoryEntry {
	t.Helper()
	return &HistoryEntry{
		Timestamp:  timestamp,
		Command:    command,
		Cwd:        "/home/user",
		ExitCode:   0,
		Hostname:   "localhost",
		User:       "testuser",
		Shell:      "bash",
		DurationMs: 100,
		GitBranch:  "main",
		Hash:       command, // Using command as hash for simplicity in tests
		SessionID:  "session-123",
	}
}

func TestInsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry := createTestEntry(t, "ls -la", time.Now().Unix())

	err := db.Insert(entry)
	assert.NoError(t, err)

	// Verify entry was inserted
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestInsert_DuplicateHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry1 := createTestEntry(t, "ls -la", time.Now().Unix())
	entry2 := createTestEntry(t, "ls -la", time.Now().Unix()+1)

	// First insert should succeed
	err := db.Insert(entry1)
	assert.NoError(t, err)

	// Second insert with same hash should fail
	err = db.Insert(entry2)
	assert.Error(t, err)
}

func TestQuery_All(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert test entries
	entries := []*HistoryEntry{
		createTestEntry(t, "ls -la", 1000),
		createTestEntry(t, "cd /tmp", 2000),
		createTestEntry(t, "pwd", 3000),
	}

	for _, entry := range entries {
		require.NoError(t, db.Insert(entry))
	}

	// Query all
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Results should be ordered by timestamp DESC
	assert.Equal(t, "pwd", results[0].Command)
	assert.Equal(t, "cd /tmp", results[1].Command)
	assert.Equal(t, "ls -la", results[2].Command)
}

func TestQuery_WithSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entries := []*HistoryEntry{
		createTestEntry(t, "git status", 1000),
		createTestEntry(t, "git commit", 2000),
		createTestEntry(t, "ls -la", 3000),
	}

	for _, entry := range entries {
		require.NoError(t, db.Insert(entry))
	}

	// Search for git commands
	results, err := db.Query(QueryFilters{Search: "git"})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "git commit", results[0].Command)
	assert.Equal(t, "git status", results[1].Command)
}

func TestQuery_WithCwd(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry1 := createTestEntry(t, "ls", 1000)
	entry1.Cwd = "/home/user"

	entry2 := createTestEntry(t, "pwd", 2000)
	entry2.Cwd = "/tmp"
	entry2.Hash = "pwd" // Different hash

	require.NoError(t, db.Insert(entry1))
	require.NoError(t, db.Insert(entry2))

	// Query by cwd
	results, err := db.Query(QueryFilters{Cwd: "/tmp"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "pwd", results[0].Command)
}

func TestQuery_WithTimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entries := []*HistoryEntry{
		createTestEntry(t, "cmd1", 1000),
		createTestEntry(t, "cmd2", 2000),
		createTestEntry(t, "cmd3", 3000),
	}

	for _, entry := range entries {
		require.NoError(t, db.Insert(entry))
	}

	// Query with time range
	results, err := db.Query(QueryFilters{
		After:  1500,
		Before: 2500,
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "cmd2", results[0].Command)
}

func TestQuery_WithExitCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry1 := createTestEntry(t, "success", 1000)
	entry1.ExitCode = 0

	entry2 := createTestEntry(t, "failure", 2000)
	entry2.ExitCode = 1
	entry2.Hash = "failure"

	require.NoError(t, db.Insert(entry1))
	require.NoError(t, db.Insert(entry2))

	// Query failed commands
	exitCode := 1
	results, err := db.Query(QueryFilters{ExitCode: &exitCode})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "failure", results[0].Command)
}

func TestQuery_WithPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert 10 entries
	for i := 0; i < 10; i++ {
		entry := createTestEntry(t, "cmd", int64(i*1000))
		entry.Hash = entry.Command + string(rune(i)) // Make unique
		require.NoError(t, db.Insert(entry))
	}

	// First page
	results, err := db.Query(QueryFilters{Limit: 5, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, results, 5)

	// Second page
	results, err = db.Query(QueryFilters{Limit: 5, Offset: 5})
	require.NoError(t, err)
	assert.Len(t, results, 5)
}

func TestQuery_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entries := []*HistoryEntry{
		createTestEntry(t, "git status", 1000),
		createTestEntry(t, "git commit", 2000),
		createTestEntry(t, "ls -la", 3000),
		createTestEntry(t, "git push", 4000),
	}

	for _, entry := range entries {
		require.NoError(t, db.Insert(entry))
	}

	// Combine search and time range
	results, err := db.Query(QueryFilters{
		Search: "git",
		After:  1500,
		Before: 3500,
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "git commit", results[0].Command)
}

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry := createTestEntry(t, "test command", time.Now().Unix())
	require.NoError(t, db.Insert(entry))

	// Get all to find ID
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Get by ID
	found, err := db.GetByID(results[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "test command", found.Command)
}

func TestGetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.GetByID(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initially empty
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Insert entries
	for i := 0; i < 5; i++ {
		entry := createTestEntry(t, "cmd", int64(i*1000))
		entry.Hash = entry.Command + string(rune(i))
		require.NoError(t, db.Insert(entry))
	}

	count, err = db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry := createTestEntry(t, "delete me", time.Now().Unix())
	require.NoError(t, db.Insert(entry))

	// Get ID
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	id := results[0].ID

	// Delete
	err = db.Delete(id)
	assert.NoError(t, err)

	// Verify deleted
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.Delete(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteByFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entries := []*HistoryEntry{
		createTestEntry(t, "git status", 1000),
		createTestEntry(t, "git commit", 2000),
		createTestEntry(t, "ls -la", 3000),
	}

	for _, entry := range entries {
		require.NoError(t, db.Insert(entry))
	}

	// Delete git commands
	deleted, err := db.DeleteByFilter(QueryFilters{Search: "git"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify remaining
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
