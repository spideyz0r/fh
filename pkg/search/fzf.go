package search

import (
	"fmt"
	"strings"
	"time"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spideyz0r/fh/pkg/storage"
)

// FzfSearch launches an interactive FZF selector using ktr0731/go-fuzzyfinder.
func FzfSearch(entries []*storage.HistoryEntry, preFilter string) (*storage.HistoryEntry, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no history entries found")
	}

	// If preFilter is provided, filter entries first
	filteredEntries := entries
	if preFilter != "" {
		filteredEntries = filterEntries(entries, preFilter)
		if len(filteredEntries) == 0 {
			return nil, fmt.Errorf("no entries match filter: %s", preFilter)
		}
	}

	// Use ktr0731/go-fuzzyfinder
	idx, err := fuzzyfinder.Find(
		filteredEntries,
		func(i int) string {
			// Return the display string for fuzzy matching
			return FormatEntry(filteredEntries[i])
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			entry := filteredEntries[i]

			// Build preview
			preview := fmt.Sprintf("Command: %s\n\n", entry.Command)
			preview += fmt.Sprintf("Time:     %s\n", time.Unix(entry.Timestamp, 0).Format("2006-01-02 15:04:05"))
			preview += fmt.Sprintf("Cwd:      %s\n", entry.Cwd)
			preview += fmt.Sprintf("Exit:     %d\n", entry.ExitCode)
			if entry.DurationMs > 0 {
				preview += fmt.Sprintf("Duration: %dms\n", entry.DurationMs)
			}
			if entry.GitBranch != "" {
				preview += fmt.Sprintf("Branch:   %s\n", entry.GitBranch)
			}
			preview += fmt.Sprintf("Host:     %s\n", entry.Hostname)
			preview += fmt.Sprintf("User:     %s\n", entry.User)
			preview += fmt.Sprintf("Shell:    %s\n", entry.Shell)

			return preview
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("fzf search failed: %w", err)
	}

	return filteredEntries[idx], nil
}

// filterEntries filters entries by command text.
func filterEntries(entries []*storage.HistoryEntry, query string) []*storage.HistoryEntry {
	query = strings.ToLower(query)
	var filtered []*storage.HistoryEntry
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Command), query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// FormatEntry formats a history entry for FZF display.
// Format: timestamp | cwd | command.
func FormatEntry(entry *storage.HistoryEntry) string {
	// Format timestamp
	ts := time.Unix(entry.Timestamp, 0).Format("2006-01-02 15:04:05")

	// Truncate cwd if too long
	cwd := entry.Cwd
	if len(cwd) > 40 {
		cwd = "..." + cwd[len(cwd)-37:]
	}

	// Format duration
	duration := ""
	if entry.DurationMs > 0 {
		if entry.DurationMs < 1000 {
			duration = fmt.Sprintf("%dms", entry.DurationMs)
		} else {
			duration = fmt.Sprintf("%.1fs", float64(entry.DurationMs)/1000.0)
		}
	}

	// Build parts
	parts := []string{ts}

	if cwd != "" {
		parts = append(parts, fmt.Sprintf("%-40s", cwd))
	}

	if duration != "" {
		parts = append(parts, fmt.Sprintf("[%s]", duration))
	}

	if entry.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("[exit:%d]", entry.ExitCode))
	}

	parts = append(parts, entry.Command)

	return strings.Join(parts, " │ ")
}

// ExtractCommand extracts the command from a formatted entry line.
// This is useful if you need to parse FZF output back to command.
func ExtractCommand(formattedEntry string) string {
	// Split by separator
	parts := strings.Split(formattedEntry, " │ ")
	if len(parts) == 0 {
		return ""
	}

	// Command is the last part
	return parts[len(parts)-1]
}
