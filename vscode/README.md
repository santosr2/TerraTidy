# TerraTidy for Visual Studio Code

TerraTidy is a comprehensive quality platform for Terraform and Terragrunt.
This extension brings TerraTidy's powerful formatting, style checking, linting, and policy enforcement capabilities directly into VS Code.

## Features

- **Format on Save**: Automatically format your Terraform files when saving
- **Real-time Diagnostics**: See issues highlighted in your editor as you type
- **Quick Fixes**: One-click fixes for common style and formatting issues
- **Integrated Commands**: Run TerraTidy checks directly from the command palette
- **Context Menu Integration**: Right-click on files to run TerraTidy commands

## Requirements

- [TerraTidy CLI](https://github.com/santosr2/terratidy) must be installed and available in your PATH
- VS Code 1.85.0 or higher

### Installing TerraTidy CLI

```bash
# Using Go
go install github.com/santosr2/terratidy/cmd/terratidy@latest

# Using Homebrew (macOS)
brew install santosr2/tap/terratidy

# Download from releases
# https://github.com/santosr2/terratidy/releases
```

## Extension Settings

This extension contributes the following settings:

| Setting | Default | Description |
|---------|---------|-------------|
| `terratidy.executablePath` | `""` | Path to the terratidy executable |
| `terratidy.configPath` | `""` | Path to terratidy configuration file |
| `terratidy.profile` | `""` | Configuration profile to use |
| `terratidy.runOnSave` | `false` | Run TerraTidy checks when saving |
| `terratidy.formatOnSave` | `false` | Format files on save |
| `terratidy.fixOnSave` | `false` | Auto-fix issues on save |
| `terratidy.engines.fmt` | `true` | Enable format engine |
| `terratidy.engines.style` | `true` | Enable style engine |
| `terratidy.engines.lint` | `true` | Enable lint engine |
| `terratidy.engines.policy` | `false` | Enable policy engine |
| `terratidy.severityThreshold` | `warning` | Minimum severity to show |

## Commands

The extension provides the following commands:

| Command | Description |
|---------|-------------|
| `TerraTidy: Run All Checks` | Run all enabled checks on the current file |
| `TerraTidy: Format Document` | Format the current Terraform file |
| `TerraTidy: Lint` | Run linting checks |
| `TerraTidy: Check Style` | Run style checks |
| `TerraTidy: Fix All Issues` | Auto-fix all fixable issues |
| `TerraTidy: Initialize Configuration` | Create a new .terratidy.yaml |
| `TerraTidy: Show Output` | Show the TerraTidy output channel |

## Usage

### Running Checks

1. Open a Terraform (`.tf`) or HCL (`.hcl`) file
2. Press `Cmd+Shift+P` (macOS) or `Ctrl+Shift+P` (Windows/Linux)
3. Type "TerraTidy" and select a command

### Configuring TerraTidy

Create a `.terratidy.yaml` file in your workspace root:

```yaml
version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

severity_threshold: warning
```

### Using Profiles

You can define profiles in your configuration:

```yaml
profiles:
  ci:
    engines:
      fmt: { enabled: true }
      style: { enabled: true }
      lint: { enabled: true }
      policy: { enabled: true }

  development:
    engines:
      fmt: { enabled: true }
      style: { enabled: true }
      lint: { enabled: false }
      policy: { enabled: false }
```

Then select the profile in VS Code settings:

```json
{
  "terratidy.profile": "development"
}
```

## Recommended Settings

For the best experience, add these settings to your `settings.json`:

```json
{
  "terratidy.runOnSave": true,
  "terratidy.formatOnSave": true,
  "[terraform]": {
    "editor.defaultFormatter": "santosr2.terratidy",
    "editor.formatOnSave": true
  },
  "[hcl]": {
    "editor.defaultFormatter": "santosr2.terratidy",
    "editor.formatOnSave": true
  }
}
```

## Troubleshooting

### TerraTidy not found

Make sure TerraTidy is installed and in your PATH:

```bash
which terratidy
terratidy --version
```

Or specify the full path in settings:

```json
{
  "terratidy.executablePath": "/usr/local/bin/terratidy"
}
```

### No diagnostics appearing

1. Check the TerraTidy output channel for errors
2. Verify your configuration file is valid
3. Make sure the file is recognized as Terraform/HCL

## Contributing

Contributions are welcome! Please see the [TerraTidy repository](https://github.com/santosr2/terratidy) for contribution guidelines.

## License

MIT License - see [LICENSE](../LICENSE) for details.
