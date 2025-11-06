package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
)

// Format represents an export format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// Options contains export configuration
type Options struct {
	Format  Format
	Filters storage.QueryFilters
}

// Export writes history entries to the writer in the specified format
func Export(db storage.Store, writer io.Writer, opts Options) error {
	// Query entries with filters
	entries, err := db.Query(opts.Filters)
	if err != nil {
		return fmt.Errorf("failed to query entries: %w", err)
	}

	switch opts.Format {
	case FormatText:
		return exportText(entries, writer)
	case FormatJSON:
		return exportJSON(entries, writer)
	case FormatCSV:
		return exportCSV(entries, writer)
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}
}

// exportText exports entries as plain text (one command per line)
func exportText(entries []*storage.HistoryEntry, writer io.Writer) error {
	for _, entry := range entries {
		_, err := fmt.Fprintln(writer, entry.Command)
		if err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}
	return nil
}

// exportJSON exports entries as JSON array with full metadata
func exportJSON(entries []*storage.HistoryEntry, writer io.Writer) error {
	// Convert entries to JSON-friendly format
	type JSONEntry struct {
		ID         int64  `json:"id"`
		Command    string `json:"command"`
		Timestamp  int64  `json:"timestamp"`
		ExitCode   int    `json:"exit_code"`
		Cwd        string `json:"cwd"`
		Hostname   string `json:"hostname"`
		User       string `json:"user"`
		Shell      string `json:"shell"`
		DurationMs int64  `json:"duration_ms"`
		GitBranch  string `json:"git_branch,omitempty"`
		SessionID  string `json:"session_id"`
		CreatedAt  string `json:"created_at,omitempty"`
	}

	jsonEntries := make([]JSONEntry, len(entries))
	for i, entry := range entries {
		jsonEntries[i] = JSONEntry{
			ID:         entry.ID,
			Command:    entry.Command,
			Timestamp:  entry.Timestamp,
			ExitCode:   entry.ExitCode,
			Cwd:        entry.Cwd,
			Hostname:   entry.Hostname,
			User:       entry.User,
			Shell:      entry.Shell,
			DurationMs: entry.DurationMs,
			GitBranch:  entry.GitBranch,
			SessionID:  entry.SessionID,
		}
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonEntries); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// exportCSV exports entries as CSV
func exportCSV(entries []*storage.HistoryEntry, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"id",
		"timestamp",
		"command",
		"exit_code",
		"cwd",
		"hostname",
		"user",
		"shell",
		"duration_ms",
		"git_branch",
		"session_id",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write entries
	for _, entry := range entries {
		record := []string{
			strconv.FormatInt(entry.ID, 10),
			formatTimestamp(entry.Timestamp),
			entry.Command,
			strconv.Itoa(entry.ExitCode),
			entry.Cwd,
			entry.Hostname,
			entry.User,
			entry.Shell,
			strconv.FormatInt(entry.DurationMs, 10),
			entry.GitBranch,
			entry.SessionID,
		}
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// formatTimestamp formats a Unix timestamp as ISO 8601
func formatTimestamp(ts int64) string {
	return time.Unix(ts, 0).Format(time.RFC3339)
}

// ParseFormat parses a format string
func ParseFormat(s string) (Format, error) {
	switch s {
	case "text", "txt":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("unknown format: %s (supported: text, json, csv)", s)
	}
}
