package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// setupInitTest changes to tmpDir and registers cleanup for flags and working directory.
func setupInitTest(t *testing.T, tmpDir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	require.NoError(t, err)

	// Register cleanup before chdir to ensure restoration even on early failure
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
		initInteractive = false
		initSplit = false
		initMonorepo = false
		initForce = false
		rootCmd.SetArgs(nil)
	})

	require.NoError(t, os.Chdir(tmpDir))
}

// TestInitCmd_CreatesDefaultConfig verifies that init creates .terratidy.yaml
// with default configuration in an empty directory.
func TestInitCmd_CreatesDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	setupInitTest(t, tmpDir)

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "version: 1")
	assert.Contains(t, string(content), "engines:")
	assert.Contains(t, string(content), "fmt:")
	assert.Contains(t, string(content), "style:")
	assert.Contains(t, string(content), "lint:")

	// Validate YAML parses correctly
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(content, &parsed))
}

// TestInitCmd_SkipsExistingConfig verifies that init refuses to overwrite
// an existing config file without --force.
func TestInitCmd_SkipsExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	existingContent := "# existing config\nversion: 1\n"
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0o600))

	setupInitTest(t, tmpDir)

	rootCmd.SetArgs([]string{"init"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Contains(t, err.Error(), "--force")

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}

// TestInitCmd_ForceOverwrite verifies that init --force overwrites
// an existing config file.
func TestInitCmd_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	existingContent := "# existing config\nversion: 1\n"
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0o600))

	setupInitTest(t, tmpDir)

	rootCmd.SetArgs([]string{"init", "--force"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify new config is a valid, complete default config
	assert.Contains(t, string(content), "version: 1")
	assert.Contains(t, string(content), "engines:")
	assert.Contains(t, string(content), "fmt:")
	assert.Contains(t, string(content), "style:")

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(content, &parsed))
}

// TestInitCmd_MonorepoTemplate verifies that init --monorepo creates
// configuration optimized for monorepos with profiles.
func TestInitCmd_MonorepoTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	setupInitTest(t, tmpDir)

	rootCmd.SetArgs([]string{"init", "--monorepo"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Monorepo config includes environment-specific profiles
	assert.Contains(t, string(content), "profiles:")
	assert.Contains(t, string(content), "ci:")
	assert.Contains(t, string(content), "development:")

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(content, &parsed))

	// Verify profiles structure exists
	_, hasProfiles := parsed["profiles"]
	assert.True(t, hasProfiles, "monorepo config should define profiles")
}
