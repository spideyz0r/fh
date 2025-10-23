# Contributing to fh

Thank you for your interest in contributing to **fh** (Fast History)! We welcome contributions from the community.

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Make
- golangci-lint (for linting)
- Git

### Getting Started

1. **Fork and clone the repository:**

```bash
git clone https://github.com/YOUR_USERNAME/fh.git
cd fh
```

2. **Install dependencies:**

```bash
make deps
```

3. **Build the project:**

```bash
make build
```

4. **Run tests:**

```bash
make test
```

5. **Run linters:**

```bash
make lint
```

## Development Workflow

### Branch Naming

- Feature branches: `feature/your-feature-name`
- Bug fixes: `fix/bug-description`
- Documentation: `docs/what-you-changed`

### Making Changes

1. Create a new branch from `main`:

```bash
git checkout -b feature/my-new-feature
```

2. Make your changes following our code style guidelines

3. Add tests for your changes

4. Run the full test suite:

```bash
make check
```

5. Commit your changes with clear, descriptive messages:

```bash
git commit -m "Add feature X

- Implement Y
- Update Z
- Add tests for X"
```

## Code Style Guidelines

- Follow standard Go conventions
- Run `make fmt` before committing
- Ensure `make lint` passes without errors
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions small and focused

## Testing

- Write tests for all new features
- Maintain or improve code coverage (target: >80%)
- Use table-driven tests where appropriate
- Mock external dependencies
- Test edge cases and error conditions

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific test
go test -v ./pkg/storage/...
```

## Pull Request Process

1. **Update documentation** if you're adding or changing features

2. **Ensure all tests pass:**

```bash
make check
```

3. **Open a Pull Request** with:
   - Clear title describing the change
   - Description of what changed and why
   - Link to any related issues
   - Screenshots (if applicable)

4. **Wait for review:**
   - Address review comments
   - Keep your branch up to date with main
   - Be patient and respectful

5. **After approval**, a maintainer will merge your PR

## Project Structure

```
fh/
├── cmd/fh/          # Main application entry point
├── pkg/             # Public packages
│   ├── capture/     # Command capture logic
│   ├── storage/     # Database/storage layer
│   ├── search/      # Search and FZF integration
│   ├── ai/          # AI-powered search
│   ├── sync/        # Remote sync functionality
│   └── config/      # Configuration management
├── shell/           # Shell integration scripts
├── test/            # Integration tests
└── docs/            # Documentation
```

## Commit Message Guidelines

We follow conventional commits format:

```
type(scope): subject

body (optional)

footer (optional)
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

**Examples:**

```
feat(storage): add SQLite WAL mode support

- Enable WAL mode for better concurrency
- Add tests for concurrent access
```

```
fix(search): handle empty query gracefully

Fixes #123
```

## Reporting Issues

### Bug Reports

Please include:
- Clear description of the issue
- Steps to reproduce
- Expected behavior vs actual behavior
- Environment details (OS, Go version, shell)
- Error messages or logs
- Screenshots if relevant

### Feature Requests

Please include:
- Clear description of the feature
- Use case / motivation
- Proposed implementation (optional)
- Examples of similar features in other tools (optional)

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## Questions?

- Open a discussion on GitHub
- Check existing issues and PRs
- Read the documentation in [design.md](design.md) and [plan.md](plan.md)

## License

By contributing to fh, you agree that your contributions will be licensed under the GNU General Public License v3.0.
