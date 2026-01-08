# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Automated changelog generation with git-cliff
- Version alias tags for releases (v0, v0.1 for stable; v0-alpha, v0.1-alpha for pre-releases)

### Changed

- Pre-release versions no longer publish to Homebrew or create Docker alias tags (latest, v0, v0.1)
- Improved release notes with installation instructions and documentation links

### Fixed

- GitHub Action embeds version info via ldflags when building from source

## [0.2.0-alpha] - 2026-01-08

### Added

- GitHub Actions annotations output format (`--format github` or `--format gha`)
- Table output format with color support (`--format table`)
- TFLint integration note in `rules list` command output
- Parallel engine execution with `--parallel` / `-p` flag (~24% faster)
- File caching for parsed HCL files (~65x faster on cache hits)
- Internal runner package for concurrent engine execution
- Internal cache package with TTL and LRU eviction
- Benchmark suite for performance testing
- Style rules for naming conventions and file organization

### Changed

- Consolidated GitHub Action (removed duplicate action/ directory)
- Updated GitHub Action inputs to use `--skip-*` flags instead of `--engines`

### Fixed

- VCS test using `--no-verify` for git commits to avoid pre-commit hook interference
- GitHub Action now uses correct `terratidy check` command (was using non-existent `run`)
- GitHub Action SARIF file path for working directory support
- GitHub Action stdout/stderr separation for JSON and SARIF formats
- GitHub Action builds from source when testing in terratidy repo
- SARIF output line/column numbers now comply with spec (must be >= 1)
- Style engine blank line counting excludes comments
- Version display uses Go build info when ldflags not set

## [0.1.0] - 2025-12-22

### Added

#### Core Platform

- Complete Terraform/Terragrunt quality platform with four engines
- CLI framework with Cobra for intuitive command structure
- Configuration system with YAML support, imports, and profiles
- Comprehensive test coverage across all packages (>80% overall)
- CI/CD pipelines with GitHub Actions
- Release automation with GoReleaser
- Docker support with multi-stage builds
- Pre-commit hooks integration

#### Format Engine (`fmt`)

- Terraform fmt wrapper with enhanced features
- Recursive directory processing
- Parallel file processing for performance
- In-place file modification with `--fix` flag
- Detailed formatting reports

#### Style Engine (`style`)

- Complete style checking implementation
- Built-in rules:
  - Blank lines between blocks
  - Consistent indentation
  - Naming conventions
  - Comment formatting
- Configurable rule severity levels
- Auto-fix capabilities for style violations
- Clear, actionable error messages with line numbers

#### Lint Engine (`lint`)

- Comprehensive linting with built-in rules:
  - `terraform_required_version` - Terraform version constraints
  - `terraform_deprecated_syntax` - Deprecated syntax detection
  - `terraform_unused_declarations` - Unused variable detection
  - `terraform_documented_variables` - Variable description requirements
- Configurable rule severity (error, warning, info)
- Parallel directory processing
- Detailed violation reports

#### Policy Engine (`policy`)

- OPA (Open Policy Agent) integration
- Rego policy support
- Built-in security and compliance policies
- Custom policy loading
- Clear policy violation reports
- Rule disable directives support

#### Output System

- Multiple output formats:
  - Text (human-readable with colors)
  - JSON (structured output)
  - JSON Compact (single-line JSON)
  - SARIF (for CI/CD integration)
  - HTML (visual reports)
- Filtering by severity threshold
- Comprehensive summary statistics

#### Language Server Protocol (LSP)

- Full LSP server implementation
- Real-time diagnostics
- Document formatting
- Range formatting
- Code actions and quick fixes
- Hover documentation
- Configuration synchronization
- Editor-agnostic integration

#### VSCode Extension

- LSP client integration for real-time analysis
- Auto-formatting on save
- Problems panel integration
- Quick fixes via lightbulb
- Configuration commands
- Hover documentation
- Multi-engine support
- Modern build tooling (Bun + Biome)

#### Documentation

- Comprehensive user documentation
- API reference
- Integration guides (GitHub Actions, Pre-commit, Docker, LSP)
- Configuration examples
- Best practices guide
- Troubleshooting section

#### Publishing & Distribution

- GitHub Action for CI/CD integration
- Pre-commit hooks for local workflows
- Docker images (ghcr.io/santosr2/terratidy)
- Homebrew formula (via homebrew-tap)
- Multiple platform releases (Linux, macOS, Windows)
- Shell completions (bash, zsh, fish)

### Changed

- Optimized file processing with parallel execution
- Improved error messages with actionable guidance
- Enhanced configuration flexibility with profiles
- Streamlined CLI interface

### Fixed

- Various bug fixes in style rule enforcement
- Improved HCL parsing error handling
- Fixed policy engine test flakiness
- Corrected configuration inheritance behavior

[Unreleased]: https://github.com/santosr2/terratidy/compare/v0.2.0-alpha...HEAD
[0.2.0-alpha]: https://github.com/santosr2/terratidy/compare/v0.1.0...v0.2.0-alpha
[0.1.0]: https://github.com/santosr2/terratidy/releases/tag/v0.1.0
