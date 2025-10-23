package storage

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// GenerateHash creates a SHA256 hash for a command
// This is used for deduplication
func GenerateHash(command string) string {
	// Normalize the command (trim whitespace)
	normalized := strings.TrimSpace(command)

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(normalized))

	// Return hex string
	return fmt.Sprintf("%x", hash)
}

// GenerateHashWithContext creates a hash including context
// This can be used for context-aware deduplication
func GenerateHashWithContext(command, cwd string) string {
	// Combine command and working directory
	combined := fmt.Sprintf("%s:%s", strings.TrimSpace(command), cwd)

	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}
