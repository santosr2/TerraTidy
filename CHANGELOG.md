# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0-alpha.2] - 2026-01-12

### Added

- **release**: Add automated changelog and fix pre-release handling ([3426e86](https://github.com/santosr2/TerraTidy/commit/3426e86)) by [@santosr2](https://github.com/santosr2)
- Add bump-my-version for version management ([12e9a9f](https://github.com/santosr2/TerraTidy/commit/12e9a9f)) by [@santosr2](https://github.com/santosr2)

### Changed

- **style**: Split rules.go into modular files ([ba5aa31](https://github.com/santosr2/TerraTidy/commit/ba5aa31)) by [@santosr2](https://github.com/santosr2)

### Documentation

- Update changelog with v0.2.0-alpha release ([96c06be](https://github.com/santosr2/TerraTidy/commit/96c06be)) by [@santosr2](https://github.com/santosr2)
- Add community documentation files by [@santosr2](https://github.com/santosr2)
- Update CLI output examples to match actual output by [@santosr2](https://github.com/santosr2)

### Fixed

- **action**: Embed version info in build from source ([bf17e1c](https://github.com/santosr2/TerraTidy/commit/bf17e1c)) by [@santosr2](https://github.com/santosr2)
- **style**: Implement proper attribute ordering with auto-fix by [@santosr2](https://github.com/santosr2)
- **style**: Implement blank line rules between and inside blocks by [@santosr2](https://github.com/santosr2)
- **test**: Make TestToAbsPath cross-platform for Windows ([f8c8a40](https://github.com/santosr2/TerraTidy/commit/f8c8a40)) by [@santosr2](https://github.com/santosr2)
- **release**: Add RELEASE_NOTES.md to gitignore to prevent dirty state ([c42f118](https://github.com/santosr2/TerraTidy/commit/c42f118)) by [@santosr2](https://github.com/santosr2)
- **release**: Improve release notes and tag alias workflow ([8e7d7cd](https://github.com/santosr2/TerraTidy/commit/8e7d7cd)) by [@santosr2](https://github.com/santosr2)

### Other

- Add bump-my-version to mise and update bumpversion config ([12e9a9f](https://github.com/santosr2/TerraTidy/commit/12e9a9f)) by [@santosr2](https://github.com/santosr2)

### CI

- Add coverpkg flag for accurate cross-package coverage ([ac0c54c](https://github.com/santosr2/TerraTidy/commit/ac0c54c)) by [@santosr2](https://github.com/santosr2)

## [0.2.0-alpha] - 2026-01-08

### Added

- **cli**: Add TFLint integration info to rules list command by [@santosr2](https://github.com/santosr2)
- **output**: Add GitHub Actions annotations output format ([508924c](https://github.com/santosr2/TerraTidy/commit/508924c)) by [@santosr2](https://github.com/santosr2)
- Add performance optimizations and parallel execution ([3123266](https://github.com/santosr2/TerraTidy/commit/3123266)) by [@santosr2](https://github.com/santosr2)
- Wire up output formatters and file cache by [@santosr2](https://github.com/santosr2)
- **release**: Add version alias tags for stable and pre-releases by [@santosr2](https://github.com/santosr2)
- **output**: Add table format with color support by [@santosr2](https://github.com/santosr2)
- **style**: Add naming and file organization rules by [@santosr2](https://github.com/santosr2)

### Documentation

- Update changelog and readme for GitHub Actions format ([9b35bce](https://github.com/santosr2/TerraTidy/commit/9b35bce)) by [@santosr2](https://github.com/santosr2)
- Update all documentation for accuracy by [@santosr2](https://github.com/santosr2)
- Fix version references to v0.1.0 by [@santosr2](https://github.com/santosr2)
- Add table format to output formats documentation by [@santosr2](https://github.com/santosr2)

### Fixed

- **test**: Use --no-verify for git commits in tests ([c50c7c4](https://github.com/santosr2/TerraTidy/commit/c50c7c4)) by [@santosr2](https://github.com/santosr2)
- **action**: Use correct terratidy check command and consolidate actions by [@santosr2](https://github.com/santosr2)
- Correct changelog.md symlink path by [@santosr2](https://github.com/santosr2)
- **style**: Exclude comments when counting blank lines between blocks by [@santosr2](https://github.com/santosr2)
- **version**: Use Go build info for version when ldflags not set by [@santosr2](https://github.com/santosr2)
- **action**: Correct SARIF file path for working directory and output redirection by [@santosr2](https://github.com/santosr2)
- **action**: Separate stdout and stderr for JSON/SARIF output formats by [@santosr2](https://github.com/santosr2)
- **action**: Build from source when testing in terratidy repo by [@santosr2](https://github.com/santosr2)
- **sarif**: Ensure line/column numbers are at least 1 per SARIF spec by [@santosr2](https://github.com/santosr2)

### Other

- Update gitignore and mise configuration ([3d4cabc](https://github.com/santosr2/TerraTidy/commit/3d4cabc)) by [@santosr2](https://github.com/santosr2)

## [0.1.0] - 2025-12-22

### Added

- Initialize TerraTidy project foundation by [@santosr2](https://github.com/santosr2)
- Add initial Fmt and Style engines structure by [@santosr2](https://github.com/santosr2)
- Add initial Lint engine and output types by [@santosr2](https://github.com/santosr2)
- Add comprehensive tooling and documentation by [@santosr2](https://github.com/santosr2)
- Add tests and rename fmt package to format by [@santosr2](https://github.com/santosr2)
- Add hardcoded secrets detection and enhance CI by [@santosr2](https://github.com/santosr2)
- **vscode**: Add LSP client integration by [@santosr2](https://github.com/santosr2)

### Changed

- Reduce complexity and re-enable revive rules by [@santosr2](https://github.com/santosr2)
- Reduce complexity in style rules by [@santosr2](https://github.com/santosr2)
- Reduce complexity in policy engine and style rules by [@santosr2](https://github.com/santosr2)

### Documentation

- Add missing documentation pages for mkdocs site by [@santosr2](https://github.com/santosr2)
- Fix key feautures list by [@santosr2](https://github.com/santosr2)
- Update CHANGELOG for v0.1.0 release by [@santosr2](https://github.com/santosr2)

### Fixed

- Workflow version by [@santosr2](https://github.com/santosr2)
- Update policy engine to OPA v1 Rego syntax by [@santosr2](https://github.com/santosr2)
- Resolve revive linter warnings by [@santosr2](https://github.com/santosr2)
- **ci**: Resolve Windows test failure with coverage path by [@santosr2](https://github.com/santosr2)
- **ci**: Run coverage only on Ubuntu to avoid Windows path issues by [@santosr2](https://github.com/santosr2)
- **test**: Make groupFilesByDirectory tests platform-agnostic by [@santosr2](https://github.com/santosr2)
- **test**: Make tests platform-agnostic for Windows CI by [@santosr2](https://github.com/santosr2)
- **release**: Update goreleaser config for v2 compatibility by [@santosr2](https://github.com/santosr2)
- **docker**: Simplify Dockerfile for goreleaser compatibility by [@santosr2](https://github.com/santosr2)
- **ci**: Add Docker login for ghcr.io in release workflow by [@santosr2](https://github.com/santosr2)
- **release**: Use TerraTidy repo for homebrew formula by [@santosr2](https://github.com/santosr2)
- **release**: Use github-native changelog format for proper attribution by [@santosr2](https://github.com/santosr2)

### Other

- Add assets by [@santosr2](https://github.com/santosr2)
- Sync everywhere Go to 1.25 by [@santosr2](https://github.com/santosr2)
- Fix pre-commit issues and update OPA import by [@santosr2](https://github.com/santosr2)
- Exclude test files from complexity rules in revive by [@santosr2](https://github.com/santosr2)
- **vscode**: Add .gitignore for build artifacts by [@santosr2](https://github.com/santosr2)

[0.2.0-alpha.2]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha...v0.2.0-alpha.2
[0.2.0-alpha]: https://github.com/santosr2/TerraTidy/compare/v0.1.0...v0.2.0-alpha
[0.1.0]: https://github.com/santosr2/TerraTidy/releases/tag/v0.1.0
