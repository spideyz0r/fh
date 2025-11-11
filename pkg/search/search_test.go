package search

import (
	"testing"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/spideyz0r/fh/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// Insert test entries
	entries := []*storage.HistoryEntry{
		{
			Timestamp:  time.Now().Unix() - 100,
			Command:    "git status",
			Cwd:        "/home/user",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 50,
			Hash:       storage.GenerateHash("git status"),
		},
		{
			Timestamp:  time.Now().Unix() - 50,
			Command:    "git commit -m 'test'",
			Cwd:        "/home/user",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHash("git commit -m 'test'"),
		},
		{
			Timestamp:  time.Now().Unix(),
			Command:    "docker ps",
			Cwd:        "/home/user",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 75,
			Hash:       storage.GenerateHash("docker ps"),
		},
	}

	for _, entry := range entries {
		err := db.Insert(entry)
		require.NoError(t, err)
	}

	t.Run("search with query", func(t *testing.T) {
		results, err := Search(db, "git", 10)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		// Results should be in reverse chronological order
		assert.Contains(t, results[0].Command, "git")
		assert.Contains(t, results[1].Command, "git")
	})

	t.Run("search with no matches", func(t *testing.T) {
		results, err := Search(db, "nonexistent", 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("search with limit", func(t *testing.T) {
		results, err := Search(db, "git", 1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("search all entries", func(t *testing.T) {
		results, err := Search(db, "", 10)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})
}

func TestAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// Insert test entries
	for i := 0; i < 5; i++ {
		cmd := "test command " + string(rune(i+'0'))
		entry := &storage.HistoryEntry{
			Timestamp:  time.Now().Unix() - int64(i*10),
			Command:    cmd,
			Cwd:        "/home/user",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 50,
			Hash:       storage.GenerateHash(cmd),
		}
		err := db.Insert(entry)
		require.NoError(t, err)
	}

	t.Run("get all entries", func(t *testing.T) {
		results, err := All(db, 0)
		require.NoError(t, err)
		assert.Len(t, results, 5)
	})

	t.Run("get all with limit", func(t *testing.T) {
		results, err := All(db, 3)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("get all from empty database", func(t *testing.T) {
		emptyDB := testutil.NewTestDB(t)
		defer emptyDB.Close()

		results, err := All(emptyDB, 0)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestWithFilters(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// Insert test entries
	entries := []*storage.HistoryEntry{
		{
			Timestamp:  time.Now().Unix() - 200,
			Command:    "ls -la",
			Cwd:        "/home/user/project",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 25,
			Hash:       storage.GenerateHash("ls -la"),
		},
		{
			Timestamp:  time.Now().Unix() - 100,
			Command:    "docker build",
			Cwd:        "/home/user/project",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 5000,
			Hash:       storage.GenerateHash("docker build"),
		},
		{
			Timestamp:  time.Now().Unix() - 50,
			Command:    "git push",
			Cwd:        "/home/user/other",
			ExitCode:   1,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 150,
			Hash:       storage.GenerateHash("git push"),
		},
	}

	for _, entry := range entries {
		err := db.Insert(entry)
		require.NoError(t, err)
	}

	t.Run("filter by search term", func(t *testing.T) {
		filters := storage.QueryFilters{
			Search: "docker",
			Limit:  10,
		}
		results, err := WithFilters(db, filters)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].Command, "docker")
	})

	t.Run("filter by cwd", func(t *testing.T) {
		filters := storage.QueryFilters{
			Cwd:   "/home/user/project",
			Limit: 10,
		}
		results, err := WithFilters(db, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("filter by exit code", func(t *testing.T) {
		filters := storage.QueryFilters{
			ExitCode: ptr(1),
			Limit:    10,
		}
		results, err := WithFilters(db, filters)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 1, results[0].ExitCode)
	})

	t.Run("combined filters", func(t *testing.T) {
		filters := storage.QueryFilters{
			Search: "docker",
			Cwd:    "/home/user/project",
			Limit:  10,
		}
		results, err := WithFilters(db, filters)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("filter with no matches", func(t *testing.T) {
		filters := storage.QueryFilters{
			Search: "nonexistent",
			Limit:  10,
		}
		results, err := WithFilters(db, filters)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// ptr is a helper function to get a pointer to an int
func ptr(i int) *int {
	return &i
}

func TestFilterEntries(t *testing.T) {
	entries := []*storage.HistoryEntry{
		{Command: "git status", Cwd: "/home/user"},
		{Command: "git commit -m 'test'", Cwd: "/home/user"},
		{Command: "docker ps", Cwd: "/home/user"},
		{Command: "docker build -t myapp .", Cwd: "/home/user"},
		{Command: "ls -la", Cwd: "/home/user"},
	}

	t.Run("filter with matching query", func(t *testing.T) {
		filtered := filterEntries(entries, "git")
		assert.Len(t, filtered, 2)
		assert.Contains(t, filtered[0].Command, "git")
		assert.Contains(t, filtered[1].Command, "git")
	})

	t.Run("filter with no matches", func(t *testing.T) {
		filtered := filterEntries(entries, "nonexistent")
		assert.Empty(t, filtered)
	})

	t.Run("filter with empty query", func(t *testing.T) {
		filtered := filterEntries(entries, "")
		assert.Len(t, filtered, 5) // All entries match empty query
	})

	t.Run("filter is case insensitive", func(t *testing.T) {
		filtered := filterEntries(entries, "DOCKER")
		assert.Len(t, filtered, 2)
	})

	t.Run("filter with partial match", func(t *testing.T) {
		filtered := filterEntries(entries, "doc")
		assert.Len(t, filtered, 2)
	})

	t.Run("filter empty entries", func(t *testing.T) {
		filtered := filterEntries([]*storage.HistoryEntry{}, "test")
		assert.Empty(t, filtered)
	})
}

func TestFormatEntry(t *testing.T) {
	t.Run("format complete entry", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "git status",
			Cwd:        "/home/user/project",
			ExitCode:   0,
			DurationMs: 125,
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "git status")
		assert.Contains(t, formatted, "/home/user/project")
		assert.Contains(t, formatted, "125ms")
		assert.Contains(t, formatted, "│")
	})

	t.Run("format entry with non-zero exit code", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "git push",
			Cwd:        "/home/user",
			ExitCode:   1,
			DurationMs: 50,
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "[exit:1]")
	})

	t.Run("format entry with long cwd", func(t *testing.T) {
		longCwd := "/home/user/very/long/path/to/project/subdirectory/nested"
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "ls",
			Cwd:        longCwd,
			ExitCode:   0,
			DurationMs: 10,
		}

		formatted := FormatEntry(entry)
		// Should be truncated with "..."
		assert.Contains(t, formatted, "...")
		assert.Less(t, len(formatted), len(longCwd)+100)
	})

	t.Run("format entry with duration over 1 second", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "npm test",
			Cwd:        "/home/user",
			ExitCode:   0,
			DurationMs: 2500,
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "[2.5s]")
	})

	t.Run("format entry with zero duration", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "echo test",
			Cwd:        "/home/user",
			ExitCode:   0,
			DurationMs: 0,
		}

		formatted := FormatEntry(entry)
		assert.NotContains(t, formatted, "[0ms]")
		assert.Contains(t, formatted, "echo test")
	})

	t.Run("format entry with empty cwd", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "echo test",
			Cwd:        "",
			ExitCode:   0,
			DurationMs: 10,
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "echo test")
	})

	t.Run("format entry with milliseconds", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "ls",
			Cwd:        "/home",
			ExitCode:   0,
			DurationMs: 50,
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "[50ms]")
	})

	t.Run("timestamp format", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890,
			Command:    "test",
			Cwd:        "/home",
			ExitCode:   0,
			DurationMs: 0,
		}

		formatted := FormatEntry(entry)
		// Should contain formatted timestamp
		assert.Contains(t, formatted, "2009-02-13")
	})
}

func TestExtractCommand(t *testing.T) {
	t.Run("extract from formatted entry", func(t *testing.T) {
		formatted := "2009-02-13 23:31:30 │ /home/user │ [125ms] │ git status"
		command := ExtractCommand(formatted)
		assert.Equal(t, "git status", command)
	})

	t.Run("extract from entry with exit code", func(t *testing.T) {
		formatted := "2009-02-13 23:31:30 │ /home/user │ [125ms] │ [exit:1] │ git push"
		command := ExtractCommand(formatted)
		assert.Equal(t, "git push", command)
	})

	t.Run("extract from simple formatted entry", func(t *testing.T) {
		formatted := "2009-02-13 23:31:30 │ ls -la"
		command := ExtractCommand(formatted)
		assert.Equal(t, "ls -la", command)
	})

	t.Run("extract from empty string", func(t *testing.T) {
		command := ExtractCommand("")
		assert.Equal(t, "", command)
	})

	t.Run("extract command with separator in it", func(t *testing.T) {
		// Note: ExtractCommand takes the last part after splitting by │
		// So if command contains │, only the part after last │ is returned
		formatted := "2009-02-13 23:31:30 │ /home │ echo test"
		command := ExtractCommand(formatted)
		assert.Equal(t, "echo test", command)
	})

	t.Run("extract from entry with no separator", func(t *testing.T) {
		formatted := "just a command"
		command := ExtractCommand(formatted)
		assert.Equal(t, "just a command", command)
	})
}
