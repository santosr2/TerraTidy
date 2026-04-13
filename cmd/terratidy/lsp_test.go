package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: No TestLspCmdExecution test exists because the LSP server blocks on stdin,
// making it unsuitable for unit testing. The LSP protocol and server behavior are
// thoroughly tested in internal/lsp/server_test.go. These CLI tests verify command
// registration and flag wiring only.

func TestLspCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "lsp", lspCmd.Use)
		assert.Equal(t, "Start the Language Server Protocol server", lspCmd.Short)
		assert.NotEmpty(t, lspCmd.Long)
		// lspCmd intentionally has no Example field; usage examples are in Long description
		assert.Empty(t, lspCmd.Example)
	})

	t.Run("has log-level flag", func(t *testing.T) {
		flag := lspCmd.Flags().Lookup("log-level")
		require.NotNil(t, flag)
		assert.Equal(t, "info", flag.DefValue)
		assert.Contains(t, flag.Usage, "off")
		assert.Contains(t, flag.Usage, "error")
		assert.Contains(t, flag.Usage, "warn")
		assert.Contains(t, flag.Usage, "info")
		assert.Contains(t, flag.Usage, "debug")
	})

	t.Run("has log-file flag", func(t *testing.T) {
		flag := lspCmd.Flags().Lookup("log-file")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
		assert.Contains(t, flag.Usage, "log file")
	})

	t.Run("is registered on root command", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"lsp"})
		require.NoError(t, err)
		assert.Equal(t, "lsp", cmd.Name())
	})
}

func TestLspCmd_LogLevelFlag(t *testing.T) {
	// Test that flag setting updates the package-level variable that RunE reads.
	// ParseLogLevel behavior is tested in internal/lsp/server_test.go.
	tests := []struct {
		name  string
		value string
	}{
		{"off level", "off"},
		{"error level", "error"},
		{"warn level", "warn"},
		{"info level", "info"},
		{"debug level", "debug"},
		{"uppercase", "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to default before each test
			lspLogLevel = "info"
			t.Cleanup(func() { lspLogLevel = "info" })

			err := lspCmd.Flags().Set("log-level", tt.value)
			require.NoError(t, err)

			// Verify the package-level variable is updated (what RunE reads)
			assert.Equal(t, tt.value, lspLogLevel)
		})
	}
}

func TestLspCmd_LogFileFlag(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"absolute path", "/tmp/terratidy-lsp.log", "/tmp/terratidy-lsp.log"},
		{"relative path", "logs/lsp.log", "logs/lsp.log"},
		{"empty value", "", ""},
		{"path with spaces", "/tmp/my logs/terratidy.log", "/tmp/my logs/terratidy.log"},
		{"tilde path verbatim", "~/logs/lsp.log", "~/logs/lsp.log"}, // tilde expansion is shell's job
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to default before each test
			lspLogFile = ""
			t.Cleanup(func() { lspLogFile = "" })

			err := lspCmd.Flags().Set("log-file", tt.value)
			require.NoError(t, err)

			// Verify the package-level variable is updated (what RunE reads)
			assert.Equal(t, tt.expected, lspLogFile)
		})
	}
}

func TestLspCmd_InvalidLogLevel(t *testing.T) {
	// Invalid log levels are accepted at CLI level (Cobra doesn't validate string values).
	// The LSP server gracefully defaults to info for unknown values.
	// This test verifies the CLI accepts the input; server behavior is tested elsewhere.
	tests := []struct {
		name  string
		value string
	}{
		{"unknown level", "unknown"},
		{"numeric", "123"},
		{"typo", "infoo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lspLogLevel = "info"
			t.Cleanup(func() { lspLogLevel = "info" })

			// CLI accepts any string value
			err := lspCmd.Flags().Set("log-level", tt.value)
			require.NoError(t, err, "flag setting should accept any value")

			// Variable is updated even for invalid values
			assert.Equal(t, tt.value, lspLogLevel)
		})
	}
}
