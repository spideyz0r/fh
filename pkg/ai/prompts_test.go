package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/spideyz0r/fh/pkg/stats"
	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSQLPrompt(t *testing.T) {
	statistics := &stats.Stats{
		TotalCommands:  1000,
		UniqueCommands: 500,
		FirstCommand:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		LastCommand:    time.Date(2024, 11, 7, 0, 0, 0, 0, time.UTC),
		SuccessRate:    95.5,
		TopCommands: []stats.CommandCount{
			{Command: "git status", Count: 50},
			{Command: "ls -la", Count: 40},
			{Command: "cd", Count: 30},
		},
	}

	userQuery := "show me git commands from today"

	prompt := GenerateSQLPrompt(statistics, userQuery)

	// Verify prompt contains key components
	assert.Contains(t, prompt, "Database Schema:")
	assert.Contains(t, prompt, "table: history")
	assert.Contains(t, prompt, "Total commands: 1000")
	assert.Contains(t, prompt, "Unique commands: 500")
	assert.Contains(t, prompt, "Success rate: 95.5%")
	assert.Contains(t, prompt, "git status (50 times)")
	assert.Contains(t, prompt, "ls -la (40 times)")
	assert.Contains(t, prompt, userQuery)
	assert.Contains(t, prompt, "strftime")
	assert.Contains(t, prompt, "LIMIT")

	// Verify date ranges are formatted
	assert.Contains(t, prompt, "2024-01-01")
	assert.Contains(t, prompt, "2024-11-07")
}

func TestGenerateSQLRetryPrompt(t *testing.T) {
	previousSQL := "SELECT * FROM history WHERE invalid_column = 1"
	sqlError := "no such column: invalid_column"

	prompt := GenerateSQLRetryPrompt(previousSQL, sqlError)

	assert.Contains(t, prompt, previousSQL)
	assert.Contains(t, prompt, sqlError)
	assert.Contains(t, prompt, "error")
	assert.Contains(t, prompt, "fix")
}

func TestGenerateFormatPrompt(t *testing.T) {
	userQuery := "what did I do yesterday?"
	results := []*storage.HistoryEntry{
		{
			Timestamp: time.Date(2024, 11, 6, 10, 30, 0, 0, time.UTC).Unix(),
			Command:   "git commit -m 'fix bug'",
			Cwd:       "/home/user/project",
		},
		{
			Timestamp: time.Date(2024, 11, 6, 11, 15, 0, 0, time.UTC).Unix(),
			Command:   "git push origin main",
			Cwd:       "/home/user/project",
		},
	}

	prompt := GenerateFormatPrompt(userQuery, results)

	assert.Contains(t, prompt, userQuery)
	assert.Contains(t, prompt, "git commit -m 'fix bug'")
	assert.Contains(t, prompt, "git push origin main")
	assert.Contains(t, prompt, "/home/user/project")
	assert.Contains(t, prompt, "2024-11-06")
	assert.Contains(t, prompt, "2 commands")
	assert.Contains(t, prompt, "CLI output")
	assert.Contains(t, prompt, "NO markdown")
}

func TestGenerateChunkSummaryPrompt(t *testing.T) {
	chunk := []*storage.HistoryEntry{
		{
			Timestamp: time.Date(2024, 11, 6, 10, 30, 0, 0, time.UTC).Unix(),
			Command:   "docker ps",
		},
		{
			Timestamp: time.Date(2024, 11, 6, 11, 15, 0, 0, time.UTC).Unix(),
			Command:   "docker logs container_id",
		},
	}

	prompt := GenerateChunkSummaryPrompt(chunk)

	assert.Contains(t, prompt, "docker ps")
	assert.Contains(t, prompt, "docker logs container_id")
	assert.Contains(t, prompt, "2 total")
	assert.Contains(t, prompt, "Summarize")
	assert.Contains(t, prompt, "2024-11-06")
}

func TestGenerateFinalSynthesisPrompt(t *testing.T) {
	userQuery := "what docker commands did I run?"
	summaries := []string{
		"Checked running containers and inspected logs",
		"Built and pushed docker images",
	}

	prompt := GenerateFinalSynthesisPrompt(userQuery, summaries)

	assert.Contains(t, prompt, userQuery)
	assert.Contains(t, prompt, summaries[0])
	assert.Contains(t, prompt, summaries[1])
	assert.Contains(t, prompt, "answer")
	assert.Contains(t, prompt, "NO markdown")
}

func TestSchemaPrompt(t *testing.T) {
	// Verify schema prompt contains all expected columns
	assert.Contains(t, schemaPrompt, "history")
	assert.Contains(t, schemaPrompt, "timestamp")
	assert.Contains(t, schemaPrompt, "command")
	assert.Contains(t, schemaPrompt, "cwd")
	assert.Contains(t, schemaPrompt, "exit_code")
	assert.Contains(t, schemaPrompt, "hostname")
	assert.Contains(t, schemaPrompt, "user")
	assert.Contains(t, schemaPrompt, "shell")
	assert.Contains(t, schemaPrompt, "duration_ms")
	assert.Contains(t, schemaPrompt, "git_branch")
	assert.Contains(t, schemaPrompt, "session_id")
}

func TestGenerateSQLPrompt_TopCommandsLimit(t *testing.T) {
	// Test that only top 5 commands are included
	statistics := &stats.Stats{
		TotalCommands:  100,
		UniqueCommands: 50,
		FirstCommand:   time.Now().Add(-30 * 24 * time.Hour),
		LastCommand:    time.Now(),
		SuccessRate:    90.0,
		TopCommands: []stats.CommandCount{
			{Command: "cmd1", Count: 10},
			{Command: "cmd2", Count: 9},
			{Command: "cmd3", Count: 8},
			{Command: "cmd4", Count: 7},
			{Command: "cmd5", Count: 6},
			{Command: "cmd6", Count: 5}, // Should not be included
			{Command: "cmd7", Count: 4}, // Should not be included
		},
	}

	prompt := GenerateSQLPrompt(statistics, "test query")

	assert.Contains(t, prompt, "cmd1")
	assert.Contains(t, prompt, "cmd2")
	assert.Contains(t, prompt, "cmd3")
	assert.Contains(t, prompt, "cmd4")
	assert.Contains(t, prompt, "cmd5")
	assert.NotContains(t, prompt, "cmd6")
	assert.NotContains(t, prompt, "cmd7")
}

func TestGenerateFormatPrompt_EmptyResults(t *testing.T) {
	userQuery := "show me commands"
	results := []*storage.HistoryEntry{}

	prompt := GenerateFormatPrompt(userQuery, results)

	assert.Contains(t, prompt, userQuery)
	assert.Contains(t, prompt, "0 commands")
}

func TestPromptConsistency(t *testing.T) {
	// Verify all prompts mention avoiding markdown for CLI output
	statistics := &stats.Stats{
		TotalCommands:  100,
		UniqueCommands: 50,
		FirstCommand:   time.Now(),
		LastCommand:    time.Now(),
		SuccessRate:    90.0,
		TopCommands:    []stats.CommandCount{{Command: "test", Count: 1}},
	}

	sqlPrompt := GenerateSQLPrompt(statistics, "test")
	formatPrompt := GenerateFormatPrompt("test", []*storage.HistoryEntry{})
	synthesisPrompt := GenerateFinalSynthesisPrompt("test", []string{"summary"})

	// SQL prompt should avoid markdown in output instructions
	assert.Contains(t, strings.ToLower(sqlPrompt), "no markdown")

	// Format and synthesis prompts should explicitly avoid markdown
	assert.Contains(t, strings.ToUpper(formatPrompt), "NO MARKDOWN")
	assert.Contains(t, strings.ToUpper(synthesisPrompt), "NO MARKDOWN")
}
