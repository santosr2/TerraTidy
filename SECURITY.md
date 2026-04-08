# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :warning: Deprecated at v0.2.0 release (broken module path) |

## Reporting a Vulnerability

We take the security of TerraTidy seriously. If you believe you have found a security vulnerability, please report it responsibly.

### How to Report

Please report security vulnerabilities via one of the following methods:

1. **GitHub Security Advisories** (Preferred): Use [GitHub's private vulnerability reporting](https://github.com/santosr2/TerraTidy/security/advisories/new)

2. **GitHub Issues**: For non-sensitive security questions, open a [GitHub Issue](https://github.com/santosr2/TerraTidy/issues) with the "security" label

### What to Include

Please include the following information in your report:

- Type of vulnerability (e.g., command injection, path traversal, etc.)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 1 week
- **Resolution Target**: Within 30 days for critical issues

### What to Expect

1. We will acknowledge your report within 48 hours
2. We will investigate and provide an initial assessment
3. We will work with you to understand and resolve the issue
4. We will credit you in the security advisory (unless you prefer to remain anonymous)

## Trust Model

TerraTidy operates with a clear trust boundary:

**Trusted:**

- Configuration files (`.terratidy.yaml`) — read but not executed
- HCL/Terraform files in your workspace — parsed, never executed
- Built-in engines and rules — compiled into the binary

**Untrusted (require verification):**

- Go plugins (`.so` files) — executable code, verify checksums
- Bash rules (`.sh` files) — executable scripts, verify checksums
- External tools (TFLint) — validated path only, not content

**Security boundaries:**

- LSP server: Confined to workspace directory, rejects path traversal
- GitHub Action: Validates inputs against safe character patterns
- Plugin system: SHA256 verification available (warn-only by default)

## Security Best Practices

When using TerraTidy:

### Configuration Files

- Store `.terratidy.yaml` in version control
- Avoid including sensitive data in configuration files
- Use environment variables for secrets if needed in custom rules

### Custom Rules

- Review third-party rules before using them
- Only run custom rules from trusted sources
- Sandbox custom Bash rules when possible

### Plugin Integrity Verification

TerraTidy supports SHA256 checksum verification for Go plugins and Bash rules via
the `plugins.verify_integrity` configuration option (enabled by default).

**How it works:**

1. Create a manifest file `.terratidy-plugins.sha256` in your plugin directory
2. The manifest uses standard `sha256sum` format: `<hash>  <filename>`
3. TerraTidy verifies each plugin/script against the manifest before loading

**Current behavior (warn-only mode):**

- If verification fails, a warning is logged but the plugin still loads
- This allows gradual adoption without breaking existing setups
- Future releases will enforce verification by default

**Security considerations for Bash rules:**

Bash rules execute arbitrary shell commands. Unlike Go plugins (which are compiled
and can be code-reviewed), Bash rules are scripts that run with full shell access.
This creates additional risks:

- **Command injection**: Malicious scripts could execute harmful commands
- **Data exfiltration**: Scripts have access to environment variables and files
- **Privilege escalation**: If TerraTidy runs with elevated privileges, so do scripts

**Mitigations:**

1. Enable `plugins.verify_integrity: true` (default) and maintain checksums
2. Review all Bash rule scripts before adding them to your project
3. Run TerraTidy with minimal required privileges
4. Prefer Go plugins or YAML rules over Bash rules when possible
5. Use `plugins.verify_integrity: false` only in trusted environments

### LSP Server

The Language Server Protocol implementation includes several security measures:

- **Path traversal protection**: File URIs are validated to stay within the workspace
- **Resource limits**: Max 1000 documents, 10 MB per document, 10 concurrent diagnostics
- **Content-Length cap**: Messages larger than 10 MB are rejected
- **Session isolation**: Temp files use per-session directories with automatic cleanup

### CI/CD Integration

- Use pinned versions in GitHub Actions (`@v0.2.0` not `@latest`)
- Review the action permissions required
- Use the `fail-on-error` input appropriately
- Action inputs are validated against safe character patterns

### Docker Usage

- Use specific version tags, not `latest` in production
- Run containers with minimal privileges
- Mount only necessary directories

## Scope

The following are in scope for security reports:

- TerraTidy CLI binary
- Built-in engines (fmt, style, lint, policy)
- GitHub Action
- LSP server
- VSCode extension

The following are out of scope:

- Third-party plugins or custom rules
- Issues in dependencies (report to the respective project)
- Social engineering attacks
- Physical attacks

## Recognition

We appreciate the security research community's efforts in helping keep TerraTidy secure. Contributors who report valid security issues will be:

- Credited in the security advisory (unless they prefer anonymity)
- Listed in our security acknowledgments
- Thanked publicly (with permission)

## Security Updates

Security updates are released as patch versions. We recommend:

1. Subscribing to [GitHub releases](https://github.com/santosr2/TerraTidy/releases) for notifications
2. Following the [CHANGELOG](CHANGELOG.md) for security-related changes
3. Upgrading promptly when security patches are released
