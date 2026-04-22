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
terratidy lint --tflint-config .tflint.custom.hcl

# Enable specific rules
terratidy lint --rule terraform-required-version

# Enable a provider plugin
terratidy lint --plugin aws
```

## Configuration

```yaml
engines:
  lint:
    enabled: true
    config_file: .tflint.hcl      # Path to TFLint config
    plugins:                       # TFLint plugins to enable
      - aws
      - google
    args:                          # Extra arguments to pass to TFLint
      - --force
      - --no-color
    use_tflint: true               # Enable TFLint subprocess integration
    tflint_path: /usr/local/bin/tflint  # Custom TFLint binary path
    fallback_builtin: true         # Use built-in rules if TFLint unavailable
    rules:                         # Rule-specific configuration
      lint.terraform-deprecated-syntax:
        enabled: true
        severity: error
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the lint engine |
| `config_file` | string | `.tflint.hcl` | Path to TFLint configuration file |
| `plugins` | list | `[]` | TFLint plugins to enable |
| `args` | list | `[]` | Extra arguments to pass to TFLint |
| `use_tflint` | bool | `false` | Enable TFLint subprocess integration |
| `tflint_path` | string | `tflint` | Custom path to TFLint binary |
| `fallback_builtin` | bool | `false` | Use built-in rules if TFLint is unavailable |
| `rules` | map | `{}` | Rule-specific configuration |

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

The lint command is a **read-only analysis** that never modifies files. Unlike
`fmt` and `style`, there is no `--check` or `--diff` flag because lint only
reports issues. To fix lint issues, you must edit your Terraform code manually.

For formatting and style issues that can be auto-fixed, use
[`terratidy fix`](../../getting-started/quickstart.md) or `terratidy style --fix`.

## Plugin Rules

The lint engine supports custom rules loaded from plugin directories. Plugin rules
run alongside built-in lint rules and produce findings in the same output.

### Enabling Plugin Rules

```yaml
plugins:
  enabled: true
  directories:
    - .terratidy/plugins
    - ~/.terratidy/plugins
```

Plugin rules can be YAML files, Bash scripts, or Go plugins. See the
[Plugin Development](../../development/plugins.md) guide for details on creating custom rules.

### Plugin Rule Example

A YAML rule that checks for required tags:

```yaml
# .terratidy/plugins/require-tags.yaml
name: require-tags
description: Resources must have tags
severity: warning
enabled: true
message: "Resource is missing a 'tags' attribute"
patterns:
  required_attributes:
    - tags
```

### Configuring Plugin Rules

Plugin rules can be enabled, disabled, or have their severity overridden:

```yaml
plugins:
  enabled: true
  directories:
    - .terratidy/plugins
  rules:
    require-tags:
      enabled: true
      severity: error  # Override default severity
```

## Disabling Rules

Suppress specific rules using inline annotations:

```hcl
# Suppress on the next block
# terratidy:ignore:lint.terraform-unused-declarations
variable "legacy_var" {
  type = string
}

# Suppress inline (same line as code)
variable "temp_var" { type = string } # terratidy:ignore:lint.terraform-typed-variables

# Suppress for the entire file
# terratidy:ignore-file:lint.terraform-required-version

# Suppress all lint rules for the file
# terratidy:ignore-file:lint.*
```

Or disable globally in configuration:

```yaml
engines:
  lint:
    rules:
      lint.terraform-unused-declarations:
        enabled: false
```

## License Note

TFLint is invoked as an external CLI tool (subprocess), not embedded or linked as a library.
TFLint uses MPL-2.0 for its own code and BUSL-1.1 for embedded Terraform code (required since
Terraform's license change in August 2023). TerraTidy's subprocess invocation pattern is
compliant with these licenses. TerraTidy itself remains MIT-licensed.
