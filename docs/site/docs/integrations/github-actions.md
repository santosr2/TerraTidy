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
      - uses: actions/checkout@v4

      - name: Run TerraTidy
        uses: santosr2/terratidy-action@v1
        with:
          format: text
```

## All Options

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy-action@v1
  with:
    # TerraTidy version (default: latest)
    version: 'latest'

    # Path to configuration file
    config: '.terratidy.yaml'

    # Configuration profile to use
    profile: ''

    # Output format: text, json, json-compact, sarif, html
    format: 'text'

    # Working directory
    working-directory: '.'

    # Comma-separated engines: fmt,style,lint,policy
    engines: ''

    # Auto-fix issues
    fix: 'false'

    # Fail on errors (default: true)
    fail-on-error: 'true'

    # Fail on warnings (default: false)
    fail-on-warning: 'false'

    # GitHub token for PR annotations
    github-token: ${{ secrets.GITHUB_TOKEN }}
```

## Outputs

| Output | Description |
|--------|-------------|
| `findings-count` | Total number of findings |
| `errors-count` | Number of error-level findings |
| `warnings-count` | Number of warning-level findings |
| `sarif-file` | Path to SARIF file (if sarif format) |

## Examples

### SARIF Upload to GitHub

```yaml
jobs:
  terratidy:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    steps:
      - uses: actions/checkout@v4

      - name: Run TerraTidy
        uses: santosr2/terratidy-action@v1
        with:
          format: sarif
          fail-on-error: 'false'
          github-token: ${{ secrets.GITHUB_TOKEN }}
```

### Check with Profile

```yaml
- name: Run TerraTidy CI checks
  uses: santosr2/terratidy-action@v1
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
      - uses: actions/checkout@v4

      - name: Run TerraTidy
        uses: santosr2/terratidy-action@v1
        with:
          working-directory: ${{ matrix.directory }}
```

### Format Check Only

```yaml
- name: Check Formatting
  uses: santosr2/terratidy-action@v1
  with:
    engines: fmt
    fail-on-error: 'true'
```

### Auto-fix in PR

```yaml
jobs:
  autofix:
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.head_ref }}

      - name: Run TerraTidy Fix
        uses: santosr2/terratidy-action@v1
        with:
          fix: 'true'

      - name: Commit fixes
        run: |
          git config --global user.name 'github-actions[bot]'
          git config --global user.email 'github-actions[bot]@users.noreply.github.com'
          git add -A
          git diff --staged --quiet || git commit -m "style: auto-fix terraform formatting"
          git push
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
      - uses: actions/checkout@v4

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.6.0"

      - name: Terraform Init
        run: terraform init

      - name: Terraform Validate
        run: terraform validate

      - name: TerraTidy Check
        uses: santosr2/terratidy-action@v1
        with:
          format: sarif
          profile: ci
          github-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const findings = '${{ steps.terratidy.outputs.findings-count }}';
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `TerraTidy found ${findings} issue(s).`
            })
```
