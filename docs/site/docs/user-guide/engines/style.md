# Style Engine

The style engine enforces consistent naming conventions and organizational patterns in your Terraform code.

## Overview

The `style` engine checks for naming conventions, attribute ordering, and structural consistency
to maintain a uniform codebase.

## Usage

```bash
# Run style checks
terratidy style

# Show fixable issues
terratidy style --fix

# Check specific directory
terratidy style ./modules/
```

## Configuration

```yaml
engines:
  style:
    enabled: true
    config:
      naming_convention: snake_case  # snake_case, kebab-case, camelCase
      attribute_order:
        - count
        - for_each
        - provider
        - depends_on
        - lifecycle
```

## Rules

### Naming Conventions

| Rule | Description |
|------|-------------|
| `resource-naming` | Resources should follow naming convention |
| `variable-naming` | Variables should follow naming convention |
| `output-naming` | Outputs should follow naming convention |
| `module-naming` | Module calls should follow naming convention |

### Attribute Ordering

The style engine can enforce a consistent attribute order within blocks:

1. Meta-arguments (`count`, `for_each`, `provider`)
2. Required attributes
3. Optional attributes
4. Nested blocks
5. Lifecycle meta-arguments (`depends_on`, `lifecycle`)

### File Organization

| Rule | Description |
|------|-------------|
| `variables-file` | Variables should be in `variables.tf` |
| `outputs-file` | Outputs should be in `outputs.tf` |
| `providers-file` | Provider configs should be in `providers.tf` |

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

## Disabling Rules

Disable specific rules inline:

```hcl
# terratidy:ignore:resource-naming
resource "aws_instance" "MyServer" {
  # ...
}
```

Or in configuration:

```yaml
engines:
  style:
    enabled: true
    rules:
      resource-naming:
        enabled: false
```
