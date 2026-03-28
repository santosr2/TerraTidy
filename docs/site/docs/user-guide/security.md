# Security

Security considerations when using TerraTidy.

## Policy Engine

The policy engine uses OPA (Open Policy Agent) v1.15.0 to evaluate Rego policies against
your Terraform configuration. Policies run in-process with read-only access to the parsed
HCL input.

Built-in policies cover common security checks:

- `no-public-ssh` - Blocks security groups allowing SSH from 0.0.0.0/0
- `no-public-s3` - Blocks S3 buckets with public-read ACL
- `no-public-rds` - Blocks publicly accessible RDS instances

See [Policy Rules](../rules/policy-rules.md) for the full list.

## Plugin Trust Model

Plugins run with the same privileges as the TerraTidy process. There is no sandboxing.

**Implications:**

- Go plugins (`.so`) execute compiled code with full process access
- Bash rules execute shell scripts with full shell access (30-second timeout)
- YAML rules are declarative and pattern-based (no code execution)

**Recommendations:**

- Only use plugins from trusted sources
- Review Bash rule scripts before installing them
- Use project-local plugin directories (`.terratidy/plugins/`) over global ones for
  better auditability
- Pin plugin versions in version control

## Secure CI/CD

### Principle of least privilege

Run TerraTidy with read-only access where possible:

```yaml
# GitHub Actions: only needs read access for checks
permissions:
  contents: read
  security-events: write  # Only if using SARIF upload
```

### Pin versions

Pin the TerraTidy version in CI to avoid unexpected behavior from upgrades:

```yaml
- uses: santosr2/terratidy@v0
  with:
    version: 'v0.2.0-alpha.3'
```

### Scan results

Use SARIF output for GitHub Code Scanning integration:

```yaml
- name: Run TerraTidy
  run: terratidy check --format sarif > results.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: results.sarif
```

## Environment Variables in Config

Configuration supports environment variable expansion. Avoid storing secrets directly
in `.terratidy.yaml`:

```yaml
# Good: reference env var
engines:
  policy:
    config:
      api_key: ${API_KEY}

# Bad: hardcoded secret
engines:
  policy:
    config:
      api_key: sk-1234567890
```
