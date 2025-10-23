# fh

> A modern shell history replacement with fuzzy search, deduplication, and AI-powered queries

[![CI](https://github.com/spideyz0r/fh/workflows/test/badge.svg)](https://github.com/spideyz0r/fh/actions)
[![Coverage](https://codecov.io/gh/spideyz0r/fh/branch/main/graph/badge.svg)](https://codecov.io/gh/spideyz0r/fh)
[![Go Report Card](https://goreportcard.com/badge/github.com/spideyz0r/fh)](https://goreportcard.com/report/github.com/spideyz0r/fh)
[![License](https://img.shields.io/github/license/spideyz0r/fh)](LICENSE)

---

## 🚧 Work in Progress

**fh** is currently under active development. Check back soon for the first release!

**Current Status**: Phase 0 - Project Foundation

---

## Vision

`fh` (pronounced "fast history" or "find history") is designed to be a modern replacement for traditional shell history tools. It addresses common pain points like duplicate commands, poor search capabilities, and lack of context by providing:

- **Fast fuzzy search** powered by FZF
- **Automatic deduplication** of repeated commands
- **Rich metadata** capture (timestamps, exit codes, working directory, git branch)
- **AI-powered semantic search** to find commands by describing what you were doing
- **Encrypted remote backups** for syncing across machines
- **Cross-shell support** (bash, zsh, and more to come)

---

## Features (Planned)

### Core Features
- [x] Project foundation and CI/CD setup
- [ ] SQLite-based history storage
- [ ] FZF-powered interactive search
- [ ] Automatic command deduplication
- [ ] Bash and Zsh shell integration
- [ ] Import from existing history files

### Advanced Features
- [ ] Statistics and analytics
- [ ] Export/import in multiple formats (JSON, CSV)
- [ ] Encrypted remote backups (SFTP)
- [ ] AI-powered semantic search
- [ ] Multi-machine sync

---

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/spideyz0r/fh/main/install.sh | bash
```

### Using Go

```bash
go install github.com/spideyz0r/fh@latest
```

### Manual Download

Download the latest binary from [releases](https://github.com/spideyz0r/fh/releases) and place it in your PATH.

### From Source

```bash
git clone https://github.com/spideyz0r/fh.git
cd fh
make build
sudo make install  # or copy ./fh to a directory in your PATH
```

---

## Quick Start

### 1. Initialize fh

This sets up shell integration and imports your existing history:

```bash
fh --init
```

Then restart your shell or source your rc file:

```bash
# For bash
source ~/.bashrc

# For zsh
source ~/.zshrc
```

### 2. Use it!

```bash
# Search your history interactively (or just press Ctrl-R)
fh

# Search with a query
fh kubectl

# Show statistics
fh --stats
```

---

## Usage

```bash
# Launch FZF search (also bound to Ctrl-R)
fh

# Search with pre-filter
fh kubectl get pods

# Show statistics
fh --stats

# Export history
fh --export --format json > history.json

# Sync to remote backup (after configuration)
fh --sync

# AI-powered search (requires API key configuration)
fh --ask "what command did I use yesterday to debug the API?"

# Show help
fh --help
```

---

## Architecture

See [design.md](design.md) for detailed architecture and design decisions.

See [plan.md](plan.md) for the complete development roadmap.

### Key Design Principles

- **No persistent daemon**: Simple, fast, and reliable
- **SQLite for storage**: Fast queries, no external dependencies
- **Shell hook integration**: Seamless capture without changing workflows
- **Privacy-first**: All data stored locally, optional self-hosted sync
- **Minimal dependencies**: Keep it simple and maintainable

---

## Documentation

- [Design Document](design.md) - Architecture and technical decisions
- [Development Plan](plan.md) - Detailed roadmap and tasks
- [Contributing Guide](CONTRIBUTING.md) - How to contribute

---

## Comparison with Alternatives

| Feature | bash/zsh history | hishtory | fh |
|---------|------------------|----------|----------|
| FZF search | Manual binding | ✓ | ✓ (planned) |
| Deduplication | ✗ | ✓ | ✓ (planned) |
| Rich metadata | Limited | ✓ | ✓ (planned) |
| Remote sync | ✗ | ✓ (cloud) | ✓ (self-hosted, planned) |
| AI search | ✗ | ✗ | ✓ (planned) |
| Encryption | ✗ | ✓ | ✓ (planned) |
| No daemon | ✓ | ✗ | ✓ |

**Why fh?**
- **No daemon required** - simpler and more reliable than tools requiring background processes
- **AI-powered search** - find commands by describing what you were doing
- **Self-hosted sync** - your data stays under your control
- **Straightforward architecture** - easy to understand, modify, and contribute to

---

## License

[GNU General Public License v3.0](LICENSE)
