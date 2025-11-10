package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectShell(t *testing.T) {
	t.Run("detect bash", func(t *testing.T) {
		oldShell := os.Getenv("SHELL")
		defer os.Setenv("SHELL", oldShell)

		os.Setenv("SHELL", "/bin/bash")
		shell, err := DetectShell()
		require.NoError(t, err)
		assert.Equal(t, ShellBash, shell)
	})

	t.Run("detect zsh", func(t *testing.T) {
		oldShell := os.Getenv("SHELL")
		defer os.Setenv("SHELL", oldShell)

		os.Setenv("SHELL", "/usr/bin/zsh")
		shell, err := DetectShell()
		require.NoError(t, err)
		assert.Equal(t, ShellZsh, shell)
	})

	t.Run("detect fish", func(t *testing.T) {
		oldShell := os.Getenv("SHELL")
		defer os.Setenv("SHELL", oldShell)

		os.Setenv("SHELL", "/usr/local/bin/fish")
		shell, err := DetectShell()
		require.NoError(t, err)
		assert.Equal(t, ShellFish, shell)
	})

	t.Run("no SHELL environment variable", func(t *testing.T) {
		oldShell := os.Getenv("SHELL")
		defer os.Setenv("SHELL", oldShell)

		os.Unsetenv("SHELL")
		_, err := DetectShell()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SHELL environment variable not set")
	})

	t.Run("unsupported shell", func(t *testing.T) {
		oldShell := os.Getenv("SHELL")
		defer os.Setenv("SHELL", oldShell)

		os.Setenv("SHELL", "/bin/ksh")
		_, err := DetectShell()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported shell")
	})
}

func TestGetHookContent(t *testing.T) {
	t.Run("get bash hook", func(t *testing.T) {
		content, err := GetHookContent(ShellBash)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, content, "bash")
	})

	t.Run("get zsh hook", func(t *testing.T) {
		content, err := GetHookContent(ShellZsh)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, content, "zsh")
	})

	t.Run("fish not supported", func(t *testing.T) {
		_, err := GetHookContent(ShellFish)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fish shell not yet supported")
	})

	t.Run("unsupported shell type", func(t *testing.T) {
		_, err := GetHookContent(ShellType("unknown"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported shell")
	})
}

func TestGetRCFile(t *testing.T) {
	t.Run("get bash RC file", func(t *testing.T) {
		// Create a temporary home directory
		tempHome := t.TempDir()
		oldHome := os.Getenv("HOME")
		defer os.Setenv("HOME", oldHome)
		os.Setenv("HOME", tempHome)

		// Create .bashrc
		bashrc := filepath.Join(tempHome, ".bashrc")
		err := os.WriteFile(bashrc, []byte(""), 0644)
		require.NoError(t, err)

		rcFile, err := GetRCFile(ShellBash)
		require.NoError(t, err)
		assert.Equal(t, bashrc, rcFile)
	})

	t.Run("get bash profile when bashrc doesn't exist", func(t *testing.T) {
		tempHome := t.TempDir()
		oldHome := os.Getenv("HOME")
		defer os.Setenv("HOME", oldHome)
		os.Setenv("HOME", tempHome)

		// Don't create .bashrc, should return .bash_profile
		rcFile, err := GetRCFile(ShellBash)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tempHome, ".bash_profile"), rcFile)
	})

	t.Run("get zsh RC file", func(t *testing.T) {
		tempHome := t.TempDir()
		oldHome := os.Getenv("HOME")
		defer os.Setenv("HOME", oldHome)
		os.Setenv("HOME", tempHome)

		rcFile, err := GetRCFile(ShellZsh)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tempHome, ".zshrc"), rcFile)
	})

	t.Run("get zsh RC file with ZDOTDIR", func(t *testing.T) {
		tempHome := t.TempDir()
		tempZdot := t.TempDir()
		oldHome := os.Getenv("HOME")
		oldZdot := os.Getenv("ZDOTDIR")
		defer os.Setenv("HOME", oldHome)
		defer os.Setenv("ZDOTDIR", oldZdot)

		os.Setenv("HOME", tempHome)
		os.Setenv("ZDOTDIR", tempZdot)

		rcFile, err := GetRCFile(ShellZsh)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tempZdot, ".zshrc"), rcFile)
	})

	t.Run("unsupported shell", func(t *testing.T) {
		_, err := GetRCFile(ShellType("unknown"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported shell")
	})
}

func TestIsHookInstalled(t *testing.T) {
	t.Run("hook is installed", func(t *testing.T) {
		tempDir := t.TempDir()
		rcFile := filepath.Join(tempDir, ".bashrc")

		content := `# fh - Fast History
some hook content here
`
		err := os.WriteFile(rcFile, []byte(content), 0644)
		require.NoError(t, err)

		installed, err := IsHookInstalled(rcFile)
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("hook is not installed", func(t *testing.T) {
		tempDir := t.TempDir()
		rcFile := filepath.Join(tempDir, ".bashrc")

		content := `export PATH=$PATH:/usr/local/bin
alias ll='ls -la'
`
		err := os.WriteFile(rcFile, []byte(content), 0644)
		require.NoError(t, err)

		installed, err := IsHookInstalled(rcFile)
		require.NoError(t, err)
		assert.False(t, installed)
	})

	t.Run("RC file does not exist", func(t *testing.T) {
		installed, err := IsHookInstalled("/nonexistent/file")
		require.NoError(t, err)
		assert.False(t, installed)
	})
}

func TestInstallHook(t *testing.T) {
	t.Run("install bash hook", func(t *testing.T) {
		tempDir := t.TempDir()
		rcFile := filepath.Join(tempDir, ".bashrc")

		// Create initial RC file
		initialContent := "export PATH=$PATH:/usr/local/bin\n"
		err := os.WriteFile(rcFile, []byte(initialContent), 0644)
		require.NoError(t, err)

		result, err := InstallHook(ShellBash, rcFile)
		require.NoError(t, err)
		assert.True(t, result.Installed)
		assert.Equal(t, rcFile, result.RCFile)
		assert.Equal(t, rcFile+".fh.backup", result.BackupFile)

		// Verify hook was installed
		content, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "# fh - Fast History")
		assert.Contains(t, string(content), initialContent)

		// Verify backup was created
		backupContent, err := os.ReadFile(result.BackupFile)
		require.NoError(t, err)
		assert.Equal(t, initialContent, string(backupContent))
	})

	t.Run("install hook when already installed", func(t *testing.T) {
		tempDir := t.TempDir()
		rcFile := filepath.Join(tempDir, ".bashrc")

		// Create RC file with hook already installed
		content := `export PATH=$PATH:/usr/local/bin
# fh - Fast History
hook content here
`
		err := os.WriteFile(rcFile, []byte(content), 0644)
		require.NoError(t, err)

		result, err := InstallHook(ShellBash, rcFile)
		require.NoError(t, err)
		assert.False(t, result.Installed)
		assert.Equal(t, rcFile, result.RCFile)

		// Verify no changes were made
		newContent, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(newContent))
	})

	t.Run("install hook creates RC file if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		rcFile := filepath.Join(tempDir, ".bashrc")

		result, err := InstallHook(ShellBash, rcFile)
		require.NoError(t, err)
		assert.True(t, result.Installed)

		// Verify RC file was created with hook
		_, err = os.Stat(rcFile)
		require.NoError(t, err)

		content, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "# fh - Fast History")
	})

	t.Run("install zsh hook", func(t *testing.T) {
		tempDir := t.TempDir()
		rcFile := filepath.Join(tempDir, ".zshrc")

		err := os.WriteFile(rcFile, []byte(""), 0644)
		require.NoError(t, err)

		result, err := InstallHook(ShellZsh, rcFile)
		require.NoError(t, err)
		assert.True(t, result.Installed)

		content, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "# fh - Fast History")
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("copy file successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "source.txt")
		dst := filepath.Join(tempDir, "dest.txt")

		content := "test content\nline 2\n"
		err := os.WriteFile(src, []byte(content), 0644)
		require.NoError(t, err)

		err = copyFile(src, dst)
		require.NoError(t, err)

		dstContent, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, content, string(dstContent))
	})

	t.Run("copy non-existent file", func(t *testing.T) {
		tempDir := t.TempDir()
		dst := filepath.Join(tempDir, "dest.txt")

		err := copyFile("/nonexistent/file", dst)
		assert.Error(t, err)
	})
}
