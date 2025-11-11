package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	assert.NotNil(t, db)
	assert.Equal(t, dbPath, db.Path())

	// Verify database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestOpen_CreatesParentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "subdir", "nested", "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify parent directories were created
	_, err = os.Stat(filepath.Dir(dbPath))
	assert.NoError(t, err)
}

func TestInitialize_EnablesWAL(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Check WAL mode is enabled
	var journalMode string
	err = db.conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
}

func TestInitialize_CreatesTables(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify schema_version table exists
	var count int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='schema_version'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify history table exists
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='history'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInitialize_CreatesIndexes(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify indexes exist
	expectedIndexes := []string{
		"idx_timestamp",
		"idx_command",
		"idx_hash",
		"idx_session",
		"idx_cwd",
	}

	for _, indexName := range expectedIndexes {
		var count int
		err = db.conn.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name=?
		`, indexName).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "index %s should exist", indexName)
	}
}

func TestGetSchemaVersion(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	version, err := db.getSchemaVersion()
	require.NoError(t, err)
	assert.Equal(t, CurrentSchema, version)
}

func TestMigrate_IdempotentV1(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Open database first time
	db1, err := Open(dbPath)
	require.NoError(t, err)
	db1.Close()

	// Open database second time (should not fail)
	db2, err := Open(dbPath)
	require.NoError(t, err)
	defer db2.Close()

	version, err := db2.getSchemaVersion()
	require.NoError(t, err)
	assert.Equal(t, CurrentSchema, version)
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)

	err = db.Close()
	assert.NoError(t, err)

	// Trying to use closed connection should fail
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM history").Scan(&count)
	assert.Error(t, err)
}

func TestQueryContext(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert test data
	entry := &HistoryEntry{
		Timestamp:  1234567890,
		Command:    "test command",
		Cwd:        "/home/user",
		ExitCode:   0,
		Hostname:   "localhost",
		User:       "testuser",
		Shell:      "bash",
		DurationMs: 100,
		Hash:       GenerateHash("test command"),
	}
	err = db.Insert(entry)
	require.NoError(t, err)

	t.Run("query with context", func(t *testing.T) {
		ctx := context.Background()
		rows, err := db.QueryContext(ctx, "SELECT * FROM history WHERE command = ?", "test command")
		require.NoError(t, err)
		defer rows.Close()

		// Verify we got results
		hasRow := rows.Next()
		assert.True(t, hasRow)
	})

	t.Run("query with timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, "SELECT * FROM history")
		require.NoError(t, err)
		defer rows.Close()

		// Should complete successfully
		assert.NotNil(t, rows)
	})

	t.Run("query with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := db.QueryContext(ctx, "SELECT * FROM history")
		// Should get a context error
		assert.Error(t, err)
	})
}
