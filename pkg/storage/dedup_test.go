package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple command",
			command:  "ls -la",
			expected: GenerateHash("ls -la"),
		},
		{
			name:     "command with whitespace",
			command:  "  ls -la  ",
			expected: GenerateHash("ls -la"), // Should trim
		},
		{
			name:     "same command produces same hash",
			command:  "git status",
			expected: GenerateHash("git status"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := GenerateHash(tt.command)
			assert.NotEmpty(t, hash)
			assert.Len(t, hash, 64) // SHA256 hex is 64 chars
			assert.Equal(t, tt.expected, hash)
		})
	}
}

func TestGenerateHash_Consistency(t *testing.T) {
	command := "test command"

	hash1 := GenerateHash(command)
	hash2 := GenerateHash(command)

	assert.Equal(t, hash1, hash2, "Same command should produce same hash")
}

func TestGenerateHashWithContext(t *testing.T) {
	hash1 := GenerateHashWithContext("ls", "/home/user")
	hash2 := GenerateHashWithContext("ls", "/tmp")

	// Same command in different directories should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

func TestInsertWithDedup_Disabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DedupConfig{Enabled: false}

	entry1 := createTestEntry(t, "ls -la", 1000)
	entry2 := createTestEntry(t, "ls -la", 2000)

	err := db.InsertWithDedup(entry1, config)
	require.NoError(t, err)

	// Should fail on duplicate hash when dedup disabled
	err = db.InsertWithDedup(entry2, config)
	assert.Error(t, err, "Should fail on duplicate hash")
}

func TestInsertWithDedup_KeepFirst(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DedupConfig{
		Enabled:  true,
		Strategy: KeepFirst,
	}

	entry1 := createTestEntry(t, "ls -la", 1000)
	entry2 := createTestEntry(t, "ls -la", 2000) // Same command, later timestamp

	err := db.InsertWithDedup(entry1, config)
	require.NoError(t, err)

	// Second insert should be ignored
	err = db.InsertWithDedup(entry2, config)
	require.NoError(t, err)

	// Should only have one entry
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Should have the first timestamp
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(1000), results[0].Timestamp)
}

func TestInsertWithDedup_KeepLast(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DedupConfig{
		Enabled:  true,
		Strategy: KeepLast,
	}

	entry1 := createTestEntry(t, "ls -la", 1000)
	entry2 := createTestEntry(t, "ls -la", 2000) // Same command, later timestamp

	err := db.InsertWithDedup(entry1, config)
	require.NoError(t, err)

	// Second insert should update timestamp
	err = db.InsertWithDedup(entry2, config)
	require.NoError(t, err)

	// Should only have one entry
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Should have the latest timestamp
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(2000), results[0].Timestamp)
}

func TestInsertWithDedup_KeepAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DedupConfig{
		Enabled:  true,
		Strategy: KeepAll,
	}

	// Insert same command multiple times with different contexts
	timestamps := []int64{1000, 2000, 3000}
	cwds := []string{"/home/user", "/tmp", "/var/log"}
	exitCodes := []int{0, 1, 0}

	for i, ts := range timestamps {
		entry := createTestEntry(t, "ls -la", ts)
		entry.Cwd = cwds[i]
		entry.ExitCode = exitCodes[i]

		err := db.InsertWithDedup(entry, config)
		require.NoError(t, err)
	}

	// Should have all three entries (no deduplication)
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify each entry has different context
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// All should be the same command but different contexts
	for _, r := range results {
		assert.Equal(t, "ls -la", r.Command)
	}

	// Different working directories
	cwdMap := make(map[string]bool)
	for _, r := range results {
		cwdMap[r.Cwd] = true
	}
	assert.Len(t, cwdMap, 3, "Should have 3 different working directories")
}

func TestInsertWithDedup_KeepAll_PreservesContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DedupConfig{
		Enabled:  true,
		Strategy: KeepAll,
	}

	// Simulate running the same command multiple times in different contexts
	// This is valuable for AI to understand patterns
	entry1 := createTestEntry(t, "git status", 1000)
	entry1.Cwd = "/home/user/project1"
	entry1.ExitCode = 0

	entry2 := createTestEntry(t, "git status", 2000)
	entry2.Cwd = "/home/user/project2"
	entry2.ExitCode = 1 // Failed

	entry3 := createTestEntry(t, "git status", 3000)
	entry3.Cwd = "/home/user/project1"
	entry3.ExitCode = 0

	require.NoError(t, db.InsertWithDedup(entry1, config))
	require.NoError(t, db.InsertWithDedup(entry2, config))
	require.NoError(t, db.InsertWithDedup(entry3, config))

	// All three should be stored
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// AI can now see the pattern: git status was run 3 times,
	// succeeded in project1 twice, failed in project2
	failedCommands := 0
	for _, r := range results {
		if r.ExitCode != 0 {
			failedCommands++
			assert.Equal(t, "/home/user/project2", r.Cwd)
		}
	}
	assert.Equal(t, 1, failedCommands)
}

func TestCheckHashExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	entry := createTestEntry(t, "test", time.Now().Unix())
	entry.Hash = GenerateHash("test") // Explicitly set hash
	require.NoError(t, db.Insert(entry))

	hash := GenerateHash("test")

	exists, id, err := db.checkHashExists(hash)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NotZero(t, id)

	// Non-existent hash
	exists, _, err = db.checkHashExists("nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestGetDuplicates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Manually insert duplicates by disabling the UNIQUE constraint
	// We'll drop and recreate the table without the constraint for this test
	_, err := db.conn.Exec(`DROP TABLE IF EXISTS history`)
	require.NoError(t, err)

	_, err = db.conn.Exec(`
		CREATE TABLE history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			command TEXT NOT NULL,
			cwd TEXT,
			exit_code INTEGER,
			hostname TEXT,
			user TEXT,
			shell TEXT,
			duration_ms INTEGER,
			git_branch TEXT,
			hash TEXT,
			session_id TEXT,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		)
	`)
	require.NoError(t, err)

	hash1 := GenerateHash("ls -la")

	// Insert three "ls -la" with same hash
	entry1 := createTestEntry(t, "ls -la", 1000)
	entry1.Hash = hash1
	_, err = db.conn.Exec(`
		INSERT INTO history (timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry1.Timestamp, entry1.Command, entry1.Cwd, entry1.ExitCode, entry1.Hostname, entry1.User, entry1.Shell, entry1.DurationMs, entry1.GitBranch, entry1.Hash, entry1.SessionID)
	require.NoError(t, err)

	entry2 := createTestEntry(t, "ls -la", 2000)
	entry2.Hash = hash1
	entry2.SessionID = "session-456"
	_, err = db.conn.Exec(`
		INSERT INTO history (timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry2.Timestamp, entry2.Command, entry2.Cwd, entry2.ExitCode, entry2.Hostname, entry2.User, entry2.Shell, entry2.DurationMs, entry2.GitBranch, entry2.Hash, entry2.SessionID)
	require.NoError(t, err)

	entry3 := createTestEntry(t, "ls -la", 4000)
	entry3.Hash = hash1
	entry3.SessionID = "session-789"
	_, err = db.conn.Exec(`
		INSERT INTO history (timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry3.Timestamp, entry3.Command, entry3.Cwd, entry3.ExitCode, entry3.Hostname, entry3.User, entry3.Shell, entry3.DurationMs, entry3.GitBranch, entry3.Hash, entry3.SessionID)
	require.NoError(t, err)

	// Get duplicates
	dups, err := db.GetDuplicates()
	require.NoError(t, err)

	// Should have 3 entries (all "ls -la" instances)
	assert.Len(t, dups, 3)
	for _, d := range dups {
		assert.Equal(t, "ls -la", d.Command)
	}
}

func TestDeduplicateExisting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Manually insert duplicates by disabling the UNIQUE constraint
	// We'll drop and recreate the table without the constraint for this test
	_, err := db.conn.Exec(`DROP TABLE IF EXISTS history`)
	require.NoError(t, err)

	_, err = db.conn.Exec(`
		CREATE TABLE history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			command TEXT NOT NULL,
			cwd TEXT,
			exit_code INTEGER,
			hostname TEXT,
			user TEXT,
			shell TEXT,
			duration_ms INTEGER,
			git_branch TEXT,
			hash TEXT,
			session_id TEXT,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		)
	`)
	require.NoError(t, err)

	hash1 := GenerateHash("ls -la")
	hash2 := GenerateHash("pwd")

	// Insert three "ls -la" with same hash
	for i, ts := range []int64{1000, 2000, 3000} {
		entry := createTestEntry(t, "ls -la", ts)
		entry.Hash = hash1
		entry.SessionID = fmt.Sprintf("session-%d", i)
		_, err := db.conn.Exec(`
			INSERT INTO history (timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, entry.Timestamp, entry.Command, entry.Cwd, entry.ExitCode, entry.Hostname, entry.User, entry.Shell, entry.DurationMs, entry.GitBranch, entry.Hash, entry.SessionID)
		require.NoError(t, err)
	}

	// Insert one "pwd"
	entry := createTestEntry(t, "pwd", 4000)
	entry.Hash = hash2
	_, err = db.conn.Exec(`
		INSERT INTO history (timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.Timestamp, entry.Command, entry.Cwd, entry.ExitCode, entry.Hostname, entry.User, entry.Shell, entry.DurationMs, entry.GitBranch, entry.Hash, entry.SessionID)
	require.NoError(t, err)

	// Should have 4 entries
	count, err := db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(4), count)

	// Deduplicate
	removed, err := db.DeduplicateExisting()
	require.NoError(t, err)
	assert.Equal(t, int64(2), removed) // Should remove 2 duplicates

	// Should have 2 entries left (latest "ls -la" and "pwd")
	count, err = db.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify we kept the most recent "ls -la"
	results, err := db.Query(QueryFilters{Search: "ls"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int64(3000), results[0].Timestamp)
}

func TestInsertWithDedup_AutoGeneratesHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DedupConfig{Enabled: true, Strategy: KeepFirst}

	entry := createTestEntry(t, "ls -la", 1000)
	entry.Hash = "" // Empty hash

	err := db.InsertWithDedup(entry, config)
	require.NoError(t, err)

	// Verify hash was generated
	results, err := db.Query(QueryFilters{})
	require.NoError(t, err)
	assert.NotEmpty(t, results[0].Hash)
	assert.Equal(t, GenerateHash("ls -la"), results[0].Hash)
}
