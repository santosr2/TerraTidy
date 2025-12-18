# TerraTidy

A comprehensive quality platform for Terraform and Terragrunt.

## What is TerraTidy?

TerraTidy is an all-in-one code quality tool for Terraform and Terragrunt that provides:

- **Formatting** - Consistent code formatting based on HashiCorp style
- **Style Checking** - Enforce naming conventions and code organization
- **Linting** - Catch common mistakes and best practice violations
- **Policy Enforcement** - Use OPA/Rego policies for custom rules

## Key Features

- Single binary with no external dependencies
- Supports `.tf`, `.tfvars`, and `.hcl` files
- Auto-fix capability for many issues
- Multiple output formats (text, JSON, SARIF, HTML)
- IDE integrations (VS Code, LSP)
- CI/CD integrations (GitHub Actions, pre-commit)
- Extensible plugin system

## Quick Start

```bash
# Install TerraTidy
go install github.com/santosr2/terratidy/cmd/terratidy@latest

# Initialize configuration
terratidy init

# Run all checks
terratidy check

# Auto-fix issues
terratidy fix
```

## Why TerraTidy?

| Feature | terraform fmt | tflint | checkov | terratidy |
|---------|--------------|--------|---------|-----------|
| Formatting | Yes | No | No | Yes |
| Style Rules | No | Partial | No | Yes |
| Linting | No | Yes | Partial | Yes |
| Policy | No | No | Yes | Yes |
| Single Binary | Yes | Yes | No | Yes |
| Auto-fix | No | Partial | No | Yes |

## Getting Help

- [GitHub Issues](https://github.com/santosr2/terratidy/issues) - Report bugs and request features
- [Documentation](/) - Read the full documentation
- [Examples](https://github.com/santosr2/terratidy/tree/main/examples) - See example configurations

## License

TerraTidy is open source software licensed under the MIT License.
