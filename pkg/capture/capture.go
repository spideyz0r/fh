package capture

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Cache for metadata that doesn't change
var (
	metadataMutex  sync.Once
	cachedHostname string
	cachedUser     string
	cachedShell    string
)

// Metadata contains information about the command execution environment
type Metadata struct {
	Command    string
	ExitCode   int
	Cwd        string
	Hostname   string
	User       string
	Shell      string
	Timestamp  int64
	DurationMs int64
	GitBranch  string
	SessionID  string
}

// initMetadataCache initializes the cached metadata that doesn't change
func initMetadataCache() {
	// Get hostname (doesn't change)
	hostname, err := os.Hostname()
	if err != nil {
		cachedHostname = "unknown"
	} else {
		cachedHostname = hostname
	}

	// Get username (doesn't change)
	currentUser, err := user.Current()
	if err != nil {
		cachedUser = "unknown"
	} else {
		cachedUser = currentUser.Username
	}

	// Get shell from environment (doesn't change within a session)
	shell := os.Getenv("SHELL")
	if shell == "" {
		cachedShell = "unknown"
	} else {
		// Extract just the shell name (bash, zsh, etc.)
		cachedShell = filepath.Base(shell)
	}
}

// Collect gathers metadata about the command execution
func Collect(command string, exitCode int, durationMs int64) (*Metadata, error) {
	// Initialize cache once
	metadataMutex.Do(initMetadataCache)

	meta := &Metadata{
		Command:    command,
		ExitCode:   exitCode,
		Timestamp:  time.Now().Unix(),
		DurationMs: durationMs,
		Hostname:   cachedHostname,
		User:       cachedUser,
		Shell:      cachedShell,
	}

	// Get current working directory (this can change)
	cwd, err := os.Getwd()
	if err != nil {
		// Non-fatal, use empty string
		meta.Cwd = ""
	} else {
		meta.Cwd = cwd
	}

	// Try to detect git branch (can change)
	meta.GitBranch = detectGitBranch(meta.Cwd)

	// Generate session ID from shell PID and start time
	meta.SessionID = generateSessionID()

	return meta, nil
}

// detectGitBranch tries to detect the current git branch
func detectGitBranch(cwd string) string {
	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	if cwd != "" {
		cmd.Dir = cwd
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	branch := strings.TrimSpace(string(output))
	return branch
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	// Use shell PID if available (PPID), otherwise use our PID
	ppid := os.Getppid()
	if ppid == 0 {
		ppid = os.Getpid()
	}

	// Include timestamp to make it more unique
	return fmt.Sprintf("%d-%d", ppid, time.Now().Unix())
}
