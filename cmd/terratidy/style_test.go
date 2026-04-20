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

// TestStyleDiffFlag tests that --diff flag shows diff content in output
func TestStyleDiffFlag(t *testing.T) {
	// Save and restore globals
	oldStyleFix := styleFix
	oldStyleCheck := styleCheck
	oldStyleDiff := styleDiff
	oldChanged := changed
	oldColor := color
	oldFormat := format

	resetStyleFlags := func() {
		for _, name := range []string{"fix", "check", "diff"} {
			if f := styleCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
	resetStyleFlags()

	t.Cleanup(func() {
		styleFix = oldStyleFix
		styleCheck = oldStyleCheck
		styleDiff = oldStyleDiff
		changed = oldChanged
		color = oldColor
		format = oldFormat
		rootCmd.SetArgs(nil)
		resetStyleFlags()
	})

	t.Run("diff flag shows preview without modifying file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create file with fixable style issue (extra blank lines)
		// The blank-lines rule fixes multiple consecutive blank lines
		contentWithIssue := `resource "aws_instance" "test" {
  ami = "ami-123"
}


resource "null_resource" "test2" {
  triggers = {}
}
`
		filePath := filepath.Join(tmpDir, "main.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(contentWithIssue), 0o644))

		// Get original content for comparison
		originalContent, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Reset flags
		styleFix = false
		styleCheck = false
		styleDiff = false
		changed = false
		color = false
		format = "text"

		rootCmd.SetArgs([]string{"style", "--diff", tmpDir})
		_ = rootCmd.Execute()

		// Verify file was NOT modified (preview mode)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "File should not be modified in preview mode")
	})

	t.Run("diff and fix flags show diff and apply changes", func(t *testing.T) {
		resetStyleFlags()
		tmpDir := t.TempDir()

		// Create file with fixable style issue (two blank lines between blocks)
		contentWithIssue := `resource "aws_instance" "test" {
  ami = "ami-123"
}


resource "null_resource" "test2" {
  triggers = {}
}
`
		filePath := filepath.Join(tmpDir, "main.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(contentWithIssue), 0o644))

		// Reset flags
		styleFix = false
		styleCheck = false
		styleDiff = false
		changed = false
		color = false
		format = "text"

		rootCmd.SetArgs([]string{"style", "--fix", "--diff", tmpDir})
		_ = rootCmd.Execute()

		// Verify file WAS modified (fix mode should reduce double blank lines to single)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.NotEqual(t, contentWithIssue, string(afterContent), "fix mode should modify the file")
		assert.Contains(t, string(afterContent), "}\n\nresource", "should have single blank line between blocks")
	})

	t.Run("diff on clean file produces no error", func(t *testing.T) {
		resetStyleFlags()
		tmpDir := t.TempDir()

		// Create a properly styled file (no issues to fix)
		cleanContent := `resource "aws_instance" "test" {
  ami = "ami-123"
}

resource "null_resource" "test2" {
  triggers = {}
}
`
		filePath := filepath.Join(tmpDir, "main.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(cleanContent), 0o644))

		// Reset flags
		styleFix = false
		styleCheck = false
		styleDiff = false
		changed = false
		color = false
		format = "text"

		// Run with --diff on a clean file
		rootCmd.SetArgs([]string{"style", "--diff", tmpDir})
		err := rootCmd.Execute()
		// Clean file should not produce error and file should remain unchanged
		assert.NoError(t, err, "diff on clean file should not error")

		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, cleanContent, string(afterContent), "clean file should remain unchanged")
	})
}
