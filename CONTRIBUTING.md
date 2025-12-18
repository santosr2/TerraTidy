# Contributing to TerraTidy

Thank you for your interest in contributing to TerraTidy!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/santosr2/terratidy`
3. Set up development environment:

   ```bash
   mise install
   make setup
   make build
   make test
   ```

## Development Guidelines

### Code Style

- Follow Go standard style (gofmt, golangci-lint)
- Write tests for new features
- Use table-driven tests
- Add godoc comments for public APIs

### Testing

```bash
make test           # Run unit tests
make integration    # Run integration tests
make lint           # Run linters
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat: add new feature`
- `fix: bug fix`
- `docs: documentation changes`
- `test: add tests`
- `chore: maintenance`

### Pull Requests

1. Create a feature branch: `git checkout -b feature/my-feature`
2. Make your changes with tests
3. Run tests and linters: `make check`
4. Commit with conventional commit messages
5. Push and create PR

### AI Development

We support AI-assisted development! See:

- [AGENT.md](AGENT.md) for AI development guide
- [CLAUDE.md](CLAUDE.md) for Claude-specific instructions

## Adding Features

### New Rules

```bash
make init-rule NAME=my-rule TYPE=go
# Edit generated files
make test-rule RULE=my-rule
```

### New Engines

1. Create package in `internal/engines/`
2. Implement Engine interface
3. Add tests
4. Update documentation

## Questions?

- Open an issue for bugs
- Start a discussion for feature requests
- Check existing issues first

## Code of Conduct

Be respectful and inclusive. We're here to build great software together!
