# Contributing

Thank you for your interest in contributing to TerraTidy!

For the quick-start contributing guide, see [CONTRIBUTING.md](https://github.com/santosr2/TerraTidy/blob/main/CONTRIBUTING.md) at the repository root.

This page provides the comprehensive development reference.

## Getting Started

### Prerequisites

- Go 1.26 or later (development uses 1.27)
- [mise](https://mise.jdx.dev/) task runner
- Git

### Clone the Repository

```bash
git clone https://github.com/santosr2/TerraTidy.git
cd TerraTidy
```

### Install Dependencies

```bash
mise run setup
```

### Build

```bash
mise run build
```

### Run Tests

```bash
mise run test
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
mise run fmt

# Run linter
mise run lint

# Run all tests
mise run test

# Build
mise run build
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
mise run test:integration
```

### Coverage

```bash
mise run test:coverage
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
│   ├── runner/         # Engine runner, parallel execution
│   ├── config/         # Configuration loading
│   ├── output/         # Output formatting
│   ├── engines/        # Engine implementations
│   ├── lsp/            # Language server
│   └── plugins/        # Plugin system
├── pkg/sdk/            # Public SDK
├── docs/               # Documentation
└── testdata/           # Test fixtures
```

## Pre-commit Setup

Install pre-commit hooks for automatic checks on commit:

```bash
# pre-commit is installed by mise (see mise.toml)
mise install
pre-commit install
pre-commit install --hook-type commit-msg  # Conventional commit validation
```

The hooks run formatting, linting, and commit message validation automatically.

## CI Pipeline

The CI pipeline runs on every PR:

**Test workflow** (`.github/workflows/test.yml`):

- **Build:** Compiles on ubuntu, macOS, and Windows
- **Tests:** `go test -v -race -cover ./...` with coverage collection
- **Linting:** golangci-lint and revive
- **Coverage:** Uploaded to Codecov (ubuntu-only)
- **Reproducible build:** Builds the release artifact through GoReleaser twice, from different
  paths and seconds apart, and fails if the binaries differ

**Security workflow** (`.github/workflows/security.yml`):

- **govulncheck:** Scans Go dependencies for known vulnerabilities
- **dependency-review:** Reviews dependency changes in PRs for advisories
- **gitleaks:** Scans for accidentally committed secrets
- **go-licenses:** Verifies all dependencies use approved licenses
- **zizmor:** Scans GitHub Actions workflows for security issues
- **gorelease:** Checks `pkg/sdk` API compatibility on PRs (only when SDK files change)

**Fuzz workflow** (`.github/workflows/fuzz.yml`):

- Runs every fuzz target in the repository, one parallel job per package
- 30s per target on PRs labeled `fuzz`, 5m weekly for deeper exploration

See [Security](security.md) for details on the scanning pipeline.

### Required checks

A pull request into `main` can't merge until two groups of checks pass:

- **Checks that run on every pull request:** `pr-title`, `pre-commit`, `govulncheck`,
  `dependency-review`, `gitleaks`, `licenses`, `osv-scanner`, `zizmor`, `api-compat`,
  `Analyze go` and `Analyze actions`.
- **One result check per workflow that only tests some files:** `Test result`,
  `Container result`, `GitHub Action result`, `VS Code result`, `Example rules result`
  and `Pre-commit hooks result`.

The workflows behind the result checks run on every pull request, but their first job,
`Detect relevant changes`, compares the changed files with the paths that workflow
covers. If none match, the other jobs skip, and the result check passes within seconds.
Otherwise the jobs run, and the result check passes only if each one passed. A docs-only
pull request therefore doesn't wait for the full test matrix, while a Go change can't
merge without it.

If you add a job to one of those workflows, list it in that workflow's `result` job
`needs`. A job the result check doesn't wait on can fail without blocking a merge; the
`workflow-result-jobs` pre-commit hook rejects the change if you forget. To change which
files a workflow covers, edit the `PATTERN` of its `changes` job.

The branch ruleset for `main` is the authoritative list of required checks.

**Benchmark workflow** (`.github/workflows/benchmark.yml`):

- Runs on main pushes (when baseline changes) or PRs with `benchmark` label
- Compares against `benchmarks/baseline.txt`
- 15% regression threshold triggers a warning and PR failure
- Posts comparison results as PR comment

See [Performance](performance.md) for the full benchmarking guide.

**Pre-commit hook test workflow** (`.github/workflows/precommit-test.yml`):

- Validates `.pre-commit-hooks.yaml` YAML syntax and structure
- Tests all hooks (`terratidy-fmt`, `terratidy-check`, `terratidy-fix`, etc.) against real Terraform fixtures
- Verifies hooks work with the actual hook definitions from the repository
- Runs on Ubuntu and macOS (Windows excluded due to PATH/extension handling complexity)

### API Stability

The `pkg/sdk` package is the public API for rule authors. Changes to exported types and functions
are checked by `gorelease` on every PR that modifies `pkg/sdk/**`. Breaking changes require a major
version bump.

## Commit Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix       | Use For                    |
| ------------ | -------------------------- |
| `feat:`      | New features               |
| `fix:`       | Bug fixes                  |
| `docs:`      | Documentation changes      |
| `test:`      | Test additions/changes     |
| `chore:`     | Maintenance, dependencies  |
| `refactor:`  | Code restructuring         |
| `perf:`      | Performance improvements   |

## PR Requirements

- Clear description of what changed and why
- Tests for new behavior
- Updated documentation for user-facing changes
- All CI checks passing
- Conventional commit messages

## Release Process

Releases are driven by a signed version tag. Pushing it triggers the pipeline:

1. Bump the version: `mise run bump:patch`, `bump:minor`, or `bump:major`. Each opens a pre-release cycle
   (`0.3.0` → `0.4.0-alpha`), so for a stable release follow it with `mise run bump:stable`.
   The bump commits the change but does not tag it.
2. Create and push a signed tag: `git tag -s v1.2.3 -m "v1.2.3" && git push origin v1.2.3`.
   Pushing the tag is the only way to get correctly signed release artifacts.
3. The release workflow handles everything else:
    - GoReleaser cross-compiles binaries, builds Docker images, and publishes the Homebrew cask to `santosr2/homebrew-tap`
    - git-cliff generates release notes from conventional commits
    - CHANGELOG.md is automatically updated and committed to main
    - Version alias tags (e.g., `v1`, `v1.2`) are created for stable releases
    - Docker alias tags are updated (`:latest` always points to the latest stable release)
    - Checksums are signed with cosign and build provenance is attested
    - SBOMs are generated for each release archive
    - Post-release smoke tests verify the binary on ubuntu and macOS
    - The Homebrew cask is install-tested on macOS (stable releases only)

## Getting Help

- Open an issue for bugs
- Use discussions for questions

## Code of Conduct

Be respectful and inclusive. We follow the [Contributor Covenant](https://www.contributor-covenant.org/).

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
