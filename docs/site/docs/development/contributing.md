# Contributing

Thank you for your interest in contributing to TerraTidy!

## Getting Started

### Prerequisites

- Go 1.21 or later
- Make
- Git

### Clone the Repository

```bash
git clone https://github.com/santosr2/terratidy.git
cd terratidy
```

### Install Dependencies

```bash
make deps
```

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

## Development Workflow

### Create a Branch

```bash
git checkout -b feature/my-feature
```

### Make Changes

1. Write your code
2. Add tests
3. Update documentation

### Run Checks

```bash
# Format code
make fmt

# Run linter
make lint

# Run all tests
make test

# Build
make build
```

### Commit

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git commit -m "feat: add new style rule for attribute ordering"
git commit -m "fix: correct HCL parsing edge case"
git commit -m "docs: update installation instructions"
```

### Create Pull Request

1. Push your branch
2. Open a PR against `main`
3. Fill out the PR template
4. Wait for review

## Code Style

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint` for linting

### Documentation

- Use clear, concise language
- Include code examples
- Keep lines under 100 characters

## Testing

### Unit Tests

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic case", "input", "expected"},
        {"edge case", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Integration Tests

```bash
make test-integration
```

### Coverage

```bash
make coverage
```

## Adding New Features

### New Engine

1. Create package in `internal/engines/`
2. Implement `Engine` interface
3. Add command in `cmd/terratidy/`
4. Add configuration support
5. Write tests
6. Update documentation

### New Rule

1. Add rule to appropriate engine
2. Implement `Rule` interface
3. Add configuration options
4. Write tests
5. Document in rules reference

### New Output Format

1. Implement `Formatter` interface
2. Register in output factory
3. Add CLI flag support
4. Write tests
5. Document usage

## Project Structure

```text
terratidy/
├── cmd/terratidy/      # CLI commands
├── internal/
│   ├── core/           # Core logic
│   ├── engines/        # Engine implementations
│   ├── lsp/            # Language server
│   └── plugins/        # Plugin system
├── pkg/sdk/            # Public SDK
├── docs/               # Documentation
└── testdata/           # Test fixtures
```

## Documentation

### Code Comments

```go
// MyFunction does something important.
// It handles edge cases by...
func MyFunction(input string) string {
    // Implementation
}
```

### User Documentation

- Add to `docs/site/docs/`
- Update `mkdocs.yml` navigation
- Include examples

## Release Process

1. Update CHANGELOG.md
2. Tag release: `git tag v1.2.3`
3. Push tag: `git push origin v1.2.3`
4. GoReleaser handles the rest

## Getting Help

- Open an issue for bugs
- Use discussions for questions
- Join our community chat

## Code of Conduct

Be respectful and inclusive. We follow the [Contributor Covenant](https://www.contributor-covenant.org/).

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
