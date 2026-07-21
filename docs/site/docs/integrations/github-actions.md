# GitHub Actions

TerraTidy provides a GitHub Action for easy CI/CD integration.

## Basic Usage

```yaml
name: Terraform Quality

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  terratidy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7.0.0

      - name: Run TerraTidy
        uses: santosr2/terratidy@v0
        with:
          format: text
```

## All Options

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v0
  with:
    # TerraTidy version (default: latest)
    version: 'latest'

    # Path to configuration file
    config: '.terratidy.yaml'

    # Configuration profile to use
    profile: ''

    # Output format: text, table, json, json-compact, sarif, html, junit, markdown, github
    format: 'text'

    # Run engines in parallel
    parallel: 'false'

    # Working directory
    working-directory: '.'

    # Skip individual engines
    skip-fmt: 'false'
    skip-style: 'false'
    skip-lint: 'false'
    skip-policy: 'false'

    # Exclude patterns (comma-separated glob patterns)
    exclude: ''

    # Disable recursive directory traversal
    no-recurse: 'false'

    # Output absolute file paths instead of relative
    absolute-paths: 'false'

    # Only check files changed in git
    changed: 'false'

    # Minimum severity to report: info, warning, error (default: uses config or warning)
    severity-threshold: ''

    # Fail on errors (default: true)
    fail-on-error: 'true'

    # Fail on warnings (default: false)
    fail-on-warning: 'false'

    # GitHub token for PR annotations
    github-token: ${{ secrets.GITHUB_TOKEN }}
```

## Outputs

| Output             | Description                          |
| ------------------ | ------------------------------------ |
| `findings-count`   | Total number of findings             |
| `errors-count`     | Number of error-level findings       |
| `warnings-count`   | Number of warning-level findings     |
| `sarif-file`       | Path to SARIF file (if sarif format) |

**Note:** Accurate counts for `findings-count`, `errors-count`, and `warnings-count` are
available when using `format: json`, `format: json-compact`, or when `fail-on-warning: true`
is set (triggers a JSON pre-run). For other configurations, output values are based on the
exit code:

- `findings-count`: the exit code (0 = clean, 1 = findings exist, 2 = config error, 3 = internal error)
- `errors-count`: same as `findings-count` when non-zero (exit 1 means at least one error-level finding exists; warning-only runs exit 0)
- `warnings-count`: 0 (not tracked in fallback mode)

Exit codes 2 and 3 indicate errors, not finding counts.

## Examples

### SARIF Upload to GitHub

The action automatically uploads SARIF results when `format: sarif` and `github-token` are provided:

```yaml
jobs:
  terratidy:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7.0.0

      - name: Run TerraTidy
        id: terratidy
        uses: santosr2/terratidy@v0
        with:
          format: sarif
          fail-on-error: 'false'
          github-token: ${{ secrets.GITHUB_TOKEN }}
      # Note: SARIF upload is automatic when github-token is provided
      # The sarif-file output can be used for manual upload if needed:
      # ${{ steps.terratidy.outputs.sarif-file }}
```

### Check with Profile

```yaml
- name: Run TerraTidy CI checks
  uses: santosr2/terratidy@v0
  with:
    profile: ci
    fail-on-warning: 'true'
```

### Multiple Directories

```yaml
jobs:
  terratidy:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        directory: [modules/vpc, modules/ecs, environments/prod]
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7.0.0

      - name: Run TerraTidy
        uses: santosr2/terratidy@v0
        with:
          working-directory: ${{ matrix.directory }}
```

### Format Check Only

```yaml
- name: Check Formatting
  uses: santosr2/terratidy@v0
  with:
    skip-style: 'true'
    skip-lint: 'true'
    skip-policy: 'true'
    fail-on-error: 'true'
```

### Exclude Generated Files

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v0
  with:
    exclude: '**/*.generated.tf,vendor/**,test/**'
    fail-on-error: 'true'
```

### Check Specific Directory (Non-Recursive)

```yaml
- name: Check Root Module Only
  uses: santosr2/terratidy@v0
  with:
    working-directory: 'modules/vpc'
    no-recurse: 'true'
```

### Complete Workflow

```yaml
name: Terraform CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
      pull-requests: write

    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7.0.0

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@5e8dbf3c6d9deaf4193ca7a8fb23f2ac83bb6c85 # v4.0.0
        with:
          terraform_version: "1.6.0"

      - name: Terraform Init
        run: terraform init

      - name: Terraform Validate
        run: terraform validate

      - name: TerraTidy Check
        id: terratidy
        uses: santosr2/terratidy@v0
        with:
          format: sarif
          profile: ci
          github-token: ${{ secrets.GITHUB_TOKEN }}

      # Note: findings-count with sarif format is the exit code (0/1/2/3), not
      # the actual count. For accurate counts, use format: json or add fail-on-warning: true.
      - name: Comment on PR
        if: github.event_name == 'pull_request' && steps.terratidy.outputs.findings-count != '0'
        uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd # v8.0.0
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: 'TerraTidy found issues. See the SARIF results in the Security tab.'
            })
```

## Status Badges

Add a TerraTidy status badge to your repository README using GitHub's built-in workflow badge:

```markdown
[![TerraTidy](https://github.com/<owner>/<repo>/actions/workflows/<workflow>.yml/badge.svg)](https://github.com/<owner>/<repo>/actions)
```
