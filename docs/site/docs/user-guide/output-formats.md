# Output Formats

TerraTidy supports multiple output formats for different use cases.

## Available Formats

| Format         | Flag                    | Description                    | Use Case                  |
| -------------- | ----------------------- | ------------------------------ | ------------------------- |
| `text`         | `--format text`         | Human-readable output          | Terminal use (default)    |
| `table`        | `--format table`        | Colored table with columns     | Terminal, quick review    |
| `json`         | `--format json`         | Structured JSON                | CI/CD, scripts            |
| `json-compact` | `--format json-compact` | Single-line JSON               | Logging, streaming        |
| `sarif`        | `--format sarif`        | SARIF 2.1.0 format             | GitHub Code Scanning      |
| `html`         | `--format html`         | Visual HTML report             | Reports, sharing          |
| `github`       | `--format github`       | GitHub Actions workflow cmds   | GitHub Actions inline     |
| `junit`        | `--format junit`        | JUnit XML format               | CI/CD (Jenkins, GitLab)   |
| `markdown`     | `--format markdown`     | Markdown summary               | PR comments, summaries    |

## Usage

```bash
# Default text output
terratidy check

# Table format with colors
terratidy check --format table

# Disable colors (useful for CI or piping)
terratidy check --format table --color=false

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

The default format for terminal output:

```text
main.tf:15:3: error [style.block-label-case] Resource name should use snake_case
main.tf:23:1: warning [style.tags-at-end] Place tags attribute at end of resource
variables.tf:8:1: info [lint.terraform-documented-variables] Variable should have a description

Found 3 issues (1 error, 1 warning, 1 info)
```

## Table Format

Columnar format with color-coded severity for quick visual scanning:

```bash
terratidy check --format table
```

Output:

```text
SEVERITY   LOCATION                                           MESSAGE
────────────────────────────────────────────────────────────────────────────────────────────────────
ERROR      main.tf:15:3                                       Resource name should use snake_case
WARNING    main.tf:23:1                                       Place tags attribute at end of resource
INFO       variables.tf:8:1                                   Variable should have a description

────────────────────────────────────────────────────────────────────────────────────────────────────
Summary: 1 error(s) 1 warning(s) 1 info
```

Colors:

- **Red**: Errors
- **Yellow**: Warnings
- **Cyan**: Info

Use `--color=false` to disable colors:

```bash
terratidy check --format table --color=false
```

## JSON Format

Machine-readable format for automation:

```json
{
  "version": "0.2.0-alpha.2",
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
          "version": "0.2.0-alpha.2",
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

## JUnit XML Format

Standard test result format for CI/CD integration:

```bash
terratidy check --format junit > results.xml
```

Output:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="TerraTidy" tests="3" failures="2" errors="0" time="0.5">
  <testsuite name="style" tests="2" failures="1" errors="0" time="0.3">
    <testcase name="style.block-label-case" classname="main.tf" time="0.1">
      <failure message="Resource name should use snake_case" type="error">
        File: main.tf
        Line: 15
        Column: 3
      </failure>
    </testcase>
    <testcase name="style.variable-naming" classname="variables.tf" time="0.1"/>
  </testsuite>
  <testsuite name="lint" tests="1" failures="1" errors="0" time="0.2">
    <testcase name="lint.terraform-documented-variables" classname="variables.tf" time="0.1">
      <failure message="Variable should have a description" type="warning">
        File: variables.tf
        Line: 8
        Column: 1
      </failure>
    </testcase>
  </testsuite>
</testsuites>
```

### Jenkins Integration

```groovy
pipeline {
    stages {
        stage('TerraTidy') {
            steps {
                sh 'terratidy check --format junit > terratidy-results.xml'
            }
            post {
                always {
                    junit 'terratidy-results.xml'
                }
            }
        }
    }
}
```

### GitLab CI Integration

```yaml
terratidy:
  script:
    - terratidy check --format junit > terratidy-results.xml
  artifacts:
    reports:
      junit: terratidy-results.xml
```

## Markdown Format

Human-readable markdown summary, ideal for PR comments:

```bash
terratidy check --format markdown > summary.md
```

Output:

```markdown
# TerraTidy Report

**Summary:** 3 issues found (1 error, 1 warning, 1 info)

## Errors (1)

| File | Line | Rule | Message |
|------|------|------|---------|
| main.tf | 15 | style.block-label-case | Resource name should use snake_case |

## Warnings (1)

| File | Line | Rule | Message |
|------|------|------|---------|
| main.tf | 23 | style.tags-at-end | Place tags attribute at end of resource |

## Info (1)

| File | Line | Rule | Message |
|------|------|------|---------|
| variables.tf | 8 | lint.terraform-documented-variables | Variable should have a description |
```

### GitHub Actions Summary

```yaml
- name: Run TerraTidy
  run: |
    terratidy check --format markdown > $GITHUB_STEP_SUMMARY
```

### PR Comment with GitHub CLI

```yaml
- name: Run TerraTidy
  run: |
    terratidy check --format markdown > terratidy-report.md
    gh pr comment ${{ github.event.pull_request.number }} --body-file terratidy-report.md
```

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
