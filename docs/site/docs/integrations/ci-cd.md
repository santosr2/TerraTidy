# CI/CD Integration

Integrate TerraTidy into your CI/CD pipeline.

## GitHub Actions

Use the official GitHub Action:

```yaml
- uses: santosr2/terratidy@v0
  with:
    format: github    # Inline PR annotations
```

See [GitHub Actions](github-actions.md) for full documentation.

## Jenkins

```groovy
pipeline {
    agent any

    stages {
        stage('Terraform Quality') {
            steps {
                sh 'go install github.com/santosr2/TerraTidy/cmd/terratidy@latest'
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

## GitLab CI

```yaml
terratidy:
  image: ghcr.io/santosr2/terratidy:v0.3.0
  stage: validate
  script:
    - terratidy check --format junit > terratidy-results.xml
  artifacts:
    reports:
      junit: terratidy-results.xml
  rules:
    - changes:
        - "**/*.tf"
        - "**/*.hcl"
```

## Bitbucket Pipelines

```yaml
pipelines:
  pull-requests:
    '**':
      - step:
          name: TerraTidy
          image: golang:1.26.4
          script:
            - go install github.com/santosr2/TerraTidy/cmd/terratidy@latest
            - terratidy check --format text
```

## CircleCI

```yaml
version: 2.1

jobs:
  terratidy:
    docker:
      - image: ghcr.io/santosr2/terratidy:v0.3.0
    steps:
      - checkout
      - run:
          name: TerraTidy Check
          command: terratidy check --format text

workflows:
  validate:
    jobs:
      - terratidy
```

## Docker-Based CI

For any CI system, use the Docker image:

```bash
docker run --rm \
  -v $(pwd):/app \
  ghcr.io/santosr2/terratidy:v0.3.0 \
  check --format json
```

## Output Format Selection

Choose the output format based on your CI system:

| CI System | Recommended Format | Why |
|-----------|-------------------|-----|
| GitHub Actions | `github` | Inline PR annotations |
| Jenkins | `junit` | JUnit plugin integration |
| GitLab CI | `junit` | Built-in artifact reports |
| Any CI | `json` | Machine-readable, scriptable |
| PR comments | `markdown` | Human-readable summaries |
| Code scanning | `sarif` | GitHub/GitLab security tab |

## Best Practices

1. **Use `--changed` for PR checks** to only validate modified files
2. **Use `--parallel` for full scans** to speed up CI runs
3. **Use `--exclude` to skip generated or vendored files** (e.g., `--exclude "vendor/**,**/*.generated.tf"`)
4. **Pin the version** to avoid surprises from upgrades
5. **Use profiles** for different CI stages (fast PR check vs thorough merge gate)
6. **Upload SARIF** for GitHub Code Scanning integration
7. **Fail on errors only** with `--severity-threshold error` for lenient checks
