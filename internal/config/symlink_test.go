//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Symlink tests verify config loading handles symlinks correctly.
// These tests require actual filesystem operations and are skipped on Windows
// where symlinks require elevated privileges.

func TestLoad_SymlinkedConfig(t *testing.T) {
	// Test that config file can be a symlink to another file.
	// This is useful when sharing configs across projects or when
	// config is stored in a central location.

	tmpDir := t.TempDir()

	// Create the real config file in a subdirectory
	realDir := filepath.Join(tmpDir, "configs")
	require.NoError(t, os.MkdirAll(realDir, 0o755))

	realConfig := filepath.Join(realDir, "shared.yaml")
	content := `version: 1
severity_threshold: error
fail_fast: true
engines:
  fmt:
    enabled: true
  style:
    enabled: false
`
	require.NoError(t, os.WriteFile(realConfig, []byte(content), 0o644))

	// Create a symlink to the config file
	symlinkConfig := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.Symlink(realConfig, symlinkConfig))

	// Load config via the symlink
	cfg, err := Load(symlinkConfig)
	require.NoError(t, err, "should load config through symlink")

	// Verify config was loaded correctly
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "error", cfg.SeverityThreshold)
	assert.True(t, cfg.IsFailFast())
	assert.True(t, cfg.Engines.Fmt.IsEnabled())
	assert.False(t, cfg.Engines.Style.IsEnabled())
}

func TestLoad_SymlinkedConfigWithImports(t *testing.T) {
	// Test that a symlinked config can import files relative to the symlink location.

	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   project/
	//     .terratidy.yaml -> ../configs/main.yaml (symlink)
	//     local-rules.yaml (import target)
	//   configs/
	//     main.yaml (real config)

	configsDir := filepath.Join(tmpDir, "configs")
	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configsDir, 0o755))
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Create the real config with an import
	mainConfig := filepath.Join(configsDir, "main.yaml")
	mainContent := `version: 1
imports:
  - local-rules.yaml
engines:
  fmt:
    enabled: true
`
	require.NoError(t, os.WriteFile(mainConfig, []byte(mainContent), 0o644))

	// Create the import target in the project directory
	localRules := filepath.Join(projectDir, "local-rules.yaml")
	localContent := `severity_threshold: warning
engines:
  style:
    enabled: false
`
	require.NoError(t, os.WriteFile(localRules, []byte(localContent), 0o644))

	// Create symlink in project directory pointing to main config
	symlinkConfig := filepath.Join(projectDir, ".terratidy.yaml")
	require.NoError(t, os.Symlink(mainConfig, symlinkConfig))

	// Load config via symlink - imports should resolve relative to symlink's directory
	cfg, err := Load(symlinkConfig)
	require.NoError(t, err, "should load symlinked config with imports")

	// Verify both main config and import were applied
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "warning", cfg.SeverityThreshold) // From local-rules.yaml
	assert.True(t, cfg.Engines.Fmt.IsEnabled())       // From main.yaml
	assert.False(t, cfg.Engines.Style.IsEnabled())    // From local-rules.yaml
}

func TestLoad_ImportFromSymlinkedDirectory(t *testing.T) {
	// Test that imports can reference files through a symlinked directory.
	// This is important for workspaces where shared configs are symlinked.

	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   project/
	//     .terratidy.yaml (main config)
	//     shared/ -> ../shared-configs/ (symlinked directory)
	//   shared-configs/
	//     rules.yaml (import target through symlink)

	projectDir := filepath.Join(tmpDir, "project")
	sharedConfigsDir := filepath.Join(tmpDir, "shared-configs")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.MkdirAll(sharedConfigsDir, 0o755))

	// Create shared rules in the real directory
	sharedRules := filepath.Join(sharedConfigsDir, "rules.yaml")
	sharedContent := `severity_threshold: error
engines:
  lint:
    enabled: false
`
	require.NoError(t, os.WriteFile(sharedRules, []byte(sharedContent), 0o644))

	// Create symlink to shared-configs directory
	symlinkDir := filepath.Join(projectDir, "shared")
	require.NoError(t, os.Symlink(sharedConfigsDir, symlinkDir))

	// Create main config that imports through the symlinked directory
	mainConfig := filepath.Join(projectDir, ".terratidy.yaml")
	mainContent := `version: 1
imports:
  - shared/rules.yaml
engines:
  fmt:
    enabled: true
`
	require.NoError(t, os.WriteFile(mainConfig, []byte(mainContent), 0o644))

	// Load config - should follow symlinked directory for imports
	cfg, err := Load(mainConfig)
	require.NoError(t, err, "should load imports through symlinked directory")

	// Verify both configs were applied
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "error", cfg.SeverityThreshold) // From shared/rules.yaml
	assert.True(t, cfg.Engines.Fmt.IsEnabled())     // From main config
	assert.False(t, cfg.Engines.Lint.IsEnabled())   // From shared/rules.yaml
}

func TestLoad_SymlinkedImportFile(t *testing.T) {
	// Test that an import target can be a symlink to another file.

	tmpDir := t.TempDir()

	// Create the real import target
	realImport := filepath.Join(tmpDir, "real-rules.yaml")
	realContent := `severity_threshold: info
engines:
  policy:
    enabled: true
`
	require.NoError(t, os.WriteFile(realImport, []byte(realContent), 0o644))

	// Create a symlink that will be imported
	symlinkImport := filepath.Join(tmpDir, "rules-link.yaml")
	require.NoError(t, os.Symlink(realImport, symlinkImport))

	// Create main config that imports the symlink
	mainConfig := filepath.Join(tmpDir, ".terratidy.yaml")
	mainContent := `version: 1
imports:
  - rules-link.yaml
engines:
  fmt:
    enabled: true
`
	require.NoError(t, os.WriteFile(mainConfig, []byte(mainContent), 0o644))

	// Load config - should follow symlinked import
	cfg, err := Load(mainConfig)
	require.NoError(t, err, "should load symlinked import file")

	// Verify both configs were applied
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "info", cfg.SeverityThreshold) // From symlinked import
	assert.True(t, cfg.Engines.Fmt.IsEnabled())    // From main config
	assert.True(t, cfg.Engines.Policy.IsEnabled()) // From symlinked import
}

func TestLoad_BrokenSymlink(t *testing.T) {
	// Test that a broken symlink (dangling target) returns default config.
	// This matches the behavior for a missing config file: os.Stat on a broken
	// symlink returns IsNotExist=true, so Load treats it as "no config file".

	tmpDir := t.TempDir()

	// Create a symlink pointing to a non-existent file
	brokenSymlink := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.Symlink("/non/existent/config.yaml", brokenSymlink))

	// Load should return default config (same as missing file)
	cfg, err := Load(brokenSymlink)
	require.NoError(t, err, "broken symlink treated as missing file")
	assert.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.Version)
	assert.True(t, cfg.Engines.Fmt.IsEnabled(), "should have default fmt enabled")
}
