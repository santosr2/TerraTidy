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
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

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

    # Output format: text, json, json-compact, sarif, html, table, github
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

## Examples

### SARIF Upload to GitHub

```yaml
jobs:
  terratidy:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

      - name: Run TerraTidy
        uses: santosr2/terratidy@v0
        with:
          format: sarif
          fail-on-error: 'false'
          github-token: ${{ secrets.GITHUB_TOKEN }}
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
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

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
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@5e8dbf3c6d9deaf4193ca7a8fb23f2ac83bb6c85 # v4.0.0
        with:
          terraform_version: "1.6.0"

      - name: Terraform Init
        run: terraform init

      - name: Terraform Validate
        run: terraform validate

      - name: TerraTidy Check
        uses: santosr2/terratidy@v0
        with:
          format: sarif
          profile: ci
          github-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd # v8.0.0
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

## Status Badges

Add TerraTidy status badges to your repository README to show that your Terraform code is checked:

```markdown
<!-- Passing badge -->
![TerraTidy](https://raw.githubusercontent.com/santosr2/terratidy/main/assets/terratidy-status-passing.svg)

<!-- Failing badge -->
![TerraTidy](https://raw.githubusercontent.com/santosr2/terratidy/main/assets/terratidy-status-failing.svg)
```

To dynamically show pass/fail status based on your workflow, use GitHub's built-in workflow badge:

```markdown
[![TerraTidy](https://github.com/<owner>/<repo>/actions/workflows/<workflow>.yml/badge.svg)](https://github.com/<owner>/<repo>/actions)
```
