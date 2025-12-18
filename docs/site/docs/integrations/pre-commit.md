# Pre-commit Hooks

TerraTidy provides pre-commit hooks for local development.

## Installation

Install pre-commit:

```bash
pip install pre-commit
```

## Configuration

Add to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-fmt
      - id: terratidy-style
      - id: terratidy-lint
```

## Available Hooks

| Hook ID | Description |
|---------|-------------|
| `terratidy-fmt` | Format Terraform files |
| `terratidy-fmt-check` | Check formatting (no changes) |
| `terratidy-style` | Check style issues |
| `terratidy-style-fix` | Check and fix style issues |
| `terratidy-lint` | Run linting checks |
| `terratidy-check` | Run all checks |
| `terratidy-fix` | Auto-fix all issues |
| `terratidy-policy` | Run policy checks |

## Examples

### Minimal Setup

Format and lint checks:

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-fmt
      - id: terratidy-lint
```

### Strict Setup

All checks with auto-fix:

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-fix
```

### CI-like Setup

Full checks without auto-fix:

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-check
```

### With Custom Config

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-check
        args: ['--config', '.terratidy-ci.yaml']
```

### With Profile

```yaml
repos:
  - repo: https://github.com/santosr2/terratidy
    rev: v0.1.0
    hooks:
      - id: terratidy-check
        args: ['--profile', 'development']
```

## Install Hooks

After adding configuration:

```bash
pre-commit install
```

## Manual Run

```bash
# Run on all files
pre-commit run --all-files

# Run specific hook
pre-commit run terratidy-lint --all-files

# Run on staged files
pre-commit run
```

## Updating Hooks

```bash
pre-commit autoupdate
```

## Troubleshooting

### Hook Not Found

Ensure TerraTidy is installed:

```bash
go install github.com/santosr2/terratidy/cmd/terratidy@latest
```

### Slow Hooks

For large projects, use:

```yaml
hooks:
  - id: terratidy-lint
    stages: [commit]
    # Skip for merge commits
    exclude: ^$
```

Or run full checks only in CI.
