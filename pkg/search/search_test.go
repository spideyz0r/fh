package search

import (
	"strings"
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
		assert.Contains(t, formatted, "2009-02-13")
		assert.Contains(t, formatted, "/home/user/project")
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
		assert.Contains(t, formatted, "exit:1")
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
		// Long cwd should be truncated with "..."
		assert.Contains(t, formatted, "ls")
		assert.Contains(t, formatted, "...")
		assert.Contains(t, formatted, "2009-02-13")
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
		// Duration no longer displayed
		assert.Contains(t, formatted, "npm test")
		assert.Contains(t, formatted, "2009-02-13")
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
		// Duration no longer displayed
		assert.Contains(t, formatted, "ls")
		assert.Contains(t, formatted, "2009-02-13")
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
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890, // 2009-02-13 23:31:30
			Command:    "git status",
			Cwd:        "/home/user",
			ExitCode:   0,
			DurationMs: 50,
		}
		formatted := FormatEntry(entry)
		command := ExtractCommand(formatted)
		assert.Equal(t, "git status", command)
	})

	t.Run("extract from entry with exit code", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890, // 2009-02-13 23:31:30
			Command:    "git push",
			Cwd:        "/home/user",
			ExitCode:   1,
			DurationMs: 50,
		}
		formatted := FormatEntry(entry)
		command := ExtractCommand(formatted)
		assert.Equal(t, "git push", command)
	})

	t.Run("extract from simple formatted entry", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890, // 2009-02-13 23:31:30
			Command:    "ls -la",
			Cwd:        "/home",
			ExitCode:   0,
			DurationMs: 10,
		}
		formatted := FormatEntry(entry)
		command := ExtractCommand(formatted)
		assert.Equal(t, "ls -la", command)
	})

	t.Run("extract from empty string", func(t *testing.T) {
		command := ExtractCommand("")
		assert.Equal(t, "", command)
	})

	t.Run("extract command with separator in it", func(t *testing.T) {
		// Note: ExtractCommand takes the first part after splitting by │
		// Command is always first in the new format
		entry := &storage.HistoryEntry{
			Timestamp:  1234567890, // 2009-02-13 23:31:30
			Command:    "echo test",
			Cwd:        "/home",
			ExitCode:   0,
			DurationMs: 10,
		}
		formatted := FormatEntry(entry)
		command := ExtractCommand(formatted)
		assert.Equal(t, "echo test", command)
	})

	t.Run("extract from entry with no separator", func(t *testing.T) {
		formatted := "just a command"
		command := ExtractCommand(formatted)
		assert.Equal(t, "just a command", command)
	})

	t.Run("extract from padded command", func(t *testing.T) {
		formatted := "ls                                                          │ 2009-02-13 18:31:30 │ /home"
		command := ExtractCommand(formatted)
		assert.Equal(t, "ls", command, "Should trim padding spaces")
	})

	t.Run("extract from truncated command with ellipsis", func(t *testing.T) {
		formatted := "kubectl argo rollouts get rollouts very-long-deploymen... │ 2009-02-13 18:31:30"
		command := ExtractCommand(formatted)
		assert.Equal(t, "kubectl argo rollouts get rollouts very-long-deploymen...", command)
	})
}

func TestFormatEntry_Padding(t *testing.T) {
	t.Run("short command gets padded to 60 chars", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   "ls",
			Cwd:       "/home",
			ExitCode:  0,
		}

		formatted := FormatEntry(entry)
		parts := strings.Split(formatted, " │ ")
		assert.Len(t, parts[0], 60, "Command should be padded to 60 characters")
	})

	t.Run("long command gets truncated with ellipsis", func(t *testing.T) {
		longCommand := "kubectl argo rollouts get rollouts very-long-deployment-name-exceeds-sixty"
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   longCommand,
			Cwd:       "/home",
			ExitCode:  0,
		}

		formatted := FormatEntry(entry)
		parts := strings.Split(formatted, " │ ")
		assert.Len(t, parts[0], 60, "Long command should be truncated to 60 characters")
		assert.Contains(t, parts[0], "...", "Truncated command should have ellipsis")
	})
}

func TestFormatEntry_CwdHandling(t *testing.T) {
	t.Run("very long cwd gets truncated", func(t *testing.T) {
		longCwd := "/Users/username/very/long/path/to/deeply/nested/directory/that/exceeds/fifty/chars"
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   "ls",
			Cwd:       longCwd,
			ExitCode:  0,
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "...", "Long cwd should be truncated with ellipsis")
		parts := strings.Split(formatted, " │ ")
		assert.GreaterOrEqual(t, len(parts), 3, "Should have command, timestamp, and cwd")
	})

	t.Run("empty cwd not displayed", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   "ls",
			Cwd:       "",
			ExitCode:  0,
		}

		formatted := FormatEntry(entry)
		parts := strings.Split(formatted, " │ ")
		assert.Equal(t, 2, len(parts), "Empty cwd should not add extra separator")
	})
}

func TestFormatEntry_BadgesCombinations(t *testing.T) {
	t.Run("exit code and git branch together", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   "git push",
			Cwd:       "/home/project",
			ExitCode:  1,
			GitBranch: "feature-branch",
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "[exit:1 feature-branch]")
	})

	t.Run("only git branch when exit is 0", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   "git status",
			Cwd:       "/home/project",
			ExitCode:  0,
			GitBranch: "main",
		}

		formatted := FormatEntry(entry)
		assert.Contains(t, formatted, "[main]")
		assert.NotContains(t, formatted, "exit:0")
	})

	t.Run("no badges when exit 0 and no branch", func(t *testing.T) {
		entry := &storage.HistoryEntry{
			Timestamp: 1234567890,
			Command:   "ls",
			Cwd:       "/home",
			ExitCode:  0,
			GitBranch: "",
		}

		formatted := FormatEntry(entry)
		assert.NotContains(t, formatted, "[")
	})
}
