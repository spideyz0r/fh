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

#### 1.5 Configuration System ✅
- [x] Define config structure (pkg/config/config.go):
  ```go
  type Config struct {
      Database    DatabaseConfig
      Deduplicate DeduplicateConfig
      Ignore      IgnoreConfig
  }
  ```
- [x] Implement config loading:
  - [x] Load from ~/.fh/config.yaml
  - [x] Merge with defaults (Default() function)
  - [x] Validate configuration
- [x] Use gopkg.in/yaml.v3 for YAML parsing (not viper, keeping it simple)
- [x] Write tests:
  - [x] Test default config
  - [x] Test loading from file
  - [x] Test invalid config handling
  - [x] Test config validation
  - [x] Test GetDedupConfig() conversion
  - [x] Test Save() functionality
- [x] Update main.go to use config:
  - [x] Load config in handleSave()
  - [x] Use config for database path
  - [x] Use config for deduplication strategy

**Deliverable**: Working storage layer with manual `--save` command

**Testing Milestone**: Coverage >80% for pkg/storage, pkg/capture, pkg/config

---

## Phase 2: Search & FZF Integration

**Goal**: Implement search functionality and FZF integration for interactive history browsing

### Tasks

#### 2.1 Query Builder ✅ (Completed in Phase 1.2)
- [x] Implement query filters (pkg/storage/store.go):
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
- [x] Build SQL WHERE clauses dynamically (in Query() and DeleteByFilter())
- [ ] Add full-text search support (FTS5 extension) - deferred
- [x] Write tests:
  - [x] Test each filter independently
  - [x] Test combined filters
  - [x] SQL injection prevented by using prepared statements
  - [ ] Test performance with large datasets - deferred

#### 2.2 Search Command ✅
- [x] Implement search logic (pkg/search/search.go):
  - [x] Query database with filters
  - [x] Search() and SearchAll() functions
  - [x] SearchWithFilters() for custom queries
  - [x] Results sorted by timestamp DESC (most recent first)
- [x] Handle search query parsing:
  - [x] Plain text search (case-insensitive)
  - [ ] Regex support - deferred
- [ ] Write tests:
  - [ ] Test search accuracy - TODO
  - [ ] Test result ordering - TODO
  - [ ] Test empty results - TODO

#### 2.3 FZF Integration ✅
- [x] Research FZF integration options:
  - [x] Chose github.com/koki-develop/go-fzf (pure Go, used in kubesw)
  - [x] No external fzf binary required
- [x] Implement FZF search (pkg/search/fzf.go):
  - [x] Format entries for FZF: `timestamp | cwd | duration | exit_code | command`
  - [x] FormatEntry() function with proper formatting
  - [x] FzfSearch() launches interactive selector
  - [x] Pre-filter support (filters before FZF)
  - [x] filterEntries() for query-based filtering
  - [x] Handle FZF output (selected command)
  - [x] Return selected HistoryEntry
- [x] Configure FZF options:
  - [x] WithNoLimit(true) - show all results
  - [ ] Preview window - deferred
  - [ ] Multi-select - deferred
- [x] ExtractCommand() utility for parsing formatted entries
- [ ] Write tests:
  - [ ] Test FZF output formatting - TODO
  - [ ] Test filterEntries() - TODO
  - [ ] Test selection parsing - TODO

#### 2.4 Main CLI Entry Point ✅
- [x] Implement main.go with flag parsing:
  - [x] No args → FZF search
  - [x] Args without flags → FZF search with pre-filter (e.g., `fh kubectl`)
  - [x] --save → Save command (from Phase 1)
  - [x] --help, -h → Show usage
  - [x] --version, -v → Show version
  - [ ] --init → Placeholder for Phase 3 - TODO
- [x] Implement clean flag parsing (no Cobra):
  - [x] Switch statement for known flags
  - [x] Default case treats args as search query
  - [x] handleSearch() function for FZF browsing
- [x] Print selected command to stdout (for shell integration)
- [ ] Write tests:
  - [ ] Test flag detection - TODO (CLI tests usually manual)
  - [ ] Test argument parsing - TODO
  - [ ] Test help output - TODO
  - [ ] Test version output - TODO

#### 2.5 Error Handling & User Experience 🔄
- [x] Implement graceful error messages:
  - [x] Config load errors
  - [x] Database errors
  - [x] Empty history → friendly message
  - [x] FZF cancellation → silent exit
  - [ ] Database not found → suggest --init - TODO
- [ ] Add debug mode (--debug flag) - TODO
- [ ] Write tests for error cases - TODO

**Deliverable**: Working `fh` and `fh <query>` with FZF ✅

**Testing Milestone**: Coverage >80% for pkg/search, main.go
- pkg/search: No tests yet (TODO)
- main.go: 0% (CLI, tested manually)

**Current Status**:
- ✅ Interactive FZF search working
- ✅ Pre-filter working (`fh kubectl`)
- ✅ Config-based search limit
- ✅ 52 tests passing (no search tests yet)

**Update README.md**:
- [ ] Add installation instructions (go install) - TODO
- [ ] Add usage examples with screenshots/gifs - TODO
- [ ] Document configuration options - TODO

---

## Phase 3: Shell Integration

**Goal**: Seamless integration with bash and zsh shells

### Tasks

#### 3.1 Shell Hook Generation ✅
- [x] Create shell integration templates (pkg/capture/shell/bash.sh):
  ```bash
  __fh_save() {
      # Capture command, exit code, metadata
      # Call fh --save in background
  }
  PROMPT_COMMAND="__fh_save; $PROMPT_COMMAND"
  bind '"\C-r": "..."'  # Bind Ctrl-R to fh
  ```
- [x] Create zsh integration (pkg/capture/shell/zsh.sh):
  ```zsh
  precmd_functions+=(__fh_save)
  bindkey '^r' fh-widget
  ```
- [x] Implement hook generator (pkg/capture/hook.go):
  - [x] Detect current shell (DetectShell)
  - [x] Generate appropriate hooks (GetHookContent)
  - [x] Handle edge cases (IsHookInstalled, idempotent)
  - [x] Embedded shell scripts using go:embed

#### 3.2 Init Command ✅
- [x] Implement --init (handleInit in main.go):
  - [x] Create ~/.fh/ directory
  - [x] Initialize database
  - [x] Create default config.yaml
  - [x] Detect shell (bash/zsh)
  - [x] Backup existing rc files
  - [x] Append hooks to rc files
  - [x] Print setup instructions with backup location
  - [x] Import existing history - deferred to Phase 3.3
- [x] Idempotency tested (running --init twice doesn't duplicate hooks)
- [ ] Add --force flag to reinitialize - deferred
- [ ] Add --dry-run to preview changes - deferred
- [ ] Write tests:
  - [ ] Test directory creation - TODO
  - [ ] Test rc file modification - TODO (manually tested)
  - [ ] Test backup creation - TODO (manually tested)
  - [ ] Test idempotency - TODO (manually tested)

#### 3.3 Import Existing History ✅
- [x] Implement bash history parser (pkg/importer/bash.go):
  - [x] Parse ~/.bash_history
  - [x] Handle HISTTIMEFORMAT entries (#timestamp format)
  - [x] Extract commands and timestamps
  - [x] Handle files without timestamps (use current time)
- [x] Implement zsh history parser (pkg/importer/zsh.go):
  - [x] Parse ~/.zsh_history
  - [x] Handle extended_history format (: timestamp:duration;command)
  - [x] Extract commands, timestamps, duration
  - [x] Support ZDOTDIR environment variable
- [x] Implement import logic (pkg/importer/import.go):
  - [x] Deduplicate during import (using config dedup strategy)
  - [x] Preserve chronological order
  - [x] ImportHistory() - auto-detects shell and imports
  - [x] ImportFromFile() - import from specific file path
  - [x] Handle corrupt entries gracefully (skip and continue)
  - [x] Return ImportResult with statistics
- [x] Integrated into --init command
  - [x] Automatically imports existing history on first setup
  - [x] Shows count of imported commands
  - [x] Continues on import errors (non-fatal)
- [ ] Write tests:
  - [ ] Test bash history parsing - TODO
  - [ ] Test zsh history parsing - TODO
  - [ ] Test import with various formats - TODO
  - [ ] Test large history files (10k+ entries) - TODO

#### 3.4 Background Save Optimization ✅
- [x] Benchmarked baseline performance: ~40ms per --save
- [x] Implemented config caching (pkg/config/config.go):
  - [x] Thread-safe cache with sync.RWMutex
  - [x] File modification time checking
  - [x] ClearCache() function for manual invalidation
- [x] Implemented metadata caching (pkg/capture/capture.go):
  - [x] Cache immutable data: hostname, user, shell
  - [x] Use sync.Once for one-time initialization
- [x] Benchmarked optimized performance: ~30ms per --save (25% improvement)
- [x] Analysis of remaining overhead:
  - Process startup: ~15-20ms (Go runtime, libraries)
  - SQLite connection: ~5-10ms (database open)
  - Database write: ~5ms (INSERT operation)
- [x] Decision: 30ms is acceptable for background operations
  - Original <10ms target would require daemon architecture (rejected in Phase 0)
  - Commands run in background with disown (imperceptible to users)
  - 25% improvement is good pragmatic optimization
- [ ] Connection pooling - Not applicable (no daemon)
- [ ] Batch inserts - Deferred (complexity vs benefit)
- [ ] Metrics collection - Deferred to future phase

#### 3.5 Shell Integration Testing ✅
- [x] Created integration test suite (test/integration/shell_test.go):
  - [x] TestShellHookGeneration - Verifies bash/zsh hooks contain required functions
  - [x] TestInitCommand - Tests --init in isolated environment
  - [x] TestSaveCommand - Tests --save command directly
  - [x] TestSaveWithSpecialCharacters - Tests quotes, pipes, redirects, etc.
  - [x] TestRapidSaves - Tests 20 concurrent --save operations
  - [x] TestMetadataCapture - Verifies hostname, user, cwd, timestamp
  - [x] TestHookIdempotency - Verifies --init twice doesn't duplicate hooks
- [x] All tests use isolated temp directories (t.TempDir())
- [x] All tests use custom HOME environment (safe, no real shell config touched)
- [x] Tests run in CI on multiple OS (Linux, macOS)
- [x] All 7 tests passing
- [ ] Manual testing - Skipped (automated tests sufficient)

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

#### 4.1 Statistics Implementation ✅
- [x] Implemented stats collection (pkg/stats/stats.go):
  - [x] Total commands
  - [x] Unique commands
  - [x] Top N most used commands
  - [x] Success rate (exit_code = 0)
  - [x] Average commands per day
  - [x] Commands by directory
  - [x] Commands by time of day (histogram with ASCII bars)
- [x] Implemented --stats command (main.go handleStats):
  - [x] Query database for stats
  - [x] Format output nicely (formatted text with percentages)
  - [x] ASCII charts for hour distribution
- [x] Implemented CollectFiltered for future filter support (--since, --until)
- [x] Written comprehensive tests (pkg/stats/stats_test.go):
  - [x] TestCollect_EmptyDatabase
  - [x] TestCollect_SingleCommand
  - [x] TestCollect_MultipleCommands
  - [x] TestCollect_TimeDistribution
  - [x] TestCollect_AveragePerDay
  - [x] TestFormat_EmptyStats
  - [x] TestFormat_WithData
  - [x] TestCollectFiltered
  - [x] All 8 tests passing

#### 4.2 Export Functionality ✅
- [x] Implemented export formats (pkg/export/export.go):
  - [x] Plain text (one command per line)
  - [x] JSON (structured with full metadata)
  - [x] CSV (importable to spreadsheets with header)
- [x] Implemented --export command (main.go handleExport):
  - [x] --format flag (text, json, csv)
  - [x] --output flag (file path or stdout)
  - [x] --search filter support
  - [x] --limit support
  - [x] ParseFormat() for format validation
- [x] Written comprehensive tests (pkg/export/export_test.go):
  - [x] TestExportText
  - [x] TestExportJSON
  - [x] TestExportCSV
  - [x] TestExportWithFilters
  - [x] TestExportWithLimit
  - [x] TestExportEmpty
  - [x] TestParseFormat
  - [x] TestFormatTimestamp
  - [x] All 8 tests passing

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

## Phase 5: Encryption ~~& Remote Sync~~ ✅ **COMPLETE (Modified)**

**Goal**: ~~Encrypted backup and sync to remote endpoints~~ Add optional encryption to export/import

**Status**: ✅ Complete (SFTP sync deferred to TODO - see Future Improvements)

**What was built**:
- ✅ AES-256-GCM encryption package with 14 comprehensive tests
- ✅ `--encrypt` flag for export command (prompts for passphrase)
- ✅ `--decrypt` flag for import command (prompts for passphrase)
- ✅ Simplified approach: encryption as optional layer on export/import
- ✅ No separate backup/restore commands needed

**What was deferred**:
- ❌ SFTP sync (moved to TODO - users can script their own sync)
- ❌ Backup rotation (not needed with export/import approach)
- ❌ Key management (passphrase-based is simpler)

### Tasks

#### 5.1 Encryption Implementation ✅ COMPLETE
- [x] Implement crypto package (pkg/crypto/):
  - [x] AES-256-GCM encryption
  - [x] PBKDF2 key derivation from passphrase (100k iterations)
  - [x] Encrypt/decrypt functions
  - [x] Generate random salt/nonce (crypto/rand)
- [x] ~~Implement key management~~ Deferred - using passphrase-based encryption
- [x] Write comprehensive tests:
  - [x] Test encryption/decryption roundtrip
  - [x] Test with various input sizes (empty, 1MB)
  - [x] Test error handling (wrong key, corrupt data)
  - [x] 14 tests total, all passing

#### 5.2 Export/Import Integration ✅ COMPLETE
- [x] Add `--encrypt` flag to export command
- [x] Add `--decrypt` flag to import command
- [x] Prompt for passphrase with confirmation
- [x] Integrate with crypto package
- [x] Update help text with examples

#### ~~5.3-5.5 SFTP Sync~~ ❌ DEFERRED
**Moved to TODO section** - SFTP sync, backup rotation, and remote restore deferred.
Users can achieve similar functionality with:
```bash
# Encrypted backup
fh --export --format json --output backup.json.enc --encrypt

# Upload manually
rsync backup.json.enc user@server:/backups/

# Restore
fh --import --input backup.json.enc --decrypt
```

**Deliverable**: ✅ Optional encryption for export/import

**Testing Milestone**: ✅ 14 crypto tests passing, 100% coverage for pkg/crypto

**Update README.md**: (Deferred to Phase 7)
- [ ] Add security section (encryption details)
- [ ] Add encrypted backup examples

---

## Phase 6: AI-Powered Search ✅ **COMPLETE**

**Goal**: Natural language search using OpenAI API

**Status**: ✅ Complete (Gemini/Anthropic support deferred to TODO)

**What was built**:
- ✅ OpenAI client integration with official SDK
- ✅ Two-phase AI workflow: SQL generation → Execution → Result formatting
- ✅ Smart retry logic with error feedback (max 10 retries, configurable)
- ✅ Token estimation and chunking for large result sets
- ✅ SQL validation (whitelist SELECT, blacklist dangerous keywords)
- ✅ Configuration with model selection, timeouts, retry limits
- ✅ `--ask` command for natural language queries
- ✅ Context-aware prompts with database schema, stats, and current date

**What was deferred**:
- ❌ Provider interface (moved to TODO - currently OpenAI only)
- ❌ Gemini/Anthropic support (can add later)
- ❌ Query caching (not needed initially)
- ❌ Cost tracking (deferred)

### Tasks

#### 6.1 OpenAI Client Implementation ✅ COMPLETE
- [x] Implemented OpenAI provider (pkg/ai/openai.go):
  - [x] API client using official openai-go SDK
  - [x] Model mapping (gpt-4o, gpt-4o-mini, gpt-4, gpt-3.5-turbo)
  - [x] Query() method with context support
  - [x] OPENAI_API_KEY environment variable
- [x] Add configuration (pkg/config/config.go):
  ```yaml
  ai:
    enabled: true
    provider: openai
    model: gpt-4o-mini
    sql_timeout_secs: 60
    max_sql_retries: 10
    max_chunk_tokens: 10000
  ```
- [ ] Provider interface - **Deferred to TODO**
- [ ] Implement Gemini provider - **Deferred to TODO**
- [x] Write tests - ✅ **COMPLETE** (28 tests, all passing)

#### 6.2 Prompt Engineering ✅ COMPLETE
- [x] Implemented comprehensive prompts (pkg/ai/prompts.go):
  - [x] GenerateSQLPrompt() - SQL generation with schema, stats, current date
  - [x] GenerateSQLRetryPrompt() - Retry with error feedback
  - [x] GenerateFormatPrompt() - CLI-friendly result formatting
  - [x] GenerateChunkSummaryPrompt() - Summarize large result chunks
  - [x] GenerateFinalSynthesisPrompt() - Synthesize chunk summaries
- [x] Context window management:
  - [x] Database schema included in prompt
  - [x] Database stats (total commands, date range, top commands)
  - [x] Current date/time for relative queries ("yesterday", "last week")
  - [x] Token estimation for chunking (rough: ~4 chars per token)
  - [x] Chunk results if > max_chunk_tokens (10k default)

#### 6.3 Ask Command ✅ COMPLETE
- [x] Implemented --ask command (main.go handleAsk):
  - [x] Parse natural language query from args
  - [x] Check if AI enabled in config
  - [x] Two-phase workflow:
    1. Generate SQL query from user question
    2. Execute SQL with timeout
    3. Format results for CLI output
  - [x] Retry SQL generation up to max_sql_retries (10)
  - [x] SQL validation (must start with SELECT, no dangerous keywords)
  - [x] Clean SQL response (remove markdown code blocks)
  - [x] Handle errors: API errors, SQL errors, empty results, timeouts
  - [x] Debug mode (--ask --debug) - shows prompts, responses, SQL queries, scan errors
- [x] Implemented core logic (pkg/ai/ask.go):
  - [x] Ask() orchestrates the workflow
  - [x] generateSQLWithRetry() - retry loop with error feedback
  - [x] executeSQLQuery() - execute with context timeout
  - [x] formatResults() - format with chunking if needed
  - [x] validateSQL() - security validation
  - [x] estimateTokens() and chunkResults() - chunking logic
  - [x] Handle NULL columns with COALESCE (hash, git_branch)
  - [x] Full command display in formatted output
- [x] Updated help text with examples
- [x] Debug mode implementation with detailed logging
- [x] Write comprehensive tests (28 tests, all passing):
  - [x] prompts_test.go - 9 tests for prompt generation
  - [x] ask_test.go - 13 tests for utility functions
  - [x] openai_test.go - 6 tests for client initialization
- [ ] Cost tracking - **Deferred**

#### 6.4 Caching & Optimization ❌ DEFERRED
- [ ] Query caching - **Moved to TODO**
- [ ] Rate limiting - **Moved to TODO**

**Deliverable**: ✅ AI-powered natural language search with OpenAI

**Testing Milestone**: ✅ 28 tests passing, covers prompts, query execution, and client init

**Update README.md**: (Deferred to Phase 7)
- [ ] Document AI features
- [ ] Add setup instructions (OPENAI_API_KEY)
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

## TODO / Future Improvements

This section tracks improvements and features that are deferred for future releases.

### AI Provider Interface (Deferred from Phase 6)
- [ ] **LLM Provider Abstraction**
  - **Rationale**: OpenAI works well for MVP, interface can be added when second provider is needed
  - **Current approach**: Direct OpenAI implementation in pkg/ai/openai.go
  - **Deferred features**:
    - Define Provider interface (Query, Name, MaxTokens methods)
    - Refactor OpenAI client to implement interface
    - Add Gemini provider (Google AI)
    - Add Anthropic provider (Claude)
    - Provider selection via config (ai.provider field already exists)
  - **Implementation notes**:
    - Config already has `provider` field set to "openai"
    - Model mapping can be moved to provider-specific code
    - Each provider will need its own prompt formatting
  - **Priority**: Medium (needed for multi-LLM support)
  - **Tracked in**: Phase 8+ or when users request Gemini/Claude

### AI Improvements (Deferred from Phase 6)
- [ ] **Query caching** - Cache common AI queries to reduce API costs
- [ ] **Cost tracking** - Track token usage and estimate costs
- [ ] **Rate limiting** - Prevent excessive API usage

### Remote Sync (Deferred from Phase 5)
- [ ] **SFTP Sync Implementation**
  - **Rationale**: Export/import with encryption already handles backup/restore elegantly
  - **Current approach**: Users can use `fh --export --encrypt` for encrypted backups
  - **Deferred features**:
    - SFTP client for remote upload/download
    - Automatic sync on interval
    - Conflict resolution for multi-device sync
    - Backup rotation on remote server
  - **Implementation notes**:
    - Crypto package (AES-256-GCM) is ready for use
    - Can add SFTP as optional sync backend later
    - Alternative: Users can script their own sync (rsync, Dropbox, etc.)
  - **Priority**: Low (nice to have for multi-device users)
  - **Tracked in**: Post-v1.0 or Phase 8+

### FZF Improvements
- [x] **Switched to ktr0731/go-fuzzyfinder for performance** ✅
  - **Issue**: koki-develop/go-fzf was noticeably slow with large datasets (44k entries)
  - **Investigation**:
    - Tested native fzf binary - "waaaaaaaaay" faster than go-fzf
    - Tried ktr0731/go-fuzzyfinder - also "waaaaaaaaay" faster
  - **Solution**: Switched from koki-develop/go-fzf to ktr0731/go-fuzzyfinder
  - **Implementation**:
    - Added github.com/ktr0731/go-fuzzyfinder dependency
    - Created pkg/search/fzf_ktr.go with FzfSearchKtr()
    - Includes preview window showing command details
    - Updated main.go to use new implementation
  - **Files changed**:
    - go.mod/go.sum: Added go-fuzzyfinder and deps (tcell, etc.)
    - pkg/search/fzf_ktr.go: New implementation with preview
    - cmd/fh/main.go: Switch to FzfSearchKtr()
  - **Result**: Handles all 44,861 entries efficiently without slowdown
  - **Performance**: Fast fuzzy search, comparable to native fzf
  - **Bonus**: Preview window shows command metadata (time, cwd, exit code, etc.)
  - **Completed**: 2025-11-07

- [x] **Unlimited search by default** ✅
  - **Issue**: Default search limit of 1000 was too small for imported bash history (44k+ commands)
  - **Solution**: Changed default search.limit from 1000 to 0 (unlimited)
  - **Files changed**:
    - pkg/config/config.go: Default() now sets Limit: 0
    - ~/.fh/config.yaml: Updated existing config to limit: 0
  - **Result**: All 44,861 commands searchable (no hidden results)
  - **Note**: Works efficiently with ktr0731/go-fuzzyfinder
  - **Completed**: 2025-11-07

- [ ] **PageUp/PageDown support in FZF**
  - **Issue**: go-fzf library doesn't support multi-line scrolling (PageUp/PageDown only moves 1 line)
  - **Current workaround**: Added `pgup`/`pgdown` to keybindings, but they behave like arrow keys
  - **Potential solutions**:
    1. Switch to native fzf binary (requires external dependency)
    2. Contribute PageUp/PageDown feature to go-fzf library
    3. Fork go-fzf and add the feature ourselves
    4. Implement custom pager with proper page scrolling
  - **Priority**: Medium (quality of life improvement)
  - **Tracked in**: Phase 7 (Polish) or post-v1.0

### Shell Integration
- [ ] Fish shell support (Phase 3 was deferred)
- [ ] PowerShell support for Windows

### Performance
- [ ] Full-text search with FTS5 extension (deferred from Phase 2.1)
- [ ] Performance testing with large datasets (100k+ entries)

### Testing
- [ ] Search package unit tests (deferred from Phase 2)
- [ ] CLI integration tests (deferred from Phase 2)
- [ ] Shell hook integration tests (deferred from Phase 3)

### Documentation
- [ ] Animated GIFs/screenshots for README
- [ ] Man page generation
- [ ] Website (GitHub Pages)

---

## Conclusion

This plan provides a comprehensive roadmap from initial setup to v1.0 release. Each phase builds on the previous, delivering working software incrementally. The focus on testing, documentation, and CI/CD from day one ensures quality and maintainability.

The MVP (Phases 0-3) delivers a working history replacement with shell integration in 2-3 weeks, providing immediate value while building toward the full vision.
