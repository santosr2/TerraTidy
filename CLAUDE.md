# TerraTidy

Single-binary Terraform/Terragrunt quality platform. Go 1.25+ (dev: 1.26.1), library-first, extensible plugin system.

## Architecture

```text
cmd/terratidy/          # CLI (Cobra)
internal/
  benchmark/            # Benchmarking utilities
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
examples/               # Example configs and custom rules
Formula/                # Homebrew formula
vscode/                 # VS Code extension
tools/scripts/          # Development scripts
```

## Core Interfaces

```go
// Every engine implements this (internal/runner/runner.go)
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
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false  # Opt-in for policy checking

# Global settings
severity_threshold: warning  # info|warning|error
fail_fast: false
parallel: true

imports:
  - .terratidy/*.yaml    # Glob patterns for modular configs

profiles:
  production:
    engines:
      policy:
        enabled: true

plugins:
  enabled: true
  directories:
    - .terratidy/plugins
    - ~/.terratidy/plugins

overrides:
  rules:
    my-rule:
      enabled: true
      severity: error

custom_rules:
  my-custom-rule:
    enabled: true
    severity: warning
```

## Development

```bash
mise install              # Install Go 1.26.1 + tools
make setup                # Install dependencies
make build                # Build binary
make test                 # Unit tests
make integration          # Integration tests
make lint                 # golangci-lint
make init-rule NAME=x TYPE=go|rego|yaml   # Scaffold new rule (default: rego)
```

## Testing

- Table-driven tests with testify (`require`, `assert`)
- Fixture-based tests for HCL parsing/formatting
- Integration tests for CLI commands
- Benchmarks for performance-critical paths (`go test -bench=. -benchmem`)
- Target: 80%+ coverage

### Key Packages

| Package | Tests |
| --- | --- |
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
    "github.com/santosr2/terratidy/pkg/sdk"
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
pattern:
  type: resource
  missing_attribute: description
message: "Resource is missing a description attribute"
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
