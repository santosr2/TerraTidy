package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Load with no config file (should return defaults)
	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.Version)
	assert.True(t, cfg.Engines.Fmt.Enabled)
	assert.True(t, cfg.Engines.Style.Enabled)
	assert.True(t, cfg.Engines.Lint.Enabled)
	assert.False(t, cfg.Engines.Policy.Enabled) // Policy is opt-in
}

func TestLoad_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	content := `version: 1
severity_threshold: error
fail_fast: true
parallel: false

engines:
  fmt:
    enabled: true
  style:
    enabled: false
  lint:
    enabled: true
  policy:
    enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "error", cfg.SeverityThreshold)
	assert.True(t, cfg.FailFast)
	assert.False(t, cfg.Parallel)
	assert.True(t, cfg.Engines.Fmt.Enabled)
	assert.False(t, cfg.Engines.Style.Enabled)
	assert.True(t, cfg.Engines.Lint.Enabled)
	assert.True(t, cfg.Engines.Policy.Enabled)
}

func TestLoad_WithImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main config file
	mainConfig := `version: 1
imports:
  - "configs/*.yaml"

engines:
  fmt:
    enabled: true
`
	mainPath := filepath.Join(tmpDir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainConfig), 0o644))

	// Create configs directory
	configsDir := filepath.Join(tmpDir, "configs")
	require.NoError(t, os.MkdirAll(configsDir, 0o755))

	// Create imported config
	importedConfig := `custom_rules:
  my-rule:
    enabled: true
    severity: warning
`
	importPath := filepath.Join(configsDir, "custom.yaml")
	require.NoError(t, os.WriteFile(importPath, []byte(importedConfig), 0o644))

	cfg, err := Load(mainPath)
	require.NoError(t, err)

	// Check that custom rule was imported
	assert.Contains(t, cfg.CustomRules, "my-rule")
	assert.True(t, cfg.CustomRules["my-rule"].Enabled)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	content := `version: 1
engines:
  fmt:
    enabled: [invalid yaml
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	_, err := Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config")
}

func TestLoad_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	content := `version: 99
engines:
  fmt:
    enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	_, err := Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Version: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid version",
			cfg: &Config{
				Version: 99,
			},
			wantErr: true,
		},
		{
			name: "version 0 gets defaulted",
			cfg: &Config{
				Version: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 1, cfg.Version)
	assert.True(t, cfg.Engines.Fmt.Enabled)
	assert.True(t, cfg.Engines.Style.Enabled)
	assert.True(t, cfg.Engines.Lint.Enabled)
	assert.False(t, cfg.Engines.Policy.Enabled)
	assert.Equal(t, "warning", cfg.SeverityThreshold)
	assert.False(t, cfg.FailFast)
	assert.True(t, cfg.Parallel)
}

func TestConfig_merge(t *testing.T) {
	cfg := &Config{
		CustomRules: map[string]RuleConfig{
			"rule1": {Enabled: true},
		},
		Profiles: map[string]Profile{
			"profile1": {Name: "profile1"},
		},
	}

	other := &Config{
		CustomRules: map[string]RuleConfig{
			"rule2": {Enabled: true},
		},
		Profiles: map[string]Profile{
			"profile2": {Name: "profile2"},
		},
		Overrides: OverridesConfig{
			Rules: map[string]RuleConfig{
				"override1": {Enabled: false},
			},
		},
	}

	cfg.merge(other)

	// Check that rules were merged
	assert.Contains(t, cfg.CustomRules, "rule1")
	assert.Contains(t, cfg.CustomRules, "rule2")

	// Check that profiles were merged
	assert.Contains(t, cfg.Profiles, "profile1")
	assert.Contains(t, cfg.Profiles, "profile2")

	// Check that overrides were merged
	assert.Contains(t, cfg.Overrides.Rules, "override1")
}

func TestConfig_merge_NilMaps(t *testing.T) {
	cfg := &Config{} // All maps are nil
	other := &Config{
		CustomRules: map[string]RuleConfig{
			"rule": {Enabled: true},
		},
	}

	cfg.merge(other)

	assert.NotNil(t, cfg.CustomRules)
	assert.Contains(t, cfg.CustomRules, "rule")
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err) // Should return default config
	assert.NotNil(t, cfg)
}

func TestLoadPartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")

	content := `custom_rules:
  partial-rule:
    enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := loadPartialConfig(configPath)
	require.NoError(t, err)
	assert.Contains(t, cfg.CustomRules, "partial-rule")
}

func TestLoad_WithProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	content := `version: 1

profiles:
  ci:
    profile: ci
    description: "CI profile"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true

  dev:
    profile: dev
    description: "Development profile"
    inherits: ci
    engines:
      policy:
        enabled: false
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Len(t, cfg.Profiles, 2)
	assert.Contains(t, cfg.Profiles, "ci")
	assert.Contains(t, cfg.Profiles, "dev")
	assert.Equal(t, "ci", cfg.Profiles["dev"].Inherits)
}

func TestValidate_SeverityThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
		wantErr   bool
	}{
		{"valid error", "error", false},
		{"valid warning", "warning", false},
		{"valid info", "info", false},
		{"empty allowed", "", false},
		{"invalid critical", "critical", true},
		{"invalid debug", "debug", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version:           1,
				SeverityThreshold: tt.threshold,
			}
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid severity_threshold")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_CircularInheritance(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"a": {Name: "a", Inherits: "b"},
			"b": {Name: "b", Inherits: "c"},
			"c": {Name: "c", Inherits: "a"}, // Circular!
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular inheritance")
}

func TestValidate_NonExistentInheritedProfile(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"dev": {Name: "dev", Inherits: "nonexistent"},
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inherits from non-existent profile")
}

func TestValidate_CustomRuleSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		wantErr  bool
	}{
		{"valid error", "error", false},
		{"valid warning", "warning", false},
		{"valid info", "info", false},
		{"empty allowed", "", false},
		{"invalid fatal", "fatal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: 1,
				CustomRules: map[string]RuleConfig{
					"my-rule": {Enabled: true, Severity: tt.severity},
				},
			}
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid severity")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_OverrideRuleSeverity(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Overrides: OverridesConfig{
			Rules: map[string]RuleConfig{
				"some-rule": {Enabled: true, Severity: "invalid"},
			},
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid severity")
}

func TestValidate_EmptyPluginDirectory(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Plugins: PluginsConfig{
			Enabled:     true,
			Directories: []string{"./plugins", ""},
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin directory cannot be empty")
}

func TestValidate_ValidProfileInheritance(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"base":    {Name: "base"},
			"ci":      {Name: "ci", Inherits: "base"},
			"staging": {Name: "staging", Inherits: "ci"},
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "simple variable",
			input:    "value: ${MY_VAR}",
			envVars:  map[string]string{"MY_VAR": "hello"},
			expected: "value: hello",
		},
		{
			name:     "variable with default - var set",
			input:    "value: ${MY_VAR:-default}",
			envVars:  map[string]string{"MY_VAR": "hello"},
			expected: "value: hello",
		},
		{
			name:     "variable with default - var unset",
			input:    "value: ${MY_VAR:-default}",
			envVars:  map[string]string{},
			expected: "value: default",
		},
		{
			name:     "multiple variables",
			input:    "env: ${ENV}\nregion: ${REGION:-us-east-1}",
			envVars:  map[string]string{"ENV": "prod"},
			expected: "env: prod\nregion: us-east-1",
		},
		{
			name:     "no variables",
			input:    "value: plain text",
			envVars:  map[string]string{},
			expected: "value: plain text",
		},
		{
			name:     "unset variable",
			input:    "value: ${UNSET_VAR}",
			envVars:  map[string]string{},
			expected: "value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
				defer func(key string) { _ = os.Unsetenv(key) }(k)
			}

			result := expandEnvVars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Set environment variable
	_ = os.Setenv("TT_SEVERITY", "error")
	defer func() { _ = os.Unsetenv("TT_SEVERITY") }()

	content := `version: 1
severity_threshold: ${TT_SEVERITY}
fail_fast: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "error", cfg.SeverityThreshold)
	assert.True(t, cfg.FailFast)
}

func TestLoad_WithEnvVarsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	// Ensure the variable is NOT set
	_ = os.Unsetenv("TT_MISSING_VAR")

	content := `version: 1
severity_threshold: ${TT_MISSING_VAR:-warning}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "warning", cfg.SeverityThreshold)
}

func TestGetProfile_NoInheritance(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"base": {
				Name:        "base",
				Description: "Base profile",
				Engines: Engines{
					Fmt:    EngineConfig{Enabled: true},
					Style:  EngineConfig{Enabled: true},
					Lint:   EngineConfig{Enabled: false},
					Policy: EngineConfig{Enabled: false},
				},
			},
		},
	}

	profile, err := cfg.GetProfile("base")
	require.NoError(t, err)
	assert.Equal(t, "base", profile.Name)
	assert.True(t, profile.Engines.Fmt.Enabled)
	assert.True(t, profile.Engines.Style.Enabled)
	assert.False(t, profile.Engines.Lint.Enabled)
}

func TestGetProfile_WithInheritance(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"base": {
				Name:        "base",
				Description: "Base profile",
				Engines: Engines{
					Fmt:    EngineConfig{Enabled: true},
					Style:  EngineConfig{Enabled: true},
					Lint:   EngineConfig{Enabled: true},
					Policy: EngineConfig{Enabled: true},
				},
			},
			"dev": {
				Name:            "dev",
				Inherits:        "base",
				DisabledEngines: []string{"policy"}, // Explicitly disable policy
			},
		},
	}

	profile, err := cfg.GetProfile("dev")
	require.NoError(t, err)

	// Should inherit from base
	assert.True(t, profile.Engines.Fmt.Enabled)
	assert.True(t, profile.Engines.Style.Enabled)
	assert.True(t, profile.Engines.Lint.Enabled)
	// Should be explicitly disabled
	assert.False(t, profile.Engines.Policy.Enabled)
}

func TestGetProfile_MultiLevelInheritance(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"base": {
				Name: "base",
				Engines: Engines{
					Fmt:    EngineConfig{Enabled: true},
					Style:  EngineConfig{Enabled: true},
					Lint:   EngineConfig{Enabled: true},
					Policy: EngineConfig{Enabled: true},
				},
				Overrides: OverridesConfig{
					Rules: map[string]RuleConfig{
						"rule1": {Enabled: true},
					},
				},
			},
			"ci": {
				Name:     "ci",
				Inherits: "base",
				Overrides: OverridesConfig{
					Rules: map[string]RuleConfig{
						"rule2": {Enabled: true},
					},
				},
			},
			"staging": {
				Name:            "staging",
				Inherits:        "ci",
				DisabledEngines: []string{"policy"}, // Explicitly disable policy
				Overrides: OverridesConfig{
					Rules: map[string]RuleConfig{
						"rule3": {Enabled: true},
					},
				},
			},
		},
	}

	profile, err := cfg.GetProfile("staging")
	require.NoError(t, err)

	// Should have all engines from base, with policy explicitly disabled
	assert.True(t, profile.Engines.Fmt.Enabled)
	assert.False(t, profile.Engines.Policy.Enabled)

	// Should have merged overrides from all levels
	assert.Contains(t, profile.Overrides.Rules, "rule1")
	assert.Contains(t, profile.Overrides.Rules, "rule2")
	assert.Contains(t, profile.Overrides.Rules, "rule3")
}

func TestGetProfile_NotFound(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Profiles: map[string]Profile{},
	}

	_, err := cfg.GetProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestApplyProfile(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Engines: Engines{
			Fmt:    EngineConfig{Enabled: true},
			Style:  EngineConfig{Enabled: true},
			Lint:   EngineConfig{Enabled: true},
			Policy: EngineConfig{Enabled: true},
		},
		Profiles: map[string]Profile{
			"minimal": {
				Name: "minimal",
				Engines: Engines{
					Fmt:    EngineConfig{Enabled: true},
					Style:  EngineConfig{Enabled: false},
					Lint:   EngineConfig{Enabled: false},
					Policy: EngineConfig{Enabled: false},
				},
			},
		},
	}

	err := cfg.ApplyProfile("minimal")
	require.NoError(t, err)

	// Config should now reflect the profile settings
	assert.True(t, cfg.Engines.Fmt.Enabled)
	assert.False(t, cfg.Engines.Style.Enabled)
	assert.False(t, cfg.Engines.Lint.Enabled)
	assert.False(t, cfg.Engines.Policy.Enabled)
}
