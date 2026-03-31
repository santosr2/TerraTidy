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
}
