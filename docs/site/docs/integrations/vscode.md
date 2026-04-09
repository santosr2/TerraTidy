# VS Code Integration

TerraTidy provides a VS Code extension for seamless integration with your editor.

## Installation

### From Marketplace

Install from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=santosr2.vscode-terratidy):

1. Open VS Code
2. Go to Extensions (Ctrl+Shift+X / Cmd+Shift+X)
3. Search for "TerraTidy"
4. Click Install

Or install via command line:

```bash
code --install-extension santosr2.vscode-terratidy
```

### From VSIX

```bash
# Download the extension
wget https://github.com/santosr2/TerraTidy/releases/latest/download/terratidy.vsix

# Install
code --install-extension terratidy.vsix
```

## Requirements

- TerraTidy CLI must be installed and in your PATH
- VS Code 1.110.0 or higher

### Syntax Highlighting

TerraTidy does not provide syntax highlighting. Install the
[HashiCorp Terraform](https://marketplace.visualstudio.com/items?itemName=HashiCorp.terraform)
extension for Terraform syntax highlighting, or another HCL extension for
Terragrunt and other HCL files. TerraTidy is designed to work alongside
these extensions without conflicts.

## Features

### Real-time Diagnostics

Issues are highlighted as you type using push diagnostics. The LSP server
automatically sends diagnostics when you open, edit, or save a file:

- Errors shown with red squiggly underlines
- Warnings with yellow underlines
- Info with blue underlines

The server runs lint and style engines to produce diagnostics. Rule overrides
from `.terratidy.yaml` are respected, including plugin rules loaded from
configured plugin directories (see [Configuration](../getting-started/configuration.md)).

### Document Formatting

Format Terraform and HCL files using `hclwrite`:

```json
{
  "[terraform]": {
    "editor.defaultFormatter": "santosr2.vscode-terratidy",
    "editor.formatOnSave": true
  }
}
```

### Quick Fixes

Click the lightbulb icon or press `Ctrl+.` / `Cmd+.` to apply formatting
fixes for diagnostics. When a file has formatting issues, code actions
offer to reformat the document.

## Commands

Access via Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`):

| Command | Description |
|---------|-------------|
| `TerraTidy: Initialize Configuration` | Create .terratidy.yaml |
| `TerraTidy: Show Output` | Show output channel |
| `TerraTidy: Restart Language Server` | Restart the LSP server |

## Settings

Configure in VS Code settings (`Ctrl+,` / `Cmd+,`):

```json
{
  // Path to terratidy executable (if not in PATH)
  "terratidy.executablePath": "",

  // Path to configuration file
  "terratidy.configPath": "",

  // Configuration profile to use
  "terratidy.profile": "",

  // Run checks on save (default: false)
  "terratidy.runOnSave": false,

  // Format on save (default: false)
  "terratidy.formatOnSave": false,

  // Auto-fix on save
  "terratidy.fixOnSave": false,

  // Enable/disable engines
  "terratidy.engines.fmt": true,
  "terratidy.engines.style": true,
  "terratidy.engines.lint": true,
  "terratidy.engines.policy": false,

  // Minimum severity to show
  "terratidy.severityThreshold": "warning",

  // LSP server trace level (for debugging)
  "terratidy.trace.server": "off"
}
```

## Workspace Settings

Create `.vscode/settings.json` for project-specific settings:

```json
{
  "terratidy.profile": "development",
  "terratidy.engines.policy": true,
  "terratidy.configPath": ".terratidy.yaml"
}
```

## Keyboard Shortcuts

Default shortcuts:

| Shortcut | Command |
|----------|---------|
| `Shift+Alt+F` / `Shift+Option+F` | Format Document |
| `Ctrl+.` / `Cmd+.` | Quick Fix |

Customize in Keyboard Shortcuts (`Ctrl+K Ctrl+S`):

```json
{
  "key": "ctrl+alt+t",
  "command": "terratidy.restartServer",
  "when": "editorLangId == terraform"
}
```

## Troubleshooting

### TerraTidy Not Found

Check the output channel for errors:

1. Open Command Palette
2. Run "TerraTidy: Show Output"

Verify installation:

```bash
which terratidy
terratidy version
```

Set explicit path:

```json
{
  "terratidy.executablePath": "/usr/local/bin/terratidy"
}
```

### No Diagnostics Appearing

1. Check file is recognized as Terraform (`.tf`) or HCL (`.hcl`)
2. Verify configuration file is valid
3. Check severity threshold setting
4. Look at output channel for errors

### Performance Issues

For large projects:

```json
{
  "terratidy.runOnSave": false,
  "terratidy.engines.lint": false
}
```

Run checks manually when needed.

## Limitations

### Engine Selection for Diagnostics

Document formatting (`Format Document` command, `Shift+Alt+F`) works
regardless of engine settings. For real-time diagnostics (squiggly
underlines), the `engines.style` and `engines.lint` toggles control
which engines run. Disabling an engine prevents its diagnostics from
appearing.

The `engines.fmt` toggle does not affect diagnostics since formatting
is handled separately via the `Format Document` command. The
`engines.policy` toggle is accepted but policy diagnostics are not
yet implemented in the LSP server; policy engine diagnostics are
planned for a future release.

### Multi-root Workspaces

The LSP server currently uses only the first workspace folder's root.
Diagnostics and configuration are scoped to that folder. Multi-root
workspace support is planned for a future release.

### Remote Development

The extension sets `extensionKind: ["workspace"]` so it runs on the
remote side when using Remote Development (SSH, Containers, WSL).
Ensure TerraTidy is installed in the remote environment.

### File Activation

The extension activates on file patterns (`*.tf`, `*.tfvars`, `*.hcl`)
rather than language IDs. This ensures it works regardless of which
syntax highlighting extension is installed, but also means features
like `Format Document` may require explicitly selecting TerraTidy as
the formatter via `Editor: Format Document With...` if another
extension also registers for these file types.
