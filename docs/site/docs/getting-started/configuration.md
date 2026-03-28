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

## Configuration Precedence

Settings are resolved in this order (highest priority first):

1. **CLI flags** (`--config`, `--profile`, `--severity-threshold`, etc.)
2. **Environment variables** (`TERRATIDY_PROFILE`)
3. **Config file** (`.terratidy.yaml`)
4. **Defaults** (fmt/style/lint enabled, policy disabled, severity=warning)

## Environment Variables

Configuration values can use environment variables with three syntaxes:

| Syntax              | Behavior                                      |
| ------------------- | --------------------------------------------- |
| `${VAR}`            | Substitutes the value; empty string if unset  |
| `${VAR:-default}`   | Uses `default` if `VAR` is unset              |
| `${VAR:?error}`     | Required variable (empty string if unset)     |

```yaml
engines:
  policy:
    config:
      # Simple variable
      api_key: ${API_KEY}

      # With default value
      region: ${AWS_REGION:-us-east-1}

      # Required variable
      account_id: ${AWS_ACCOUNT_ID:?must be set}
```

Select a profile via environment variable:

```bash
export TERRATIDY_PROFILE=ci
terratidy check  # Uses the "ci" profile
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

## Global Settings

### fail_fast

When enabled, stops processing after the first engine that reports error-severity findings.
Only triggers on `error` severity, not warnings or info. Only applies to sequential execution
(not `--parallel`).

```yaml
fail_fast: true  # Stop after first engine with errors
```

### Cache

TerraTidy caches parsed HCL files to avoid redundant reads. The cache is managed automatically
and rarely needs configuration.

| Option     | Type     | Default | Description                          |
| ---------- | -------- | ------- | ------------------------------------ |
| `MaxAge`   | duration | 5m      | Maximum age of cache entries         |
| `MaxSize`  | int      | 1000    | Maximum number of entries (LRU)      |
| `Disabled` | bool     | false   | Disable caching entirely             |

Cache is invalidated when a file's modification time changes.

### Lint Engine

The lint engine integrates with TFLint. Configure it under `engines.lint.config`:

```yaml
engines:
  lint:
    enabled: true
    config:
      config_file: .tflint.hcl    # Path to TFLint config (default: .tflint.hcl)
      plugins:                     # TFLint plugins to enable
        - aws
        - terraform
```

If TFLint is not installed, the engine falls back to built-in lint rules.

## File Discovery

### Supported File Types

TerraTidy processes files with these extensions:

- `.tf` - Terraform configuration files
- `.hcl` - HCL configuration files
- `.tfvars` - Terraform variable files

All three types are handled by all engines (fmt, style, lint, policy) and supported by
the LSP, dev watch mode, and `--changed` flag.

### Skipped Directories

These directories are automatically skipped during file discovery:

- `node_modules/` - npm dependencies
- `vendor/` - Go dependencies
- `.terraform/` - Terraform provider cache
- `.terragrunt-cache/` - Terragrunt cache
- `__pycache__/` - Python cache
- Hidden directories (starting with `.`) except the current directory

## Command Line Overrides

Configuration can be overridden via command line:

```bash
terratidy check \
  --config custom.yaml \
  --profile ci \
  --severity-threshold error \
  --format json
```
