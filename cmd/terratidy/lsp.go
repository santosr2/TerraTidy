package main

import (
	"os"

	"github.com/santosr2/terratidy/internal/lsp"
	"github.com/spf13/cobra"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Language Server Protocol server",
	Long: `Start the TerraTidy Language Server Protocol (LSP) server.

The LSP server communicates via stdin/stdout using the JSON-RPC protocol
as defined by the Language Server Protocol specification.

This allows TerraTidy to integrate with any editor that supports LSP,
including:
  - Visual Studio Code (with TerraTidy extension)
  - Neovim (with nvim-lspconfig)
  - Emacs (with lsp-mode)
  - Sublime Text (with LSP package)
  - Any other LSP-compatible editor

Example configurations:

Neovim (lua):
  require('lspconfig').terratidy.setup{
    cmd = { "terratidy", "lsp" },
    filetypes = { "terraform", "hcl" },
  }

VSCode (settings.json):
  Use the TerraTidy extension which handles this automatically.

The server provides:
  - Real-time diagnostics
  - Document formatting
  - Code actions for fixable issues`,
	Run: func(cmd *cobra.Command, args []string) {
		server := lsp.NewServer(os.Stdin, os.Stdout)
		if err := server.Run(); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
}
