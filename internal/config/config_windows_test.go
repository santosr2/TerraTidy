//go:build windows

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_WindowsPath exercises config loading via real Windows paths
// returned by t.TempDir(). This validates the full I/O path on Windows.
func TestLoad_WindowsPath(t *testing.T) {
	tmpDir := t.TempDir() // Returns Windows-style path on Windows
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	content := `version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.True(t, cfg.Engines.Fmt.IsEnabled())
	assert.True(t, cfg.Engines.Style.IsEnabled())
}

// TestPluginDirectories_TildePreserved verifies that ~ is preserved in config
// (expansion happens during plugin loading, not config parsing).
func TestPluginDirectories_TildePreserved(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	content := `version: 1
plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
    - ./local-plugins
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	require.NotNil(t, cfg.Plugins)
	require.Len(t, cfg.Plugins.Directories, 2)

	// ~ is preserved in config; expansion happens in plugin loader
	assert.Equal(t, "~/.terratidy/plugins", cfg.Plugins.Directories[0])
	assert.Equal(t, "./local-plugins", cfg.Plugins.Directories[1])
}
