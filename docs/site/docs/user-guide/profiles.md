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
    severity_threshold: warning
    fail_fast: true

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
    severity_threshold: error

  strict:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
        rules:
          resource-naming:
            enabled: true
            severity: error
      lint:
        enabled: true
      policy:
        enabled: true
    severity_threshold: info
```

## Using Profiles

### Command Line

```bash
# Use a specific profile
terratidy check --profile ci

# Override profile settings
terratidy check --profile ci --severity error
```

### Environment Variable

```bash
export TERRATIDY_PROFILE=ci
terratidy check
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
    config:
      naming_convention: snake_case

profiles:
  # Inherits fmt and style from base, adds lint
  ci:
    engines:
      lint:
        enabled: true

  # Overrides style config from base
  strict:
    engines:
      style:
        config:
          naming_convention: snake_case
          enforce_ordering: true
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
    severity_threshold: error
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
    severity_threshold: warning
    fail_fast: false
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
    fail_fast: true
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

extends:
  - https://example.com/terratidy/base-config.yaml

profiles:
  team-a:
    extends: ci
    engines:
      policy:
        config:
          policy_dirs:
            - ./policies/team-a
```

## Listing Available Profiles

```bash
# List all profiles
terratidy config profiles

# Show profile details
terratidy config show --profile ci
```
