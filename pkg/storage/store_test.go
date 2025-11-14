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

func TestDeleteByFilter_EdgeCases(t *testing.T) {
	t.Run("delete with no matches", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entry := createTestEntry(t, "ls -la", 1000)
		require.NoError(t, db.Insert(entry))

		deleted, err := db.DeleteByFilter(QueryFilters{Search: "nonexistent"})
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)

		count, err := db.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("delete by cwd", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "cmd1", 1000),
			createTestEntry(t, "cmd2", 2000),
		}
		entries[0].Cwd = "/home/user"
		entries[1].Cwd = "/tmp"

		for _, entry := range entries {
			require.NoError(t, db.Insert(entry))
		}

		deleted, err := db.DeleteByFilter(QueryFilters{Cwd: "/tmp"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)
	})

	t.Run("delete by exit code", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "success", 1000),
			createTestEntry(t, "failed", 2000),
		}
		entries[0].ExitCode = 0
		entries[1].ExitCode = 1

		for _, entry := range entries {
			require.NoError(t, db.Insert(entry))
		}

		exitCode := 1
		deleted, err := db.DeleteByFilter(QueryFilters{ExitCode: &exitCode})
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)
	})

	t.Run("delete by time range", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "old", 1000),
			createTestEntry(t, "middle", 2000),
			createTestEntry(t, "new", 3000),
		}

		for _, entry := range entries {
			require.NoError(t, db.Insert(entry))
		}

		deleted, err := db.DeleteByFilter(QueryFilters{
			Before: 2500,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)
	})

	t.Run("delete from empty database", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		deleted, err := db.DeleteByFilter(QueryFilters{Search: "anything"})
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})
}

func TestQuery_EdgeCases(t *testing.T) {
	t.Run("query with limit 0 returns all", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		for i := 0; i < 10; i++ {
			entry := createTestEntry(t, "cmd", int64(i*1000))
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		results, err := db.Query(QueryFilters{Limit: 0})
		require.NoError(t, err)
		assert.Equal(t, 10, len(results))
	})

	t.Run("query with very large limit", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entry := createTestEntry(t, "cmd", 1000)
		require.NoError(t, db.Insert(entry))

		results, err := db.Query(QueryFilters{Limit: 10000})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("query with both before and after", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "cmd1", 1000),
			createTestEntry(t, "cmd2", 2000),
			createTestEntry(t, "cmd3", 3000),
			createTestEntry(t, "cmd4", 4000),
		}

		for _, entry := range entries {
			require.NoError(t, db.Insert(entry))
		}

		results, err := db.Query(QueryFilters{
			After:  1500,
			Before: 3500,
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("query with empty search term", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entry := createTestEntry(t, "test", 1000)
		require.NoError(t, db.Insert(entry))

		results, err := db.Query(QueryFilters{Search: ""})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

func TestInsert_EdgeCases(t *testing.T) {
	t.Run("insert entry with empty command", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entry := createTestEntry(t, "", 1000)
		err := db.Insert(entry)
		assert.NoError(t, err)

		count, err := db.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("insert entry with special characters in command", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		specialCommands := []string{
			"echo 'hello \"world\"'",
			"grep -r \"pattern\" | awk '{print $1}'",
			"for i in $(seq 1 10); do echo $i; done",
			"command with\nnewline",
			"command with\ttab",
		}

		for _, cmd := range specialCommands {
			entry := createTestEntry(t, cmd, time.Now().Unix())
			entry.Hash = GenerateHash(cmd)
			err := db.Insert(entry)
			require.NoError(t, err)
		}

		count, err := db.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("insert entry with very long command", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		longCmd := ""
		for i := 0; i < 1000; i++ {
			longCmd += "a"
		}

		entry := createTestEntry(t, longCmd, 1000)
		entry.Hash = GenerateHash(longCmd)
		err := db.Insert(entry)
		assert.NoError(t, err)
	})

	t.Run("insert entry with negative exit code", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entry := createTestEntry(t, "cmd", 1000)
		entry.ExitCode = -1
		err := db.Insert(entry)
		assert.NoError(t, err)
	})

	t.Run("insert entry with negative timestamp", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entry := createTestEntry(t, "cmd", -1)
		err := db.Insert(entry)
		assert.NoError(t, err)
	})
}

func TestCount_EdgeCases(t *testing.T) {
	t.Run("count after delete", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		for i := 0; i < 5; i++ {
			entry := createTestEntry(t, "cmd", int64(i*1000))
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		count, err := db.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		// Delete some
		results, err := db.Query(QueryFilters{Limit: 2})
		require.NoError(t, err)
		for _, r := range results {
			db.Delete(r.ID)
		}

		count, err = db.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}

func TestQuery_Distinct(t *testing.T) {
	t.Run("distinct returns unique commands only", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert duplicate commands with different timestamps
		entries := []*HistoryEntry{
			createTestEntry(t, "ls -la", 1000),
			createTestEntry(t, "git status", 2000),
			createTestEntry(t, "ls -la", 3000),      // Duplicate of ls -la
			createTestEntry(t, "git status", 4000),  // Duplicate of git status
			createTestEntry(t, "pwd", 5000),         // Unique
			createTestEntry(t, "ls -la", 6000),      // Another duplicate of ls -la
		}

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i)) // Make hashes unique
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true
		results, err := db.Query(QueryFilters{Distinct: true})
		require.NoError(t, err)

		// Should return only 3 unique commands
		assert.Len(t, results, 3)

		// Commands should be unique
		commands := make(map[string]bool)
		for _, r := range results {
			assert.False(t, commands[r.Command], "Command %s appears twice", r.Command)
			commands[r.Command] = true
		}

		// Should have all three unique commands
		assert.True(t, commands["ls -la"])
		assert.True(t, commands["git status"])
		assert.True(t, commands["pwd"])
	})

	t.Run("distinct returns most recent entry for each command", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert same command at different times with different metadata
		entries := []*HistoryEntry{
			createTestEntry(t, "git status", 1000),
			createTestEntry(t, "git status", 2000),
			createTestEntry(t, "git status", 3000),
		}
		entries[0].Cwd = "/home/old"
		entries[1].Cwd = "/home/middle"
		entries[2].Cwd = "/home/recent"

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true
		results, err := db.Query(QueryFilters{Distinct: true})
		require.NoError(t, err)

		// Should return only the most recent entry
		assert.Len(t, results, 1)
		assert.Equal(t, "git status", results[0].Command)
		assert.Equal(t, int64(3000), results[0].Timestamp)
		assert.Equal(t, "/home/recent", results[0].Cwd)
	})

	t.Run("distinct false returns all entries", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert duplicate commands
		entries := []*HistoryEntry{
			createTestEntry(t, "ls -la", 1000),
			createTestEntry(t, "ls -la", 2000),
			createTestEntry(t, "ls -la", 3000),
		}

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=false (default behavior)
		results, err := db.Query(QueryFilters{Distinct: false})
		require.NoError(t, err)

		// Should return all entries
		assert.Len(t, results, 3)
	})

	t.Run("distinct with same timestamp uses max id as tiebreaker", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert same command with same timestamp (rapid-fire commands)
		timestamp := int64(1000)
		entries := []*HistoryEntry{
			createTestEntry(t, "echo test", timestamp),
			createTestEntry(t, "echo test", timestamp),
			createTestEntry(t, "echo test", timestamp),
		}
		entries[0].Cwd = "/first"
		entries[1].Cwd = "/second"
		entries[2].Cwd = "/third"

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true
		results, err := db.Query(QueryFilters{Distinct: true})
		require.NoError(t, err)

		// Should return only one entry (the one with highest ID)
		assert.Len(t, results, 1)
		assert.Equal(t, "echo test", results[0].Command)
		// Should be the last inserted entry (highest ID)
		assert.Equal(t, "/third", results[0].Cwd)
	})

	t.Run("distinct with search filter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "git status", 1000),
			createTestEntry(t, "git commit", 2000),
			createTestEntry(t, "git status", 3000),  // Duplicate
			createTestEntry(t, "ls -la", 4000),      // Not a git command
			createTestEntry(t, "git commit", 5000),  // Duplicate
		}

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true and search filter
		results, err := db.Query(QueryFilters{
			Distinct: true,
			Search:   "git",
		})
		require.NoError(t, err)

		// Should return only 2 unique git commands
		assert.Len(t, results, 2)

		commands := make(map[string]bool)
		for _, r := range results {
			commands[r.Command] = true
		}
		assert.True(t, commands["git status"])
		assert.True(t, commands["git commit"])
	})

	t.Run("distinct with time range filter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "cmd1", 1000),
			createTestEntry(t, "cmd2", 2000),
			createTestEntry(t, "cmd1", 3000),  // Duplicate, but in range
			createTestEntry(t, "cmd2", 4000),  // Duplicate, but in range
			createTestEntry(t, "cmd1", 5000),  // Duplicate, but out of range
		}

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true and time range
		results, err := db.Query(QueryFilters{
			Distinct: true,
			After:    1500,
			Before:   4500,
		})
		require.NoError(t, err)

		// Should return 2 unique commands within the time range
		assert.Len(t, results, 2)

		// Should be the most recent entries within the range
		commands := make(map[string]int64)
		for _, r := range results {
			commands[r.Command] = r.Timestamp
		}
		assert.Equal(t, int64(3000), commands["cmd1"])
		assert.Equal(t, int64(4000), commands["cmd2"])
	})

	t.Run("distinct with cwd filter", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "ls", 1000),
			createTestEntry(t, "pwd", 2000),
			createTestEntry(t, "ls", 3000),  // Duplicate
		}
		entries[0].Cwd = "/home/user"
		entries[1].Cwd = "/tmp"
		entries[2].Cwd = "/home/user"

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true and cwd filter
		results, err := db.Query(QueryFilters{
			Distinct: true,
			Cwd:      "/home/user",
		})
		require.NoError(t, err)

		// Should return only unique commands in /home/user
		assert.Len(t, results, 1)
		assert.Equal(t, "ls", results[0].Command)
		assert.Equal(t, int64(3000), results[0].Timestamp) // Most recent
	})

	t.Run("distinct with limit", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Insert multiple unique commands
		entries := []*HistoryEntry{
			createTestEntry(t, "cmd1", 1000),
			createTestEntry(t, "cmd2", 2000),
			createTestEntry(t, "cmd3", 3000),
			createTestEntry(t, "cmd4", 4000),
			createTestEntry(t, "cmd5", 5000),
		}

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		// Query with Distinct=true and limit
		results, err := db.Query(QueryFilters{
			Distinct: true,
			Limit:    3,
		})
		require.NoError(t, err)

		// Should return only 3 results (most recent)
		assert.Len(t, results, 3)
		assert.Equal(t, "cmd5", results[0].Command)
		assert.Equal(t, "cmd4", results[1].Command)
		assert.Equal(t, "cmd3", results[2].Command)
	})

	t.Run("distinct with empty database", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		results, err := db.Query(QueryFilters{Distinct: true})
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("distinct ordering is by timestamp desc", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		entries := []*HistoryEntry{
			createTestEntry(t, "cmd1", 1000),
			createTestEntry(t, "cmd2", 5000),
			createTestEntry(t, "cmd3", 3000),
		}

		for i, entry := range entries {
			entry.Hash = entry.Command + string(rune(i))
			require.NoError(t, db.Insert(entry))
		}

		results, err := db.Query(QueryFilters{Distinct: true})
		require.NoError(t, err)

		// Should be ordered by timestamp descending
		assert.Len(t, results, 3)
		assert.Equal(t, "cmd2", results[0].Command)
		assert.Equal(t, "cmd3", results[1].Command)
		assert.Equal(t, "cmd1", results[2].Command)
	})
}
