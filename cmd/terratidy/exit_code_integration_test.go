package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// resetCheckGlobals saves and restores all global state that check command uses.
// This prevents test pollution when tests run sequentially.
func resetCheckGlobals(t *testing.T) {
	t.Helper()

	// Save all globals that check command touches.
	oldFormat := format
	oldCfgFile := cfgFile
	oldChanged := changed
	oldSeverityThreshold := severityThreshold
	oldCheckSkipFmt := checkSkipFmt
	oldCheckSkipStyle := checkSkipStyle
	oldCheckSkipLint := checkSkipLint
	oldCheckSkipPolicy := checkSkipPolicy
	oldCheckParallel := checkParallel
	oldProfile := profile
	oldNoRecurse := noRecurse
	oldAbsolutePaths := absolutePaths
	oldColor := color

	t.Cleanup(func() {
		// Restore all globals.
		format = oldFormat
		cfgFile = oldCfgFile
		changed = oldChanged
		severityThreshold = oldSeverityThreshold
		checkSkipFmt = oldCheckSkipFmt
		checkSkipStyle = oldCheckSkipStyle
		checkSkipLint = oldCheckSkipLint
		checkSkipPolicy = oldCheckSkipPolicy
		checkParallel = oldCheckParallel
		profile = oldProfile
		noRecurse = oldNoRecurse
		absolutePaths = oldAbsolutePaths
		color = oldColor
		// Clear rootCmd args to avoid bleeding into next test.
		rootCmd.SetArgs(nil)
	})
}

// TestExitCode_Success verifies that the CLI exits with code 0 when no findings exist.
func TestExitCode_Success(t *testing.T) {
	resetCheckGlobals(t)

	// Create a well-formatted Terraform file that passes all checks.
	dir := t.TempDir()
	content := `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o644))

	// Set test state.
	format = "text"
	cfgFile = ""
	changed = false

	// Run the check command (skip lint to avoid terraform warnings).
	rootCmd.SetArgs([]string{"check", "--skip-policy", "--skip-lint", dir})
	err := rootCmd.Execute()

	// Should succeed with no error (exit 0).
	assert.NoError(t, err, "check command should succeed with no findings")
}

// TestExitCode_Findings verifies that the CLI exits with code 1 when error-severity findings exist.
func TestExitCode_Findings(t *testing.T) {
	resetCheckGlobals(t)

	// Create an unformatted Terraform file that will trigger findings.
	dir := t.TempDir()
	content := `resource "aws_instance" "bad"   {
ami="ami-123"
instance_type="t2.micro"
}`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o644))

	// Create a config that makes fmt findings errors (default severity).
	configContent := `version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: false
  lint:
    enabled: false
  policy:
    enabled: false
`
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(configContent), 0o644))

	// Set test state.
	format = "text"
	cfgFile = cfgPath
	changed = false

	// Run the check command.
	rootCmd.SetArgs([]string{"check", dir})
	err := rootCmd.Execute()

	// Should return ExitError with code 1.
	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "expected ExitError, got: %v", err)
	assert.Equal(t, 1, exitErr.Code, "exit code should be 1 for findings")
}

// TestExitCode_Error verifies that config parse failures return a plain error (not ExitError).
// Config loading errors are not findings, so they should not use the ExitError mechanism.
func TestExitCode_Error(t *testing.T) {
	resetCheckGlobals(t)

	// Create a directory with invalid config to trigger an error.
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(`resource "x" "y" {}`), 0o644))

	// Create an invalid config file (malformed YAML).
	invalidConfig := `version: 1
engines:
  fmt: [this is invalid yaml
`
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(invalidConfig), 0o644))

	// Set test state.
	format = "text"
	cfgFile = cfgPath
	changed = false

	// Run the check command.
	rootCmd.SetArgs([]string{"check", dir})
	err := rootCmd.Execute()

	// Should return an error (config parse failure).
	require.Error(t, err, "check command should fail with invalid config")

	// Config errors should NOT be wrapped as ExitError (they are not findings).
	var exitErr *sdk.ExitError
	assert.False(t, errors.As(err, &exitErr), "config error should not be an ExitError")

	// Verify it's a config loading error.
	assert.Contains(t, err.Error(), "loading config", "error should mention config loading")
}

// TestExitCode_SeverityThreshold_Warning verifies that --severity-threshold=warning
// filters out info-level findings but warnings still don't cause exit code 1.
// Exit code 1 only happens when there are error-severity findings.
func TestExitCode_SeverityThreshold_Warning(t *testing.T) {
	resetCheckGlobals(t)

	// Create a file with style issues that generate warnings (two blank lines between blocks).
	dir := t.TempDir()
	content := `resource "aws_instance" "one" {
  ami = "ami-123"
}


resource "aws_instance" "two" {
  ami = "ami-456"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o644))

	// Config: only style enabled, findings are warnings by default.
	configContent := `version: 1
engines:
  fmt:
    enabled: false
  style:
    enabled: true
  lint:
    enabled: false
  policy:
    enabled: false
`
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(configContent), 0o644))

	// Set test state.
	format = "text"
	cfgFile = cfgPath
	changed = false
	severityThreshold = "warning"

	// Run the check command with severity threshold.
	rootCmd.SetArgs([]string{"check", "--severity-threshold", "warning", dir})
	err := rootCmd.Execute()

	// Should succeed because warnings don't cause exit 1 (only errors do).
	// The file has two blank lines between blocks which triggers the blank-line-between-blocks
	// style rule (a warning), but warnings should not cause exit failure.
	assert.NoError(t, err, "warnings should not cause exit 1")
}

// TestExitCode_SeverityThreshold_Error verifies that --severity-threshold=error
// filters out warnings so they don't appear, but errors still cause exit 1.
func TestExitCode_SeverityThreshold_Error(t *testing.T) {
	resetCheckGlobals(t)

	// Create an unformatted file (generates error).
	dir := t.TempDir()
	content := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(content), 0o644))

	// Config: fmt enabled (will find error), style disabled.
	configContent := `version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: false
  lint:
    enabled: false
  policy:
    enabled: false
`
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(configContent), 0o644))

	// Set test state.
	format = "text"
	cfgFile = cfgPath
	changed = false
	severityThreshold = "error"

	// Run the check command with error threshold.
	rootCmd.SetArgs([]string{"check", "--severity-threshold", "error", dir})
	err := rootCmd.Execute()

	// Should return ExitError because there are error-severity findings.
	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "expected ExitError for errors, got: %v", err)
	assert.Equal(t, 1, exitErr.Code, "exit code should be 1 for errors")
}
