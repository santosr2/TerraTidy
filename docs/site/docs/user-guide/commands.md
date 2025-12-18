# Commands Reference

Complete reference for all TerraTidy commands.

## Global Flags

These flags are available for all commands:

| Flag | Description |
|------|-------------|
| `--config` | Path to configuration file (default: `.terratidy.yaml`) |
| `--profile` | Configuration profile to use |
| `--format` | Output format: `text`, `json`, `sarif`, `html` |
| `--paths` | Paths to check (comma-separated) |
| `--changed` | Only check files changed in git |
| `--severity-threshold` | Minimum severity: `info`, `warning`, `error` |

## terratidy check

Run all enabled checks.

```bash
terratidy check [flags]
```

**Examples:**

```bash
# Check all files
terratidy check

# Check specific directory
terratidy check --paths ./modules/

# Check with CI profile
terratidy check --profile ci

# Output as JSON
terratidy check --format json
```

## terratidy fmt

Format Terraform and Terragrunt files.

```bash
terratidy fmt [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--check` | Check formatting without modifying files |
| `--diff` | Show diff of changes |

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
terratidy style [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
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
terratidy lint [flags]
```

**Examples:**

```bash
# Run linting
terratidy lint

# Use TFLint integration
terratidy lint --use-tflint
```

## terratidy policy

Run OPA/Rego policy checks.

```bash
terratidy policy [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--policy-dir` | Directory containing .rego files |

## terratidy fix

Auto-fix all fixable issues.

```bash
terratidy fix [flags]
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

| Flag | Description |
|------|-------------|
| `--profile` | Base profile: `default`, `strict`, `minimal` |
| `--force` | Overwrite existing configuration |

## terratidy rules

Manage rules.

```bash
terratidy rules [command]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `list` | List all available rules |
| `info [rule]` | Show rule details |
| `enable [rule]` | Enable a rule |
| `disable [rule]` | Disable a rule |

## terratidy config

Configuration management.

```bash
terratidy config [command]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `show` | Display current configuration |
| `validate` | Validate configuration file |

## terratidy plugins

Plugin management.

```bash
terratidy plugins [command]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `list` | List installed plugins |
| `info [name]` | Show plugin details |
| `init [name]` | Create new plugin project |

## terratidy lsp

Start the Language Server Protocol server.

```bash
terratidy lsp
```

## terratidy dev

Development mode with file watching.

```bash
terratidy dev [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--port` | Web UI port |
| `--no-browser` | Don't open browser |

## terratidy version

Show version information.

```bash
terratidy version
```
