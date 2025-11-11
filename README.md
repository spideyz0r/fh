# fh - Fast History

[![codecov](https://codecov.io/gh/spideyz0r/fh/branch/main/graph/badge.svg)](https://codecov.io/gh/spideyz0r/fh)

A modern shell history replacement with fuzzy search, statistics, and AI-powered queries.

> **Note:** This application was built as an experiment in AI-assisted development - the entire codebase was created through collaborative coding with Claude.

## Features

- **Fast fuzzy search** - Handles 40k+ commands instantly with interactive preview
- **AI-powered search** - Find commands using natural language queries
- **Rich metadata** - Captures timestamps, exit codes, duration, working directory, git branch
- **Statistics** - Analyze your command usage patterns
- **Export/Import** - Multiple formats (JSON, CSV, text) with optional AES-256 encryption
- **Shell integration** - Seamless bash/zsh integration with Ctrl-R binding
- **Privacy-first** - All data stored locally, no telemetry
- **No daemon** - Simple architecture, no background process required

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

### AI-Powered Search

```bash
# Set your OpenAI API key
export OPENAI_API_KEY='sk-...'

# Ask questions in natural language
fh --ask "what git commands did I run today?"
fh --ask "show me failed commands from last week"
fh --ask "how did I deploy the API to staging?"
```

### Statistics

```bash
fh --stats
```

### Export & Import

```bash
# Export
fh --export --format json --output history.json
fh --export --format json --output backup.json.enc --encrypt

# Import
fh --import --input history.json
fh --import --input backup.json.enc --decrypt
```

---

## Configuration

Edit `~/.fh/config.yaml`:

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

For AI features, add your OpenAI API key to your shell RC file (`~/.bashrc` or `~/.zshrc`):
```bash
export OPENAI_API_KEY='sk-...'
```

## How It Works

When you run `fh --init`:
1. Creates `~/.fh/` directory and SQLite database
2. Imports your existing bash/zsh history
3. Adds hooks to your shell RC file to capture new commands
4. Binds Ctrl-R to launch fh

Every command is automatically saved with metadata (timestamp, exit code, duration, working directory, git branch).

No daemon required - command capture happens via shell hooks. All data stored locally in `~/.fh/history.db`.

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

**Option 2: Use zsh**
```bash
chsh -s /bin/zsh
```

---

## Troubleshooting

**Ctrl-R doesn't work**: Run `fh --init` and restart your shell (`source ~/.bashrc` or `source ~/.zshrc`)

**AI search not working**: Set `export OPENAI_API_KEY='sk-...'` in your shell RC file

**No history entries**: Check that shell hooks are in `~/.bashrc` or `~/.zshrc`

## License

[GNU General Public License v3.0](LICENSE)

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

Built with [go-fuzzyfinder](https://github.com/ktr0731/go-fuzzyfinder), [OpenAI Go SDK](https://github.com/openai/openai-go), and SQLite.
