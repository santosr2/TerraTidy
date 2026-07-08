# Commands Reference

Complete reference for all TerraTidy commands.

## Global Flags

These flags are available for all commands:

| Flag                   | Description                                                                                              |
| ---------------------- | -------------------------------------------------------------------------------------------------------- |
| `--config`, `-c`       | Path to configuration file (default: `.terratidy.yaml`)                                                  |
| `--profile`, `-p`      | Configuration profile to use                                                                             |
| `--format`, `-f`       | Output format: `text`, `table`, `json`, `json-compact`, `sarif`, `html`, `junit`, `markdown`, `github`   |
| `--changed`            | Only check files changed in git                                                                          |
| `--no-recurse`         | Disable recursive directory traversal (scan only specified directories, not subdirectories)             |
| `--exclude`            | Glob patterns to exclude (repeatable or comma-separated)                                                 |
| `--color`              | Enable colored output (default: true)                                                                    |
| `--absolute-paths`     | Output absolute file paths instead of relative                                                           |
| `--severity-threshold` | Minimum severity: `info`, `warning`, `error`                                                             |

## Exit Codes

All commands use consistent exit codes for scripting and CI/CD:

| Code | Meaning |
|------|---------|
| `0`  | Success, no issues found |
| `1`  | Findings found (formatting issues, style violations, lint errors, policy failures) |
| `2`  | Configuration error (invalid config file, missing required config, plugin loading failure) |
| `3`  | Internal error (engine failure, filesystem errors, unexpected issues) |

The `--check` flag on `fmt` and `style` exits with code 1 if any issues are found (findings at or above `severity_threshold`).
The `check` command exits with code 1 when error-severity findings are present.

**Scripting example:**

```bash
terratidy check ./modules
case $? in
  0) echo "All checks passed" ;;
  1) echo "Found issues to fix" ;;
  2) echo "Configuration problem - check .terratidy.yaml" ;;
  3) echo "Internal error - please report a bug" ;;
esac
```

## VCS Integration (`--changed`)

The `--changed` flag uses git to detect modified files:

```bash
# Only check files changed since the default branch
terratidy check --changed

# Works with any command
terratidy fmt --changed
terratidy style --changed
```

**How it works:**

1. Detects the default branch (`main` or `master`) via `git symbolic-ref refs/remotes/origin/HEAD`
2. Finds the merge-base between HEAD and the default branch
3. Combines staged, unstaged, and untracked `.tf`/`.hcl`/`.tfvars` files
4. Deduplicates and converts to absolute paths

This means `--changed` catches all local modifications, whether staged, unstaged, or new files.

## Non-Recursive Scanning (`--no-recurse`)

By default, TerraTidy recursively scans all subdirectories. Use `--no-recurse` to scan only the specified directories:

```bash
# Scan only the current directory (not subdirectories)
terratidy check --no-recurse

# Scan only the modules/ directory (not modules/vpc/, modules/rds/, etc.)
terratidy check --no-recurse modules/

# Scan multiple specific directories without recursion
terratidy check --no-recurse . modules/ environments/
```

**With `--changed`:**

When combined with `--changed`, only changed files directly in the specified directories are processed:

```bash
# Only check changed files in the current directory (not subdirs)
terratidy check --changed --no-recurse
```

**Via config:**

You can also set this in `.terratidy.yaml`:

```yaml
recursive: false
```

The CLI flag `--no-recurse` takes precedence over the config file setting.

## Glob Exclusions (`--exclude`)

Exclude files or directories from processing using glob patterns:

```bash
# Exclude generated files
terratidy check --exclude "**/*.generated.tf"

# Exclude test fixtures
terratidy check --exclude "test/**"

# Exclude multiple patterns (comma-separated)
terratidy check --exclude "**/*.generated.tf,test/**,vendor/**"

# Exclude multiple patterns (repeated flag)
terratidy check --exclude "**/*.generated.tf" --exclude "test/**"
```

**Pattern syntax:**

- `*` - matches any sequence of characters (except `/`)
- `**` - matches zero or more directory levels
- `?` - matches any single character

**Examples:**

| Pattern | Matches |
|---------|---------|
| `*.tf` | `main.tf`, `variables.tf` |
| `**/*.tf` | All `.tf` files in any directory |
| `modules/*/` | `modules/vpc/`, `modules/rds/` |
| `test/**` | Everything under `test/` |

**With config file:**

CLI `--exclude` patterns are combined with config file `exclude` patterns:

```yaml
# .terratidy.yaml
exclude:
  - ".terraform/**"
  - "vendor/**"
```

```bash
# This combines config patterns + CLI pattern
terratidy check --exclude "test/**"
```

## Absolute Paths (`--absolute-paths`)

By default, TerraTidy outputs relative file paths for readability. Use `--absolute-paths` when you need full paths:

```bash
# Default: relative paths
terratidy check
# Output: modules/vpc/main.tf:15: ...

# Absolute paths
terratidy check --absolute-paths
# Output: /home/user/project/modules/vpc/main.tf:15: ...
```

**When to use:**

- Tooling that requires absolute paths
- CI systems that run from different working directories
- Editor integrations that need full paths for navigation

**Via config:**

```yaml
output:
  absolute_paths: true
```

The CLI flag `--absolute-paths` takes precedence over the config file setting.

## Inline Rule Suppression

Suppress specific rules on a per-line or per-file basis using comment annotations.
Works with style, lint, and policy engines.

```hcl
# Suppress a rule on the next block
# terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }

# Suppress a rule inline (same line as code)
resource "aws_s3_bucket" "Test" { } # terratidy:ignore:style.block-label-case

# Suppress a rule for the entire file
# terratidy:ignore-file:lint.terraform-required-version

# Suppress all rules for an engine using wildcards
# terratidy:ignore-file:policy.*
```

Both `#` and `//` comment styles are supported.

See [Style Rules - Disabling Rules](../rules/style-rules.md#disabling-rules) for configuration-based disabling.

## terratidy check

Run all enabled checks. This is the recommended command for CI/CD.

```bash
terratidy check [paths...] [flags]
```

**Flags:**

| Flag            | Description                                                 |
| --------------- | ----------------------------------------------------------- |
| `--parallel`    | Run engines in parallel                                     |
| `--no-parallel` | Force sequential execution (overrides `--parallel` and config) |
| `--skip-fmt`    | Skip formatting checks                                      |
| `--skip-style`  | Skip style checks                                           |
| `--skip-lint`   | Skip linting checks                                         |
| `--skip-policy` | Skip policy checks                                          |

When `fail_fast` is enabled in config, the check stops after the first engine that reports errors.
`fail_fast` only takes effect in sequential mode; combine it with `--no-parallel` (or
`parallel: false` in config) to make it active.

**Examples:**

```bash
# Check all files
terratidy check

# Check specific directory
terratidy check ./modules/

# Check with CI profile
terratidy check --profile ci

# Output as JSON
terratidy check --format json

# Run in parallel for faster execution (default)
terratidy check --parallel

# Force sequential execution (e.g. to combine with fail_fast: true)
terratidy check --no-parallel

# Skip policy checks
terratidy check --skip-policy
```

## terratidy fmt

Format Terraform and Terragrunt files.

```bash
terratidy fmt [paths...] [flags]
```

**Flags:**

| Flag      | Description |
| --------- | ----------- |
| `--check` | Check formatting without modifying |
| `--diff`  | Show diff of changes |
| `--all`   | Also run style engine alongside fmt (fixes in default mode, checks with `--check`) |

**`--diff` behavior:**

When `--diff` is set, each file that needs formatting includes a unified diff in its output showing the exact changes. Works with or without `--check`:

- `--diff` alone: formats the file and shows the diff
- `--check --diff`: shows the diff without modifying files

**`--all` workflow:**

When `fmt --all` applies style fixes (i.e. fix mode, not `--check`), style rewrites can disrupt
`hclwrite`'s equal-sign alignment. After the style pass reports `Fixed N style issue(s)`, fmt
prints `Re-aligning attributes after style fixes...` and re-runs the format engine over the
same files to restore alignment. This second pass only runs when style fixes were applied;
`fmt --all --check` and `fmt --all` runs that produced no style fixes skip it.

**Structured output (`--format`):**

`fmt` honours the global `--format` flag (`table`, `json`, `json-compact`, `sarif`, `html`,
`junit`, `markdown`, `github`). When `--format` is anything other than `text` (the default),
fmt suppresses its human-readable banners (`Formatting N file(s)...`, per-file `[!]`/`[+]`
lines, the `Re-aligning attributes...` notice, and the trailing summary) and emits all
findings — including `fmt.needs-formatting` and `fmt.formatted` — through the shared
formatter at the end. The re-alignment pass under `--all` still runs; only the progress
text is suppressed. This keeps the structured payload clean for downstream tools.

**Examples:**

```bash
# Format all files
terratidy fmt

# Check formatting only
terratidy fmt --check

# Show what would change
terratidy fmt --check --diff

# Format and apply style fixes (prints "Re-aligning..." after style fixes apply)
terratidy fmt --all

# Emit findings as JSON for a CI pipeline (no banners, no progress lines)
terratidy fmt --check --format json

# JSON output also works with --all and --diff (diff text lands in finding messages)
terratidy fmt --check --all --diff --format json
```

**Comparison with style --fix:**

| Command                 | HCL Formatting | Style Fixes |
|-------------------------|----------------|-------------|
| `terratidy fmt`         | ✓              |             |
| `terratidy style --fix` |                | ✓           |
| `terratidy fmt --all`   | ✓              | ✓           |

## terratidy style

Check and fix style issues.

```bash
terratidy style [paths...] [flags]
```

**Flags:**

| Flag      | Description                                               |
| --------- | --------------------------------------------------------- |
| `--fix`   | Auto-fix style issues                                     |
| `--check` | Check only, exit with error if issues found               |
| `--diff`  | Show diff of style changes (preview mode or with `--fix`) |

**`--diff` behavior:**

The `--diff` flag works in two modes:

- **Preview mode** (`--diff` alone): Shows what would change without modifying files. The engine
  applies fixes in memory and generates a unified diff, then restores the original file.
- **With `--fix`** (`--fix --diff`): Shows the diff and applies the changes.

```bash
# Preview what would change (file not modified)
terratidy style --diff

# Apply fixes and show diff
terratidy style --fix --diff
```

**Examples:**

```bash
# Check style
terratidy style

# Fix style issues
terratidy style --fix

# Preview what would change (without modifying files)
terratidy style --diff

# Apply fixes and show diff
terratidy style --fix --diff

# Check only (exit with error if issues found)
terratidy style --check
```

## terratidy lint

Run linting checks.

```bash
terratidy lint [paths...] [flags]
```

**Flags:**

| Flag              | Description                                  |
| ----------------- | -------------------------------------------- |
| `--tflint-config` | Path to TFLint config (default: .tflint.hcl) |
| `--plugin`        | Plugins to enable (aws, google, azurerm)     |
| `--rule`          | Specific rules to enable                     |

**`--rule`** enables specific rules by name. Works for both built-in lint rules and TFLint
rules (when TFLint is installed). Multiple `--rule` flags can be passed. Each enabled rule
defaults to warning severity.

**Examples:**

```bash
# Run linting
terratidy lint

# Enable specific rules
terratidy lint --rule terraform-required-version --rule terraform-required-providers

# Use a specific TFLint config
terratidy lint --tflint-config .tflint-strict.hcl

# Enable AWS plugin
terratidy lint --plugin aws
```

## terratidy policy

Run OPA/Rego policy checks.

```bash
terratidy policy [paths...] [flags]
```

**Flags:**

| Flag            | Description                            |
| --------------- | -------------------------------------- |
| `--policy-dirs` | Directories containing .rego files     |
| `--policy-file` | Individual Rego policy files           |
| `--show-input`  | Show input JSON for debugging policies |

**`--show-input`** outputs the JSON representation of the Terraform configuration that gets
passed to OPA for evaluation. Useful for debugging why a policy rule does or doesn't match.

**Examples:**

```bash
# Run policy checks
terratidy policy

# Run with custom policies
terratidy policy --policy-dirs ./policies

# Debug: see what OPA receives as input
terratidy policy --show-input
```

## terratidy fix

Auto-fix all fixable issues.

```bash
terratidy fix [paths...] [flags]
```

**Examples:**

```bash
# Fix all issues
terratidy fix

# Fix only changed files
terratidy fix --changed
```

## terratidy init

Initialize TerraTidy configuration.

```bash
terratidy init [flags]
```

**Flags:**

| Flag            | Description                         |
| --------------- | ----------------------------------- |
| `--interactive`, `-i` | Interactive configuration setup     |
| `--force`             | Overwrite existing configuration    |
| `--split`             | Create modular split configuration  |
| `--monorepo`    | Set up for monorepo                 |

**`--split`** creates a modular `.terratidy/` directory with separate config files per engine:

```text
.terratidy.yaml         # Main config with imports
.terratidy/fmt.yaml
.terratidy/style.yaml
.terratidy/lint.yaml
.terratidy/policy.yaml
```

**`--monorepo`** generates config tailored for monorepos with a central `./policies` directory
and two profiles: `ci` (strict) and `development` (relaxed).

**Examples:**

```bash
# Initialize with defaults
terratidy init

# Interactive setup
terratidy init --interactive

# Create split configuration
terratidy init --split

# Monorepo configuration
terratidy init --monorepo
```

## terratidy init-rule

Initialize a new custom rule.

```bash
terratidy init-rule [flags]
```

**Flags:**

| Flag       | Description                                        |
| ---------- | -------------------------------------------------- |
| `--name`   | Rule name (required)                               |
| `--type`   | Rule type: `go`, `rego`, `yaml` (default: `rego`)  |
| `--output` | Output directory (default: `.`)                    |

**Examples:**

```bash
# Create a new Go rule
terratidy init-rule --name my-custom-rule --type go

# Create a Rego policy
terratidy init-rule --name require-encryption --type rego

# Create a YAML rule in a specific directory
terratidy init-rule --name my-rule --type yaml --output ./rules
```

## terratidy test-rule

Test a specific rule against fixture files.

```bash
terratidy test-rule [rule-path] [flags]
```

**Flags:**

| Flag         | Description                                      |
| ------------ | ------------------------------------------------ |
| `--fixtures` | Fixtures directory (default: `test_fixtures/`)   |
| `--expect`   | Expected findings file (YAML or JSON)            |

**Expected Findings Format:**

The `--expect` file uses YAML or JSON with this schema:

```yaml
findings:
  - rule: "policy-rule-name"        # Required; matched as suffix
    severity: "error"               # Optional; must match exactly
    message: "expected substring"   # Optional; substring match
```

Matching rules: `rule` is matched as a suffix (e.g., `"no-public-ssh"` matches
`"policy.no-public-ssh"`). `severity` must match exactly if specified. `message` is
a substring match if specified.

**Examples:**

```bash
# Test a Rego policy
terratidy test-rule ./policies/my-rule.rego

# Test with specific fixtures
terratidy test-rule ./policies/my-rule.rego --fixtures ./test_fixtures

# Test with expected findings
terratidy test-rule ./policies/my-rule.rego --expect ./expected.yaml
```

## terratidy rules

Manage and inspect TerraTidy rules.

```bash
terratidy rules [command]
```

**Subcommands:**

| Command | Description                    |
| ------- | ------------------------------ |
| `list`  | List all available rules       |
| `docs`  | Generate rule documentation    |

**Flags (list):**

| Flag              | Description                                     |
| ----------------- | ----------------------------------------------- |
| `--engine`        | Filter by engine: `style`, `lint`, `policy`     |
| `--verbose`, `-v` | Show detailed descriptions                      |

**Flags (docs):**

| Flag       | Description                                     |
| ---------- | ----------------------------------------------- |
| `--engine` | Filter by engine: `style`, `lint`, `policy`     |

**`rules list`** shows a table of all registered rules with name, severity, and description.
By default, descriptions are truncated; use `-v` for full text.

**`rules docs`** generates markdown documentation for all rules, suitable for piping to a file
or integrating into CI to verify rules are documented.

**Examples:**

```bash
# List all rules
terratidy rules list

# List rules for a specific engine
terratidy rules list --engine style

# List with verbose output
terratidy rules list --verbose

# Generate documentation
terratidy rules docs

# Generate docs for a specific engine
terratidy rules docs --engine style

# Save generated docs to a file
terratidy rules docs > rules-reference.md
```

## terratidy config

Configuration management.

```bash
terratidy config [command]
```

**Subcommands:**

| Command          | Description                            |
| ---------------- | -------------------------------------- |
| `show`           | Display current configuration          |
| `validate`       | Validate configuration file            |
| `split`          | Split configuration into modules       |
| `merge`          | Merge split configurations             |
| `init-profile`   | Initialize a new configuration profile |

**`config show`** displays the fully resolved configuration after all imports, profile merges,
and overrides are applied.

**`config validate`** checks for syntax errors, invalid values, and missing required fields.
Warns if no engines are enabled.

**`config split`** converts a single `.terratidy.yaml` into a modular `.terratidy/` directory
with separate files per engine.

**`config merge`** combines modular `.terratidy/*.yaml` files back into a single `.terratidy.yaml`.

**`config init-profile`** creates a new named profile in the config. Usage: `terratidy config init-profile ci`

**Flags (`show`):**

| Flag       | Description                                           |
| ---------- | ----------------------------------------------------- |
| `--output` | Serialization format: `yaml`, `json` (default: `yaml`) |

**Examples:**

```bash
# Show resolved config
terratidy config show

# Show as JSON
terratidy config show --output json

# Validate configuration
terratidy config validate

# Split config into modules
terratidy config split

# Merge modules back
terratidy config merge

# Create a new profile
terratidy config init-profile production
```

## terratidy plugins

Plugin management.

```bash
terratidy plugins [command]
```

**Subcommands:**

| Command | Description                  |
| ------- | ---------------------------- |
| `list`  | List installed plugins       |
| `info`  | Show plugin details          |
| `init`  | Create new plugin project    |

## terratidy lsp

Start the Language Server Protocol server.

```bash
terratidy lsp [flags]
```

Used by IDE extensions for real-time diagnostics.

**Flags:**

| Flag          | Description                                            |
| ------------- | ------------------------------------------------------ |
| `--log-level` | Log level: off, error, warn, info, debug (default: info) |
| `--log-file`  | Path to log file (default: stderr)                     |

## terratidy dev

Development mode with file watching. Watches for changes to `.rego`, `.tf`, `.hcl`, and `.tfvars`
files and re-runs checks automatically with a 500ms debounce delay.

```bash
terratidy dev [flags]
```

**Flags:**

| Flag       | Description                                      |
| ---------- | ------------------------------------------------ |
| `--watch`  | Directory to watch (default: `policies/`)        |
| `--target` | Target directory to check (default: `.`)         |

**Behavior:**

- Watches the `--watch` directory recursively for file changes (write and create events)
- Waits 500ms after the last change before re-running checks (debounce)
- Displays findings with severity summary and timestamp
- Continues until interrupted with Ctrl+C

**Examples:**

```bash
# Watch rules directory
terratidy dev

# Watch specific directory
terratidy dev --watch ./policies --target ./modules
```

## terratidy version

Show version information.

```bash
terratidy version [flags]
```

**Flags:**

| Flag      | Description                |
| --------- | -------------------------- |
| `--short` | Print only version number  |

The version command respects the global `--format` flag but only supports `text`, `json`, and `json-compact` formats.

**Examples:**

```bash
# Show full version info
terratidy version

# Show only version number
terratidy version --short

# Output as JSON (pretty-printed)
terratidy version --format json

# Output as compact JSON
terratidy version --format json-compact
```

**Output:**

```text
TerraTidy version 0.2.0-alpha.4
  Commit:      abc1234
  Build date:  2025-12-22
  Go version:  go1.26.4
  Platform:    darwin/arm64
```
