# TerraTidy

<div align="center">

![TerraTidy Logo](assets/terratidy-icon.svg)

<b>A comprehensive quality platform for Terraform and Terragrunt</b>

[![Build Status](https://github.com/santosr2/terratidy/workflows/Test/badge.svg)](https://github.com/santosr2/terratidy/actions)
[![codecov](https://codecov.io/gh/santosr2/terratidy/branch/main/graph/badge.svg)](https://codecov.io/gh/santosr2/terratidy)
[![Go Report Card](https://goreportcard.com/badge/github.com/santosr2/terratidy)](https://goreportcard.com/report/github.com/santosr2/terratidy)
[![Go Version](https://img.shields.io/github/go-mod/go-version/santosr2/terratidy)](go.mod)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

## Overview

TerraTidy is a single-binary quality platform for Terraform and Terragrunt that provides:

- **Formatting** - Format `.tf` and `.hcl` files using the HCL formatter
- **Style Checking** - Custom style rules for layout, ordering, and conventions
- **Linting** - TFLint integration for best practices and errors
- **Policy Enforcement** - OPA policy checks for compliance

### Key Features

- ✅ **Single Binary** - No external dependencies, all tools vendored
- ⚡ **10-100x Faster** - Library-first architecture, no subprocess overhead
- 🔌 **Extensible** - Custom rules in Go, Rego, YAML, or Bash
- 📦 **Modular Config** - Split large configs into organized files
- 🎯 **Great DX** - Interactive setup, hot-reload dev mode, helpful errors
- 🔧 **Auto-fix** - Automatically fix formatting and style issues
- 🌐 **Multi-platform** - Linux, macOS, Windows (amd64 & arm64)

## Installation

### Homebrew (macOS/Linux)

```bash
brew install santosr2/tap/terratidy
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/santosr2/terratidy/releases).

### Docker

```bash
# latest always points to the most recent release (including pre-releases)
docker pull ghcr.io/santosr2/terratidy:latest
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check
```

### From Source

```bash
go install github.com/santosr2/terratidy/cmd/terratidy@latest
```

## Quick Start

### 1. Initialize Configuration

```bash
cd your-terraform-project
terratidy init --interactive
```

This creates a `.terratidy.yaml` configuration file with recommended settings.

### 2. Run Checks

```bash
terratidy check
```

Example output:

```text
Checking 3 files...

1. Checking formatting...
   Found 0 issue(s)

2. Checking style...
   Found 2 issue(s)

3. Running linter...
   Found 1 issue(s)

4. Running policy checks...
   Found 0 issue(s)

---
Summary: 3 total issue(s)

  Warnings: 3

Run individual commands for details:
  terratidy fmt --check
  terratidy style
  terratidy lint
  terratidy policy
```

### 3. Auto-fix Issues

```bash
terratidy fix
```

## Commands

| Command | Description |
|---------|-------------|
| `terratidy check` | Run all checks (recommended for CI) |
| `terratidy fix` | Auto-fix all fixable issues |
| `terratidy fmt` | Format files |
| `terratidy style` | Check/fix style issues |
| `terratidy lint` | Run linting |
| `terratidy policy` | Run policy checks |
| `terratidy init` | Initialize configuration |
| `terratidy dev` | Development mode with file watching |
| `terratidy lsp` | Start the Language Server Protocol server |
| `terratidy init-rule` | Initialize a new custom rule |
| `terratidy test-rule` | Test a specific rule |
| `terratidy plugins` | Plugin management commands |
| `terratidy config` | Configuration management commands |
| `terratidy rules list` | List available rules |
| `terratidy rules docs` | Generate markdown documentation |
| `terratidy version` | Show version info |

### Useful Flags

| Flag                 | Description                                                                          |
| -------------------- | ------------------------------------------------------------------------------------ |
| `--parallel` / `-p`  | Run engines in parallel (faster)                                                     |
| `--changed`          | Only check files changed in git                                                      |
| `--format`           | Output format: text, table, json, json-compact, sarif, html, junit, markdown, github |
| `--skip-fmt`         | Skip formatting checks                                                               |
| `--skip-style`       | Skip style checks                                                                    |
| `--skip-lint`        | Skip linting checks                                                                  |
| `--skip-policy`      | Skip policy checks                                                                   |

## Configuration

### Simple Configuration

```yaml
# .terratidy.yaml
version: 1

engines:
  fmt: { enabled: true }
  style: { enabled: true }
  lint: { enabled: true }
  policy: { enabled: false }

severity_threshold: warning
```

### Modular Configuration (for large projects)

```yaml
# .terratidy.yaml
version: 1

# Import rules from organized files
imports:
  - .terratidy/rules/**/*.yaml
  - .terratidy/profiles/default.yaml

severity_threshold: warning
```

See [Configuration Guide](docs/site/docs/getting-started/configuration.md) for details.

## Integrations

| Method | When | Best For |
| -------------- | -------------------- | --------------------------------- |
| CLI | Manual runs | Local development, scripting |
| Pre-commit | On git commit | Catching issues before push |
| GitHub Actions | On PR/push | CI/CD quality gates |
| LSP / VS Code | Real-time in editor | Instant feedback while coding |
| Docker | Isolated environments | CI pipelines without Go installed |

### Pre-commit Hook

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.2.0-alpha.3  # or use a stable version when available
    hooks:
      - id: terratidy-check
```

### GitHub Action

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v0  # Floating tag, tracks latest v0.x release
  with:
    format: sarif
    parallel: true
    github-token: ${{ secrets.GITHUB_TOKEN }}
```

Pin to a specific release for reproducible builds: `santosr2/terratidy@v0.2.0-alpha.3`

Available inputs: `version`, `config`, `profile`, `format`, `parallel`, `working-directory`,
`skip-fmt`, `skip-style`, `skip-lint`, `skip-policy`, `fail-on-error`, `fail-on-warning`, `github-token`.

### VSCode Extension

The TerraTidy VSCode extension provides real-time diagnostics via LSP.
See [vscode/README.md](vscode/README.md) for installation instructions.

## Custom Rules

Create custom rules in three formats:

### Go Plugin (most powerful)

```go
package custom

func (r *EnforceTaggingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    // Full HCL AST access
}
```

### YAML Rule (simple, declarative)

```yaml
rule: custom.naming_convention
pattern:
  block_type: resource
  conditions:
    - attribute: "name"
      regex: "^[a-z][a-z0-9_]*$"
```

### Bash Script (quick prototypes)

```bash
#!/usr/bin/env bash
# Output JSON findings to stdout
```

See [Custom Rules Guide](docs/site/docs/rules/custom-rules.md) for details.

## Documentation

- [Installation](docs/site/docs/getting-started/installation.md)
- [Configuration](docs/site/docs/getting-started/configuration.md)
- [Architecture](docs/site/docs/development/architecture.md)
- [Linting](docs/site/docs/user-guide/engines/lint.md)
- [Contributing](CONTRIBUTING.md)

Full documentation is available at [docs/site/docs/](docs/site/docs/).

## Development

### Setup

```bash
git clone https://github.com/santosr2/terratidy
cd terratidy
mise install        # Install Go 1.26.1 and tools
mise run setup      # Install dev tools (repomix, air, etc.)
make build          # Build binary
```

### Run Tests

```bash
make test           # Unit tests
make integration    # Integration tests
make lint           # Run linters
make check          # All checks
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:

- [HashiCorp HCL](https://github.com/hashicorp/hcl) for parsing
- [TFLint](https://github.com/terraform-linters/tflint) for linting
- [Open Policy Agent](https://github.com/open-policy-agent/opa) for policies
- [Cobra](https://github.com/spf13/cobra) for CLI

## Support

- 📝 [Documentation](docs/site/docs/)
- 🐛 [Issue Tracker](https://github.com/santosr2/terratidy/issues)
- 💬 [Discussions](https://github.com/santosr2/terratidy/discussions)
