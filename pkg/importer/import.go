package importer

import (
	"fmt"

	"github.com/spideyz0r/fh/pkg/capture"
	"github.com/spideyz0r/fh/pkg/storage"
)

// ImportResult contains statistics about the import operation
type ImportResult struct {
	TotalEntries    int
	ImportedEntries int
	SkippedEntries  int
	Errors          []error
}

// ImportHistory imports history from shell-specific history files
// It detects the shell type and imports from the appropriate file
func ImportHistory(db *storage.DB, shell capture.ShellType, dedupConfig storage.DedupConfig) (*ImportResult, error) {
	switch shell {
	case capture.ShellBash:
		return importBashHistory(db, dedupConfig)
	case capture.ShellZsh:
		return importZshHistory(db, dedupConfig)
	default:
		return nil, fmt.Errorf("unsupported shell: %s", shell)
	}
}

// importBashHistory imports bash history
func importBashHistory(db *storage.DB, dedupConfig storage.DedupConfig) (*ImportResult, error) {
	result := &ImportResult{}

	entries, err := ParseBashHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to parse bash history: %w", err)
	}

	result.TotalEntries = len(entries)

	// Get current user metadata for filling in missing fields
	meta, err := capture.Collect("", 0, 0)
	if err != nil {
		// Continue with defaults if we can't collect metadata
		meta = &capture.Metadata{
			Cwd:      "",
			Hostname: "",
			User:     "",
			Shell:    string(capture.ShellBash),
		}
	}

	for _, entry := range entries {
		historyEntry := &storage.HistoryEntry{
			Timestamp:  entry.Timestamp,
			Command:    entry.Command,
			Cwd:        meta.Cwd, // Use current cwd as we don't have historical cwd
			ExitCode:   0,        // Unknown for historical entries
			Hostname:   meta.Hostname,
			User:       meta.User,
			Shell:      string(capture.ShellBash),
			DurationMs: 0,  // Unknown for bash history
			GitBranch:  "", // Unknown for historical entries
			SessionID:  "", // Not applicable for imports
		}

		// Insert with deduplication
		if err := db.InsertWithDedup(historyEntry, dedupConfig); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import command '%s': %w", entry.Command, err))
			result.SkippedEntries++
		} else {
			result.ImportedEntries++
		}
	}

	return result, nil
}

// importZshHistory imports zsh history
func importZshHistory(db *storage.DB, dedupConfig storage.DedupConfig) (*ImportResult, error) {
	result := &ImportResult{}

	entries, err := ParseZshHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to parse zsh history: %w", err)
	}

	result.TotalEntries = len(entries)

	// Get current user metadata for filling in missing fields
	meta, err := capture.Collect("", 0, 0)
	if err != nil {
		// Continue with defaults if we can't collect metadata
		meta = &capture.Metadata{
			Cwd:      "",
			Hostname: "",
			User:     "",
			Shell:    string(capture.ShellZsh),
		}
	}

	for _, entry := range entries {
		historyEntry := &storage.HistoryEntry{
			Timestamp:  entry.Timestamp,
			Command:    entry.Command,
			Cwd:        meta.Cwd, // Use current cwd as we don't have historical cwd
			ExitCode:   0,        // Unknown for historical entries
			Hostname:   meta.Hostname,
			User:       meta.User,
			Shell:      string(capture.ShellZsh),
			DurationMs: entry.Duration * 1000, // Convert seconds to milliseconds
			GitBranch:  "",                    // Unknown for historical entries
			SessionID:  "",                    // Not applicable for imports
		}

		// Insert with deduplication
		if err := db.InsertWithDedup(historyEntry, dedupConfig); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to import command '%s': %w", entry.Command, err))
			result.SkippedEntries++
		} else {
			result.ImportedEntries++
		}
	}

	return result, nil
}

// ImportFromFile imports history from a specific file path
// Useful for importing from backups or other machines
func ImportFromFile(db *storage.DB, shell capture.ShellType, filePath string, dedupConfig storage.DedupConfig) (*ImportResult, error) {
	result := &ImportResult{}

	var entries interface{}
	var err error

	switch shell {
	case capture.ShellBash:
		entries, err = ParseBashHistoryFile(filePath)
	case capture.ShellZsh:
		entries, err = ParseZshHistoryFile(filePath)
	default:
		return nil, fmt.Errorf("unsupported shell: %s", shell)
	}

	if err != nil {
		return nil, err
	}

	// Get current user metadata
	meta, err := capture.Collect("", 0, 0)
	if err != nil {
		meta = &capture.Metadata{
			Cwd:      "",
			Hostname: "",
			User:     "",
			Shell:    string(shell),
		}
	}

	// Process entries based on shell type
	switch shell {
	case capture.ShellBash:
		bashEntries, ok := entries.([]*BashHistoryEntry)
		if !ok {
			return nil, fmt.Errorf("failed to cast entries to BashHistoryEntry")
		}
		result.TotalEntries = len(bashEntries)
		for _, entry := range bashEntries {
			historyEntry := &storage.HistoryEntry{
				Timestamp:  entry.Timestamp,
				Command:    entry.Command,
				Cwd:        meta.Cwd,
				ExitCode:   0,
				Hostname:   meta.Hostname,
				User:       meta.User,
				Shell:      string(capture.ShellBash),
				DurationMs: 0,
				GitBranch:  "",
				SessionID:  "",
			}

			if err := db.InsertWithDedup(historyEntry, dedupConfig); err != nil {
				result.Errors = append(result.Errors, err)
				result.SkippedEntries++
			} else {
				result.ImportedEntries++
			}
		}

	case capture.ShellZsh:
		zshEntries, ok := entries.([]*ZshHistoryEntry)
		if !ok {
			return nil, fmt.Errorf("failed to cast entries to ZshHistoryEntry")
		}
		result.TotalEntries = len(zshEntries)
		for _, entry := range zshEntries {
			historyEntry := &storage.HistoryEntry{
				Timestamp:  entry.Timestamp,
				Command:    entry.Command,
				Cwd:        meta.Cwd,
				ExitCode:   0,
				Hostname:   meta.Hostname,
				User:       meta.User,
				Shell:      string(capture.ShellZsh),
				DurationMs: entry.Duration * 1000,
				GitBranch:  "",
				SessionID:  "",
			}

			if err := db.InsertWithDedup(historyEntry, dedupConfig); err != nil {
				result.Errors = append(result.Errors, err)
				result.SkippedEntries++
			} else {
				result.ImportedEntries++
			}
		}
	}

	return result, nil
}
