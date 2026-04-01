# SDK Reference

Public API for rule authors and plugin developers.

Package: `github.com/santosr2/terratidy/pkg/sdk`

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
    // Config holds rule-specific configuration from .terratidy.yaml.
    // Keys and values correspond to the "options" map under a rule's config.
    Config map[string]any

    // WorkDir is the working directory TerraTidy was invoked from.
    WorkDir string

    // File is the absolute path to the file being checked.
    File string
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

    // Location is the line/column range from the HCL parser.
    Location hcl.Range `json:"location"`

    // Severity indicates the importance: error, warning, or info.
    Severity Severity `json:"severity"`

    // Fixable indicates whether this finding can be auto-fixed.
    Fixable bool `json:"fixable"`

    // FixFunc is the function that applies the fix. Not serialized to JSON.
    FixFunc func() ([]byte, error) `json:"-"`
}
```

## Severity

```go
type Severity string

const (
    SeverityError   Severity = "error"   // Must be fixed; causes non-zero exit
    SeverityWarning Severity = "warning" // Should be fixed; reported but doesn't fail by default
    SeverityInfo    Severity = "info"    // Suggestion; informational only
)
```

## Usage Example

```go
package main

import (
    "github.com/hashicorp/hcl/v2"
    "github.com/santosr2/terratidy/pkg/sdk"
)

type MyRule struct{}

func (r *MyRule) Name() string        { return "my-org.my-rule" }
func (r *MyRule) Description() string { return "Checks something important" }

func (r *MyRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    // Access rule config
    if val, ok := ctx.Config["my_option"]; ok {
        _ = val // use it
    }

    // Return findings
    return []sdk.Finding{{
        Rule:     r.Name(),
        Message:  "Something needs attention",
        File:     ctx.File,
        Severity: sdk.SeverityWarning,
    }}, nil
}

// Optional: implement sdk.Fixer for auto-fix support
// func (r *MyRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
//     return fixedContent, nil
// }
```
