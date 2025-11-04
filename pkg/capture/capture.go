package capture

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
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

// Collect gathers metadata about the command execution
func Collect(command string, exitCode int, durationMs int64) (*Metadata, error) {
	meta := &Metadata{
		Command:    command,
		ExitCode:   exitCode,
		Timestamp:  time.Now().Unix(),
		DurationMs: durationMs,
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// Non-fatal, use empty string
		meta.Cwd = ""
	} else {
		meta.Cwd = cwd
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		meta.Hostname = "unknown"
	} else {
		meta.Hostname = hostname
	}

	// Get username
	currentUser, err := user.Current()
	if err != nil {
		meta.User = "unknown"
	} else {
		meta.User = currentUser.Username
	}

	// Get shell from environment
	shell := os.Getenv("SHELL")
	if shell == "" {
		meta.Shell = "unknown"
	} else {
		// Extract just the shell name (bash, zsh, etc.)
		meta.Shell = filepath.Base(shell)
	}

	// Try to detect git branch
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
