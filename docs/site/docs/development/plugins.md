# Plugin Development

Guide to developing custom plugins for TerraTidy.

## Overview

TerraTidy supports custom plugins for:

- Custom style rules
- Custom lint rules
- Custom output formatters
- Engine extensions

## Getting Started

### Prerequisites

- Go 1.21 or later
- TerraTidy SDK (`pkg/sdk`)

### Plugin Structure

```text
my-plugin/
├── go.mod
├── go.sum
├── main.go
└── rules/
    ├── rule1.go
    └── rule2.go
```

### Basic Plugin

```go
// main.go
package main

import (
    "github.com/santosr2/terratidy/pkg/sdk"
)

// Plugin exports
var (
    Name    = "my-plugin"
    Version = "1.0.0"
    Rules   = []sdk.Rule{
        &MyCustomRule{},
    }
)

// MyCustomRule implements a custom rule
type MyCustomRule struct{}

func (r *MyCustomRule) Name() string {
    return "my-custom-rule"
}

func (r *MyCustomRule) Description() string {
    return "Enforces my custom requirement"
}

func (r *MyCustomRule) Severity() sdk.Severity {
    return sdk.SeverityWarning
}

func (r *MyCustomRule) Check(ctx *sdk.RuleContext) []sdk.Finding {
    var findings []sdk.Finding

    for _, resource := range ctx.Resources() {
        // Implement your check logic
        if !isValid(resource) {
            findings = append(findings, sdk.Finding{
                Rule:     r.Name(),
                Message:  "Resource violates custom rule",
                File:     resource.File,
                Location: resource.Range,
                Severity: r.Severity(),
            })
        }
    }

    return findings
}
```

## SDK Reference

### Rule Interface

```go
type Rule interface {
    // Name returns the rule identifier
    Name() string

    // Description returns a human-readable description
    Description() string

    // Severity returns the default severity
    Severity() Severity

    // Check executes the rule and returns findings
    Check(ctx *RuleContext) []Finding
}
```

### RuleContext

```go
type RuleContext struct {
    // File information
    File     string
    Content  []byte

    // Parsed structures
    Resources  []Resource
    DataBlocks []DataBlock
    Variables  []Variable
    Outputs    []Output
    Locals     []Local
    Modules    []Module

    // Configuration
    Config map[string]any
}
```

### Resource Type

```go
type Resource struct {
    Type       string
    Name       string
    File       string
    Range      hcl.Range
    Attributes map[string]any
    Blocks     []Block
}
```

### Finding Type

```go
type Finding struct {
    Rule     string
    Message  string
    File     string
    Location hcl.Range
    Severity Severity
    Fixable  bool
    Fix      *Fix
}
```

### Fix Type

```go
type Fix struct {
    Description string
    Edits       []Edit
}

type Edit struct {
    Range   hcl.Range
    NewText string
}
```

## Building Plugins

### As Go Plugin

```bash
go build -buildmode=plugin -o my-plugin.so
```

### As Standalone Binary

```bash
go build -o my-plugin
```

## Installing Plugins

### Local Installation

```bash
# Project-local
cp my-plugin.so .terratidy/rules/

# User-global
cp my-plugin.so ~/.terratidy/rules/
```

### Configuration

```yaml
plugins:
  enabled: true
  directories:
    - .terratidy/rules
    - ~/.terratidy/rules
```

## Examples

### Tag Validation Rule

```go
type RequiredTagsRule struct {
    requiredTags []string
}

func NewRequiredTagsRule(tags []string) *RequiredTagsRule {
    return &RequiredTagsRule{requiredTags: tags}
}

func (r *RequiredTagsRule) Name() string {
    return "required-tags"
}

func (r *RequiredTagsRule) Check(ctx *sdk.RuleContext) []sdk.Finding {
    var findings []sdk.Finding

    for _, resource := range ctx.Resources() {
        if !isTaggable(resource.Type) {
            continue
        }

        tags := extractTags(resource)
        for _, required := range r.requiredTags {
            if _, ok := tags[required]; !ok {
                findings = append(findings, sdk.Finding{
                    Rule:     r.Name(),
                    Message:  fmt.Sprintf("Missing required tag: %s", required),
                    File:     resource.File,
                    Location: resource.Range,
                    Severity: sdk.SeverityWarning,
                })
            }
        }
    }

    return findings
}
```

### Naming Convention Rule

```go
type NamingConventionRule struct {
    pattern *regexp.Regexp
}

func NewNamingConventionRule(pattern string) *NamingConventionRule {
    return &NamingConventionRule{
        pattern: regexp.MustCompile(pattern),
    }
}

func (r *NamingConventionRule) Check(ctx *sdk.RuleContext) []sdk.Finding {
    var findings []sdk.Finding

    for _, resource := range ctx.Resources() {
        if !r.pattern.MatchString(resource.Name) {
            findings = append(findings, sdk.Finding{
                Rule:     r.Name(),
                Message:  fmt.Sprintf("Name '%s' doesn't match pattern", resource.Name),
                File:     resource.File,
                Location: resource.Range,
                Severity: sdk.SeverityWarning,
                Fixable:  true,
                Fix: &sdk.Fix{
                    Description: "Rename to match pattern",
                    Edits: []sdk.Edit{{
                        Range:   resource.NameRange,
                        NewText: toSnakeCase(resource.Name),
                    }},
                },
            })
        }
    }

    return findings
}
```

### Security Scanning Rule

```go
type NoHardcodedSecretsRule struct{}

func (r *NoHardcodedSecretsRule) Name() string {
    return "no-hardcoded-secrets"
}

func (r *NoHardcodedSecretsRule) Check(ctx *sdk.RuleContext) []sdk.Finding {
    var findings []sdk.Finding

    secretPatterns := []string{
        `(?i)password\s*=\s*"[^"]+`,
        `(?i)secret\s*=\s*"[^"]+`,
        `(?i)api_key\s*=\s*"[^"]+`,
        `AKIA[0-9A-Z]{16}`, // AWS Access Key
    }

    for _, pattern := range secretPatterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindAllIndex(ctx.Content, -1)

        for _, match := range matches {
            findings = append(findings, sdk.Finding{
                Rule:     r.Name(),
                Message:  "Potential hardcoded secret detected",
                File:     ctx.File,
                Severity: sdk.SeverityError,
            })
        }
    }

    return findings
}
```

## Testing Plugins

### Unit Tests

```go
func TestRequiredTagsRule(t *testing.T) {
    rule := NewRequiredTagsRule([]string{"Environment", "Team"})

    ctx := &sdk.RuleContext{
        Resources: []sdk.Resource{{
            Type: "aws_instance",
            Name: "example",
            Attributes: map[string]any{
                "tags": map[string]string{
                    "Environment": "prod",
                    // Missing "Team" tag
                },
            },
        }},
    }

    findings := rule.Check(ctx)

    if len(findings) != 1 {
        t.Errorf("Expected 1 finding, got %d", len(findings))
    }
}
```

### Integration Tests

```go
func TestPluginIntegration(t *testing.T) {
    // Create temp Terraform file
    content := `
resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
    tmpfile := createTempFile(t, content)

    // Run plugin
    ctx := sdk.NewContext(tmpfile)
    rule := &MyCustomRule{}
    findings := rule.Check(ctx)

    // Assert
    assert.Len(t, findings, 1)
}
```

## Best Practices

1. **Clear naming**: Use descriptive rule names
2. **Good messages**: Provide actionable error messages
3. **Appropriate severity**: Match severity to impact
4. **Provide fixes**: Implement auto-fix when possible
5. **Test thoroughly**: Write unit and integration tests
6. **Document**: Include usage documentation
