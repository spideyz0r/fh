package capture

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed shell/bash.sh
var bashHook string

//go:embed shell/zsh.sh
var zshHook string

// ShellType represents the type of shell
type ShellType string

const (
	ShellBash ShellType = "bash"
	ShellZsh  ShellType = "zsh"
	ShellFish ShellType = "fish"
)

// DetectShell detects the current shell from environment
func DetectShell() (ShellType, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "", fmt.Errorf("SHELL environment variable not set")
	}

	// Extract shell name from path
	shellName := filepath.Base(shell)

	switch shellName {
	case "bash":
		return ShellBash, nil
	case "zsh":
		return ShellZsh, nil
	case "fish":
		return ShellFish, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}
}

// GetHookContent returns the shell hook content for the given shell type
func GetHookContent(shell ShellType) (string, error) {
	switch shell {
	case ShellBash:
		return bashHook, nil
	case ShellZsh:
		return zshHook, nil
	case ShellFish:
		return "", fmt.Errorf("fish shell not yet supported")
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

// GetRCFile returns the RC file path for the given shell type
func GetRCFile(shell ShellType) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch shell {
	case ShellBash:
		// Try .bashrc first, then .bash_profile
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc, nil
		}
		return filepath.Join(home, ".bash_profile"), nil

	case ShellZsh:
		// For zsh, check ZDOTDIR first
		zdotdir := os.Getenv("ZDOTDIR")
		if zdotdir != "" {
			return filepath.Join(zdotdir, ".zshrc"), nil
		}
		return filepath.Join(home, ".zshrc"), nil

	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

// IsHookInstalled checks if fh hook is already installed in the RC file
func IsHookInstalled(rcFile string) (bool, error) {
	content, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read RC file: %w", err)
	}

	// Check for fh marker
	return strings.Contains(string(content), "# fh - Fast History"), nil
}

// HookInstallResult contains information about the hook installation
type HookInstallResult struct {
	RCFile     string // Path to the RC file that was modified
	BackupFile string // Path to the backup file
	Installed  bool   // Whether the hook was newly installed
}

// InstallHook installs the fh hook into the RC file
func InstallHook(shell ShellType, rcFile string) (*HookInstallResult, error) {
	result := &HookInstallResult{
		RCFile: rcFile,
	}

	// Check if already installed
	installed, err := IsHookInstalled(rcFile)
	if err != nil {
		return nil, err
	}

	if installed {
		result.Installed = false
		return result, nil
	}

	// Get hook content
	hookContent, err := GetHookContent(shell)
	if err != nil {
		return nil, err
	}

	// Create RC file if it doesn't exist
	if _, err := os.Stat(rcFile); os.IsNotExist(err) {
		if err := os.WriteFile(rcFile, []byte{}, 0644); err != nil {
			return nil, fmt.Errorf("failed to create RC file: %w", err)
		}
	}

	// Backup RC file
	backupFile := rcFile + ".fh.backup"
	result.BackupFile = backupFile

	if err := copyFile(rcFile, backupFile); err != nil {
		return nil, fmt.Errorf("failed to backup RC file: %w", err)
	}

	// Append hook to RC file
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open RC file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Error closing file after writing hook
		}
	}()

	// Add newline before hook
	hookWithNewline := "\n" + hookContent + "\n"

	if _, err := f.WriteString(hookWithNewline); err != nil {
		return nil, fmt.Errorf("failed to write hook to RC file: %w", err)
	}

	result.Installed = true
	return result, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
