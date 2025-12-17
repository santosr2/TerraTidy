# TerraTidy GitHub Action

Run TerraTidy quality checks on your Terraform/Terragrunt code in GitHub Actions.

## Usage

### Basic Example

```yaml
name: TerraTidy

on: [push, pull_request]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run TerraTidy
        uses: santosr2/terratidy@v1
```

### With SARIF Upload

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v1
  with:
    command: check
    format: sarif
    upload_sarif: true
```

### Custom Configuration

```yaml
- name: Run TerraTidy
  uses: santosr2/terratidy@v1
  with:
    version: 'v0.1.0'
    command: check
    config_path: '.terratidy.yaml'
    working_directory: './terraform'
    severity_threshold: error
```

### Monorepo Example

```yaml
jobs:
  check-project-a:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: santosr2/terratidy@v1
        with:
          working_directory: './projects/project-a'
          
  check-project-b:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: santosr2/terratidy@v1
        with:
          working_directory: './projects/project-b'
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `version` | TerraTidy version to install | No | `latest` |
| `command` | Command to run (check, fmt, style, lint, policy) | No | `check` |
| `format` | Output format (text, json, sarif) | No | `sarif` |
| `config_path` | Path to config file | No | ` ` |
| `working_directory` | Working directory | No | `.` |
| `upload_sarif` | Auto-upload SARIF to Code Scanning | No | `true` |
| `severity_threshold` | Minimum severity to fail (info, warning, error) | No | `warning` |

## Outputs

When `format` is set to `sarif` and `upload_sarif` is `true`, results are automatically uploaded to GitHub Code Scanning and will appear in the Security tab of your repository.

## Permissions

To upload SARIF results, ensure your workflow has the necessary permissions:

```yaml
permissions:
  security-events: write
  contents: read
```

## Examples

### Run Only on Changed Files

```yaml
- name: Get changed files
  id: changed
  uses: tj-actions/changed-files@v41
  with:
    files: |
      **/*.tf
      **/*.hcl

- name: Run TerraTidy
  if: steps.changed.outputs.any_changed == 'true'
  uses: santosr2/terratidy@v1
  with:
    command: check --changed
```

### Matrix Strategy for Multiple Directories

```yaml
strategy:
  matrix:
    directory: ['infra/dev', 'infra/staging', 'infra/prod']

steps:
  - uses: actions/checkout@v4
  - uses: santosr2/terratidy@v1
    with:
      working_directory: ${{ matrix.directory }}
```

