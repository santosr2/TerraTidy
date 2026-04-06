package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// TestOutputStyleResults_CheckMode verifies that outputStyleResults returns an
// error in check mode when findings are present.
func TestOutputStyleResults_CheckMode(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	findings := []sdk.Finding{
		{Rule: "style.blank-lines", Message: "test", Severity: sdk.SeverityWarning, File: "main.tf"},
	}

	err := outputStyleResults(findings, true)
	assert.Error(t, err, "check mode with findings should return error")
	assert.Contains(t, err.Error(), "style issue")
}

func TestOutputStyleResults_NoCheckMode(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	findings := []sdk.Finding{
		{Rule: "style.blank-lines", Message: "test", Severity: sdk.SeverityWarning, File: "main.tf"},
	}

	err := outputStyleResults(findings, false)
	assert.NoError(t, err, "non-check mode should not return error for warnings")
}

func TestOutputStyleResults_NoFindings(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	err := outputStyleResults(nil, true)
	assert.NoError(t, err, "check mode with no findings should not error")
}

// TestOutputLintResults_WithFindings verifies that outputLintResults writes
// findings without returning an error (it delegates to outputResults which
// handles exit codes separately).
func TestOutputLintResults_WithFindings(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	findings := []sdk.Finding{
		{Rule: "lint.terraform-required-version", Message: "missing", Severity: sdk.SeverityWarning, File: "main.tf"},
	}

	err := outputLintResults(findings)
	assert.NoError(t, err)
}

func TestOutputLintResults_NoFindings(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	err := outputLintResults(nil)
	assert.NoError(t, err)
}

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

	t.Run("fix from config when CLI flag is false", func(t *testing.T) {
		appCfg := &config.Config{}
		appCfg.Engines.Style.Fix = true

		cfg := buildStyleConfig(appCfg, false)
		assert.True(t, cfg.Fix, "fix should be enabled from config")
	})

	t.Run("CLI fix flag overrides config", func(t *testing.T) {
		appCfg := &config.Config{}
		appCfg.Engines.Style.Fix = false

		cfg := buildStyleConfig(appCfg, true)
		assert.True(t, cfg.Fix, "CLI flag should override config")
	})

	t.Run("with engine config rules", func(t *testing.T) {
		appCfg := &config.Config{}
		appCfg.Engines.Style.Rules = map[string]config.RuleConfig{
			"blank-line-between-blocks": {
				Enabled:  true,
				Severity: "warning",
			},
		}

		cfg := buildStyleConfig(appCfg, false)
		require.Contains(t, cfg.Rules, "blank-line-between-blocks")
		assert.True(t, cfg.Rules["blank-line-between-blocks"].Enabled)
		assert.Equal(t, "warning", cfg.Rules["blank-line-between-blocks"].Severity)
	})

	t.Run("with overrides", func(t *testing.T) {
		appCfg := config.DefaultConfig()
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
		appCfg.Engines.Lint.ConfigFile = "custom.hcl"
		appCfg.Engines.Lint.Plugins = []string{"aws", "google"}

		cfg := buildLintConfig(appCfg)
		assert.Equal(t, "custom.hcl", cfg.ConfigFile)
		assert.Equal(t, []string{"aws", "google"}, cfg.Plugins)
	})

	t.Run("with extra args", func(t *testing.T) {
		appCfg := &config.Config{}
		appCfg.Engines.Lint.Args = []string{"--minimum-tf-version=1.0.0", "--no-color"}

		cfg := buildLintConfig(appCfg)
		assert.Equal(t, []string{"--minimum-tf-version=1.0.0", "--no-color"}, cfg.Args)
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
	appCfg.Engines.Style.Rules = map[string]config.RuleConfig{
		"rule-with-options": {
			Enabled:  true,
			Severity: "error",
			Config:   map[string]any{"max_lines": 100},
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

func TestRunAllChecksSequentialWithConfig(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(`resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`), 0o644))

	ctx := context.Background()

	t.Run("sequential mode with default config runs all enabled engines", func(t *testing.T) {
		cfg := config.DefaultConfig()
		// Policy is opt-in and disabled by default, which avoids loading rego files.
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		oldParallel := checkParallel
		checkParallel = false
		defer func() { checkParallel = oldParallel }()

		findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tfFile}, true, nil)
		require.NoError(t, err)
		assert.NotNil(t, findings)
	})

	t.Run("sequential mode skips disabled engines", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tfFile}, true, nil)
		require.NoError(t, err)
		assert.Empty(t, findings)
	})
}
