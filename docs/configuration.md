# Configuration Guide

## Configuration File

TerraTidy looks for `.terratidy.yaml` in the current directory.

## Basic Configuration

```yaml
version: 1

# Enable/disable engines
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

# Global settings
severity_threshold: warning  # info|warning|error
fail_fast: false
parallel: true
```

## Modular Configuration

For large projects, split configuration into modules:

```yaml
# .terratidy.yaml
version: 1

# Import rules from organized files
imports:
  - .terratidy/rules/**/*.yaml
  - .terratidy/engines/*.yaml
  - .terratidy/profiles/${TERRATIDY_PROFILE:-default}.yaml

severity_threshold: warning
```

## Profiles

Define different profiles for different environments:

```yaml
# .terratidy/profiles/strict.yaml
profile: strict
inherits: default

severity_threshold: error
fail_fast: true

engines:
  policy: { enabled: true }

overrides:
  rules:
    style.*:
      severity: error
```

Use with:

```bash
terratidy check --profile strict
```

## Rule Configuration

### Style Rules

```yaml
engines:
  style:
    enabled: true
    rules:
      # Spacing rules with options
      blank-line-between-blocks:
        enabled: true
        severity: warning
        options:
          min_lines: 1  # Minimum blank lines between blocks
          max_lines: 1  # Maximum blank lines between blocks

      no-empty-blocks:
        enabled: true
        options:
          allowed_blocks: ["lifecycle", "provisioner"]  # Additional allowed empty blocks
          override_defaults: false  # Set true to ignore default allowed blocks

      # Naming rules with case options
      variable-naming:
        enabled: true
        options:
          case: snake_case  # snake_case | camelCase | kebab-case | PascalCase | custom
          pattern: "^[a-z][a-z0-9_]*$"  # Custom regex (only used when case: custom)

      # Ordering rules with custom order
      variable-order:
        enabled: true
        options:
          order: ["description", "type", "default", "sensitive", "nullable"]

      output-order:
        enabled: true
        options:
          order: ["description", "value", "sensitive", "depends_on"]

      # Advanced naming (disabled by default)
      output-prefix:
        enabled: true
        options:
          prefix: ""  # Required prefix for outputs
          suffix: ""  # Required suffix for outputs

      # File organization (disabled by default)
      variables-in-file:
        enabled: true
      outputs-in-file:
        enabled: true
      providers-in-file:
        enabled: true
      scoped-file-organization:
        enabled: true
      terraform-files-structure:
        enabled: true

      # Block organization (disabled by default)
      meta-arguments-order:
        enabled: true
      lifecycle-attribute-order:
        enabled: true
      nested-block-order:
        enabled: true

      # Comment/format rules (disabled by default)
      comment-syntax:
        enabled: true
      no-trailing-whitespace:
        enabled: true
      no-consecutive-blank-lines:
        enabled: true
```

### Custom Rules

```yaml
custom_rules:
  custom.enforce_tagging:
    enabled: true
    severity: error
    config:
      required_tags:
        - "Environment"
        - "Owner"
        - "CostCenter"
```

### Naming Convention Cases

TerraTidy supports multiple naming conventions for naming rules:

| Case | Example | Description |
|------|---------|-------------|
| `snake_case` | `my_variable` | Lowercase with underscores (default) |
| `camelCase` | `myVariable` | Lowercase first word, uppercase subsequent |
| `kebab-case` | `my-variable` | Lowercase with hyphens |
| `PascalCase` | `MyVariable` | Uppercase first letter of each word |
| `custom` | (regex) | Custom pattern via `pattern` option |

## Configuration Precedence

1. CLI flags (highest)
2. Environment variables (`TERRATIDY_*`)
3. Main config overrides
4. Profile settings
5. Imported configs
6. Defaults (lowest)

## Environment Variables

- `TERRATIDY_CONFIG` - Config file path
- `TERRATIDY_PROFILE` - Profile to use
- `TERRATIDY_SEVERITY` - Severity threshold

## Commands

### Split Configuration

Convert single file to modular:

```bash
terratidy config split
```

### Show Resolved Config

See final merged configuration:

```bash
terratidy config show
```

### Validate Config

Check for errors:

```bash
terratidy config validate
```

## Examples

See [examples](../examples/) directory for complete configurations.
