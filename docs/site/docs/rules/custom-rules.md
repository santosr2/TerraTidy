# Custom Rules

Create your own rules using Go plugins, YAML definitions, Bash scripts, or OPA policies.

## Rule Interface

All custom rules implement the `sdk.Rule` interface:

```go
type Rule interface {
    Name() string
    Description() string
    Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error)
}
```

Rules that support auto-fixing also implement `sdk.Fixer`:

```go
type Fixer interface {
    Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error)
}
```

The `sdk.Context` provides runtime information:

```go
type Context struct {
    context.Context              // Embedded for cancellation/deadline support

    Options  map[string]any      // Rule-specific config from .terratidy.yaml
    WorkDir  string              // Directory TerraTidy was invoked from
    File     string              // Absolute path to file being checked
    AllFiles map[string][]byte   // All files being processed (for cross-file rules)
}
```

## Scaffolding

Use `terratidy init-rule` to scaffold new rules:

```bash
terratidy init-rule --name my-rule --type go     # Go plugin with test file
terratidy init-rule --name my-rule --type rego    # OPA/Rego policy with test
terratidy init-rule --name my-rule --type yaml    # YAML rule config
```

Bash rules cannot be scaffolded via `init-rule`. Create them manually (the format is simple).

## Go Plugin Rules

Create custom rules in Go using the plugin system. Go plugins implement
the `sdk.Rule` interface, are compiled as `.so` files, and loaded from
plugin directories at runtime.

For a complete example, SDK reference, plugin exports, and build
instructions, see the [Plugin Development Guide](../development/plugins.md#go-plugins).

## YAML Rules

For simple pattern-based checks, define rules declaratively in YAML.
No Go code required. YAML rules check resource blocks for required
attributes based on patterns you define.

For the full YAML rule structure and examples, see the
[Plugin Development Guide](../development/plugins.md#yaml-rules).

## Bash Rules

Shell scripts that analyze Terraform files and output JSON findings.
They receive a file path as `$1` and output a JSON object with a
`findings` array.

For the contract, examples, and output format, see the
[Plugin Development Guide](../development/plugins.md#bash-rules).

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
