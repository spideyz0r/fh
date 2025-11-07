package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spideyz0r/fh/pkg/config"
	"github.com/spideyz0r/fh/pkg/stats"
	"github.com/spideyz0r/fh/pkg/storage"
)

// Ask performs an AI-powered search query
func Ask(db *storage.DB, userQuery string, cfg *config.Config) (string, error) {
	// Check if AI is enabled
	if !cfg.AI.Enabled {
		return "", fmt.Errorf("AI search is disabled in configuration")
	}

	// Create OpenAI client
	client, err := NewOpenAIClient(cfg.AI.Model)
	if err != nil {
		return "", err
	}

	// Get database statistics
	statistics, err := stats.Collect(db)
	if err != nil {
		return "", fmt.Errorf("failed to collect database stats: %w", err)
	}

	// Phase 1: Generate SQL query with retry
	sqlQuery, err := generateSQLWithRetry(client, statistics, userQuery, cfg.AI.MaxSQLRetries)
	if err != nil {
		return "", err
	}

	// Phase 2: Execute SQL query
	results, err := executeSQLQuery(db, sqlQuery, time.Duration(cfg.AI.SQLTimeoutSecs)*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}

	// Check if we got results
	if len(results) == 0 {
		return "Could not find any data for that specific query", nil
	}

	// Phase 3: Format results (with chunking if needed)
	output, err := formatResults(client, userQuery, results, cfg.AI.MaxChunkTokens)
	if err != nil {
		return "", err
	}

	return output, nil
}

// generateSQLWithRetry attempts to generate a valid SQL query with retries
func generateSQLWithRetry(client *OpenAIClient, statistics *stats.Stats, userQuery string, maxRetries int) (string, error) {
	ctx := context.Background()
	var lastSQL string
	var lastError string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		var prompt string
		if attempt == 1 {
			// First attempt - use full prompt
			prompt = GenerateSQLPrompt(statistics, userQuery)
		} else {
			// Retry - use error feedback
			prompt = GenerateSQLRetryPrompt(lastSQL, lastError)
		}

		// Get SQL from OpenAI
		response, err := client.Query(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("OpenAI API error: %w", err)
		}

		// Clean up response (remove markdown, extra whitespace)
		sqlQuery := cleanSQLResponse(response)
		lastSQL = sqlQuery

		// Validate SQL (basic check)
		if err := validateSQL(sqlQuery); err != nil {
			lastError = err.Error()
			continue
		}

		return sqlQuery, nil
	}

	return "", fmt.Errorf("could not generate valid query after %d attempts", maxRetries)
}

// executeSQLQuery executes the SQL query with a timeout
func executeSQLQuery(db *storage.DB, sqlQuery string, timeout time.Duration) ([]*storage.HistoryEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute query
	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("SQL error: %w", err)
	}
	defer rows.Close()

	// Parse results
	var results []*storage.HistoryEntry
	for rows.Next() {
		entry := &storage.HistoryEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Command,
			&entry.Cwd,
			&entry.ExitCode,
			&entry.Hostname,
			&entry.User,
			&entry.Shell,
			&entry.DurationMs,
			&entry.GitBranch,
			&entry.SessionID,
		)
		if err != nil {
			// Try scanning partial columns (in case query doesn't select all)
			// For now, just skip rows that don't match
			continue
		}
		results = append(results, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading results: %w", err)
	}

	return results, nil
}

// formatResults formats query results using OpenAI, with chunking for large result sets
func formatResults(client *OpenAIClient, userQuery string, results []*storage.HistoryEntry, maxChunkTokens int) (string, error) {
	ctx := context.Background()

	// Estimate tokens (rough: ~4 chars per token)
	estimatedTokens := estimateTokens(results)

	// If small enough, format in one go
	if estimatedTokens < maxChunkTokens {
		prompt := GenerateFormatPrompt(userQuery, results)
		response, err := client.Query(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("failed to format results: %w", err)
		}
		return response, nil
	}

	// Large result set - chunk and summarize
	chunks := chunkResults(results, maxChunkTokens)
	var summaries []string

	for _, chunk := range chunks {
		prompt := GenerateChunkSummaryPrompt(chunk)
		summary, err := client.Query(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("failed to summarize chunk: %w", err)
		}
		summaries = append(summaries, summary)
	}

	// Final synthesis
	finalPrompt := GenerateFinalSynthesisPrompt(userQuery, summaries)
	finalResponse, err := client.Query(ctx, finalPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize final response: %w", err)
	}

	return finalResponse, nil
}

// cleanSQLResponse removes markdown code blocks and extra whitespace
func cleanSQLResponse(response string) string {
	// Remove markdown code blocks
	response = strings.TrimPrefix(response, "```sql")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")

	// Trim whitespace
	response = strings.TrimSpace(response)

	return response
}

// validateSQL performs basic validation on the SQL query
func validateSQL(sqlQuery string) error {
	upper := strings.ToUpper(sqlQuery)

	// Must start with SELECT
	if !strings.HasPrefix(strings.TrimSpace(upper), "SELECT") {
		return fmt.Errorf("query must start with SELECT")
	}

	// Must reference history table
	if !strings.Contains(upper, "FROM HISTORY") {
		return fmt.Errorf("query must select from history table")
	}

	// Blacklist dangerous keywords
	dangerous := []string{"DROP", "DELETE", "INSERT", "UPDATE", "ALTER", "CREATE"}
	for _, keyword := range dangerous {
		if strings.Contains(upper, keyword) {
			return fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	return nil
}

// estimateTokens roughly estimates the number of tokens for a set of results
func estimateTokens(results []*storage.HistoryEntry) int {
	totalChars := 0
	for _, entry := range results {
		// Rough estimate: timestamp + cwd + command
		totalChars += len(entry.Command) + len(entry.Cwd) + 30 // 30 for timestamp and formatting
	}
	// Rough conversion: ~4 chars per token
	return totalChars / 4
}

// chunkResults splits results into chunks based on token limit
func chunkResults(results []*storage.HistoryEntry, maxTokensPerChunk int) [][]*storage.HistoryEntry {
	var chunks [][]*storage.HistoryEntry
	var currentChunk []*storage.HistoryEntry
	currentTokens := 0

	for _, entry := range results {
		entryTokens := (len(entry.Command) + len(entry.Cwd) + 30) / 4
		if currentTokens+entryTokens > maxTokensPerChunk && len(currentChunk) > 0 {
			// Chunk is full, start a new one
			chunks = append(chunks, currentChunk)
			currentChunk = []*storage.HistoryEntry{entry}
			currentTokens = entryTokens
		} else {
			currentChunk = append(currentChunk, entry)
			currentTokens += entryTokens
		}
	}

	// Add remaining chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
