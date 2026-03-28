# Policy Rules

Built-in policy rules enforced by the policy engine using OPA/Rego.

These rules are always available and do not require custom policy files. Enable the policy
engine to use them:

```yaml
engines:
  policy:
    enabled: true
```

## Security Rules

### no-public-ssh

Detects security groups that allow SSH access from the internet.

| Property | Value |
|----------|-------|
| Rule ID | `policy.no-public-ssh` |
| Default Severity | Error |
| Applies To | `aws_security_group` |

Flags `aws_security_group` resources with ingress rules allowing port 22 from `0.0.0.0/0`.

### no-public-s3

Detects S3 buckets with public-read ACL.

| Property | Value |
|----------|-------|
| Rule ID | `policy.no-public-s3` |
| Default Severity | Error |
| Applies To | `aws_s3_bucket` |

Flags `aws_s3_bucket` resources where `acl` is set to `public-read`.

### no-public-rds

Detects publicly accessible RDS instances.

| Property | Value |
|----------|-------|
| Rule ID | `policy.no-public-rds` |
| Default Severity | Error |
| Applies To | `aws_db_instance` |

Flags `aws_db_instance` resources where `publicly_accessible` is `true`.

## Terraform Best Practices

### required-terraform-block

Ensures a `terraform` block exists in the module.

| Property | Value |
|----------|-------|
| Rule ID | `policy.required-terraform-block` |
| Default Severity | Warning |
| Applies To | All modules |

### required-version

Ensures `required_version` is specified in the `terraform` block.

| Property | Value |
|----------|-------|
| Rule ID | `policy.required-version` |
| Default Severity | Warning |
| Applies To | `terraform` block |

### required-providers

Ensures providers used in the module have version constraints in `required_providers`.

| Property | Value |
|----------|-------|
| Rule ID | `policy.required-providers` |
| Default Severity | Warning |
| Applies To | `terraform.required_providers` |

### required-tags

Ensures taggable resources have a `tags` attribute.

| Property | Value |
|----------|-------|
| Rule ID | `policy.required-tags` |
| Default Severity | Warning |
| Applies To | `aws_instance`, `aws_s3_bucket` |

### module-version

Ensures external modules specify a `version` constraint.

| Property | Value |
|----------|-------|
| Rule ID | `policy.module-version` |
| Default Severity | Warning |
| Applies To | Module blocks with non-local sources |

Local modules (paths starting with `./` or `../`) are excluded.

## Rule Summary

| Rule | Severity | Description |
|------|----------|-------------|
| `no-public-ssh` | Error | Security groups cannot allow SSH from 0.0.0.0/0 |
| `no-public-s3` | Error | S3 buckets cannot have public-read ACL |
| `no-public-rds` | Error | RDS instances cannot be publicly accessible |
| `required-terraform-block` | Warning | Terraform block must exist |
| `required-version` | Warning | `required_version` must be specified |
| `required-providers` | Warning | Providers must have version constraints |
| `required-tags` | Warning | Resources should have tags |
| `module-version` | Warning | External modules should have version constraints |
