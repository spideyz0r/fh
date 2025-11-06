package stats

import (
	"testing"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/spideyz0r/fh/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollect_EmptyDatabase(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	dbPath := tempDir + "/test.db"
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	stats, err := Collect(db)
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.TotalCommands)
	assert.Equal(t, int64(0), stats.UniqueCommands)
	assert.Equal(t, 0.0, stats.SuccessRate)
	assert.Equal(t, 0.0, stats.AvgPerDay)
	assert.Empty(t, stats.TopCommands)
	assert.Empty(t, stats.CommandsByDir)
}

func TestCollect_SingleCommand(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	dbPath := tempDir + "/test.db"
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert a command
	entry := &storage.HistoryEntry{
		Command:    "echo test",
		Timestamp:  time.Now().Unix(),
		ExitCode:   0,
		Cwd:        "/tmp",
		Hostname:   "testhost",
		User:       "testuser",
		Shell:      "bash",
		DurationMs: 100,
		Hash:       storage.GenerateHash("echo test"),
	}
	err = db.Insert(entry)
	require.NoError(t, err)

	stats, err := Collect(db)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.TotalCommands)
	assert.Equal(t, int64(1), stats.UniqueCommands)
	assert.Equal(t, 100.0, stats.SuccessRate) // 1/1 = 100%
	assert.Len(t, stats.TopCommands, 1)
	assert.Equal(t, "echo test", stats.TopCommands[0].Command)
	assert.Equal(t, 1, stats.TopCommands[0].Count)
}

func TestCollect_MultipleCommands(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	dbPath := tempDir + "/test.db"
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert multiple commands
	baseTime := time.Now().Unix()
	commands := []struct {
		cmd      string
		exitCode int
		cwd      string
	}{
		{"echo test", 0, "/tmp"},
		{"ls -la", 0, "/tmp"},
		{"echo test", 0, "/home"},     // Duplicate command
		{"git status", 1, "/tmp"},     // Failed command
		{"echo hello", 0, "/home"},
		{"echo test", 0, "/tmp"},      // Another duplicate
	}

	for i, cmd := range commands {
		entry := &storage.HistoryEntry{
			Command:    cmd.cmd,
			Timestamp:  baseTime + int64(i),
			ExitCode:   cmd.exitCode,
			Cwd:        cmd.cwd,
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext(cmd.cmd, cmd.cwd+string(rune(i))), // Make each entry truly unique
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	stats, err := Collect(db)
	require.NoError(t, err)

	assert.Equal(t, int64(6), stats.TotalCommands)
	assert.Equal(t, int64(4), stats.UniqueCommands) // echo test, ls -la, git status, echo hello
	assert.Equal(t, 83.3, roundToOneDecimal(stats.SuccessRate)) // 5/6 = 83.33%

	// Check top commands (should be sorted by count)
	require.GreaterOrEqual(t, len(stats.TopCommands), 1)
	assert.Equal(t, "echo test", stats.TopCommands[0].Command) // Appears 3 times
	assert.Equal(t, 3, stats.TopCommands[0].Count)

	// Check directories
	require.Len(t, stats.CommandsByDir, 2)
	// Should be sorted by count
	assert.Equal(t, "/tmp", stats.CommandsByDir[0].Directory) // 4 commands
	assert.Equal(t, 4, stats.CommandsByDir[0].Count)
	assert.Equal(t, "/home", stats.CommandsByDir[1].Directory) // 2 commands
	assert.Equal(t, 2, stats.CommandsByDir[1].Count)
}

func TestCollect_TimeDistribution(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	dbPath := tempDir + "/test.db"
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert commands at different hours
	now := time.Now()
	hours := []int{9, 9, 10, 10, 10, 14, 14, 18, 22}

	for i, hour := range hours {
		timestamp := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()).Unix()
		entry := &storage.HistoryEntry{
			Command:    "echo test",
			Timestamp:  timestamp,
			ExitCode:   0,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext("echo test", "/tmp"+string(rune(i))), // Make each entry unique
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	stats, err := Collect(db)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.TimeDistribution[9])  // 9am: 2 commands
	assert.Equal(t, 3, stats.TimeDistribution[10]) // 10am: 3 commands
	assert.Equal(t, 2, stats.TimeDistribution[14]) // 2pm: 2 commands
	assert.Equal(t, 1, stats.TimeDistribution[18]) // 6pm: 1 command
	assert.Equal(t, 1, stats.TimeDistribution[22]) // 10pm: 1 command
	assert.Equal(t, 0, stats.TimeDistribution[12]) // 12pm: 0 commands
}

func TestCollect_AveragePerDay(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	dbPath := tempDir + "/test.db"
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert commands over multiple days
	now := time.Now()
	for i := 0; i < 10; i++ {
		// 10 commands spread over 5 days = 2 per day average
		timestamp := now.Add(-time.Duration(i/2) * 24 * time.Hour).Unix()
		entry := &storage.HistoryEntry{
			Command:    "echo test",
			Timestamp:  timestamp,
			ExitCode:   0,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext("echo test", "/tmp"+string(rune(i))), // Make each entry unique
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	stats, err := Collect(db)
	require.NoError(t, err)

	assert.Equal(t, int64(10), stats.TotalCommands)
	// Average should be around 2 commands per day (10 commands / 5 days)
	assert.GreaterOrEqual(t, stats.AvgPerDay, 1.5)
	assert.LessOrEqual(t, stats.AvgPerDay, 2.5)
}

func TestFormat_EmptyStats(t *testing.T) {
	stats := &Stats{
		TimeDistribution: make(map[int]int),
	}

	output := stats.Format(10)
	assert.Contains(t, output, "No commands in history yet")
}

func TestFormat_WithData(t *testing.T) {
	stats := &Stats{
		TotalCommands:  100,
		UniqueCommands: 50,
		SuccessRate:    95.5,
		AvgPerDay:      10.5,
		FirstCommand:   time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
		LastCommand:    time.Date(2025, 1, 10, 15, 0, 0, 0, time.UTC),
		TopCommands: []CommandCount{
			{Command: "ls", Count: 30},
			{Command: "git status", Count: 20},
			{Command: "echo test", Count: 15},
		},
		CommandsByDir: []DirectoryCount{
			{Directory: "/tmp", Count: 50},
			{Directory: "/home", Count: 30},
		},
		TimeDistribution: map[int]int{
			9:  10,
			10: 20,
			14: 15,
		},
	}

	output := stats.Format(3)

	// Verify key sections exist
	assert.Contains(t, output, "fh - History Statistics")
	assert.Contains(t, output, "Total Commands:   100")
	assert.Contains(t, output, "Unique Commands:  50")
	assert.Contains(t, output, "Success Rate:     95.5%")
	assert.Contains(t, output, "Avg Per Day:      10.5")

	// Verify top commands
	assert.Contains(t, output, "Top 3 Commands:")
	assert.Contains(t, output, "ls")
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "echo test")

	// Verify directories
	assert.Contains(t, output, "Top 2 Directories:")
	assert.Contains(t, output, "/tmp")
	assert.Contains(t, output, "/home")

	// Verify hour distribution
	assert.Contains(t, output, "Commands by Hour:")
}

func TestCollectFiltered(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	dbPath := tempDir + "/test.db"
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert commands with different exit codes
	baseTime := time.Now().Unix()
	for i := 0; i < 10; i++ {
		exitCode := 0
		if i%3 == 0 {
			exitCode = 1 // Every 3rd command fails
		}

		entry := &storage.HistoryEntry{
			Command:    "echo test",
			Timestamp:  baseTime + int64(i),
			ExitCode:   exitCode,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext("echo test", "/tmp"+string(rune(i))), // Make each entry unique
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	// Collect stats filtered by successful commands only
	exitCodeZero := 0
	stats, err := CollectFiltered(db, storage.QueryFilters{
		ExitCode: &exitCodeZero,
	})
	require.NoError(t, err)

	// Should only count successful commands
	// i%3==0 fails: 0, 3, 6, 9 = 4 failures, so 10 - 4 = 6 successful
	assert.Equal(t, int64(6), stats.TotalCommands)
	assert.Equal(t, 100.0, stats.SuccessRate)     // All filtered commands are successful
}

// Helper function to round to one decimal place
func roundToOneDecimal(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
