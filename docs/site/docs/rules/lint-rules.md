# Lint Rules

Complete reference for lint rules in TerraTidy. The lint engine provides built-in AST-based analysis rules and optional TFLint integration for additional provider-specific checks.

## Built-in Rules

TerraTidy includes 10 built-in lint rules that work without external dependencies.

### terraform-required-version

Ensures the terraform block contains a `required_version` constraint.

| Property         | Value                              |
| ---------------- | ---------------------------------- |
| Rule ID          | `lint.terraform-required-version`  |
| Default Severity | Warning                            |
| Fixable          | No                                 |
| Default          | Enabled                            |

**Example:**

```hcl
# Bad - no required_version
terraform {
  required_providers {
    aws = { source = "hashicorp/aws" }
  }
}

# Good - required_version specified
terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = { source = "hashicorp/aws" }
  }
}
```

### terraform-required-providers

Ensures the terraform block contains a `required_providers` block with version constraints.

| Property         | Value                                 |
| ---------------- | ------------------------------------- |
| Rule ID          | `lint.terraform-required-providers`   |
| Default Severity | Info                                  |
| Fixable          | No                                    |
| Default          | Enabled                               |

**Example:**

```hcl
# Bad - no required_providers
terraform {
  required_version = ">= 1.0"
}

# Good - required_providers with versions
terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
```

### terraform-deprecated-syntax

Detects deprecated interpolation-only expressions like `"${var.x}"`.

| Property         | Value                             |
| ---------------- | --------------------------------- |
| Rule ID          | `lint.terraform-deprecated-syntax`|
| Default Severity | Warning                           |
| Fixable          | No                                |
| Default          | Enabled                           |

**Example:**

```hcl
# Deprecated
resource "aws_instance" "example" {
  ami = "${var.ami_id}"  # Unnecessary interpolation
}

# Correct
resource "aws_instance" "example" {
  ami = var.ami_id
}
```

### terraform-documented-variables

Ensures all variables have description attributes.

| Property         | Value                                  |
| ---------------- | -------------------------------------- |
| Rule ID          | `lint.terraform-documented-variables`  |
| Default Severity | Warning                                |
| Fixable          | No                                     |
| Default          | Enabled                                |

**Example:**

```hcl
# Bad - no description
variable "instance_type" {
  type = string
}

# Good - has description
variable "instance_type" {
  description = "The EC2 instance type to use"
  type        = string
}
```

### terraform-typed-variables

Ensures all variables have explicit type constraints.

| Property         | Value                            |
| ---------------- | -------------------------------- |
| Rule ID          | `lint.terraform-typed-variables` |
| Default Severity | Info                             |
| Fixable          | No                               |
| Default          | Enabled                          |

**Example:**

```hcl
# Bad - no type constraint
variable "instance_type" {
  description = "The EC2 instance type"
  default     = "t2.micro"
}

# Good - explicit type
variable "instance_type" {
  description = "The EC2 instance type"
  type        = string
  default     = "t2.micro"
}
```

### terraform-documented-outputs

Ensures all outputs have description attributes.

| Property         | Value                               |
| ---------------- | ----------------------------------- |
| Rule ID          | `lint.terraform-documented-outputs` |
| Default Severity | Info                                |
| Fixable          | No                                  |
| Default          | Enabled                             |

**Example:**

```hcl
# Bad - no description
output "instance_ip" {
  value = aws_instance.web.public_ip
}

# Good - has description
output "instance_ip" {
  description = "The public IP address of the web instance"
  value       = aws_instance.web.public_ip
}
```

### terraform-module-pinned-source

Ensures module sources are pinned to specific versions or refs.

| Property         | Value                                |
| ---------------- | ------------------------------------ |
| Rule ID          | `lint.terraform-module-pinned-source`|
| Default Severity | Warning                              |
| Fixable          | No                                   |
| Default          | Enabled                              |

**Example:**

```hcl
# Bad - registry module without version
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}

# Good - registry module with version
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}

# Bad - git source without ref
module "vpc" {
  source = "git::https://github.com/example/module.git"
}

# Good - git source with ref
module "vpc" {
  source = "git::https://github.com/example/module.git?ref=v1.0.0"
}
```

### terraform-unused-declarations

Detects declared but unused variables and locals.

| Property         | Value                                 |
| ---------------- | ------------------------------------- |
| Rule ID          | `lint.terraform-unused-declarations`  |
| Default Severity | Warning                               |
| Fixable          | No                                    |
| Default          | Enabled                               |

**Example:**

```hcl
# Warning - variable declared but never used
variable "unused_var" {
  type = string
}

resource "aws_instance" "web" {
  ami = "ami-12345"  # var.unused_var is never referenced
}
```

### terraform-resource-count

Warns when a file has too many resources, suggesting it should be split.

| Property         | Value                            |
| ---------------- | -------------------------------- |
| Rule ID          | `lint.terraform-resource-count`  |
| Default Severity | Info                             |
| Fixable          | No                               |
| Default          | Enabled                          |
| Threshold        | 15 resources per file            |

**Configuration:**

```yaml
engines:
  lint:
    rules:
      lint.terraform-resource-count:
        enabled: true
        config:
          threshold: 10  # Custom threshold
```

### terraform-hardcoded-secrets

Detects potential hardcoded secrets like AWS keys, passwords, and API tokens.

| Property         | Value                               |
| ---------------- | ----------------------------------- |
| Rule ID          | `lint.terraform-hardcoded-secrets`  |
| Default Severity | Error                               |
| Fixable          | No                                  |
| Default          | Enabled                             |

**Detected patterns:**

- AWS Access Keys (AKIA...)
- AWS Secret Keys
- Generic API keys and tokens
- Private keys (PEM format)
- Hardcoded passwords in sensitive attributes

**Example:**

```hcl
# Error - hardcoded AWS key
provider "aws" {
  access_key = "AKIAIOSFODNN7EXAMPLE"  # Detected
  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}

# Good - use variables or environment
provider "aws" {
  # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY env vars
}

# Warning - hardcoded password
resource "aws_db_instance" "db" {
  password = "mySecretPassword123"  # Detected
}

# Good - use variable
resource "aws_db_instance" "db" {
  password = var.db_password
}
```

## TFLint Integration

TerraTidy can optionally invoke TFLint as an external CLI tool (subprocess) for additional
provider-specific rules. TFLint is **not embedded or linked** as a library.

### Enabling TFLint

```yaml
engines:
  lint:
    enabled: true
    use_tflint: true            # Enable TFLint subprocess integration
    config_file: .tflint.hcl    # Path to TFLint config
    plugins:                    # TFLint provider plugins
      - aws
    fallback_builtin: true      # Use built-in rules if TFLint unavailable
```

### TFLint Rules

When TFLint is enabled, rules are prefixed with `tflint.`:

- `tflint.terraform_deprecated_syntax`
- `tflint.aws_instance_invalid_type`
- `tflint.aws_security_group_invalid_protocol`
- And many more from TFLint plugins

### TFLint Config File

Create `.tflint.hcl` for TFLint-specific configuration:

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

## Configuration

### TerraTidy Config

```yaml
engines:
  lint:
    enabled: true
    rules:
      lint.terraform-documented-variables:
        enabled: true
        severity: warning
      lint.terraform-typed-variables:
        enabled: true
        severity: info
      lint.terraform-resource-count:
        enabled: true
        config:
          threshold: 20
```

## Disabling Rules

### Inline (TFLint style)

```hcl
# tflint-ignore: terraform_naming_convention
resource "aws_instance" "WebServer" { }
```

### Inline (TerraTidy style)

```hcl
# terratidy:ignore:lint.terraform-documented-variables
variable "instance_type" { }
```

### Configuration

```yaml
engines:
  lint:
    rules:
      lint.terraform-documented-variables:
        enabled: false
```

## Rule Summary

| Rule                          | Severity | Fixable | Description                                    |
| ----------------------------- | -------- | ------- | ---------------------------------------------- |
| `terraform-required-version`  | Warning  | No      | Requires terraform required_version constraint |
| `terraform-required-providers`| Info     | No      | Requires required_providers block              |
| `terraform-deprecated-syntax` | Warning  | No      | Detects deprecated interpolation syntax        |
| `terraform-documented-variables` | Warning | No   | Variables must have descriptions               |
| `terraform-typed-variables`   | Info     | No      | Variables must have type constraints           |
| `terraform-documented-outputs`| Info     | No      | Outputs must have descriptions                 |
| `terraform-module-pinned-source` | Warning | No   | Module sources must be version-pinned          |
| `terraform-unused-declarations` | Warning | No    | Detects unused variables and locals            |
| `terraform-resource-count`    | Info     | No      | Warns on too many resources per file           |
| `terraform-hardcoded-secrets` | Error    | No      | Detects hardcoded secrets and credentials      |
