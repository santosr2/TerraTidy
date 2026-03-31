package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStyleCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "style [paths...]", styleCmd.Use)
		assert.Equal(t, "Check and fix style issues", styleCmd.Short)
		assert.NotEmpty(t, styleCmd.Long)
		assert.NotEmpty(t, styleCmd.Example)
	})

	t.Run("has fix flag", func(t *testing.T) {
		flag := styleCmd.Flags().Lookup("fix")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has check flag", func(t *testing.T) {
		flag := styleCmd.Flags().Lookup("check")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has diff flag", func(t *testing.T) {
		flag := styleCmd.Flags().Lookup("diff")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestStyleCmdExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a properly styled file
	content := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0o644))

	t.Run("no files found", func(t *testing.T) {
		emptyDir := t.TempDir()
		styleFix = false
		styleCheck = false
		styleDiff = false
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"style", emptyDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("style check on valid file", func(t *testing.T) {
		styleFix = false
		styleCheck = false
		styleDiff = false
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"style", tmpDir})
		err := rootCmd.Execute()
		// Style may find issues (returning ExitError), but must not fail unexpectedly
		assert.NoError(t, err, "style on valid tf file should not error")
	})
}
