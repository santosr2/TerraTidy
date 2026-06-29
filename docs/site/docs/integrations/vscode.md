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

#### Debouncing

To avoid running expensive diagnostics on every keystroke, the LSP server
debounces document changes. When you type, diagnostics are deferred until
typing pauses for 500ms. This provides a smooth editing experience while
still catching issues quickly.

#### Configuration Auto-Reload

The LSP server watches your `.terratidy.yaml` and any imported configuration
files. When you modify and save a config file, the server automatically
reloads the configuration and republishes diagnostics for all open documents.
No need to restart VS Code or the language server.

Config file changes are also debounced (100ms) to handle rapid file events
from some editors on save.

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

### Severity Threshold

`terratidy.severityThreshold` is forwarded to the language server only when
you have written the key into your workspace, workspace folder, or user
`settings.json`. If you have not set it there, the extension omits the value
and the LSP server falls back to `severity_threshold:` in `.terratidy.yaml`
(see [Configuration](../getting-started/configuration.md#severity_threshold)).

To let VS Code control the threshold instead of `.terratidy.yaml`, add the
key to one of those `settings.json` files. Even setting it to the value
shown in the Settings UI counts as an explicit override and takes precedence
over `.terratidy.yaml`.

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

### Language Server Stopped Unexpectedly

If the LSP server crashes or exits unexpectedly, a warning notification appears with options:

- **Restart** - restarts the language server immediately
- **View Logs** - opens the output channel to inspect error details

You can also restart manually using the `TerraTidy: Restart Language Server` command.

If the server encounters repeated errors (5+ in quick succession), it will shut down automatically to avoid spamming the output channel. Check the logs for the underlying cause.

### Performance Issues

For large projects:

```json
{
  "terratidy.engines.lint": false
}
```

Run checks manually when needed.

## Versioning

The VS Code extension version is managed independently from the main TerraTidy
version to comply with VS Code Marketplace requirements.

### Version Scheme

VS Code Marketplace only supports numeric versions (X.Y.Z) without semver
pre-release suffixes like `-alpha.4`. The extension is published alongside
each TerraTidy CLI release but uses a separate version number:

| Component | Rule |
|-----------|------|
| major.minor | Matches TerraTidy CLI major.minor |
| patch | Incremented manually via `mise run vscode:bump:patch` before each publish |
| Pre-release | Indicated via `--pre-release` flag, not version string |

### Version Mapping

| TerraTidy CLI | VS Code Extension | Marketplace Status |
|---------------|-------------------|-------------------|
| 0.2.0-alpha.4 | 0.2.0 | Pre-release |
| 0.2.0-alpha.5 | 0.2.1 | Pre-release |
| 0.2.0 | 0.2.2 | Stable |
| 0.3.0-alpha.1 | 0.3.0 | Pre-release |

### Installing Pre-release Versions

To install pre-release versions from the Marketplace:

1. Open the TerraTidy extension page in VS Code
2. Click the dropdown arrow next to "Install"
3. Select "Install Pre-Release Version"

Or via command line:

```bash
code --install-extension santosr2.vscode-terratidy --pre-release
```

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
