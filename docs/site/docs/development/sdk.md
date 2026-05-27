# SDK Reference

Public API for rule authors and plugin developers.

Package: `github.com/santosr2/TerraTidy/pkg/sdk`

## Stability

The `pkg/sdk` package is the public API. Types and interfaces in this package follow
semantic versioning. Breaking changes only occur in major version bumps.

Internal packages (`internal/`) are private and may change without notice.

## Rule Interface

All rules (built-in and custom) implement this interface:

```go
type Rule interface {
    // Name returns a unique identifier for the rule (e.g., "style.block-label-case").
    Name() string

    // Description returns a human-readable description of what the rule checks.
    Description() string

    // Check evaluates the rule against a parsed HCL file and returns any findings.
    // Return nil findings and nil error if the file passes the check.
    Check(ctx *Context, file *hcl.File) ([]Finding, error)
}
```

Rules that support auto-fixing also implement `Fixer`:

```go
type Fixer interface {
    // Fix returns the set of byte-range edits the engine should apply to the
    // file's current content. Return nil, nil (or a *FixResult with no edits)
    // when no fix is applicable; the engine treats either form as a no-op.
    //
    // Multiple findings against the same file each call Fix independently; the
    // engine collects every returned edit and applies them in a single pass.
    // See FixResult for the exact ordering, overlap, and whole-file rules.
    Fix(ctx *Context, file *hcl.File) (*FixResult, error)
}
```

`Fix` no longer returns the rewritten file in full. Rules describe the change
as one or more `TextEdit`s wrapped in a `*FixResult`; the engine collects edits
across all fixable findings and applies them in a single pass. See the
[TextEdit](#textedit) and [FixResult](#fixresult) sections below for the byte-offset
semantics and the whole-file exclusivity rule.

## Engine Interface

All analysis engines (fmt, style, lint, policy) implement:

```go
type Engine interface {
    Name() string
    Run(ctx context.Context, files []string) ([]Finding, error)
}
```

## Context

Runtime context passed to every rule invocation:

```go
type Context struct {
    context.Context  // Embedded for cancellation and deadline support

    // Options holds rule-specific options from the "options" map in .terratidy.yaml.
    Options map[string]any

    // WorkDir is the absolute path to the directory TerraTidy was invoked from.
    WorkDir string

    // File is the absolute path to the file being checked.
    File string

    // AllFiles contains the raw content of all files being processed in this run.
    // Useful for cross-file rules that need to analyze multiple files together.
    AllFiles map[string][]byte
}
```

## Finding

A single issue detected by a rule:

```go
type Finding struct {
    // Rule is the identifier of the rule that produced this finding.
    Rule string `json:"rule"`

    // Message is a human-readable description of the issue.
    Message string `json:"message"`

    // File is the path to the file where the issue was found.
    File string `json:"file"`

    // Location is the source code range where the issue was found.
    Location Location `json:"location"`

    // Severity indicates the importance: error, warning, or info.
    Severity Severity `json:"severity"`

    // Fixable is true when the rule that produced this finding implements Fixer.
    // The engine sets this; rules must not set it directly.
    Fixable bool `json:"fixable,omitempty"`

    // IsDiff is true when Message holds a unified diff rather than a
    // human-readable description. Consumers (CLI, LSP, formatters) route
    // diff-message findings through a diff renderer instead of plain text.
    // Set by the fmt and style engines in diff mode; rules must not set it.
    IsDiff bool `json:"is_diff,omitempty"`
}
```

A finding is auto-fixable when `Fixable` is `true`. The engine sets this field by
type-asserting the rule against `sdk.Fixer` and stamping it on every finding the
rule produces. To collect the `TextEdit`s, the engine calls
`Fixer.Fix(ctx, file)` lazily — only in fix or diff mode.

A finding with `IsDiff` set to `true` carries a unified diff in `Message` instead
of a description. The CLI gates diff rendering on this field; the JSON output
format exposes it as `"is_diff": true`.

Plugin authors migrating from the pre-`Fixable` SDK (where `Finding.Fix` was a
`*FixResult` pointer carrying precomputed bytes and a diff string) should see
the [upgrade guide migration note](../getting-started/upgrade.md#v020-alpha5-sdk-findingfix-replaced-with-fixable-flag)
for a before/after walk-through. The `FixResult` name has since been
reintroduced for an unrelated purpose; see [FixResult](#fixresult) below.

## TextEdit

A single byte-range edit. Rules return one or more `TextEdit`s inside a
[`FixResult`](#fixresult); the engine sorts edits by `Start` in descending order
and applies them in a single write per pass.

```go
type TextEdit struct {
    // Start is the inclusive byte offset where the edit begins.
    Start int `json:"start"`
    // End is the exclusive byte offset where the edit ends. End must be >= Start
    // and <= len(content); the engine bounds-checks every edit and errors otherwise.
    End int `json:"end"`
    // Replacement is the bytes to insert in place of content[Start:End].
    // An empty or nil slice means deletion.
    Replacement []byte `json:"replacement"`
}
```

Offsets are **half-open**: `[Start, End)`.

| Shape          | Start vs End                                | Replacement       |
| -------------- | ------------------------------------------- | ----------------- |
| Pure insertion | `Start == End`                              | non-empty         |
| Pure deletion  | `Start < End`                               | empty or nil      |
| Replacement    | `Start < End`                               | non-empty         |
| Whole-file     | `Start == 0 && End == len(content)`         | the entire rewrite |

Whole-file edits trigger exclusive-this-pass behavior; see
[FixResult](#fixresult).

## FixResult

The return value of `Fixer.Fix`. Wraps the edits a single `Fix` call produced.

```go
type FixResult struct {
    // Edits is the set of byte-range edits to apply. An empty or nil slice
    // indicates no fix; equivalent to returning a nil *FixResult.
    Edits []TextEdit `json:"edits"`
}
```

The struct shape (rather than a bare `[]TextEdit`) is a forward-compat hatch:
future fields can be added without changing the `Fixer` signature again.

### Apply order

The engine sorts edits by `Start` in **descending** order before applying, so
earlier (lower-offset) splices do not invalidate the byte offsets of later
edits in the same pass. Rule authors may return edits in any order.

### Whole-file exclusivity

If any edit collected in a pass has `Start == 0 && End == len(content)`, it is
applied alone — all other edits in the same pass are discarded and re-emit
against the rewritten content on the next pass. This avoids ambiguous
interactions between whole-file rewriters and narrow edits.

Rules migrating from the old `[]byte`-returning `Fix` can build a whole-file
`FixResult` directly:

```go
&sdk.FixResult{Edits: []sdk.TextEdit{{
    Start:       0,
    End:         len(original),
    Replacement: newContent,
}}}
```

Return `nil, nil` when `newContent` equals `original` to signal a no-op fix.

!!! note "Name reuse"
    The `FixResult` name was previously used (pre-v0.2.0-alpha.5) for a
    different type carrying precomputed bytes and a diff string. That type was
    removed alongside the `Finding.Fix` field. The current `FixResult` is
    unrelated to the old shape. See the
    [upgrade guide](../getting-started/upgrade.md#v020-alpha5-sdk-fixerfix-returns-fixresult-instead-of-byte)
    for the chronology.

## Location

Represents a source code location. This type replaces direct use of `hcl.Range` in the public API.

```go
type Location struct {
    Filename    string `json:"filename"`     // Path to the source file
    StartLine   int    `json:"start_line"`   // 1-based line number where the issue starts
    StartColumn int    `json:"start_column"` // 1-based column number where the issue starts
    EndLine     int    `json:"end_line"`     // 1-based line number where the issue ends
    EndColumn   int    `json:"end_column"`   // 1-based column number where the issue ends
}
```

Use `LocationFromRange(hcl.Range)` to convert from HCL ranges when implementing rules.

## Severity

```go
type Severity string

const (
    SeverityError   Severity = "error"   // Must be fixed; causes non-zero exit
    SeverityWarning Severity = "warning" // Should be fixed; reported but doesn't fail by default
    SeverityInfo    Severity = "info"    // Suggestion; informational only
)
```

### Helper Functions

Parse severity strings with a default fallback:

```go
sev := sdk.ParseSeverity("warning", sdk.SeverityInfo)  // Returns SeverityWarning
sev := sdk.ParseSeverity("unknown", sdk.SeverityInfo)  // Returns SeverityInfo (default)
```

Get numeric level for filtering or comparison:

```go
// Levels: error=2, warning=1, info=0
if finding.Severity.Level() >= sdk.SeverityWarning.Level() {
    // Handle warnings and errors
}
```

## Usage Example

```go
package main

import (
    "github.com/hashicorp/hcl/v2"
    "github.com/santosr2/TerraTidy/pkg/sdk"
)

type MyRule struct{}

func (r *MyRule) Name() string        { return "my-org.my-rule" }
func (r *MyRule) Description() string { return "Checks something important" }

func (r *MyRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    // Access rule options
    if val, ok := ctx.Options["my_option"]; ok {
        _ = val // use it
    }

    // Return findings with location
    return []sdk.Finding{{
        Rule:     r.Name(),
        Message:  "Something needs attention",
        File:     ctx.File,
        Location: sdk.Location{Filename: ctx.File, StartLine: 1, StartColumn: 1},
        Severity: sdk.SeverityWarning,
    }}, nil
}

// Optional: implement sdk.Fixer for auto-fix support
// func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) (*sdk.FixResult, error) {
//     return &sdk.FixResult{Edits: []sdk.TextEdit{{
//         Start:       findingStart,
//         End:         findingEnd,
//         Replacement: []byte("corrected"),
//     }}}, nil
// }
```
