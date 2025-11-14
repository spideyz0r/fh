package storage

import (
	"database/sql"
	"fmt"
)

// Store defines the interface for history storage operations
type Store interface {
	Insert(entry *HistoryEntry) error
	Query(filters QueryFilters) ([]*HistoryEntry, error)
	GetByID(id int64) (*HistoryEntry, error)
	Count() (int64, error)
	Delete(id int64) error
	Close() error
}

// QueryFilters defines filters for querying history
type QueryFilters struct {
	Search   string // Text search in command
	Cwd      string // Filter by directory
	After    int64  // After timestamp
	Before   int64  // Before timestamp
	ExitCode *int   // Filter by exit code
	Limit    int    // Max results
	Offset   int    // Pagination offset
	Distinct bool   // Only return unique commands (most recent entry for each)
}

// Insert adds a new history entry to the database
func (db *DB) Insert(entry *HistoryEntry) error {
	query := `
		INSERT INTO history (
			timestamp, command, cwd, exit_code, hostname,
			user, shell, duration_ms, git_branch, hash, session_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		entry.Hash,
		entry.SessionID,
	)

	if err != nil {
		return fmt.Errorf("failed to insert entry: %w", err)
	}

	return nil
}

// Query retrieves history entries matching the given filters
func (db *DB) Query(filters QueryFilters) ([]*HistoryEntry, error) {
	var query string
	args := []interface{}{}

	if filters.Distinct {
		// Use subquery to get only unique commands (most recent entry for each)
		query = `SELECT h.id, h.timestamp, h.command, h.cwd, h.exit_code, h.hostname, h.user, h.shell, h.duration_ms, h.git_branch, h.hash, h.session_id, h.created_at
		FROM history h
		INNER JOIN (
			SELECT command, MAX(timestamp) as max_ts, MAX(id) as max_id
			FROM history
			WHERE 1=1`

		// Apply filters to subquery
		if filters.Search != "" {
			query += " AND command LIKE ?"
			args = append(args, "%"+filters.Search+"%")
		}

		if filters.Cwd != "" {
			query += " AND cwd = ?"
			args = append(args, filters.Cwd)
		}

		if filters.After > 0 {
			query += " AND timestamp >= ?"
			args = append(args, filters.After)
		}

		if filters.Before > 0 {
			query += " AND timestamp <= ?"
			args = append(args, filters.Before)
		}

		if filters.ExitCode != nil {
			query += " AND exit_code = ?"
			args = append(args, *filters.ExitCode)
		}

		query += `
			GROUP BY command
		) latest ON h.command = latest.command AND h.timestamp = latest.max_ts AND h.id = latest.max_id
		ORDER BY h.timestamp DESC`
	} else {
		// Standard query - return all entries
		query = "SELECT id, timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id, created_at FROM history WHERE 1=1"

		// Build WHERE clause
		if filters.Search != "" {
			query += " AND command LIKE ?"
			args = append(args, "%"+filters.Search+"%")
		}

		if filters.Cwd != "" {
			query += " AND cwd = ?"
			args = append(args, filters.Cwd)
		}

		if filters.After > 0 {
			query += " AND timestamp >= ?"
			args = append(args, filters.After)
		}

		if filters.Before > 0 {
			query += " AND timestamp <= ?"
			args = append(args, filters.Before)
		}

		if filters.ExitCode != nil {
			query += " AND exit_code = ?"
			args = append(args, *filters.ExitCode)
		}

		// Order by timestamp descending (most recent first)
		query += " ORDER BY timestamp DESC"
	}

	// Pagination (applies to both queries)
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

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

// GetByID retrieves a single history entry by ID
func (db *DB) GetByID(id int64) (*HistoryEntry, error) {
	query := "SELECT id, timestamp, command, cwd, exit_code, hostname, user, shell, duration_ms, git_branch, hash, session_id, created_at FROM history WHERE id = ?"

	entry := &HistoryEntry{}
	var createdAt int64
	var hash sql.NullString

	err := db.conn.QueryRow(query, id).Scan(
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

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	if hash.Valid {
		entry.Hash = hash.String
	}

	return entry, nil
}

// Count returns the total number of history entries
func (db *DB) Count() (int64, error) {
	var count int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM history").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count entries: %w", err)
	}
	return count, nil
}

// Delete removes a history entry by ID
func (db *DB) Delete(id int64) error {
	result, err := db.conn.Exec("DELETE FROM history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entry not found")
	}

	return nil
}

// DeleteByFilter removes history entries matching filters
func (db *DB) DeleteByFilter(filters QueryFilters) (int64, error) {
	query := "DELETE FROM history WHERE 1=1"
	args := []interface{}{}

	// Build WHERE clause (same as Query)
	if filters.Search != "" {
		query += " AND command LIKE ?"
		args = append(args, "%"+filters.Search+"%")
	}

	if filters.Cwd != "" {
		query += " AND cwd = ?"
		args = append(args, filters.Cwd)
	}

	if filters.After > 0 {
		query += " AND timestamp >= ?"
		args = append(args, filters.After)
	}

	if filters.Before > 0 {
		query += " AND timestamp <= ?"
		args = append(args, filters.Before)
	}

	if filters.ExitCode != nil {
		query += " AND exit_code = ?"
		args = append(args, *filters.ExitCode)
	}

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
