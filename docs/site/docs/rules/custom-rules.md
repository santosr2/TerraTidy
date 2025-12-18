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

```rego
package terraform

deny[msg] {
    resource := input.resources[_]
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

has_environment_tag(resource) {
    tags := resource.tags
    contains(tags, "Environment")
}
```

### Policy with Functions

```rego
package terraform

# Helper function to check for required tags
missing_required_tags(resource, required) = missing {
    provided := {tag | resource.tags[tag]}
    missing := required - provided
}

# Check all taggable resources
deny[msg] {
    required_tags := {"Environment", "Team", "CostCenter"}
    resource := input.resources[_]
    taggable_types[resource.type]
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

# Read from external data
import data.config

deny[msg] {
    resource := input.resources[_]
    resource.type == "aws_instance"
    not valid_instance_type(resource.instance_type)
    msg := {
        "msg": sprintf("Instance type %s is not in approved list",
            [resource.instance_type]),
        "rule": "approved-instance-types",
        "severity": "error"
    }
}

valid_instance_type(t) {
    config.approved_instance_types[_] == t
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

import data.terraform

test_require_encryption_pass {
    result := terraform.deny with input as {
        "resources": [{
            "type": "aws_ebs_volume",
            "name": "encrypted_volume",
            "encrypted": "true"
        }]
    }
    count(result) == 0
}

test_require_encryption_fail {
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
