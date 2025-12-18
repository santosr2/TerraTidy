# Style Rules

Complete reference for all style rules enforced by the style engine.

## Naming Rules

### resource-naming

Ensures resource names follow the configured naming convention.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

**Configuration:**

```yaml
engines:
  style:
    rules:
      resource-naming:
        enabled: true
        severity: warning
        config:
          convention: snake_case  # snake_case, kebab-case, camelCase
```

**Example:**

```hcl
# Bad
resource "aws_instance" "MyServer" { }

# Good
resource "aws_instance" "my_server" { }
```

### variable-naming

Ensures variable names follow the configured naming convention.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

**Example:**

```hcl
# Bad
variable "instanceType" { }

# Good
variable "instance_type" { }
```

### output-naming

Ensures output names follow the configured naming convention.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

### module-naming

Ensures module call names follow the configured naming convention.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

### local-naming

Ensures local value names follow the configured naming convention.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | Yes |
| Default | Enabled |

## Ordering Rules

### attribute-order

Ensures attributes within blocks follow a consistent order.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | Yes |
| Default | Enabled |

**Default Order:**

1. `count` / `for_each`
2. `provider`
3. Required attributes
4. Optional attributes
5. Nested blocks
6. `depends_on`
7. `lifecycle`

**Configuration:**

```yaml
engines:
  style:
    rules:
      attribute-order:
        config:
          order:
            - count
            - for_each
            - provider
            - depends_on
            - lifecycle
```

### block-order

Ensures blocks within a file follow a consistent order.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | Yes |
| Default | Disabled |

**Default Order:**

1. `terraform`
2. `provider`
3. `variable`
4. `locals`
5. `data`
6. `resource`
7. `module`
8. `output`

## File Organization Rules

### variables-in-file

Variables should be defined in `variables.tf`.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | No |
| Default | Disabled |

### outputs-in-file

Outputs should be defined in `outputs.tf`.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | No |
| Default | Disabled |

### providers-in-file

Provider configurations should be in `providers.tf` or `versions.tf`.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | No |
| Default | Disabled |

## Documentation Rules

### variable-description

Variables should have a description.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | No |
| Default | Enabled |

**Example:**

```hcl
# Bad
variable "instance_type" {
  type = string
}

# Good
variable "instance_type" {
  description = "The EC2 instance type"
  type        = string
}
```

### output-description

Outputs should have a description.

| Property | Value |
|----------|-------|
| Default Severity | Info |
| Fixable | No |
| Default | Enabled |

### variable-type

Variables should have an explicit type.

| Property | Value |
|----------|-------|
| Default Severity | Warning |
| Fixable | No |
| Default | Enabled |

## Disabling Rules

### Inline

```hcl
# terratidy:ignore:resource-naming
resource "aws_instance" "MyServer" { }
```

### Configuration

```yaml
engines:
  style:
    rules:
      resource-naming:
        enabled: false
```

### File-level

```hcl
# terratidy:ignore-file:resource-naming
```
