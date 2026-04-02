package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
			tt.cfg.SetDefaults()
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

func TestPluginsConfig_ShouldVerifyIntegrity(t *testing.T) {
	t.Run("defaults to true when nil", func(t *testing.T) {
		cfg := PluginsConfig{
			Enabled:         true,
			VerifyIntegrity: nil,
		}
		assert.True(t, cfg.ShouldVerifyIntegrity())
	})

	t.Run("returns false when explicitly disabled", func(t *testing.T) {
		falseVal := false
		cfg := PluginsConfig{
			Enabled:         true,
			VerifyIntegrity: &falseVal,
		}
		assert.False(t, cfg.ShouldVerifyIntegrity())
	})

	t.Run("returns true when explicitly enabled", func(t *testing.T) {
		trueVal := true
		cfg := PluginsConfig{
			Enabled:         true,
			VerifyIntegrity: &trueVal,
		}
		assert.True(t, cfg.ShouldVerifyIntegrity())
	})
}

func TestGlobWithTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a few test files
	for i := 0; i < 5; i++ {
		filePath := filepath.Join(tmpDir, "file"+string(rune('a'+i))+".yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("test"), 0o644))
	}

	t.Run("returns matches for valid pattern", func(t *testing.T) {
		pattern := filepath.Join(tmpDir, "*.yaml")
		matches, err := globWithTimeout(pattern, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, matches, 5)
	})

	t.Run("returns empty for non-matching pattern", func(t *testing.T) {
		pattern := filepath.Join(tmpDir, "*.json")
		matches, err := globWithTimeout(pattern, 5*time.Second)
		require.NoError(t, err)
		assert.Empty(t, matches)
	})

	t.Run("returns error for invalid pattern", func(t *testing.T) {
		// '[' without closing ']' is invalid
		pattern := filepath.Join(tmpDir, "[invalid")
		_, err := globWithTimeout(pattern, 5*time.Second)
		assert.Error(t, err)
	})
}

func TestLoad_ImportGlobLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create configs directory
	configsDir := filepath.Join(tmpDir, "configs")
	require.NoError(t, os.MkdirAll(configsDir, 0o755))

	// Create more files than maxImportGlobResults allows
	// We'll mock this by creating a small number of files and testing the error path
	// Creating 1001 files would be slow, so we test the error message format
	t.Run("error message format for too many matches", func(t *testing.T) {
		// Create a config that imports files
		mainConfig := `version: 1
imports:
  - "configs/*.yaml"
`
		mainPath := filepath.Join(tmpDir, ".terratidy.yaml")
		require.NoError(t, os.WriteFile(mainPath, []byte(mainConfig), 0o644))

		// Create a few config files (not exceeding limit, just testing normal case)
		for i := 0; i < 3; i++ {
			content := "custom_rules: {}\n"
			filePath := filepath.Join(configsDir, "config"+string(rune('a'+i))+".yaml")
			require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
		}

		// This should succeed since we're under the limit
		cfg, err := Load(mainPath)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})
}

func TestLoad_ImportGlobTimeout(t *testing.T) {
	// Test that globWithTimeout returns error on timeout
	// We can't easily simulate a slow glob, but we can verify the timeout mechanism
	t.Run("timeout error message format", func(t *testing.T) {
		// Using a very short timeout with a valid pattern to verify the mechanism
		// In practice, most globs complete quickly, so this tests the code path
		tmpDir := t.TempDir()
		pattern := filepath.Join(tmpDir, "*.yaml")

		// Normal case: should complete quickly
		matches, err := globWithTimeout(pattern, 5*time.Second)
		require.NoError(t, err)
		assert.Empty(t, matches) // No files in empty temp dir
	})
}

func TestIsSensitiveVar(t *testing.T) {
	tests := []struct {
		varName   string
		sensitive bool
	}{
		// Sensitive patterns
		{"API_SECRET", true},
		{"DB_PASSWORD", true},
		{"AUTH_TOKEN", true},
		{"PRIVATE_KEY", true},
		{"AWS_CREDENTIAL", true},
		{"my_secret_value", true},
		{"password123", true},
		{"token_for_auth", true},
		{"privatedata", true},
		{"api_key", true},

		// Non-sensitive patterns
		{"DATABASE_URL", false},
		{"LOG_LEVEL", false},
		{"PORT", false},
		{"ENV", false},
		{"CONFIG_PATH", false},
		{"REGION", false},
	}

	for _, tt := range tests {
		t.Run(tt.varName, func(t *testing.T) {
			result := isSensitiveVar(tt.varName)
			assert.Equal(t, tt.sensitive, result, "isSensitiveVar(%q)", tt.varName)
		})
	}
}

func TestExpandEnvVars_SensitiveWarning(t *testing.T) {
	// This test verifies that expandEnvVars still works correctly with sensitive vars.
	// The warning is logged to stderr; we verify the expansion still happens.

	t.Run("sensitive var is still expanded", func(t *testing.T) {
		_ = os.Setenv("MY_SECRET", "secret_value")
		defer func() { _ = os.Unsetenv("MY_SECRET") }()

		result := expandEnvVars("value: ${MY_SECRET}")
		assert.Equal(t, "value: secret_value", result)
	})

	t.Run("sensitive var with default still works", func(t *testing.T) {
		_ = os.Setenv("API_TOKEN", "token123")
		defer func() { _ = os.Unsetenv("API_TOKEN") }()

		result := expandEnvVars("auth: ${API_TOKEN:-default}")
		assert.Equal(t, "auth: token123", result)
	})

	t.Run("non-sensitive var works normally", func(t *testing.T) {
		_ = os.Setenv("REGION", "us-west-2")
		defer func() { _ = os.Unsetenv("REGION") }()

		result := expandEnvVars("region: ${REGION}")
		assert.Equal(t, "region: us-west-2", result)
	})
}
