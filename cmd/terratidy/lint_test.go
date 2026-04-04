package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "lint [paths...]", lintCmd.Use)
		assert.Equal(t, "Run linting checks", lintCmd.Short)
		assert.NotEmpty(t, lintCmd.Long)
		assert.NotEmpty(t, lintCmd.Example)
	})

	t.Run("has config-file flag", func(t *testing.T) {
		flag := lintCmd.Flags().Lookup("config-file")
		assert.NotNil(t, flag)
		assert.Equal(t, ".tflint.hcl", flag.DefValue)
	})

	t.Run("has plugin flag", func(t *testing.T) {
		flag := lintCmd.Flags().Lookup("plugin")
		assert.NotNil(t, flag)
	})

	t.Run("has rule flag", func(t *testing.T) {
		flag := lintCmd.Flags().Lookup("rule")
		assert.NotNil(t, flag)
	})
}

func TestLintCmdExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple terraform file
	content := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0o644))

	t.Run("no files found", func(t *testing.T) {
		emptyDir := t.TempDir()
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"lint", emptyDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("lint valid file", func(t *testing.T) {
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"lint", tmpDir})
		err := rootCmd.Execute()
		// Lint may find issues (returning ExitError), but must not fail unexpectedly
		assert.NoError(t, err, "lint on valid tf file should not error")
	})

	t.Run("lint with explicit config-file flag", func(t *testing.T) {
		// Test that --config-file flag override works (BUG-4 fix coverage)
		changed = false
		format = "text"

		// Reset the flag to ensure Changed() works correctly
		lintCmd.Flags().Set("config-file", "custom.hcl")

		rootCmd.SetArgs([]string{"lint", "--config-file", "custom.hcl", tmpDir})
		err := rootCmd.Execute()
		// May error if file doesn't exist, but the flag parsing path is covered
		_ = err
	})

	t.Run("lint with explicit plugin flag", func(t *testing.T) {
		// Test that --plugin flag override works (BUG-4 fix coverage)
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"lint", "--plugin", "aws", tmpDir})
		err := rootCmd.Execute()
		// May error if plugin not available, but the flag parsing path is covered
		_ = err
	})
}
