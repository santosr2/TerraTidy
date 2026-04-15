package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Plugin management behavior is extensively tested in internal/plugins.
// These CLI tests verify command registration, argument handling, and basic
// execution paths (enabled/disabled, empty state, scaffold creation).

func TestPluginsCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "plugins", pluginsCmd.Use)
		assert.Equal(t, "Plugin management commands", pluginsCmd.Short)
		assert.NotEmpty(t, pluginsCmd.Long)
		assert.Contains(t, pluginsCmd.Long, "extend TerraTidy")
		assert.Contains(t, pluginsCmd.Long, ".terratidy.yaml")
	})

	t.Run("has subcommands", func(t *testing.T) {
		subcommands := pluginsCmd.Commands()
		require.Len(t, subcommands, 3)

		names := make([]string, len(subcommands))
		for i, cmd := range subcommands {
			names[i] = cmd.Name()
		}
		assert.Contains(t, names, "list")
		assert.Contains(t, names, "info")
		assert.Contains(t, names, "init")
	})

	t.Run("is registered on root command", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"plugins"})
		require.NoError(t, err)
		assert.Equal(t, "plugins", cmd.Name())
	})
}

func TestPluginsListCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "list", pluginsListCmd.Use)
		assert.Equal(t, "List installed plugins", pluginsListCmd.Short)
	})

	t.Run("is registered on plugins command", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"plugins", "list"})
		require.NoError(t, err)
		assert.Equal(t, "list", cmd.Name())
	})
}

func TestPluginsListCmd_Disabled(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create config with plugins disabled
	configContent := `version: 1
plugins:
  enabled: false
`
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Save and restore global state
	oldCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = oldCfgFile })
	cfgFile = configPath

	// Capture stdout with proper cleanup
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "creating stdout pipe")
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	// Run the command
	err = pluginsListCmd.RunE(pluginsListCmd, nil)
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)

	// Verify output indicates plugins are disabled
	assert.Contains(t, output, "Plugins are not enabled in configuration")
}

func TestPluginsListCmd_NoPlugins(t *testing.T) {
	// Create temp directory for test with a plugins subdirectory
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0o750)
	require.NoError(t, err)

	// Create config with plugins enabled but empty directory
	configContent := `version: 1
plugins:
  enabled: true
  directories:
    - ` + pluginsDir + `
`
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Save and restore global state
	oldCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = oldCfgFile })
	cfgFile = configPath

	// Capture stdout with proper cleanup
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "creating stdout pipe")
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	// Run the command
	err = pluginsListCmd.RunE(pluginsListCmd, nil)
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)

	// With plugins enabled but no plugins found, should show empty message
	assert.Contains(t, output, "No plugins installed")
	// Verify the searched directory is listed in output
	assert.Contains(t, output, pluginsDir)
}

func TestPluginsInfoCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "info [plugin-name]", pluginsInfoCmd.Use)
		assert.Equal(t, "Show detailed information about a plugin", pluginsInfoCmd.Short)
	})

	t.Run("rejects zero arguments", func(t *testing.T) {
		err := pluginsInfoCmd.Args(pluginsInfoCmd, []string{})
		require.Error(t, err)
	})

	t.Run("accepts exactly one argument", func(t *testing.T) {
		err := pluginsInfoCmd.Args(pluginsInfoCmd, []string{"my-plugin"})
		require.NoError(t, err)
	})

	t.Run("rejects two arguments", func(t *testing.T) {
		err := pluginsInfoCmd.Args(pluginsInfoCmd, []string{"plugin1", "plugin2"})
		require.Error(t, err)
	})

	t.Run("is registered on plugins command", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"plugins", "info"})
		require.NoError(t, err)
		assert.Equal(t, "info", cmd.Name())
	})
}

func TestPluginsInfoCmd_ValidPlugin(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0o750)
	require.NoError(t, err)

	// Minimal valid YAML rule for info command testing
	// (patterns/message not needed since we only test the info display, not rule evaluation)
	yamlContent := `name: test-rule
description: A test rule for plugin info
severity: warning
enabled: true
`
	err = os.WriteFile(filepath.Join(pluginsDir, "test-rule.yaml"), []byte(yamlContent), 0o600)
	require.NoError(t, err)

	// Create config with plugins enabled
	configContent := `version: 1
plugins:
  enabled: true
  directories:
    - ` + pluginsDir + `
`
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Save and restore global state
	oldCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = oldCfgFile })
	cfgFile = configPath

	// Capture stdout with proper cleanup
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "creating stdout pipe")
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	// Run the command with plugin name
	err = pluginsInfoCmd.RunE(pluginsInfoCmd, []string{"test-rule"})
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)

	// Verify output contains plugin details with specific field formats
	// Note: YAML plugins don't populate PluginMetadata.Description (only rule.Description()),
	// so Description: line is empty. Also, YAML rules don't implement RulePlugin interface,
	// so the "Provides N rule(s)" section is not shown (only Go plugins show that).
	assert.Contains(t, output, "Name:        test-rule")
	assert.Contains(t, output, "Type:        rule")
	assert.Contains(t, output, "Description: ") // Empty for YAML plugins
	assert.Contains(t, output, "Path:")
	assert.Contains(t, output, "test-rule.yaml")
}

func TestPluginsInfoCmd_NotFound(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0o750)
	require.NoError(t, err)

	// Create config with plugins enabled but empty directory
	configContent := `version: 1
plugins:
  enabled: true
  directories:
    - ` + pluginsDir + `
`
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Save and restore global state
	oldCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = oldCfgFile })
	cfgFile = configPath

	// Run the command with non-existent plugin name (no stdout capture needed, only error checked)
	err = pluginsInfoCmd.RunE(pluginsInfoCmd, []string{"nonexistent-plugin"})

	// Should return error for plugin not found
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin not found")
	assert.Contains(t, err.Error(), "nonexistent-plugin")

	// Plugin not found is user-correctable input, so ConfigError (exit code 2)
	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "should be an ExitError, got: %v", err)
	assert.Equal(t, sdk.ExitConfig, exitErr.Code, "should have config exit code")
}

func TestPluginsInitCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "init [name]", pluginsInitCmd.Use)
		assert.Equal(t, "Initialize a new plugin project", pluginsInitCmd.Short)
	})

	t.Run("rejects zero arguments", func(t *testing.T) {
		err := pluginsInitCmd.Args(pluginsInitCmd, []string{})
		require.Error(t, err)
	})

	t.Run("accepts exactly one argument", func(t *testing.T) {
		err := pluginsInitCmd.Args(pluginsInitCmd, []string{"my-plugin"})
		require.NoError(t, err)
	})

	t.Run("rejects two arguments", func(t *testing.T) {
		err := pluginsInitCmd.Args(pluginsInitCmd, []string{"plugin1", "plugin2"})
		require.Error(t, err)
	})

	t.Run("is registered on plugins command", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"plugins", "init"})
		require.NoError(t, err)
		assert.Equal(t, "init", cmd.Name())
	})
}

func TestPluginsInitCmd_Scaffold(t *testing.T) {
	// NOTE: This test uses os.Chdir which modifies process-global state.
	// The init command hard-codes filepath.Join(".", pluginName) so we must chdir.
	// This test must NOT be run with t.Parallel() and relies on sequential execution.
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "creating stdout pipe")
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	// Run the init command
	err = pluginsInitCmd.RunE(pluginsInitCmd, []string{"my-test-plugin"})
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)

	// Verify output indicates success
	assert.Contains(t, output, "Plugin project created")
	assert.Contains(t, output, "my-test-plugin")
	assert.Contains(t, output, "Next steps")

	// Verify scaffold files were created
	pluginDir := filepath.Join(tmpDir, "my-test-plugin")
	assert.DirExists(t, pluginDir)

	// Check main.go exists and has expected content
	mainPath := filepath.Join(pluginDir, "main.go")
	assert.FileExists(t, mainPath)
	mainContent, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	assert.Contains(t, string(mainContent), "my-test-plugin")
	assert.Contains(t, string(mainContent), "PluginMetadata")
	assert.Contains(t, string(mainContent), "sdk.Rule")

	// Check go.mod exists
	goModPath := filepath.Join(pluginDir, "go.mod")
	assert.FileExists(t, goModPath)
	goModContent, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	assert.Contains(t, string(goModContent), "module my-test-plugin")
	assert.Contains(t, string(goModContent), "github.com/santosr2/TerraTidy")

	// Check Makefile exists
	makefilePath := filepath.Join(pluginDir, "Makefile")
	assert.FileExists(t, makefilePath)
	makefileContent, err := os.ReadFile(makefilePath)
	require.NoError(t, err)
	assert.Contains(t, string(makefileContent), "build:")
	assert.Contains(t, string(makefileContent), "buildmode=plugin")
}

func TestPluginsInitCmd_ExistingFile(t *testing.T) {
	// NOTE: See TestPluginsInitCmd_Scaffold comment about os.Chdir limitation.
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Create a file (not directory) with the plugin name
	conflictPath := filepath.Join(tmpDir, "conflict-plugin")
	err = os.WriteFile(conflictPath, []byte("existing file"), 0o600)
	require.NoError(t, err)

	// Run the init command - should fail because path exists as file
	err = pluginsInitCmd.RunE(pluginsInitCmd, []string{"conflict-plugin"})

	// Should return error because os.MkdirAll will fail on existing file
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating directory")

	// Filesystem error is InternalError (exit code 3)
	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "should be an ExitError, got: %v", err)
	assert.Equal(t, sdk.ExitInternal, exitErr.Code, "should have internal exit code")
}
