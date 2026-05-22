# Upgrade Guide

How to upgrade TerraTidy between versions.

## Checking Your Version

```bash
terratidy version
terratidy version --short        # Just the version number
terratidy version --format json  # Machine-readable
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

TerraTidy requires Go 1.26.3 or later for building from source and compiling Go plugins.
Go plugins (`.so` files) must be compiled with the same Go version as TerraTidy.

### OPA Version

The policy engine uses OPA v1.15.0 with Rego v1 syntax. Policies must use
`import rego.v1` and the `contains`/`if` keywords.

## Breaking Changes

### v0.2.0-alpha.5: Distinct Exit Codes

Exit codes now distinguish between different error types:

| Code | Before | After |
|------|--------|-------|
| `0`  | Success | Success |
| `1`  | All errors | Findings found only |
| `2`  | N/A | Configuration errors |
| `3`  | N/A | Internal errors |

**Migration:** If your CI/CD scripts check for non-zero exit codes, update them to handle
the new codes appropriately:

```bash
# Before: "if exit != 0 then fail"
# After: distinguish error types
terratidy check
case $? in
  0) echo "Pass" ;;
  1) echo "Findings found - fail the build" ;;
  2) echo "Config error - fail the build" ;;
  3) echo "Internal error - fail the build" ;;
esac
```

Most scripts that just check for non-zero will still work correctly.

### v0.2.0-alpha.5: CLI Flag Shorthand Reassignments

Short flags have been reassigned to more commonly used global flags:

| Short | Before | After |
|-------|--------|-------|
| `-p`  | `check --parallel` | `--profile` (global) |
| `-f`  | `init --force` | `--format` (global) |
| `-c`  | N/A | `--config` (global) |

**Migration:** Update any scripts using the old shorthands:

```bash
# Before
terratidy check -p           # Meant --parallel
terratidy init -f            # Meant --force

# After
terratidy check --parallel   # Use long form
terratidy init --force       # Use long form
```

### v0.2.0-alpha.5: Version Command JSON Output

The `version --json` flag has been replaced with `version --format json`:

```bash
# Before
terratidy version --json

# After
terratidy version --format json
```

JSON field names changed to snake_case for consistency:

| Before | After |
|--------|-------|
| `goVersion` | `go_version` |

### v0.2.0-alpha.5: SDK `Finding.Fix` Replaced with `Fixable` Flag

Applies to authors of Go SDK plugins (`pkg/sdk`). The `Finding.Fix *FixResult`
field and the `FixResult` struct have been removed. `Check()` no longer
precomputes fix content; the style engine calls `Fixer.Fix()` lazily, only when
running in fix or diff mode.

| Before | After |
|--------|-------|
| `Finding.Fix *FixResult` | `Finding.Fixable bool` |
| `FixResult.Content []byte` | Removed — call `Fixer.Fix()` to obtain content |
| `FixResult.Diff string` | Use `Finding.Message` + `Finding.IsDiff bool` |

**Migration for SDK plugin authors:**

```go
// Before: Check() returned a Finding with precomputed fix content.
func (r *MyRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    fixed := applyFix(file)
    return []sdk.Finding{{
        Rule:    "my-rule",
        Message: "issue found",
        Fix:     &sdk.FixResult{Content: fixed},
    }}, nil
}

// After: Check() reports findings only; Fix() produces the corrected bytes.
func (r *MyRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    return []sdk.Finding{{
        Rule:    "my-rule",
        Message: "issue found",
    }}, nil
}

func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
    return applyFix(file), nil
}
```

The engine sets `Finding.Fixable = true` automatically for any rule that
implements `sdk.Fixer`. The field is read-only from a plugin perspective; the
engine populates it.

**Migration for finding consumers (formatters, IDE integrations):**

```go
// Before: nil-check on the pointer field.
for _, f := range findings {
    if f.Fix != nil {
        renderFixable(f, f.Fix.Diff)
    } else {
        renderPlain(f)
    }
}

// After: read the boolean flag; if the message carries a diff, route it
// through a diff renderer.
for _, f := range findings {
    switch {
    case f.IsDiff:
        renderDiff(f, f.Message)
    case f.Fixable:
        renderFixable(f)
    default:
        renderPlain(f)
    }
}
```

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
