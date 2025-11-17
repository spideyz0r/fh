package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spideyz0r/fh/pkg/capture"
	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellHookGeneration tests that shell hooks are generated correctly
func TestShellHookGeneration(t *testing.T) {
	tests := []struct {
		name      string
		shellType capture.ShellType
		wantFuncs []string // Functions that should exist in the hook
	}{
		{
			name:      "bash hook",
			shellType: capture.ShellBash,
			wantFuncs: []string{"__fh_save", "__fh_widget", "PROMPT_COMMAND", "bind"},
		},
		{
			name:      "zsh hook",
			shellType: capture.ShellZsh,
			wantFuncs: []string{"__fh_save", "__fh_widget", "precmd_functions", "bindkey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := capture.GetHookContent(tt.shellType, "ctrl-r")
			require.NoError(t, err)
			require.NotEmpty(t, content)

			for _, fn := range tt.wantFuncs {
				assert.Contains(t, content, fn, "hook should contain %s", fn)
			}

			// Verify it calls fh --save
			assert.Contains(t, content, "fh --save")
			assert.Contains(t, content, "--cmd")
			assert.Contains(t, content, "--exit-code")
		})
	}
}

// TestInitCommand tests the --init command in isolation
func TestInitCommand(t *testing.T) {
	// Create isolated temp directory
	tempDir := t.TempDir()

	// Build fh binary
	fhBinary := buildFhBinary(t)

	// Run --init in isolated environment
	cmd := exec.Command(fhBinary, "--init")
	cmd.Env = []string{
		"HOME=" + tempDir,
		"SHELL=/bin/bash", // Simulate bash shell
		"PATH=" + os.Getenv("PATH"),
	}

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "init should succeed: %s", output)

	// Verify directory structure created
	fhDir := filepath.Join(tempDir, ".fh")
	assert.DirExists(t, fhDir)

	// Verify database created
	dbPath := filepath.Join(fhDir, "history.db")
	assert.FileExists(t, dbPath)

	// Verify config created
	configPath := filepath.Join(fhDir, "config.yaml")
	assert.FileExists(t, configPath)

	// Verify bash hooks installed
	bashProfile := filepath.Join(tempDir, ".bash_profile")
	assert.FileExists(t, bashProfile)

	// Verify backup created
	backupPath := bashProfile + ".fh.backup"
	assert.FileExists(t, backupPath)

	// Verify hook content
	content, err := os.ReadFile(bashProfile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "__fh_save")
	assert.Contains(t, string(content), "__fh_widget")
}

// TestSaveCommand tests the --save command directly
func TestSaveCommand(t *testing.T) {
	tempDir := t.TempDir()
	fhBinary := buildFhBinary(t)

	// Run --init to set up everything properly
	initCmd := exec.Command(fhBinary, "--init")
	initCmd.Env = []string{
		"HOME=" + tempDir,
		"SHELL=/bin/bash",
		"PATH=" + os.Getenv("PATH"),
	}
	output, err := initCmd.CombinedOutput()
	require.NoError(t, err, "init should succeed: %s", output)

	dbPath := filepath.Join(tempDir, ".fh", "history.db")

	// Save a command
	cmd := exec.Command(fhBinary, "--save",
		"--cmd", "echo 'test command'",
		"--exit-code", "0",
		"--duration", "100",
	)
	cmd.Env = []string{
		"HOME=" + tempDir,
		"PATH=" + os.Getenv("PATH"),
	}

	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "save should succeed: %s", output)

	// Give it a moment to write
	time.Sleep(100 * time.Millisecond)

	// Open database and verify command was saved
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	entries, err := db.Query(storage.QueryFilters{Limit: 10})
	require.NoError(t, err)
	require.Len(t, entries, 1, "should have exactly 1 entry")

	entry := entries[0]
	assert.Equal(t, "echo 'test command'", entry.Command)
	assert.Equal(t, 0, entry.ExitCode)
	assert.Equal(t, int64(100), entry.DurationMs)
}

// TestSaveWithSpecialCharacters tests saving commands with special characters
func TestSaveWithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	fhBinary := buildFhBinary(t)

	// Run --init
	initCmd := exec.Command(fhBinary, "--init")
	initCmd.Env = []string{
		"HOME=" + tempDir,
		"SHELL=/bin/bash",
		"PATH=" + os.Getenv("PATH"),
	}
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, ".fh", "history.db")

	testCases := []struct {
		name    string
		command string
	}{
		{"single quotes", "echo 'test'"},
		{"double quotes", `echo "test"`},
		{"pipes", "ls | grep test"},
		{"redirects", "echo test > /tmp/file"},
		{"ampersand", "echo test && echo done"},
		{"semicolon", "echo test; echo done"},
		{"dollar sign", "echo $HOME"},
		{"backticks", "echo `date`"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(fhBinary, "--save",
				"--cmd", tc.command,
				"--exit-code", "0",
				"--duration", "0",
			)
			cmd.Env = []string{
				"HOME=" + tempDir,
				"PATH=" + os.Getenv("PATH"),
			}

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "save should succeed for %s: %s", tc.command, output)
		})
	}

	time.Sleep(200 * time.Millisecond)

	// Open database and verify all commands were saved
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	entries, err := db.Query(storage.QueryFilters{Limit: 100})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), len(testCases), "should save all special character commands")
}

// TestRapidSaves tests multiple rapid --save operations
func TestRapidSaves(t *testing.T) {
	tempDir := t.TempDir()
	fhBinary := buildFhBinary(t)

	// Run --init
	initCmd := exec.Command(fhBinary, "--init")
	initCmd.Env = []string{
		"HOME=" + tempDir,
		"SHELL=/bin/bash",
		"PATH=" + os.Getenv("PATH"),
	}
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, ".fh", "history.db")

	// Save 20 commands rapidly
	for i := 0; i < 20; i++ {
		cmd := exec.Command(fhBinary, "--save",
			"--cmd", "rapid command "+string(rune('0'+i)),
			"--exit-code", "0",
			"--duration", "0",
		)
		cmd.Env = []string{
			"HOME=" + tempDir,
			"PATH=" + os.Getenv("PATH"),
		}

		// Run in background (don't wait)
		err := cmd.Start()
		require.NoError(t, err)
	}

	// Wait for all saves to complete
	time.Sleep(2 * time.Second)

	// Open database and verify most commands were saved
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	entries, err := db.Query(storage.QueryFilters{Limit: 100})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 15, "should save most rapid commands")

	t.Logf("Saved %d/20 rapid commands", len(entries))
}

// TestMetadataCapture tests that metadata is captured correctly
func TestMetadataCapture(t *testing.T) {
	tempDir := t.TempDir()
	fhBinary := buildFhBinary(t)

	// Run --init
	initCmd := exec.Command(fhBinary, "--init")
	initCmd.Env = []string{
		"HOME=" + tempDir,
		"SHELL=/bin/bash",
		"PATH=" + os.Getenv("PATH"),
	}
	_, err := initCmd.CombinedOutput()
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, ".fh", "history.db")

	// Save a command
	cmd := exec.Command(fhBinary, "--save",
		"--cmd", "test command",
		"--exit-code", "0",
		"--duration", "0",
	)
	cmd.Env = []string{
		"HOME=" + tempDir,
		"PATH=" + os.Getenv("PATH"),
	}

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "save should succeed: %s", output)

	time.Sleep(100 * time.Millisecond)

	// Open database and verify metadata was captured
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	entries, err := db.Query(storage.QueryFilters{Limit: 1})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.NotEmpty(t, entry.Hostname, "should capture hostname")
	assert.NotEmpty(t, entry.User, "should capture user")
	assert.NotEmpty(t, entry.Cwd, "should capture cwd")
	assert.Greater(t, entry.Timestamp, int64(0), "should have timestamp")
}

// TestHookIdempotency tests that running --init twice doesn't break things
func TestHookIdempotency(t *testing.T) {
	tempDir := t.TempDir()
	fhBinary := buildFhBinary(t)

	runInit := func() error {
		cmd := exec.Command(fhBinary, "--init")
		cmd.Env = []string{
			"HOME=" + tempDir,
			"SHELL=/bin/bash",
			"PATH=" + os.Getenv("PATH"),
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Init output: %s", output)
		}
		return err
	}

	// Run --init first time
	err := runInit()
	require.NoError(t, err, "first init should succeed")

	// Read the bash_profile
	bashProfile := filepath.Join(tempDir, ".bash_profile")
	content1, err := os.ReadFile(bashProfile)
	require.NoError(t, err)

	// Count how many times __fh_save appears
	count1 := strings.Count(string(content1), "__fh_save")

	// Run --init second time
	err = runInit()
	require.NoError(t, err, "second init should succeed")

	// Read bash_profile again
	content2, err := os.ReadFile(bashProfile)
	require.NoError(t, err)

	count2 := strings.Count(string(content2), "__fh_save")

	// Should have same number of __fh_save (not duplicated)
	assert.Equal(t, count1, count2, "hooks should not be duplicated")
}

// buildFhBinary builds the fh binary and returns its path
func buildFhBinary(t *testing.T) string {
	t.Helper()

	// Check if binary already exists
	binaryPath := "../../build/fh"
	if _, err := os.Stat(binaryPath); err == nil {
		abs, _ := filepath.Abs(binaryPath)
		return abs
	}

	// Build the binary
	t.Log("Building fh binary...")
	cmd := exec.Command("make", "build")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", output)
		t.Fatalf("failed to build fh: %v", err)
	}

	abs, err := filepath.Abs(binaryPath)
	require.NoError(t, err)
	return abs
}
