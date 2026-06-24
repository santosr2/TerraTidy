# TerraTidy Examples

This directory contains example configurations and integrations for TerraTidy.

## Configuration Files

### Basic Configuration

- **[terratidy-minimal.yaml](terratidy-minimal.yaml)** - Minimal configuration with defaults
- **[terratidy.yaml](terratidy.yaml)** - Complete configuration with all options
- **[terratidy-lint.yaml](terratidy-lint.yaml)** - Lint-focused configuration with TFLint integration options
- **[terragrunt.yaml](terragrunt.yaml)** - Terragrunt-focused configuration highlighting `style.terragrunt-include-first` (try it on `rule-test-files/terragrunt-include-order.hcl`)

**Usage:**

```bash
# Use default config (.terratidy.yaml in project root)
terratidy check

# Use specific config file
terratidy check --config examples/terratidy.yaml

# Use profile from config
terratidy check --profile ci
```

## Integration Examples

### Pre-commit Hooks

**File:** [pre-commit-config.yaml](pre-commit-config.yaml)

Copy to `.pre-commit-config.yaml` in your project root.

**Installation:**

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

**Features:**

- ✅ Format checking
- ✅ Style validation
- ✅ Linting
- ✅ Runs only on changed files
- ✅ Fast execution

### GitHub Actions

**File:** [github-workflow.yaml](github-workflow.yaml)

Copy to `.github/workflows/terratidy.yml` in your project.

**Features:**

- ✅ Runs on PRs and pushes
- ✅ SARIF upload for Code Scanning
- ✅ PR comments with results
- ✅ Auto-fix on develop branch
- ✅ Exclusion patterns for generated files
- ✅ Matrix builds for checking modules independently

## Custom Rules

TerraTidy supports three types of custom rules: Go plugins, YAML rules, and Bash scripts.

### Go Rules

**Directory:** [go-rule/](go-rule/)

Compiled Go plugins for advanced rule logic with full access to the HCL AST.

| File | Description |
|------|-------------|
| `main.go` | Example rule requiring tags on resources |

**Usage:**

```bash
# Build the plugin (Linux/macOS only)
cd examples/go-rule && go build -buildmode=plugin -o require-tags.so

# Configure plugin directory in .terratidy.yaml
plugins:
  enabled: true
  directories:
    - examples/go-rule
```

For more information, see the [Plugin Development Guide](../docs/site/docs/development/plugins.md#go-plugins).

### Bash Rules

**Directory:** [bash-rule/](bash-rule/)

Shell scripts for simple pattern matching and external tool integration.

| File | Description |
|------|-------------|
| `no-hardcoded-account-id.sh` | Detect hardcoded AWS account IDs |

**Usage:**

```bash
# Make script executable
chmod +x examples/bash-rule/no-hardcoded-account-id.sh

# Configure plugin directory in .terratidy.yaml
plugins:
  enabled: true
  directories:
    - examples/bash-rule
```

For more information, see the [Plugin Development Guide](../docs/site/docs/development/plugins.md#bash-rules).

### YAML Rules

**Directory:** [yaml-rule/](yaml-rule/)

Declarative rules for checking Terraform/HCL files without writing Go code.

| File | Description |
|------|-------------|
| `require-description.yaml` | Require description on resources |
| `require-variable-description.yaml` | Require description on variables |
| `no-deprecated-s3-args.yaml` | Forbid deprecated S3 arguments |
| `bucket-naming-convention.yaml` | Enforce bucket naming pattern |
| `s3-best-practices.yaml` | Combined example using all features |

**Usage:**

```bash
# Copy to plugins directory
cp examples/yaml-rule/*.yaml ~/.terratidy/plugins/
```

For more information, see the [Plugin Development Guide](../docs/site/docs/development/plugins.md#yaml-rules).

## Quick Start

### 1. Initialize Configuration

```bash
# Create minimal config
cat > .terratidy.yaml << 'EOF'
version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
EOF
```

### 2. Run First Check

```bash
# Check all files
terratidy check

# Check specific directory
terratidy check modules/

# Auto-fix issues
terratidy fix
```

### 3. Set Up Pre-commit (Optional)

```bash
# Copy example config
cp examples/pre-commit-config.yaml .pre-commit-config.yaml

# Install hooks
pre-commit install

# Test
pre-commit run --all-files
```

### 4. Set Up GitHub Actions (Optional)

```bash
# Create workflow directory
mkdir -p .github/workflows

# Copy example workflow
cp examples/github-workflow.yaml .github/workflows/terratidy.yml

# Commit and push
git add .github/workflows/terratidy.yml
git commit -m "Add TerraTidy workflow"
git push
```

## Common Scenarios

### Scenario 1: Format Only

Just want to format your files?

```bash
terratidy fmt
```

### Scenario 2: CI/CD Integration

Want to run in CI and fail on errors?

```bash
terratidy check --format sarif > results.sarif
exit_code=$?

# Upload SARIF to GitHub
# (see github-workflow.yaml)

exit $exit_code
```

### Scenario 3: Pre-commit Hook

Want to check files before commit?

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/santosr2/TerraTidy
    rev: v0.2.0-alpha.4
    hooks:
      - id: terratidy-check
```

### Scenario 4: Exclude Files and Directories

Want to skip generated files or test fixtures?

```bash
# Via CLI flag
terratidy check --exclude "**/*.generated.tf,test/**"

# Via config file
cat > .terratidy.yaml << 'EOF'
version: 1

exclude:
  - "**/*.generated.tf"
  - "vendor/**"
  - "test/fixtures/**"

engines:
  fmt: { enabled: true }
  style: { enabled: true }
EOF
```

### Scenario 5: Check Specific Module (Non-Recursive)

Want to check only a specific module without descending into submodules?

```bash
# Check only files in modules/vpc/, not modules/vpc/submodule/
terratidy check --no-recurse modules/vpc/
```

### Scenario 6: Per-Rule Configuration

Want to customize rule behavior?

```yaml
# .terratidy.yaml
version: 1

engines:
  style:
    enabled: true
    # Per-rule configuration - enable/disable rules, change severity
    rules:
      style.blank-line-between-blocks:
        enabled: true
        severity: warning
      style.block-label-case:
        enabled: true
        severity: error
```

## Output Formats

### Text (Default)

Human-readable output for terminal:

```bash
terratidy check
```

### JSON

Machine-readable for parsing:

```bash
terratidy check --format json > results.json
```

### SARIF

GitHub Code Scanning compatible:

```bash
terratidy check --format sarif > results.sarif
```

## Tips & Tricks

### 1. Run on Changed Files Only

```bash
# Using --changed flag (recommended)
terratidy check --changed

# Or using git directly
terratidy check $(git diff --name-only --diff-filter=ACM | grep -E '\.(tf|hcl)$')
```

### 2. Run Specific Engine

```bash
terratidy fmt --check    # Just formatting
terratidy style          # Just style
terratidy lint           # Just linting
```

### 3. Auto-fix Everything

```bash
terratidy fix           # Fix all auto-fixable issues
```

### 4. Detailed Output

```bash
terratidy check --format table
```

### 5. Fail on Specific Severity

```bash
terratidy check --severity-threshold error  # Only fail on errors
```

### 6. Exclude Generated or Vendored Files

```bash
terratidy check --exclude "**/*.generated.tf,vendor/**,test/**"
```

### 7. Check Directory Without Descending

```bash
terratidy check --no-recurse modules/  # Only top-level, not subdirectories
```

## Troubleshooting

### Issue: Pre-commit hook too slow

**Solution:** Run on changed files only (already configured in example)

### Issue: GitHub Action fails to install

**Solution:** Use the pre-built binary from releases:

```yaml
- name: Install TerraTidy
  run: |
    curl -L https://github.com/santosr2/TerraTidy/releases/download/v0.2.0-alpha.4/terratidy-linux-amd64 -o terratidy
    chmod +x terratidy
    sudo mv terratidy /usr/local/bin/
```

### Issue: Too many findings

**Solution:** Start with just formatting, then add style and lint:

```yaml
# .terratidy.yaml
version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: false  # Enable later
  lint:
    enabled: false  # Enable later
```

## More Information

- **Main Documentation:** [../README.md](../README.md)
- **Configuration Reference:** [terratidy.yaml](terratidy.yaml)
- **Contributing Guide:** [../CONTRIBUTING.md](../CONTRIBUTING.md)
