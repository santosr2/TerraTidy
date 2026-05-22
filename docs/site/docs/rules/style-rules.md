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
      style.blank-line-between-blocks:
        enabled: true
        config:
          options:
            min_lines: 1
            max_lines: 2  # Allow up to 2 blank lines
```

### no-empty-blocks

Ensures blocks are not empty without content.

| Property | Value |
|----------|-------|
| Rule ID | `style.no-empty-blocks` |
| Default Severity | Info |
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

Ensures variable names follow naming conventions.

| Property | Value |
|----------|-------|
| Rule ID | `style.variable-naming` |
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

**Configuration Options:**

| Option    | Type   | Default      | Description                                              |
|-----------|--------|--------------|----------------------------------------------------------|
| `case`    | string | `snake_case` | Naming convention: `snake_case`, `camelCase`, `kebab-case`, `PascalCase`, `custom` |
| `pattern` | string | ""           | Custom regex pattern (only used when case is `custom`)   |

**Example:**

```hcl
# Bad (with default snake_case)
variable "instanceType" { }
variable "MyVariable" { }

# Good
variable "instance_type" { }
variable "my_variable" { }
```

**Configuration Example:**

```yaml
engines:
  style:
    rules:
      style.variable-naming:
        enabled: true
        config:
          options:
            case: camelCase  # Allow camelCase instead of snake_case
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

Ensures `lifecycle` block is at the end of `resource`, `data`, `module`, and `check` blocks.

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

Ensures `tags`/`labels` appear just before any `lifecycle` block in `resource`
and `module` blocks, after all other attributes and nested blocks.

| Property | Value |
|----------|-------|
| Rule ID | `style.tags-at-end` |
| Default Severity | Warning / Info (see below) |
| Fixable | Yes |
| Default | Enabled |

The rule fires when any attribute or nested block appears after `tags`/`labels`,
other than the three items conventionally tolerated as trailing: `tags_all`
(provider-derived from inherited tags), `depends_on` (a meta-argument that
typically goes last), and `lifecycle` (the trailing nested block the fix moves
`tags` in front of). A single trailing item is enough to flag the violation —
the older "more than two trailing attributes" threshold was relaxed so that
layouts like `tags = {} ; ingress {} ; lifecycle {}` no longer pass silently.

Severity is `warning` when `tags`/`labels` is placed **after** a `lifecycle`
block (the more visually disruptive misplacement). When `tags` is in the wrong
spot but ahead of any `lifecycle` block (or no `lifecycle` block is present),
severity is `info`. Both paths share the same fix — moving `tags` to land just
before `lifecycle`, or at the end of the block if no `lifecycle` is present.

When both `tags` and `tags_all` are present the rule operates on `tags`; `labels`
takes precedence over `tags_all` for providers that use it. The fix is
non-destructive: only the `tags` region (including its leading comment) moves;
other attributes and nested blocks stay in their source positions.

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

Ensures `depends_on` is at the end of `resource`, `data`, and `module` blocks (just before any `lifecycle` block).

| Property | Value |
|----------|-------|
| Rule ID | `style.depends-on-order` |
| Default Severity | Warning |
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
| Fixable | Yes |
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
| Fixable | Yes |
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
| Fixable | Yes |
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

Variables should be defined in `variables.tf`. The singular `variable.tf` is also
accepted as a valid container; suggestions still point to the canonical `variables.tf`.

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

Outputs should be defined in `outputs.tf`. The singular `output.tf` is also accepted
as a valid container; suggestions still point to the canonical `outputs.tf`.

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

Provider configurations should be in `providers.tf` or `versions.tf`. The singular
`provider.tf` is also accepted as a valid container; suggestions still point to the
canonical `providers.tf`.

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
      style.variables-in-file:
        enabled: true
      style.outputs-in-file:
        enabled: true
      style.providers-in-file:
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
| `variables.tf` (or `variable.tf`) | Variable definitions |
| `outputs.tf` (or `output.tf`) | Output definitions |
| `providers.tf` (or `provider.tf`) / `versions.tf` | Provider configurations and version constraints |
| `locals.tf` | Local value definitions |
| `data.tf` | Data source definitions |
| `terraform.tf` | Terraform block (required_version, backend, required_providers) |

Singular file names (`variable.tf`, `output.tf`, `provider.tf`) are recognized as
valid containers. The rule will not suggest moving a block when either the plural
or singular form already exists in the directory.

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

## Block Organization Rules

These rules enforce consistent ordering within blocks. They are **disabled by default** and can be enabled via configuration.

### meta-arguments-order

Ensures meta-arguments (for_each, count, provider, depends_on) appear in a consistent order.

| Property | Value |
|----------|-------|
| Rule ID | `style.meta-arguments-order` |
| Default Severity | Info |
| Fixable | Yes |
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
| Fixable | Yes |
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

1. `timeouts`
2. `connection`
3. `provisioner`
4. `lifecycle`

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

Ensures full-line comments use `#` syntax instead of `//`. Only lines whose first
non-whitespace characters are `//` are flagged; trailing `//` after a value and
`#` comments whose body contains `//` (for example URLs) pass through unchanged.

| Property | Value |
|----------|-------|
| Rule ID | `style.comment-syntax` |
| Default Severity | Info |
| Fixable | Yes |
| Default | **Disabled** |

**Flagged:**

```hcl
// Bad - full-line C-style comment
resource "aws_instance" "web" { }
```

`--fix` rewrites the leading `//` to `#` while preserving any text after it.

**Not flagged:**

```hcl
# Good - HCL-style comment
resource "aws_instance" "web" { }

# See https://example.com/path - `//` inside a # comment is fine
resource "aws_instance" "web2" {
  ami = "ami-12345" // trailing comment after a value is also fine
}
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

## Multi-Pass Fixing

When running with `--fix`, the style engine applies fixes in a loop until a fixed point is
reached. Some rules interact with each other (e.g., reordering attributes may create new
spacing issues), so the engine re-runs all rules after applying fixes to catch any new
violations introduced by the previous pass.

```bash
# Fix mode loops until no further fixes apply
terratidy style --fix

# Show what would change without modifying files
terratidy style --fix --diff
```

Each pass:

1. Parses the file fresh
2. Hashes the pre-fix content, checks it against the set of hashes seen this run, and adds it to that set on non-cycle passes
3. Runs all enabled rules
4. Applies fixes from fixable findings
5. If any fixes were applied, continues to the next pass

The engine stops as soon as a pass applies no fixes (fixed point), so most files only need
1-2 passes. There is no arbitrary pass cap: termination is driven by content stabilization,
not by counting passes.

**Cycle detection.** If a pass reads content whose hash matches a previously-seen pass, the
engine concludes a rule's `Fix()` re-triggers its own finding (or two rules ping-pong the
same content). Rather than loop forever or silently truncate, it emits a `style.fix-loop`
error finding naming the last rule applied before the cycle was detected and stops further
fix passes for that file. Findings already accumulated from earlier passes are still
returned alongside the `style.fix-loop` finding. This is a hard error (severity: error) —
the affected file should be inspected and the offending rule reported as a bug.

When invoked via `terratidy fmt --all` (fix mode), a final HCL format pass runs after all style
fixes have been applied, to restore equal-sign alignment that style rewrites can disrupt. See
the [`--all` workflow](../user-guide/commands.md#terratidy-fmt) section of the commands reference
for details.

### Comment Preservation

Attribute ordering rules (like `for-each-count-first`, `tags-at-end`, `depends-on-order`) preserve leading
comments when reordering attributes. Comments that appear on lines immediately before an attribute stay
attached to that attribute after fixing:

```hcl
# Before fix
resource "aws_instance" "web" {
  ami = "ami-12345"
  # This comment describes the instance type
  instance_type = "t3.micro"
  for_each = var.instances
}

# After fix (comment stays with instance_type)
resource "aws_instance" "web" {
  for_each = var.instances
  ami = "ami-12345"
  # This comment describes the instance type
  instance_type = "t3.micro"
}

## Configuration

### TerraTidy Config

```yaml
engines:
  style:
    enabled: true
    rules:
      style.blank-line-between-blocks:
        enabled: true
        severity: warning
      style.block-label-case:
        enabled: true
        severity: warning
      style.for-each-count-first:
        enabled: true
      style.variable-order:
        enabled: true
        severity: info
```

## Disabling Rules

**Precedence** (highest to lowest):

1. Inline comments (`# terratidy:ignore:rule-name`)
2. File-level comments (`# terratidy:ignore-file:rule-name`)
3. Rule config (`engines.<engine>.rules`)
4. Profile engine settings
5. Base config engine settings

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
      style.block-label-case:
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
| `no-empty-blocks` | Info | No | Blocks should not be empty |
| `no-leading-trailing-blank-lines` | Info | Yes | No leading/trailing blank lines in blocks |
| `attribute-group-spacing` | Info | Yes | Blank lines between attribute groups |
| `block-label-case` | Warning | No | Block labels use snake_case |
| `variable-naming` | Warning | No | Variable names use snake_case |
| `output-naming` | Warning | No | Output names use snake_case |
| `local-naming` | Warning | No | Local value names use snake_case |
| `for-each-count-first` | Warning | Yes | for_each/count as first attribute |
| `lifecycle-at-end` | Warning | Yes | lifecycle block at end |
| `tags-at-end` | Warning | Yes | tags near end of block |
| `depends-on-order` | Warning | Yes | depends_on at end of block |
| `source-version-grouped` | Warning | Yes | source and version together |
| `variable-order` | Info | Yes | Variable attribute ordering |
| `output-order` | Info | Yes | Output attribute ordering |
| `terraform-block-first` | Warning | Yes | terraform block first in file |
| `provider-block-order` | Warning | Yes | provider after terraform, before resources |

### Disabled by Default (Opt-in)

#### File Organization Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `variables-in-file` | Info | No | Variables in variables.tf |
| `outputs-in-file` | Info | No | Outputs in outputs.tf |
| `providers-in-file` | Info | No | Providers in providers.tf/versions.tf |
| `scoped-file-organization` | Info | No | Resources in scoped files (network.tf, etc.) |
| `terraform-files-structure` | Info | No | Standard file structure conventions |

#### Advanced Naming Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `resource-name-matches-type` | Info | No | Resource names should be descriptive |
| `output-prefix` | Info | No | Outputs follow prefix/suffix patterns |
| `module-name-convention` | Info | No | Module names follow conventions |

#### Block Organization Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `meta-arguments-order` | Info | Yes | Meta-arguments in consistent order |
| `lifecycle-attribute-order` | Info | Yes | Lifecycle attributes in order |
| `nested-block-order` | Info | No | Nested blocks in consistent order |
| `one-line-attribute-spacing` | Info | No | Consistent single vs multi-line blocks |

#### Comment and Format Rules

| Rule | Severity | Fixable | Description |
|------|----------|---------|-------------|
| `comment-syntax` | Info | Yes | Use # instead of // for comments |
| `no-trailing-whitespace` | Info | Yes | No trailing whitespace on lines |
| `consistent-quotes` | Warning | No | Use double quotes (Terraform standard) |
| `no-consecutive-blank-lines` | Info | Yes | No more than one consecutive blank line |
