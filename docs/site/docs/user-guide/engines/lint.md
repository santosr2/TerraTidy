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

# Use a custom TFLint config file
terratidy lint --config-file .tflint.custom.hcl

# Enable specific rules
terratidy lint --rule terraform_required_version

# Enable a provider plugin
terratidy lint --plugin aws
```

## Configuration

```yaml
engines:
  lint:
    enabled: true
    config:
      config_file: .tflint.hcl  # Path to TFLint config
      plugins:
        - aws
        - google
```

## Rule Categories

### Built-in Rules

TerraTidy includes 11 built-in Terraform lint rules covering versioning,
documentation, naming, security, and more. A few examples:

| Rule | Description |
|------|-------------|
| `terraform-required-version` | Requires a `terraform.required_version` constraint |
| `terraform-deprecated-syntax` | Detects deprecated Terraform syntax |
| `terraform-unused-declarations` | Finds unused variables and locals |
| `terraform-hardcoded-secrets` | Detects hardcoded secrets in configuration |

For the full list, see [Lint Rules](../../rules/lint-rules.md).

### Provider-Specific Rules

Provider-specific rules (AWS, Google Cloud, Azure) are supplied by TFLint plugins,
not built into TerraTidy. Enable them via your `.tflint.hcl` configuration or the
`plugins` config key. See the [TFLint ruleset registry](https://github.com/terraform-linters/tflint/blob/master/docs/user-guide/plugins.md)
for available provider rulesets.

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

The lint command is read-only and does not modify files. To auto-fix formatting
and style issues, use [`terratidy fix`](../../getting-started/quickstart.md)
or `terratidy style --fix`.

## Disabling Rules

Disable specific rules inline:

```hcl
# terratidy:ignore:terraform-unused-declarations
variable "legacy_var" {
  type = string
}
```

Or globally in configuration:

```yaml
overrides:
  rules:
    terraform-unused-declarations:
      enabled: false
```
