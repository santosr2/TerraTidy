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
```

## Configuration

```yaml
engines:
  fmt:
    enabled: true
    config:
      # No additional config needed - uses HCL defaults
```

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
