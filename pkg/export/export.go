package export

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
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

// Import imports history from a reader with the given format
func Import(db *storage.DB, r io.Reader, format Format, dedupConfig storage.DedupConfig) (int, error) {
	switch format {
	case FormatText:
		return importText(db, r, dedupConfig)
	case FormatJSON:
		return importJSON(db, r, dedupConfig)
	case FormatCSV:
		return importCSV(db, r, dedupConfig)
	default:
		return 0, fmt.Errorf("unsupported import format: %s", format)
	}
}

// importText imports from plain text format (one command per line)
func importText(db *storage.DB, r io.Reader, dedupConfig storage.DedupConfig) (int, error) {
	scanner := bufio.NewScanner(r)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry := &storage.HistoryEntry{
			Timestamp:  time.Now().Unix(),
			Command:    line,
			Cwd:        "",
			ExitCode:   0,
			Hostname:   "",
			User:       "",
			Shell:      "",
			DurationMs: 0,
			GitBranch:  "",
			SessionID:  "",
		}

		if err := db.InsertWithDedup(entry, dedupConfig); err != nil {
			// Skip entries that fail to insert (e.g., duplicates)
			continue
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("error reading text: %w", err)
	}

	return count, nil
}

// importJSON imports from JSON format
func importJSON(db *storage.DB, r io.Reader, dedupConfig storage.DedupConfig) (int, error) {
	var entries []*storage.HistoryEntry

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&entries); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	count := 0
	for _, entry := range entries {
		// Validate entry
		if entry.Command == "" {
			continue
		}

		// Ensure timestamp is set
		if entry.Timestamp == 0 {
			entry.Timestamp = time.Now().Unix()
		}

		if err := db.InsertWithDedup(entry, dedupConfig); err != nil {
			// Skip entries that fail to insert
			continue
		}
		count++
	}

	return count, nil
}

// importCSV imports from CSV format
func importCSV(db *storage.DB, r io.Reader, dedupConfig storage.DedupConfig) (int, error) {
	reader := csv.NewReader(r)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build column index map
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	// Verify required columns
	if _, ok := colMap["command"]; !ok {
		return 0, fmt.Errorf("CSV missing required column: command")
	}

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("error reading CSV: %w", err)
		}

		// Parse entry from CSV row
		entry := &storage.HistoryEntry{}

		// Command (required)
		if idx, ok := colMap["command"]; ok && idx < len(record) {
			entry.Command = record[idx]
		}
		if entry.Command == "" {
			continue
		}

		// Timestamp (parse from ISO 8601 if present)
		if idx, ok := colMap["timestamp"]; ok && idx < len(record) {
			// Try to parse as ISO 8601 first
			if t, err := time.Parse(time.RFC3339, record[idx]); err == nil {
				entry.Timestamp = t.Unix()
			} else if ts, err := strconv.ParseInt(record[idx], 10, 64); err == nil {
				// Fallback to Unix timestamp
				entry.Timestamp = ts
			}
		}
		if entry.Timestamp == 0 {
			entry.Timestamp = time.Now().Unix()
		}

		// Other fields
		if idx, ok := colMap["cwd"]; ok && idx < len(record) {
			entry.Cwd = record[idx]
		}
		if idx, ok := colMap["exit_code"]; ok && idx < len(record) {
			if code, err := strconv.Atoi(record[idx]); err == nil {
				entry.ExitCode = code
			}
		}
		if idx, ok := colMap["hostname"]; ok && idx < len(record) {
			entry.Hostname = record[idx]
		}
		if idx, ok := colMap["user"]; ok && idx < len(record) {
			entry.User = record[idx]
		}
		if idx, ok := colMap["shell"]; ok && idx < len(record) {
			entry.Shell = record[idx]
		}
		if idx, ok := colMap["duration_ms"]; ok && idx < len(record) {
			if dur, err := strconv.ParseInt(record[idx], 10, 64); err == nil {
				entry.DurationMs = dur
			}
		}
		if idx, ok := colMap["git_branch"]; ok && idx < len(record) {
			entry.GitBranch = record[idx]
		}
		if idx, ok := colMap["session_id"]; ok && idx < len(record) {
			entry.SessionID = record[idx]
		}

		if err := db.InsertWithDedup(entry, dedupConfig); err != nil {
			// Skip entries that fail to insert
			continue
		}
		count++
	}

	return count, nil
}

// DetectFormat attempts to auto-detect the format from file content
func DetectFormat(r io.Reader) (Format, io.Reader, error) {
	// Read first few bytes to detect format
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Create a new reader with buffered data
	newReader := io.MultiReader(bytes.NewReader(buf[:n]), r)

	content := string(buf[:n])

	// Detect JSON (starts with [ or {)
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return FormatJSON, newReader, nil
	}

	// Detect CSV (has comma-separated values with headers)
	if strings.Contains(content, ",") && strings.Contains(content, "command") {
		lines := strings.Split(content, "\n")
		if len(lines) > 0 && strings.Contains(lines[0], ",") {
			return FormatCSV, newReader, nil
		}
	}

	// Default to text
	return FormatText, newReader, nil
}
