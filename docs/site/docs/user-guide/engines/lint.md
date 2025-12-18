# Lint Engine

The lint engine performs static analysis to detect potential errors, best practice violations,
and security issues in your Terraform code.

## Overview

The `lint` engine uses TFLint under the hood, providing deep analysis of Terraform configurations
including provider-specific rules.

## Usage

```bash
# Run linting
terratidy lint

# With specific ruleset
terratidy lint --ruleset aws

# Show all issues including info
terratidy lint --severity info
```

## Configuration

```yaml
engines:
  lint:
    enabled: true
    config:
      tflint_config: .tflint.hcl  # Path to TFLint config
      rulesets:
        - aws
        - google
```

## Rule Categories

### Terraform Core Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `deprecated-syntax` | Warning | Detects deprecated Terraform syntax |
| `unused-declarations` | Warning | Finds unused variables and locals |
| `missing-required` | Error | Missing required attributes |

### AWS Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `aws-instance-type` | Warning | Invalid EC2 instance type |
| `aws-region` | Error | Invalid AWS region |
| `aws-security-group` | Warning | Overly permissive security groups |

### Google Cloud Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `google-machine-type` | Warning | Invalid machine type |
| `google-zone` | Error | Invalid zone |

### Azure Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `azure-vm-size` | Warning | Invalid VM size |
| `azure-location` | Error | Invalid location |

## TFLint Integration

TerraTidy integrates with TFLint for comprehensive linting. You can use existing
TFLint configuration files:

```hcl
# .tflint.hcl
plugin "aws" {
  enabled = true
  version = "0.27.0"
  source  = "github.com/terraform-linters/tflint-ruleset-aws"
}

rule "aws_instance_invalid_type" {
  enabled = true
}
```

## Example Output

```text
main.tf:15:1: error: aws_instance_invalid_type - "t2.superxlarge" is an invalid value as instance_type
main.tf:23:5: warning: aws_security_group_rule - Security group allows unrestricted ingress
variables.tf:8:1: warning: terraform_unused_declarations - variable "unused_var" is declared but not used
```

## Fixing Issues

Some lint issues can be auto-fixed:

```bash
# Auto-fix fixable issues
terratidy lint --fix
```

## Disabling Rules

Disable specific rules:

```hcl
# terratidy:ignore:aws-instance-type
resource "aws_instance" "example" {
  instance_type = "custom.type"
}
```

Or globally in configuration:

```yaml
engines:
  lint:
    rules:
      aws-instance-type:
        enabled: false
```
