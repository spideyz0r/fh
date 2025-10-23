# fh - Design Document

## Overview

`fh` (pronounced "fast history" or "fuzzy history") is a modern replacement for shell history that provides:
- Fast, fuzzy search with FZF integration
- Deduplication of commands
- Rich metadata capture (cwd, exit codes, timestamps, git branch, etc.)
- Optional AI-powered semantic search
- Encrypted remote backups
- Support for bash and zsh

## Philosophy

**Keep it simple:**
- No persistent daemon/background process
- SQLite for storage (fast, reliable, serverless)
- Direct shell hook integration
- Minimal dependencies
- No Cobra - simple flag parsing for better performance and control

## Architecture

### Directory Structure

```
fh/
├── main.go                  # Simple flag parsing + dispatch
├── pkg/
│   ├── capture/
│   │   └── hook.go          # Generate shell hooks
│   ├── storage/
│   │   ├── db.go            # SQLite operations
│   │   └── import.go        # Import existing history
│   ├── search/
│   │   └── fzf.go           # FZF integration
│   ├── ai/
│   │   └── query.go         # AI context search
│   └── sync/
│       ├── sftp.go          # Remote backup
│       └── encrypt.go       # Encryption
├── shell/
│   ├── bash.sh              # Bash integration hooks
│   └── zsh.sh               # Zsh integration hooks
└── config/
    └── config.go            # Load ~/.fh/config.yaml
```

### Data Storage

**SQLite Database:** `~/.fh/history.db`

```sql
CREATE TABLE history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,           -- Unix timestamp
    command TEXT NOT NULL,                -- The actual command
    cwd TEXT,                             -- Working directory
    exit_code INTEGER,                    -- Command exit status
    hostname TEXT,                        -- Machine name
    user TEXT,                            -- Username
    shell TEXT,                           -- bash/zsh
    duration_ms INTEGER,                  -- Execution time (if measurable)
    git_branch TEXT,                      -- Git branch if in repo
    hash TEXT UNIQUE,                     -- SHA256 for deduplication
    session_id TEXT                       -- Shell session identifier
);

CREATE INDEX idx_timestamp ON history(timestamp DESC);
CREATE INDEX idx_command ON history(command);
CREATE INDEX idx_hash ON history(hash);
```

**WAL Mode:** Enabled for concurrent reads during writes

## CLI Interface

### Primary Usage Patterns

```bash
# Launch FZF interface (default)
fh

# Search with pre-filter
fh kubectl get pods

# Special modes (with flags)
fh --ask "what did I do yesterday for auth bug?"
fh --sync
fh --init
fh --stats
fh --export
fh --import
fh --restore
```

### Design Rationale

**No subcommands:** Most operations are searches, so make that the default
- No args → Launch FZF with full history
- Args without flags → FZF pre-filtered by query
- Args with `--flags` → Special operations

**Why `--` prefix?** Allows natural search queries without ambiguity:
```bash
fh kubectl           # Search for "kubectl"
fh --stats           # Show statistics
```

### Flag Reference

| Flag | Description |
|------|-------------|
| `--init` | Setup shell integration (modifies .bashrc/.zshrc) |
| `--save` | Internal: Save command to history (called by shell hook) |
| `--sync` | Upload encrypted backup to remote endpoint |
| `--restore` | Download and restore from remote backup |
| `--import` | Import existing ~/.bash_history or ~/.zsh_history |
| `--export` | Export history to file |
| `--stats` | Show history statistics |
| `--ask <query>` | AI-powered semantic search |
| `--config` | Show current configuration |
| `--help` | Show help message |

## Shell Integration

### Initialization

```bash
fh --init
```

This command:
1. Creates `~/.fh/` directory
2. Initializes SQLite database
3. Imports existing history
4. Adds hooks to `.bashrc` or `.zshrc`
5. Creates default `config.yaml`

### Bash Hook

Added to `~/.bashrc`:

```bash
# fh integration
__fh_save() {
    local exit_code=$?
    local cmd="$(HISTTIMEFORMAT='' history 1 | sed 's/^[ ]*[0-9]*[ ]*//')"

    # Run in background to avoid blocking prompt
    (fh --save \
        --cmd "$cmd" \
        --exit-code $exit_code \
        --cwd "$(pwd)" \
        --timestamp $(date +%s) \
        &)

    return $exit_code
}

if [[ "$PROMPT_COMMAND" != *__fh_save* ]]; then
    PROMPT_COMMAND="__fh_save; $PROMPT_COMMAND"
fi

# Bind Ctrl-R to fh
bind '"\C-r": "\C-e\C-u`fh`\e\C-e\er\C-m"'
```

### Zsh Hook

Added to `~/.zshrc`:

```zsh
# fh integration
__fh_save() {
    local exit_code=$?
    local cmd="${1}"

    # Run in background
    (fh --save \
        --cmd "$cmd" \
        --exit-code $exit_code \
        --cwd "$(pwd)" \
        --timestamp $(date +%s) \
        &)

    return $exit_code
}

precmd_functions+=(__fh_save)

# Bind Ctrl-R to fh
bindkey -s '^r' 'fh\n'
```

## Features

### 1. Deduplication

**Strategy:** Hash-based deduplication with context awareness

```go
// Generate hash from command + optional context
hash := sha256.Sum256([]byte(command))

// On insert:
// 1. Check if hash exists
// 2. If exists: update timestamp (or skip, configurable)
// 3. If new: insert
```

**Configuration options:**
- `deduplicate: true/false` - Enable/disable deduplication
- `deduplicate_strategy: "keep_first" | "keep_last" | "keep_all"` - How to handle duplicates

### 2. FZF Search

**Default behavior:**

```bash
fh [query]
```

**Implementation:**
1. Query SQLite for matching commands
2. Format results: `timestamp | cwd | command`
3. Pipe to FZF with options:
   - `--tac`: Reverse order (recent first)
   - `--no-sort`: Preserve chronological order
   - `--preview`: Show command details
   - `--multi`: Allow selecting multiple commands

**FZF Preview Window:**
```
Command:    kubectl get pods
Timestamp:  2025-10-23 14:32:15
Directory:  ~/projects/api
Exit Code:  0
Duration:   245ms
Git Branch: feature/auth
```

### 3. AI-Powered Search

**Usage:**

```bash
fh --ask "show me docker commands from last week when debugging the API"
```

**Implementation:**
1. Query relevant history window (e.g., last week)
2. Format commands with context
3. Send to LLM with prompt:
   ```
   Given this shell command history, answer the user's question.

   History:
   [timestamp] [cwd] [command]
   ...

   Question: {user_query}

   Return only the relevant commands.
   ```
4. Display results (optionally pipe to FZF)

**Supported Providers:**
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Local models (via API-compatible endpoints)

**Cost Management:**
- Configurable max tokens per query
- Cache frequent queries
- Option to limit history window sent to API

### 4. Remote Sync & Backup

**Upload:**

```bash
fh --sync
```

Process:
1. Create snapshot of `history.db`
2. Encrypt with configured key
3. Upload to remote endpoint
4. Filename: `history-{hostname}-{timestamp}.db.enc`

**Restore:**

```bash
fh --restore
```

Process:
1. List available backups from remote
2. User selects via FZF
3. Download and decrypt
4. Option to merge or replace local history

**Supported Protocols:**
- SFTP (primary, most common)
- SCP (SSH-based)
- Future: rsync, S3, WebDAV

**Encryption:**
- Algorithm: AES-256-GCM
- Key derivation: PBKDF2 from passphrase
- Key storage: `~/.fh/key` (encrypted) or system keychain

### 5. Statistics

```bash
fh --stats
```

Example output:
```
fh Statistics
===================
Total commands:        45,234
Unique commands:       12,456
Most used commands:
  1. git status        (2,345)
  2. kubectl get pods  (1,892)
  3. cd                (1,654)

Commands by day:
  Mon: ████████████ 3,456
  Tue: ██████████   2,987
  Wed: ███████████  3,123
  ...

Success rate:          94.2%
Average commands/day:  234
```

## Configuration

**Location:** `~/.fh/config.yaml`

```yaml
# Database location
database: ~/.fh/history.db

# Deduplication settings
deduplicate: true
deduplicate_strategy: keep_last  # keep_first, keep_last, keep_all

# Encrypt backups
encrypt_backups: true

# Sync/Backup configuration
sync:
  enabled: false
  protocol: sftp
  host: backup.example.com
  port: 22
  path: /backups/fh
  username: myuser
  key_file: ~/.ssh/id_rsa

# AI search configuration
ai:
  enabled: false
  provider: anthropic           # openai, anthropic, local
  api_key_env: ANTHROPIC_API_KEY
  model: claude-3-5-sonnet-20241022
  max_history_items: 1000       # Max items to send in context
  max_tokens: 4000

# Commands to ignore (regex patterns)
ignore_patterns:
  - "^ls$"
  - "^cd$"
  - "^pwd$"
  - "^exit$"
  - "^clear$"

# Capture settings
capture:
  save_duration: true           # Measure command duration
  save_git_branch: true         # Capture git branch
  async_save: true              # Save in background (recommended)
```

## Implementation Details

### Performance Considerations

**Critical Path (command capture):**
- Target: <10ms overhead per command
- Use background process for DB insert
- Batch inserts if needed (buffer N commands)
- WAL mode for non-blocking writes

**FZF Search:**
- Pre-query optimization with indexes
- Limit results to recent N (e.g., 10,000)
- Lazy load older history on demand

**Database Optimization:**
- Regular VACUUM on idle
- Analyze query patterns
- Index on frequently searched columns

### Metadata Enrichment

**Git Branch Detection:**
```bash
git_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
```

**Duration Measurement:**
```bash
# Bash: Use $SECONDS
start_time=$SECONDS
# ... command execution ...
duration=$((SECONDS - start_time))

# Zsh: Use hooks
preexec() { timer=$(($(date +%s%0N)/1000000)) }
precmd() {
    duration=$(($(date +%s%0N)/1000000 - timer))
}
```

### Import Existing History

```bash
fh --import
```

Process:
1. Detect shell (bash/zsh)
2. Parse history file format
3. Extract commands and timestamps (if available)
4. Deduplicate during import
5. Preserve original order

**Format Handling:**
- Bash: Simple newline-delimited or HISTTIMEFORMAT
- Zsh: Extended history format (`:timestamp:duration;command`)

## Development Roadmap

### MVP (v0.1.0)
- [ ] Shell integration (`--init`)
- [ ] Capture to SQLite (`--save`)
- [ ] FZF search (default behavior)
- [ ] Deduplication (hash-based)
- [ ] Import existing history (`--import`)
- [ ] Basic stats (`--stats`)

### v0.2.0
- [ ] SFTP sync (`--sync`)
- [ ] Backup encryption
- [ ] Restore functionality (`--restore`)
- [ ] Configuration file support

### v0.3.0
- [ ] AI-powered search (`--ask`)
- [ ] Multiple AI provider support
- [ ] Query cost management

### v1.0.0
- [ ] Fish shell support
- [ ] Advanced filtering (by date, cwd, exit code)
- [ ] Export formats (JSON, CSV)
- [ ] Multi-machine merge strategies
- [ ] Web UI (optional, view history in browser)

## Testing Strategy

**Unit Tests:**
- Storage layer (CRUD operations)
- Deduplication logic
- Encryption/decryption
- Config parsing

**Integration Tests:**
- Shell hook integration
- FZF interaction
- Sync workflow
- Import from native history

**Performance Tests:**
- Capture overhead (<10ms)
- Search speed (sub-second for 100k+ records)
- Database growth over time

## Security Considerations

**Sensitive Data:**
- Commands may contain passwords, tokens, API keys
- Option to ignore patterns (configured regexes)
- Encryption for remote backups
- Secure key storage

**Best Practices:**
```yaml
# Recommended ignore patterns
ignore_patterns:
  - ".*password.*"
  - ".*token.*"
  - ".*api[_-]?key.*"
  - "curl.*Authorization"
```

**Backup Security:**
- Always encrypt before uploading
- Use SSH keys (not passwords) for SFTP
- Option to exclude sensitive commands from backups

## Comparison with Existing Tools

| Feature | bash/zsh history | hishtory | fh |
|---------|------------------|----------|----------|
| FZF search | Manual binding | ✓ | ✓ |
| Deduplication | ✗ | ✓ | ✓ |
| Metadata | Limited | ✓ | ✓ |
| Sync | ✗ | ✓ (cloud) | ✓ (self-hosted) |
| AI search | ✗ | ✗ | ✓ |
| Encryption | ✗ | ✓ | ✓ |
| No daemon | ✓ | ✗ | ✓ |

**Key Differentiators:**
- No persistent daemon (simpler than hishtory)
- AI-powered semantic search
- Self-hosted sync (more privacy)
- Minimal dependencies (no Cobra)

## Future Ideas

**Community Sharing:**
- Share useful command snippets
- Public command library (opt-in)
- Learn from others' workflows

**Shell Completion:**
- Context-aware suggestions
- Learn from history patterns
- Predict next command

**Analytics:**
- Command patterns over time
- Productivity metrics
- Error rate trends

**Integration:**
- Export to note-taking tools
- Slack/Discord bot (query team's history)
- CI/CD pipeline insights
