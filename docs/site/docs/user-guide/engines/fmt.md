# Format Engine (fmt)

The format engine ensures consistent formatting across your Terraform and Terragrunt files.

## Overview

The `fmt` engine uses the HCL formatter to standardize code formatting, similar to
`terraform fmt` but with additional capabilities.

## Usage

```bash
# Format all files
terratidy fmt

# Check formatting without changes
terratidy fmt --check

# Show diff of changes
terratidy fmt --diff

# Format only changed files
terratidy fmt --changed

# Format and apply style fixes
terratidy fmt --all
```

## fmt vs style --fix vs fmt --all

TerraTidy provides different commands for different formatting needs:

| Command                 | HCL Formatting | Style Fixes | Use Case                                    |
|-------------------------|----------------|-------------|---------------------------------------------|
| `terratidy fmt`         | ✓              |             | Standard formatting (whitespace, alignment) |
| `terratidy style --fix` |                | ✓           | Style fixes only (naming, block ordering)   |
| `terratidy fmt --all`   | ✓              | ✓           | Complete formatting and style fixes         |
| `terratidy fix`         | ✓              | ✓           | Same as `fmt --all` (alias)                 |

- **`fmt`**: Applies HCL formatting rules (indentation, alignment, spacing). This is equivalent to `terraform fmt`.
- **`style --fix`**: Applies TerraTidy style rules (naming conventions, block ordering, file organization) without HCL formatting.
- **`fmt --all`**: Combines both - first formats files, then applies style fixes. Use this when you want complete code cleanup.

## Configuration

```yaml
engines:
  fmt:
    enabled: true
    check: false  # Check mode - report but don't modify files
    diff: false   # Show diff of changes
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the format engine |
| `check` | bool | `false` | Check mode - report formatting issues without modifying files |
| `diff` | bool | `false` | Show unified diff of changes |

## What Gets Formatted

- Indentation (2 spaces)
- Attribute alignment
- Block spacing
- Trailing whitespace removal
- Consistent line endings

## Example

Before:

```hcl
resource "aws_instance" "example" {
ami = "ami-12345"
  instance_type="t2.micro"
    tags={
Name="example"
  }
}
```

After:

```hcl
resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  tags = {
    Name = "example"
  }
}
```

## Integration with CI/CD

Use `--check` in CI to fail if files need formatting:

```yaml
- name: Check formatting
  run: terratidy fmt --check
```
