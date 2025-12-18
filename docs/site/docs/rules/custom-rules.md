# Custom Rules

Create your own rules using the plugin system or OPA policies.

## Plugin-based Rules

Create custom rules in Go using the plugin system.

### Rule Interface

```go
package myrule

import (
    "github.com/santosr2/terratidy/pkg/sdk"
)

type MyRule struct{}

func (r *MyRule) Name() string {
    return "my-custom-rule"
}

func (r *MyRule) Description() string {
    return "Enforces my custom requirement"
}

func (r *MyRule) Severity() sdk.Severity {
    return sdk.SeverityWarning
}

func (r *MyRule) Check(ctx *sdk.Context) []sdk.Finding {
    var findings []sdk.Finding

    for _, resource := range ctx.Resources() {
        if resource.Type == "aws_instance" {
            if !hasRequiredTag(resource, "Environment") {
                findings = append(findings, sdk.Finding{
                    Rule:     r.Name(),
                    Message:  "EC2 instance must have Environment tag",
                    File:     resource.File,
                    Location: resource.Range,
                    Severity: r.Severity(),
                    Fixable:  false,
                })
            }
        }
    }

    return findings
}

func hasRequiredTag(r *sdk.Resource, tag string) bool {
    if tags, ok := r.Attributes["tags"].(map[string]any); ok {
        _, exists := tags[tag]
        return exists
    }
    return false
}
```

### Building Plugins

```bash
# Build as a Go plugin
go build -buildmode=plugin -o myrule.so myrule.go
```

### Installing Plugins

Place plugins in one of these directories:

- `.terratidy/rules/` (project-local)
- `~/.terratidy/rules/` (user-global)

### Configuration

```yaml
plugins:
  enabled: true
  directories:
    - .terratidy/rules
    - ~/.terratidy/rules
```

## OPA/Rego Policies

For simpler custom rules, use OPA policies.

### Basic Policy

TerraTidy uses OPA v1, which requires the `import rego.v1` statement and updated
rule syntax with `contains` and `if` keywords.

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

| Severity | Use For |
|----------|---------|
| Error | Security issues, broken code |
| Warning | Best practice violations |
| Info | Style suggestions |

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
