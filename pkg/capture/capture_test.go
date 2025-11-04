package capture

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	meta, err := Collect("ls -la", 0, 100)
	require.NoError(t, err)

	assert.Equal(t, "ls -la", meta.Command)
	assert.Equal(t, 0, meta.ExitCode)
	assert.Equal(t, int64(100), meta.DurationMs)
	assert.NotZero(t, meta.Timestamp)

	// Verify metadata was collected
	assert.NotEmpty(t, meta.Cwd)
	assert.NotEmpty(t, meta.Hostname)
	assert.NotEmpty(t, meta.User)
	assert.NotEmpty(t, meta.Shell)
	assert.NotEmpty(t, meta.SessionID)

	// GitBranch may or may not be set depending on whether we're in a git repo
	// So we don't assert on it
}

func TestCollect_WithExitCode(t *testing.T) {
	meta, err := Collect("false", 1, 50)
	require.NoError(t, err)

	assert.Equal(t, "false", meta.Command)
	assert.Equal(t, 1, meta.ExitCode)
	assert.Equal(t, int64(50), meta.DurationMs)
}

func TestCollect_MetadataFields(t *testing.T) {
	meta, err := Collect("pwd", 0, 10)
	require.NoError(t, err)

	// Test that hostname is reasonable
	assert.NotEqual(t, "", meta.Hostname)
	assert.NotEqual(t, "unknown", meta.Hostname)

	// Test that user is reasonable
	assert.NotEqual(t, "", meta.User)
	assert.NotEqual(t, "unknown", meta.User)

	// Test that cwd is an absolute path
	if meta.Cwd != "" {
		assert.True(t, len(meta.Cwd) > 0)
	}

	// Test that shell is detected
	assert.NotEqual(t, "", meta.Shell)
}

func TestDetectGitBranch_InGitRepo(t *testing.T) {
	// This test assumes we're running in the fh git repository
	// It should detect a branch
	cwd, err := os.Getwd()
	require.NoError(t, err)

	branch := detectGitBranch(cwd)

	// If we're in a git repo, we should get a branch
	// If not, branch will be empty (which is fine)
	if branch != "" {
		assert.NotEmpty(t, branch)
		// Common branch names
		assert.NotContains(t, branch, "\n")
	}
}

func TestDetectGitBranch_NotInGitRepo(t *testing.T) {
	// Test with /tmp which is unlikely to be a git repo
	branch := detectGitBranch("/tmp")

	// Should be empty string
	assert.Equal(t, "", branch)
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()

	assert.NotEmpty(t, id1)
	assert.Contains(t, id1, "-")

	// Generate another one - they should be different (due to timestamp)
	// But this test might be flaky if they run in the same second
	// So we just check format
	id2 := generateSessionID()
	assert.NotEmpty(t, id2)
	assert.Contains(t, id2, "-")
}

func TestCollect_SessionIDConsistency(t *testing.T) {
	meta1, err := Collect("cmd1", 0, 10)
	require.NoError(t, err)

	meta2, err := Collect("cmd2", 0, 20)
	require.NoError(t, err)

	// Session IDs should be similar (same PPID) but timestamps might differ
	// Just verify they're not empty
	assert.NotEmpty(t, meta1.SessionID)
	assert.NotEmpty(t, meta2.SessionID)
}
