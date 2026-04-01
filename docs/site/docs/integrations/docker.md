# Docker

Running TerraTidy in Docker containers.

## Docker Image

```bash
# Latest stable version
docker pull ghcr.io/santosr2/terratidy:latest

# Pin to a specific version (recommended for CI)
docker pull ghcr.io/santosr2/terratidy:v0.2.0-alpha.3
```

## Basic Usage

Mount your Terraform code into `/app` (the container's working directory):

```bash
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check
```

## Common Commands

```bash
# Check all files
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check

# Format files
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy fmt

# Fix style issues
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy fix

# JSON output
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check --format json

# SARIF output to file
docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check --format sarif > results.sarif
```

## Build from Source

Build a local Docker image for development or testing:

```bash
# Build binary and Docker image
make docker-build

# Run the locally built image
make docker-run

# Or build manually
make build
cp bin/terratidy terratidy
docker build -t terratidy-dev .
rm terratidy
docker run --rm -v $(pwd):/app terratidy-dev check
```

## Environment Variables

Pass environment variables for config expansion:

```bash
docker run --rm \
  -v $(pwd):/app \
  -e AWS_REGION=us-east-1 \
  -e AWS_ACCOUNT_ID=123456789 \
  ghcr.io/santosr2/terratidy check
```

## Custom Config

Mount a custom config file:

```bash
docker run --rm \
  -v $(pwd):/app \
  -v $(pwd)/custom.yaml:/app/.terratidy.yaml \
  ghcr.io/santosr2/terratidy check
```

## CI/CD Patterns

### GitHub Actions

```yaml
- name: Run TerraTidy
  run: |
    docker run --rm \
      -v ${{ github.workspace }}:/app \
      ghcr.io/santosr2/terratidy:v0.2.0-alpha.3 \
      check --format github
```

### GitLab CI

```yaml
terratidy:
  image: ghcr.io/santosr2/terratidy:v0.2.0-alpha.3
  script:
    - terratidy check --format junit > results.xml
  artifacts:
    reports:
      junit: results.xml
```

## Performance Tips

- **Layer caching:** The Docker image is small and self-contained, so pulls are fast
- **Volume mounts:** Mount only the directories you need to check
- **Format selection:** Use `--format json` or `--format junit` for CI (no color overhead)
- **Skip engines:** Use `--skip-lint --skip-policy` if you only need formatting checks
