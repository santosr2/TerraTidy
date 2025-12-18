# Linting with TerraTidy

TerraTidy includes a powerful linting engine to detect errors and best practice violations in your Terraform code.

## Features

- **Built-in Rules**: Common Terraform linting rules
- **Custom Rules**: Configure rule severity and behavior
- **Fast Performance**: Efficient parallel processing
- **Helpful Output**: Clear error messages with locations

## Basic Usage

```bash
# Lint current directory
terratidy lint

# Lint specific files
terratidy lint main.tf variables.tf

# Lint with specific rules
terratidy lint --rule terraform_required_version
```

## Configuration

### In .terratidy.yaml

```yaml
engines:
  lint:
    enabled: true
    config_file: .tflint.hcl

    rules:
      terraform_required_version:
        enabled: true
        severity: error

      terraform_deprecated_syntax:
        enabled: true
        severity: warning

      terraform_documented_variables:
        enabled: true
        severity: warning
```

### In .tflint.hcl (optional)

TerraTidy respects existing `.tflint.hcl` files:

```hcl
rule "terraform_required_version" {
  enabled = true
}

rule "terraform_deprecated_syntax" {
  enabled = true
}
```

## Available Rules

### terraform_required_version

Checks that a `required_version` constraint is specified in the Terraform configuration.

**Example:**

```terraform
terraform {
  required_version = ">= 1.0"
}
```

**Severity:** error (recommended)

### terraform_deprecated_syntax

Detects deprecated Terraform syntax that should be updated.

**Example of deprecated syntax:**

```terraform
# Deprecated
resource "aws_instance" "example" {
  ami = "${var.ami_id}"
}

# Preferred
resource "aws_instance" "example" {
  ami = var.ami_id
}
```

**Severity:** warning

### terraform_unused_declarations

Finds variables or outputs that are declared but never used.

**Example:**

```terraform
# This variable is never referenced
variable "unused_var" {
  type = string
}
```

**Severity:** warning

### terraform_documented_variables

Requires that all variables have a description.

**Example:**

```terraform
# Missing description - will be flagged
variable "region" {
  type = string
}

# Has description - OK
variable "region" {
  type        = string
  description = "AWS region to deploy resources"
}
```

**Severity:** warning

## Command-Line Options

### --config-file

Specify a custom TFLint configuration file:

```bash
terratidy lint --config-file path/to/.tflint.hcl
```

### --plugin

Enable specific plugins:

```bash
terratidy lint --plugin aws
terratidy lint --plugin aws --plugin google
```

### --rule

Enable specific rules:

```bash
terratidy lint --rule terraform_required_version
terratidy lint --rule terraform_required_version --rule terraform_documented_variables
```

## Output Format

TerraTidy provides clear, actionable output:

```
🔍 Running linter on 5 file(s)...

❌ main.tf:1:1 - Missing terraform required_version constraint (lint.terraform_required_version)
⚠️  variables.tf:3:1 - Variable 'region' is missing a description (lint.terraform_documented_variables)
⚠️  variables.tf:8:1 - Variable 'environment' is declared but never used (lint.terraform_unused_declarations)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Summary: 3 issue(s) found
   ❌ Errors:   1
   ⚠️  Warnings: 2
```

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Lint Terraform
  run: terratidy lint

- name: Lint with specific rules
  run: |
    terratidy lint \
      --rule terraform_required_version \
      --rule terraform_documented_variables
```

### GitLab CI

```yaml
lint:
  script:
    - terratidy lint
  only:
    changes:
      - "**/*.tf"
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running Terraform linting..."
if ! terratidy lint; then
    echo "Linting failed. Please fix the issues above."
    exit 1
fi
```

## Customizing Rules

### Severity Levels

- **error**: CI/CD will fail
- **warning**: Warning only, doesn't fail
- **info**: Informational message

### Enabling/Disabling Rules

```yaml
engines:
  lint:
    rules:
      terraform_required_version:
        enabled: true     # Enable this rule
        severity: error   # Make it an error

      terraform_documented_variables:
        enabled: false    # Disable this rule
```

## Best Practices

1. **Run Early**: Lint during development, not just in CI/CD
2. **Start Strict**: Use error severity for critical rules
3. **Document Everything**: Enable `terraform_documented_variables`
4. **Keep Updated**: Use `terraform_required_version` to ensure compatibility
5. **Team Standards**: Share `.terratidy.yaml` in version control

## Troubleshooting

### No Issues Found

- Ensure files have `.tf` extension
- Check that rules are enabled in configuration
- Verify files contain Terraform code

### Too Many Warnings

- Adjust rule severity levels
- Disable non-critical rules
- Fix issues incrementally

### Config File Not Found

TerraTidy looks for `.tflint.hcl` in:

1. Current directory
2. Specified path with `--config-file`

## See Also

- [Configuration Guide](./configuration.md)
- [Style Engine](./style.md)
- [Custom Rules](./custom-rules.md)
- [Example Configurations](../examples/)
