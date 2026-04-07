<div align="center">

<picture>
  <img src="assets/brand.png" alt="TerraTidy" width="400" style="max-width: 100%; height: auto;">
</picture>

<b>A comprehensive quality platform for Terraform and Terragrunt</b>

[![Latest Release](https://img.shields.io/github/v/release/santosr2/terratidy)](https://github.com/santosr2/TerraTidy/releases/latest)
[![Build Status](https://github.com/santosr2/TerraTidy/workflows/Test/badge.svg)](https://github.com/santosr2/TerraTidy/actions)
[![codecov](https://codecov.io/gh/santosr2/terratidy/branch/main/graph/badge.svg)](https://codecov.io/gh/santosr2/terratidy)
[![Go Report Card](https://goreportcard.com/badge/github.com/santosr2/TerraTidy)](https://goreportcard.com/report/github.com/santosr2/TerraTidy)
[![Go Version](https://img.shields.io/github/go-mod/go-version/santosr2/terratidy)](go.mod)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/santosr2/TerraTidy/badge)](https://scorecard.dev/viewer/?uri=github.com/santosr2/TerraTidy)

</div>

## Overview

TerraTidy is a single-binary quality platform for Terraform and Terragrunt that provides:

- **Formatting** -- Format `.tf` and `.hcl` files using the HCL formatter
- **Style Checking** -- Custom style rules for layout, ordering, and conventions
- **Linting** -- 11 built-in AST rules plus optional TFLint integration for provider-specific checks
- **Policy Enforcement** -- OPA policy checks for compliance

### Key Features

- **Single Binary** -- No external dependencies for core functionality
- **Library-first Architecture** -- Uses Go libraries (hclwrite, OPA SDK) directly instead of shelling out
- **Extensible** -- Custom rules in Go, YAML, Rego, or Bash
- **Modular Config** -- Split large configs into organized files with glob imports
- **Auto-fix** -- Automatically fix formatting and style issues
- **Multiple Output Formats** -- Text, table, JSON, SARIF, HTML, JUnit, Markdown, GitHub Actions annotations
- **Multi-platform** -- Linux, macOS, Windows (amd64 and arm64)

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap santosr2/tap https://github.com/santosr2/TerraTidy
brew install santosr2/tap/terratidy
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/santosr2/TerraTidy/releases).

### Docker

```bash
docker pull ghcr.io/santosr2/terratidy:latest

# Pin to a specific version in CI
docker pull ghcr.io/santosr2/terratidy:v0.2.0-alpha.4

docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check
```

### From Source

```bash
go install github.com/santosr2/TerraTidy/cmd/terratidy@latest
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

Example output (sequential mode, the default):

```text
Checking 3 files...

1. Checking formatting...
   Found 1 issue(s)

2. Checking style...
   Found 2 issue(s)

3. Running linter...
   Found 1 issue(s)

4. Running policy checks...
   Found 0 issue(s)

✗ modules/networking/main.tf:0:0: File needs formatting (fmt.needs-formatting)
⚠ modules/networking/main.tf:12:1: Missing blank line between blocks (style.blank-line-between-blocks)
⚠ modules/networking/variables.tf:5:1: Missing blank line between blocks (style.blank-line-between-blocks)
⚠ modules/networking/main.tf:8:1: resource name 'public-subnet' should use snake_case (lint.terraform-naming-convention)
---
Summary: 4 total issue(s)

  Errors:   1
  Warnings: 3

Run individual commands for details:
  terratidy fmt --check
  terratidy style
  terratidy lint
  terratidy policy
```

With `--parallel` (`-p`), the output is more compact:

```text
Checking 3 files...

Running checks in parallel mode...
  fmt: 1 issue(s)
  style: 2 issue(s)
  lint: 1 issue(s)

✗ modules/networking/main.tf:0:0: File needs formatting (fmt.needs-formatting)
⚠ modules/networking/main.tf:12:1: Missing blank line between blocks (style.blank-line-between-blocks)
⚠ modules/networking/variables.tf:5:1: Missing blank line between blocks (style.blank-line-between-blocks)
⚠ modules/networking/main.tf:8:1: resource name 'public-subnet' should use snake_case (lint.terraform-naming-convention)
---
Summary: 4 total issue(s)

  Errors:   1
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

| Command                | Description                               |
|------------------------|-------------------------------------------|
| `terratidy check`      | Run all checks (recommended for CI)       |
| `terratidy fix`        | Auto-fix all fixable issues               |
| `terratidy fmt`        | Format files                              |
| `terratidy style`      | Check/fix style issues                    |
| `terratidy lint`       | Run linting                               |
| `terratidy policy`     | Run policy checks                         |
| `terratidy init`       | Initialize configuration                  |
| `terratidy dev`        | Development mode with file watching       |
| `terratidy lsp`        | Start the Language Server Protocol server |
| `terratidy init-rule`  | Initialize a new custom rule              |
| `terratidy test-rule`  | Test a specific rule                      |
| `terratidy plugins`    | Plugin management commands                |
| `terratidy config`     | Configuration management commands         |
| `terratidy rules list` | List available rules                      |
| `terratidy rules docs` | Generate markdown documentation           |
| `terratidy version`    | Show version info                         |

### Global Flags

These flags apply to all commands:

| Flag                       | Description                                                                          |
| -------------------------- | ------------------------------------------------------------------------------------ |
| `--config`                 | Path to config file (default: `.terratidy.yaml`)                                     |
| `--profile`                | Configuration profile to use                                                         |
| `--format`                 | Output format: text, table, json, json-compact, sarif, html, junit, markdown, github |
| `--changed`                | Only check files changed in git                                                      |
| `--no-recurse`             | Disable recursive directory traversal                                                |
| `--exclude`                | Glob patterns to exclude (repeatable or comma-separated)                             |
| `--severity-threshold`     | Minimum severity to fail: info, warning, error                                       |
| `--color`                  | Enable colored output (default: true)                                                |
| `--absolute-paths`         | Output absolute file paths instead of relative                                       |

### Check Command Flags

These flags are specific to `terratidy check`:

| Flag                 | Description                  |
| -------------------- | ---------------------------- |
| `--parallel` / `-p`  | Run engines in parallel      |
| `--skip-fmt`         | Skip formatting checks       |
| `--skip-style`       | Skip style checks            |
| `--skip-lint`        | Skip linting checks          |
| `--skip-policy`      | Skip policy checks           |

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

### Modular Configuration

For larger projects, split configuration into organized files:

```yaml
# .terratidy.yaml
version: 1

imports:
  - .terratidy/rules/*.yaml
  - .terratidy/profiles/default.yaml

severity_threshold: warning
```

See the [Configuration Guide](docs/site/docs/getting-started/configuration.md) for details.

## Integrations

| Method         | When                 | Best For                          |
| -------------- | -------------------- | --------------------------------- |
| CLI            | Manual runs          | Local development, scripting      |
| Pre-commit     | On git commit        | Catching issues before push       |
| GitHub Actions | On PR/push           | CI/CD quality gates               |
| LSP / VS Code  | Real-time in editor  | Instant feedback while coding     |
| Docker         | Isolated environments| CI pipelines without Go installed |

### Pre-commit Hook

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/santosr2/TerraTidy
    rev: v0.2.0-alpha.4
    hooks:
      - id: terratidy-check
```

Available hook IDs: `terratidy-fmt`, `terratidy-fmt-check`, `terratidy-style`, `terratidy-style-fix`, `terratidy-lint`, `terratidy-check`, `terratidy-fix`, `terratidy-policy`.

### GitHub Action

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v0
  with:
    format: sarif
    parallel: 'true'
    github-token: ${{ secrets.GITHUB_TOKEN }}
```

Pin to a specific release for reproducible builds: `santosr2/terratidy@v0.2.0-alpha.4`

Available inputs: `version`, `config`, `profile`, `format`, `parallel`, `working-directory`,
`skip-fmt`, `skip-style`, `skip-lint`, `skip-policy`, `exclude`, `no-recurse`, `absolute-paths`,
`changed`, `severity-threshold`, `fail-on-error`, `fail-on-warning`, `github-token`.

### VS Code Extension

The TerraTidy VS Code extension provides real-time diagnostics via LSP.
See [vscode/README.md](vscode/README.md) for installation instructions.

## Custom Rules

Create custom rules in three formats:

### Go Plugin

```go
package main

import (
    "github.com/hashicorp/hcl/v2"
    "github.com/santosr2/TerraTidy/pkg/sdk"
)

type MyRule struct{}

func (r *MyRule) Name() string        { return "my-rule" }
func (r *MyRule) Description() string { return "Checks something" }

func (r *MyRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    var findings []sdk.Finding
    // Full HCL AST access via file.Body
    return findings, nil
}
```

### YAML Rule

```yaml
name: require-description
description: Resources must have a description attribute
severity: warning
enabled: true
message: "Resource is missing a description attribute"
patterns:
  resource_types:
    - aws_instance
    - aws_s3_bucket
  required_attributes:
    - description
```

### Bash Script

```bash
#!/usr/bin/env bash
# Output JSON findings to stdout
```

See the [Custom Rules Guide](docs/site/docs/rules/custom-rules.md) for details.

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
git clone https://github.com/santosr2/TerraTidy
cd terratidy
mise install        # Install Go 1.26.1 and tools
mise run setup      # Download and tidy Go modules
mise run build      # Build binary
```

### Run Tests

```bash
mise run test              # Unit tests
mise run test:integration  # Integration tests
mise run lint              # Run linters
mise run check             # All quality checks (fmt, vet, lint, test)
```

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT License -- see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:

- [HashiCorp HCL](https://github.com/hashicorp/hcl) for parsing
- [TFLint](https://github.com/terraform-linters/tflint) for optional provider-specific linting (invoked as subprocess)
- [Open Policy Agent](https://github.com/open-policy-agent/opa) for policies
- [Cobra](https://github.com/spf13/cobra) for CLI

## Support

- [Documentation](docs/site/docs/)
- [Issue Tracker](https://github.com/santosr2/TerraTidy/issues)
- [Discussions](https://github.com/santosr2/TerraTidy/discussions)
