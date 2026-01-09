# Commands Reference

Complete reference for all TerraTidy commands.

## Global Flags

These flags are available for all commands:

| Flag                   | Description                                                                       |
| ---------------------- | --------------------------------------------------------------------------------- |
| `--config`             | Path to configuration file (default: `.terratidy.yaml`)                           |
| `--profile`            | Configuration profile to use                                                      |
| `--format`             | Output format: `text`, `table`, `json`, `json-compact`, `sarif`, `html`, `github` |
| `--paths`              | Paths to check (comma-separated)                                                  |
| `--changed`            | Only check files changed in git                                                   |
| `--color`              | Enable colored output (default: true)                                             |
| `--severity-threshold` | Minimum severity: `info`, `warning`, `error`                                      |

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

| Flag      | Description                             |
| --------- | --------------------------------------- |
| `--check` | Check formatting without modifying      |
| `--diff`  | Show diff of changes                    |
| `--fix`   | Auto-fix formatting issues (default)    |

**Examples:**

```bash
# Format all files
terratidy fmt

# Check formatting only
terratidy fmt --check
```

## terratidy style

Check and fix style issues.

```bash
terratidy style [paths...] [flags]
```

**Flags:**

| Flag    | Description           |
| ------- | --------------------- |
| `--fix` | Auto-fix style issues |

**Examples:**

```bash
# Check style
terratidy style

# Fix style issues
terratidy style --fix
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

| Flag           | Description                          |
| -------------- | ------------------------------------ |
| `--policy-dir` | Directory containing .rego files     |

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
terratidy init-rule [name] [flags]
```

**Examples:**

```bash
# Create a new Go rule
terratidy init-rule my-custom-rule --type go

# Create a YAML rule
terratidy init-rule my-rule --type yaml
```

## terratidy test-rule

Test a specific rule.

```bash
terratidy test-rule [rule-name] [flags]
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

**Examples:**

```bash
# List all rules
terratidy rules list

# List with verbose output
terratidy rules list --verbose

# Generate documentation
terratidy rules docs
```

## terratidy config

Configuration management.

```bash
terratidy config [command]
```

**Subcommands:**

| Command        | Description                            |
| -------------- | -------------------------------------- |
| `show`         | Display current configuration          |
| `validate`     | Validate configuration file            |
| `split`        | Split configuration into modules       |
| `merge`        | Merge split configurations             |
| `init-profile` | Initialize a new configuration profile |

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

| Flag       | Description                            |
| ---------- | -------------------------------------- |
| `--watch`  | Directory to watch (default: policies/)|
| `--target` | Target directory to check (default: .) |

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
terratidy version
```

**Output:**

```text
TerraTidy version 0.1.0
  Commit:      abc1234
  Build date:  2025-12-22
  Go version:  go1.25.0
  Platform:    darwin/arm64
```
