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
    rev: v0.3.0  # Update this
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

TerraTidy requires Go 1.25 or later to build from source; official releases are built
with Go 1.26.4. Go plugins (`.so` files) must be compiled with the same Go version as
the TerraTidy binary you run them against.

### OPA Version

The policy engine uses OPA v1.x with Rego v1 syntax. Policies must use
`import rego.v1` and the `contains`/`if` keywords.

## Breaking Changes

### v0.2.0: Distinct Exit Codes

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

### v0.2.0: CLI Flag Shorthand Reassignments

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

### v0.2.0: Version Command JSON Output

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

### v0.2.0: SDK `Finding.Fix` Replaced with `Fixable` Flag

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

> **Note:** This intermediate signature was a development step that never
> shipped in a release. It was superseded before v0.2.0 by the byte-range edits
> change documented below — see the next upgrade section. It is recorded here
> only to explain how the API arrived at its current shape; do not copy this
> signature for new code.

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
    return applyFix(file), nil // your existing fix logic, returning rewritten bytes
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

### v0.2.0: SDK `Fixer.Fix` Returns `*FixResult` Instead of `[]byte`

Applies to authors of Go SDK plugins (`pkg/sdk`). The `Fixer.Fix` method now
returns a `*FixResult` carrying one or more byte-range `TextEdit`s, replacing
the previous "rewrite the whole file as `[]byte`" contract. This lets the
engine collect edits from every fixable finding in a file and apply them in a
single write, instead of looping one rule at a time.

YAML and Bash rule authors are **not affected** — those rule types don't write
Go and their plugin stubs remain non-fixing.

| Before | After |
|--------|-------|
| `Fix(ctx, file) ([]byte, error)` | `Fix(ctx, file) (*FixResult, error)` |
| Return `nil` for no-op | Return `nil` *or* `&FixResult{}` with no edits |
| Engine rewrites the whole file each pass | Engine splices byte ranges in descending `Start` order |

!!! note "`FixResult` name reuse"
    The `FixResult` type name was previously used in the SDK (pre-v0.2.0)
    for an unrelated type (`{Content []byte; Diff string}`) that has since been
    removed — see [v0.2.0: SDK `Finding.Fix` Replaced with `Fixable` Flag](#v020-sdk-findingfix-replaced-with-fixable-flag).
    The new `FixResult` introduced here has a different shape (`{Edits []TextEdit}`)
    and is unrelated to the old type. A reader walking the upgrade history
    chronologically sees the removal first, then the reintroduction.

**Migration for SDK plugin authors:**

The fastest migration is to wrap your existing whole-file output as a single
`TextEdit` spanning the full input range. This preserves your rule's existing
algorithm; the engine handles the byte-range splice for you.

```go
import (
    "bytes"

    "github.com/hashicorp/hcl/v2"
    "github.com/santosr2/TerraTidy/pkg/sdk"
)

// Before: return the rewritten file as []byte.
func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
    return applyFix(file), nil // your existing fix logic, returning rewritten bytes
}

// After: wrap the rewritten content as a whole-file TextEdit; return nil for
// no-op (engine treats nil result or empty Edits the same as no fix).
func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) (*sdk.FixResult, error) {
    original := file.Bytes
    fixed := applyFix(file) // your existing fix logic, returning rewritten bytes
    if bytes.Equal(original, fixed) {
        return nil, nil
    }
    return &sdk.FixResult{
        Edits: []sdk.TextEdit{{
            Start:       0,
            End:         len(original),
            Replacement: fixed,
        }},
    }, nil
}
```

The `bytes.Equal` guard matches the prior contract where rules returned nil
bytes to signal "no change". TerraTidy's in-repo style rules use a shared
`WholeFileEdit` helper at `internal/engines/style/rules/helpers.go` that
collapses these lines — feel free to copy it into your own plugin module if
you have many rules following this pattern.

**Producing narrow edits (recommended for new rules):**

```go
func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) (*sdk.FixResult, error) {
    return &sdk.FixResult{
        Edits: []sdk.TextEdit{
            // Delete bytes [42, 50).
            {Start: 42, End: 50, Replacement: nil},
            // Insert " # tidied" at offset 100.
            {Start: 100, End: 100, Replacement: []byte(" # tidied")},
        },
    }, nil
}
```

Offsets are half-open `[Start, End)`. The engine sorts edits by `Start`
descending before applying, so earlier splices don't invalidate later offsets.
Out-of-bounds edits (`End > len(content)`) are rejected with an error. Return
`nil, nil` (or `&sdk.FixResult{}` with no edits) when no fix is applicable;
the engine treats both forms identically as a no-op.

**Whole-file exclusivity:** if any edit in a pass spans the entire file
(`Start == 0 && End == len(content)`), it is applied alone — all other edits
in the same pass are discarded and re-emit against the rewritten content on
the next pass. This avoids ambiguous interactions between whole-file rewriters
and narrow edits.

See the SDK reference for the full contract:
[`Fixer` interface](../development/sdk.md#rule-interface),
[`TextEdit`](../development/sdk.md#textedit), and
[`FixResult`](../development/sdk.md#fixresult) — including the apply ordering,
overlap rules, and field semantics.

### v0.3.0: Naming-Convention Rules Consolidated into the Style Engine

Name-casing checks used to be spread across four loosely named style rules plus one
lint rule. They now live in the style engine under a single `<block>-name-convention`
naming scheme, one rule per block type.

| Before | After |
|--------|-------|
| `style.block-label-case` | `style.resource-name-convention` and `style.data-name-convention` |
| `style.variable-naming` | `style.variable-name-convention` |
| `style.output-naming` | `style.output-name-convention` |
| `style.local-naming` | `style.local-name-convention` |
| `lint.terraform-naming-convention` | Removed — the style rules above cover it |

`style.block-label-case` checked resources and data sources with one rule ID; those are
now two rules, so a resource-only or data-only override is possible for the first time.
`style.module-name-descriptive` is new: it flags generic module names like `this` or
`main` at `info` severity, and is opt-in. The pre-existing `style.module-name-convention`,
which checks module name casing, is unchanged.

**Migration:** rename the rule keys under `engines.style.rules` in `.terratidy.yaml`, and
update any `terratidy:ignore` annotations and SARIF rule-ID filters that reference the old
IDs.

```yaml
# Before
engines:
  style:
    rules:
      style.block-label-case: { enabled: true, severity: error }
      style.variable-naming:
        enabled: true
        config:
          options:
            case: snake_case
  lint:
    rules:
      lint.terraform-naming-convention: { enabled: false }

# After
engines:
  style:
    rules:
      style.resource-name-convention: { enabled: true, severity: error }
      style.data-name-convention: { enabled: true, severity: error }
      style.variable-name-convention:
        enabled: true
        config:
          options:
            case: snake_case
```

Unrecognized rule keys are ignored rather than rejected, so a config left on the old IDs
will load without error and silently stop applying your overrides. Grep your configs for
the old names rather than relying on `terratidy config validate` to catch them.

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
