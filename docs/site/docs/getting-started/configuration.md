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

  style:
    enabled: true
    fix: false  # Auto-fix mode

  lint:
    enabled: true
    config_file: .tflint.hcl  # Path to TFLint config

  policy:
    enabled: true
    policy_dirs:
      - ./policies

# Rule overrides go here, NOT under engines.*.config
overrides:
  rules:
    style.block-label-case:
      enabled: true
      severity: warning
```

## Configuration Precedence

Settings are resolved in this order (highest priority first):

1. **CLI flags** (`--config`, `--profile`, `--severity-threshold`, etc.)
2. **Config file** (`.terratidy.yaml`)
3. **Defaults** (fmt/style/lint enabled, policy disabled, severity=warning)

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
    # Simple variable
    api_key: ${API_KEY}

    # With default value
    region: ${AWS_REGION:-us-east-1}

    # Required variable
    account_id: ${AWS_ACCOUNT_ID:?must be set}
```

Select a profile via CLI flag:

```bash
terratidy check --profile ci  # Uses the "ci" profile
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

Set `enabled: false` to turn off engines from a parent profile:

```yaml
profiles:
  minimal:
    inherits: base
    engines:
      lint: { enabled: false }
      policy: { enabled: false }
```

## Rule Overrides

Override specific rule configurations:

```yaml
overrides:
  rules:
    # Disable a rule
    style.blank-line-between-blocks:
      enabled: false

    # Change severity
    lint.terraform-required-version:
      severity: error

    # Style rule with options (note the nested 'options' key)
    style.variable-naming:
      enabled: true
      config:
        options:
          case: camelCase  # Options: snake_case, camelCase, kebab-case, PascalCase

    # Style rule with numeric options
    style.blank-line-between-blocks:
      enabled: true
      config:
        options:
          min_lines: 1
          max_lines: 2
```

## Plugins

Enable and configure plugins:

```yaml
plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
    - ./plugins
  verify_integrity: true  # Verify plugin checksums (default: true)
```

### Plugin Rule Configuration

Override specific plugin rule settings (enable/disable, severity):

```yaml
plugins:
  enabled: true
  directories:
    - ./plugins
  rules:
    require-description:
      enabled: true
      severity: error  # Override plugin's default severity
    deprecated-resource:
      enabled: false   # Disable this plugin rule
```

Plugin rules not listed in `plugins.rules` use their defaults (enabled with their defined severity).

### Plugin Integrity Verification

When `verify_integrity` is enabled (default), TerraTidy verifies Go plugin checksums
against a `.terratidy-plugins.sha256` manifest file in each plugin directory.

Create the manifest using `sha256sum`:

```bash
cd ~/.terratidy/plugins
sha256sum *.so > .terratidy-plugins.sha256
```

The manifest format is compatible with `sha256sum` output:

```text
e3b0c44298fc1c149afbf4c8996fb924...  myplugin.so
b94d27b9934d3e08a52e52d7da7dabfa...  anotherplugin.so
```

If verification fails or the manifest is missing, a warning is logged but the plugin
still loads (warn-only mode). Set `verify_integrity: false` to disable verification entirely.

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

Imported files can themselves contain `imports`, which are loaded recursively.
Circular imports (e.g., `a.yaml` imports `b.yaml` which imports `a.yaml`) are
detected and produce a clear error.

## Full Example

```yaml
version: 1

imports:
  - ./terratidy-rules.yaml

severity_threshold: warning
fail_fast: false
parallel: true

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
    config_file: .tflint.hcl
  policy:
    enabled: true
    policy_dirs:
      - ./policies

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

# Rule overrides - enable/disable rules, change severity
overrides:
  rules:
    style.block-label-case:
      enabled: true
      severity: warning
    lint.terraform-required-providers:
      severity: error

plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
  rules:
    require-description:
      enabled: true
      severity: warning
```

## Global Settings

### severity_threshold

Filters findings to only report those at or above the specified severity level.
Valid values: `info`, `warning`, `error`. Default: `warning`.

```yaml
severity_threshold: error  # Only report errors, hide warnings and info
```

| Value     | Shows                           |
| --------- | ------------------------------- |
| `info`    | All findings                    |
| `warning` | Warnings and errors (default)   |
| `error`   | Errors only                     |

Can be overridden with `--severity-threshold` CLI flag.

### parallel

Run engines in parallel for faster execution. Default: `true`.

```yaml
parallel: true   # Run engines concurrently
parallel: false  # Run engines sequentially (fmt -> style -> lint -> policy)
```

Parallel mode is faster but output order is non-deterministic. Sequential mode is useful
for debugging or when `fail_fast` is enabled (fail_fast only works in sequential mode).

Can be overridden with `--parallel` CLI flag.

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

```yaml
cache:
  max_age: 10m    # Maximum age of cache entries (default: 5m)
  max_size: 500   # Maximum number of entries, LRU eviction (default: 1000)
  disabled: false # Disable caching entirely (default: false)
```

| Option     | Type     | Default | Description                          |
| ---------- | -------- | ------- | ------------------------------------ |
| `max_age`  | duration | `5m`    | Maximum age of cache entries         |
| `max_size` | int      | `1000`  | Maximum number of entries (LRU)      |
| `disabled` | bool     | `false` | Disable caching entirely             |

Cache is invalidated when a file's modification time changes.

### Lint Engine

The lint engine provides 11 built-in AST rules and can optionally invoke TFLint as an
external subprocess for provider-specific checks:

```yaml
engines:
  lint:
    enabled: true
    config_file: .tflint.hcl    # Path to TFLint config (optional)
    plugins:                     # TFLint plugins to enable (optional)
      - aws
      - terraform
```

Built-in rules work without TFLint. If TFLint is installed and configured, provider-specific
rules are also available. TFLint is invoked as a subprocess, not embedded or linked.

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

### Exclude Patterns

Exclude files or directories from processing using glob patterns:

```yaml
exclude:
  - "**/*.generated.tf"      # Exclude generated files
  - "vendor/**"              # Exclude vendor directory
  - ".terraform/**"          # Exclude Terraform cache
  - "examples/legacy/**"     # Exclude specific directory
```

Patterns support:

- `*` - matches any sequence of characters (except `/`)
- `**` - matches zero or more directory levels
- `?` - matches any single character

Exclude patterns from config and CLI flag `--exclude` are combined.

```bash
# Exclude additional patterns via CLI
terratidy check --exclude "**/*.generated.tf,test/**"
```

Multiple patterns can be comma-separated or specified multiple times:

```bash
terratidy check --exclude "**/*.generated.tf" --exclude "test/**"
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
