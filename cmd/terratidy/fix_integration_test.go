package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestRunFmtFix(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
instance_type = "t2.micro"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	findings, formatted, err := runFmtFix(ctx, []string{tmpFile})
	require.NoError(t, err)
	assert.Greater(t, formatted, 0, "should have formatted at least one file")
	_ = findings

	// Verify file was actually modified
	newContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.NotEqual(t, content, string(newContent), "file should have been reformatted")
}

func TestRunStyleFixWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	cfg := config.DefaultConfig()
	findings, fixed, err := runStyleFixWithConfig(ctx, cfg, []string{tmpFile})
	require.NoError(t, err)
	_ = findings
	_ = fixed
}

func TestPrintFixSummary(t *testing.T) {
	// Should not panic with various inputs
	printFixSummary(nil, 0)
	printFixSummary([]sdk.Finding{{Fix: nil}}, 1)
	printFixSummary(nil, 5)
}
