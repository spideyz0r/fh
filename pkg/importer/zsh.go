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

// ZshHistoryEntry represents a parsed entry from zsh history
type ZshHistoryEntry struct {
	Timestamp  int64
	Duration   int64 // in seconds
	Command    string
}

// ParseZshHistory parses ~/.zsh_history and returns all entries
func ParseZshHistory() ([]*ZshHistoryEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check ZDOTDIR first
	zdotdir := os.Getenv("ZDOTDIR")
	var historyPath string
	if zdotdir != "" {
		historyPath = filepath.Join(zdotdir, ".zsh_history")
	} else {
		historyPath = filepath.Join(home, ".zsh_history")
	}

	return ParseZshHistoryFile(historyPath)
}

// ParseZshHistoryFile parses a zsh history file at the given path
// Zsh extended_history format: : <timestamp>:<duration>;<command>
func ParseZshHistoryFile(path string) ([]*ZshHistoryEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ZshHistoryEntry{}, nil // Return empty if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []*ZshHistoryEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := parseZshLine(line)
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading history file: %w", err)
	}

	return entries, nil
}

// parseZshLine parses a single line from zsh history
func parseZshLine(line string) *ZshHistoryEntry {
	// Extended history format: : <timestamp>:<duration>;<command>
	// Example: : 1234567890:0;ls -la

	if strings.HasPrefix(line, ": ") {
		// Extended history format
		rest := line[2:] // Remove ": " prefix

		// Find the semicolon that separates metadata from command
		semicolonIdx := strings.Index(rest, ";")
		if semicolonIdx == -1 {
			// Malformed line, treat as plain command
			return &ZshHistoryEntry{
				Command:   line,
				Timestamp: time.Now().Unix(),
			}
		}

		metadata := rest[:semicolonIdx]
		command := rest[semicolonIdx+1:]

		// Parse metadata: <timestamp>:<duration>
		parts := strings.Split(metadata, ":")
		if len(parts) >= 2 {
			timestamp, err1 := strconv.ParseInt(parts[0], 10, 64)
			duration, err2 := strconv.ParseInt(parts[1], 10, 64)

			if err1 == nil {
				entry := &ZshHistoryEntry{
					Timestamp: timestamp,
					Command:   command,
				}
				if err2 == nil {
					entry.Duration = duration
				}
				return entry
			}
		}

		// If parsing failed, use the command part with current timestamp
		return &ZshHistoryEntry{
			Command:   command,
			Timestamp: time.Now().Unix(),
		}
	}

	// Plain format (no extended_history)
	return &ZshHistoryEntry{
		Command:   line,
		Timestamp: time.Now().Unix(),
	}
}

// GetZshHistoryPath returns the path to zsh history file
func GetZshHistoryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check ZDOTDIR first
	zdotdir := os.Getenv("ZDOTDIR")
	if zdotdir != "" {
		return filepath.Join(zdotdir, ".zsh_history"), nil
	}

	return filepath.Join(home, ".zsh_history"), nil
}
