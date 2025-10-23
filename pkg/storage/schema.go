package storage

// HistoryEntry represents a single command in the history
type HistoryEntry struct {
	ID         int64  `db:"id"`
	Timestamp  int64  `db:"timestamp"`
	Command    string `db:"command"`
	Cwd        string `db:"cwd"`
	ExitCode   int    `db:"exit_code"`
	Hostname   string `db:"hostname"`
	User       string `db:"user"`
	Shell      string `db:"shell"`
	DurationMs int64  `db:"duration_ms"`
	GitBranch  string `db:"git_branch"`
	Hash       string `db:"hash"` // Can be empty for KeepAll strategy
	SessionID  string `db:"session_id"`
}

// Schema versions for migration tracking
const (
	SchemaVersion1 = 1
	CurrentSchema  = SchemaVersion1
)

// SQL schema for version 1
const schemaV1 = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    command TEXT NOT NULL,
    cwd TEXT,
    exit_code INTEGER,
    hostname TEXT,
    user TEXT,
    shell TEXT,
    duration_ms INTEGER,
    git_branch TEXT,
    hash TEXT UNIQUE,
    session_id TEXT,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_timestamp ON history(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_command ON history(command);
CREATE INDEX IF NOT EXISTS idx_hash ON history(hash);
CREATE INDEX IF NOT EXISTS idx_session ON history(session_id);
CREATE INDEX IF NOT EXISTS idx_cwd ON history(cwd);
`

// GetSchema returns the SQL schema for the given version
func GetSchema(version int) string {
	switch version {
	case SchemaVersion1:
		return schemaV1
	default:
		return ""
	}
}
