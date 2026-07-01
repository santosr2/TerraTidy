# YAML Rule Examples

Declarative rules for checking Terraform/HCL files without writing Go code.

## Examples

| File | Description |
|------|-------------|
| `require-description.yaml` | Require description on resources |
| `require-variable-description.yaml` | Require description on variables (block_types) |
| `no-deprecated-s3-args.yaml` | Forbid deprecated S3 arguments (forbidden_attributes) |
| `bucket-naming-convention.yaml` | Enforce bucket naming pattern (attribute_patterns) |
| `s3-best-practices.yaml` | Combined example using all features |

## Usage

Place YAML files in your plugins directory:

```bash
cp *.yaml ~/.terratidy/plugins/
```

## YAML Rule Fields

| Field        | Required | Description                                |
| ------------ | -------- | ------------------------------------------ |
| `name`       | Yes      | Unique rule identifier                     |
| `description`| Yes      | What the rule checks                       |
| `severity`   | No       | `info`, `warning`, or `error` (default: warning) |
| `enabled`    | No       | Enable/disable the rule (default: true)    |
| `message`    | No       | Custom message when rule triggers          |
| `tags`       | No       | Tags for filtering and grouping            |
| `patterns`   | Yes      | Match criteria (see below)                 |

### Patterns

| Field | Description |
|-------|-------------|
| `block_types` | HCL block types to check: `resource`, `variable`, `data`, `output`, `locals`, `module` (empty = all) |
| `resource_types` | Resource types to match (empty = all) |
| `required_attributes` | Attributes that must be present |
| `forbidden_attributes` | Attributes that must NOT be present |
| `attribute_patterns` | Regex validation for attribute values |

### Attribute Patterns

Each entry in `attribute_patterns` has:

| Field | Required | Description |
|-------|----------|-------------|
| `attribute` | Yes | Attribute name to validate |
| `pattern` | Yes | Regex pattern the value must match |
| `message` | No | Custom message when pattern fails |

## Plugin Integrity

The `.terratidy-plugins.sha256` manifest is committed and verified at load time. If you edit any of the YAML rule files in this directory, regenerate the manifest from the repo root:

```bash
cd examples/yaml-rule && sha256sum *.yaml > .terratidy-plugins.sha256
```

A stale manifest currently produces a `verification failed` warning and the rule loads anyway; future releases will treat the mismatch as a hard error.

## See Also

- [Plugins Documentation](../../docs/site/docs/development/plugins.md) for complete reference
