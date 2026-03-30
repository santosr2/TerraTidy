# Security

TerraTidy's CI/CD pipeline includes automated security scanning at multiple stages.

## CI Security Scanning

The security workflow (`.github/workflows/security.yml`) runs on every push to main and on pull requests that change Go source files.

### govulncheck

Scans Go dependencies against the [Go vulnerability database](https://vuln.go.dev/). Fails the build if any known vulnerabilities are found in code paths actually used by TerraTidy.

### dependency-review

Runs on pull requests only. Reviews dependency changes (additions, updates, removals) and flags any with known security advisories before they are merged.

### gitleaks

Scans the repository for accidentally committed secrets (API keys, tokens, private keys, etc.). Runs with full git history to catch secrets in any commit.

### go-licenses

Verifies all dependencies use approved open-source licenses:

- Apache-2.0, BSD-2-Clause-FreeBSD, BSD-3-Clause, MIT, MPL-2.0

If a new dependency uses a license not on this list, the check will fail. Open an issue if you believe a license should be added.

## Release Security

### Trivy Container Scan

After every release, Trivy scans the published Docker image for `CRITICAL` and `HIGH` severity vulnerabilities. Results are uploaded to the GitHub Security tab as SARIF.

### Supply Chain Verification

Every release includes cosign signatures, SBOMs (via syft), and GitHub build provenance attestations.
See [Verification](../getting-started/verification.md) for details.

### OpenSSF Scorecard

The project is evaluated weekly by [OpenSSF Scorecard](https://scorecard.dev/) for security best practices. Results are uploaded as SARIF to the GitHub Security tab.

## Local Security

### Pre-commit Hooks

The `.pre-commit-config.yaml` includes security-relevant hooks:

- **detect-private-key:** Catches PEM-style private keys
- **gitleaks:** Broader pattern matching for API keys, tokens, and other secrets

Install with:

```bash
pip install pre-commit
pre-commit install
```

## Interpreting Findings

### govulncheck

If govulncheck reports a vulnerability, check whether it affects your usage:

```bash
# Run locally to see details
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

The output shows which vulnerable functions are called. If the vulnerability is in an unused code path, govulncheck will not report it.

### gitleaks

If gitleaks flags a false positive, add an inline comment `# gitleaks:allow` or configure a `.gitleaksignore` file. Never commit actual secrets; rotate them immediately if found.

### Trivy

Review findings in the GitHub Security tab under "Code scanning alerts". Filter by severity to prioritize CRITICAL fixes first.
