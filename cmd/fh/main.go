package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spideyz0r/fh/pkg/ai"
	"github.com/spideyz0r/fh/pkg/capture"
	"github.com/spideyz0r/fh/pkg/config"
	"github.com/spideyz0r/fh/pkg/crypto"
	"github.com/spideyz0r/fh/pkg/export"
	"github.com/spideyz0r/fh/pkg/importer"
	"github.com/spideyz0r/fh/pkg/search"
	"github.com/spideyz0r/fh/pkg/stats"
	"github.com/spideyz0r/fh/pkg/storage"
	"golang.org/x/term"
)

const (
	version = "1.2.0"
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
	exportEncrypt := exportCmd.Bool("encrypt", false, "Encrypt the export with a passphrase")

	importCmd := flag.NewFlagSet("import", flag.ExitOnError)
	importFormat := importCmd.String("format", "auto", "Import format (auto, text, json, csv)")
	importInput := importCmd.String("input", "-", "Input file (- for stdin)")
	importDecrypt := importCmd.Bool("decrypt", false, "Decrypt the import with a passphrase")

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

	case "--ask":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: query required for --ask\n")
			os.Exit(1)
		}
		// Check for --debug flag
		debug := false
		args := os.Args[2:]
		if len(args) > 0 && args[0] == "--debug" {
			debug = true
			args = args[1:]
		}
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Error: query required for --ask\n")
			os.Exit(1)
		}
		query := strings.Join(args, " ")
		handleAsk(query, debug)

	case "--export", "export":
		if err := exportCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing export flags: %v\n", err)
			os.Exit(1)
		}
		handleExport(*exportFormat, *exportOutput, *exportSearch, *exportLimit, *exportEncrypt)

	case "--import", "import":
		if err := importCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing import flags: %v\n", err)
			os.Exit(1)
		}
		handleImport(*importFormat, *importInput, *importDecrypt)

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

	// Search history with configured limit and deduplication
	filters := storage.QueryFilters{
		Limit:    cfg.Search.Limit,
		Distinct: cfg.Search.Deduplicate,
	}
	entries, err := search.WithFilters(db, filters)
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
		// User canceled or error - exit silently
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
	_ = db.Close()
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

	// Install hooks with configured keybinding
	result, err := capture.InstallHook(shell, rcFile, cfg.GetKeybinding())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing hooks: %v\n", err)
		os.Exit(1)
	}

	if result.Installed {
		fmt.Printf("✓ Installed shell hooks (backup: %s)\n", result.BackupFile)
	} else if result.KeybindingUpdate {
		fmt.Printf("✓ Shell hooks already installed (updated keybinding to %s)\n", cfg.GetKeybinding())
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

func handleAsk(query string, debug bool) {
	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if AI is enabled
	if !cfg.AI.Enabled {
		fmt.Fprintf(os.Stderr, "Error: AI search is disabled in configuration\n")
		fmt.Fprintf(os.Stderr, "Enable it in ~/.fh/config.yaml or set OPENAI_API_KEY environment variable\n")
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

	// Perform AI-powered search
	result, err := ai.Ask(db, query, cfg, debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print result
	fmt.Println(result)
}

// promptForPassphrase prompts the user for a passphrase twice and confirms they match
func promptForPassphrase() (string, error) {
	// Prompt for passphrase
	fmt.Fprint(os.Stderr, "Enter passphrase for encryption: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("error reading passphrase: %w", err)
	}

	if len(passphrase) == 0 {
		return "", fmt.Errorf("passphrase cannot be empty")
	}

	// Confirm passphrase
	fmt.Fprint(os.Stderr, "Confirm passphrase: ")
	confirm, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("error reading passphrase confirmation: %w", err)
	}

	if !bytes.Equal(passphrase, confirm) {
		return "", fmt.Errorf("passphrases do not match")
	}

	return string(passphrase), nil
}

// exportWithEncryption exports data to a buffer, encrypts it, and writes to the writer
func exportWithEncryption(db *storage.DB, writer io.Writer, opts export.Options) error {
	var buf bytes.Buffer
	if err := export.Export(db, &buf, opts); err != nil {
		return fmt.Errorf("error exporting: %w", err)
	}

	passphrase, err := promptForPassphrase()
	if err != nil {
		return err
	}

	encrypted, err := crypto.Encrypt(buf.Bytes(), passphrase)
	if err != nil {
		return fmt.Errorf("error encrypting: %w", err)
	}

	if _, err := writer.Write(encrypted); err != nil {
		return fmt.Errorf("error writing encrypted data: %w", err)
	}

	return nil
}

func handleExport(formatStr, outputPath, searchTerm string, limit int, encrypt bool) {
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
		defer func() {
			_ = writer.Close()
		}()
	}

	// Export
	opts := export.Options{
		Format:  format,
		Filters: filters,
	}

	// If encryption is requested, use encryption helper
	if encrypt {
		if err := exportWithEncryption(db, writer, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Normal export without encryption
		if err := export.Export(db, writer, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
			os.Exit(1)
		}
	}

	// Print success message to stderr if writing to file
	if outputPath != "-" && outputPath != "" {
		if encrypt {
			fmt.Fprintf(os.Stderr, "Exported and encrypted to %s\n", outputPath)
		} else {
			fmt.Fprintf(os.Stderr, "Exported to %s\n", outputPath)
		}
	}
}

// promptForDecryptPassphrase prompts for a decryption passphrase
func promptForDecryptPassphrase() (string, error) {
	fmt.Fprint(os.Stderr, "Enter passphrase to decrypt: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("error reading passphrase: %w", err)
	}

	if len(passphrase) == 0 {
		return "", fmt.Errorf("passphrase cannot be empty")
	}

	return string(passphrase), nil
}

// decryptReader reads encrypted data from a reader and returns a reader with decrypted data
func decryptReader(reader io.Reader) (io.Reader, error) {
	// Read all encrypted data
	encryptedData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading encrypted data: %w", err)
	}

	passphrase, err := promptForDecryptPassphrase()
	if err != nil {
		return nil, err
	}

	// Decrypt
	decrypted, err := crypto.Decrypt(encryptedData, passphrase)
	if err != nil {
		return nil, fmt.Errorf("error decrypting: %w", err)
	}

	return bytes.NewReader(decrypted), nil
}

// importWithAutoDetect handles import with format auto-detection
func importWithAutoDetect(db *storage.DB, reader io.Reader, dedupConfig storage.DedupConfig) error {
	detectedFormat, newReader, err := export.DetectFormat(reader)
	if err != nil {
		return fmt.Errorf("error detecting format: %w", err)
	}

	// Read all data into buffer from the new reader
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, newReader); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Auto-detected format: %s\n", detectedFormat)

	// Import from buffer
	count, err := export.Import(db, &buf, detectedFormat, dedupConfig)
	if err != nil {
		return fmt.Errorf("error importing: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Imported %d commands\n", count)
	return nil
}

func handleImport(formatStr, inputPath string, decrypt bool) {
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

	// Determine input reader
	var reader io.Reader
	var file *os.File
	if inputPath == "-" || inputPath == "" {
		reader = os.Stdin
	} else {
		file, err = os.Open(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing input file: %v\n", err)
			}
		}()
		reader = file
	}

	// Handle decryption if requested
	if decrypt {
		reader, err = decryptReader(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	dedupConfig := cfg.GetDedupConfig()

	// Handle auto-detect format
	if formatStr == "auto" {
		if err := importWithAutoDetect(db, reader, dedupConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Parse explicit format
	format, err := export.ParseFormat(formatStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Import
	count, err := export.Import(db, reader, format, dedupConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error importing: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Imported %d commands\n", count)
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

    --ask <query>       AI-powered natural language search
                        Requires OPENAI_API_KEY environment variable
        --debug         Show debug output (SQL query, responses, etc.)

    --export            Export history to different formats
        --format <fmt>      Format: text, json, csv (default: text)
        --output <file>     Output file (default: stdout)
        --search <term>     Filter by search term
        --limit <n>         Limit results (default: 0 = unlimited)
        --encrypt           Encrypt the export with AES-256-GCM

    --import            Import history from file
        --format <fmt>      Format: auto, text, json, csv (default: auto)
        --input <file>      Input file (default: stdin)
        --decrypt           Decrypt the import (AES-256-GCM)

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

    # AI-powered search (requires OPENAI_API_KEY)
    fh --ask "what git commands did I run today?"
    fh --ask "show me failed commands from last week"
    fh --ask "what docker commands did I use yesterday?"
    fh --ask --debug "what testing commands did I run today?"  # With debug output

    # Export history as JSON
    fh --export --format json --output history.json

    # Export recent 100 commands as CSV
    fh --export --format csv --limit 100 > recent.csv

    # Import history from JSON file
    fh --import --input history.json

    # Import from stdin (auto-detect format)
    cat history.csv | fh --import

    # Create encrypted backup (export with encryption)
    fh --export --format json --output backup.json.enc --encrypt

    # Restore from encrypted backup
    fh --import --input backup.json.enc --decrypt

    # Show version
    fh --version

ENVIRONMENT:
    FH_DB_PATH          Override database path (default: ~/.fh/history.db)
    OPENAI_API_KEY      OpenAI API key (required for --ask command)

For more information, visit: https://github.com/spideyz0r/fh
`, version)
}
