package config

import (
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
		CustomRules: map[string]RuleConfig{
			"rule-a": {Enabled: true, Severity: "warning"},
		},
	}
	base.Overrides.Rules = map[string]RuleConfig{
		"override-a": {Enabled: true},
	}

	other := &Config{
		CustomRules: map[string]RuleConfig{
			"rule-a": {Enabled: false, Severity: "error"}, // override
			"rule-b": {Enabled: true},                     // new
		},
	}
	other.Overrides.Rules = map[string]RuleConfig{
		"override-a": {Enabled: false}, // override
		"override-b": {Enabled: true},  // new
	}

	base.merge(other)

	// Custom rules should be merged with override
	require.Contains(t, base.CustomRules, "rule-a")
	assert.False(t, base.CustomRules["rule-a"].Enabled, "rule-a should be overridden to disabled")
	assert.Equal(t, "error", base.CustomRules["rule-a"].Severity)
	require.Contains(t, base.CustomRules, "rule-b")
	assert.True(t, base.CustomRules["rule-b"].Enabled)

	// Override rules should be merged
	require.Contains(t, base.Overrides.Rules, "override-a")
	assert.False(t, base.Overrides.Rules["override-a"].Enabled)
	require.Contains(t, base.Overrides.Rules, "override-b")
}

func TestLoad_EmptyPath_ReturnsDefaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.True(t, cfg.Engines.Fmt.Enabled)
}
