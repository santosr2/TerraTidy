# Lint Rules

Complete reference for lint rules powered by TFLint integration.

## Terraform Core Rules

### terraform_deprecated_syntax

Detects deprecated Terraform syntax.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | Some |
| Default | Enabled |

**Example:**

```hcl
# Deprecated
resource "aws_instance" "example" {
  count = "${var.count}"  # Deprecated interpolation
}

# Correct
resource "aws_instance" "example" {
  count = var.count
}
```

### terraform_unused_declarations

Finds unused variables, locals, and data sources.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

### terraform_required_version

Requires `required_version` in terraform block.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

### terraform_required_providers

Requires version constraints for all providers.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

## AWS Rules

### aws_instance_invalid_type

Validates EC2 instance types exist.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Error: Invalid instance type
resource "aws_instance" "example" {
  instance_type = "t2.superxlarge"  # Does not exist
}
```

### aws_instance_previous_type

Warns about previous generation instance types.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

### aws_security_group_invalid_protocol

Validates security group protocol values.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

### aws_s3_bucket_invalid_acl

Validates S3 bucket ACL values.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

### aws_db_instance_invalid_type

Validates RDS instance types.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

### aws_elasticache_cluster_invalid_type

Validates ElastiCache node types.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

## Google Cloud Rules

### google_compute_instance_invalid_machine_type

Validates GCE machine types.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

### google_compute_instance_invalid_zone

Validates GCE zones.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

## Azure Rules

### azurerm_virtual_machine_invalid_size

Validates Azure VM sizes.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

### azurerm_resource_invalid_location

Validates Azure resource locations.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

## Security Rules

### security_group_unrestricted_ingress

Detects security groups allowing unrestricted ingress.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Warning: Unrestricted ingress
resource "aws_security_group_rule" "example" {
  type        = "ingress"
  from_port   = 22
  to_port     = 22
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"]  # Too permissive
}
```

### public_s3_bucket

Detects S3 buckets with public access.

| Property | Value |
|----------|-------|
| Default Severity | Error |
| Fixable | No |
| Default | Enabled |

## Configuration

### TFLint Config File

Create `.tflint.hcl` for detailed TFLint configuration:

```hcl
plugin "aws" {
  enabled = true
  version = "0.27.0"
  source  = "github.com/terraform-linters/tflint-ruleset-aws"
}

rule "aws_instance_invalid_type" {
  enabled = true
}

rule "aws_instance_previous_type" {
  enabled = false
}
```

### TerraTidy Config

```yaml
engines:
  lint:
    enabled: true
    config:
      tflint_config: .tflint.hcl
    rules:
      terraform_unused_declarations:
        enabled: true
        severity: error
```

## Disabling Rules

### Inline

```hcl
# tflint-ignore: aws_instance_invalid_type
resource "aws_instance" "example" {
  instance_type = "custom.type"
}
```

### Configuration

```yaml
engines:
  lint:
    rules:
      aws_instance_invalid_type:
        enabled: false
```
