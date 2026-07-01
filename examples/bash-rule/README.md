# Bash Rule Example

A bash script rule that detects hardcoded AWS account IDs in Terraform files.

## Usage

```bash
# Make executable
chmod +x no-hardcoded-account-id.sh

# Install to plugins directory
cp no-hardcoded-account-id.sh ~/.terratidy/plugins/
```

## How It Works

The script receives a file path as `$1`, scans for 12-digit numbers that look like AWS account IDs, and outputs JSON:

```json
{
  "findings": [
    {
      "file": "main.tf",
      "line": 5,
      "message": "Hardcoded AWS account ID detected; use a variable or data source",
      "severity": "warning"
    }
  ]
}
```

## Requirements

- `bash`, `grep`, `jq`
- Script must be executable (`chmod +x`)

## Plugin Integrity

The `.terratidy-plugins.sha256` manifest is committed and verified at load time. If you edit `no-hardcoded-account-id.sh`, regenerate the manifest from the repo root:

```bash
cd examples/bash-rule && sha256sum no-hardcoded-account-id.sh > .terratidy-plugins.sha256
```

A stale manifest currently produces a `verification failed` warning and the rule loads anyway; future releases will treat the mismatch as a hard error.
