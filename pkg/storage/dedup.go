package storage

import (
	"database/sql"
	"fmt"
)

// DedupStrategy defines how to handle duplicate commands
type DedupStrategy string

const (
	// KeepFirst keeps the first occurrence and ignores duplicates
	KeepFirst DedupStrategy = "keep_first"

	// KeepLast updates the timestamp of the existing entry
	KeepLast DedupStrategy = "keep_last"

	// KeepAll allows all duplicates (no deduplication)
	KeepAll DedupStrategy = "keep_all"
)

// DedupConfig holds deduplication configuration
type DedupConfig struct {
	Enabled  bool
	Strategy DedupStrategy
}

// InsertWithDedup inserts an entry with deduplication logic
func (db *DB) InsertWithDedup(entry *HistoryEntry, config DedupConfig) error {
	// If deduplication is disabled, insert normally
	if !config.Enabled {
		return db.Insert(entry)
	}

	// Generate hash if not already set
	if entry.Hash == "" {
		entry.Hash = GenerateHash(entry.Command)
	}

	// Check if entry with same hash exists
	exists, existingID, err := db.checkHashExists(entry.Hash)
	if err != nil {
		return fmt.Errorf("failed to check for duplicates: %w", err)
	}

	if !exists {
		// No duplicate, insert normally
		return db.Insert(entry)
	}

	// Handle duplicate based on strategy
	switch config.Strategy {
	case KeepFirst:
		// Ignore the new entry
		return nil

	case KeepLast:
		// Update the existing entry's timestamp
		return db.updateEntryTimestamp(existingID, entry.Timestamp)

	case KeepAll:
		// Allow duplicate by removing hash constraint temporarily
		// This is a special case - we'll insert without hash uniqueness
		return db.insertWithoutHashCheck(entry)

	default:
		return fmt.Errorf("unknown deduplication strategy: %s", config.Strategy)
	}
}

// checkHashExists checks if an entry with the given hash exists
func (db *DB) checkHashExists(hash string) (bool, int64, error) {
	var id int64
	err := db.conn.QueryRow("SELECT id FROM history WHERE hash = ?", hash).Scan(&id)

	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	return true, id, nil
}

// updateEntryTimestamp updates the timestamp of an existing entry
func (db *DB) updateEntryTimestamp(id int64, timestamp int64) error {
	_, err := db.conn.Exec(
		"UPDATE history SET timestamp = ? WHERE id = ?",
		timestamp, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update timestamp: %w", err)
	}
	return nil
}

// insertWithoutHashCheck inserts an entry, allowing duplicate hashes
// This is used for KeepAll strategy
func (db *DB) insertWithoutHashCheck(entry *HistoryEntry) error {
	// Insert without hash to bypass UNIQUE constraint
	query := `
		INSERT INTO history (
			timestamp, command, cwd, exit_code, hostname,
			user, shell, duration_ms, git_branch, session_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(
		query,
		entry.Timestamp,
		entry.Command,
		entry.Cwd,
		entry.ExitCode,
		entry.Hostname,
		entry.User,
		entry.Shell,
		entry.DurationMs,
		entry.GitBranch,
		entry.SessionID,
	)

	if err != nil {
		return fmt.Errorf("failed to insert entry: %w", err)
	}

	return nil
}

// GetDuplicates returns all entries with duplicate commands
func (db *DB) GetDuplicates() ([]*HistoryEntry, error) {
	query := `
		SELECT h.id, h.timestamp, h.command, h.cwd, h.exit_code, h.hostname,
		       h.user, h.shell, h.duration_ms, h.git_branch, h.hash, h.session_id, h.created_at
		FROM history h
		INNER JOIN (
			SELECT hash
			FROM history
			WHERE hash IS NOT NULL
			GROUP BY hash
			HAVING COUNT(*) > 1
		) dups ON h.hash = dups.hash
		ORDER BY h.hash, h.timestamp DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query duplicates: %w", err)
	}
	defer rows.Close()

	var entries []*HistoryEntry
	for rows.Next() {
		entry := &HistoryEntry{}
		var createdAt int64
		var hash sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Command,
			&entry.Cwd,
			&entry.ExitCode,
			&entry.Hostname,
			&entry.User,
			&entry.Shell,
			&entry.DurationMs,
			&entry.GitBranch,
			&hash,
			&entry.SessionID,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}

		if hash.Valid {
			entry.Hash = hash.String
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entries, nil
}

// DeduplicateExisting removes duplicates from existing history
// Keeps the most recent entry for each unique command
func (db *DB) DeduplicateExisting() (int64, error) {
	// Delete all but the most recent entry for each hash
	query := `
		DELETE FROM history
		WHERE id NOT IN (
			SELECT MAX(id)
			FROM history
			WHERE hash IS NOT NULL
			GROUP BY hash
		) AND hash IS NOT NULL
	`

	result, err := db.conn.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to deduplicate: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
