# fh - Development Plan

## Project Scope

Build a modern shell history replacement in incremental phases, starting with core functionality and expanding to advanced features. Each phase should be fully tested, documented, and production-ready before moving to the next.

## Development Principles

1. **Test-Driven Development**: Write tests first, aim for >80% coverage
2. **Incremental Delivery**: Ship working software at each phase
3. **Documentation First**: Update README.md as features are added
4. **CI/CD from Day 1**: Automated testing and releases
5. **Backward Compatibility**: Each version should import/migrate from previous versions

---

## Phase 0: Project Foundation

**Goal**: Set up project structure, tooling, and CI/CD pipeline

### Tasks

#### 0.1 Project Initialization ✅
- [x] Initialize Go module (`go mod init github.com/spideyz0r/fh`)
- [x] Create directory structure (cmd/, pkg/, shell/, config/, test/)
- [x] Set up .gitignore for Go projects
- [x] Choose Go version (1.21+)
- [x] Add LICENSE file
- [x] Create initial README.md with project vision and installation placeholder

#### 0.2 Development Tooling ✅
- [x] Set up Makefile with common tasks:
  - [x] `make build` - Build binary
  - [x] `make test` - Run all tests
  - [x] `make coverage` - Generate coverage report
  - [x] `make lint` - Run linters
  - [x] `make install` - Install to $GOPATH/bin
  - [x] `make clean` - Clean build artifacts
- [x] Configure golangci-lint with sensible defaults
- [ ] Set up pre-commit hooks (optional but recommended)

#### 0.3 Testing Infrastructure ✅
- [x] Choose testing libraries:
  - [x] Standard `testing` package
  - [x] `github.com/stretchr/testify` for assertions
  - [x] `github.com/DATA-DOG/go-sqlmock` for database mocking
- [x] Create test helper utilities (pkg/testutil/)
- [x] Set up table-driven test patterns
- [x] Configure coverage reporting (codecov or coveralls)

#### 0.4 CI/CD Pipeline ✅
- [x] Create `.github/workflows/test.yml`:
  - [x] Run on: push, pull_request
  - [x] Test on multiple Go versions (1.21, 1.22, 1.23)
  - [x] Test on multiple OS (linux, macos)
  - [x] Upload coverage reports
  - [x] Fail if coverage drops below threshold (80%)
- [x] Create `.github/workflows/lint.yml`:
  - [x] Run golangci-lint
  - [x] Check formatting (gofmt)
  - [x] Check go mod tidy
- [x] Create `.github/workflows/release.yml`:
  - [x] Trigger on: tag push (v*.*.*)
  - [x] Use goreleaser for multi-platform builds
  - [x] Create GitHub release with binaries
  - [x] Generate changelog from commits
- [x] Configure goreleaser.yml:
  - [x] Build for: linux, darwin, windows
  - [x] Architectures: amd64, arm64
  - [x] Archive formats: tar.gz, zip
  - [x] Checksums and signatures

#### 0.5 Documentation ✅
- [x] Create README.md structure:
  - [x] Project description
  - [x] Features (will expand)
  - [x] Installation (placeholder)
  - [x] Quick start (placeholder)
  - [x] Usage (placeholder)
  - [x] Configuration (placeholder)
  - [x] Development (how to contribute)
  - [x] License
- [x] Create CONTRIBUTING.md:
  - [x] How to set up development environment
  - [x] How to run tests
  - [x] Code style guidelines
  - [x] PR process
- [ ] Create CODE_OF_CONDUCT.md (skipped)

**Deliverable**: Working project structure with CI/CD, ready for development

---

## Phase 1: Core Storage & Capture (MVP Foundation)

**Goal**: Implement SQLite storage and basic command capture without shell integration

### Tasks

#### 1.1 Database Schema & Migrations ✅
- [x] Design SQLite schema (pkg/storage/schema.go):
  ```go
  type HistoryEntry struct {
      ID        int64
      Timestamp int64
      Command   string
      Cwd       string
      ExitCode  int
      Hostname  string
      User      string
      Shell     string
      Hash      string
  }
  ```
- [x] Create migration system (simple version-based):
  - [x] Schema version tracking table
  - [x] Migration functions (v1, v2, etc.)
  - [x] Automatic migration on DB open
- [x] Implement database initialization (pkg/storage/db.go):
  - [x] Create database file if not exists
  - [x] Enable WAL mode
  - [x] Create tables and indexes
  - [x] Set pragmas for performance
- [x] Write tests:
  - [x] Test database creation
  - [x] Test migrations
  - [x] Test schema integrity

#### 1.2 Storage Layer (CRUD Operations) ✅
- [x] Implement storage interface (pkg/storage/store.go):
  ```go
  type Store interface {
      Insert(entry *HistoryEntry) error
      Query(filters QueryFilters) ([]*HistoryEntry, error)
      GetByID(id int64) (*HistoryEntry, error)
      Count() (int64, error)
      Delete(id int64) error
      Close() error
  }
  ```
- [x] Implement SQLite store (pkg/storage/store.go):
  - [x] Insert with prepared statements
  - [x] Query with WHERE clause building
  - [x] Efficient pagination
  - [x] DeleteByFilter functionality
- [x] Write comprehensive tests:
  - [x] Test Insert (single)
  - [x] Test Query with various filters
  - [x] Test pagination
  - [x] Test GetByID
  - [x] Test Count
  - [x] Test Delete
  - [x] Test DeleteByFilter

#### 1.3 Deduplication Logic ✅
- [x] Implement hash generation (pkg/storage/hash.go):
  - [x] SHA256 of command text
  - [x] Handle edge cases (whitespace, etc.)
  - [x] GenerateHashWithContext for context-aware dedup
- [x] Implement deduplication strategies (pkg/storage/dedup.go):
  - [x] keep_first: Skip duplicate inserts
  - [x] keep_last: Update timestamp on duplicate
  - [x] keep_all: Allow duplicates (preserves context for AI)
- [x] Add DedupConfig structure for configuration
- [x] Implement GetDuplicates() and DeduplicateExisting() utilities
- [x] Write tests:
  - [x] Test hash consistency
  - [x] Test each dedup strategy
  - [x] Test KeepAll preserves context for AI
  - [x] Test GetDuplicates and DeduplicateExisting
  - [x] Test auto hash generation

#### 1.4 Command Capture (Manual) ✅
- [x] Implement capture package (pkg/capture/capture.go):
  - [x] Collect command metadata:
    - [x] Current working directory
    - [x] Hostname
    - [x] Username
    - [x] Shell type (from $SHELL)
    - [x] Timestamp
  - [x] Optional metadata:
    - [x] Git branch (detect .git)
    - [x] Exit code (passed as parameter)
    - [x] Duration in milliseconds
    - [x] Session ID generation
- [x] Write tests for metadata collection:
  - [x] Test metadata collection
  - [x] Test git branch detection
  - [x] Test session ID generation
- [x] Implement main.go entry point:
  - [x] Parse command line args (no Cobra, simple flag parsing)
  - [x] Handle --save flag with --cmd, --exit-code, --duration
  - [x] Handle --help and --version flags
- [x] Implement --save command handler:
  - [x] Create HistoryEntry from metadata
  - [x] Insert to database with KeepAll dedup config
  - [x] Handle errors gracefully (silent exit on success)
- [x] Manual testing:
  - [x] Test save with various commands
  - [x] Verified metadata collection (cwd, hostname, user, shell, git branch)
  - [x] Verified database storage and retrieval

#### 1.5 Configuration System
- [ ] Define config structure (pkg/config/config.go):
  ```go
  type Config struct {
      Database         string
      Deduplicate      bool
      DeduplicateStrategy string
      IgnorePatterns   []string
  }
  ```
- [ ] Implement config loading:
  - [ ] Load from ~/.fh/config.yaml
  - [ ] Merge with defaults
  - [ ] Validate configuration
- [ ] Use viper for config management
- [ ] Write tests:
  - [ ] Test default config
  - [ ] Test loading from file
  - [ ] Test invalid config handling
  - [ ] Test config validation

**Deliverable**: Working storage layer with manual `--save` command

**Testing Milestone**: Coverage >80% for pkg/storage, pkg/capture, pkg/config

---

## Phase 2: Search & FZF Integration

**Goal**: Implement search functionality and FZF integration for interactive history browsing

### Tasks

#### 2.1 Query Builder
- [ ] Implement query filters (pkg/storage/query.go):
  ```go
  type QueryFilters struct {
      Search    string   // Text search in command
      Cwd       string   // Filter by directory
      After     int64    // After timestamp
      Before    int64    // Before timestamp
      ExitCode  *int     // Filter by exit code
      Limit     int      // Max results
      Offset    int      // Pagination
  }
  ```
- [ ] Build SQL WHERE clauses dynamically
- [ ] Add full-text search support (FTS5 extension)
- [ ] Write tests:
  - [ ] Test each filter independently
  - [ ] Test combined filters
  - [ ] Test SQL injection prevention
  - [ ] Test performance with large datasets

#### 2.2 Search Command
- [ ] Implement search logic (pkg/search/search.go):
  - [ ] Query database with filters
  - [ ] Format results for display
  - [ ] Sort by relevance/timestamp
- [ ] Handle search query parsing:
  - [ ] Plain text search
  - [ ] Regex support (optional)
  - [ ] Case-insensitive by default
- [ ] Write tests:
  - [ ] Test search accuracy
  - [ ] Test result ordering
  - [ ] Test empty results
  - [ ] Test special characters in search

#### 2.3 FZF Integration
- [ ] Research FZF integration options:
  - [ ] Option 1: Shell out to fzf binary
  - [ ] Option 2: Use go-fzf library
- [ ] Implement FZF search (pkg/search/fzf.go):
  - [ ] Format entries for FZF: `timestamp | cwd | command`
  - [ ] Configure FZF options:
    - [ ] --tac (reverse order)
    - [ ] --no-sort (preserve order)
    - [ ] --preview (show details)
    - [ ] --multi (select multiple)
  - [ ] Handle FZF output (selected command)
  - [ ] Return selected command to stdout
- [ ] Implement preview window (pkg/search/preview.go):
  - [ ] Format entry details nicely
  - [ ] Show all metadata
- [ ] Write tests:
  - [ ] Test FZF output formatting
  - [ ] Test FZF binary detection
  - [ ] Test fallback if FZF not installed
  - [ ] Test selection parsing

#### 2.4 Main CLI Entry Point
- [ ] Implement main.go with flag parsing:
  - [ ] No args → FZF search
  - [ ] Args without flags → FZF search with pre-filter
  - [ ] --save → Save command (from Phase 1)
  - [ ] --init → Placeholder for Phase 3
  - [ ] --help → Show usage
  - [ ] --version → Show version
- [ ] Implement clean flag parsing (no Cobra):
  ```go
  // Detect flags vs search query
  if hasPrefix("--") {
      handleFlag()
  } else {
      handleSearch(args)
  }
  ```
- [ ] Write tests:
  - [ ] Test flag detection
  - [ ] Test argument parsing
  - [ ] Test help output
  - [ ] Test version output

#### 2.5 Error Handling & User Experience
- [ ] Implement graceful error messages:
  - [ ] Database not found → suggest --init
  - [ ] FZF not installed → suggest installation
  - [ ] Empty history → friendly message
- [ ] Add debug mode (--debug flag):
  - [ ] Show SQL queries
  - [ ] Show performance metrics
  - [ ] Verbose logging
- [ ] Write tests for error cases

**Deliverable**: Working `fh` and `fh <query>` with FZF

**Testing Milestone**: Coverage >80% for pkg/search, main.go

**Update README.md**:
- [ ] Add installation instructions (go install)
- [ ] Add usage examples with screenshots/gifs
- [ ] Document FZF requirement

---

## Phase 3: Shell Integration

**Goal**: Seamless integration with bash and zsh shells

### Tasks

#### 3.1 Shell Hook Generation
- [ ] Create shell integration templates (shell/bash.sh):
  ```bash
  __fh_save() {
      # Capture command, exit code, metadata
      # Call fh --save in background
  }
  PROMPT_COMMAND="__fh_save; $PROMPT_COMMAND"
  bind '"\C-r": "..."'  # Bind Ctrl-R to fh
  ```
- [ ] Create zsh integration (shell/zsh.sh):
  ```zsh
  precmd_functions+=(__fh_save)
  bindkey '^r' fh-widget
  ```
- [ ] Implement hook generator (pkg/capture/hook.go):
  - [ ] Detect current shell
  - [ ] Generate appropriate hooks
  - [ ] Handle edge cases (existing PROMPT_COMMAND, etc.)

#### 3.2 Init Command
- [ ] Implement --init (cmd/init.go):
  - [ ] Create ~/.fh/ directory
  - [ ] Initialize database
  - [ ] Create default config.yaml
  - [ ] Detect shell (bash/zsh)
  - [ ] Backup existing rc files
  - [ ] Append hooks to rc files
  - [ ] Import existing history (Phase 3.3)
  - [ ] Print setup instructions
- [ ] Add --force flag to reinitialize
- [ ] Add --dry-run to preview changes
- [ ] Write tests:
  - [ ] Test directory creation
  - [ ] Test rc file modification
  - [ ] Test backup creation
  - [ ] Test idempotency (running --init twice)

#### 3.3 Import Existing History
- [ ] Implement bash history parser (pkg/importer/bash.go):
  - [ ] Parse ~/.bash_history
  - [ ] Handle HISTTIMEFORMAT entries
  - [ ] Extract commands and timestamps
- [ ] Implement zsh history parser (pkg/importer/zsh.go):
  - [ ] Parse ~/.zsh_history
  - [ ] Handle extended_history format
  - [ ] Extract commands, timestamps, duration
- [ ] Implement import logic (pkg/importer/import.go):
  - [ ] Deduplicate during import
  - [ ] Preserve chronological order
  - [ ] Show progress for large imports
  - [ ] Handle corrupt entries gracefully
- [ ] Write tests:
  - [ ] Test bash history parsing
  - [ ] Test zsh history parsing
  - [ ] Test import with various formats
  - [ ] Test large history files (10k+ entries)

#### 3.4 Background Save Optimization
- [ ] Optimize --save for speed:
  - [ ] Connection pooling
  - [ ] Batch inserts (queue multiple commands)
  - [ ] Async writes
  - [ ] Measure and log timing (debug mode)
- [ ] Benchmark save operation:
  - [ ] Target: <10ms per command
  - [ ] Test concurrent saves
- [ ] Add metrics collection (optional):
  - [ ] Track save latency
  - [ ] Track errors

#### 3.5 Shell Integration Testing
- [ ] Create integration test suite:
  - [ ] Spawn bash shell with hooks
  - [ ] Execute commands
  - [ ] Verify history capture
  - [ ] Test Ctrl-R binding
- [ ] Test on different OS:
  - [ ] Linux (Ubuntu, Fedora)
  - [ ] macOS
- [ ] Manual testing checklist:
  - [ ] Test in fresh bash shell
  - [ ] Test in fresh zsh shell
  - [ ] Test multiline commands
  - [ ] Test commands with special characters
  - [ ] Test rapid command execution

**Deliverable**: Full shell integration with automatic capture and Ctrl-R binding

**Testing Milestone**: Coverage >75% (integration tests are harder to unit test)

**Update README.md**:
- [ ] Add installation section with --init command
- [ ] Add animated GIF showing Ctrl-R usage
- [ ] Document shell support (bash, zsh)
- [ ] Add troubleshooting section

---

## Phase 4: Statistics & Export

**Goal**: Add utility commands for history analysis and data portability

### Tasks

#### 4.1 Statistics Implementation
- [ ] Implement stats collection (pkg/stats/stats.go):
  - [ ] Total commands
  - [ ] Unique commands
  - [ ] Top N most used commands
  - [ ] Commands per day/week/month
  - [ ] Success rate (exit_code = 0)
  - [ ] Average commands per day
  - [ ] Commands by directory
  - [ ] Commands by time of day (histogram)
- [ ] Implement --stats command (cmd/stats.go):
  - [ ] Query database for stats
  - [ ] Format output nicely (tables, charts)
  - [ ] Add filtering options (--since, --until)
- [ ] Optional: ASCII charts for terminal display
- [ ] Write tests:
  - [ ] Test stats calculations
  - [ ] Test with empty database
  - [ ] Test with edge cases

#### 4.2 Export Functionality
- [ ] Implement export formats (pkg/export/):
  - [ ] Plain text (one command per line)
  - [ ] JSON (structured with metadata)
  - [ ] CSV (importable to spreadsheets)
- [ ] Implement --export command (cmd/export.go):
  - [ ] --format flag (text, json, csv)
  - [ ] --output flag (file path, default stdout)
  - [ ] Apply filters (--since, --until, --search)
- [ ] Write tests:
  - [ ] Test each export format
  - [ ] Test filtering during export
  - [ ] Test large exports

#### 4.3 Import from Export
- [ ] Implement --import command (cmd/import.go):
  - [ ] Auto-detect format
  - [ ] Parse and validate
  - [ ] Deduplicate on import
  - [ ] Handle schema differences
- [ ] Write tests:
  - [ ] Test import from each format
  - [ ] Test invalid data handling

**Deliverable**: Statistics and export/import functionality

**Testing Milestone**: Coverage >80% for pkg/stats, pkg/export

**Update README.md**:
- [ ] Document --stats with examples
- [ ] Document --export and --import
- [ ] Add use cases (backing up history, migrating machines)

---

## Phase 5: Remote Sync & Encryption

**Goal**: Encrypted backup and sync to remote endpoints

### Tasks

#### 5.1 Encryption Implementation
- [ ] Implement crypto package (pkg/crypto/):
  - [ ] AES-256-GCM encryption
  - [ ] PBKDF2 key derivation from passphrase
  - [ ] Encrypt/decrypt functions
  - [ ] Generate random salt/nonce
- [ ] Implement key management (pkg/crypto/keys.go):
  - [ ] Generate master key
  - [ ] Store encrypted key at ~/.fh/key
  - [ ] Derive encryption key from passphrase
  - [ ] Optional: Use system keychain (macOS, Linux)
- [ ] Write comprehensive tests:
  - [ ] Test encryption/decryption roundtrip
  - [ ] Test with various input sizes
  - [ ] Test error handling (wrong key, corrupt data)
  - [ ] Security tests (timing attacks, etc.)

#### 5.2 SFTP Sync Implementation
- [ ] Implement SFTP client (pkg/sync/sftp.go):
  - [ ] Connect with SSH key
  - [ ] Upload file
  - [ ] Download file
  - [ ] List remote files
  - [ ] Delete remote file
- [ ] Use pkg/sftp library
- [ ] Add connection retry logic
- [ ] Write tests:
  - [ ] Mock SFTP server tests
  - [ ] Test upload/download
  - [ ] Test error handling (connection failures)

#### 5.3 Sync Command
- [ ] Implement --sync command (cmd/sync.go):
  - [ ] Create database snapshot
  - [ ] Encrypt snapshot
  - [ ] Generate filename: `history-{hostname}-{timestamp}.db.enc`
  - [ ] Upload to remote endpoint
  - [ ] Keep N most recent backups, delete old ones
  - [ ] Show progress during upload
- [ ] Add sync configuration to config.yaml:
  ```yaml
  sync:
    enabled: true
    protocol: sftp
    host: backup.example.com
    port: 22
    path: /backups/fh
    username: user
    key_file: ~/.ssh/id_rsa
    keep_backups: 10
  ```
- [ ] Write tests:
  - [ ] Test backup creation
  - [ ] Test encryption before upload
  - [ ] Test cleanup of old backups

#### 5.4 Restore Command
- [ ] Implement --restore command (cmd/restore.go):
  - [ ] List available backups from remote
  - [ ] FZF selection of backup
  - [ ] Download selected backup
  - [ ] Decrypt backup
  - [ ] Offer merge or replace options:
    - [ ] Replace: Backup current, restore from remote
    - [ ] Merge: Import remote history into current
- [ ] Write tests:
  - [ ] Test restore workflow
  - [ ] Test merge logic
  - [ ] Test error handling

#### 5.5 Encryption Key Setup
- [ ] Implement --setup-encryption command:
  - [ ] Prompt for passphrase
  - [ ] Generate and store master key
  - [ ] Update config to enable encryption
- [ ] Add --change-passphrase command
- [ ] Write tests for key management

**Deliverable**: Encrypted remote backup and restore

**Testing Milestone**: Coverage >80% for pkg/crypto, pkg/sync

**Update README.md**:
- [ ] Document sync configuration
- [ ] Add security section (encryption details)
- [ ] Add backup/restore guide

---

## Phase 6: AI-Powered Search

**Goal**: Semantic search using LLM APIs

### Tasks

#### 6.1 LLM Client Abstraction
- [ ] Define LLM provider interface (pkg/ai/provider.go):
  ```go
  type Provider interface {
      Query(prompt string, history []string) (string, error)
      Name() string
      MaxTokens() int
  }
  ```
- [ ] Implement OpenAI provider (pkg/ai/openai.go):
  - [ ] API client using official SDK
  - [ ] Prompt construction
  - [ ] Error handling (rate limits, etc.)
- [ ] Implement Anthropic provider (pkg/ai/anthropic.go):
  - [ ] Claude API client
  - [ ] Prompt construction
  - [ ] Error handling
- [ ] Add configuration:
  ```yaml
  ai:
    enabled: true
    provider: anthropic
    api_key_env: ANTHROPIC_API_KEY
    model: claude-3-5-sonnet-20241022
    max_history_items: 1000
    max_tokens: 4000
  ```
- [ ] Write tests:
  - [ ] Mock API responses
  - [ ] Test each provider
  - [ ] Test error handling

#### 6.2 Prompt Engineering
- [ ] Design system prompt (pkg/ai/prompts.go):
  ```
  You are a shell history assistant. Given a list of commands
  with timestamps and context, help the user find relevant commands.

  History format:
  [timestamp] [directory] command

  User question: {query}

  Return only the relevant commands, one per line.
  ```
- [ ] Implement context window management:
  - [ ] Limit history sent to API
  - [ ] Prioritize recent history
  - [ ] Summarize old history if needed
- [ ] Test prompts with various queries

#### 6.3 Ask Command
- [ ] Implement --ask command (cmd/ask.go):
  - [ ] Parse natural language query
  - [ ] Fetch relevant history window
  - [ ] Format history for LLM
  - [ ] Send to configured provider
  - [ ] Parse and display results
  - [ ] Optional: Pipe results to FZF
- [ ] Add cost tracking:
  - [ ] Estimate tokens before sending
  - [ ] Log API calls and costs
  - [ ] Warn if approaching limits
- [ ] Write tests:
  - [ ] Test query parsing
  - [ ] Test history formatting
  - [ ] Test result parsing

#### 6.4 Caching & Optimization
- [ ] Implement query cache (pkg/ai/cache.go):
  - [ ] Cache common queries
  - [ ] TTL-based expiration
  - [ ] Local SQLite cache table
- [ ] Add rate limiting:
  - [ ] Max queries per day
  - [ ] Configurable limits
- [ ] Write tests for caching

**Deliverable**: AI-powered semantic search

**Testing Milestone**: Coverage >75% (API mocking is complex)

**Update README.md**:
- [ ] Document AI features
- [ ] Add setup instructions (API keys)
- [ ] Show example queries
- [ ] Document cost considerations

---

## Phase 7: Polish & Release v1.0

**Goal**: Production-ready release with full documentation and polish

### Tasks

#### 7.1 Performance Optimization
- [ ] Profile application with pprof:
  - [ ] CPU profiling
  - [ ] Memory profiling
  - [ ] Identify bottlenecks
- [ ] Optimize hot paths:
  - [ ] Database queries
  - [ ] FZF integration
  - [ ] Save operation
- [ ] Benchmark suite:
  - [ ] Capture overhead
  - [ ] Search speed
  - [ ] Import speed
- [ ] Target metrics:
  - [ ] Save: <10ms
  - [ ] Search: <500ms for 100k records
  - [ ] FZF launch: <200ms

#### 7.2 Error Handling & Robustness
- [ ] Review all error paths
- [ ] Add proper error messages
- [ ] Handle edge cases:
  - [ ] Disk full
  - [ ] Database corruption
  - [ ] Network failures
  - [ ] Invalid config
- [ ] Add recovery mechanisms:
  - [ ] Database repair
  - [ ] Config validation and reset
- [ ] Write chaos tests

#### 7.3 Documentation
- [ ] Complete README.md:
  - [ ] Professional landing page
  - [ ] Feature highlights
  - [ ] Installation (multiple methods)
  - [ ] Quick start guide
  - [ ] Full usage documentation
  - [ ] Configuration reference
  - [ ] Troubleshooting guide
  - [ ] FAQ
  - [ ] Contributing guide
  - [ ] Comparison with alternatives
- [ ] Create docs/ directory:
  - [ ] Architecture overview
  - [ ] Database schema
  - [ ] Shell integration details
  - [ ] Security best practices
- [ ] Add man page (optional)
- [ ] Create website (GitHub Pages) (optional)

#### 7.4 User Experience
- [ ] Add color output (configurable):
  - [ ] Color FZF results
  - [ ] Color stats output
  - [ ] Color error messages
- [ ] Add progress bars for long operations:
  - [ ] Import
  - [ ] Export
  - [ ] Sync
- [ ] Improve help messages:
  - [ ] Better examples
  - [ ] Clear descriptions
- [ ] Add shell completions:
  - [ ] Bash completion
  - [ ] Zsh completion
  - [ ] Fish completion (if supported)

#### 7.5 Security Audit
- [ ] Review all security-sensitive code:
  - [ ] SQL injection prevention
  - [ ] Command injection prevention
  - [ ] File path validation
  - [ ] Encryption implementation
- [ ] Test with malicious inputs
- [ ] Add security.md document
- [ ] Consider external security audit

#### 7.6 Release Process
- [ ] Finalize goreleaser config:
  - [ ] All target platforms
  - [ ] Homebrew formula (macOS)
  - [ ] AUR package (Arch Linux)
  - [ ] deb/rpm packages
- [ ] Create installation scripts:
  - [ ] install.sh for Unix systems
  - [ ] Chocolatey package (Windows)
- [ ] Prepare release notes:
  - [ ] Changelog
  - [ ] Breaking changes
  - [ ] Upgrade guide
- [ ] Tag v1.0.0 and release

#### 7.7 Post-Release
- [ ] Submit to package managers:
  - [ ] Homebrew
  - [ ] AUR
  - [ ] apt/yum repos (via packagecloud.io)
- [ ] Announce on:
  - [ ] Hacker News
  - [ ] Reddit (r/commandline, r/golang)
  - [ ] Twitter/X
  - [ ] Dev.to
- [ ] Monitor issues and feedback
- [ ] Plan v1.1 based on feedback

**Deliverable**: Production-ready v1.0.0 release

**Testing Milestone**: Overall coverage >80%, all critical paths tested

---

## Testing Strategy (Across All Phases)

### Unit Tests
- **Goal**: Test individual functions and packages in isolation
- **Coverage Target**: >80% for all packages
- **Focus Areas**:
  - Database operations (CRUD)
  - Query building
  - Hash generation
  - Encryption/decryption
  - Config parsing
  - Import/export logic

### Integration Tests
- **Goal**: Test component interactions
- **Focus Areas**:
  - End-to-end command capture
  - Search with real database
  - Shell hook integration
  - Sync workflow (with mock servers)

### Performance Tests
- **Goal**: Ensure performance targets are met
- **Benchmarks**:
  - Save operation: <10ms
  - Search query: <500ms (100k records)
  - Import: >1000 commands/sec
  - Database growth: linear scaling

### Manual Testing Checklist
- [ ] Test on fresh Linux install
- [ ] Test on fresh macOS install
- [ ] Test bash integration end-to-end
- [ ] Test zsh integration end-to-end
- [ ] Test with large history (100k+ commands)
- [ ] Test sync to real SFTP server
- [ ] Test AI queries with real API
- [ ] Test upgrade from previous version

---

## Documentation Updates (Continuous)

### README.md Updates by Phase
- **Phase 1**: Basic project description, development setup
- **Phase 2**: Installation, basic usage (search)
- **Phase 3**: Shell integration, full installation guide
- **Phase 4**: Statistics and export features
- **Phase 5**: Sync and backup configuration
- **Phase 6**: AI search setup and examples
- **Phase 7**: Complete documentation overhaul

### Documentation Checklist
- [ ] Keep README.md in sync with features
- [ ] Add screenshots/GIFs for visual features
- [ ] Document all configuration options
- [ ] Provide example configs
- [ ] Write troubleshooting guide
- [ ] Document architecture (design.md already exists)
- [ ] Add API documentation (godoc comments)

---

## Quality Gates (Each Phase)

Before moving to the next phase, ensure:

1. **Tests Pass**: All tests green in CI
2. **Coverage Met**: Phase-specific coverage targets met
3. **Linters Pass**: golangci-lint reports no issues
4. **Manual Testing**: Phase-specific manual tests completed
5. **Documentation Updated**: README.md reflects new features
6. **No Regressions**: Previous features still work
7. **Performance**: No significant performance degradation

---

## Dependency Management

### External Dependencies

#### Core Dependencies (Minimal)
- `github.com/mattn/go-sqlite3` - SQLite driver
- `github.com/spf13/viper` - Configuration management

#### Optional/Phase-Specific
- `github.com/ktr0731/go-fzf` or shell out to fzf binary
- `github.com/pkg/sftp` - SFTP client
- `golang.org/x/crypto` - Encryption primitives
- OpenAI/Anthropic SDKs (for AI features)

#### Testing Dependencies
- `github.com/stretchr/testify` - Assertions
- `github.com/DATA-DOG/go-sqlmock` - Database mocking

### Dependency Strategy
- Keep dependencies minimal
- Prefer standard library when possible
- Vet all dependencies for security and maintenance
- Pin versions in go.mod
- Regular dependency updates

---

## Risk Management

### Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Database corruption | High | Regular backups, WAL mode, repair tools |
| Save operation too slow | High | Benchmark early, optimize, async saves |
| FZF not installed | Medium | Detect and guide user, fallback search |
| Shell compatibility | Medium | Extensive testing, community feedback |
| API costs (AI) | Medium | Rate limiting, caching, clear warnings |
| Security vulnerabilities | High | Security audit, careful input handling |
| Large history performance | Medium | Indexing, pagination, archive old data |

---

## Success Metrics

### Technical Metrics
- Unit test coverage: >80%
- Integration test coverage: >70%
- Save operation: <10ms (p99)
- Search operation: <500ms (p99, 100k records)
- Binary size: <20MB (all platforms)
- Memory usage: <50MB (typical workload)

### User Metrics (Post-Launch)
- GitHub stars (target: 500+ in 3 months)
- Installation count (homebrew analytics)
- Issue response time: <48 hours
- Bug fix turnaround: <1 week
- Community contributions: 5+ in first 6 months

---

## Future Phases (Post v1.0)

### Phase 8: Advanced Features
- [ ] Fish shell support
- [ ] PowerShell support (Windows)
- [ ] Multi-machine merge strategies
- [ ] Shared team history (optional cloud service)
- [ ] Web UI for browsing history
- [ ] Browser extension (capture terminal commands from web docs)

### Phase 9: Analytics & Insights
- [ ] Command pattern analysis
- [ ] Productivity metrics
- [ ] Error pattern detection
- [ ] Recommendations based on history

### Phase 10: Ecosystem
- [ ] Plugin system
- [ ] Community command library
- [ ] Integration with other tools (tmux, vim, etc.)

---

## Appendix: Command Reference (Full Scope)

```bash
# Core
fh                    # Launch FZF search
fh <query>            # Search with pre-filter
fh --help             # Show help
fh --version          # Show version

# Setup
fh --init             # Initialize and setup shell integration
fh --init --force     # Reinitialize (overwrite)

# Search & Query
fh --debug            # Launch with debug output
fh --since 2024-01-01 # Search from date
fh --cwd /path        # Search by directory

# Statistics
fh --stats            # Show statistics
fh --stats --since 1w # Stats for last week

# Import/Export
fh --import           # Import from bash/zsh history
fh --export           # Export to stdout (plain text)
fh --export --format json --output history.json
fh --import --file history.json

# Sync/Backup
fh --sync             # Upload backup to remote
fh --restore          # Restore from remote backup
fh --setup-encryption # Setup encryption key

# AI Search
fh --ask "what did I do yesterday for auth?"

# Internal (called by shell hooks)
fh --save --cmd "..." --exit-code 0 --cwd /path
```

---

## Development Timeline Estimate

**Note**: This is an estimate for a single developer working part-time

- **Phase 0**: 1-2 days
- **Phase 1**: 3-5 days
- **Phase 2**: 3-4 days
- **Phase 3**: 4-6 days
- **Phase 4**: 2-3 days
- **Phase 5**: 5-7 days (encryption + SFTP)
- **Phase 6**: 4-6 days (AI integration)
- **Phase 7**: 3-5 days (polish)

**Total**: 25-38 days (5-8 weeks part-time)

**MVP (Phases 0-3)**: 11-17 days (2-3.5 weeks)

---

## Conclusion

This plan provides a comprehensive roadmap from initial setup to v1.0 release. Each phase builds on the previous, delivering working software incrementally. The focus on testing, documentation, and CI/CD from day one ensures quality and maintainability.

The MVP (Phases 0-3) delivers a working history replacement with shell integration in 2-3 weeks, providing immediate value while building toward the full vision.
