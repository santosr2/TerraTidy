# Language Server Protocol (LSP)

TerraTidy provides an LSP server for integration with any editor that supports LSP.

## Overview

The TerraTidy LSP server provides:

- Real-time diagnostics
- Document formatting
- Code actions (quick fixes)

## Starting the Server

```bash
# Start LSP server
terratidy lsp

# With debug logging to a file
terratidy lsp --log-level debug --log-file /tmp/terratidy-lsp.log

# Log levels: off, error, warn, info (default), debug
terratidy lsp --log-level error
```

## Editor Integration

### Neovim

Using `nvim-lspconfig`:

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Define TerraTidy LSP
if not configs.terratidy then
  configs.terratidy = {
    default_config = {
      cmd = { 'terratidy', 'lsp', '--log-level', 'info' },
      filetypes = { 'terraform', 'hcl' },
      root_dir = lspconfig.util.root_pattern(
        '.terratidy.yaml',
        '.terraform',
        '.git'
      ),
      settings = {},
    },
  }
end

-- Setup
lspconfig.terratidy.setup({
  on_attach = function(client, bufnr)
    -- Enable formatting
    vim.api.nvim_buf_set_option(bufnr, 'formatexpr', 'v:lua.vim.lsp.formatexpr()')

    -- Keybindings
    local opts = { buffer = bufnr }
    vim.keymap.set('n', '<leader>f', vim.lsp.buf.format, opts)
    vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)
  end,
})
```

### Emacs

Using `lsp-mode`:

```elisp
(use-package lsp-mode
  :hook ((terraform-mode . lsp)
         (hcl-mode . lsp))
  :config
  (lsp-register-client
   (make-lsp-client
    :new-connection (lsp-stdio-connection '("terratidy" "lsp"))
    :major-modes '(terraform-mode hcl-mode)
    :server-id 'terratidy)))
```

Using `eglot`:

```elisp
(add-to-list 'eglot-server-programs
             '((terraform-mode hcl-mode) . ("terratidy" "lsp")))
```

### Sublime Text

Install LSP package, then configure:

```json
{
  "clients": {
    "terratidy": {
      "command": ["terratidy", "lsp"],
      "selector": "source.terraform, source.hcl",
      "initializationOptions": {}
    }
  }
}
```

### Helix

In `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "hcl"
scope = "source.hcl"
file-types = ["tf", "hcl"]
language-server = { command = "terratidy", args = ["lsp"] }
auto-format = true

[[language]]
name = "terraform"
scope = "source.terraform"
file-types = ["tf", "tfvars"]
language-server = { command = "terratidy", args = ["lsp"] }
auto-format = true
```

### Zed

In settings:

```json
{
  "languages": {
    "Terraform": {
      "language_servers": ["terratidy"]
    }
  },
  "lsp": {
    "terratidy": {
      "binary": {
        "path": "terratidy",
        "arguments": ["lsp"]
      }
    }
  }
}
```

## LSP Capabilities

### Text Document Synchronization

The server uses **full sync mode** (TextDocumentSyncKind = 1), meaning the client sends
the complete document content on every change. Save notifications include the document text.

- `textDocument/didOpen`
- `textDocument/didChange` (full content)
- `textDocument/didSave` (with text)
- `textDocument/didClose`

### Diagnostics

Diagnostics are published on:

- Document open
- Document change
- Document save

### Formatting

- `textDocument/formatting` - Format entire document

### Code Actions

- Quick fixes for diagnostics with associated rule codes

## Configuration

The LSP server reads configuration from:

1. `.terratidy.yaml` in workspace
2. Global `~/.terratidy/config.yaml`
3. Initialization options from client

### Initialization Options

The server reads these options from the client's `initializationOptions`:

```json
{
  "initializationOptions": {
    "profile": "development",
    "configPath": ".terratidy.yaml",
    "engines": {
      "fmt": true,
      "style": true,
      "lint": true,
      "policy": false
    },
    "severityThreshold": "warning"
  }
}
```

- `profile`: Applies a named profile from the config file
- `configPath`: Overrides the default config file path
- `severityThreshold`: Minimum severity level to report (info, warning, error)

## Troubleshooting

### Check Server Status

```bash
# Test if server starts
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | terratidy lsp
```

### Common Issues

**Server not starting:**

- Check TerraTidy is in PATH
- Verify configuration file syntax
- Check log file for errors

**No diagnostics:**

- Ensure file is saved
- Check engine configuration
- Verify severity threshold

**Slow response:**

- Disable lint engine for large projects
- Use a more restrictive profile
