# Lint Engine

The lint engine performs static analysis to detect potential errors, best practice violations,
and security issues in your Terraform code.

## Overview

The `lint` engine provides static analysis through built-in AST-based rules and optional TFLint
integration. Built-in rules cover core Terraform hygiene (versioning, naming, security), while
TFLint integration adds provider-specific checks when TFLint is installed.

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

TerraTidy can optionally invoke TFLint as an external CLI tool (subprocess) for comprehensive
provider-specific linting. TFLint is **not embedded or linked** as a library. You can use
existing TFLint configuration files:

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

## License Note

TFLint is invoked as an external CLI tool (subprocess), not embedded or linked as a library.
TFLint uses MPL-2.0 for its own code and BUSL-1.1 for embedded Terraform code (required since
Terraform's license change in August 2023). TerraTidy's subprocess invocation pattern is
compliant with these licenses. TerraTidy itself remains MIT-licensed.
