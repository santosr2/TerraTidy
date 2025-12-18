# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial project structure
- CLI framework with Cobra
- Configuration system with imports and profiles
- Pre-commit hooks support
- GitHub Action
- AI development documentation (AGENTS.md, .claude/, .cursor/, .opencode/, .github/copilot/)
- Comprehensive documentation
- CI/CD pipelines
- Release automation with GoReleaser
- **Lint Engine**: Complete linting implementation with built-in rules
  - `terraform_required_version` - Check for required Terraform version constraint
  - `terraform_deprecated_syntax` - Detect deprecated Terraform syntax
  - `terraform_unused_declarations` - Find unused variables
  - `terraform_documented_variables` - Require variable descriptions
  - Configurable rule severity (error, warning, info)
  - Parallel directory processing
  - Clear, actionable output with line numbers
  - CLI flags: `--config-file`, `--plugin`, `--rule`

### In Progress

- Fmt Engine implementation
- Style Engine implementation
- Policy Engine implementation
- Output formatters (text, JSON, SARIF)
- Plugin system for custom rules
- VSCode extension

## [0.1.0] - TBD

### Added

- First release

[Unreleased]: https://github.com/santosr2/terratidy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/santosr2/terratidy/releases/tag/v0.1.0
