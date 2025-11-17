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

// Shell type constants
const (
	// ShellBash represents Bash shell
	ShellBash ShellType = "bash"
	// ShellZsh represents Zsh shell
	ShellZsh ShellType = "zsh"
	// ShellFish represents Fish shell
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

// GetHookContent returns the shell hook content for the given shell type with keybinding
func GetHookContent(shell ShellType, keybinding string) (string, error) {
	var hookTemplate string

	switch shell {
	case ShellBash:
		hookTemplate = bashHook
	case ShellZsh:
		hookTemplate = zshHook
	case ShellFish:
		return "", fmt.Errorf("fish shell not yet supported")
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	// Convert keybinding name to display format and code
	display, code, err := parseKeybinding(shell, keybinding)
	if err != nil {
		return "", err
	}

	// Replace placeholders in template
	content := strings.ReplaceAll(hookTemplate, "{{KEYBINDING_DISPLAY}}", display)
	content = strings.ReplaceAll(content, "{{KEYBINDING_CODE}}", code)

	return content, nil
}

// parseKeybinding converts a keybinding name to display format and shell-specific code
// Supports format like "ctrl-r", "ctrl-g", "ctrl-f", etc.
func parseKeybinding(shell ShellType, keybinding string) (display string, code string, err error) {
	// Normalize to lowercase
	kb := strings.ToLower(strings.TrimSpace(keybinding))

	// Parse ctrl-X format
	if strings.HasPrefix(kb, "ctrl-") {
		key := strings.TrimPrefix(kb, "ctrl-")
		if len(key) != 1 {
			return "", "", fmt.Errorf("invalid keybinding format: %s (expected ctrl-X where X is a single letter)", keybinding)
		}

		// Display format: Ctrl-R, Ctrl-G, etc
		display = "Ctrl-" + strings.ToUpper(key)

		// Shell-specific code
		switch shell {
		case ShellBash:
			// Bash format: \C-r, \C-g, etc
			code = "\\C-" + key
		case ShellZsh:
			// Zsh format: ^R, ^G, etc
			code = "^" + strings.ToUpper(key)
		default:
			code = "\\C-" + key
		}

		return display, code, nil
	}

	return "", "", fmt.Errorf("unsupported keybinding format: %s (expected ctrl-X format like 'ctrl-r' or 'ctrl-g')", keybinding)
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
	RCFile           string // Path to the RC file that was modified
	BackupFile       string // Path to the backup file
	Installed        bool   // Whether the hook was newly installed
	KeybindingUpdate bool   // Whether the keybinding was updated
}

// InstallHook installs the fh hook into the RC file with the specified keybinding
func InstallHook(shell ShellType, rcFile string, keybinding string) (*HookInstallResult, error) {
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

		// Check if keybinding needs to be updated
		currentKeybinding, err := extractCurrentKeybinding(rcFile, shell)
		if err != nil {
			// If we can't extract the current keybinding, just return
			// (this might happen with old installations)
			return result, nil
		}

		// Normalize keybindings for comparison
		desired := strings.ToLower(strings.TrimSpace(keybinding))
		current := strings.ToLower(strings.TrimSpace(currentKeybinding))

		if desired != current {
			// Keybinding has changed, update it
			if err := updateKeybinding(rcFile, shell, keybinding); err != nil {
				return nil, fmt.Errorf("failed to update keybinding: %w", err)
			}
			result.KeybindingUpdate = true
		}

		return result, nil
	}

	// Get hook content with keybinding
	hookContent, err := GetHookContent(shell, keybinding)
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
		_ = f.Close()
	}()

	// Add newline before hook
	hookWithNewline := "\n" + hookContent + "\n"

	if _, err := f.WriteString(hookWithNewline); err != nil {
		return nil, fmt.Errorf("failed to write hook to RC file: %w", err)
	}

	result.Installed = true
	return result, nil
}

// extractCurrentKeybinding extracts the current keybinding from the RC file
func extractCurrentKeybinding(rcFile string, shell ShellType) (string, error) {
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return "", fmt.Errorf("failed to read RC file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch shell {
		case ShellBash:
			// Look for: bind -x '"\C-r": __fh_widget'
			if strings.Contains(line, "bind -x") && strings.Contains(line, "__fh_widget") {
				// Extract \C-X from the line
				if idx := strings.Index(line, `"\C-`); idx != -1 {
					start := idx + 4 // Skip past "\C-
					if start < len(line) {
						key := string(line[start])
						return "ctrl-" + strings.ToLower(key), nil
					}
				}
			}
		case ShellZsh:
			// Look for: bindkey '^R' __fh_widget
			if strings.Contains(line, "bindkey") && strings.Contains(line, "__fh_widget") {
				// Extract ^X from the line
				if idx := strings.Index(line, "'^"); idx != -1 {
					start := idx + 2 // Skip past '^
					if start < len(line) {
						key := string(line[start])
						return "ctrl-" + strings.ToLower(key), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("keybinding not found in RC file")
}

// updateKeybinding updates the keybinding line in the RC file
func updateKeybinding(rcFile string, shell ShellType, keybinding string) error {
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return fmt.Errorf("failed to read RC file: %w", err)
	}

	// Get the new keybinding code
	_, newCode, err := parseKeybinding(shell, keybinding)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch shell {
		case ShellBash:
			// Look for: bind -x '"\C-r": __fh_widget'
			if strings.Contains(trimmed, "bind -x") && strings.Contains(trimmed, "__fh_widget") {
				lines[i] = fmt.Sprintf("bind -x '\"%s\": __fh_widget'", newCode)
				modified = true
			}
		case ShellZsh:
			// Look for: bindkey '^R' __fh_widget
			if strings.Contains(trimmed, "bindkey") && strings.Contains(trimmed, "__fh_widget") {
				lines[i] = fmt.Sprintf("bindkey '%s' __fh_widget", newCode)
				modified = true
			}
		}
	}

	if !modified {
		return fmt.Errorf("keybinding line not found in RC file")
	}

	// Write back to file
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(rcFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write RC file: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
