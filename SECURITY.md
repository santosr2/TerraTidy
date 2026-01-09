# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of TerraTidy seriously. If you believe you have found a security vulnerability, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **GitHub Security Advisories** (Preferred): Use [GitHub's private vulnerability reporting](https://github.com/santosr2/terratidy/security/advisories/new)

2. **Email**: Contact the maintainers directly (include "SECURITY" in the subject line)

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

### CI/CD Integration

- Use pinned versions in GitHub Actions (`@v0.2.0` not `@latest`)
- Review the action permissions required
- Use the `fail-on-error` input appropriately

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

1. Subscribing to [GitHub releases](https://github.com/santosr2/terratidy/releases) for notifications
2. Following the [CHANGELOG](CHANGELOG.md) for security-related changes
3. Upgrading promptly when security patches are released
