package ai

import (
	"testing"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestCleanSQLResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean SQL without markdown",
			input:    "SELECT * FROM history LIMIT 10",
			expected: "SELECT * FROM history LIMIT 10",
		},
		{
			name:     "SQL with markdown code block (sql)",
			input:    "```sql\nSELECT * FROM history LIMIT 10\n```",
			expected: "SELECT * FROM history LIMIT 10",
		},
		{
			name:     "SQL with markdown code block (generic)",
			input:    "```\nSELECT * FROM history LIMIT 10\n```",
			expected: "SELECT * FROM history LIMIT 10",
		},
		{
			name:     "SQL with extra whitespace",
			input:    "  \n  SELECT * FROM history LIMIT 10  \n  ",
			expected: "SELECT * FROM history LIMIT 10",
		},
		{
			name:     "SQL with markdown and whitespace",
			input:    "```sql\n  SELECT * FROM history LIMIT 10  \n```",
			expected: "SELECT * FROM history LIMIT 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanSQLResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSQL(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid SELECT query",
			sql:         "SELECT * FROM history WHERE command LIKE '%git%' LIMIT 10",
			expectError: false,
		},
		{
			name:        "Valid SELECT with ORDER BY",
			sql:         "SELECT * FROM history ORDER BY timestamp DESC LIMIT 100",
			expectError: false,
		},
		{
			name:        "Valid SELECT with WHERE and LIMIT",
			sql:         "SELECT id, command, timestamp FROM history WHERE exit_code = 0 LIMIT 50",
			expectError: false,
		},
		{
			name:        "Query without SELECT",
			sql:         "UPDATE history SET command = 'test'",
			expectError: true,
			errorMsg:    "must start with SELECT",
		},
		{
			name:        "Query without FROM history",
			sql:         "SELECT * FROM users LIMIT 10",
			expectError: true,
			errorMsg:    "must select from history table",
		},
		{
			name:        "Query with DROP",
			sql:         "SELECT * FROM history; DROP TABLE history;",
			expectError: true,
			errorMsg:    "DROP",
		},
		{
			name:        "Query with DELETE",
			sql:         "DELETE FROM history WHERE id = 1",
			expectError: true,
			errorMsg:    "SELECT", // Caught by first validation
		},
		{
			name:        "Query with INSERT",
			sql:         "INSERT INTO history (command) VALUES ('test')",
			expectError: true,
			errorMsg:    "SELECT", // Caught by first validation
		},
		{
			name:        "Query with UPDATE",
			sql:         "UPDATE history SET command = 'test' WHERE id = 1",
			expectError: true,
			errorMsg:    "SELECT", // Caught by first validation
		},
		{
			name:        "Query with ALTER",
			sql:         "ALTER TABLE history ADD COLUMN new_col TEXT",
			expectError: true,
			errorMsg:    "SELECT", // Caught by first validation
		},
		{
			name:        "Query with CREATE",
			sql:         "CREATE TABLE malicious (id INT)",
			expectError: true,
			errorMsg:    "SELECT", // Caught by first validation
		},
		{
			name:        "SELECT with lowercase",
			sql:         "select * from history limit 10",
			expectError: false,
		},
		{
			name:        "Mixed case with dangerous keyword",
			sql:         "SeLeCt * FrOm HiStOrY; dRoP tAbLe history;",
			expectError: true,
			errorMsg:    "DROP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQL(tt.sql)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		results  []*storage.HistoryEntry
		expected int
	}{
		{
			name:     "Empty results",
			results:  []*storage.HistoryEntry{},
			expected: 0,
		},
		{
			name: "Single short entry",
			results: []*storage.HistoryEntry{
				{Command: "ls", Cwd: "/home"},
			},
			// (2 + 5 + 30) / 4 = 37 / 4 = 9.25 ≈ 9
			expected: 9,
		},
		{
			name: "Multiple entries",
			results: []*storage.HistoryEntry{
				{Command: "git status", Cwd: "/home/project"},
				{Command: "git commit -m 'test'", Cwd: "/home/project"},
				{Command: "git push", Cwd: "/home/project"},
			},
			// Entry 1: (10 + 13 + 30) / 4 = 53 / 4 = 13.25
			// Entry 2: (20 + 13 + 30) / 4 = 63 / 4 = 15.75
			// Entry 3: (8 + 13 + 30) / 4 = 51 / 4 = 12.75
			// Total: 53 + 63 + 51 = 167 / 4 = 41.75 ≈ 41
			expected: 41,
		},
		{
			name: "Long command",
			results: []*storage.HistoryEntry{
				{
					Command: "docker run -it --rm -v /home/user/data:/data -p 8080:80 nginx:latest",
					Cwd:     "/home/user/projects/web",
				},
			},
			// (70 + 23 + 30) / 4 = 123 / 4 = 30.75 ≈ 30
			expected: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokens(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChunkResults(t *testing.T) {
	// Create test entries
	createEntry := func(cmd string, cwd string) *storage.HistoryEntry {
		return &storage.HistoryEntry{
			Command: cmd,
			Cwd:     cwd,
		}
	}

	tests := []struct {
		name              string
		results           []*storage.HistoryEntry
		maxTokensPerChunk int
		expectedChunks    int
		checkFirstChunk   bool
		firstChunkSize    int
	}{
		{
			name:              "Empty results",
			results:           []*storage.HistoryEntry{},
			maxTokensPerChunk: 100,
			expectedChunks:    0,
		},
		{
			name: "Single entry within limit",
			results: []*storage.HistoryEntry{
				createEntry("ls", "/home"),
			},
			maxTokensPerChunk: 100,
			expectedChunks:    1,
			checkFirstChunk:   true,
			firstChunkSize:    1,
		},
		{
			name: "Multiple entries within single chunk",
			results: []*storage.HistoryEntry{
				createEntry("ls", "/home"),
				createEntry("pwd", "/home"),
				createEntry("cd /tmp", "/home"),
			},
			maxTokensPerChunk: 100,
			expectedChunks:    1,
			checkFirstChunk:   true,
			firstChunkSize:    3,
		},
		{
			name: "Entries requiring multiple chunks",
			results: []*storage.HistoryEntry{
				// Each entry: ~10 tokens
				createEntry("ls -la", "/home/user"),
				createEntry("pwd", "/home/user"),
				createEntry("cd /tmp", "/home/user"),
				createEntry("echo test", "/home/user"),
				createEntry("cat file", "/home/user"),
				createEntry("rm file", "/home/user"),
			},
			maxTokensPerChunk: 25, // Should fit ~2 entries per chunk
			expectedChunks:    3,
		},
		{
			name: "Very small chunk size",
			results: []*storage.HistoryEntry{
				createEntry("ls", "/home"),
				createEntry("pwd", "/home"),
				createEntry("cd", "/home"),
			},
			maxTokensPerChunk: 5, // Each entry is ~9 tokens, so 1 per chunk
			expectedChunks:    3,
			checkFirstChunk:   true,
			firstChunkSize:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkResults(tt.results, tt.maxTokensPerChunk)
			assert.Equal(t, tt.expectedChunks, len(chunks))

			if tt.checkFirstChunk && len(chunks) > 0 {
				assert.Equal(t, tt.firstChunkSize, len(chunks[0]))
			}

			// Verify all entries are accounted for
			totalEntries := 0
			for _, chunk := range chunks {
				totalEntries += len(chunk)
			}
			assert.Equal(t, len(tt.results), totalEntries)
		})
	}
}

func TestChunkResults_EntryOrder(t *testing.T) {
	// Verify that chunking preserves entry order
	results := []*storage.HistoryEntry{
		{Command: "cmd1", Cwd: "/home"},
		{Command: "cmd2", Cwd: "/home"},
		{Command: "cmd3", Cwd: "/home"},
		{Command: "cmd4", Cwd: "/home"},
	}

	chunks := chunkResults(results, 15) // Small chunk size to force splits

	// Collect all commands in order
	var allCommands []string
	for _, chunk := range chunks {
		for _, entry := range chunk {
			allCommands = append(allCommands, entry.Command)
		}
	}

	// Verify order is preserved
	assert.Equal(t, []string{"cmd1", "cmd2", "cmd3", "cmd4"}, allCommands)
}

func TestEstimateTokens_Consistency(t *testing.T) {
	// Test that estimation is consistent
	entry := &storage.HistoryEntry{
		Command: "git commit -m 'test message'",
		Cwd:     "/home/user/project",
	}

	results1 := []*storage.HistoryEntry{entry}
	results2 := []*storage.HistoryEntry{entry}

	tokens1 := estimateTokens(results1)
	tokens2 := estimateTokens(results2)

	assert.Equal(t, tokens1, tokens2, "Token estimation should be consistent")
}

func TestValidateSQL_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "Empty string",
			sql:         "",
			expectError: true,
		},
		{
			name:        "Only whitespace",
			sql:         "   \n\t   ",
			expectError: true,
		},
		{
			name:        "SELECT with comments",
			sql:         "SELECT * FROM history -- comment\nLIMIT 10",
			expectError: false,
		},
		{
			name:        "Multi-line SELECT",
			sql:         "SELECT\n  id,\n  command,\n  timestamp\nFROM history\nLIMIT 10",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQL(tt.sql)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChunkResults_LargeDataset(t *testing.T) {
	// Create a large dataset
	var results []*storage.HistoryEntry
	for i := 0; i < 1000; i++ {
		results = append(results, &storage.HistoryEntry{
			Timestamp: time.Now().Unix(),
			Command:   "test command with some length to it",
			Cwd:       "/home/user/project/subdirectory",
		})
	}

	chunks := chunkResults(results, 5000) // ~500 entries per chunk

	// Verify chunking worked
	assert.Greater(t, len(chunks), 0)
	assert.Less(t, len(chunks), 10) // Should be reasonable number of chunks

	// Verify all entries are present
	totalEntries := 0
	for _, chunk := range chunks {
		totalEntries += len(chunk)
	}
	assert.Equal(t, 1000, totalEntries)
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "String shorter than limit",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "String equal to limit",
			input:    "HelloWorld",
			maxLen:   10,
			expected: "HelloWorld",
		},
		{
			name:     "String longer than limit",
			input:    "Hello World, this is a long string",
			maxLen:   10,
			expected: "Hello Worl...",
		},
		{
			name:     "Empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "Single character truncation",
			input:    "AB",
			maxLen:   1,
			expected: "A...",
		},
		{
			name:     "Zero length limit",
			input:    "Hello",
			maxLen:   0,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
