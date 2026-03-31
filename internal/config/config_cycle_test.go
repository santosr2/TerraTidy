package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadImports_CircularImport(t *testing.T) {
	dir := t.TempDir()

	// a.yaml imports b.yaml, b.yaml imports a.yaml
	aPath := filepath.Join(dir, "a.yaml")
	bPath := filepath.Join(dir, "b.yaml")

	aContent := "version: 1\nimports:\n  - " + bPath + "\n"
	bContent := "version: 1\nimports:\n  - " + aPath + "\n"

	require.NoError(t, os.WriteFile(aPath, []byte(aContent), 0o644))
	require.NoError(t, os.WriteFile(bPath, []byte(bContent), 0o644))

	_, err := Load(aPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular import detected")
}

func TestLoadImports_SameFileViaRelativeAndAbsolute(t *testing.T) {
	dir := t.TempDir()

	// Create a shared import file
	sharedPath := filepath.Join(dir, "shared.yaml")
	require.NoError(t, os.WriteFile(sharedPath, []byte("version: 1\n"), 0o644))

	// Main config imports the same file twice (relative and absolute)
	mainContent := "version: 1\nimports:\n  - shared.yaml\n  - " + sharedPath + "\n"
	mainPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0o644))

	_, err := Load(mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular import detected")
}

func TestLoadImports_NoCircle(t *testing.T) {
	dir := t.TempDir()

	// a.yaml and b.yaml imported by main, no cycle
	aPath := filepath.Join(dir, "a.yaml")
	bPath := filepath.Join(dir, "b.yaml")

	require.NoError(t, os.WriteFile(aPath, []byte("engines:\n  fmt:\n    enabled: true\n"), 0o644))
	require.NoError(t, os.WriteFile(bPath, []byte("engines:\n  style:\n    enabled: true\n"), 0o644))

	mainContent := "version: 1\nimports:\n  - " + aPath + "\n  - " + bPath + "\n"
	mainPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0o644))

	cfg, err := Load(mainPath)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
}

func TestSetDefaults(t *testing.T) {
	t.Run("version 0 gets set to 1", func(t *testing.T) {
		cfg := &Config{Version: 0}
		cfg.SetDefaults()
		assert.Equal(t, 1, cfg.Version)
	})

	t.Run("version 1 unchanged", func(t *testing.T) {
		cfg := &Config{Version: 1}
		cfg.SetDefaults()
		assert.Equal(t, 1, cfg.Version)
	})
}
