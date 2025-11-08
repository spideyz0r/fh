# fh - Fast History

> A modern shell history replacement with fuzzy search, statistics, and AI-powered queries

[![CI](https://github.com/spideyz0r/fh/workflows/test/badge.svg)](https://github.com/spideyz0r/fh/actions)
[![Coverage](https://codecov.io/gh/spideyz0r/fh/branch/main/graph/badge.svg)](https://codecov.io/gh/spideyz0r/fh)
[![Go Report Card](https://goreportcard.com/badge/github.com/spideyz0r/fh)](https://goreportcard.com/report/github.com/spideyz0r/fh)
[![License](https://img.shields.io/github/license/spideyz0r/fh)](LICENSE)

---

## Features

- ⚡ **Fast fuzzy search** - Handles 40k+ commands instantly with interactive preview
- 🔍 **Smart deduplication** - Keeps your history clean while preserving context for AI
- 📊 **Rich metadata** - Captures timestamps, exit codes, duration, working directory, git branch
- 🤖 **AI-powered search** - Find commands using natural language (OpenAI integration)
- 📦 **Export/Import** - Multiple formats (JSON, CSV, text) with optional AES-256 encryption
- 📈 **Statistics** - Analyze your command usage patterns with detailed insights
- 🐚 **Shell integration** - Seamless bash/zsh integration with Ctrl-R binding
- 🔐 **Privacy-first** - All data stored locally, optional encrypted backups

---

## Quick Start

### Installation

**Using Go:**
```bash
go install github.com/spideyz0r/fh/cmd/fh@latest
```

**From source:**
```bash
git clone https://github.com/spideyz0r/fh.git
cd fh
make build
make install
```

### Setup

Initialize fh and import your existing history:

```bash
fh --init
```

Restart your shell:
```bash
# Bash
source ~/.bashrc

# Zsh
source ~/.zshrc
```

That's it! Press **Ctrl-R** to search your history.

---

## Usage

### Interactive Search

```bash
# Launch fuzzy search (or press Ctrl-R)
fh

# Search with pre-filter
fh docker

# Search for kubectl commands
fh kubectl get pods
```

The fuzzy finder includes a preview window showing:
- Full command
- Timestamp
- Working directory
- Exit code
- Duration
- Git branch (if applicable)

### AI-Powered Search

Ask questions about your command history in natural language:

```bash
# Set your OpenAI API key
export OPENAI_API_KEY='sk-...'

# Find commands by asking questions
fh --ask "what git commands did I run today?"
fh --ask "show me failed commands from last week"
fh --ask "what docker commands did I use yesterday?"
fh --ask "how did I deploy the API to staging?"

# Debug mode to see the SQL query
fh --ask --debug "what testing commands did I run?"
```

The AI understands:
- Relative dates ("yesterday", "last week", "today")
- Command patterns and semantics
- Your command history context

### Statistics

View detailed statistics about your command usage:

```bash
fh --stats
```

Shows:
- Total and unique commands
- Success rate
- Most used commands
- Commands per day average
- Activity by hour of day
- Top directories

### Export & Backup

```bash
# Export as JSON
fh --export --format json --output history.json

# Export as CSV
fh --export --format csv --output history.csv

# Export recent 100 commands as text
fh --export --format text --limit 100

# Create encrypted backup
fh --export --format json --output backup.json.enc --encrypt

# Search and export
fh --export --format json --search docker --output docker-commands.json
```

### Import

```bash
# Import from file (auto-detects format)
fh --import --input history.json

# Import from stdin
cat history.csv | fh --import

# Restore from encrypted backup
fh --import --input backup.json.enc --decrypt
```

---

## Configuration

Configuration file: `~/.fh/config.yaml`

### Default Configuration

```yaml
database:
  path: ~/.fh/history.db

deduplicate:
  enabled: true
  strategy: keep_all  # keep_first, keep_last, or keep_all

ignore:
  patterns:
    - ^ls$
    - '^ls '
    - ^cd$
    - '^cd '
    - ^pwd$
    - ^exit$
    - ^clear$

search:
  limit: 0  # 0 = unlimited (recommended)

ai:
  enabled: true
  provider: openai
  model: gpt-4o-mini  # gpt-4o, gpt-4, gpt-3.5-turbo
  sql_timeout_secs: 60
  max_sql_retries: 10
  max_chunk_tokens: 10000
```

### Deduplication Strategies

- **`keep_first`**: Keep only the first occurrence of each command
- **`keep_last`**: Update timestamp when command repeats (saves space)
- **`keep_all`**: Keep all occurrences with full context (recommended for AI features)

### AI Configuration

Set your OpenAI API key:
```bash
export OPENAI_API_KEY='sk-...'
```

Or disable AI features:
```yaml
ai:
  enabled: false
```

Available models:
- `gpt-4o-mini` (default, fast and cheap)
- `gpt-4o` (more capable, more expensive)
- `gpt-4` (most capable, highest cost)
- `gpt-3.5-turbo` (fastest, cheapest)

---

## How It Works

### Shell Integration

When you run `fh --init`, it:
1. Creates `~/.fh/` directory and SQLite database
2. Imports your existing bash/zsh history
3. Adds hooks to your shell RC file to capture new commands
4. Binds Ctrl-R to launch fh

Every command you run is automatically saved with metadata:
- Command text
- Timestamp
- Exit code
- Duration
- Working directory
- Git branch (if in a git repo)
- Hostname and user

### No Daemon Required

Unlike some history tools, fh doesn't run a background daemon. Command capture happens via shell hooks that call `fh --save` after each command. This is:
- **Simpler** - No process management, no crashes
- **Faster** - ~30ms overhead per command (unnoticeable)
- **More reliable** - Works in any environment

### Privacy & Security

- **Local-first**: All data stored in `~/.fh/history.db`
- **Encryption**: Optional AES-256-GCM encryption for backups
- **No telemetry**: We don't collect any data
- **Self-hosted**: Sync your own way (rsync, Dropbox, etc.)

---

## Requirements

- **Go**: 1.21+ (only for building from source)
- **Bash**: 4.0+ or **Zsh**: any recent version

### macOS Users

macOS ships with bash 3.2 which is **not compatible**. Either:

**Option 1: Upgrade bash**
```bash
brew install bash
echo /opt/homebrew/bin/bash | sudo tee -a /etc/shells
chsh -s /opt/homebrew/bin/bash
```

**Option 2: Use zsh (recommended)**
```bash
chsh -s /bin/zsh
```

---

## Troubleshooting

### Ctrl-R doesn't work

Make sure you ran `fh --init` and restarted your shell:
```bash
fh --init
source ~/.bashrc  # or ~/.zshrc
```

### Database not found

Run initialization:
```bash
fh --init
```

### No history entries found

Check that shell hooks are working:
```bash
# Run a test command
echo "test"

# Check if it was saved
fh test
```

If nothing appears, check your shell RC file (`~/.bashrc` or `~/.zshrc`) for the fh hooks.

### AI search not working

Make sure your OpenAI API key is set:
```bash
export OPENAI_API_KEY='sk-...'

# Add to your shell RC file to persist
echo "export OPENAI_API_KEY='sk-...'" >> ~/.bashrc
```

### Import didn't capture all history

If `--init` didn't import everything, manually import:
```bash
# Bash
fh --import --format text --input ~/.bash_history

# Zsh
fh --import --format text --input ~/.zsh_history
```

---

## Architecture

- **Storage**: SQLite with WAL mode for performance
- **Fuzzy Finder**: ktr0731/go-fuzzyfinder (fast pure-Go implementation)
- **AI**: OpenAI API with smart SQL generation and retry logic
- **Encryption**: AES-256-GCM with PBKDF2 key derivation

See [design.md](design.md) for detailed architecture.
See [plan.md](plan.md) for development roadmap.

---

## Comparison with Alternatives

| Feature | bash/zsh history | hishtory | atuin | fh |
|---------|------------------|----------|-------|-----|
| Fuzzy search | Manual | ✓ | ✓ | ✓ |
| Rich metadata | Limited | ✓ | ✓ | ✓ |
| Statistics | ✗ | ✓ | ✓ | ✓ |
| AI search | ✗ | ✗ | ✗ | ✓ |
| Export/Import | ✗ | ✓ | ✓ | ✓ |
| Encryption | ✗ | ✓ | ✓ | ✓ |
| No daemon | ✓ | ✗ | ✗ | ✓ |
| Self-hosted sync | ✗ | ✗ | ✓ | Manual |

**Why fh?**
- **No daemon** - Simpler, more reliable
- **AI-powered search** - Find commands by describing what you did
- **Fast** - Handles 40k+ commands with instant fuzzy search
- **Privacy-first** - Local storage, optional encrypted backups
- **Simple architecture** - Easy to understand and modify

---

## Development

```bash
# Build
make build

# Run tests
make test

# Run linters
make lint

# Install locally
make install
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

---

## Roadmap

- [x] **Phase 0**: Project foundation and CI/CD
- [x] **Phase 1**: Core storage and capture
- [x] **Phase 2**: Search and FZF integration
- [x] **Phase 3**: Shell integration (bash/zsh)
- [x] **Phase 4**: Statistics and export/import
- [x] **Phase 5**: Encryption for backups
- [x] **Phase 6**: AI-powered search
- [ ] **Phase 7**: Polish and v1.0 release (in progress)

See [plan.md](plan.md) for detailed roadmap.

---

## License

[GNU General Public License v3.0](LICENSE)

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## Credits

Built with:
- [go-fuzzyfinder](https://github.com/ktr0731/go-fuzzyfinder) - Fast fuzzy finder
- [OpenAI Go SDK](https://github.com/openai/openai-go) - AI integration
- [SQLite](https://www.sqlite.org/) - Reliable local storage

---

**Developed by [@spideyz0r](https://github.com/spideyz0r) with AI assistance from Claude**
