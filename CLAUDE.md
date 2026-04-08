# TerraTidy

Single-binary Terraform/Terragrunt quality platform. Go 1.25+ (dev: 1.26), library-first, extensible plugin system.

## Architecture

```text
cmd/terratidy/          # CLI (Cobra)
internal/
  annotations/          # Suppression annotation parsing and filtering
  buildinfo/            # Build information and versioning
  cache/                # Caching layer
  config/               # YAML config with imports, profiles, glob patterns
  engines/              # format, style, lint, policy
  lsp/                  # Language Server Protocol implementation
  output/               # Text, JSON, SARIF, JUnit, Markdown, HTML, Table, GitHub Actions formatters
  plugins/              # Go (.so), YAML, Bash rule loader
  runner/               # Orchestration, parallel file processing
  vcs/                  # Git integration (--changed flag)
pkg/
  sdk/                  # Public SDK for rule authors
.github/workflows/      # CI/CD pipelines (test, release, security, docs)
assets/                 # Brand assets, icons, logos
docs/site/              # MkDocs documentation site
examples/               # Example configs and custom rules
Formula/                # Homebrew formula (auto-generated)
vscode/                 # VS Code extension (TypeScript, Bun)
tools/scripts/          # Development scripts
Dockerfile              # Container image definition
action.yml              # GitHub Action definition
```

## Core Interfaces

```go
// Every engine implements this (pkg/sdk/types.go)
type Engine interface {
    Name() string
    Run(ctx context.Context, files []string) ([]sdk.Finding, error)
}

// Every rule implements this (pkg/sdk/types.go)
type Rule interface {
    Name() string
    Description() string
    Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error)
}

// Rules that support auto-fixing also implement Fixer
type Fixer interface {
    Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error)
}
```

## Non-Negotiable Rules

- **Library-first**: Use Go libraries (hclwrite, OPA SDK) where possible. TFLint is invoked as a CLI subprocess. Never `exec.Command("terraform", ...)`.
- **No panic**: Return errors with context: `fmt.Errorf("loading config: %w", err)`
- **No global state**: Pass config through context or parameters
- **No circular deps**: `internal/` is private, `pkg/` is public API
- **Actionable errors**: Include file paths, line numbers, suggestions for the user

## Key Libraries

| Library | Purpose |
| --- | --- |
| `github.com/hashicorp/hcl/v2` | HCL parse and write |
| `github.com/hashicorp/hcl/v2/hclwrite` | AST-based formatting |
| `github.com/open-policy-agent/opa` | Policy engine |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/fsnotify/fsnotify` | File watching |
| `golang.org/x/text` | Text processing |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/stretchr/testify` | Test assertions |

## Config System

```yaml
# .terratidy.yaml
version: 1

engines:
  fmt:
    enabled: true
    check: false   # Dry-run mode (don't modify files)
    diff: false    # Show diff of changes
  style:
    enabled: true
    fix: false     # Auto-fix style issues
    diff: false
  lint:
    enabled: true
    use_tflint: false        # Enable TFLint integration
    fallback_builtin: true   # Use built-in rules if TFLint unavailable
  policy:
    enabled: false           # Opt-in for policy checking
    policy_dirs:
      - ./policies

# Global settings
severity_threshold: warning  # info|warning|error
fail_fast: false
parallel: true
recursive: true              # Recursive directory traversal (default: true)

# Caching settings
cache:
  disabled: false
  max_age: "5m"
  max_size: 1000

# Output settings
output:
  absolute_paths: false      # Use absolute paths in findings

imports:
  - .terratidy/*.yaml        # Glob patterns for modular configs

exclude:
  - "**/*.generated.tf"      # Glob patterns for files/dirs to exclude
  - "vendor/**"

profiles:
  production:
    description: "Production checks with policy enforcement"
    inherits: ""             # Optional: inherit from another profile
    engines:
      policy:
        enabled: true

plugins:
  enabled: true
  verify_integrity: true     # Verify plugin integrity (default: true)
  directories:
    - .terratidy/plugins
    - ~/.terratidy/plugins
  tags: []                   # Filter plugins by tag (empty = all)

overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
      severity: warning
```

## Development

```bash
# Go development
mise install              # Install Go 1.26 + tools
mise run setup            # Install dependencies
mise run build            # Build binary
mise run test             # Unit tests
mise run test:integration # Integration tests
mise run lint             # golangci-lint
mise run check            # fmt + vet + lint + test (run before PR)
mise run build && ./bin/terratidy init-rule --name x --type go|rego|yaml  # Scaffold new rule

# VSCode extension development
cd vscode && bun install     # Install extension deps
cd vscode && bun run compile # Build extension
cd vscode && bun run test    # Run extension tests
cd vscode && bun run lint    # Biome lint/format check
```

## Code Quality

All static analysis runs through `golangci-lint` (`.golangci.yml`):

- **Linters**: errcheck, govet, staticcheck, unused, gosec, revive, gocritic, cyclop, funlen, errorlint, and more
- **Formatters**: gofumpt, goimports
- **Config**: `revive.toml` for revive-specific rules
- **Run**: `mise run lint` (or `mise run check` for fmt + vet + lint + test)

Thresholds: cyclomatic complexity 25, function length 120 lines / 60 statements.

## Testing

- Table-driven tests with testify (`require`, `assert`)
- Fixture-based tests for HCL parsing/formatting
- Integration tests for CLI commands (tagged, run via `mise run test:integration`)
- Benchmarks for performance-critical paths (`mise run benchmark`)
- Fuzz tests: `FuzzConfigParse`, `FuzzFormat`, `FuzzYAMLRuleParse`
- VSCode extension tests: mocha + @vscode/test-cli (`cd vscode && bun run test`)
- Target: 80%+ coverage

### Key Packages

| Package | Tests |
| --- | --- |
| `internal/annotations` | Suppression annotation parsing and filtering |
| `internal/config` | Config loading, imports, profiles, validation |
| `internal/engines/format` | HCL formatting |
| `internal/engines/style` | Style rule checks (9 rule files, 9 test files in rules/) |
| `internal/engines/lint` | Lint rule checks |
| `internal/engines/policy` | OPA policy evaluation |
| `internal/plugins` | Plugin loading (Go, YAML, Bash) |
| `internal/runner` | Orchestration, parallel execution |
| `internal/output` | Text, JSON, SARIF, JUnit, Markdown, HTML, Table, GitHub Actions formatters |
| `internal/lsp` | Language Server Protocol |
| `internal/cache` | Caching layer |
| `internal/vcs` | Git integration |
| `pkg/sdk` | Public SDK interfaces |

## VSCode Extension

TypeScript extension using Bun package manager and Biome linter/formatter.

- **Location**: `vscode/src/extension.ts` (main entry)
- **LSP client**: Uses `vscode-languageclient` to connect to TerraTidy LSP server
- **Commands**: `terratidy.init`, `terratidy.showOutput`, `terratidy.restartServer`
- **Settings**: 12 configuration options (executablePath, configPath, profile, engines.*, etc.)
- **Assets**: Icon, CHANGELOG, LICENSE are symlinks from root (do not duplicate)
- **VS Code engine**: ^1.110.0

When adding a new setting, update these files:

1. `vscode/package.json` (contributes.configuration)
2. `vscode/src/extension.ts` (getInitializationOptions)
3. `internal/lsp/types.go` (InitializationOptions struct)
4. `docs/site/docs/integrations/vscode.md`

## LSP Server

stdio-based Language Server Protocol implementation for real-time diagnostics.

- **Location**: `internal/lsp/server.go`, `internal/lsp/types.go`
- **Transport**: stdio only (no socket support)
- **Protocol**: initialize, shutdown, textDocument/didOpen, didChange, didClose, didSave, formatting, codeAction
- **Diagnostics**: Push-only via `textDocument/publishDiagnostics` (no pull diagnostics to avoid duplication)
- **Config**: Reads `.terratidy.yaml`, applies `overrides.rules` to style engine via `buildStyleConfig()`
- **Engines**: style, lint (format engine used for formatting requests only)
- **Thread safety**: Write mutex for concurrent operations

## HCL Guidelines

- Always use `hclwrite` for formatting (never subprocess)
- Preserve comments when modifying AST
- Handle both `.tf` and `.hcl` files
- Test with real Terraform code samples

## Performance

- `sync.WaitGroup` + worker pools for parallel file processing
- Buffer channels appropriately
- Cache expensive operations (HCL parsing)
- Profile with pprof before optimizing

## Rule Templates

### Go Rule

```go
package main

import (
    "github.com/santosr2/TerraTidy/pkg/sdk"
    "github.com/hashicorp/hcl/v2"
)

type MyRule struct{}

func (r *MyRule) Name() string        { return "my-rule" }
func (r *MyRule) Description() string { return "Checks something" }

func (r *MyRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    var findings []sdk.Finding
    return findings, nil
}

// Optional: implement sdk.Fixer for auto-fix support
// func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
//     return fixedContent, nil
// }
```

### YAML Rule

```yaml
name: require-description
description: Resources must have a description
severity: warning
enabled: true
message: "Resource is missing a description attribute"
patterns:
  block_types:
    - resource          # resource, data, variable, output, locals, module
  resource_types:
    - aws_instance      # Optional: filter by resource type
  required_attributes:
    - description
  forbidden_attributes:
    - deprecated_field  # Optional: attributes that must NOT be present
  attribute_patterns:   # Optional: regex validation
    - attribute: name
      pattern: "^[a-z][a-z0-9-]+$"
      message: "Name must be lowercase with hyphens"
```

### Bash Rule

```bash
#!/usr/bin/env bash
set -euo pipefail
FILE="$1"
# Output: {"findings": [{"file": "path", "line": 1, "message": "msg", "severity": "error"}]}
```

## Documentation

| Topic | Location |
| --- | --- |
| Documentation site | `docs/site/` (MkDocs) |
| Architecture | `docs/site/docs/development/architecture.md` |
| Configuration | `docs/site/docs/getting-started/configuration.md` |
| Installation | `docs/site/docs/getting-started/installation.md` |
| Quickstart | `docs/site/docs/getting-started/quickstart.md` |
| Engines | `docs/site/docs/user-guide/engines/` |
| Rules | `docs/site/docs/rules/` |
| Integrations | `docs/site/docs/integrations/` |
| VS Code extension | `docs/site/docs/integrations/vscode.md` |
| Changelog | `CHANGELOG.md` |

## CI/CD

| Workflow | Trigger | Purpose |
| --- | --- | --- |
| `test.yml` | push/PR | Tests on 3 OSes x 2 Go versions, coverage to Codecov |
| `release.yml` | tag `v*` | GoReleaser, Docker, Homebrew, cosign signing |
| `quality.yml` | push/PR | PR title validation, pre-commit hooks |
| `security.yml` | push/PR | govulncheck, gitleaks, license check, API compat |
| `fuzz.yml` | push/PR + weekly | Fuzz tests (30s CI, 5m scheduled) |
| `docs.yml` | push main | MkDocs build + GitHub Pages deploy |
| `action-test.yml` | push/PR | GitHub Action self-test on 3 OSes |
| `benchmark.yml` | push/PR (Go files) | Performance benchmarks with regression detection |
| `examples-test.yml` | push/PR (examples/) | Test example rules (Go, YAML, Bash) |
| `scorecard.yml` | weekly | OpenSSF security scorecard |

PR requirements: conventional commit title, all tests pass on 3 OSes, coverage maintained.

## Release Process

- **Versioning**: Semver with pre-release (`0.2.0-alpha.3`), managed by `bump-my-version`
- **Version bump**: `mise run bump:patch|minor|major|pre|release` (updates 27+ files)
- **Changelog**: Auto-generated by `git-cliff` (never edit `CHANGELOG.md` manually)
- **Tags**: Format `v<version>`, signed commits required
- **GoReleaser**: Builds binaries, archives, checksums, SBOMs, Docker images, Homebrew formula
- **Signing**: Cosign keyless signing on release checksums
- **Homebrew**: `Formula/terratidy.rb` auto-generated (never edit manually)

## GitHub Action

Defined in `action.yml` at repo root. Entry point: `tools/scripts/run-action.sh`.

**Inputs**: version, config, profile, format, parallel, skip-*, fail-on-error, fail-on-warning

**Outputs**: findings-count, errors-count, warnings-count, sarif-file

Self-tested in `action-test.yml` on Ubuntu, macOS, Windows.

## Docker

- **Base**: Alpine 3.23 (pinned digest)
- **User**: Non-root `terratidy` user
- **Health check**: `terratidy version`
- **Registry**: `ghcr.io/santosr2/terratidy`
- **Tags**: Stable releases get `latest`, `v1`, `v1.2`; pre-releases get version-only tag
