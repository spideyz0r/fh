package search

import (
	"fmt"
	"strings"
	"time"

	"github.com/koki-develop/go-fzf"
	"github.com/spideyz0r/fh/pkg/storage"
)

// FzfSearch launches an interactive FZF selector with history entries
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

	// Create FZF instance with custom keybindings
	// Note: go-fzf doesn't support PageUp/PageDown natively
	// TODO: Consider switching to native fzf binary for full feature support
	f, err := fzf.New(
		fzf.WithNoLimit(true), // Show all results
		fzf.WithKeyMap(fzf.KeyMap{
			Up:     []string{"up", "ctrl-k", "ctrl-p", "pgup"},     // Added pgup
			Down:   []string{"down", "ctrl-j", "ctrl-n", "pgdown"}, // Added pgdown
			Toggle: []string{"tab"},
			Choose: []string{"enter"},
			Abort:  []string{"esc", "ctrl-c"},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fzf: %w", err)
	}

	// Format entries for display
	items := make([]string, len(filteredEntries))
	for i, entry := range filteredEntries {
		items[i] = FormatEntry(entry)
	}

	// Find with FZF
	indexes, err := f.Find(items, func(i int) string { return items[i] })
	if err != nil {
		return nil, fmt.Errorf("fzf search failed: %w", err)
	}

	// Return selected entry (first one if multiple selected)
	if len(indexes) == 0 {
		return nil, fmt.Errorf("no selection made")
	}

	return filteredEntries[indexes[0]], nil
}

// filterEntries filters entries by command text
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

// FormatEntry formats a history entry for FZF display
// Format: timestamp | cwd | command
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

// ExtractCommand extracts the command from a formatted entry line
// This is useful if you need to parse FZF output back to command
func ExtractCommand(formattedEntry string) string {
	// Split by separator
	parts := strings.Split(formattedEntry, " │ ")
	if len(parts) == 0 {
		return ""
	}

	// Command is the last part
	return parts[len(parts)-1]
}
