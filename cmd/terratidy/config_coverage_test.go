package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigPath(t *testing.T) {
	t.Run("uses global cfgFile when set", func(t *testing.T) {
		old := cfgFile
		cfgFile = "/custom/path/.terratidy.yaml"
		defer func() { cfgFile = old }()

		assert.Equal(t, "/custom/path/.terratidy.yaml", getConfigPath())
	})

	t.Run("defaults to .terratidy.yaml when empty", func(t *testing.T) {
		old := cfgFile
		cfgFile = ""
		defer func() { cfgFile = old }()

		assert.Equal(t, ".terratidy.yaml", getConfigPath())
	})
}

func TestWriteYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	data := map[string]any{
		"version": 1,
		"engines": map[string]any{
			"fmt": map[string]any{"enabled": true},
		},
	}

	err := writeYAMLFile(path, data)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "version: 1")
	assert.Contains(t, string(content), "engines:")
}

func TestWriteYAMLFile_InvalidPath(t *testing.T) {
	err := writeYAMLFile("/nonexistent/dir/file.yaml", map[string]any{})
	assert.Error(t, err)
}

func TestRunConfigShow(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\nengines:\n  fmt:\n    enabled: true\n"), 0o644))

	old := cfgFile
	cfgFile = cfgPath
	defer func() { cfgFile = old }()

	err := runConfigShow(nil, nil)
	// Should succeed (prints to stdout)
	assert.NoError(t, err)
}

func TestRunConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, ".terratidy.yaml")
		require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\n"), 0o644))

		old := cfgFile
		cfgFile = cfgPath
		defer func() { cfgFile = old }()

		err := runConfigValidate(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("nonexistent config file errors", func(t *testing.T) {
		old := cfgFile
		cfgFile = "/nonexistent/.terratidy.yaml"
		defer func() { cfgFile = old }()

		err := runConfigValidate(nil, nil)
		assert.Error(t, err)
	})
}

func TestRunConfigSplit(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\nengines:\n  fmt:\n    enabled: true\n  style:\n    enabled: true\n"), 0o644))

	old := cfgFile
	cfgFile = cfgPath
	defer func() { cfgFile = old }()

	// runConfigSplit creates .terratidy/ in cwd, so chdir to temp
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runConfigSplit(nil, nil)
	require.NoError(t, err)

	splitDir := filepath.Join(dir, ".terratidy")
	_, err = os.Stat(splitDir)
	assert.NoError(t, err)
}

func TestRunConfigMerge(t *testing.T) {
	dir := t.TempDir()

	splitDir := filepath.Join(dir, ".terratidy")
	require.NoError(t, os.MkdirAll(splitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(splitDir, "fmt.yaml"), []byte("engines:\n  fmt:\n    enabled: true\n"), 0o644))

	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\nimports:\n  - .terratidy/*.yaml\n"), 0o644))

	old := cfgFile
	cfgFile = cfgPath
	defer func() { cfgFile = old }()

	// runConfigMerge also works in cwd context
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runConfigMerge(nil, nil)
	require.NoError(t, err)
}
