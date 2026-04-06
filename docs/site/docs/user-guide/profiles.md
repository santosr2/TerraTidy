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
    description: "Full checks for CI pipelines"
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
    description: "Fast feedback for local development"
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

### Implicit Inheritance (from base config)

All profiles implicitly inherit from the base configuration. When you apply a profile, its settings override the base:

```yaml
version: 1

# Base configuration
engines:
  fmt:
    enabled: true
  style:
    enabled: true

profiles:
  # Overrides base config: adds lint, keeps fmt and style from base
  ci:
    engines:
      lint:
        enabled: true
```

### Explicit Inheritance (from another profile)

Use the `inherits` field to inherit from another profile:

```yaml
version: 1

profiles:
  base:
    description: "Base profile with standard checks"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true

  # Inherits all settings from 'base', overrides policy
  strict:
    description: "Strict profile for production"
    inherits: base
    engines:
      policy:
        enabled: true
    overrides:
      rules:
        style.block-label-case:
          severity: error
```

Child profile settings take precedence over parent profile settings.

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
      policy:
        enabled: true
        policy_dirs:
          - ./policies/security
    overrides:
      rules:
        security-group-unrestricted:
          enabled: true
          severity: error
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
        policy_dirs:
          - ./policies/team-a
```

## Viewing Profile Configuration

```bash
# Show resolved configuration for a profile
terratidy config show --profile ci
```
