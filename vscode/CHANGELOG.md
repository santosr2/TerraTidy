# Change Log

All notable changes to the TerraTidy VSCode extension will be documented in this file.

## [0.2.0] - Unreleased

### Added

- Language Server Protocol (LSP) integration for real-time diagnostics
- Automatic diagnostics as you type
- Code actions and quick fixes via LSP
- Hover documentation for rules
- Restart Language Server command
- Dynamic configuration updates without restart

### Changed

- Replaced command-based execution with LSP client
- Improved performance with incremental analysis
- Streamlined commands (removed redundant manual commands)
- Updated documentation to reflect LSP architecture

### Removed

- Manual run/format/lint/style/fix commands (now handled by LSP)
- Context menu items (functionality now via standard VSCode features)

## [0.1.0] - Initial Release

### Added

- Basic TerraTidy integration
- Format command
- Lint command
- Style check command
- Fix command
- Initialize configuration command
- Format on save support
- Run on save support
- Engine configuration options
- Profile support
