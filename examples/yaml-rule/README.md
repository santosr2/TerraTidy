# YAML Rule Example

A declarative rule that checks resources for a `description` attribute.

## Usage

Place the YAML file in your plugins directory:

```bash
cp require-description.yaml ~/.terratidy/plugins/
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

- `resource_types` - List of resource types to check (empty = all)
- `required_attributes` - Attributes that must be present
