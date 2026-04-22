# Style Engine

The style engine enforces consistent naming conventions and organizational patterns in your Terraform code.

## Overview

The `style` engine checks for naming conventions, attribute ordering, and structural consistency
to maintain a uniform codebase.

## Usage

```bash
# Run style checks
terratidy style

# Exit with code 1 if issues found (useful for CI)
terratidy style --check

# Fix style issues
terratidy style --fix

# Preview what would change (without modifying files)
terratidy style --diff

# Fix and show diff of changes
terratidy style --fix --diff

# Check specific directory
terratidy style ./modules/
```

## style --fix vs fmt vs fmt --all

The style engine focuses on semantic organization rather than whitespace formatting:

| Command                 | HCL Formatting | Style Fixes | Use Case                                    |
|-------------------------|----------------|-------------|---------------------------------------------|
| `terratidy fmt`         | ✓              |             | Standard formatting (whitespace, alignment) |
| `terratidy style --fix` |                | ✓           | Style fixes only (naming, block ordering)   |
| `terratidy fmt --all`   | ✓              | ✓           | Complete formatting and style fixes         |

Use `style --fix` when you only want to fix style issues without reformatting whitespace.
Use `fmt --all` or `terratidy fix` when you want both HCL formatting and style fixes applied together.

## Configuration

```yaml
engines:
  style:
    enabled: true
    fix: false   # Auto-fix mode - apply fixes automatically
    diff: false  # Show diff of changes when fixing
    rules:       # Engine-level rule configuration
      style.blank-line-between-blocks:
        enabled: true
        severity: warning

```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the style engine |
| `fix` | bool | `false` | Auto-fix mode - apply fixes automatically |
| `diff` | bool | `false` | Show colored unified diff (works standalone for preview, or with `fix` to show applied changes) |
| `rules` | map | `{}` | Rule configuration (enabled, severity, options) |

## Rules

### Naming Conventions

| Rule | Description |
|------|-------------|
| `style.block-label-case` | Block labels should follow naming convention |
| `style.variable-naming` | Variables should follow naming convention |
| `style.output-naming` | Outputs should follow naming convention |
| `style.local-naming` | Locals should follow naming convention |
| `style.module-name-convention` | Module calls should follow naming convention |

### Attribute Ordering

The style engine enforces attribute ordering through individual rules:

| Rule | Description |
|------|-------------|
| `style.for-each-count-first` | `for_each`/`count` should be first attribute |
| `style.lifecycle-at-end` | `lifecycle` block should be at end |
| `style.tags-at-end` | `tags` should be near end (before lifecycle) |
| `style.depends-on-order` | `depends_on` should be at end |
| `style.source-version-grouped` | `source` and `version` grouped at start of module |
| `style.variable-order` | Variable attributes in consistent order |
| `style.output-order` | Output attributes in consistent order |

See [Style Rules](../../rules/style-rules.md) for complete details.

### File Organization

| Rule | Description |
|------|-------------|
| `style.variables-in-file` | Variables should be in `variables.tf` |
| `style.outputs-in-file` | Outputs should be in `outputs.tf` |
| `style.providers-in-file` | Provider configs should be in `providers.tf` |
| `style.scoped-file-organization` | Resources in scoped files (network.tf, etc.) |
| `style.terraform-files-structure` | Standard file structure conventions |

**Note:** File organization rules are disabled by default. Enable them in configuration.

## Example

Before:

```hcl
resource "aws_instance" "MyServer" {
  lifecycle {
    create_before_destroy = true
  }
  ami           = var.ami_id
  instance_type = "t2.micro"
  count         = 2
}
```

After (with fixes applied):

```hcl
resource "aws_instance" "my_server" {
  count         = 2
  ami           = var.ami_id
  instance_type = "t2.micro"

  lifecycle {
    create_before_destroy = true
  }
}
```

## Plugin Rules

The style engine supports custom rules loaded from plugin directories. Plugin rules
run alongside built-in style rules and produce findings in the same output.

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

A YAML rule that checks for a required attribute:

```yaml
# .terratidy/plugins/require-description.yaml
name: require-description
description: Resources must have a description
severity: warning
enabled: true
message: "Resource is missing a 'description' attribute"
patterns:
  required_attributes:
    - description
```

## Disabling Rules

Suppress specific rules using inline annotations:

```hcl
# Suppress on the next block
# terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" {
  # ...
}

# Suppress inline (same line as code)
resource "aws_s3_bucket" "Test" { } # terratidy:ignore:style.block-label-case

# Suppress for the entire file
# terratidy:ignore-file:style.variable-naming

# Suppress all style rules for the file
# terratidy:ignore-file:style.*
```

Or disable globally in configuration:

```yaml
engines:
  style:
    rules:
      style.block-label-case:
        enabled: false
```
