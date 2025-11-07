package search

import (
	"fmt"
	"time"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spideyz0r/fh/pkg/storage"
)

// FzfSearchKtr launches an interactive FZF selector using ktr0731/go-fuzzyfinder.
func FzfSearchKtr(entries []*storage.HistoryEntry, preFilter string) (*storage.HistoryEntry, error) {
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
