# Configuration Profiles

Profiles allow you to define multiple configurations for different environments or use cases.

## Overview

Profiles enable you to:

- Define different rule sets for development vs CI
- Create team-specific configurations
- Switch between strict and relaxed checking

## Defining Profiles

Add profiles to your `.terratidy.yaml`:

```yaml
version: 1

# Default configuration
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

# Profile definitions
profiles:
  ci:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true

  development:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: false
      policy:
        enabled: false

  strict:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true
    overrides:
      rules:
        style.block-label-case:
          severity: error
```

## Using Profiles

### Command Line

```bash
# Use a specific profile
terratidy check --profile ci

# Override severity threshold
terratidy check --profile ci --severity-threshold error
```

### CLI Flag

```bash
terratidy check --profile ci
```

### VS Code

Configure in settings:

```json
{
  "terratidy.profile": "development"
}
```

## Profile Inheritance

Profiles inherit from the base configuration and override specific settings:

```yaml
version: 1

# Base configuration
engines:
  fmt:
    enabled: true
  style:
    enabled: true

profiles:
  # Inherits fmt and style from base, adds lint
  ci:
    engines:
      lint:
        enabled: true

  # Overrides rules from base
  strict:
    engines:
      lint:
        enabled: true
    overrides:
      rules:
        style.block-label-case:
          severity: error
```

## Common Profile Patterns

### Development Profile

Fast feedback with minimal checks:

```yaml
profiles:
  development:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: false
      policy:
        enabled: false
```

### CI Profile

Comprehensive checks for pull requests:

```yaml
profiles:
  ci:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true
```

### Pre-commit Profile

Quick checks for commit hooks:

```yaml
profiles:
  pre-commit:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: false
      policy:
        enabled: false
```

### Security Profile

Security-focused checks only:

```yaml
profiles:
  security:
    engines:
      fmt:
        enabled: false
      style:
        enabled: false
      lint:
        enabled: true
        rules:
          security-group-unrestricted:
            enabled: true
            severity: error
      policy:
        enabled: true
        config:
          policy_dirs:
            - ./policies/security
```

## Team Profiles

Share profiles across your organization:

```yaml
# .terratidy.yaml in your module
version: 1

imports:
  - .terratidy/*.yaml

profiles:
  team-a:
    inherits: ci
    engines:
      policy:
        config:
          policy_dirs:
            - ./policies/team-a
```

## Viewing Profile Configuration

```bash
# Show resolved configuration for a profile
terratidy config show --profile ci
```
