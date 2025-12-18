# TerraTidy Examples

This directory contains example configurations and integrations for TerraTidy.

## Configuration Files

### Basic Configuration
- **[terratidy-minimal.yaml](terratidy-minimal.yaml)** - Minimal configuration with defaults
- **[terratidy.yaml](terratidy.yaml)** - Complete configuration with all options

**Usage:**
```bash
# Use default config (.terratidy.yaml in project root)
terratidy check

# Use specific config file
terratidy check --config examples/terratidy.yaml

# Use profile from config
terratidy check --profile production
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
- ✅ Caching for faster runs

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
  - repo: local
    hooks:
      - id: terratidy
        name: TerraTidy
        entry: terratidy check
        language: system
        files: \.(tf|hcl)$
```

### Scenario 4: Custom Rules
Want to add custom style rules?

```yaml
# .terratidy.yaml
version: 1

engines:
  style:
    enabled: true
    config:
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
# Using git
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

### 4. Verbose Output
```bash
terratidy check --verbose
```

### 5. Fail on Specific Severity
```bash
terratidy check --severity-threshold error  # Only fail on errors
```

## Troubleshooting

### Issue: Pre-commit hook too slow
**Solution:** Run on changed files only (already configured in example)

### Issue: GitHub Action fails to install
**Solution:** Use the pre-built binary from releases:
```yaml
- name: Install TerraTidy
  run: |
    curl -L https://github.com/santosr2/terratidy/releases/download/v1.0.0/terratidy-linux-amd64 -o terratidy
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
- **Development Guide:** [../AGENT.md](../AGENT.md)
- **Build Progress:** [../BUILD_PROGRESS.md](../BUILD_PROGRESS.md)
