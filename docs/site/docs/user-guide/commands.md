# Commands Reference

Complete reference for all TerraTidy commands.

## Global Flags

These flags are available for all commands:

| Flag                   | Description                                                                                              |
| ---------------------- | -------------------------------------------------------------------------------------------------------- |
| `--config`             | Path to configuration file (default: `.terratidy.yaml`)                                                  |
| `--profile`            | Configuration profile to use                                                                             |
| `--format`             | Output format: `text`, `json`, `json-compact`, `sarif`, `html`, `github`, `table`, `junit`, `markdown`   |
| `--paths`              | Paths to check (comma-separated)                                                                         |
| `--changed`            | Only check files changed in git                                                                          |
| `--color`              | Enable colored output (default: true)                                                                    |
| `--severity-threshold` | Minimum severity: `info`, `warning`, `error`                                                             |

## terratidy check

Run all enabled checks. This is the recommended command for CI/CD.

```bash
terratidy check [paths...] [flags]
```

**Flags:**

| Flag            | Description                     |
| --------------- | ------------------------------- |
| `--parallel`    | Run engines in parallel         |
| `-p`            | Short for --parallel            |
| `--skip-fmt`    | Skip formatting checks          |
| `--skip-style`  | Skip style checks               |
| `--skip-lint`   | Skip linting checks             |
| `--skip-policy` | Skip policy checks              |

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

# Run in parallel for faster execution
terratidy check --parallel

# Skip policy checks
terratidy check --skip-policy
```

## terratidy fmt

Format Terraform and Terragrunt files.

```bash
terratidy fmt [paths...] [flags]
```

**Flags:**

| Flag      | Description                                              |
| --------- | -------------------------------------------------------- |
| `--check` | Check formatting without modifying                       |
| `--diff`  | Show diff of changes                                     |
| `--all`   | Also apply style fixes (equivalent to fmt + style --fix) |

**Examples:**

```bash
# Format all files
terratidy fmt

# Check formatting only
terratidy fmt --check

# Format and apply style fixes
terratidy fmt --all
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

| Flag      | Description                                 |
| --------- | ------------------------------------------- |
| `--fix`   | Auto-fix style issues                       |
| `--check` | Check only, exit with error if issues found |
| `--diff`  | Show diff of style changes                  |

**Examples:**

```bash
# Check style
terratidy style

# Fix style issues
terratidy style --fix

# Check only (exit with error if issues found)
terratidy style --check
```

## terratidy lint

Run linting checks.

```bash
terratidy lint [paths...] [flags]
```

**Flags:**

| Flag            | Description                                  |
| --------------- | -------------------------------------------- |
| `--config-file` | Path to TFLint config (default: .tflint.hcl) |
| `--plugin`      | Plugins to enable (aws, google, azurerm)     |
| `--rule`        | Specific rules to enable                     |

**Examples:**

```bash
# Run linting
terratidy lint

# Enable specific rule
terratidy lint --rule terraform_required_version
```

## terratidy policy

Run OPA/Rego policy checks.

```bash
terratidy policy [paths...] [flags]
```

**Flags:**

| Flag            | Description                            |
| --------------- | -------------------------------------- |
| `--policy-dir`  | Directories containing .rego files     |
| `--policy-file` | Individual Rego policy files           |
| `--show-input`  | Show input JSON for debugging policies |

**Examples:**

```bash
# Run policy checks
terratidy policy

# Run with custom policies
terratidy policy --policy-dir ./policies

# Show input JSON for debugging
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
| `--interactive` | Interactive configuration setup     |
| `-i`            | Short for --interactive             |
| `--force`       | Overwrite existing configuration    |
| `-f`            | Short for --force                   |
| `--split`       | Create modular split configuration  |
| `--monorepo`    | Set up for monorepo                 |

**Examples:**

```bash
# Initialize with defaults
terratidy init

# Interactive setup
terratidy init --interactive

# Create split configuration
terratidy init --split
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
| `-v`         | Verbose output                                   |

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

| Flag       | Description                                     |
| ---------- | ----------------------------------------------- |
| `--engine` | Filter by engine: `style`, `lint`, `policy`     |
| `-v`       | Show detailed descriptions                      |

**Flags (docs):**

| Flag       | Description                                     |
| ---------- | ----------------------------------------------- |
| `--engine` | Filter by engine: `style`, `lint`, `policy`     |

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
terratidy rules docs --engine lint
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

**Flags (`show`):**

| Flag       | Description                                     |
| ---------- | ----------------------------------------------- |
| `--format` | Output format: `yaml`, `json` (default: `yaml`) |

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

## terratidy dev

Development mode with file watching.

```bash
terratidy dev [flags]
```

**Flags:**

| Flag       | Description                                      |
| ---------- | ------------------------------------------------ |
| `--watch`  | Directory to watch (default: `policies/`)        |
| `--target` | Target directory to check (default: `.`)         |

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
| `--json`  | Output in JSON format      |

**Examples:**

```bash
# Show full version info
terratidy version

# Show only version number
terratidy version --short

# Output as JSON
terratidy version --json
```

**Output:**

```text
TerraTidy version 0.2.0-alpha.3
  Commit:      abc1234
  Build date:  2025-12-22
  Go version:  go1.26.1
  Platform:    darwin/arm64
```
