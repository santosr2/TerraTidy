# Monorepos

Using TerraTidy in repositories with multiple Terraform modules.

## Setup

Initialize a monorepo configuration:

```bash
terratidy init --monorepo
```

This creates a `.terratidy.yaml` with:

- Central `./policies` directory for shared OPA policies
- `ci` profile (all engines enabled)
- `development` profile (lint and policy disabled for speed)

## Config Placement

Place `.terratidy.yaml` at the repository root. TerraTidy processes files relative to
the current working directory:

```text
repo/
  .terratidy.yaml          # Root config
  policies/                # Shared OPA policies
  modules/
    networking/
      main.tf
      variables.tf
    compute/
      main.tf
      variables.tf
  environments/
    staging/
      main.tf
    production/
      main.tf
```

## Running Checks

### Specific modules

```bash
terratidy check ./modules/networking
terratidy check ./modules/compute
```

### All modules

```bash
terratidy check ./modules ./environments
```

### Changed files only

```bash
terratidy check --changed
```

This checks all modified `.tf`/`.hcl`/`.tfvars` files across the entire repo.

### Exclude specific modules

```bash
# Exclude legacy modules during migration
terratidy check --exclude "modules/legacy/**"

# Check everything except test fixtures
terratidy check --exclude "test/**,**/testdata/**"
```

## Profile-Based Workflows

### Local development (fast)

```bash
terratidy check --profile development
```

### CI/CD (thorough)

```bash
terratidy check --profile ci --parallel
```

### Production deployment gate

```bash
terratidy check --profile ci --severity-threshold error
```

## Module Overrides

Override rules for specific modules using config imports:

```yaml
# .terratidy.yaml
version: 1

imports:
  - ./modules/networking/.terratidy-overrides.yaml

engines:
  style:
    enabled: true
```

```yaml
# modules/networking/.terratidy.yaml
engines:
  style:
    rules:
      style.terraform-files-structure:
        enabled: false  # Networking module uses a different structure
```

## CI Integration

Run checks per module in parallel CI jobs:

```yaml
# GitHub Actions
strategy:
  matrix:
    module: [networking, compute, database, iam]

steps:
  - uses: actions/checkout@v4
  - uses: santosr2/terratidy@v0
    with:
      working-directory: modules/${{ matrix.module }}
      profile: ci
```
