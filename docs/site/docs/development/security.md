# Security

TerraTidy's CI/CD pipeline includes automated security scanning at multiple stages. Scanners that produce SARIF upload it to the repository's **Security → Code scanning** tab.

## CI Security Scanning

The security workflow (`.github/workflows/security.yml`) runs on every push to main and every pull request targeting main, covering all changes rather than only Go source edits.

### govulncheck

Scans Go dependencies against the [Go vulnerability database](https://vuln.go.dev/). Fails the build if any known vulnerabilities are found in code paths actually used by TerraTidy.

### dependency-review

Runs on pull requests only. Reviews dependency changes (additions, updates, removals) and flags any with known security advisories before they are merged.

### gitleaks

Scans the repository for accidentally committed secrets (API keys, tokens, private keys, etc.). Runs with full git history to catch secrets in any commit.

### go-licenses

Verifies all dependencies use approved open-source licenses:

- Apache-2.0, BSD-2-Clause, BSD-2-Clause-FreeBSD, BSD-3-Clause, MIT, MPL-2.0, Unicode-DFS-2016

If a new dependency uses a license not on this list, the check will fail. Open an issue if you believe a license should be added.

### zizmor

Audits the GitHub Actions workflows themselves for security problems: unpinned actions, overly broad
token permissions, credentials persisted after checkout, and untrusted input reaching a shell.
Findings are uploaded to Code scanning.

### osv-scanner

Scans the VS Code extension's dependencies (`vscode/bun.lock`) against the
[OSV database](https://osv.dev/); govulncheck covers the Go side. Findings are uploaded to Code
scanning. A scanner error, such as the lockfile moving or failing to parse, fails the job rather
than passing silently.

### api-compat

Runs on pull requests that change `pkg/sdk/`. It uses `gorelease` to report incompatible changes to
the public SDK against the latest release. It is informational for now: before v1.0 the SDK makes no
compatibility guarantee.

### CodeQL

The CodeQL workflow (`.github/workflows/codeql.yml`) analyzes the Go code and the GitHub Actions
workflows on every pull request, on every push to main, and weekly. Unlike the pattern-based
scanners above, it follows data through the code, so it can catch a problem that spans several
functions, such as untrusted input reaching a file path or shell command. For workflows, it traces
values through composite actions and reusable workflows.

## Container Scanning

Trivy scans the Docker image in two places. Results are uploaded to Code scanning.

- **Container test workflow** (`.github/workflows/container-test.yml`): scans the image built from
  the current code, on pull requests and pushes to main that change the image or the Go code.
- **Release workflow**: scans the published image after each release.

Both report every severity. Advisories that have been reviewed and accepted are listed in
`.trivyignore`, each with the reason next to it. An entry is removed as soon as a fix is available.

## Release Security

### Supply Chain Verification

Every release includes cosign signatures, SBOMs (via syft), and GitHub build provenance attestations.
See [Verification](../getting-started/verification.md) for details.

### OpenSSF Scorecard

The project is evaluated by [OpenSSF Scorecard](https://scorecard.dev/) weekly and on every push to main. Results are uploaded as SARIF to the GitHub Security tab.

## Local Security

### Pre-commit Hooks

The `.pre-commit-config.yaml` includes security-relevant hooks:

- **detect-private-key:** Catches PEM-style private keys
- **gitleaks:** Broader pattern matching for API keys, tokens, and other secrets
- **zizmor:** The same workflow audit CI runs, so problems surface before you push

Install with:

```bash
mise install          # installs pre-commit along with the rest of the toolchain
pre-commit install
```

## Interpreting Findings

### govulncheck

If govulncheck reports a vulnerability, check whether it affects your usage:

```bash
# Run locally to see details
mise run vuln
```

Use the mise task rather than installing govulncheck separately: it pins a govulncheck build that
can read the Go version the project uses, and an older build fails before it scans anything.

The output shows which vulnerable functions are called. If the vulnerability is in an unused code path, govulncheck will not report it.

### gitleaks

If gitleaks flags a false positive, add an inline comment `# gitleaks:allow` or configure a `.gitleaksignore` file. Never commit actual secrets; rotate them immediately if found.

### Trivy

Review findings in the GitHub Security tab under "Code scanning alerts". Filter by severity to prioritize CRITICAL fixes first.

Upgrade the affected dependency when a fix exists. When no fix exists, check whether TerraTidy
actually reaches the vulnerable code. If it doesn't, add the advisory ID to `.trivyignore` together
with a comment explaining why, so the next person can tell when the entry can go.
