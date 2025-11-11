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
