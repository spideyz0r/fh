package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/spideyz0r/fh/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportText(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	db, err := storage.Open(tempDir + "/test.db")
	require.NoError(t, err)
	defer db.Close()

	// Insert test entries
	entries := []string{"echo test1", "ls -la", "git status"}
	for i, cmd := range entries {
		entry := &storage.HistoryEntry{
			Command:    cmd,
			Timestamp:  time.Now().Unix() + int64(i),
			ExitCode:   0,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext(cmd, "/tmp"+string(rune(i))),
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	// Export to text
	var buf bytes.Buffer
	opts := Options{
		Format: FormatText,
	}

	err = Export(db, &buf, opts)
	require.NoError(t, err)

	// Verify output
	output := buf.String()
	for _, cmd := range entries {
		assert.Contains(t, output, cmd)
	}

	// Count lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3)
}

func TestExportJSON(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	db, err := storage.Open(tempDir + "/test.db")
	require.NoError(t, err)
	defer db.Close()

	// Insert test entry
	entry := &storage.HistoryEntry{
		Command:    "echo test",
		Timestamp:  1234567890,
		ExitCode:   0,
		Cwd:        "/tmp",
		Hostname:   "testhost",
		User:       "testuser",
		Shell:      "bash",
		DurationMs: 150,
		GitBranch:  "main",
		SessionID:  "session123",
		Hash:       storage.GenerateHash("echo test"),
	}
	err = db.Insert(entry)
	require.NoError(t, err)

	// Export to JSON
	var buf bytes.Buffer
	opts := Options{
		Format: FormatJSON,
	}

	err = Export(db, &buf, opts)
	require.NoError(t, err)

	// Parse JSON
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Verify
	require.Len(t, result, 1)
	assert.Equal(t, "echo test", result[0]["command"])
	assert.Equal(t, float64(1234567890), result[0]["timestamp"])
	assert.Equal(t, float64(0), result[0]["exit_code"])
	assert.Equal(t, "/tmp", result[0]["cwd"])
	assert.Equal(t, "testhost", result[0]["hostname"])
	assert.Equal(t, "testuser", result[0]["user"])
	assert.Equal(t, "bash", result[0]["shell"])
	assert.Equal(t, float64(150), result[0]["duration_ms"])
	assert.Equal(t, "main", result[0]["git_branch"])
	assert.Equal(t, "session123", result[0]["session_id"])
}

func TestExportCSV(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	db, err := storage.Open(tempDir + "/test.db")
	require.NoError(t, err)
	defer db.Close()

	// Insert test entries
	commands := []string{"echo test", "ls -la"}
	for i, cmd := range commands {
		entry := &storage.HistoryEntry{
			Command:    cmd,
			Timestamp:  time.Now().Unix() + int64(i),
			ExitCode:   i,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100 * int64(i+1),
			Hash:       storage.GenerateHashWithContext(cmd, "/tmp"+string(rune(i))),
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	// Export to CSV
	var buf bytes.Buffer
	opts := Options{
		Format: FormatCSV,
	}

	err = Export(db, &buf, opts)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Verify header
	assert.Len(t, records, 3) // header + 2 entries
	header := records[0]
	assert.Equal(t, "id", header[0])
	assert.Equal(t, "timestamp", header[1])
	assert.Equal(t, "command", header[2])
	assert.Equal(t, "exit_code", header[3])

	// Verify data
	for i := 1; i < len(records); i++ {
		record := records[i]
		assert.NotEmpty(t, record[0])                                  // id
		assert.NotEmpty(t, record[1])                                  // timestamp
		assert.Contains(t, []string{"echo test", "ls -la"}, record[2]) // command
	}
}

func TestExportWithFilters(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	db, err := storage.Open(tempDir + "/test.db")
	require.NoError(t, err)
	defer db.Close()

	// Insert test entries
	commands := []string{"echo test", "ls -la", "echo hello", "git status"}
	for i, cmd := range commands {
		entry := &storage.HistoryEntry{
			Command:    cmd,
			Timestamp:  time.Now().Unix() + int64(i),
			ExitCode:   0,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext(cmd, "/tmp"+string(rune(i))),
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	// Export with search filter
	var buf bytes.Buffer
	opts := Options{
		Format: FormatText,
		Filters: storage.QueryFilters{
			Search: "echo",
		},
	}

	err = Export(db, &buf, opts)
	require.NoError(t, err)

	// Verify only "echo" commands are exported
	output := buf.String()
	assert.Contains(t, output, "echo test")
	assert.Contains(t, output, "echo hello")
	assert.NotContains(t, output, "ls -la")
	assert.NotContains(t, output, "git status")

	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 2)
}

func TestExportWithLimit(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	db, err := storage.Open(tempDir + "/test.db")
	require.NoError(t, err)
	defer db.Close()

	// Insert 10 test entries
	for i := 0; i < 10; i++ {
		entry := &storage.HistoryEntry{
			Command:    "echo " + string(rune('0'+i)),
			Timestamp:  time.Now().Unix() + int64(i),
			ExitCode:   0,
			Cwd:        "/tmp",
			Hostname:   "testhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 100,
			Hash:       storage.GenerateHashWithContext("echo", "/tmp"+string(rune(i))),
		}
		err = db.Insert(entry)
		require.NoError(t, err)
	}

	// Export with limit
	var buf bytes.Buffer
	opts := Options{
		Format: FormatText,
		Filters: storage.QueryFilters{
			Limit: 3,
		},
	}

	err = Export(db, &buf, opts)
	require.NoError(t, err)

	// Verify only 3 entries exported
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3)
}

func TestExportEmpty(t *testing.T) {
	tempDir, cleanup := testutil.TempDir(t)
	defer cleanup()

	db, err := storage.Open(tempDir + "/test.db")
	require.NoError(t, err)
	defer db.Close()

	// Export empty database
	var buf bytes.Buffer
	opts := Options{
		Format: FormatText,
	}

	err = Export(db, &buf, opts)
	require.NoError(t, err)

	// Should be empty
	assert.Empty(t, buf.String())
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		wantErr  bool
	}{
		{"text", FormatText, false},
		{"txt", FormatText, false},
		{"json", FormatJSON, false},
		{"csv", FormatCSV, false},
		{"XML", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			format, err := ParseFormat(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, format)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	// Test timestamp formatting
	ts := int64(1234567890) // 2009-02-13 23:31:30 UTC
	formatted := formatTimestamp(ts)

	// Should be in RFC3339 format
	assert.Contains(t, formatted, "2009-02-13")
	assert.Contains(t, formatted, "T")

	// Should be parseable
	parsed, err := time.Parse(time.RFC3339, formatted)
	require.NoError(t, err)
	assert.Equal(t, ts, parsed.Unix())
}
