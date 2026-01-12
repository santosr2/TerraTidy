# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0-alpha.2] - 2026-01-12

### Fixed

- **release**: Improve release notes and tag alias workflow

## [0.2-alpha] - 2026-01-12

### Added

- **release**: Add automated changelog and fix pre-release handling
- Add bump-my-version for version management

### Changed

- **style**: Split rules.go into modular files

### Documentation

- Update changelog with v0.2.0-alpha release
- Add community documentation files
- Update CLI output examples to match actual output

### Fixed

- **action**: Embed version info in build from source
- **style**: Implement proper attribute ordering with auto-fix
- **style**: Implement blank line rules between and inside blocks
- **test**: Make TestToAbsPath cross-platform for Windows
- **release**: Add RELEASE_NOTES.md to gitignore to prevent dirty state

### Other

- Add bump-my-version to mise and update bumpversion config
- Bump version to 0.2.0-alpha.2

### Ci

- Add coverpkg flag for accurate cross-package coverage

## [0.2.0-alpha] - 2026-01-08

### Added

- **cli**: Add TFLint integration info to rules list command
- **output**: Add GitHub Actions annotations output format
- Add performance optimizations and parallel execution
- Wire up output formatters and file cache
- **release**: Add version alias tags for stable and pre-releases
- **output**: Add table format with color support
- **style**: Add naming and file organization rules

### Documentation

- Update changelog and readme for GitHub Actions format
- Update all documentation for accuracy
- Fix version references to v0.1.0
- Add table format to output formats documentation

### Fixed

- **test**: Use --no-verify for git commits in tests
- **action**: Use correct terratidy check command and consolidate actions
- Correct changelog.md symlink path
- **style**: Exclude comments when counting blank lines between blocks
- **version**: Use Go build info for version when ldflags not set
- **action**: Correct SARIF file path for working directory and output redirection
- **action**: Separate stdout and stderr for JSON/SARIF output formats
- **action**: Build from source when testing in terratidy repo
- **sarif**: Ensure line/column numbers are at least 1 per SARIF spec

### Other

- Update gitignore and mise configuration

## [0.1.0] - 2025-12-22

### Added

- Initialize TerraTidy project foundation
- Add initial Fmt and Style engines structure
- Add initial Lint engine and output types
- Add comprehensive tooling and documentation
- Add tests and rename fmt package to format
- Add hardcoded secrets detection and enhance CI
- **vscode**: Add LSP client integration

### Changed

- Reduce complexity and re-enable revive rules
- Reduce complexity in style rules
- Reduce complexity in policy engine and style rules

### Documentation

- Add missing documentation pages for mkdocs site
- Fix key feautures list
- Update CHANGELOG for v0.1.0 release

### Fixed

- Workflow version
- Update policy engine to OPA v1 Rego syntax
- Resolve revive linter warnings
- **ci**: Resolve Windows test failure with coverage path
- **ci**: Run coverage only on Ubuntu to avoid Windows path issues
- **test**: Make groupFilesByDirectory tests platform-agnostic
- **test**: Make tests platform-agnostic for Windows CI
- **release**: Update goreleaser config for v2 compatibility
- **docker**: Simplify Dockerfile for goreleaser compatibility
- **ci**: Add Docker login for ghcr.io in release workflow
- **release**: Use TerraTidy repo for homebrew formula
- **release**: Use github-native changelog format for proper attribution

### Other

- Add assets
- Sync everywhere Go to 1.25
- Fix pre-commit issues and update OPA import
- Exclude test files from complexity rules in revive
- **vscode**: Add .gitignore for build artifacts

[0.2.0-alpha.2]: https://github.com/santosr2/terratidy/compare/v0.2-alpha...v0.2.0-alpha.2
[0.2-alpha]: https://github.com/santosr2/terratidy/compare/v0.2.0-alpha...v0.2-alpha
[0.2.0-alpha]: https://github.com/santosr2/terratidy/compare/v0.1.0...v0.2.0-alpha

