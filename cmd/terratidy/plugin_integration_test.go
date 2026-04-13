//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
)

// TestPlugin_LoadError_InvalidPath verifies that when a configured plugin directory
// points to a file (not a directory), the check command returns an appropriate error.
// This tests the CLI-level handling of plugin load errors.
func TestPlugin_LoadError_InvalidPath(t *testing.T) {
	resetCheckGlobals(t)

	dir := t.TempDir()

	// Create a valid .tf file to check
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Create a FILE with the same path where the plugin directory should be.
	// This simulates a misconfigured plugin directory pointing to a file.
	invalidPluginPath := filepath.Join(dir, "plugins")
	require.NoError(t, os.WriteFile(invalidPluginPath, []byte("not a directory"), 0o644))

	// Create config with plugins enabled, pointing to the file (not directory)
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{invalidPluginPath}
	cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
	cfg.Engines.Style.Enabled = config.BoolPtr(false)
	cfg.Engines.Lint.Enabled = config.BoolPtr(false)
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)

	// loadPluginRules uses the plugin manager which calls loadFromDirectory
	// When the path is a file (not directory), it returns an error
	pluginRules, err := loadPluginRules(cfg)

	// The plugin manager should return an error because the path is not a directory
	require.Error(t, err, "expected error when plugin directory is actually a file")
	assert.Contains(t, err.Error(), "not a directory", "error should indicate path is not a directory")
	assert.Nil(t, pluginRules, "no rules should be returned on error")
}

// TestPlugin_TagFiltering verifies that the plugins.tags config option correctly
// filters plugin rules based on their tags. Only rules with matching tags should
// be loaded and executed.
func TestPlugin_TagFiltering(t *testing.T) {
	resetCheckGlobals(t)

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginsDir, 0o750))

	// Create a .tf file that would trigger findings if rules run
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Create two YAML rules with different tags
	// Rule 1: tagged with "security"
	securityRule := `name: check-security
description: A security check rule
severity: warning
enabled: true
tags:
  - security
  - compliance
patterns:
  block_types:
    - resource
  forbidden_attributes:
    - password
message: "Password attribute found"
`
	require.NoError(t, os.WriteFile(
		filepath.Join(pluginsDir, "security.yaml"),
		[]byte(securityRule),
		0o600,
	))

	// Rule 2: tagged with "style"
	styleRule := `name: check-style
description: A style check rule
severity: warning
enabled: true
tags:
  - style
patterns:
  block_types:
    - resource
  forbidden_attributes:
    - temp_name
message: "Temp name attribute found"
`
	require.NoError(t, os.WriteFile(
		filepath.Join(pluginsDir, "style.yaml"),
		[]byte(styleRule),
		0o600,
	))

	// Test 1: Load ALL rules (no tag filter)
	cfgNoFilter := config.DefaultConfig()
	cfgNoFilter.Plugins.Enabled = true
	cfgNoFilter.Plugins.Directories = []string{pluginsDir}
	cfgNoFilter.Plugins.Tags = []string{} // Empty = all tags

	rulesAll, err := loadPluginRules(cfgNoFilter)
	require.NoError(t, err)
	assert.Len(t, rulesAll, 2, "without tag filter, should load all rules")

	// Test 2: Filter by "security" tag only
	cfgSecurityOnly := config.DefaultConfig()
	cfgSecurityOnly.Plugins.Enabled = true
	cfgSecurityOnly.Plugins.Directories = []string{pluginsDir}
	cfgSecurityOnly.Plugins.Tags = []string{"security"}

	rulesSecurity, err := loadPluginRules(cfgSecurityOnly)
	require.NoError(t, err)
	require.Len(t, rulesSecurity, 1, "with 'security' tag filter, should load only one rule")
	assert.Equal(t, "check-security", rulesSecurity[0].Name())

	// Test 3: Filter by "style" tag only
	cfgStyleOnly := config.DefaultConfig()
	cfgStyleOnly.Plugins.Enabled = true
	cfgStyleOnly.Plugins.Directories = []string{pluginsDir}
	cfgStyleOnly.Plugins.Tags = []string{"style"}

	rulesStyle, err := loadPluginRules(cfgStyleOnly)
	require.NoError(t, err)
	require.Len(t, rulesStyle, 1, "with 'style' tag filter, should load only one rule")
	assert.Equal(t, "check-style", rulesStyle[0].Name())

	// Test 4: Filter by non-matching tag - should return empty
	cfgNoMatch := config.DefaultConfig()
	cfgNoMatch.Plugins.Enabled = true
	cfgNoMatch.Plugins.Directories = []string{pluginsDir}
	cfgNoMatch.Plugins.Tags = []string{"nonexistent-tag"}

	rulesNoMatch, err := loadPluginRules(cfgNoMatch)
	require.NoError(t, err)
	assert.Empty(t, rulesNoMatch, "with non-matching tag filter, should return no rules")

	// Test 5: Filter by multiple tags (OR logic - matches any)
	cfgMultiple := config.DefaultConfig()
	cfgMultiple.Plugins.Enabled = true
	cfgMultiple.Plugins.Directories = []string{pluginsDir}
	cfgMultiple.Plugins.Tags = []string{"security", "style"}

	rulesMultiple, err := loadPluginRules(cfgMultiple)
	require.NoError(t, err)
	assert.Len(t, rulesMultiple, 2, "with multiple tags, should load rules matching any tag")
}

// TestPlugin_TagFiltering_Integration verifies tag filtering works end-to-end
// through the check command, not just loadPluginRules.
func TestPlugin_TagFiltering_Integration(t *testing.T) {
	resetCheckGlobals(t)

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginsDir, 0o750))

	// Create a .tf file with forbidden attributes that would trigger findings
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  password      = "secret123"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Create a security rule that will match the "password" attribute
	securityRule := `name: check-password
description: Detects password attributes in resources
severity: warning
enabled: true
tags:
  - security
patterns:
  block_types:
    - resource
  forbidden_attributes:
    - password
message: "Password attribute should not be used in resource definitions"
`
	require.NoError(t, os.WriteFile(
		filepath.Join(pluginsDir, "security.yaml"),
		[]byte(securityRule),
		0o600,
	))

	ctx := context.Background()

	// Test 1: With matching tag filter, rule should run and produce findings
	cfgMatch := config.DefaultConfig()
	cfgMatch.Plugins.Enabled = true
	cfgMatch.Plugins.Directories = []string{pluginsDir}
	cfgMatch.Plugins.Tags = []string{"security"}
	cfgMatch.Engines.Fmt.Enabled = config.BoolPtr(false)
	cfgMatch.Engines.Style.Enabled = config.BoolPtr(true) // Plugin rules injected into style/lint engines
	cfgMatch.Engines.Lint.Enabled = config.BoolPtr(false)
	cfgMatch.Engines.Policy.Enabled = config.BoolPtr(false)

	// Load plugin rules with tag filter (detailed filtering tested in TestPlugin_TagFiltering)
	pluginRules, err := loadPluginRules(cfgMatch)
	require.NoError(t, err)

	findingsMatch, err := runAllChecksSequentialWithConfig(ctx, cfgMatch, []string{tfFile}, true, pluginRules)
	require.NoError(t, err)

	// Should have at least one finding from the plugin rule
	var hasPluginFinding bool
	for _, f := range findingsMatch {
		if f.Rule == "check-password" {
			hasPluginFinding = true
			assert.Contains(t, f.Message, "Password attribute")
			break
		}
	}
	assert.True(t, hasPluginFinding, "should have finding from check-password plugin rule")

	// Test 2: With non-matching tag filter, rule should NOT run
	cfgNoMatch := config.DefaultConfig()
	cfgNoMatch.Plugins.Enabled = true
	cfgNoMatch.Plugins.Directories = []string{pluginsDir}
	cfgNoMatch.Plugins.Tags = []string{"nonexistent"}
	cfgNoMatch.Engines.Fmt.Enabled = config.BoolPtr(false)
	cfgNoMatch.Engines.Style.Enabled = config.BoolPtr(true)
	cfgNoMatch.Engines.Lint.Enabled = config.BoolPtr(false)
	cfgNoMatch.Engines.Policy.Enabled = config.BoolPtr(false)

	// loadPluginRules filtering is already tested in TestPlugin_TagFiltering;
	// here we only verify the end-to-end outcome (no plugin findings)
	pluginRulesFiltered, err := loadPluginRules(cfgNoMatch)
	require.NoError(t, err)

	findingsNoMatch, err := runAllChecksSequentialWithConfig(ctx, cfgNoMatch, []string{tfFile}, true, pluginRulesFiltered)
	require.NoError(t, err)

	// Should have no findings from plugin rules (only built-in style rules might fire)
	for _, f := range findingsNoMatch {
		assert.NotEqual(t, "check-password", f.Rule, "plugin rule should not run with non-matching tag")
	}
}
