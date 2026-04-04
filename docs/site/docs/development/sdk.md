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
    // Fix applies an automatic fix and returns the corrected file content.
    Fix(ctx *Context, file *hcl.File) ([]byte, error)
}
```

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

    // Fix holds the pre-computed fix result. If non-nil, the finding is auto-fixable.
    Fix *FixResult `json:"fix,omitempty"`
}
```

## FixResult

Holds the result of an auto-fix operation:

```go
type FixResult struct {
    // Content is the corrected file content after applying the fix.
    Content []byte
}
```

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
// func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
//     return fixedContent, nil
// }
```
