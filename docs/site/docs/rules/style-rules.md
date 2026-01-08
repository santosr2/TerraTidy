# Style Rules

Complete reference for all style rules enforced by the style engine.

## Block Spacing Rules

### blank-line-between-blocks

Ensures there is exactly one blank line between top-level blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.blank-line-between-blocks` |
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

**Example:**

```hcl
# Bad - no blank line between blocks
resource "aws_instance" "web" {
  ami = "ami-12345"
}
resource "aws_instance" "db" {
  ami = "ami-67890"
}

# Good - exactly one blank line
resource "aws_instance" "web" {
  ami = "ami-12345"
}

resource "aws_instance" "db" {
  ami = "ami-67890"
}
```

### no-empty-blocks

Ensures blocks are not empty without content.

| Property | Value |
|----------|-------|
| Rule ID | `style.no-empty-blocks` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Note:** `lifecycle` and `provisioner` blocks are allowed to be empty.

**Example:**

```hcl
# Bad - empty resource block
resource "aws_instance" "empty" {
}

# Good - block has content
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
```

## Naming Rules

### block-label-case

Ensures block labels follow naming conventions (snake_case for resources/data).

| Property | Value |
|----------|-------|
| Rule ID | `style.block-label-case` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - camelCase or PascalCase
resource "aws_instance" "MyServer" { }
resource "aws_instance" "webServer" { }

# Good - snake_case
resource "aws_instance" "my_server" { }
resource "aws_instance" "web_server" { }
```

### variable-naming

Ensures variable names follow snake_case convention.

| Property | Value |
|----------|-------|
| Rule ID | `style.variable-naming` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad
variable "instanceType" { }
variable "MyVariable" { }

# Good
variable "instance_type" { }
variable "my_variable" { }
```

### output-naming

Ensures output names follow snake_case convention.

| Property | Value |
|----------|-------|
| Rule ID | `style.output-naming` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad
output "instanceIP" { }
output "MyOutput" { }

# Good
output "instance_ip" { }
output "my_output" { }
```

### local-naming

Ensures local value names follow snake_case convention.

| Property | Value |
|----------|-------|
| Rule ID | `style.local-naming` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad
locals {
  instanceName = "web"
  MyLocal      = "value"
}

# Good
locals {
  instance_name = "web"
  my_local      = "value"
}
```

## Attribute Ordering Rules

### for-each-count-first

Ensures `for_each` or `count` is the first attribute in resource/module/data blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.for-each-count-first` |
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

**Example:**

```hcl
# Bad - count not first
resource "aws_instance" "web" {
  ami   = "ami-12345"
  count = 3
}

# Good - count first
resource "aws_instance" "web" {
  count = 3
  ami   = "ami-12345"
}
```

### lifecycle-at-end

Ensures `lifecycle` block is at the end of resource blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.lifecycle-at-end` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - lifecycle not at end
resource "aws_instance" "web" {
  lifecycle {
    prevent_destroy = true
  }
  ami           = "ami-12345"
  instance_type = "t2.micro"
}

# Good - lifecycle at end
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  lifecycle {
    prevent_destroy = true
  }
}
```

### tags-at-end

Ensures `tags`/`labels` are near the end of resource blocks (before lifecycle).

| Property | Value |
|----------|-------|
| Rule ID | `style.tags-at-end` |
| Default Severity | Warning/Info |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - tags in the middle
resource "aws_instance" "web" {
  ami = "ami-12345"
  tags = {
    Name = "web"
  }
  instance_type = "t2.micro"
}

# Good - tags near end
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "web"
  }
}
```

### depends-on-order

Ensures `depends_on` is at the end of resource/module blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.depends-on-order` |
| Default Severity | Warning/Info |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - depends_on not at end
resource "aws_instance" "web" {
  depends_on = [aws_vpc.main]
  ami        = "ami-12345"
}

# Good - depends_on at end
resource "aws_instance" "web" {
  ami = "ami-12345"

  depends_on = [aws_vpc.main]
}
```

### source-version-grouped

Ensures `source` and `version` are grouped at the start of module blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.source-version-grouped` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - version not after source
module "vpc" {
  source     = "terraform-aws-modules/vpc/aws"
  vpc_name   = "my-vpc"
  version    = "3.0.0"
}

# Good - source and version grouped
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"

  vpc_name = "my-vpc"
}
```

### variable-order

Ensures variable blocks follow standard attribute ordering.

| Property | Value |
|----------|-------|
| Rule ID | `style.variable-order` |
| Default Severity | Info |
| Fixable | Yes |
| Default | Enabled |

**Expected Order:**

1. `description`
2. `type`
3. `default`
4. `sensitive`
5. `nullable`
6. `validation` blocks

**Example:**

```hcl
# Bad - wrong order
variable "instance_type" {
  default     = "t2.micro"
  type        = string
  description = "EC2 instance type"
}

# Good - correct order
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t2.micro"
}
```

### output-order

Ensures output blocks follow standard attribute ordering.

| Property | Value |
|----------|-------|
| Rule ID | `style.output-order` |
| Default Severity | Info |
| Fixable | Yes |
| Default | Enabled |

**Expected Order:**

1. `description`
2. `value`
3. `sensitive`
4. `depends_on`

**Example:**

```hcl
# Bad - wrong order
output "instance_ip" {
  value       = aws_instance.web.public_ip
  description = "The public IP"
}

# Good - correct order
output "instance_ip" {
  description = "The public IP"
  value       = aws_instance.web.public_ip
}
```

## Block Order Rules

### terraform-block-first

Ensures `terraform` block is the first block in the file.

| Property | Value |
|----------|-------|
| Rule ID | `style.terraform-block-first` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - terraform block not first
provider "aws" {
  region = "us-east-1"
}

terraform {
  required_version = ">= 1.0"
}

# Good - terraform block first
terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-east-1"
}
```

### provider-block-order

Ensures provider blocks come after terraform block and before resources.

| Property | Value |
|----------|-------|
| Rule ID | `style.provider-block-order` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad - provider after resources
resource "aws_instance" "web" {
  ami = "ami-12345"
}

provider "aws" {
  region = "us-east-1"
}

# Good - provider before resources
provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "web" {
  ami = "ami-12345"
}
```

## File Organization Rules

These rules help enforce consistent file organization patterns. They are **disabled by default** and can be enabled via configuration.

### variables-in-file

Variables should be defined in `variables.tf`.

| Property | Value |
|----------|-------|
| Rule ID | `style.variables-in-file` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# main.tf - Warning: variable should be in variables.tf
variable "instance_type" {
  type = string
}

resource "aws_instance" "web" {
  instance_type = var.instance_type
}
```

### outputs-in-file

Outputs should be defined in `outputs.tf`.

| Property | Value |
|----------|-------|
| Rule ID | `style.outputs-in-file` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# main.tf - Warning: output should be in outputs.tf
output "instance_ip" {
  value = aws_instance.web.public_ip
}
```

### providers-in-file

Provider configurations should be in `providers.tf` or `versions.tf`.

| Property | Value |
|----------|-------|
| Rule ID | `style.providers-in-file` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# main.tf - Warning: provider should be in providers.tf or versions.tf
provider "aws" {
  region = "us-east-1"
}
```

**Enable file organization rules:**

```yaml
engines:
  style:
    rules:
      variables-in-file:
        enabled: true
      outputs-in-file:
        enabled: true
      providers-in-file:
        enabled: true
```

## Configuration

### TerraTidy Config

```yaml
engines:
  style:
    enabled: true
    rules:
      blank-line-between-blocks:
        enabled: true
        severity: warning
      block-label-case:
        enabled: true
        severity: warning
      for-each-count-first:
        enabled: true
      variable-order:
        enabled: true
        severity: info
```

## Disabling Rules

### Inline

```hcl
# terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }
```

### Configuration

```yaml
engines:
  style:
    rules:
      block-label-case:
        enabled: false
```

### File-level

```hcl
# terratidy:ignore-file:style.block-label-case
```

## Rule Summary

| Rule | Severity | Fixable | Default | Description |
|------|----------|---------|---------|-------------|
| `blank-line-between-blocks` | Warning | Yes | Enabled | Exactly one blank line between blocks |
| `no-empty-blocks` | Warning | No | Enabled | Blocks should not be empty |
| `block-label-case` | Warning | No | Enabled | Block labels use snake_case |
| `variable-naming` | Warning | No | Enabled | Variable names use snake_case |
| `output-naming` | Warning | No | Enabled | Output names use snake_case |
| `local-naming` | Warning | No | Enabled | Local value names use snake_case |
| `for-each-count-first` | Warning | Yes | Enabled | for_each/count as first attribute |
| `lifecycle-at-end` | Warning | No | Enabled | lifecycle block at end |
| `tags-at-end` | Warning | No | Enabled | tags near end of block |
| `depends-on-order` | Warning | No | Enabled | depends_on at end of block |
| `source-version-grouped` | Warning | No | Enabled | source and version together |
| `variable-order` | Info | Yes | Enabled | Variable attribute ordering |
| `output-order` | Info | Yes | Enabled | Output attribute ordering |
| `terraform-block-first` | Warning | No | Enabled | terraform block first in file |
| `provider-block-order` | Warning | No | Enabled | provider after terraform, before resources |
| `variables-in-file` | Info | No | Disabled | Variables in variables.tf |
| `outputs-in-file` | Info | No | Disabled | Outputs in outputs.tf |
| `providers-in-file` | Info | No | Disabled | Providers in providers.tf/versions.tf |
