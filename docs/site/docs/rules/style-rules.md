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

**Configuration Options:**

| Option       | Type | Default | Description                                 |
|--------------|------|---------|---------------------------------------------|
| `min_lines`  | int  | 1       | Minimum blank lines required between blocks |
| `max_lines`  | int  | 1       | Maximum blank lines allowed between blocks  |

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

**Configuration Example:**

```yaml
engines:
  style:
    rules:
      blank-line-between-blocks:
        enabled: true
        options:
          min_lines: 1
          max_lines: 2  # Allow up to 2 blank lines
```

### no-empty-blocks

Ensures blocks are not empty without content.

| Property | Value |
|----------|-------|
| Rule ID | `style.no-empty-blocks` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Note:** `terraform` and `required_providers` blocks are allowed to be empty by default.

**Configuration Options:**

| Option              | Type     | Default | Description                                       |
|---------------------|----------|---------|---------------------------------------------------|
| `allowed_blocks`    | []string | []      | Additional block types allowed to be empty        |
| `override_defaults` | bool     | false   | If true, only use allowed_blocks (ignore defaults)|

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
| Fixable | Yes |
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
| Fixable | Yes |
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

### scoped-file-organization

Enforces scoped file organization where related resources are grouped in descriptively-named files (e.g., `network.tf` for networking resources).

| Property | Value |
|----------|-------|
| Rule ID | `style.scoped-file-organization` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Scoped File Mappings:**

| File Pattern | Expected Resource Types |
|--------------|-------------------------|
| `network*.tf` | aws_vpc, aws_subnet, aws_route_table, google_compute_network, azurerm_virtual_network |
| `compute*.tf` | aws_instance, aws_launch_template, google_compute_instance, azurerm_virtual_machine |
| `storage*.tf` | aws_s3_bucket, aws_ebs_volume, google_storage_bucket, azurerm_storage_account |
| `database*.tf` | aws_db_instance, aws_rds_cluster, google_sql_database_instance, azurerm_sql_server |
| `iam*.tf` | aws_iam_role, aws_iam_policy, google_service_account, azurerm_role_assignment |
| `security*.tf` | aws_security_group, aws_waf_web_acl, google_compute_firewall, azurerm_network_security_group |
| `monitoring*.tf` | aws_cloudwatch_metric_alarm, aws_sns_topic, google_monitoring_alert_policy |
| `lambda*.tf`, `function*.tf` | aws_lambda_function, google_cloudfunctions_function, azurerm_function_app |
| `container*.tf`, `ecs*.tf`, `kubernetes*.tf` | aws_ecs_cluster, aws_eks_cluster, google_container_cluster, azurerm_kubernetes_cluster |
| `dns*.tf` | aws_route53_zone, aws_route53_record, google_dns_managed_zone, azurerm_dns_zone |

**Example:**

```hcl
# network.tf - Good: contains network-related resources
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "private" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}

# network.tf - Warning: aws_instance should be in compute.tf
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
```

### terraform-files-structure

Enforces standard Terraform file structure conventions.

| Property | Value |
|----------|-------|
| Rule ID | `style.terraform-files-structure` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Standard File Structure:**

| File | Purpose |
|------|---------|
| `main.tf` | Primary resources and configuration |
| `variables.tf` | Variable definitions |
| `outputs.tf` | Output definitions |
| `providers.tf` / `versions.tf` | Provider configurations and version constraints |
| `locals.tf` | Local value definitions |
| `data.tf` | Data source definitions |
| `terraform.tf` | Terraform block (required_version, backend, required_providers) |

**Example:**

```hcl
# main.tf - Warning: missing recommended files
# Consider creating: variables.tf, outputs.tf, providers.tf
```

## Advanced Naming Rules

These rules enforce more specific naming conventions. They are **disabled by default** and can be enabled via configuration.

### resource-name-matches-type

Ensures resource names are descriptive and relate to the resource type, avoiding generic names.

| Property | Value |
|----------|-------|
| Rule ID | `style.resource-name-matches-type` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Generic names flagged:** `this`, `main`, `default`, `foo`, `bar`, `test`, `example`, `temp`, `tmp`

**Example:**

```hcl
# Bad - generic names
resource "aws_instance" "this" { }
resource "aws_s3_bucket" "foo" { }

# Good - descriptive names
resource "aws_instance" "web_server" { }
resource "aws_s3_bucket" "application_logs" { }
```

### output-prefix

Ensures outputs follow naming conventions with configurable prefix/suffix patterns.

| Property | Value |
|----------|-------|
| Rule ID | `style.output-prefix` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Configuration Options:**

| Option   | Type   | Default | Description                    |
|----------|--------|---------|--------------------------------|
| `prefix` | string | ""      | Required prefix for outputs    |
| `suffix` | string | ""      | Required suffix for outputs    |

**Example:**

```hcl
# With config: prefix: "out_"
# Bad
output "instance_ip" { }

# Good
output "out_instance_ip" { }
```

### module-name-convention

Ensures module names follow conventions (no uppercase, no special characters except underscore/hyphen).

| Property | Value |
|----------|-------|
| Rule ID | `style.module-name-convention` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# Bad
module "MyVPC" { }
module "web@server" { }

# Good
module "my_vpc" { }
module "web-server" { }
```

## Documentation Rules

These rules enforce documentation requirements. They are **disabled by default** and can be enabled via configuration.

### require-variable-description

Ensures all variables have a description.

| Property | Value |
|----------|-------|
| Rule ID | `style.require-variable-description` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# Bad - missing description
variable "instance_type" {
  type    = string
  default = "t2.micro"
}

# Good - has description
variable "instance_type" {
  description = "EC2 instance type to use for the web server"
  type        = string
  default     = "t2.micro"
}
```

### require-output-description

Ensures all outputs have a description.

| Property | Value |
|----------|-------|
| Rule ID | `style.require-output-description` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# Bad - missing description
output "instance_ip" {
  value = aws_instance.web.public_ip
}

# Good - has description
output "instance_ip" {
  description = "The public IP address of the web server"
  value       = aws_instance.web.public_ip
}
```

### require-variable-type

Ensures all variables have an explicit type constraint.

| Property | Value |
|----------|-------|
| Rule ID | `style.require-variable-type` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# Bad - missing type
variable "instance_type" {
  default = "t2.micro"
}

# Good - has type
variable "instance_type" {
  type    = string
  default = "t2.micro"
}
```

## Block Organization Rules

These rules enforce consistent ordering within blocks. They are **disabled by default** and can be enabled via configuration.

### meta-arguments-order

Ensures meta-arguments (for_each, count, provider, depends_on) appear in a consistent order.

| Property | Value |
|----------|-------|
| Rule ID | `style.meta-arguments-order` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Expected Order:**

1. `for_each` or `count`
2. `provider`
3. `depends_on`

**Example:**

```hcl
# Bad - wrong order
resource "aws_instance" "web" {
  depends_on = [aws_vpc.main]
  provider   = aws.west
  for_each   = var.instances
}

# Good - correct order
resource "aws_instance" "web" {
  for_each   = var.instances
  provider   = aws.west
  depends_on = [aws_vpc.main]
}
```

### lifecycle-attribute-order

Ensures attributes within lifecycle blocks follow a consistent order.

| Property | Value |
|----------|-------|
| Rule ID | `style.lifecycle-attribute-order` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Expected Order:**

1. `create_before_destroy`
2. `prevent_destroy`
3. `ignore_changes`
4. `replace_triggered_by`

**Example:**

```hcl
# Bad - wrong order
lifecycle {
  ignore_changes        = [tags]
  create_before_destroy = true
}

# Good - correct order
lifecycle {
  create_before_destroy = true
  ignore_changes        = [tags]
}
```

### nested-block-order

Ensures nested blocks within resources follow a consistent order.

| Property | Value |
|----------|-------|
| Rule ID | `style.nested-block-order` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Expected Order:**

1. `connection`
2. `provisioner`
3. `dynamic`
4. `timeouts`
5. `lifecycle`

**Example:**

```hcl
# Bad - lifecycle before timeouts
resource "aws_instance" "web" {
  ami = "ami-12345"

  lifecycle {
    prevent_destroy = true
  }

  timeouts {
    create = "30m"
  }
}

# Good - timeouts before lifecycle
resource "aws_instance" "web" {
  ami = "ami-12345"

  timeouts {
    create = "30m"
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

### one-line-attribute-spacing

Ensures consistent spacing for single-line blocks versus multi-line blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.one-line-attribute-spacing` |
| Default Severity | Info |
| Fixable | No |
| Default | **Disabled** |

**Example:**

```hcl
# Good - single attribute on one line
tags = { Name = "web" }

# Good - multiple attributes on multiple lines
tags = {
  Name        = "web"
  Environment = "prod"
}
```

## Comment and Format Rules

These rules enforce comment syntax and formatting. They are **disabled by default** and can be enabled via configuration.

### comment-syntax

Ensures comments use `#` syntax instead of `//`.

| Property | Value |
|----------|-------|
| Rule ID | `style.comment-syntax` |
| Default Severity | Info |
| Fixable | Yes |
| Default | **Disabled** |

**Example:**

```hcl
// Bad - C-style comment
resource "aws_instance" "web" { }

# Good - HCL-style comment
resource "aws_instance" "web" { }
```

### no-trailing-whitespace

Ensures no trailing whitespace on lines.

| Property | Value |
|----------|-------|
| Rule ID | `style.no-trailing-whitespace` |
| Default Severity | Info |
| Fixable | Yes |
| Default | **Disabled** |

### consistent-quotes

Ensures consistent use of double quotes (Terraform standard).

| Property | Value |
|----------|-------|
| Rule ID | `style.consistent-quotes` |
| Default Severity | Warning |
| Fixable | No |
| Default | **Disabled** |

**Note:** Terraform/HCL requires double quotes for strings. This rule catches malformed files.

**Example:**

```hcl
# Bad - single quotes (invalid HCL)
instance_type = 'test'

# Good - double quotes
instance_type = "test"
```

### no-consecutive-blank-lines

Ensures no more than one consecutive blank line.

| Property | Value |
|----------|-------|
| Rule ID | `style.no-consecutive-blank-lines` |
| Default Severity | Info |
| Fixable | Yes |
| Default | **Disabled** |

**Example:**

```hcl
# Bad - multiple blank lines
resource "aws_instance" "web" { }



resource "aws_instance" "db" { }

# Good - single blank line
resource "aws_instance" "web" { }

resource "aws_instance" "db" { }
```

### attribute-group-spacing

Ensures blank lines separate logical groups of attributes within blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.attribute-group-spacing` |
| Default Severity | Info |
| Fixable | Yes |
| Default | Enabled |

**Attribute Groups:**

1. Meta-arguments (`for_each`, `count`)
2. Source/version (`source`, `version`)
3. Regular attributes
4. Tags (`tags`, `labels`)
5. Meta blocks (`depends_on`, `lifecycle`, `provisioner`)

**Example:**

```hcl
# Good - groups separated by blank lines
resource "aws_instance" "web" {
  count = 3

  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "web"
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

### no-leading-trailing-blank-lines

Ensures no leading or trailing blank lines inside blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.no-leading-trailing-blank-lines` |
| Default Severity | Info |
| Fixable | Yes |
| Default | Enabled |

**Example:**

```hcl
# Bad - leading blank line inside block
resource "aws_instance" "web" {

  ami = "ami-12345"
}

# Bad - trailing blank line inside block
resource "aws_instance" "web" {
  ami = "ami-12345"

}

# Good - no extra blank lines
resource "aws_instance" "web" {
  ami = "ami-12345"
}
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

### Enabled by Default

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `blank-line-between-blocks` | Warning | Yes | Exactly one blank line between blocks |
| `no-empty-blocks` | Warning | No | Blocks should not be empty |
| `no-leading-trailing-blank-lines` | Info | Yes | No leading/trailing blank lines in blocks |
| `attribute-group-spacing` | Info | Yes | Blank lines between attribute groups |
| `block-label-case` | Warning | No | Block labels use snake_case |
| `variable-naming` | Warning | No | Variable names use snake_case |
| `output-naming` | Warning | No | Output names use snake_case |
| `local-naming` | Warning | No | Local value names use snake_case |
| `for-each-count-first` | Warning | Yes | for_each/count as first attribute |
| `lifecycle-at-end` | Warning | Yes | lifecycle block at end |
| `tags-at-end` | Warning | No | tags near end of block |
| `depends-on-order` | Warning | Yes | depends_on at end of block |
| `source-version-grouped` | Warning | No | source and version together |
| `variable-order` | Info | Yes | Variable attribute ordering |
| `output-order` | Info | Yes | Output attribute ordering |
| `terraform-block-first` | Warning | No | terraform block first in file |
| `provider-block-order` | Warning | No | provider after terraform, before resources |

### Disabled by Default (Opt-in)

#### File Organization Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `variables-in-file` | Info | No | Variables in variables.tf |
| `outputs-in-file` | Info | No | Outputs in outputs.tf |
| `providers-in-file` | Info | No | Providers in providers.tf/versions.tf |
| `scoped-file-organization` | Info | No | Resources in scoped files (network.tf, etc.) |
| `terraform-files-structure` | Info | No | Standard file structure conventions |

#### Documentation Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `require-variable-description` | Info | No | Variables must have descriptions |
| `require-output-description` | Info | No | Outputs must have descriptions |
| `require-variable-type` | Info | No | Variables must have explicit types |

#### Advanced Naming Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `resource-name-matches-type` | Info | No | Resource names should be descriptive |
| `output-prefix` | Info | No | Outputs follow prefix/suffix patterns |
| `module-name-convention` | Info | No | Module names follow conventions |

#### Block Organization Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `meta-arguments-order` | Info | No | Meta-arguments in consistent order |
| `lifecycle-attribute-order` | Info | No | Lifecycle attributes in order |
| `nested-block-order` | Info | No | Nested blocks in consistent order |
| `one-line-attribute-spacing` | Info | No | Consistent single vs multi-line blocks |

#### Comment and Format Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `comment-syntax` | Info | Yes | Use # instead of // for comments |
| `no-trailing-whitespace` | Info | Yes | No trailing whitespace on lines |
| `consistent-quotes` | Warning | No | Use double quotes (Terraform standard) |
| `no-consecutive-blank-lines` | Info | Yes | No more than one consecutive blank line |
