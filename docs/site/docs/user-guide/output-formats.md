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
| `junit`        | `--format junit`        | JUnit XML format               | CI/CD (Jenkins, GitLab)   |
| `markdown`     | `--format markdown`     | Markdown summary               | PR comments, summaries    |
| `github`       | `--format github`       | GitHub Actions workflow cmds   | GitHub Actions inline     |

Three formats accept a short alias: `gha` for `github`, `junit-xml` for `junit`, and `md`
for `markdown`. They produce identical output to their canonical names.

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

The default format for terminal output. Each finding is printed as
`<icon> <file>:<line>:<column>: <message> (<rule>)`, preceded by a short run
summary and followed by a severity breakdown:

```text
Checking 2 files...

Running checks in parallel mode...
  fmt: 0 issue(s)
  lint: 2 issue(s)
  style: 1 issue(s)

⚠ main.tf:1:1: resource name 'MyServer' should use snake_case (lint.terraform-naming-convention)
⚠ main.tf:4:1: Missing blank line between instance_type and tags (different attribute groups) (style.attribute-group-spacing)
⚠ variables.tf:1:1: Variable 'region' is missing a description (lint.terraform-documented-variables)
---
Summary: 3 total issue(s)

  Warnings: 3
```

The icon encodes severity: `✗` for errors, `⚠` for warnings, `ℹ` for info. The
`Errors:` / `Warnings:` / `Info:` breakdown lines are only printed for severities
that actually occur. File-level findings with no position omit the `:line:column:`
segment.

## Table Format

Columnar format with color-coded severity for quick visual scanning:

```bash
terratidy check --format table
```

Output:

```text
SEVERITY   LOCATION                                           MESSAGE
----------------------------------------------------------------------------------------------------
WARNING    main.tf:1:1                                        resource name 'MyServer' should use snake_case (lint.terraform-naming-convention)
WARNING    main.tf:4:1                                        Missing blank line between instance_type and tags (different attribute groups) (style.attribute-group-spacing)
WARNING    variables.tf:1:1                                   Variable 'region' is missing a description (lint.terraform-documented-variables)

----------------------------------------------------------------------------------------------------
Summary: 0 error(s), 3 warning(s), 0 info
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

Machine-readable format for automation. The document has two top-level keys,
`findings` and `summary`. Each finding carries a nested `location` with `start`
and `end` positions (not flat `line`/`column` fields):

```json
{
  "findings": [
    {
      "rule": "lint.terraform-naming-convention",
      "message": "resource name 'MyServer' should use snake_case",
      "file": "main.tf",
      "location": {
        "start": { "line": 1, "column": 1 },
        "end": { "line": 8, "column": 2 }
      },
      "severity": "warning",
      "fixable": false
    },
    {
      "rule": "style.attribute-group-spacing",
      "message": "Missing blank line between instance_type and tags (different attribute groups)",
      "file": "main.tf",
      "location": {
        "start": { "line": 4, "column": 1 },
        "end": { "line": 4, "column": 1 }
      },
      "severity": "warning",
      "fixable": true
    },
    {
      "rule": "lint.terraform-documented-variables",
      "message": "Variable 'region' is missing a description",
      "file": "variables.tf",
      "location": {
        "start": { "line": 1, "column": 1 },
        "end": { "line": 3, "column": 2 }
      },
      "severity": "warning",
      "fixable": false
    }
  ],
  "summary": {
    "total": 3,
    "errors": 0,
    "warnings": 3,
    "info": 0
  }
}
```

To read a finding's line number with `jq`, reach into the nested location:
`jq '.findings[].location.start.line'`.

## JSON Compact Format

Single-line JSON for log aggregation. Same schema as `json`, emitted on one line:

```bash
terratidy check --format json-compact
```

```json
{"findings":[{"rule":"lint.terraform-naming-convention","message":"resource name 'MyServer' should use snake_case","file":"main.tf","location":{"start":{"line":1,"column":1},"end":{"line":8,"column":2}},"severity":"warning","fixable":false}],"summary":{"total":1,"errors":0,"warnings":1,"info":0}}
```

## SARIF Format

Static Analysis Results Interchange Format for GitHub integration. The
`tool.driver.rules` array declares every rule that produced a result, and each
entry in `results` references one by `ruleId`. Auto-fixable findings also carry a
`fixes` array:

```json
{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "TerraTidy",
          "version": "0.2.0-alpha.4",
          "informationUri": "https://github.com/santosr2/TerraTidy",
          "rules": [
            {
              "id": "lint.terraform-naming-convention",
              "shortDescription": { "text": "lint.terraform-naming-convention" },
              "fullDescription": { "text": "" },
              "properties": { "tags": ["terraform", "terragrunt", "quality"] }
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "lint.terraform-naming-convention",
          "level": "warning",
          "message": { "text": "resource name 'MyServer' should use snake_case" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "main.tf", "uriBaseId": "%SRCROOT%" },
                "region": {
                  "startLine": 1,
                  "startColumn": 1,
                  "endLine": 8,
                  "endColumn": 2
                }
              }
            }
          ]
        }
      ]
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
  uses: github/codeql-action/upload-sarif@v4
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
::warning file=main.tf,line=1,col=1,title=lint.terraform-required-version::Missing terraform required_version constraint
::warning file=main.tf,line=1,col=1,endLine=8,endColumn=2,title=lint.terraform-naming-convention::resource name 'MyServer' should use snake_case
::warning file=main.tf,line=4,col=1,title=style.attribute-group-spacing::Missing blank line between instance_type and tags (different attribute groups)
```

Severity maps to the workflow command: errors emit `::error`, warnings `::warning`,
and info `::notice`. Findings that span multiple lines add `endLine` and `endColumn`
properties. These annotations appear directly in the GitHub PR "Files changed" view.

## JUnit XML Format

Standard test result format for CI/CD integration:

```bash
terratidy check --format junit > results.xml
```

Output. Suites are grouped by **file** (one `<testsuite>` per file, named by its
path), not by engine. Warnings become `<failure>` elements, errors become
`<error>`, and info findings render as passing test cases with no child element.
The failure body is a single XML-escaped string (`&#xA;` is a newline):

```xml
<?xml version="1.0" encoding="UTF-8"?>

<testsuites name="TerraTidy" tests="3" errors="0" failures="3" time="0" timestamp="2026-07-21T21:21:14Z">
  <testsuite name="main.tf" tests="2" errors="0" failures="2" skipped="0" time="0" timestamp="2026-07-21T21:21:14Z">
    <testcase name="lint.terraform-naming-convention" classname="main.tf" time="0">
      <failure message="resource name &#39;MyServer&#39; should use snake_case" type="warning">File: main.tf&#xA;Line: 1, Column: 1&#xA;&#xA;resource name &#39;MyServer&#39; should use snake_case</failure>
    </testcase>
    <testcase name="style.attribute-group-spacing" classname="main.tf" time="0">
      <failure message="Missing blank line between instance_type and tags (different attribute groups)" type="warning">File: main.tf&#xA;Line: 4, Column: 1&#xA;&#xA;Missing blank line between instance_type and tags (different attribute groups)</failure>
    </testcase>
  </testsuite>
  <testsuite name="variables.tf" tests="1" errors="0" failures="1" skipped="0" time="0" timestamp="2026-07-21T21:21:14Z">
    <testcase name="lint.terraform-documented-variables" classname="variables.tf" time="0">
      <failure message="Variable &#39;region&#39; is missing a description" type="warning">File: variables.tf&#xA;Line: 1, Column: 1&#xA;&#xA;Variable &#39;region&#39; is missing a description</failure>
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

Output. A `## Summary` count table is followed by a `## Findings` section grouped
by file. Severity is shown with an emoji prefix (`:x:` error, `:warning:` warning,
`:information_source:` info), and a generator footer closes the report:

```markdown
# TerraTidy Report

## Summary

| Severity | Count |
|----------|-------|
| Errors | 0 |
| Warnings | 3 |
| Info | 0 |
| **Total** | **3** |

## Findings

### `main.tf`

| Severity | Line | Rule | Message |
|----------|------|------|---------|
| :warning: warning | 1 | `lint.terraform-naming-convention` | resource name 'MyServer' should use snake_case |
| :warning: warning | 4 | `style.attribute-group-spacing` | Missing blank line between instance_type and tags (different attribute groups) |

### `variables.tf`

| Severity | Line | Rule | Message |
|----------|------|------|---------|
| :warning: warning | 1 | `lint.terraform-documented-variables` | Variable 'region' is missing a description |


---
*Generated by TerraTidy 0.2.0-alpha.4*
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

Self-contained HTML report with embedded CSS, no external dependencies:

- Summary cards: total issues, errors (red), warnings (yellow), info (cyan)
- Findings grouped by file with severity icons
- Fixable badge on auto-fixable findings
- Responsive layout for sharing

```bash
terratidy check --format html > report.html
open report.html  # View in browser
```

Useful for:

- Sharing results with non-CLI users
- Archiving quality reports
- Email attachments

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

### Path Format

By default, file paths in output are relative to the current working directory:

```bash
# Default: relative paths
terratidy check
# Output: ⚠ modules/vpc/main.tf:1:1: resource name 'MyServer' should use snake_case (lint.terraform-naming-convention)

# Use absolute paths
terratidy check --absolute-paths
# Output: ⚠ /Users/dev/project/modules/vpc/main.tf:1:1: resource name 'MyServer' should use snake_case (lint.terraform-naming-convention)
```

Relative paths are more readable in CI logs and editor integrations. Use `--absolute-paths`
when you need full paths for tooling integration or scripts that run from different directories.

This can also be set in configuration:

```yaml
output:
  absolute_paths: true
```

### Skip Specific Engines

```bash
# Skip policy checks
terratidy check --skip-policy

# Run only fmt and style
terratidy check --skip-lint --skip-policy
```
