# Upgrade Guide

How to upgrade TerraTidy between versions.

## Checking Your Version

```bash
terratidy version
terratidy version --short   # Just the version number
terratidy version --json    # Machine-readable
```

## Upgrading

### Go Install

```bash
go install github.com/santosr2/TerraTidy/cmd/terratidy@latest
```

### Homebrew

```bash
brew upgrade terratidy
```

### Docker

```bash
docker pull ghcr.io/santosr2/terratidy:latest
```

### Pre-commit

Update the `rev` in `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/santosr2/TerraTidy
    rev: v0.2.0-alpha.4  # Update this
    hooks:
      - id: terratidy-check
```

Or auto-update:

```bash
pre-commit autoupdate
```

## Version Compatibility

### Config Version

TerraTidy currently supports `version: 1` in configuration files. Future major versions
may introduce a new config format.

```yaml
version: 1  # Required
```

### Go Version

TerraTidy requires Go 1.26.1 or later for building from source and compiling Go plugins.
Go plugins (`.so` files) must be compiled with the same Go version as TerraTidy.

### OPA Version

The policy engine uses OPA v1.15.0 with Rego v1 syntax. Policies must use
`import rego.v1` and the `contains`/`if` keywords.

## Breaking Changes

### Pre-release to Stable

When TerraTidy reaches v1.0.0, expect:

- Config `version: 1` will remain supported
- New `version: 2` config format may be introduced
- Deprecated features will be removed with migration guidance
- `terratidy config validate` will warn about deprecated options

## Validating After Upgrade

After upgrading, verify your setup:

```bash
# Validate config
terratidy config validate

# Run checks and compare output
terratidy check --format json > post-upgrade.json

# Check version
terratidy version
```
