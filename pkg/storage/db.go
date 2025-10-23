package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
	path string
}

// Open opens or creates a SQLite database at the given path
func Open(path string) (*DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn: conn,
		path: path,
	}

	// Initialize database
	if err := db.initialize(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

// initialize sets up the database schema and configuration
func (db *DB) initialize() error {
	// Enable WAL mode for better concurrency
	if _, err := db.conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := db.migrate(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// migrate applies database migrations
func (db *DB) migrate() error {
	// Get current schema version
	currentVersion, err := db.getSchemaVersion()
	if err != nil {
		return err
	}

	// Apply migrations if needed
	if currentVersion < CurrentSchema {
		return db.applyMigrations(currentVersion, CurrentSchema)
	}

	return nil
}

// getSchemaVersion returns the current schema version
func (db *DB) getSchemaVersion() (int, error) {
	// Check if schema_version table exists
	var tableExists bool
	err := db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='schema_version'
		)
	`).Scan(&tableExists)
	if err != nil {
		return 0, err
	}

	if !tableExists {
		return 0, nil
	}

	// Get latest version
	var version int
	err = db.conn.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return version, nil
}

// applyMigrations applies all migrations from 'from' to 'to' version
func (db *DB) applyMigrations(from, to int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for version := from + 1; version <= to; version++ {
		schema := GetSchema(version)
		if schema == "" {
			return fmt.Errorf("no schema found for version %d", version)
		}

		// Execute schema
		if _, err := tx.Exec(schema); err != nil {
			return fmt.Errorf("failed to apply schema v%d: %w", version, err)
		}

		// Record migration
		if _, err := tx.Exec(
			"INSERT INTO schema_version (version, applied_at) VALUES (?, strftime('%s', 'now'))",
			version,
		); err != nil {
			return fmt.Errorf("failed to record migration v%d: %w", version, err)
		}
	}

	return tx.Commit()
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}
