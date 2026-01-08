# Output Formats

TerraTidy supports multiple output formats for different use cases.

## Available Formats

| Format         | Flag                    | Description                    | Use Case                  |
| -------------- | ----------------------- | ------------------------------ | ------------------------- |
| `text`         | `--format text`         | Human-readable colored output  | Terminal use              |
| `json`         | `--format json`         | Structured JSON                | CI/CD, scripts            |
| `json-compact` | `--format json-compact` | Single-line JSON               | Logging, streaming        |
| `sarif`        | `--format sarif`        | SARIF 2.1.0 format             | GitHub Code Scanning      |
| `html`         | `--format html`         | Visual HTML report             | Reports, sharing          |
| `github`       | `--format github`       | GitHub Actions workflow cmds   | GitHub Actions inline     |

## Usage

```bash
# Default text output
terratidy check

# JSON output
terratidy check --format json

# SARIF for GitHub Code Scanning
terratidy check --format sarif > results.sarif

# GitHub Actions annotations (inline in PR)
terratidy check --format github

# HTML report (redirect to file)
terratidy check --format html > report.html
```

## Text Format

The default format with colored output:

```text
main.tf:15:3: error [style.block-label-case] Resource name should use snake_case
main.tf:23:1: warning [style.tags-at-end] Place tags attribute at end of resource
variables.tf:8:1: info [lint.terraform-documented-variables] Variable should have a description

Found 3 issues (1 error, 1 warning, 1 info)
```

## JSON Format

Machine-readable format for automation:

```json
{
  "version": "1.0.0",
  "timestamp": "2024-01-15T10:30:00Z",
  "summary": {
    "total": 3,
    "errors": 1,
    "warnings": 1,
    "info": 1
  },
  "findings": [
    {
      "rule": "style.block-label-case",
      "message": "Resource name should use snake_case",
      "file": "main.tf",
      "line": 15,
      "column": 3,
      "severity": "error",
      "fixable": true
    }
  ]
}
```

## JSON Compact Format

Single-line JSON for log aggregation:

```bash
terratidy check --format json-compact
```

## SARIF Format

Static Analysis Results Interchange Format for GitHub integration:

```json
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "TerraTidy",
          "version": "0.1.0",
          "rules": [...]
        }
      },
      "results": [...]
    }
  ]
}
```

### GitHub Code Scanning

Upload SARIF results to GitHub:

```yaml
- name: Run TerraTidy
  run: terratidy check --format sarif > results.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

## GitHub Actions Format

Output GitHub workflow commands for inline PR annotations:

```bash
terratidy check --format github
```

Output:

```text
::error file=main.tf,line=15,col=3,title=style.block-label-case::Resource name should use snake_case
::warning file=main.tf,line=23,col=1,title=style.tags-at-end::Place tags attribute at end of resource
```

These annotations appear directly in the GitHub PR "Files changed" view.

## HTML Format

Visual HTML report with:

- Summary statistics
- Color-coded severity
- File and line information
- Rule descriptions

```bash
terratidy check --format html > report.html
```

## Output to File

Redirect output to a file:

```bash
# Write JSON to file
terratidy check --format json > results.json

# Write HTML report
terratidy check --format html > report.html

# Write SARIF
terratidy check --format sarif > results.sarif
```

## Combining with Other Tools

### jq for JSON Processing

```bash
# Get only errors
terratidy check --format json | jq '.findings | map(select(.severity == "error"))'

# Count by rule
terratidy check --format json | jq '.findings | group_by(.rule) | map({rule: .[0].rule, count: length})'
```

### Filtering by Severity

```bash
# Show only errors and above
terratidy check --severity-threshold error
```

### Skip Specific Engines

```bash
# Skip policy checks
terratidy check --skip-policy

# Run only fmt and style
terratidy check --skip-lint --skip-policy
```
