# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/santosr2/terratidy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/santosr2/terratidy/releases/tag/v0.1.0
