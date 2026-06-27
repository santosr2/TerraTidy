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
recursive: true
```

### Engine Configuration

Each engine can be enabled/disabled and configured:

```yaml
engines:
  fmt:
    enabled: true
    check: false   # Report formatting issues without modifying files
    diff: false    # Show diff of formatting changes

  style:
    enabled: true
    fix: false     # Auto-fix mode
    diff: false    # Show diff of style fixes
    rules:         # Rule configuration goes under each engine
      style.block-label-case:
        enabled: true
        severity: warning

  lint:
    enabled: true
    config_file: .tflint.hcl  # Path to TFLint config
    args:                     # Extra arguments forwarded to the TFLint subprocess
      - --force

  policy:
    enabled: true
    policy_dirs:
      - ./policies
    policy_files:             # Individual policy files (in addition to policy_dirs)
      - ./custom-policy.rego
    data_files:               # JSON/YAML data files passed to policy evaluation
      - ./policy-data.json
```

### Engine Options Reference

| Engine   | Option         | Type     | Default | Purpose |
| -------- | -------------- | -------- | ------- | ------- |
| `fmt`    | `enabled`      | bool     | `true`  | Enable the format engine |
| `fmt`    | `check`        | bool     | `false` | Report formatting issues without rewriting files; mirrored by the `--check` CLI flag |
| `fmt`    | `diff`         | bool     | `false` | Print a unified diff of the format changes that would be applied |
| `style`  | `enabled`      | bool     | `true`  | Enable the style engine |
| `style`  | `fix`          | bool     | `false` | Apply auto-fixes for rules that implement `sdk.Fixer` |
| `style`  | `diff`         | bool     | `false` | Print a unified diff of the fixes that would be applied |
| `lint`   | `enabled`      | bool     | `true`  | Enable the lint engine ([engine docs](../user-guide/engines/lint.md)) |
| `lint`   | `use_tflint`   | bool     | `false` | Invoke TFLint as a subprocess in addition to the built-in rules |
| `lint`   | `tflint_path`  | string   | `""`    | Custom TFLint binary path (defaults to `tflint` on `PATH`) |
| `lint`   | `fallback_builtin` | bool | `false` | Fall back to built-in rules when TFLint is unavailable (recommended: `true`) |
| `lint`   | `config_file`  | string   | `""`    | Path to a TFLint config file (`.tflint.hcl`) |
| `lint`   | `plugins`      | []string | `[]`    | TFLint plugins to enable when `use_tflint: true` |
| `lint`   | `args`         | []string | `[]`    | Extra arguments forwarded to the TFLint subprocess |
| `policy` | `enabled`      | bool     | `false` | Enable the policy engine ([engine docs](../user-guide/engines/policy.md)) |
| `policy` | `policy_dirs`  | []string | `[]`    | Directories containing Rego policy files |
| `policy` | `policy_files` | []string | `[]`    | Individual Rego files to evaluate (in addition to `policy_dirs`) |
| `policy` | `data_files`   | []string | `[]`    | JSON/YAML data files passed to policy evaluation |

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
| `${VAR:?error}`     | Fails with error message if `VAR` is unset    |

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

## Rule Configuration

Configure rules under each engine's `rules` block:

```yaml
engines:
  style:
    rules:
      # Disable a rule
      style.blank-line-between-blocks:
        enabled: false

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

  lint:
    rules:
      # Change severity
      lint.terraform-required-version:
        enabled: true
        severity: error
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
  tags:                   # Only load rules matching these tags (empty = load all)
    - aws
    - production
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

### Plugin Tag Filtering

Set `plugins.tags` to load only the plugin rules tagged with one or more of the
listed values. An empty list (the default) loads every rule. Rules that do not
expose tags are skipped while a filter is active.

```yaml
plugins:
  enabled: true
  directories:
    - ./plugins
  tags:
    - aws        # Load rules tagged "aws"
    - security   # ...or "security"
```

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
recursive: true

engines:
  fmt:
    enabled: true
  style:
    enabled: true
    rules:
      style.block-label-case:
        enabled: true
        severity: warning
  lint:
    enabled: true
    config_file: .tflint.hcl
    rules:
      lint.terraform-required-providers:
        severity: error
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

Can be overridden with `--parallel` (force on) or `--no-parallel` (force off) CLI flags.
`--no-parallel` takes precedence over both `--parallel` and the config setting.

### fail_fast

When enabled, stops processing after the first engine that reports error-severity findings.
Only triggers on `error` severity, not warnings or info. Only applies to sequential execution
(not `--parallel`).

```yaml
fail_fast: true  # Stop after first engine with errors
```

### recursive

Enable or disable recursive directory scanning. Default: `true`.

```yaml
recursive: true   # Scan all subdirectories (default)
recursive: false  # Only scan files directly in specified directories
```

When `recursive: false`, TerraTidy only processes files directly in the specified directories,
not in their subdirectories. This is useful when you want to check specific directories without
descending into nested modules.

Can be overridden with `--no-recurse` CLI flag (the CLI flag takes precedence).

**Example:**

```bash
# With recursive: true (default), scans modules/, modules/vpc/, modules/rds/, etc.
terratidy check modules/

# With recursive: false, scans only modules/ (not modules/vpc/, etc.)
terratidy check modules/
```

### Output

Configure output formatting options.

```yaml
output:
  absolute_paths: false  # Use absolute file paths instead of relative (default: false)
```

| Option          | Type | Default | Description                                |
| --------------- | ---- | ------- | ------------------------------------------ |
| `absolute_paths` | bool | `false` | Output absolute paths instead of relative |

By default, TerraTidy outputs relative paths (relative to the current working directory)
which are more readable in CI logs and editor integrations. Set `absolute_paths: true`
when you need full paths for tooling integration.

Can be overridden with `--absolute-paths` CLI flag.

### Cache

TerraTidy caches parsed HCL files to avoid redundant reads. The cache is managed automatically
and rarely needs configuration.

```yaml
cache:
  max_age: 5m     # Maximum age of cache entries (default: 5m)
  max_size: 1000  # Maximum number of entries, LRU eviction (default: 1000)
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
    use_tflint: false           # Enable TFLint integration (default: false)
    tflint_path: ""             # Custom path to TFLint binary (optional)
    fallback_builtin: true      # Use built-in rules if TFLint unavailable (default: false; recommended on)
    config_file: .tflint.hcl    # Path to TFLint config (optional)
    plugins:                     # TFLint plugins to enable (optional)
      - aws
      - terraform
    args:                        # Extra arguments forwarded to the TFLint subprocess
      - --force
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
