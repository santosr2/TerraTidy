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
style:
  rules:
    style.block_order_resource_module:
      enabled: true
      severity: error
      config:
        order:
          - "count|for_each"
          - "source|version"
          - "arguments"
          - "tags"
          - "lifecycle"
        blank_line_between_groups: true
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

