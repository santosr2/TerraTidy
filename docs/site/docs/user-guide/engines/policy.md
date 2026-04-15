# Policy Engine

The policy engine enables custom policy enforcement using OPA (Open Policy Agent) and Rego.

## Overview

The `policy` engine allows you to define and enforce organizational policies on your
Terraform configurations using the powerful Rego policy language.

## Usage

```bash
# Run policy checks
terratidy policy

# With custom policy directory
terratidy policy --policy-dirs ./policies

# Show policy input (for debugging)
terratidy policy --show-input
```

## Configuration

```yaml
engines:
  policy:
    enabled: true
    policy_dirs:                    # Directories containing Rego policy files
      - ./policies
      - ~/.terratidy/policies
    policy_files:                   # Individual policy files
      - ./custom-policy.rego
    data_files:                     # Additional data files for policies
      - ./policy-data.json
    rules:                          # Rule-specific configuration
      policy.required-tags:
        enabled: true
        severity: error
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable the policy engine (opt-in) |
| `policy_dirs` | list | `[]` | Directories containing Rego policy files |
| `policy_files` | list | `[]` | Individual policy files to load |
| `data_files` | list | `[]` | JSON data files loaded into OPA storage (accessible as `data.<key>` in Rego) |
| `rules` | map | `{}` | Rule-specific configuration |

## Writing Policies

Policies are written in Rego (v1 syntax) and evaluated against a JSON representation
of your Terraform modules. TerraTidy uses **OPA v1.15.0** with Rego v1, which requires
the `import rego.v1` statement and updated rule syntax.

### Basic Policy Structure

```rego
package terraform

import rego.v1

# Deny rule - creates an error
deny contains msg if {
    some resource in input.resources
    resource.type == "aws_s3_bucket"
    not resource.versioning
    msg := {
        "msg": sprintf("S3 bucket %s must have versioning enabled", [resource.name]),
        "rule": "s3-versioning-required",
        "severity": "error",
        "file": resource._file
    }
}

# Warn rule - creates a warning
warn contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not resource.tags
    msg := {
        "msg": sprintf("EC2 instance %s should have tags", [resource.name]),
        "rule": "required-tags",
        "severity": "warning",
        "file": resource._file
    }
}
```

### Key Rego v1 Syntax Changes

- Add `import rego.v1` at the top of every policy file
- Use `deny contains msg if { ... }` instead of `deny[msg] { ... }`
- Use `some resource in input.resources` instead of `resource := input.resources[_]`

### Input Structure

The policy engine provides the following input structure:

```json
{
  "resources": [...],
  "data": [...],
  "modules": [...],
  "variables": [...],
  "outputs": [...],
  "locals": [...],
  "providers": [...],
  "terraform": {...},
  "_files": [...]
}
```

Each resource/block includes:

- `type`: The resource type (e.g., "aws_instance")
- `name`: The resource name
- `_block_type`: The HCL block type (resource, data, module, etc.)
- `_file`: Source file path where the block is defined
- `_range`: Location object with `start_line`, `end_line`, `start_column`, `end_column`
- All attributes as key-value pairs (raw expression text)

The `_range` field is useful for precise error reporting in custom policies:

```rego
msg := {
    "msg": "Issue description",
    "rule": "my-rule",
    "severity": "error",
    "file": resource._file,
    "line": resource._range.start_line
}
```

## Built-in Policies

TerraTidy includes several built-in policies:

| Policy | Description |
|--------|-------------|
| `required-terraform-block` | Terraform block must exist |
| `required-version` | `required_version` must be specified |
| `required-providers` | Providers must have version constraints |
| `no-public-ssh` | Security groups cannot allow SSH from 0.0.0.0/0 |
| `no-public-s3` | S3 buckets cannot have public-read ACL |
| `no-public-rds` | RDS instances cannot be publicly accessible |
| `required-tags` | Resources should have tags |
| `module-version` | External modules should have version constraints |

## Disabling Rules

Suppress specific policy rules using inline annotations:

```hcl
# Suppress on the next block
# terratidy:ignore:policy.required-tags
resource "aws_instance" "temporary" {
  # ...
}

# Suppress inline (same line as code)
resource "aws_s3_bucket" "logs" { } # terratidy:ignore:policy.no-public-s3

# Suppress for the entire file
# terratidy:ignore-file:policy.required-version

# Suppress all policy rules for the file
# terratidy:ignore-file:policy.*
```

Or disable globally in configuration:

```yaml
engines:
  policy:
    rules:
      policy.required-tags:
        enabled: false
```

## Example Policies

### Require Encryption

```rego
package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_ebs_volume"
    resource.encrypted != "true"
    msg := {
        "msg": sprintf("EBS volume %s must be encrypted", [resource.name]),
        "rule": "ebs-encryption",
        "severity": "error",
        "file": resource._file
    }
}
```

### Naming Convention

```rego
package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    not regex.match("^[a-z][a-z0-9_]*$", resource.name)
    msg := {
        "msg": sprintf("Resource %s.%s must use snake_case naming", [resource.type, resource.name]),
        "rule": "naming-convention",
        "severity": "warning",
        "file": resource._file
    }
}
```

### Cost Control

```rego
package terraform

import rego.v1

expensive_types := {"aws_instance", "aws_db_instance", "aws_elasticache_cluster"}

warn contains msg if {
    some resource in input.resources
    resource.type in expensive_types
    not resource.tags.CostCenter
    msg := {
        "msg": sprintf("%s %s should have a CostCenter tag", [resource.type, resource.name]),
        "rule": "cost-center-tag",
        "severity": "warning",
        "file": resource._file
    }
}
```

## Debugging Policies

Use the `--show-input` flag to see the JSON input:

```bash
terratidy policy --show-input > input.json
```

Then test your policy with OPA directly:

```bash
opa eval --input input.json --data policies/ "data.terraform.deny"
```
