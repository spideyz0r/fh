package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/spideyz0r/fh/pkg/stats"
	"github.com/spideyz0r/fh/pkg/storage"
)

const schemaPrompt = `Database Schema:
  table: history
  columns:
    - id (INTEGER PRIMARY KEY)
    - timestamp (INTEGER, unix timestamp in seconds)
    - command (TEXT)
    - cwd (TEXT, working directory)
    - exit_code (INTEGER)
    - hostname (TEXT)
    - user (TEXT)
    - shell (TEXT)
    - duration_ms (INTEGER, command duration in milliseconds)
    - git_branch (TEXT)
    - session_id (TEXT)`

// GenerateSQLPrompt creates a prompt for SQL query generation
func GenerateSQLPrompt(statistics *stats.Stats, userQuery string) string {
	now := time.Now()

	// Format top commands
	topCommands := []string{}
	for i, cmd := range statistics.TopCommands {
		if i >= 5 {
			break
		}
		topCommands = append(topCommands, fmt.Sprintf("    - %s (%d times)", cmd.Command, cmd.Count))
	}

	return fmt.Sprintf(`You are a shell history SQL query assistant.

Current Date/Time: %s

%s

Database Stats:
  Total commands: %d
  Unique commands: %d
  Date range: %s to %s
  Success rate: %.1f%%
  Top commands:
%s

User Query: "%s"

Generate a SQLite query to answer this question.
Return ONLY the SQL query, no explanation, no markdown, no code blocks.

Important Notes:
- Use strftime() for date math (timestamp is unix epoch in seconds)
- For "last week" use: timestamp > strftime('%%s', 'now', '-7 days')
- For "yesterday" use: timestamp > strftime('%%s', 'now', '-1 day') AND timestamp < strftime('%%s', 'now', 'start of day')
- For "today" use: timestamp > strftime('%%s', 'now', 'start of day')
- Results should be ordered by timestamp DESC unless the query asks for something else
- Limit results to reasonable amounts (e.g., LIMIT 100)
- The current date is %s`,
		now.Format("2006-01-02 15:04:05 MST"),
		schemaPrompt,
		statistics.TotalCommands,
		statistics.UniqueCommands,
		statistics.FirstCommand.Format("2006-01-02"),
		statistics.LastCommand.Format("2006-01-02"),
		statistics.SuccessRate,
		strings.Join(topCommands, "\n"),
		userQuery,
		now.Format("2006-01-02"),
	)
}

// GenerateSQLRetryPrompt creates a prompt for retrying SQL generation after an error
func GenerateSQLRetryPrompt(previousSQL, sqlError string) string {
	return fmt.Sprintf(`The SQL query you generated had an error:

SQL: %s

Error: %s

Please fix the query and try again.
Return ONLY the corrected SQL query, no explanation, no markdown, no code blocks.`,
		previousSQL,
		sqlError,
	)
}

// GenerateFormatPrompt creates a prompt for formatting query results
func GenerateFormatPrompt(userQuery string, results []*storage.HistoryEntry) string {
	// Build results string
	var resultLines []string
	for _, entry := range results {
		timestamp := time.Unix(entry.Timestamp, 0).Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("[%s] %s %s", timestamp, entry.Cwd, entry.Command)
		resultLines = append(resultLines, line)
	}

	return fmt.Sprintf(`You are a shell history assistant. Format these command results for CLI display.

User asked: "%s"

Results (%d commands):
%s

Instructions:
- Format for plain text CLI output (NO markdown, NO code blocks)
- Group logically if helpful (by time, task, etc.)
- Be concise but informative
- Include timestamps or context when relevant
- If there are many similar commands, summarize them
- Use plain text formatting only (spaces, newlines, dashes)`,
		userQuery,
		len(results),
		strings.Join(resultLines, "\n"),
	)
}

// GenerateChunkSummaryPrompt creates a prompt for summarizing a chunk of results
func GenerateChunkSummaryPrompt(chunk []*storage.HistoryEntry) string {
	var resultLines []string
	for _, entry := range chunk {
		timestamp := time.Unix(entry.Timestamp, 0).Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("[%s] %s", timestamp, entry.Command)
		resultLines = append(resultLines, line)
	}

	return fmt.Sprintf(`Summarize these shell commands concisely. Focus on patterns and key activities.

Commands (%d total):
%s

Provide a brief summary (2-3 sentences max) of what these commands represent.`,
		len(chunk),
		strings.Join(resultLines, "\n"),
	)
}

// GenerateFinalSynthesisPrompt creates a prompt for synthesizing multiple summaries
func GenerateFinalSynthesisPrompt(userQuery string, summaries []string) string {
	return fmt.Sprintf(`User asked: "%s"

I've analyzed their command history in chunks. Here are the summaries:

%s

Based on these summaries, provide a final answer to the user's question.
Format for plain text CLI output (NO markdown).
Be concise and directly address their question.`,
		userQuery,
		strings.Join(summaries, "\n\n"),
	)
}
