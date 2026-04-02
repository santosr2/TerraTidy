package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		want     bool
	}{
		{"nil findings", nil, false},
		{"empty findings", []sdk.Finding{}, false},
		{"warnings only", []sdk.Finding{{Severity: sdk.SeverityWarning}}, false},
		{"info only", []sdk.Finding{{Severity: sdk.SeverityInfo}}, false},
		{"has error", []sdk.Finding{{Severity: sdk.SeverityError}}, true},
		{"mixed with error", []sdk.Finding{
			{Severity: sdk.SeverityWarning},
			{Severity: sdk.SeverityError},
			{Severity: sdk.SeverityInfo},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasErrors(tt.findings))
		})
	}
}

func TestBuildStyleConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		cfg := buildStyleConfig(nil, false)
		require.NotNil(t, cfg)
		assert.False(t, cfg.Fix)
		assert.Empty(t, cfg.Rules)
	})

	t.Run("with fix and diff", func(t *testing.T) {
		cfg := buildStyleConfig(nil, true, true)
		assert.True(t, cfg.Fix)
		assert.True(t, cfg.Diff)
	})

	t.Run("with engine config rules", func(t *testing.T) {
		appCfg := &config.Config{}
		appCfg.Engines.Style.Config = map[string]any{
			"rules": map[string]any{
				"blank-line-between-blocks": map[string]any{
					"enabled":  true,
					"severity": "warning",
				},
			},
		}

		cfg := buildStyleConfig(appCfg, false)
		require.Contains(t, cfg.Rules, "blank-line-between-blocks")
		assert.True(t, cfg.Rules["blank-line-between-blocks"].Enabled)
		assert.Equal(t, "warning", cfg.Rules["blank-line-between-blocks"].Severity)
	})

	t.Run("with overrides", func(t *testing.T) {
		appCfg := config.DefaultConfig()
		// Engine config must be non-nil for buildStyleConfig to reach the overrides loop
		appCfg.Engines.Style.Config = map[string]any{}
		appCfg.Overrides.Rules = map[string]config.RuleConfig{
			"my-rule": {Enabled: true, Severity: "error", Config: map[string]any{"key": "val"}},
		}

		cfg := buildStyleConfig(appCfg, false)
		require.Contains(t, cfg.Rules, "my-rule")
		assert.True(t, cfg.Rules["my-rule"].Enabled)
		assert.Equal(t, "error", cfg.Rules["my-rule"].Severity)
		assert.Equal(t, map[string]any{"key": "val"}, cfg.Rules["my-rule"].Options)
	})
}

func TestBuildLintConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		cfg := buildLintConfig(nil)
		require.NotNil(t, cfg)
		assert.Equal(t, ".tflint.hcl", cfg.ConfigFile)
	})

	t.Run("with engine config", func(t *testing.T) {
		appCfg := &config.Config{}
		appCfg.Engines.Lint.Config = map[string]any{
			"config_file": "custom.hcl",
			"plugins":     []any{"aws", "google"},
		}

		cfg := buildLintConfig(appCfg)
		assert.Equal(t, "custom.hcl", cfg.ConfigFile)
		assert.Equal(t, []string{"aws", "google"}, cfg.Plugins)
	})
}

func TestBuildPolicyConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		cfg := buildPolicyConfig(nil)
		require.NotNil(t, cfg)
	})
}

func TestBuildStyleConfig_EmptyEngineConfig(t *testing.T) {
	appCfg := &config.Config{}
	// No engine config set
	cfg := buildStyleConfig(appCfg, false)
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Rules)
}

func TestBuildStyleConfig_RuleConfigTypes(t *testing.T) {
	appCfg := &config.Config{}
	appCfg.Engines.Style.Config = map[string]any{
		"rules": map[string]any{
			"rule-with-options": map[string]any{
				"enabled":  true,
				"severity": "error",
				"options":  map[string]any{"max_lines": 100},
			},
		},
	}

	cfg := buildStyleConfig(appCfg, false)
	rc := cfg.Rules["rule-with-options"]
	assert.True(t, rc.Enabled)
	assert.Equal(t, "error", rc.Severity)
	assert.Equal(t, 100, rc.Options["max_lines"])
}

// Verify the style.RuleConfig type is properly imported
var _ style.RuleConfig
