package importer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// BashHistoryEntry represents a parsed entry from bash history
type BashHistoryEntry struct {
	Timestamp int64
	Command   string
}

// ParseBashHistory parses ~/.bash_history and returns all entries
func ParseBashHistory() ([]*BashHistoryEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	historyPath := filepath.Join(home, ".bash_history")
	return ParseBashHistoryFile(historyPath)
}

// ParseBashHistoryFile parses a bash history file at the given path
func ParseBashHistoryFile(path string) ([]*BashHistoryEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*BashHistoryEntry{}, nil // Return empty if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []*BashHistoryEntry
	scanner := bufio.NewScanner(file)

	var currentTimestamp int64
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if this is a timestamp line (format: #1234567890)
		if strings.HasPrefix(line, "#") && len(line) > 1 {
			// Try to parse as timestamp
			if ts, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
				currentTimestamp = ts
				continue
			}
			// If parsing failed, treat it as a comment/command
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// This is a command line
		entry := &BashHistoryEntry{
			Command:   line,
			Timestamp: currentTimestamp,
		}

		// If no timestamp was set, use current time
		if entry.Timestamp == 0 {
			entry.Timestamp = time.Now().Unix()
		}

		entries = append(entries, entry)
		currentTimestamp = 0 // Reset for next command
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading history file: %w", err)
	}

	return entries, nil
}

// GetBashHistoryPath returns the path to bash history file
func GetBashHistoryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".bash_history"), nil
}
