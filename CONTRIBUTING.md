# Contributing to TerraTidy

Thank you for your interest in contributing to TerraTidy!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/santosr2/TerraTidy`
3. Set up development environment:

   ```bash
   mise install
   mise run setup
   mise run build
   mise run test
   ```

## Development Guidelines

### Code Style

- Follow Go standard style (gofmt, golangci-lint)
- Write tests for new features
- Use table-driven tests
- Add godoc comments for public APIs

### Testing

```bash
mise run test           # Run unit tests
mise run test:integration  # Run integration tests
mise run lint           # Run linters
```

### Benchmarks

```bash
mise run benchmark      # Run benchmarks and compare with baseline
```

For performance-sensitive changes, run benchmarks before and after. The CI checks for
regressions > 15%. See [Performance Guide](docs/site/docs/development/performance.md).

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
3. Run tests and linters: `mise run check`
4. Commit with conventional commit messages
5. Push and create PR

## Adding Features

### New Rules

```bash
mise run build
./bin/terratidy init-rule --name my-rule --type go
# Edit generated files
./bin/terratidy test-rule my-rule
```

### New Engines

1. Create package in `internal/engines/`
2. Implement Engine interface
3. Add tests
4. Update documentation

### VSCode Extension

```bash
cd vscode
bun install
bun run compile    # Build extension
bun run lint       # Lint with Biome
bun run test       # Run integration tests
bun run package    # Package .vsix
```

## Learn More

For the comprehensive development guide covering CI pipeline, release process, pre-commit setup,
and detailed instructions for adding engines, rules, and output formats, see the
[full contributing documentation](docs/site/docs/development/contributing.md).

## Questions?

- Open an issue for bugs
- Start a discussion for feature requests
- Check existing issues first

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). Be respectful and inclusive!
