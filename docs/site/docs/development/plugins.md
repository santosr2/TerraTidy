# Plugin Development

Guide to developing custom plugins for TerraTidy.

## Overview

TerraTidy supports three types of custom plugins:

- **Go plugins** (`.so` files) - compiled Go plugins using the plugin system
- **YAML rules** (`.yaml`/`.yml` files) - declarative pattern-based rules
- **Bash rules** (`.sh` files) - shell scripts that output JSON findings

All plugin types are loaded automatically from configured plugin directories.

## Plugin Directories

Place plugins in one of these directories:

- `.terratidy/plugins/` (project-local)
- `~/.terratidy/plugins/` (user-global)

Configure directories in `.terratidy.yaml`:

```yaml
plugins:
  enabled: true
  directories:
    - .terratidy/plugins
    - ~/.terratidy/plugins
```

## SDK Reference

### Rule Interface

Every rule (Go, Rego, YAML, or Bash) implements the `sdk.Rule` interface:

```go
type Rule interface {
    // Name returns the rule identifier
    Name() string

    // Description returns a human-readable description
    Description() string

    // Check evaluates the rule against a parsed HCL file
    Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error)
}
```

Rules that support auto-fixing also implement `sdk.Fixer`:

```go
type Fixer interface {
    // Fix applies an automatic fix and returns the corrected file bytes
    Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error)
}
```

### Context

The `sdk.Context` provides runtime information to rules:

```go
type Context struct {
    context.Context              // Embedded for cancellation/deadline support

    Options  map[string]any      // Rule-specific config from .terratidy.yaml
    WorkDir  string              // Directory TerraTidy was invoked from
    File     string              // Absolute path to file being checked
    AllFiles map[string][]byte   // All files being processed (for cross-file rules)
}
```

### Finding

```go
type Finding struct {
    Rule     string      `json:"rule"`
    Message  string      `json:"message"`
    File     string      `json:"file"`
    Location Location    `json:"location"`
    Severity Severity    `json:"severity"`
    Fix      *FixResult  `json:"fix,omitempty"`
}
```

### FixResult

Holds the result of an auto-fix operation:

```go
type FixResult struct {
    Content []byte
}
```

A finding is auto-fixable when `Fix != nil`.

### Severity

```go
const (
    SeverityError   Severity = "error"
    SeverityWarning Severity = "warning"
    SeverityInfo    Severity = "info"
)
```

## Go Plugins

### Prerequisites

- Go 1.26.1 or later
- TerraTidy SDK (`pkg/sdk`)
- TerraTidy plugin types (`internal/plugins`)

### Plugin Structure

```text
my-plugin/
├── go.mod
├── go.sum
└── main.go
```

### Plugin Exports

Go plugins must export two symbols:

1. **`PluginMetadata`** - a `*plugins.PluginMetadata` variable with plugin info
2. **`New`** - a constructor function returning the plugin instance

```go
// PluginMetadata contains information about a plugin
type PluginMetadata struct {
    Name        string
    Version     string
    Description string
    Author      string
    Type        PluginType   // "rule", "engine", or "formatter"
    Path        string       // set automatically on load
}
```

### Plugin Interfaces

Depending on the plugin type, `New()` must return one of:

| Type        | Return Type              | Interface                                                |
| ----------- | ------------------------ | -------------------------------------------------------- |
| `rule`      | `plugins.RulePlugin`     | `GetRules() []sdk.Rule`                                  |
| `engine`    | `plugins.EnginePlugin`   | `Name() string`, `Run(ctx, files) ([]sdk.Finding, error)` |
| `formatter` | `plugins.FormatterPlugin`| `Name() string`, `Format(findings []sdk.Finding, w io.Writer) error` |

### Complete Go Plugin Example

This example checks that all resource blocks have a `tags` attribute:

```go
package main

import (
    "fmt"

    "github.com/hashicorp/hcl/v2"
    "github.com/hashicorp/hcl/v2/hclsyntax"
    "github.com/santosr2/TerraTidy/internal/plugins"
    "github.com/santosr2/TerraTidy/pkg/sdk"
)

// PluginMetadata provides information about this plugin.
var PluginMetadata = &plugins.PluginMetadata{
    Name:        "require-tags",
    Version:     "1.0.0",
    Description: "Checks that resources have a tags attribute",
    Author:      "Your Name",
    Type:        plugins.PluginTypeRule,
}

// Plugin implements the RulePlugin interface.
type Plugin struct {
    rules []sdk.Rule
}

// New creates a new instance of the plugin.
func New() plugins.RulePlugin {
    return &Plugin{rules: []sdk.Rule{&RequireTagsRule{}}}
}

// GetRules returns all rules provided by this plugin.
func (p *Plugin) GetRules() []sdk.Rule { return p.rules }

// RequireTagsRule checks that resource blocks include a tags attribute.
type RequireTagsRule struct{}

func (r *RequireTagsRule) Name() string        { return "require-tags" }
func (r *RequireTagsRule) Description() string { return "Resources must have a tags attribute" }

func (r *RequireTagsRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
    body, ok := file.Body.(*hclsyntax.Body)
    if !ok {
        return nil, nil
    }

    var findings []sdk.Finding
    for _, block := range body.Blocks {
        if block.Type != "resource" {
            continue
        }
        hasTags := false
        for _, attr := range block.Body.Attributes {
            if attr.Name == "tags" {
                hasTags = true
                break
            }
        }
        if !hasTags {
            findings = append(findings, sdk.Finding{
                Rule:     "require-tags",
                Message:  fmt.Sprintf("Resource %q is missing a tags attribute", block.Labels[0]),
                File:     ctx.File,
                Location: sdk.LocationFromRange(block.DefRange()),
                Severity: sdk.SeverityWarning,
            })
        }
    }
    return findings, nil
}

// Optional: implement sdk.Fixer for auto-fix support
// func (r *RequireTagsRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
//     return fixedContent, nil
// }
```

### Building Go Plugins

```bash
go build -buildmode=plugin -o require-tags.so
```

Then copy the `.so` file to a plugin directory:

```bash
cp require-tags.so .terratidy/plugins/
```

## YAML Rules

YAML rules provide a declarative way to define pattern-based checks without writing Go code.

### YAML Rule Structure

```yaml
name: require-description
description: All resources must have a description attribute
severity: warning
enabled: true
message: "Resource is missing a 'description' attribute"
tags:
  - documentation
  - best-practice
patterns:
  resource_types:
    - aws_instance
    - aws_s3_bucket
  required_attributes:
    - description
```

### Fields

| Field                          | Type   | Required | Description                                         |
| ------------------------------ | ------ | -------- | --------------------------------------------------- |
| `name`                         | string | Yes      | Rule identifier                                     |
| `description`                  | string | No       | Human-readable description                          |
| `severity`                     | string | No       | `error`, `warning`, or `info` (default: `warning`)  |
| `enabled`                      | bool   | No       | Whether the rule is active (default: `false`)       |
| `message`                      | string | No       | Custom message for findings                         |
| `tags`                         | list   | No       | Tags for categorization                             |
| `patterns.resource_types`      | list   | No       | Resource types to check (empty = all resources)     |
| `patterns.required_attributes` | list   | No       | Attributes that must be present                     |

### How YAML Rules Work

YAML rules iterate over HCL blocks in the file. For each `resource` block
that matches the configured `resource_types` (or all resource blocks if none
are specified), the rule checks that all `required_attributes` are present.
Missing attributes generate a finding.

### Installation

Place `.yaml` or `.yml` files in a plugin directory:

```bash
cp require-description.yaml .terratidy/plugins/
```

## Bash Rules

Bash rules execute a shell script that analyzes Terraform files and outputs findings as JSON.

### Bash Rule Contract

- The script receives the file path as the first argument (`$1`)
- It must output JSON to stdout in this format:
- Exit code 0 means success (with or without findings)
- Exit code 1 with JSON output is treated as findings (not a script error)
- The script has a 30-second timeout
- The script must be executable (`chmod +x`)

### Output Format

```json
{
  "findings": [
    {
      "file": "main.tf",
      "line": 5,
      "column": 1,
      "message": "Issue description",
      "severity": "warning",
      "rule": "optional-rule-name"
    }
  ]
}
```

If `rule` is omitted, the script filename (without extension) is used as the rule name.

### Example Bash Rule

```bash
#!/usr/bin/env bash
set -euo pipefail

FILE="$1"

# Match 12-digit AWS account IDs (standalone, not inside a variable reference)
PATTERN='[^a-zA-Z_$][0-9]{12}[^0-9]'

findings="[]"

while IFS= read -r match; do
  line=$(echo "$match" | cut -d: -f1)
  findings=$(echo "$findings" | jq --arg file "$FILE" --arg line "$line" \
    '. + [{"file": $file, "line": ($line | tonumber), "message": "Hardcoded AWS account ID detected; use a variable or data source", "severity": "warning"}]')
done < <(grep -nE "$PATTERN" "$FILE" 2>/dev/null || true)

echo "{\"findings\": $findings}"
```

### Installation

Place executable `.sh` files in a plugin directory:

```bash
cp no-hardcoded-account-id.sh .terratidy/plugins/
chmod +x .terratidy/plugins/no-hardcoded-account-id.sh
```

**Note:** Bash rules are not supported on Windows.

## Scaffolding Rules

Use `terratidy init-rule` to scaffold new rules:

```bash
terratidy init-rule --name my-rule --type go     # Go plugin with test file
terratidy init-rule --name my-rule --type rego    # OPA/Rego policy
terratidy init-rule --name my-rule --type yaml    # YAML rule config
```

**Note:** Bash rules cannot be scaffolded via `init-rule`. Create them manually since
the format is simple (executable script outputting JSON).

## Plugin Management

```bash
# List all loaded plugins
terratidy plugins list

# Show details for a specific plugin
terratidy plugins info require-tags

# Create a new Go plugin project
terratidy plugins init my-plugin
```

## Distribution

Plugins are distributed as files. There is no built-in registry or package manager.

**For teams/organizations:**

1. Store plugins in a shared Git repository
2. Copy plugin files to `~/.terratidy/plugins/` or a project-local directory
3. Use your CI/CD pipeline to install plugins before running TerraTidy

```bash
# Example: install plugins from a shared repo
git clone git@github.com:my-org/terratidy-plugins.git /tmp/plugins
cp /tmp/plugins/*.yaml .terratidy/plugins/
cp /tmp/plugins/*.sh .terratidy/plugins/
chmod +x .terratidy/plugins/*.sh
```

Plugin metadata includes a `Version` field for tracking, but there is no automatic
version resolution or update mechanism.

## Testing Plugins

### Testing Go Plugins

```go
func TestRequireTagsRule(t *testing.T) {
    rule := &RequireTagsRule{}

    src := []byte(`
resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`)
    file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
    require.False(t, diags.HasErrors())

    ctx := &sdk.Context{File: "test.tf"}
    findings, err := rule.Check(ctx, file)
    require.NoError(t, err)
    assert.Len(t, findings, 1)
    assert.Equal(t, "require-tags", findings[0].Rule)
}
```

### Testing Bash Rules

Run the script directly and check the JSON output:

```bash
# Should produce findings
./no-hardcoded-account-id.sh test-fixtures/hardcoded.tf | jq '.findings | length'

# Should produce no findings
./no-hardcoded-account-id.sh test-fixtures/clean.tf | jq '.findings | length'
```

## Best Practices

1. **Clear naming**: Use kebab-case descriptive rule names (`require-tags`, `no-public-s3-bucket`)
2. **Good messages**: Provide actionable error messages that tell the user what to fix
3. **Appropriate severity**: Match severity to impact (error for security, warning for best practices, info for suggestions)
4. **Provide fixes**: Implement the `Fix` method when an automatic correction is possible
5. **Test thoroughly**: Write unit tests for Go plugins, script tests for Bash rules
6. **Use YAML for simple checks**: If you only need to verify attribute presence, a YAML rule is simpler than Go
