package format

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_EmptyFileList(t *testing.T) {
	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_NilFileList(t *testing.T) {
	engine := New(nil)
	findings, err := engine.Run(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_NonExistentFile(t *testing.T) {
	engine := New(nil)
	_, err := engine.Run(context.Background(), []string{"/no/such/file.tf"})
	assert.Error(t, err)
}

func TestRun_DiffMode(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{Diff: true, Check: true})
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	require.NotEmpty(t, findings)

	// Diff mode findings should contain diff markers
	assert.Contains(t, findings[0].Message, "---")
}

func TestNew_NilConfig(t *testing.T) {
	engine := New(nil)
	require.NotNil(t, engine)
	assert.Equal(t, "fmt", engine.Name())
}

func TestRun_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	// Create multiple files so there's a chance to cancel between them
	for i := range 5 {
		content := `resource "aws_instance" "test" { ami = "ami-123" }`
		f := filepath.Join(dir, "test"+string(rune('0'+i))+".tf")
		require.NoError(t, os.WriteFile(f, []byte(content), 0o644))
	}

	var files []string
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		files = append(files, filepath.Join(dir, e.Name()))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = New(nil).Run(ctx, files)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRun_NonHCLFileSkipped(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte("package main"), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{goFile})
	require.NoError(t, err)
	assert.Empty(t, findings, "non-HCL files should be skipped")
}

func TestRun_CheckMode_NoModification(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{Check: true})
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	require.NotEmpty(t, findings, "check mode should report findings")

	// File should NOT be modified in check mode
	afterContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(afterContent), "file should not be modified in check mode")
}

func TestEngine_Name(t *testing.T) {
	assert.Equal(t, "fmt", New(nil).Name())
}

// TestRun_MalformedHCL_NoPanic verifies the engine handles malformed HCL gracefully without panic.
// hclwrite.Format is lenient and doesn't panic on invalid input - it returns the input unchanged
// or partially formatted. This test ensures we don't introduce panics in our wrapper code.
func TestRun_MalformedHCL_NoPanic(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "unclosed brace",
			content: `resource "aws_instance" "test" {`,
		},
		{
			name:    "unclosed quote",
			content: `resource "aws_instance" "test`,
		},
		{
			name:    "invalid syntax",
			content: `{{{{{`,
		},
		{
			name:    "random garbage",
			content: `!@#$%^&*()_+`,
		},
		{
			name:    "mixed valid and invalid",
			content: "resource \"aws_instance\" \"test\" {\n  ami = \"ami-123\"\n  invalid syntax here !!!!\n}",
		},
		{
			name:    "null bytes",
			content: "resource \"test\" \"x\" {\x00\x00}",
		},
		{
			name:    "truncated block",
			content: `resource "aws_s3_bucket" "bucket" { bucket = "my-bucket" tags = { Name = "`,
		},
		{
			name:    "whitespace only",
			content: "   \n\t\n  \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tmpFile := filepath.Join(dir, "malformed.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(&Config{Check: true})

			// This should not panic - the test passing without panic is the assertion
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			// hclwrite.Format is lenient - it may return the input unchanged
			// or partially formatted. We don't require an error, just no panic.
			if err != nil {
				assert.NotContains(t, err.Error(), "panic", "should not mention panic in error")
			}

			// In check mode, the file should NOT be modified regardless of findings
			afterContent, readErr := os.ReadFile(tmpFile)
			require.NoError(t, readErr)
			assert.Equal(t, tt.content, string(afterContent), "check mode should not modify file")

			// Findings may or may not be present depending on whether hclwrite
			// detected any formatting changes needed. We don't assert on findings
			// count since hclwrite behavior with malformed input varies.
			_ = findings
		})
	}
}
