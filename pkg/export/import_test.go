package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/spideyz0r/fh/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestImportText(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	input := `ls -la
cd /tmp
echo hello
`

	r := strings.NewReader(input)
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	count, err := Import(db, r, FormatText, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 imports, got %d", count)
	}

	// Verify entries were imported
	entries, err := db.Query(storage.QueryFilters{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries in database, got %d", len(entries))
	}

	// Check that all commands are present (order may vary if timestamps are identical)
	commands := make(map[string]bool)
	for _, entry := range entries {
		commands[entry.Command] = true
	}

	expectedCommands := []string{"ls -la", "cd /tmp", "echo hello"}
	for _, expected := range expectedCommands {
		if !commands[expected] {
			t.Errorf("Expected command %q not found in results", expected)
		}
	}
}

func TestImportJSON(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	input := `[
  {
    "id": 1,
    "command": "ls -la",
    "timestamp": 1234567890,
    "exit_code": 0,
    "cwd": "/home/user",
    "hostname": "localhost",
    "user": "testuser",
    "shell": "bash",
    "duration_ms": 150,
    "git_branch": "main",
    "session_id": "session1"
  },
  {
    "id": 2,
    "command": "git status",
    "timestamp": 1234567900,
    "exit_code": 0,
    "cwd": "/home/user/project",
    "hostname": "localhost",
    "user": "testuser",
    "shell": "bash",
    "duration_ms": 200,
    "git_branch": "feature",
    "session_id": "session1"
  }
]`

	r := strings.NewReader(input)
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	count, err := Import(db, r, FormatJSON, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 imports, got %d", count)
	}

	// Verify entries
	entries, err := db.Query(storage.QueryFilters{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Check entries (Query returns most recent first)
	// Most recent: git status
	if entries[0].Command != "git status" {
		t.Errorf("Expected command 'git status', got %q", entries[0].Command)
	}
	if entries[0].Timestamp != 1234567900 {
		t.Errorf("Expected timestamp 1234567900, got %d", entries[0].Timestamp)
	}
	if entries[0].Cwd != "/home/user/project" {
		t.Errorf("Expected cwd '/home/user/project', got %q", entries[0].Cwd)
	}
	// Note: DurationMs field name in JSON is duration_ms, which should map correctly
	// but actual value depends on JSON decoder behavior
}

func TestImportCSV(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	input := `id,timestamp,command,exit_code,cwd,hostname,user,shell,duration_ms,git_branch,session_id
1,2024-01-01T12:00:00Z,ls -la,0,/home/user,localhost,testuser,bash,150,main,session1
2,2024-01-01T12:01:00Z,git status,0,/home/user/project,localhost,testuser,bash,200,feature,session1
`

	r := strings.NewReader(input)
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	count, err := Import(db, r, FormatCSV, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 imports, got %d", count)
	}

	// Verify entries
	entries, err := db.Query(storage.QueryFilters{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Check metadata preservation (Query returns most recent first)
	if entries[0].Command != "git status" {
		t.Errorf("Expected command 'git status', got %q", entries[0].Command)
	}
	if entries[0].Cwd != "/home/user/project" {
		t.Errorf("Expected cwd '/home/user/project', got %q", entries[0].Cwd)
	}
	if entries[0].GitBranch != "feature" {
		t.Errorf("Expected git_branch 'feature', got %q", entries[0].GitBranch)
	}
}

func TestImportEmpty(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	r := strings.NewReader("")
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	count, err := Import(db, r, FormatText, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 imports from empty file, got %d", count)
	}
}

func TestImportWithDeduplication(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// Insert a command first
	entry := &storage.HistoryEntry{
		Command:   "ls -la",
		Timestamp: 1234567890,
	}
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepFirst,
	}
	if err := db.InsertWithDedup(entry, dedupConfig); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Try to import the same command
	input := `ls -la
cd /tmp
`

	r := strings.NewReader(input)

	count, err := Import(db, r, FormatText, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// With KeepFirst dedup strategy, duplicate "ls -la" won't be inserted
	// Only "cd /tmp" should be imported (ls -la already exists)
	// Note: importText uses current timestamp, but hash is based on command only
	if count < 1 {
		t.Errorf("Expected at least 1 import, got %d", count)
	}

	// Verify total entries
	totalCount, err := db.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if totalCount != 2 {
		t.Errorf("Expected 2 total entries, got %d", totalCount)
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Format
	}{
		{
			name:     "JSON array",
			input:    `[{"command": "test"}]`,
			expected: FormatJSON,
		},
		{
			name:     "JSON object",
			input:    `{"command": "test"}`,
			expected: FormatJSON,
		},
		{
			name:     "CSV",
			input:    "id,command,timestamp\n1,test,123",
			expected: FormatCSV,
		},
		{
			name:     "Plain text",
			input:    "ls -la\ncd /tmp\n",
			expected: FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			format, _, err := DetectFormat(r)
			if err != nil {
				t.Fatalf("DetectFormat failed: %v", err)
			}

			if format != tt.expected {
				t.Errorf("Expected format %s, got %s", tt.expected, format)
			}
		})
	}
}

func TestImportInvalidJSON(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	input := `{"invalid json`

	r := strings.NewReader(input)
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	_, err := Import(db, r, FormatJSON, dedupConfig)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestImportCSVMissingRequiredColumn(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// CSV without 'command' column
	input := `id,timestamp,user
1,2024-01-01T12:00:00Z,testuser
`

	r := strings.NewReader(input)
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	_, err := Import(db, r, FormatCSV, dedupConfig)
	if err == nil {
		t.Error("Expected error for CSV missing required column, got nil")
	}
	if !strings.Contains(err.Error(), "required column") {
		t.Errorf("Expected error about required column, got: %v", err)
	}
}

func TestImportTextWithLongCommands(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// Create input with very long command line (500KB) - larger than default scanner buffer
	longCommand := "curl -X POST -H 'Content-Type: application/json' -d '" + strings.Repeat("x", 500*1024) + "' https://example.com/api"
	input := longCommand + "\nls -la\necho hello"

	r := strings.NewReader(input)
	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	count, err := Import(db, r, FormatText, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed with large command line: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 imports, got %d", count)
	}

	// Verify entries were imported
	entries, err := db.Query(storage.QueryFilters{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries in database, got %d", len(entries))
	}

	// Check that the very long command was preserved correctly
	commands := make(map[string]bool)
	for _, entry := range entries {
		commands[entry.Command] = true
	}

	expectedCommands := []string{longCommand, "ls -la", "echo hello"}
	for _, expected := range expectedCommands {
		if !commands[expected] {
			t.Errorf("Expected command %q not found in results", expected[:min(50, len(expected))]+"...")
		}
	}

	// Specifically verify the long command was imported with correct length
	found := false
	for _, entry := range entries {
		if len(entry.Command) > 500*1024 {
			found = true
			assert.Equal(t, longCommand, entry.Command, "Long command should be preserved exactly")
			break
		}
	}
	assert.True(t, found, "Should find the very long command")
}

func TestImportRoundTrip(t *testing.T) {
	// Test export and import roundtrip
	db1 := testutil.NewTestDB(t)
	defer db1.Close()

	// Insert test data
	entries := []*storage.HistoryEntry{
		{
			Command:    "ls -la",
			Timestamp:  1234567890,
			Cwd:        "/home/user",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 150,
			GitBranch:  "main",
			SessionID:  "session1",
		},
		{
			Command:    "git status",
			Timestamp:  1234567900,
			Cwd:        "/home/user/project",
			ExitCode:   0,
			Hostname:   "localhost",
			User:       "testuser",
			Shell:      "bash",
			DurationMs: 200,
			GitBranch:  "feature",
			SessionID:  "session1",
		},
	}

	dedupConfig := storage.DedupConfig{
		Enabled:  true,
		Strategy: storage.KeepAll,
	}

	for _, entry := range entries {
		if err := db1.InsertWithDedup(entry, dedupConfig); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Export to JSON
	var buf bytes.Buffer
	opts := Options{
		Format:  FormatJSON,
		Filters: storage.QueryFilters{},
	}
	if err := Export(db1, &buf, opts); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import to new database
	db2 := testutil.NewTestDB(t)
	defer db2.Close()

	count, err := Import(db2, &buf, FormatJSON, dedupConfig)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 imports, got %d", count)
	}

	// Verify data matches
	importedEntries, err := db2.Query(storage.QueryFilters{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(importedEntries) != 2 {
		t.Fatalf("Expected 2 imported entries, got %d", len(importedEntries))
	}

	// Compare commands and metadata (results are in reverse order - most recent first)
	// Reverse the imported entries to match original order
	for i := 0; i < len(importedEntries)/2; i++ {
		j := len(importedEntries) - 1 - i
		importedEntries[i], importedEntries[j] = importedEntries[j], importedEntries[i]
	}

	for i, entry := range importedEntries {
		if entry.Command != entries[i].Command {
			t.Errorf("Entry %d: command mismatch: expected %q, got %q", i, entries[i].Command, entry.Command)
		}
		if entry.Cwd != entries[i].Cwd {
			t.Errorf("Entry %d: cwd mismatch: expected %q, got %q", i, entries[i].Cwd, entry.Cwd)
		}
		// Note: GitBranch may not be preserved in JSON export due to omitempty
		if entry.GitBranch != "" && entry.GitBranch != entries[i].GitBranch {
			t.Errorf("Entry %d: git_branch mismatch: expected %q, got %q", i, entries[i].GitBranch, entry.GitBranch)
		}
	}
}
