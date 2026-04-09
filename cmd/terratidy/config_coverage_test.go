package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
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

func TestRunConfigShow_WithProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	content := `version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
profiles:
  minimal:
    engines:
      style:
        enabled: false
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))

	oldCfgFile := cfgFile
	oldProfile := profile
	cfgFile = cfgPath
	profile = "minimal"
	defer func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	}()

	err := runConfigShow(nil, nil)
	assert.NoError(t, err)
}

func TestRunConfigShow_WithProfile_InvalidProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	content := `version: 1
engines:
  fmt:
    enabled: true
profiles:
  ci:
    engines:
      fmt:
        enabled: false
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))

	oldCfgFile := cfgFile
	oldProfile := profile
	cfgFile = cfgPath
	profile = "nonexistent"
	defer func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	}()

	err := runConfigShow(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying profile")
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

func TestRunConfigInitProfile(t *testing.T) {
	t.Run("creates profile in new config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, ".terratidy.yaml")

		old := cfgFile
		cfgFile = cfgPath
		defer func() { cfgFile = old }()

		err := runConfigInitProfile(nil, []string{"production"})
		require.NoError(t, err)

		content, err := os.ReadFile(cfgPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "production")
	})

	t.Run("creates profile in existing config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, ".terratidy.yaml")
		require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\n"), 0o644))

		old := cfgFile
		cfgFile = cfgPath
		defer func() { cfgFile = old }()

		err := runConfigInitProfile(nil, []string{"staging"})
		require.NoError(t, err)

		content, err := os.ReadFile(cfgPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "staging")
	})

	t.Run("errors when profile already exists", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, ".terratidy.yaml")
		require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\nprofiles:\n  dev:\n    profile: dev\n"), 0o644))

		old := cfgFile
		cfgFile = cfgPath
		defer func() { cfgFile = old }()

		err := runConfigInitProfile(nil, []string{"dev"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestRunConfigShow_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\n"), 0o644))

	old := cfgFile
	oldFmt := configSerializeFormat
	cfgFile = cfgPath
	configSerializeFormat = "json"
	defer func() {
		cfgFile = old
		configSerializeFormat = oldFmt
	}()

	err := runConfigShow(nil, nil)
	assert.NoError(t, err)
}

func TestRunConfigShow_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: 1\n"), 0o644))

	old := cfgFile
	oldFmt := configSerializeFormat
	cfgFile = cfgPath
	configSerializeFormat = "toml"
	defer func() {
		cfgFile = old
		configSerializeFormat = oldFmt
	}()

	err := runConfigShow(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestRunConfigValidate_LoadFailure(t *testing.T) {
	// Point at a file that exists but has invalid YAML to trigger load failure.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: 99\n"), 0o644))

	old := cfgFile
	cfgFile = cfgPath
	defer func() { cfgFile = old }()

	err := runConfigValidate(nil, nil)
	require.Error(t, err)
}

func TestRunConfigValidate_WithEnabledEngines(t *testing.T) {
	// Exercises the engine summary output for each enabled/disabled engine.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	content := `version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: false
  policy:
    enabled: true
profiles:
  ci:
    profile: ci
    description: "CI profile"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))

	old := cfgFile
	cfgFile = cfgPath
	defer func() { cfgFile = old }()

	err := runConfigValidate(nil, nil)
	assert.NoError(t, err)
}

func TestValidateConfig_NoEnginesEnabled(t *testing.T) {
	cfg := &config.Config{}
	// All engines nil (IsEnabled returns false for all).
	issues := validateConfig(cfg)
	assert.Contains(t, issues, "no engines are enabled")
}

func TestValidateConfig_InvalidSeverity(t *testing.T) {
	cfg := &config.Config{
		SeverityThreshold: "FATAL",
	}
	issues := validateConfig(cfg)
	require.NotEmpty(t, issues)
	assert.Contains(t, issues[0], "invalid severity_threshold")
}
