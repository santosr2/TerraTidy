package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvVars_RequiredSyntax(t *testing.T) {
	t.Run("required var set", func(t *testing.T) {
		t.Setenv("TEST_REQUIRED_VAR", "hello")
		result := expandEnvVars("${TEST_REQUIRED_VAR:?must be set}")
		assert.Equal(t, "hello", result)
	})

	t.Run("required var unset returns empty", func(t *testing.T) {
		t.Setenv("TEST_MISSING_REQUIRED_VAR", "")
		result := expandEnvVars("${TEST_MISSING_REQUIRED_VAR:?must be set}")
		assert.Equal(t, "", result)
	})
}

func TestExpandEnvVars_DefaultSyntax(t *testing.T) {
	t.Run("var set ignores default", func(t *testing.T) {
		t.Setenv("TEST_SET_VAR", "actual")
		result := expandEnvVars("${TEST_SET_VAR:-fallback}")
		assert.Equal(t, "actual", result)
	})

	t.Run("var unset uses default", func(t *testing.T) {
		t.Setenv("TEST_UNSET_VAR", "")
		result := expandEnvVars("${TEST_UNSET_VAR:-fallback}")
		assert.Equal(t, "fallback", result)
	})
}

func TestMerge_KeyOverride(t *testing.T) {
	base := &Config{
		Version:           1,
		SeverityThreshold: "warning",
		Overrides: OverridesConfig{
			Rules: map[string]RuleConfig{
				"rule-a": {Enabled: true, Severity: "warning"},
			},
		},
	}

	other := &Config{
		Overrides: OverridesConfig{
			Rules: map[string]RuleConfig{
				"rule-a": {Enabled: false, Severity: "error"}, // override
				"rule-b": {Enabled: true},                     // new
			},
		},
	}

	base.merge(other)

	// Override rules should be merged
	require.Contains(t, base.Overrides.Rules, "rule-a")
	assert.False(t, base.Overrides.Rules["rule-a"].Enabled, "rule-a should be overridden to disabled")
	assert.Equal(t, "error", base.Overrides.Rules["rule-a"].Severity)
	require.Contains(t, base.Overrides.Rules, "rule-b")
	assert.True(t, base.Overrides.Rules["rule-b"].Enabled)
}

func TestLoad_EmptyPath_ReturnsDefaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.True(t, cfg.Engines.Fmt.IsEnabled())
}

func TestLoad_EmptyFile_ReturnsDefaults(t *testing.T) {
	// Create an actual empty file on disk (0 bytes)
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0o644))

	cfg, err := Load(emptyFile)
	require.NoError(t, err, "empty config file should not error")

	// Should return default configuration
	assert.Equal(t, 1, cfg.Version, "empty file should get default version")
	assert.True(t, cfg.Engines.Fmt.IsEnabled(), "empty file should enable fmt by default")
	assert.True(t, cfg.Engines.Style.IsEnabled(), "empty file should enable style by default")
}

func TestLoad_WhitespaceOnlyFile_ReturnsDefaults(t *testing.T) {
	// Create a file with only whitespace
	tmpDir := t.TempDir()
	wsFile := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(wsFile, []byte("   \n\t\n  "), 0o644))

	cfg, err := Load(wsFile)
	require.NoError(t, err, "whitespace-only config file should not error")

	// Should return default configuration
	assert.Equal(t, 1, cfg.Version, "whitespace-only file should get default version")
	assert.True(t, cfg.Engines.Fmt.IsEnabled(), "whitespace-only file should enable fmt by default")
}

func TestLoad_MinimalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	minimalFile := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(minimalFile, []byte("version: 1\n"), 0o644))

	cfg, err := Load(minimalFile)
	require.NoError(t, err, "minimal config (version only) should load successfully")

	assert.Equal(t, 1, cfg.Version, "version should be 1")

	// Engine fields are nil when not set in config. IsEnabled() returns false for nil.
	// Contrast with Load("") which returns DefaultConfig() with engines enabled.
	assert.Nil(t, cfg.Engines.Fmt.Enabled, "fmt.enabled should be nil (not set)")
	assert.Nil(t, cfg.Engines.Style.Enabled, "style.enabled should be nil (not set)")
	assert.Nil(t, cfg.Engines.Lint.Enabled, "lint.enabled should be nil (not set)")
	assert.Nil(t, cfg.Engines.Policy.Enabled, "policy.enabled should be nil (not set)")

	assert.Equal(t, "", cfg.SeverityThreshold, "severity threshold should be empty when not set")
}

func TestLoad_NullValues(t *testing.T) {
	tests := []struct {
		name    string
		content string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name:    "tilde null for exclude",
			content: "version: 1\nexclude: ~\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Nil(t, cfg.Exclude, "exclude should be nil")
			},
		},
		{
			name:    "explicit null for exclude",
			content: "version: 1\nexclude: null\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Nil(t, cfg.Exclude, "exclude should be nil")
			},
		},
		{
			name:    "tilde null for imports",
			content: "version: 1\nimports: ~\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Nil(t, cfg.Imports, "imports should be nil")
			},
		},
		{
			name:    "null nested in engines",
			content: "version: 1\nengines:\n  fmt:\n    enabled: ~\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Nil(t, cfg.Engines.Fmt.Enabled, "fmt.enabled should be nil")
			},
		},
		{
			name:    "null for severity threshold",
			content: "version: 1\nseverity_threshold: ~\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "", cfg.SeverityThreshold, "severity_threshold should be empty string")
			},
		},
		{
			name:    "null for fail_fast bool pointer",
			content: "version: 1\nfail_fast: ~\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Nil(t, cfg.FailFast, "fail_fast should be nil")
			},
		},
		{
			name:    "null for entire engines block",
			content: "version: 1\nengines: ~\n",
			check: func(t *testing.T, cfg *Config) {
				assert.Nil(t, cfg.Engines.Fmt.Enabled, "engines.fmt.enabled should be nil")
				assert.Nil(t, cfg.Engines.Style.Enabled, "engines.style.enabled should be nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgFile := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(cfgFile, []byte(tt.content), 0o644))

			cfg, err := Load(cfgFile)
			require.NoError(t, err, "config with null values should load without error")
			assert.Equal(t, 1, cfg.Version, "version should remain 1 with null values present")
			tt.check(t, cfg)
		})
	}
}
