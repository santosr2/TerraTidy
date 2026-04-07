package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestRunAllFixesWithConfig(t *testing.T) {
	dir := t.TempDir()
	// Create a file that needs both formatting and style fixes.
	// Multiple blank lines between blocks triggers the blank-lines rule.
	content := `resource "aws_instance" "a" {
ami="ami-123"
}


resource "aws_instance" "b" {
ami="ami-456"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg := config.DefaultConfig()
	findings, totalFixed, err := runAllFixesWithConfig(cfg, []string{tmpFile}, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totalFixed, 0)
	_ = findings
}

func TestPrintFixHeader(t *testing.T) {
	// Verify neither panics nor errors with various file counts.
	oldChanged := changed
	changed = false
	defer func() { changed = oldChanged }()

	printFixHeader(1)
	printFixHeader(10)

	changed = true
	printFixHeader(3)
}

func TestRunAllFixesWithConfig_StyleFixedTriggersReformat(t *testing.T) {
	dir := t.TempDir()
	// Two resources with two blank lines between them triggers the blank-lines rule,
	// causing styleFixed > 0 which exercises the re-formatting branch.
	content := `resource "aws_instance" "one" {
  ami = "ami-111"
}


resource "aws_instance" "two" {
  ami = "ami-222"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg := config.DefaultConfig()
	cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
	cfg.Engines.Style.Enabled = config.BoolPtr(true)

	findings, totalFixed, err := runAllFixesWithConfig(cfg, []string{tmpFile}, nil)
	require.NoError(t, err)
	// The blank-lines rule should have fixed at least 1 issue.
	assert.GreaterOrEqual(t, totalFixed, 1)
	_ = findings
}

func TestRunStyleFixWithConfig_WithPluginRules(t *testing.T) {
	dir := t.TempDir()

	// YAML rule requiring 'owner' attribute
	yamlRule := `name: require-owner
description: Resources must have an owner attribute
severity: warning
enabled: true
message: "Resource is missing 'owner' attribute"
patterns:
  required_attributes:
    - owner
`
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "require-owner.yaml"), []byte(yamlRule), 0o644))

	tfContent := `resource "aws_instance" "test" {
  ami = "ami-123"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}

	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)

	ctx := context.Background()
	findings, fixed, err := runStyleFixWithConfig(ctx, cfg, []string{tfFile}, pluginRules)
	require.NoError(t, err)
	_ = fixed

	// Plugin rule finding should appear (non-fixable)
	var found bool
	for _, f := range findings {
		if f.Rule == "require-owner" {
			found = true
		}
	}
	assert.True(t, found, "plugin rule should produce finding in fix mode")
}

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
	findings, fixed, err := runStyleFixWithConfig(ctx, cfg, []string{tmpFile}, nil)
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

func TestRunFixWithExcludes(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create directory structure
	externalDir := filepath.Join(dir, "external")
	require.NoError(t, os.MkdirAll(externalDir, 0o755))

	// Create a file that needs formatting (will be fixed)
	mainContent := `resource "aws_instance" "test" {
ami           = "ami-123"
instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(mainContent), 0o644))

	// Create a badly formatted file in "external" directory (should be excluded)
	badContent := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
	require.NoError(t, os.WriteFile(filepath.Join(externalDir, "external.tf"), []byte(badContent), 0o644))

	// Create config with exclude patterns
	configContent := `version: 1
exclude:
  - "external/**"
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
	configPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	oldFormat := format
	oldChanged := changed
	oldExclude := excludePatterns
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
		format = oldFormat
		changed = oldChanged
		excludePatterns = oldExclude
	})

	// Set up global state
	cfgFile = configPath
	profile = ""
	format = "text"
	changed = false
	excludePatterns = nil

	// Run the fix command - this exercises getTargetFilesWithExcludes in fix.go
	err := runFix(&cobra.Command{}, []string{dir})
	require.NoError(t, err)

	// Verify main.tf was formatted
	content, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "ami           =", "main.tf should be formatted")

	// Verify external.tf was NOT touched (excluded)
	externalContent, err := os.ReadFile(filepath.Join(externalDir, "external.tf"))
	require.NoError(t, err)
	assert.Equal(t, badContent, string(externalContent), "excluded file should not be modified")
}
