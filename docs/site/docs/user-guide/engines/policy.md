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
terratidy policy --policy-dir ./policies

# Show policy input (for debugging)
terratidy policy --show-input
```

## Configuration

```yaml
engines:
  policy:
    enabled: true
    config:
      policy_dirs:
        - ./policies
        - ~/.terratidy/policies
      policy_files:
        - ./custom-policy.rego
```

## Writing Policies

Policies are written in Rego and evaluated against a JSON representation of your
Terraform modules.

### Basic Policy Structure

```rego
package terraform

# Deny rule - creates an error
deny[msg] {
    resource := input.resources[_]
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
warn[msg] {
    resource := input.resources[_]
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
- `_file`: Source file path
- `_range`: Line/column information
- All attributes as key-value pairs

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

## Example Policies

### Require Encryption

```rego
package terraform

deny[msg] {
    resource := input.resources[_]
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

deny[msg] {
    resource := input.resources[_]
    not re_match("^[a-z][a-z0-9_]*$", resource.name)
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

expensive_types := ["aws_instance", "aws_db_instance", "aws_elasticache_cluster"]

warn[msg] {
    resource := input.resources[_]
    resource.type == expensive_types[_]
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
