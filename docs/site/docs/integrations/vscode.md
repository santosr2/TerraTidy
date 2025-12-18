# VS Code Integration

TerraTidy provides a VS Code extension for seamless integration with your editor.

## Installation

### From Marketplace

1. Open VS Code
2. Go to Extensions (Ctrl+Shift+X / Cmd+Shift+X)
3. Search for "TerraTidy"
4. Click Install

### From VSIX

```bash
# Download the extension
wget https://github.com/santosr2/terratidy/releases/latest/download/terratidy.vsix

# Install
code --install-extension terratidy.vsix
```

## Requirements

- TerraTidy CLI must be installed and in your PATH
- VS Code 1.85.0 or higher

## Features

### Real-time Diagnostics

Issues are highlighted as you type:

- Errors shown with red squiggly underlines
- Warnings with yellow underlines
- Info with blue underlines

### Format on Save

Enable automatic formatting:

```json
{
  "terratidy.formatOnSave": true,
  "[terraform]": {
    "editor.defaultFormatter": "santosr2.terratidy",
    "editor.formatOnSave": true
  }
}
```

### Quick Fixes

Click the lightbulb icon or press `Ctrl+.` / `Cmd+.` to see available fixes:

- Fix naming convention violations
- Reorder attributes
- Add missing descriptions

### Code Actions

Right-click in the editor for:

- Format Document
- Fix All Issues
- Run TerraTidy Check

## Commands

Access via Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`):

| Command | Description |
|---------|-------------|
| `TerraTidy: Run All Checks` | Run all enabled checks |
| `TerraTidy: Format Document` | Format current file |
| `TerraTidy: Lint` | Run linting only |
| `TerraTidy: Check Style` | Run style checks only |
| `TerraTidy: Fix All Issues` | Auto-fix all fixable issues |
| `TerraTidy: Initialize Configuration` | Create .terratidy.yaml |
| `TerraTidy: Show Output` | Show output channel |

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

  // Run checks on save
  "terratidy.runOnSave": true,

  // Format on save
  "terratidy.formatOnSave": true,

  // Auto-fix on save
  "terratidy.fixOnSave": false,

  // Enable/disable engines
  "terratidy.engines.fmt": true,
  "terratidy.engines.style": true,
  "terratidy.engines.lint": true,
  "terratidy.engines.policy": false,

  // Minimum severity to show
  "terratidy.severityThreshold": "warning"
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
| `Ctrl+Shift+F` / `Cmd+Shift+F` | Format Document |
| `Ctrl+.` / `Cmd+.` | Quick Fix |

Customize in Keyboard Shortcuts (`Ctrl+K Ctrl+S`):

```json
{
  "key": "ctrl+alt+t",
  "command": "terratidy.check",
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
terratidy --version
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

## Multi-root Workspaces

Each workspace folder can have its own configuration:

```text
project/
├── .vscode/
│   └── settings.json
├── module-a/
│   └── .terratidy.yaml
└── module-b/
    └── .terratidy.yaml
```

## Remote Development

The extension works with:

- Remote - SSH
- Remote - Containers
- WSL

Ensure TerraTidy is installed in the remote environment.
