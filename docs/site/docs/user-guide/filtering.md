# Filtering and Exclusions

Control which files TerraTidy processes using exclusion patterns, recursive scanning, and VCS integration.

## Quick Reference

| Method | Scope | Example |
|--------|-------|---------|
| `--exclude` | CLI, ad-hoc | `--exclude "test/**"` |
| `exclude:` | Config file | `exclude: ["vendor/**"]` |
| `--no-recurse` | CLI, ad-hoc | `--no-recurse modules/` |
| `recursive:` | Config file | `recursive: false` |
| `--changed` | CLI, VCS-aware | `--changed` |

## Exclusion Patterns

### CLI Flag

Use `--exclude` to skip files matching glob patterns:

```bash
# Single pattern
terratidy check --exclude "**/*.generated.tf"

# Multiple patterns (comma-separated)
terratidy check --exclude "test/**,vendor/**,**/*.generated.tf"

# Multiple patterns (repeated flag)
terratidy check --exclude "test/**" --exclude "vendor/**"
```

### Configuration File

Define permanent exclusions in `.terratidy.yaml`:

```yaml
exclude:
  - "**/*.generated.tf"     # Generated files
  - "vendor/**"             # Vendored dependencies
  - ".terraform/**"         # Terraform cache
  - "test/fixtures/**"      # Test fixtures
  - "modules/legacy/**"     # Legacy code during migration
```

**Combining CLI and config:** Patterns from both sources are combined. CLI patterns add to config patterns, they don't replace them.

### Pattern Syntax

| Pattern | Description | Matches |
|---------|-------------|---------|
| `*` | Any characters except `/` | `*.tf` matches `main.tf` |
| `**` | Zero or more directories | `**/*.tf` matches `a/b/c/main.tf` |
| `?` | Any single character | `?.tf` matches `a.tf` |
| `[abc]` | Character class | `[mv]*.tf` matches `main.tf`, `vars.tf` |

### Common Patterns

```yaml
exclude:
  # Generated files
  - "**/*.generated.tf"
  - "**/*.auto.tfvars"

  # Build artifacts
  - ".terraform/**"
  - ".terragrunt-cache/**"

  # Dependencies
  - "vendor/**"
  - "node_modules/**"

  # Test files
  - "test/**"
  - "**/testdata/**"
  - "**/*_test.tf"

  # Environment-specific
  - "environments/sandbox/**"

  # Legacy code
  - "modules/deprecated/**"
```

## Recursive Scanning

### Default Behavior

By default, TerraTidy recursively scans all subdirectories:

```bash
terratidy check modules/
# Checks: modules/*.tf, modules/vpc/*.tf, modules/rds/*.tf, etc.
```

### Disable Recursion

Use `--no-recurse` to scan only the specified directories:

```bash
# Scan only modules/ (not subdirectories)
terratidy check --no-recurse modules/

# Scan multiple directories without recursion
terratidy check --no-recurse . modules/ environments/
```

### Configuration File

```yaml
recursive: false  # Disable recursive scanning globally
```

The CLI flag `--no-recurse` takes precedence over the config setting.

### Use Cases

**Module-level checks:** Check each module independently without descending:

```bash
terratidy check --no-recurse modules/vpc/
terratidy check --no-recurse modules/rds/
```

**Root-only validation:** Check only files in the current directory:

```bash
terratidy check --no-recurse .
```

## VCS Integration

### Changed Files Only

The `--changed` flag uses git to detect modified files:

```bash
terratidy check --changed
```

This checks:

- Staged files (added to index)
- Unstaged modifications
- Untracked `.tf`, `.hcl`, `.tfvars` files

**How it works:**

1. Detects the default branch (`main` or `master`)
2. Finds the merge-base between HEAD and the default branch
3. Lists all changed files since that point
4. Filters to supported file types

### Combining with Other Flags

```bash
# Changed files, excluding test fixtures
terratidy check --changed --exclude "test/**"

# Changed files in specific directory only
terratidy check --changed --no-recurse modules/
```

## Automatically Skipped Directories

These directories are always skipped, regardless of configuration:

| Directory | Reason |
|-----------|--------|
| `.terraform/` | Terraform provider cache |
| `.terragrunt-cache/` | Terragrunt cache |
| `node_modules/` | npm dependencies |
| `vendor/` | Go dependencies |
| `__pycache__/` | Python cache |
| `.*` (hidden) | Hidden directories (except `.`) |

You don't need to exclude these explicitly.

## Precedence

When multiple filtering methods are used:

1. **Automatically skipped directories** are always excluded
2. **Config file `exclude:`** patterns are applied
3. **CLI `--exclude`** patterns are added
4. **`--changed`** filters to VCS-modified files
5. **`--no-recurse`** limits directory depth

All filters are cumulative (they reduce the file set, never expand it).

## Examples

### CI Pipeline

```yaml
# .github/workflows/terratidy.yml
- name: Check changed Terraform files
  run: |
    terratidy check --changed \
      --exclude "test/**,examples/**" \
      --format github
```

### Pre-commit Hook

```bash
# Fast check on staged files only
terratidy check --changed --exclude "**/*.generated.tf"
```

### Monorepo

```yaml
# .terratidy.yaml
exclude:
  - "modules/legacy/**"      # Skip legacy during migration
  - "environments/sandbox/**" # Skip sandbox environment
  - "**/testdata/**"          # Skip test fixtures

profiles:
  ci:
    description: "Strict CI checks"
    engines:
      policy: { enabled: true }

  development:
    description: "Fast local checks"
    engines:
      lint: { enabled: false }
      policy: { enabled: false }
```

!!! note "Profile-specific exclusions"
    The `exclude:` setting is a top-level config key only. Profiles cannot override exclusions.
    For profile-specific file filtering, use separate config files with `imports:`.

### Module Development

```bash
# Check only the module you're working on
terratidy check --no-recurse modules/vpc/

# Check module and its submodules
terratidy check modules/vpc/
```

## Troubleshooting

### Files Not Being Checked

1. Check if they match an exclusion pattern:

   ```bash
   terratidy config show | grep exclude
   ```

2. Check if they're in an automatically skipped directory

3. Verify file extension is `.tf`, `.hcl`, or `.tfvars`

### Too Many Files Being Checked

1. Add exclusion patterns:

   ```yaml
   exclude:
     - "path/to/skip/**"
   ```

2. Use `--no-recurse` for targeted checks

3. Use `--changed` for incremental checks

### `--changed` Not Finding Files

1. Verify you're in a git repository
2. Check the default branch is detected:

   ```bash
   git symbolic-ref refs/remotes/origin/HEAD
   ```

3. Verify files have been modified since the merge-base
