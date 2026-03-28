# Custom Rules

Create your own rules using Go plugins, YAML definitions, Bash scripts, or OPA policies.

## Rule Interface

All custom rules implement the `sdk.Rule` interface:

```go
type Rule interface {
    Name() string
    Description() string
    Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error)
    Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error)
}
```

The `sdk.Context` provides runtime information:

```go
type Context struct {
    Config  map[string]interface{}
    Logger  *log.Logger
    WorkDir string
    File    string
}
```

## Go Plugin Rules

Create custom rules in Go using the plugin system. Requires Go 1.26.1 or later.

### Complete Example

```go
package main

import (
    "fmt"

    "github.com/hashicorp/hcl/v2"
    "github.com/hashicorp/hcl/v2/hclsyntax"
    "github.com/santosr2/terratidy/internal/plugins"
    "github.com/santosr2/terratidy/pkg/sdk"
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
                Location: block.DefRange(),
                Severity: sdk.SeverityWarning,
            })
        }
    }
    return findings, nil
}

func (r *RequireTagsRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
    return nil, nil
}
```

### Building and Installing

```bash
# Build as a Go plugin
go build -buildmode=plugin -o require-tags.so

# Install to project plugin directory
cp require-tags.so .terratidy/plugins/
```

## YAML Rules

For simple pattern-based checks, define rules declaratively in YAML.
No Go code required.

### Example

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

### How It Works

YAML rules check `resource` blocks for required attributes. If
`resource_types` is specified, only those types are checked. If omitted,
all resource blocks are checked. Each missing `required_attributes` entry
generates a finding.

### Installation

Place `.yaml` or `.yml` files in a plugin directory:

```bash
cp require-description.yaml .terratidy/plugins/
```

## Bash Rules

Shell scripts that analyze Terraform files and output JSON findings.

### Contract

- Receives the file path as `$1`
- Outputs JSON to stdout with a `findings` array
- Exit code 0 = success, exit code 1 with output = findings reported
- 30-second timeout
- Must be executable (`chmod +x`)

### Example

```bash
#!/usr/bin/env bash
set -euo pipefail

FILE="$1"

# Match 12-digit AWS account IDs
PATTERN='[^a-zA-Z_$][0-9]{12}[^0-9]'

findings="[]"

while IFS= read -r match; do
  line=$(echo "$match" | cut -d: -f1)
  findings=$(echo "$findings" | jq --arg file "$FILE" --arg line "$line" \
    '. + [{"file": $file, "line": ($line | tonumber), "message": "Hardcoded AWS account ID detected; use a variable or data source", "severity": "warning"}]')
done < <(grep -nE "$PATTERN" "$FILE" 2>/dev/null || true)

echo "{\"findings\": $findings}"
```

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

If `rule` is omitted from a finding, the script filename (without extension)
is used.

### Installation

```bash
cp no-hardcoded-account-id.sh .terratidy/plugins/
chmod +x .terratidy/plugins/no-hardcoded-account-id.sh
```

**Note:** Bash rules are not supported on Windows.

## Plugin Directory and Configuration

Place plugins in one of these directories:

- `.terratidy/plugins/` (project-local)
- `~/.terratidy/plugins/` (user-global)

```yaml
plugins:
  enabled: true
  directories:
    - .terratidy/plugins
    - ~/.terratidy/plugins
```

## OPA/Rego Policies

For policy-as-code rules, use OPA policies.

### Basic Policy

TerraTidy uses OPA v1, which requires the `import rego.v1` statement and
updated rule syntax with `contains` and `if` keywords.

```rego
package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not has_environment_tag(resource)
    msg := {
        "msg": sprintf("EC2 instance %s must have Environment tag", [resource.name]),
        "rule": "require-environment-tag",
        "severity": "error",
        "file": resource._file,
        "line": resource._range.start_line
    }
}

has_environment_tag(resource) if {
    tags := resource.tags
    contains(tags, "Environment")
}
```

### Policy with Functions

```rego
package terraform

import rego.v1

# Helper function to check for required tags
missing_required_tags(resource, required) := missing if {
    provided := {tag | some tag, _ in resource.tags}
    missing := required - provided
}

# Check all taggable resources
deny contains msg if {
    required_tags := {"Environment", "Team", "CostCenter"}
    some resource in input.resources
    resource.type in taggable_types
    missing := missing_required_tags(resource, required_tags)
    count(missing) > 0
    msg := {
        "msg": sprintf("%s %s is missing required tags: %v",
            [resource.type, resource.name, missing]),
        "rule": "required-tags",
        "severity": "warning",
        "file": resource._file
    }
}

# Types that should have tags
taggable_types := {
    "aws_instance",
    "aws_s3_bucket",
    "aws_rds_cluster",
    "aws_eks_cluster"
}
```

### Configurable Policy

```rego
package terraform

import rego.v1

# Read from external data
import data.config

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not valid_instance_type(resource.instance_type)
    msg := {
        "msg": sprintf("Instance type %s is not in approved list",
            [resource.instance_type]),
        "rule": "approved-instance-types",
        "severity": "error"
    }
}

valid_instance_type(t) if {
    some approved in config.approved_instance_types
    approved == t
}
```

With data file `policies/data.json`:

```json
{
  "config": {
    "approved_instance_types": [
      "t3.micro",
      "t3.small",
      "t3.medium"
    ]
  }
}
```

## Best Practices

### Rule Naming

- Use kebab-case: `require-environment-tag`
- Be descriptive: `no-public-s3-bucket`
- Prefix with category: `security-no-public-ssh`

### Severity Guidelines

| Severity | Use For                        |
| -------- | ------------------------------ |
| Error    | Security issues, broken code   |
| Warning  | Best practice violations       |
| Info     | Style suggestions              |

### Documentation

Document your rules:

```rego
# Rule: require-encryption
# Description: All EBS volumes must be encrypted
# Severity: Error
# Rationale: Encryption at rest is required for compliance
```

### Testing Policies

Test with the OPA CLI:

```bash
# Test policy
opa test policies/ -v

# Evaluate against sample input
opa eval --input test-input.json \
         --data policies/ \
         "data.terraform.deny"
```

### Sample Test File

```rego
package terraform_test

import rego.v1
import data.terraform

test_require_encryption_pass if {
    result := terraform.deny with input as {
        "resources": [{
            "type": "aws_ebs_volume",
            "name": "encrypted_volume",
            "encrypted": "true"
        }]
    }
    count(result) == 0
}

test_require_encryption_fail if {
    result := terraform.deny with input as {
        "resources": [{
            "type": "aws_ebs_volume",
            "name": "unencrypted_volume",
            "encrypted": "false"
        }]
    }
    count(result) == 1
}
```
