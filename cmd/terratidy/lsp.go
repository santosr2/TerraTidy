package main

import (
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/lsp"
	"github.com/spf13/cobra"
)

var (
	lspLogLevel string
	lspLogFile  string
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
    cmd = { "terratidy", "lsp", "--log-level", "debug", "--log-file", "/tmp/terratidy-lsp.log" },
    filetypes = { "terraform", "hcl" },
  }

VSCode (settings.json):
  Use the TerraTidy extension which handles this automatically.

The server provides:
  - Real-time diagnostics
  - Document formatting
  - Code actions for fixable issues`,
	RunE: func(_ *cobra.Command, _ []string) error {
		server := lsp.NewServer(os.Stdin, os.Stdout)

		server.SetLogLevel(lsp.ParseLogLevel(lspLogLevel))

		if lspLogFile != "" {
			if err := server.SetLogFile(lspLogFile); err != nil {
				return fmt.Errorf("setting log file: %w", err)
			}
		}

		if err := server.Run(); err != nil {
			return fmt.Errorf("running LSP server: %w", err)
		}
		return nil
	},
}

func init() {
	lspCmd.Flags().StringVar(&lspLogLevel, "log-level", "info",
		"Log level: off, error, warn, info, debug")
	lspCmd.Flags().StringVar(&lspLogFile, "log-file", "",
		"Path to log file (defaults to stderr)")
	rootCmd.AddCommand(lspCmd)
}
