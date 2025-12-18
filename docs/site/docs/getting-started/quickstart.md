# Quick Start

Get started with TerraTidy in under 5 minutes.

## Initialize Configuration

Create a `.terratidy.yaml` configuration file in your project:

```bash
terratidy init
```

This creates a default configuration:

```yaml
version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

severity_threshold: warning
```

## Run Checks

### Check All Files

```bash
terratidy check
```

### Check Specific Files or Directories

```bash
terratidy check --paths ./modules/
terratidy check --paths main.tf,variables.tf
```

### Check Only Changed Files (Git)

```bash
terratidy check --changed
```

## Run Individual Engines

```bash
# Format only
terratidy fmt

# Style checks only
terratidy style

# Lint only
terratidy lint

# Policy checks only
terratidy policy
```

## Fix Issues

### Auto-fix All Fixable Issues

```bash
terratidy fix
```

### Fix Formatting Only

```bash
terratidy fmt
```

### Fix Style Issues

```bash
terratidy style --fix
```

## Output Formats

```bash
# Default text output
terratidy check

# JSON output
terratidy check --format json

# SARIF for GitHub Code Scanning
terratidy check --format sarif

# HTML report
terratidy check --format html > report.html
```

## Common Workflows

### CI/CD Check

```bash
terratidy check --format sarif --severity-threshold error
```

### Pre-commit Hook

```bash
terratidy check --changed
```

### Development Workflow

```bash
# Format and fix issues
terratidy fix

# Verify all checks pass
terratidy check
```

## Next Steps

- [Configuration](configuration.md) - Customize TerraTidy
- [Commands Reference](../user-guide/commands.md) - All available commands
- [GitHub Actions](../integrations/github-actions.md) - CI/CD integration
