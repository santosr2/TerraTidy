package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/terratidy/internal/config"
	"github.com/santosr2/terratidy/pkg/sdk"
)

func TestPrintCheckSummary(t *testing.T) {
	t.Run("no findings", func(t *testing.T) {
		err := printCheckSummary(nil)
		assert.NoError(t, err)
	})

	t.Run("with errors exits", func(t *testing.T) {
		findings := []sdk.Finding{
			{Severity: sdk.SeverityError, Rule: "test.rule"},
		}
		err := printCheckSummary(findings)
		assert.Error(t, err) // Should return ExitError
	})

	t.Run("warnings only no exit", func(t *testing.T) {
		findings := []sdk.Finding{
			{Severity: sdk.SeverityWarning, Rule: "test.rule"},
		}
		err := printCheckSummary(findings)
		assert.NoError(t, err)
	})
}

func TestPrintSeverityCounts(t *testing.T) {
	// Just verify it doesn't panic with various inputs
	printSeverityCounts(0, 0, 0)
	printSeverityCounts(1, 2, 3)
	printSeverityCounts(0, 5, 0)
}

func TestPrintCheckHints(t *testing.T) {
	// Verify it doesn't panic
	printCheckHints()
}

func TestRunFmtCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	findings, err := runFmtCheckWithConfig(ctx, nil, []string{tmpFile}, 1, true)
	require.NoError(t, err)
	assert.NotEmpty(t, findings, "unformatted file should produce findings")
}

func TestRunStyleCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	cfg := config.DefaultConfig()
	findings, err := runStyleCheckWithConfig(ctx, cfg, []string{tmpFile}, 2, true)
	require.NoError(t, err)
	// May or may not have findings depending on content
	_ = findings
}

func TestRunLintCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	cfg := config.DefaultConfig()
	findings, err := runLintCheckWithConfig(ctx, cfg, []string{tmpFile}, 3, true)
	require.NoError(t, err)
	_ = findings
}

func TestOutputCheckResults(t *testing.T) {
	t.Run("no findings", func(t *testing.T) {
		old := format
		format = "text"
		defer func() { format = old }()

		err := outputCheckResults(nil, false)
		assert.NoError(t, err)
	})

	t.Run("with error findings text format", func(t *testing.T) {
		old := format
		format = "text"
		defer func() { format = old }()

		findings := []sdk.Finding{
			{Rule: "test.rule", Message: "test", Severity: sdk.SeverityError, File: "test.tf"},
		}
		err := outputCheckResults(findings, false)
		assert.Error(t, err, "should return exit error for errors")
	})

	t.Run("with error findings json format", func(t *testing.T) {
		old := format
		format = "json"
		defer func() { format = old }()

		findings := []sdk.Finding{
			{Rule: "test.rule", Message: "test", Severity: sdk.SeverityError, File: "test.tf"},
		}
		err := outputCheckResults(findings, true)
		assert.Error(t, err, "should return exit error for errors in json")
	})
}

func TestRunAllChecksWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg := config.DefaultConfig()

	t.Run("sequential", func(t *testing.T) {
		old := checkParallel
		checkParallel = false
		defer func() { checkParallel = old }()

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true)
		require.NoError(t, err)
		_ = findings
	})

	t.Run("parallel", func(t *testing.T) {
		old := checkParallel
		checkParallel = true
		defer func() { checkParallel = old }()

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true)
		require.NoError(t, err)
		_ = findings
	})
}

func TestPrintCheckHeader(t *testing.T) {
	printCheckHeader(5)
	// Just verify it doesn't panic
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Set up globals
	oldChanged := changed
	oldFormat := format
	oldCfgFile := cfgFile
	changed = false
	format = "text"
	cfgFile = ""
	defer func() {
		changed = oldChanged
		format = oldFormat
		cfgFile = oldCfgFile
	}()

	rootCmd.SetArgs([]string{"check", dir})
	err := rootCmd.Execute()
	// May return ExitError if findings have errors
	_ = err
}
