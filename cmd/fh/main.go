package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spideyz0r/fh/pkg/capture"
	"github.com/spideyz0r/fh/pkg/config"
	"github.com/spideyz0r/fh/pkg/search"
	"github.com/spideyz0r/fh/pkg/storage"
)

const (
	version = "0.1.0-dev"
)

func main() {
	// Define flags
	saveCmd := flag.NewFlagSet("save", flag.ExitOnError)
	saveCommand := saveCmd.String("cmd", "", "Command to save")
	saveExitCode := saveCmd.Int("exit-code", 0, "Exit code of the command")
	saveDuration := saveCmd.Int64("duration", 0, "Duration in milliseconds")

	// Check if we have arguments
	if len(os.Args) < 2 {
		// No arguments - launch FZF search
		handleSearch("")
		return
	}

	// Parse the command
	switch os.Args[1] {
	case "--save", "save":
		if err := saveCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing save flags: %v\n", err)
			os.Exit(1)
		}
		handleSave(*saveCommand, *saveExitCode, *saveDuration)

	case "--version", "-v":
		fmt.Printf("fh version %s\n", version)

	case "--help", "-h", "help":
		printUsage()

	default:
		// Anything else is treated as a search query
		query := strings.Join(os.Args[1:], " ")
		handleSearch(query)
	}
}

func handleSave(command string, exitCode int, durationMs int64) {
	if command == "" {
		fmt.Fprintf(os.Stderr, "Error: --cmd is required\n")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Collect metadata
	meta, err := capture.Collect(command, exitCode, durationMs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting metadata: %v\n", err)
		os.Exit(1)
	}

	// Open database
	db, err := storage.Open(cfg.GetDatabasePath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create history entry
	entry := &storage.HistoryEntry{
		Timestamp:  meta.Timestamp,
		Command:    meta.Command,
		Cwd:        meta.Cwd,
		ExitCode:   meta.ExitCode,
		Hostname:   meta.Hostname,
		User:       meta.User,
		Shell:      meta.Shell,
		DurationMs: meta.DurationMs,
		GitBranch:  meta.GitBranch,
		SessionID:  meta.SessionID,
	}

	// Get deduplication config
	dedupConfig := cfg.GetDedupConfig()

	// Insert with deduplication
	if err := db.InsertWithDedup(entry, dedupConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving command: %v\n", err)
		os.Exit(1)
	}

	// Success - silent exit (important for shell hooks)
}

func handleSearch(query string) {
	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Open database
	db, err := storage.Open(cfg.GetDatabasePath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Search history with configured limit
	limit := cfg.Search.Limit
	entries, err := search.SearchAll(db, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching history: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No history entries found\n")
		os.Exit(0)
	}

	// Launch FZF
	selected, err := search.FzfSearch(entries, query)
	if err != nil {
		// User cancelled or error - exit silently
		os.Exit(0)
	}

	// Print selected command to stdout
	fmt.Println(selected.Command)
}

func printUsage() {
	fmt.Printf(`fh - Fast History
Version: %s

USAGE:
    fh [OPTIONS]

OPTIONS:
    --save              Save a command to history
        --cmd <cmd>         Command to save (required)
        --exit-code <code>  Exit code (default: 0)
        --duration <ms>     Duration in milliseconds (default: 0)

    --version, -v       Show version
    --help, -h          Show this help

EXAMPLES:
    # Save a command (typically called from shell hooks)
    fh --save --cmd "ls -la" --exit-code 0 --duration 150

    # Show version
    fh --version

ENVIRONMENT:
    FH_DB_PATH          Override database path (default: ~/.fh/history.db)

For more information, visit: https://github.com/spideyz0r/fh
`, version)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseInt safely parses an integer
func parseInt(s string, defaultValue int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return defaultValue
}

// parseInt64 safely parses an int64
func parseInt64(s string, defaultValue int64) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return defaultValue
}
