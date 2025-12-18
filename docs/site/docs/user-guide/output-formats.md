# Output Formats

TerraTidy supports multiple output formats for different use cases.

## Available Formats

| Format | Description | Use Case |
|--------|-------------|----------|
| `text` | Human-readable colored output | Terminal use |
| `json` | Machine-readable JSON | CI/CD, scripts |
| `sarif` | SARIF 2.1.0 format | GitHub, IDE integration |
| `html` | Interactive HTML report | Reports, sharing |
| `junit` | JUnit XML format | CI test reporting |
| `checkstyle` | Checkstyle XML format | Legacy CI systems |

## Usage

```bash
# Default text output
terratidy check

# JSON output
terratidy check --format json

# SARIF for GitHub
terratidy check --format sarif > results.sarif

# HTML report
terratidy check --format html --output report.html
```

## Text Format

The default format with colored output:

```text
main.tf:15:3: error [resource-naming] Resource name should use snake_case
main.tf:23:1: warning [missing-tags] Resource should have tags
variables.tf:8:1: info [variable-description] Variable should have a description

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
      "rule": "resource-naming",
      "message": "Resource name should use snake_case",
      "file": "main.tf",
      "line": 15,
      "column": 3,
      "severity": "error",
      "engine": "style",
      "fixable": true
    }
  ]
}
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
          "version": "1.0.0",
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
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: results.sarif
```

## HTML Format

Interactive HTML report with:

- Summary statistics
- Filterable findings table
- Syntax-highlighted code snippets
- Expandable details

```bash
terratidy check --format html --output report.html
```

## JUnit Format

For CI/CD test reporting:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="TerraTidy" tests="3" failures="1" errors="0">
  <testsuite name="style" tests="2" failures="1">
    <testcase name="resource-naming" classname="main.tf">
      <failure message="Resource name should use snake_case"/>
    </testcase>
  </testsuite>
</testsuites>
```

### Jenkins Integration

```groovy
pipeline {
  stages {
    stage('Lint') {
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

## Checkstyle Format

Legacy format for older CI systems:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<checkstyle version="8.0">
  <file name="main.tf">
    <error line="15" column="3" severity="error"
           message="Resource name should use snake_case"
           source="terratidy.style.resource-naming"/>
  </file>
</checkstyle>
```

## Output to File

Use `--output` to write to a file:

```bash
# Write JSON to file
terratidy check --format json --output results.json

# Write HTML report
terratidy check --format html --output report.html
```

## Combining with Other Tools

### jq for JSON Processing

```bash
# Get only errors
terratidy check --format json | jq '.findings | map(select(.severity == "error"))'

# Count by rule
terratidy check --format json | jq '.findings | group_by(.rule) | map({rule: .[0].rule, count: length})'
```

### Filtering Output

```bash
# Show only errors
terratidy check --severity error

# Show specific engines
terratidy check --engines style,lint
```
