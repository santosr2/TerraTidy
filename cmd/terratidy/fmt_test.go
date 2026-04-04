package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFmtCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "fmt [paths...]", fmtCmd.Use)
		assert.Equal(t, "Format Terraform and Terragrunt files", fmtCmd.Short)
		assert.NotEmpty(t, fmtCmd.Long)
		assert.NotEmpty(t, fmtCmd.Example)
	})

	t.Run("has check flag", func(t *testing.T) {
		flag := fmtCmd.Flags().Lookup("check")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has diff flag", func(t *testing.T) {
		flag := fmtCmd.Flags().Lookup("diff")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has all flag", func(t *testing.T) {
		flag := fmtCmd.Flags().Lookup("all")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestFmtCmdExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a properly formatted file
	formattedContent := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "formatted.tf"), []byte(formattedContent), 0o644))

	t.Run("check mode with formatted files", func(t *testing.T) {
		// Reset flags
		fmtCheck = true
		fmtDiff = false
		changed = false

		rootCmd.SetArgs([]string{"fmt", "--check", tmpDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("no files found", func(t *testing.T) {
		emptyDir := t.TempDir()
		fmtCheck = false
		fmtDiff = false
		changed = false

		rootCmd.SetArgs([]string{"fmt", emptyDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("all flag with style issues", func(t *testing.T) {
		styleDir := t.TempDir()
		// Content with style issue (missing blank line between blocks)
		styleContent := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(styleDir, "style.tf"), []byte(styleContent), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = true
		changed = false

		rootCmd.SetArgs([]string{"fmt", "--all", styleDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Reset flag
		fmtAll = false
	})
}
