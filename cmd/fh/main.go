package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spideyz0r/fh/pkg/capture"
	"github.com/spideyz0r/fh/pkg/config"
	"github.com/spideyz0r/fh/pkg/export"
	"github.com/spideyz0r/fh/pkg/importer"
	"github.com/spideyz0r/fh/pkg/search"
	"github.com/spideyz0r/fh/pkg/stats"
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

	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	exportFormat := exportCmd.String("format", "text", "Export format (text, json, csv)")
	exportOutput := exportCmd.String("output", "-", "Output file (- for stdout)")
	exportSearch := exportCmd.String("search", "", "Filter by search term")
	exportLimit := exportCmd.Int("limit", 0, "Limit number of results (0 = unlimited)")

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

	case "--init":
		handleInit()

	case "--stats":
		handleStats()

	case "--export", "export":
		if err := exportCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing export flags: %v\n", err)
			os.Exit(1)
		}
		handleExport(*exportFormat, *exportOutput, *exportSearch, *exportLimit)

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
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing database: %v\n", err)
		}
	}()

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
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing database: %v\n", err)
		}
	}()

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

func handleInit() {
	fmt.Println("fh - Fast History Setup")
	fmt.Println("=======================")
	fmt.Println()

	// Load or create config
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create .fh directory if it doesn't exist
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	fhDir := filepath.Join(home, ".fh")
	if err := os.MkdirAll(fhDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .fh directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Created directory: %s\n", fhDir)

	// Initialize database
	db, err := storage.Open(cfg.GetDatabasePath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	db.Close()
	fmt.Printf("✓ Initialized database: %s\n", cfg.GetDatabasePath())

	// Save default config if it doesn't exist
	configPath := filepath.Join(fhDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := cfg.Save(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Created config file: %s\n", configPath)
	} else {
		fmt.Printf("✓ Config file already exists: %s\n", configPath)
	}

	// Detect shell
	shell, err := capture.DetectShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting shell: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease set your SHELL environment variable.\n")
		os.Exit(1)
	}
	fmt.Printf("✓ Detected shell: %s\n", shell)

	// Get RC file
	rcFile, err := capture.GetRCFile(shell)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting RC file: %v\n", err)
		os.Exit(1)
	}

	// Install hooks
	result, err := capture.InstallHook(shell, rcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing hooks: %v\n", err)
		os.Exit(1)
	}

	if result.Installed {
		fmt.Printf("✓ Installed shell hooks (backup: %s)\n", result.BackupFile)
	} else {
		fmt.Printf("✓ Shell hooks already installed\n")
	}

	// Import existing history
	db, err = storage.Open(cfg.GetDatabasePath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing database: %v\n", err)
		}
	}()

	dedupConfig := cfg.GetDedupConfig()
	importResult, err := importer.ImportHistory(db, shell, dedupConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not import history: %v\n", err)
	} else if importResult.ImportedEntries > 0 {
		fmt.Printf("✓ Imported %d commands\n", importResult.ImportedEntries)
	}

	// Print success message
	successMsg := "SUCCESS! Restart your shell and press Ctrl-R to search."
	fmt.Println("\n" + strings.Repeat("=", len(successMsg)))
	fmt.Println(successMsg)
	fmt.Println(strings.Repeat("=", len(successMsg)) + "\n")
}

func handleStats() {
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
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing database: %v\n", err)
		}
	}()

	// Collect statistics
	statistics, err := stats.Collect(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting statistics: %v\n", err)
		os.Exit(1)
	}

	// Format and print
	output := statistics.Format(10) // Top 10 commands
	fmt.Print(output)
}

func handleExport(formatStr, outputPath, searchTerm string, limit int) {
	// Parse format
	format, err := export.ParseFormat(formatStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

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
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing database: %v\n", err)
		}
	}()

	// Build query filters
	filters := storage.QueryFilters{
		Search: searchTerm,
		Limit:  limit,
	}

	// Determine output writer
	var writer *os.File
	if outputPath == "-" || outputPath == "" {
		writer = os.Stdout
	} else {
		writer, err = os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer writer.Close()
	}

	// Export
	opts := export.Options{
		Format:  format,
		Filters: filters,
	}

	if err := export.Export(db, writer, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
		os.Exit(1)
	}

	// Print success message to stderr if writing to file
	if outputPath != "-" && outputPath != "" {
		fmt.Fprintf(os.Stderr, "Exported to %s\n", outputPath)
	}
}

func printUsage() {
	fmt.Printf(`fh - Fast History
Version: %s

USAGE:
    fh [OPTIONS]

OPTIONS:
    --init              Initialize fh and setup shell integration

    --save              Save a command to history
        --cmd <cmd>         Command to save (required)
        --exit-code <code>  Exit code (default: 0)
        --duration <ms>     Duration in milliseconds (default: 0)

    --stats             Show statistics about your command history

    --export            Export history to different formats
        --format <fmt>      Format: text, json, csv (default: text)
        --output <file>     Output file (default: stdout)
        --search <term>     Filter by search term
        --limit <n>         Limit results (default: 0 = unlimited)

    --version, -v       Show version
    --help, -h          Show this help

EXAMPLES:
    # Initialize fh (first time setup)
    fh --init

    # Save a command (typically called from shell hooks)
    fh --save --cmd "ls -la" --exit-code 0 --duration 150

    # Search history with FZF
    fh

    # Show statistics
    fh --stats

    # Export history as JSON
    fh --export --format json --output history.json

    # Export recent 100 commands as CSV
    fh --export --format csv --limit 100 > recent.csv

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
