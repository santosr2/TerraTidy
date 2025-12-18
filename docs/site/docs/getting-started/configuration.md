# Configuration

TerraTidy is configured using a `.terratidy.yaml` file in your project root.

## Configuration File

### Basic Structure

```yaml
version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

severity_threshold: warning
fail_fast: false
parallel: true
```

### Engine Configuration

Each engine can be enabled/disabled and configured:

```yaml
engines:
  fmt:
    enabled: true
    config:
      # fmt-specific options

  style:
    enabled: true
    config:
      # style-specific options

  lint:
    enabled: true
    config:
      use_tflint: false
      tflint_config: .tflint.hcl

  policy:
    enabled: true
    config:
      policy_dirs:
        - ./policies
```

## Environment Variables

Configuration values can use environment variables:

```yaml
engines:
  policy:
    config:
      # Simple variable
      api_key: ${API_KEY}

      # With default value
      region: ${AWS_REGION:-us-east-1}
```

## Profiles

Define different configuration profiles for different contexts:

```yaml
profiles:
  ci:
    description: "CI/CD strict checks"
    engines:
      fmt: { enabled: true }
      style: { enabled: true }
      lint: { enabled: true }
      policy: { enabled: true }

  development:
    description: "Fast development checks"
    engines:
      fmt: { enabled: true }
      style: { enabled: true }
      lint: { enabled: false }
      policy: { enabled: false }
```

Use a profile:

```bash
terratidy check --profile ci
```

### Profile Inheritance

Profiles can inherit from other profiles:

```yaml
profiles:
  base:
    engines:
      fmt: { enabled: true }
      style: { enabled: true }

  strict:
    inherits: base
    engines:
      lint: { enabled: true }
      policy: { enabled: true }
```

### Disabling Inherited Engines

Use `disabled_engines` to turn off engines from a parent profile:

```yaml
profiles:
  minimal:
    inherits: base
    disabled_engines:
      - lint
      - policy
```

## Rule Overrides

Override specific rule configurations:

```yaml
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: false

    lint.terraform-required-version:
      severity: error
      config:
        min_version: "1.5.0"
```

## Custom Rules

Define custom rules:

```yaml
custom_rules:
  my-org.naming-convention:
    enabled: true
    severity: warning
    config:
      pattern: "^(dev|staging|prod)_.*"
```

## Plugins

Enable and configure plugins:

```yaml
plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
    - ./plugins
```

## Configuration Imports

Split configuration across multiple files:

```yaml
version: 1

imports:
  - ./config/base.yaml
  - ./config/rules/*.yaml

engines:
  # local overrides
```

## Full Example

```yaml
version: 1

imports:
  - ./terratidy-rules.yaml

engines:
  fmt:
    enabled: true
  style:
    enabled: true
    config:
      block_label_case: snake_case
  lint:
    enabled: true
    config:
      use_tflint: true
      tflint_config: .tflint.hcl
  policy:
    enabled: true
    config:
      policy_dirs:
        - ./policies

severity_threshold: warning
fail_fast: false
parallel: true

profiles:
  ci:
    description: "Strict CI checks"
    engines:
      fmt: { enabled: true }
      style: { enabled: true }
      lint: { enabled: true }
      policy: { enabled: true }

  dev:
    description: "Fast dev checks"
    engines:
      fmt: { enabled: true }
      style: { enabled: true }

overrides:
  rules:
    lint.terraform-required-providers:
      severity: error

plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
```

## Command Line Overrides

Configuration can be overridden via command line:

```bash
terratidy check \
  --config custom.yaml \
  --profile ci \
  --severity-threshold error \
  --format json
```
