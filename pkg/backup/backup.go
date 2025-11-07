package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spideyz0r/fh/pkg/crypto"
)

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Path      string
	Filename  string
	Hostname  string
	Timestamp time.Time
	Size      int64
}

// Create creates an encrypted backup of the database
func Create(dbPath, backupDir, passphrase string) (*BackupInfo, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Generate backup filename: history-{hostname}-{timestamp}.db.enc
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("history-%s-%s.db.enc", hostname, timestamp)
	backupPath := filepath.Join(backupDir, filename)

	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Encrypt database file
	if err := crypto.EncryptFile(dbPath, backupPath, passphrase); err != nil {
		return nil, fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Get backup file info
	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	info := &BackupInfo{
		Path:      backupPath,
		Filename:  filename,
		Hostname:  hostname,
		Timestamp: time.Now(),
		Size:      stat.Size(),
	}

	return info, nil
}

// List returns all backup files in the backup directory, sorted by timestamp (newest first)
func List(backupDir string) ([]*BackupInfo, error) {
	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []*BackupInfo{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .db.enc files
		if !strings.HasSuffix(entry.Name(), ".db.enc") {
			continue
		}

		// Parse filename: history-{hostname}-{timestamp}.db.enc
		info, err := parseBackupFilename(entry.Name())
		if err != nil {
			// Skip files that don't match expected format
			continue
		}

		// Get full path
		info.Path = filepath.Join(backupDir, entry.Name())

		// Get file size
		fileInfo, err := entry.Info()
		if err == nil {
			info.Size = fileInfo.Size()
		}

		backups = append(backups, info)
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// parseBackupFilename parses a backup filename and extracts metadata
// Expected format: history-{hostname}-{timestamp}.db.enc
// Example: history-macbook-20240101-120000.db.enc
func parseBackupFilename(filename string) (*BackupInfo, error) {
	// Remove .db.enc suffix
	name := strings.TrimSuffix(filename, ".db.enc")

	// Split by dashes
	parts := strings.SplitN(name, "-", 3)
	if len(parts) < 3 || parts[0] != "history" {
		return nil, fmt.Errorf("invalid backup filename format: %s", filename)
	}

	hostname := parts[1]
	timestampStr := parts[2]

	// Parse timestamp
	timestamp, err := time.Parse("20060102-150405", timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp in filename: %w", err)
	}

	return &BackupInfo{
		Filename:  filename,
		Hostname:  hostname,
		Timestamp: timestamp,
	}, nil
}

// Rotate removes old backups, keeping only the N most recent
func Rotate(backupDir string, keepCount int) error {
	if keepCount <= 0 {
		// 0 or negative means keep all backups
		return nil
	}

	backups, err := List(backupDir)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// If we have fewer backups than keepCount, nothing to delete
	if len(backups) <= keepCount {
		return nil
	}

	// Delete oldest backups
	toDelete := backups[keepCount:]
	for _, backup := range toDelete {
		if err := os.Remove(backup.Path); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", backup.Filename, err)
		}
	}

	return nil
}

// Restore decrypts a backup and restores it to the specified destination
func Restore(backupPath, dstPath, passphrase string) error {
	// Decrypt backup file
	if err := crypto.DecryptFile(backupPath, dstPath, passphrase); err != nil {
		return fmt.Errorf("failed to decrypt backup: %w", err)
	}

	return nil
}

// FormatSize formats a file size in human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
