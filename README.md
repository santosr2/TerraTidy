# TerraTidy

<div align="center">

![TerraTidy Logo](assets/terratidy-icon.svg)

**A comprehensive quality platform for Terraform and Terragrunt**

[![Build Status](https://github.com/santosr2/terratidy/workflows/Test/badge.svg)](https://github.com/santosr2/terratidy/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/santosr2/terratidy)](https://goreportcard.com/report/github.com/santosr2/terratidy)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

## Overview

TerraTidy is a single-binary quality platform for Terraform and Terragrunt that provides:

- **Formatting** - Format `.tf` and `.hcl` files using the HCL formatter
- **Style Checking** - Custom style rules for layout, ordering, and conventions
- **Linting** - TFLint integration for best practices and errors
- **Policy Enforcement** - OPA policy checks for compliance

### Key Features

✅ **Single Binary** - No external dependencies, all tools vendored  
⚡ **10-100x Faster** - Library-first architecture, no subprocess overhead  
🔌 **Extensible** - Custom rules in Go, YAML, or Bash  
📦 **Modular Config** - Split large configs into organized files  
🎯 **Great DX** - Interactive setup, hot-reload dev mode, helpful errors  
🔧 **Auto-fix** - Automatically fix formatting and style issues  
🌐 **Multi-platform** - Linux, macOS, Windows (amd64 & arm64)

## Installation

### Homebrew (macOS/Linux)

```bash
brew install santosr2/tap/terratidy
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/santosr2/terratidy/releases).

### Docker

```bash
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

```
✅ fmt: All files formatted correctly (23 files)
⚠️  style: 3 issues found
  ├─ main.tf:12:1 [style.block_order] resource arguments out of order
  ├─ variables.tf:5:1 [style.blank_line] missing blank line between variables
  └─ outputs.tf:8:1 [style.naming] output name should be snake_case
✅ lint: No issues (ran 47 rules)
✅ policy: All policies passed

💡 Run 'terratidy fix' to auto-fix 2/3 style issues
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
| `terratidy config split` | Split config into modules |
| `terratidy rules list` | List available rules |
| `terratidy version` | Show version info |

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

See [Configuration Guide](docs/configuration.md) for details.

## Integrations

### Pre-commit Hook

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-check
```

### GitHub Action

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v1
  with:
    command: check
    format: sarif
    upload_sarif: true
```

### VSCode Extension

Search for "TerraTidy" in the VSCode marketplace or install from [here](https://marketplace.visualstudio.com/items?itemName=santosr2.terratidy).

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

See [Custom Rules Guide](docs/custom-rules.md) for details.

## Documentation

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Rules Catalog](docs/rules.md)
- [Custom Rules](docs/custom-rules.md)
- [Integrations](docs/integrations.md)
- [Development](docs/development.md)
- [Architecture](docs/architecture.md)

## Development

### Setup

```bash
git clone https://github.com/santosr2/terratidy
cd terratidy
mise install    # Install Go 1.25 and tools
make setup      # Install dependencies
make build      # Build binary
```

### Run Tests

```bash
make test           # Unit tests
make integration    # Integration tests
make lint           # Run linters
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

- 📝 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/santosr2/terratidy/issues)
- 💬 [Discussions](https://github.com/santosr2/terratidy/discussions)
