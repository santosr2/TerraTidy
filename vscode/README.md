# TerraTidy for Visual Studio Code

TerraTidy is a comprehensive quality platform for Terraform and Terragrunt.
This extension brings TerraTidy's powerful formatting, style checking, linting, and policy enforcement capabilities directly into VS Code.

## Features

- **Language Server Protocol (LSP) Integration**: Fast, real-time analysis as you type
- **Real-time Diagnostics**: See issues highlighted in your editor instantly
- **Auto-formatting**: Format on save or on demand via standard VSCode format command
- **Code Actions**: Quick fixes for common style and formatting issues
- **Hover Documentation**: View rule documentation and configuration hints
- **Workspace Configuration**: Dynamic configuration updates without restart
- **Multi-engine Support**: Enable/disable fmt, style, lint, and policy engines independently

## Requirements

- [TerraTidy CLI](https://github.com/santosr2/TerraTidy) must be installed and available in your PATH
- VS Code 1.110.0 or higher

### Installing TerraTidy CLI

```bash
# Using Go
go install github.com/santosr2/TerraTidy/cmd/terratidy@latest

# Using Homebrew (macOS)
brew tap santosr2/tap https://github.com/santosr2/TerraTidy
brew install santosr2/tap/terratidy

# Download from releases
# https://github.com/santosr2/TerraTidy/releases
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
| `TerraTidy: Initialize Configuration` | Create a new .terratidy.yaml configuration file |
| `TerraTidy: Show Output` | Show the TerraTidy language server output/logs |
| `TerraTidy: Restart Language Server` | Restart the LSP server (useful after config changes) |

**Note**: Formatting, diagnostics, and code actions are provided automatically by the LSP server
and are accessed via standard VSCode features (Format Document command, Problems panel, Quick Fix lightbulb).

## Usage

### Automatic Diagnostics

The extension uses push diagnostics to analyze your Terraform/HCL files in real-time.
The LSP server automatically sends diagnostics when you open, edit, or save a file. Issues appear:

- As colored underlines in the editor (red=error, yellow=warning, blue=info)
- In the Problems panel (`Cmd+Shift+M` / `Ctrl+Shift+M`)
- With detailed messages on hover

The server runs style and lint engines. Rule overrides from `.terratidy.yaml` are respected:

```yaml
overrides:
  rules:
    style.resource-name-matches-type:
      enabled: true
      severity: warning
```

### Formatting

Format the current file:

1. Use the standard Format Document command (`Shift+Alt+F` on Windows/Linux, `Shift+Option+F` on macOS)
2. Or enable format-on-save in settings

### Code Actions

When you see a diagnostic (underlined issue):

1. Click the lightbulb icon that appears
2. Or press `Cmd+.` (macOS) / `Ctrl+.` (Windows/Linux)
3. Select a quick fix from the menu

### Initialize Configuration

To create a `.terratidy.yaml` configuration file:

1. Press `Cmd+Shift+P` (macOS) or `Ctrl+Shift+P` (Windows/Linux)
2. Type "TerraTidy: Initialize Configuration"
3. Press Enter

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

## Development

```bash
cd vscode
bun install            # Install dependencies
bun run compile        # Build extension
bun run lint           # Lint with Biome
bun run test           # Run integration tests (requires VS Code)
bun run package        # Package .vsix
```

## Contributing

Contributions are welcome! Please see the [TerraTidy repository](https://github.com/santosr2/TerraTidy) for contribution guidelines.

## Changelog

See the [project changelog](CHANGELOG.md) for release history.

## License

MIT License - see [LICENSE](LICENSE) for details.
