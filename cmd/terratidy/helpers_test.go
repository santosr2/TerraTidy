package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/cache"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestIsHCLFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.tf", true},
		{"MAIN.TF", true},
		{"variables.tf", true},
		{"config.hcl", true},
		{"terraform.tfvars", true},
		{"main.go", false},
		{"README.md", false},
		{"config.json", false},
		{"module/main.tf", true},
		{"path/to/file.hcl", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isHCLFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{"normal directory", "modules", false},
		{"hidden directory", ".git", true},
		{"terraform cache", ".terraform", true},
		{"terragrunt cache", ".terragrunt-cache", true},
		{"node_modules", "node_modules", true},
		{"vendor", "vendor", true},
		{"pycache", "__pycache__", true},
		{"current dir", ".", false},
		{"src directory", "src", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipDir("", tt.dirName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPathWithin(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		dirPath  string
		expected bool
	}{
		{
			name:     "file directly in directory",
			filePath: "/project/main.tf",
			dirPath:  "/project",
			expected: true,
		},
		{
			name:     "file in subdirectory",
			filePath: "/project/modules/vpc/main.tf",
			dirPath:  "/project/modules",
			expected: true,
		},
		{
			name:     "file outside directory",
			filePath: "/other/main.tf",
			dirPath:  "/project",
			expected: false,
		},
		{
			name:     "prefix mismatch",
			filePath: "/project-other/main.tf",
			dirPath:  "/project",
			expected: false,
		},
		{
			name:     "exact match",
			filePath: "/project/main.tf",
			dirPath:  "/project/main.tf",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPathWithin(tt.filePath, tt.dirPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFileCount(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "0 files"},
		{1, "1 file"},
		{2, "2 files"},
		{10, "10 files"},
		{100, "100 files"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFileCount(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToAbsPath(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		result := toAbsPath("main.tf")
		assert.True(t, filepath.IsAbs(result))
	})

	t.Run("absolute path stays absolute", func(t *testing.T) {
		// Use temp dir to get a valid absolute path on any OS
		tmpDir := t.TempDir()
		absPath := filepath.Join(tmpDir, "main.tf")
		result := toAbsPath(absPath)
		assert.True(t, filepath.IsAbs(result))
		assert.Contains(t, result, "main.tf")
	})
}

func TestFindHCLFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("# test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte("# test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# readme"), 0o644))

	// Create subdirectory with terraform files
	subDir := filepath.Join(tmpDir, "modules", "vpc")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "main.tf"), []byte("# vpc"), 0o644))

	// Create hidden directory that should be skipped
	hiddenDir := filepath.Join(tmpDir, ".terraform")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "cached.tf"), []byte("# cache"), 0o644))

	t.Run("finds all HCL files", func(t *testing.T) {
		files, err := findHCLFiles([]string{tmpDir})
		require.NoError(t, err)
		assert.Len(t, files, 3) // main.tf, variables.tf, modules/vpc/main.tf
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		files, err := findHCLFiles([]string{tmpDir})
		require.NoError(t, err)

		for _, f := range files {
			assert.NotContains(t, f, ".terraform")
		}
	})

	t.Run("handles single file path", func(t *testing.T) {
		singleFile := filepath.Join(tmpDir, "main.tf")
		files, err := findHCLFiles([]string{singleFile})
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("handles non-existent path", func(t *testing.T) {
		_, err := findHCLFiles([]string{"/non/existent/path"})
		assert.Error(t, err)
	})
}

func TestFindHCLFilesFromPaths(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.tf"), []byte("# test"), 0o644))

	t.Run("uses current directory when empty", func(t *testing.T) {
		// Change to temp directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(oldWd) }()

		files, err := findHCLFilesFromPaths([]string{})
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("uses provided paths", func(t *testing.T) {
		files, err := findHCLFilesFromPaths([]string{tmpDir})
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})
}

func TestFileCollector(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("# test"), 0o644))

	t.Run("collects unique files", func(t *testing.T) {
		collector := newFileCollector()
		err := collector.collectPath(tmpDir)
		require.NoError(t, err)
		assert.Len(t, collector.files, 1)

		// Collect same path again - should not duplicate
		err = collector.collectPath(tmpDir)
		require.NoError(t, err)
		assert.Len(t, collector.files, 1)
	})

	t.Run("handles single file", func(t *testing.T) {
		collector := newFileCollector()
		err := collector.collectPath(filepath.Join(tmpDir, "main.tf"))
		require.NoError(t, err)
		assert.Len(t, collector.files, 1)
	})
}

func TestFilterFindingsBySeverity(t *testing.T) {
	findings := []sdk.Finding{
		{Rule: "rule-error", Severity: sdk.SeverityError, Message: "error finding"},
		{Rule: "rule-warning", Severity: sdk.SeverityWarning, Message: "warning finding"},
		{Rule: "rule-info", Severity: sdk.SeverityInfo, Message: "info finding"},
	}

	tests := []struct {
		name          string
		threshold     string
		expectedCount int
		expectedRules []string
	}{
		{
			name:          "empty threshold returns all",
			threshold:     "",
			expectedCount: 3,
			expectedRules: []string{"rule-error", "rule-warning", "rule-info"},
		},
		{
			name:          "error threshold returns only errors",
			threshold:     "error",
			expectedCount: 1,
			expectedRules: []string{"rule-error"},
		},
		{
			name:          "warning threshold returns errors and warnings",
			threshold:     "warning",
			expectedCount: 2,
			expectedRules: []string{"rule-error", "rule-warning"},
		},
		{
			name:          "info threshold returns all",
			threshold:     "info",
			expectedCount: 3,
			expectedRules: []string{"rule-error", "rule-warning", "rule-info"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFindingsBySeverity(findings, tt.threshold)
			assert.Len(t, result, tt.expectedCount)

			var rules []string
			for _, f := range result {
				rules = append(rules, f.Rule)
			}
			assert.Equal(t, tt.expectedRules, rules)
		})
	}

	t.Run("nil findings returns nil", func(t *testing.T) {
		result := filterFindingsBySeverity(nil, "error")
		assert.Nil(t, result)
	})

	t.Run("empty findings returns empty", func(t *testing.T) {
		result := filterFindingsBySeverity([]sdk.Finding{}, "error")
		assert.Empty(t, result)
	})
}

func TestGetEffectiveSeverityThreshold(t *testing.T) {
	t.Run("returns CLI flag when set", func(t *testing.T) {
		oldVal := severityThreshold
		severityThreshold = "error"
		defer func() { severityThreshold = oldVal }()

		cfg := &config.Config{SeverityThreshold: "info"}
		result := getEffectiveSeverityThreshold(cfg)
		assert.Equal(t, "error", result)
	})

	t.Run("returns config when CLI flag empty", func(t *testing.T) {
		oldVal := severityThreshold
		severityThreshold = ""
		defer func() { severityThreshold = oldVal }()

		cfg := &config.Config{SeverityThreshold: "warning"}
		result := getEffectiveSeverityThreshold(cfg)
		assert.Equal(t, "warning", result)
	})

	t.Run("returns empty when both empty", func(t *testing.T) {
		oldVal := severityThreshold
		severityThreshold = ""
		defer func() { severityThreshold = oldVal }()

		cfg := &config.Config{SeverityThreshold: ""}
		result := getEffectiveSeverityThreshold(cfg)
		assert.Equal(t, "", result)
	})

	t.Run("handles nil config", func(t *testing.T) {
		oldVal := severityThreshold
		severityThreshold = ""
		defer func() { severityThreshold = oldVal }()

		result := getEffectiveSeverityThreshold(nil)
		assert.Equal(t, "", result)
	})

	t.Run("CLI flag overrides nil config", func(t *testing.T) {
		oldVal := severityThreshold
		severityThreshold = "error"
		defer func() { severityThreshold = oldVal }()

		result := getEffectiveSeverityThreshold(nil)
		assert.Equal(t, "error", result)
	})
}

func TestGetTargetFiles(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("# test"), 0o644))

	t.Run("without changed flag", func(t *testing.T) {
		files, err := getTargetFiles([]string{tmpDir}, false)
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("with changed flag outside git repo", func(t *testing.T) {
		// Create a non-git directory
		nonGitDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(nonGitDir, "main.tf"), []byte("# test"), 0o644))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(nonGitDir)
		defer func() { _ = os.Chdir(oldWd) }()

		_, err := getTargetFiles([]string{nonGitDir}, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}

func TestConfigureCacheFromConfig(t *testing.T) {
	// Reset cache to defaults after each test
	t.Cleanup(func() {
		cache.ResetDefault()
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		cache.ResetDefault()
		configureCacheFromConfig(nil)

		stats := cache.Default().Stats()
		assert.Equal(t, 5*time.Minute, stats.MaxAge)
		assert.Equal(t, 1000, stats.MaxSize)
		assert.False(t, stats.Disabled)
	})

	t.Run("unconfigured cache uses defaults", func(t *testing.T) {
		cache.ResetDefault()
		cfg := &config.Config{} // Empty config, no cache settings
		configureCacheFromConfig(cfg)

		stats := cache.Default().Stats()
		assert.Equal(t, 5*time.Minute, stats.MaxAge)
		assert.Equal(t, 1000, stats.MaxSize)
		assert.False(t, stats.Disabled)
	})

	t.Run("max_age from config", func(t *testing.T) {
		cache.ResetDefault()
		cfg := &config.Config{}
		cfg.Cache.MaxAge = config.Duration(10 * time.Minute)

		configureCacheFromConfig(cfg)

		stats := cache.Default().Stats()
		assert.Equal(t, 10*time.Minute, stats.MaxAge)
		assert.Equal(t, 1000, stats.MaxSize) // default
	})

	t.Run("max_size from config", func(t *testing.T) {
		cache.ResetDefault()
		cfg := &config.Config{}
		cfg.Cache.MaxSize = 500

		configureCacheFromConfig(cfg)

		stats := cache.Default().Stats()
		assert.Equal(t, 5*time.Minute, stats.MaxAge) // default
		assert.Equal(t, 500, stats.MaxSize)
	})

	t.Run("disabled from config", func(t *testing.T) {
		cache.ResetDefault()
		cfg := &config.Config{}
		cfg.Cache.Disabled = true

		configureCacheFromConfig(cfg)

		assert.True(t, cache.Default().Stats().Disabled)
	})

	t.Run("all options from config", func(t *testing.T) {
		cache.ResetDefault()
		cfg := &config.Config{}
		cfg.Cache.MaxAge = config.Duration(15 * time.Minute)
		cfg.Cache.MaxSize = 250
		cfg.Cache.Disabled = false

		configureCacheFromConfig(cfg)

		stats := cache.Default().Stats()
		assert.Equal(t, 15*time.Minute, stats.MaxAge)
		assert.Equal(t, 250, stats.MaxSize)
		assert.False(t, stats.Disabled)
	})
}

func TestCacheConfigIntegration_FromYAML(t *testing.T) {
	// Reset cache after test
	t.Cleanup(func() {
		cache.ResetDefault()
	})

	tests := []struct {
		name        string
		yaml        string
		wantMaxAge  time.Duration
		wantMaxSize int
		wantDisable bool
	}{
		{
			name: "max_age affects TTL",
			yaml: `
version: 1
cache:
  max_age: 10m
`,
			wantMaxAge:  10 * time.Minute,
			wantMaxSize: 1000,
			wantDisable: false,
		},
		{
			name: "max_size limits entries",
			yaml: `
version: 1
cache:
  max_size: 500
`,
			wantMaxAge:  5 * time.Minute,
			wantMaxSize: 500,
			wantDisable: false,
		},
		{
			name: "disabled bypasses cache",
			yaml: `
version: 1
cache:
  disabled: true
`,
			wantMaxAge:  5 * time.Minute,
			wantMaxSize: 1000,
			wantDisable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.ResetDefault()

			// Write config file
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tt.yaml), 0o600))

			// Load config and configure cache
			cfg, err := config.Load(cfgPath)
			require.NoError(t, err)
			configureCacheFromConfig(cfg)

			// Verify cache settings
			stats := cache.Default().Stats()
			assert.Equal(t, tt.wantMaxAge, stats.MaxAge)
			assert.Equal(t, tt.wantMaxSize, stats.MaxSize)
			assert.Equal(t, tt.wantDisable, stats.Disabled)
		})
	}
}

func TestProfileConfigIntegration_FromYAML(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		profileName     string
		wantFmtEnabled  bool
		wantStyleEnable bool
		wantLintEnabled bool
		wantError       string
	}{
		{
			name: "profile selectable via --profile",
			yaml: `
version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
profiles:
  minimal:
    engines:
      fmt:
        enabled: true
      style:
        enabled: false
      lint:
        enabled: false
`,
			profileName:     "minimal",
			wantFmtEnabled:  true,
			wantStyleEnable: false,
			wantLintEnabled: false,
		},
		{
			name: "nonexistent profile returns error",
			yaml: `
version: 1
profiles:
  ci:
    engines:
      fmt:
        enabled: true
`,
			profileName: "production",
			wantError:   "not found",
		},
		{
			name: "profile inherits parent settings with child overrides",
			yaml: `
version: 1
profiles:
  base:
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
  child:
    inherits: base
    engines:
      lint:
        enabled: false
`,
			profileName:     "child",
			wantFmtEnabled:  true,
			wantStyleEnable: true,
			wantLintEnabled: false, // Child overrides parent
		},
		{
			name: "circular inheritance detected and rejected",
			yaml: `
version: 1
profiles:
  a:
    inherits: b
    engines:
      fmt:
        enabled: true
  b:
    inherits: c
    engines:
      fmt:
        enabled: true
  c:
    inherits: a
    engines:
      fmt:
        enabled: true
`,
			profileName: "a",
			wantError:   "circular inheritance",
		},
		{
			name: "profile engines override base config",
			yaml: `
version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
profiles:
  strict:
    engines:
      fmt:
        enabled: false
      style:
        enabled: false
      lint:
        enabled: false
`,
			profileName:     "strict",
			wantFmtEnabled:  false, // Profile overrides base
			wantStyleEnable: false,
			wantLintEnabled: false,
		},
		{
			name: "profile selectively disables engines",
			yaml: `
version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
profiles:
  fast:
    engines:
      fmt:
        enabled: true
      style:
        enabled: false
      lint:
        enabled: false
`,
			profileName:     "fast",
			wantFmtEnabled:  true,  // Kept enabled
			wantStyleEnable: false, // Disabled in profile
			wantLintEnabled: false, // Disabled in profile
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global state
			oldCfgFile := cfgFile
			oldProfile := profile
			t.Cleanup(func() {
				cfgFile = oldCfgFile
				profile = oldProfile
			})

			// Write config file
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tt.yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = tt.profileName

			// Load config (uses global cfgFile and profile)
			cfg, err := loadConfig()

			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantFmtEnabled, cfg.Engines.Fmt.IsEnabled())
			assert.Equal(t, tt.wantStyleEnable, cfg.Engines.Style.IsEnabled())
			assert.Equal(t, tt.wantLintEnabled, cfg.Engines.Lint.IsEnabled())
		})
	}
}

func TestProfileOverridesIntegration_FromYAML(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	yaml := `
version: 1
overrides:
  rules:
    base-rule:
      enabled: true
      severity: warning
profiles:
  strict:
    overrides:
      rules:
        profile-rule:
          enabled: true
          severity: error
        base-rule:
          severity: error
`
	// Write config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0o600))

	// Set globals to simulate CLI flags
	cfgFile = cfgPath
	profile = "strict"

	// Load config
	cfg, err := loadConfig()
	require.NoError(t, err)

	// Profile overrides should be merged into config
	assert.Contains(t, cfg.Overrides.Rules, "base-rule", "base rule should still exist")
	assert.Contains(t, cfg.Overrides.Rules, "profile-rule", "profile rule should be added")

	// Profile override should take precedence for severity
	assert.Equal(t, "error", cfg.Overrides.Rules["base-rule"].Severity, "profile should override base severity")
	assert.Equal(t, "error", cfg.Overrides.Rules["profile-rule"].Severity, "profile rule severity should be set")
}

// TestOverridesRulesEnabled_DisablesRule verifies that overrides.rules.<name>.enabled: false
// from YAML config prevents the rule from producing findings.
func TestOverridesRulesEnabled_DisablesRule(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	dir := t.TempDir()

	// Create test TF file that would trigger blank-line-between-blocks
	// (two consecutive blocks with no blank line between them)
	tfContent := `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	tests := []struct {
		name          string
		yaml          string
		ruleDisabled  bool
		expectFinding bool
	}{
		{
			name: "rule enabled (default) produces findings",
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
`,
			ruleDisabled:  false,
			expectFinding: true,
		},
		{
			name: "rule disabled via overrides produces no findings",
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: false
`,
			ruleDisabled:  true,
			expectFinding: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Write config file
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tc.yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = ""

			// Load config
			cfg, err := loadConfig()
			require.NoError(t, err)

			// Verify rule config is correctly loaded
			if tc.ruleDisabled {
				require.Contains(t, cfg.Overrides.Rules, "style.blank-line-between-blocks")
				assert.False(t, cfg.Overrides.Rules["style.blank-line-between-blocks"].Enabled,
					"rule should be disabled in config")
			}

			// Build style config and run engine
			styleCfg := buildStyleConfig(cfg, false)
			engine := style.New(styleCfg)
			findings, err := engine.Run(context.Background(), []string{tfFile})
			require.NoError(t, err)

			// Check findings for the specific rule
			hasBlankLineFinding := false
			for _, f := range findings {
				if f.Rule == "style.blank-line-between-blocks" {
					hasBlankLineFinding = true
					break
				}
			}

			if tc.expectFinding {
				assert.True(t, hasBlankLineFinding,
					"expected finding for blank-line-between-blocks when rule is enabled")
			} else {
				assert.False(t, hasBlankLineFinding,
					"expected no finding for blank-line-between-blocks when rule is disabled")
			}
		})
	}
}

// TestOverridesRulesSeverity_ChangesFindingSeverity verifies that overrides.rules.<name>.severity
// from YAML config changes the severity of findings produced by that rule.
func TestOverridesRulesSeverity_ChangesFindingSeverity(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	dir := t.TempDir()

	// Create test TF file that triggers blank-line-between-blocks (default severity: warning)
	tfContent := `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	tests := []struct {
		name             string
		yaml             string
		expectedSeverity sdk.Severity
	}{
		{
			name: "default severity (warning)",
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
`,
			expectedSeverity: sdk.SeverityWarning,
		},
		{
			name: "severity overridden to error",
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
      severity: error
`,
			expectedSeverity: sdk.SeverityError,
		},
		{
			name: "severity overridden to info",
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
      severity: info
`,
			expectedSeverity: sdk.SeverityInfo,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Write config file
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tc.yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = ""

			// Load config
			cfg, err := loadConfig()
			require.NoError(t, err)

			// Build style config and run engine
			styleCfg := buildStyleConfig(cfg, false)
			engine := style.New(styleCfg)
			findings, err := engine.Run(context.Background(), []string{tfFile})
			require.NoError(t, err)

			// Find the blank-line-between-blocks finding and check severity
			var foundFinding *sdk.Finding
			for i := range findings {
				if findings[i].Rule == "style.blank-line-between-blocks" {
					foundFinding = &findings[i]
					break
				}
			}

			require.NotNil(t, foundFinding, "expected to find blank-line-between-blocks finding")
			assert.Equal(t, tc.expectedSeverity, foundFinding.Severity,
				"finding severity should match config override")
		})
	}
}

// TestOverridesRulesConfig_AppliesRuleOptions verifies that overrides.rules.<name>.config
// from YAML config applies rule-specific options.
func TestOverridesRulesConfig_AppliesRuleOptions(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	dir := t.TempDir()

	tests := []struct {
		name          string
		tfContent     string
		yaml          string
		expectFinding bool
		description   string
	}{
		{
			name: "default config (1 blank line required) - 0 blank lines produces finding",
			tfContent: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
`,
			expectFinding: true,
			description:   "0 blank lines should trigger finding with default config",
		},
		{
			name: "config with min_lines=0 - 0 blank lines is valid",
			tfContent: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
      config:
        options:
          min_lines: 0
          max_lines: 1
`,
			expectFinding: false,
			description:   "0 blank lines should be valid when min_lines=0",
		},
		{
			name: "config with min_lines=2 - 1 blank line produces finding",
			tfContent: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}

resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
      config:
        options:
          min_lines: 2
          max_lines: 3
`,
			expectFinding: true,
			description:   "1 blank line should trigger finding when min_lines=2",
		},
		{
			name: "config with min_lines=2 - 2 blank lines is valid",
			tfContent: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}


resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.blank-line-between-blocks:
      enabled: true
      config:
        options:
          min_lines: 2
          max_lines: 3
`,
			expectFinding: false,
			description:   "2 blank lines should be valid when min_lines=2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Write TF file
			tfFile := filepath.Join(dir, "main.tf")
			require.NoError(t, os.WriteFile(tfFile, []byte(tc.tfContent), 0o644))

			// Write config file
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tc.yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = ""

			// Load config
			cfg, err := loadConfig()
			require.NoError(t, err)

			// Build style config and run engine
			styleCfg := buildStyleConfig(cfg, false)
			engine := style.New(styleCfg)
			findings, err := engine.Run(context.Background(), []string{tfFile})
			require.NoError(t, err)

			// Check for blank-line-between-blocks finding
			hasBlankLineFinding := false
			for _, f := range findings {
				if f.Rule == "style.blank-line-between-blocks" {
					hasBlankLineFinding = true
					break
				}
			}

			if tc.expectFinding {
				assert.True(t, hasBlankLineFinding, tc.description)
			} else {
				assert.False(t, hasBlankLineFinding, tc.description)
			}
		})
	}
}

// TestVariableNamingConvention_RespectsConfig verifies that style.variable-naming
// rule respects the naming convention option from config.
func TestVariableNamingConvention_RespectsConfig(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	dir := t.TempDir()

	tests := []struct {
		name          string
		tfContent     string
		yaml          string
		expectFinding bool
		description   string
	}{
		{
			name: "default snake_case - camelCase variable produces finding",
			tfContent: `variable "instanceType" {
  description = "The instance type"
  type        = string
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.variable-naming:
      enabled: true
`,
			expectFinding: true,
			description:   "camelCase variable should trigger finding with default snake_case convention",
		},
		{
			name: "default snake_case - snake_case variable is valid",
			tfContent: `variable "instance_type" {
  description = "The instance type"
  type        = string
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.variable-naming:
      enabled: true
`,
			expectFinding: false,
			description:   "snake_case variable should be valid with default convention",
		},
		{
			name: "camelCase convention - camelCase variable is valid",
			tfContent: `variable "instanceType" {
  description = "The instance type"
  type        = string
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.variable-naming:
      enabled: true
      config:
        options:
          case: camelCase
`,
			expectFinding: false,
			description:   "camelCase variable should be valid when convention is camelCase",
		},
		{
			name: "camelCase convention - snake_case variable produces finding",
			tfContent: `variable "instance_type" {
  description = "The instance type"
  type        = string
}
`,
			yaml: `
version: 1
engines:
  style:
    enabled: true
overrides:
  rules:
    style.variable-naming:
      enabled: true
      config:
        options:
          case: camelCase
`,
			expectFinding: true,
			description:   "snake_case variable should trigger finding when convention is camelCase",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Write TF file
			tfFile := filepath.Join(dir, "variables.tf")
			require.NoError(t, os.WriteFile(tfFile, []byte(tc.tfContent), 0o644))

			// Write config file
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tc.yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = ""

			// Load config
			cfg, err := loadConfig()
			require.NoError(t, err)

			// Build style config and run engine
			styleCfg := buildStyleConfig(cfg, false)
			engine := style.New(styleCfg)
			findings, err := engine.Run(context.Background(), []string{tfFile})
			require.NoError(t, err)

			// Check for variable-naming finding
			hasNamingFinding := false
			for _, f := range findings {
				if f.Rule == "style.variable-naming" {
					hasNamingFinding = true
					break
				}
			}

			if tc.expectFinding {
				assert.True(t, hasNamingFinding, tc.description)
			} else {
				assert.False(t, hasNamingFinding, tc.description)
			}
		})
	}
}

// TestPluginsEnabled_FromYAMLConfig verifies that plugins.enabled from YAML config
// controls whether plugin rules are loaded.
func TestPluginsEnabled_FromYAMLConfig(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	dir := t.TempDir()

	// Create a plugin directory with a YAML rule
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	yamlRule := `name: test-yaml-plugin
description: Test plugin rule
severity: warning
enabled: true
message: "Missing test_attr attribute"
patterns:
  required_attributes:
    - test_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "test-rule.yaml"), []byte(yamlRule), 0o644))

	// Create test TF file that would trigger the plugin rule
	tfContent := `resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	tests := []struct {
		name              string
		pluginsEnabled    bool
		expectPluginRules bool
	}{
		{
			name:              "plugins disabled - no plugin rules loaded",
			pluginsEnabled:    false,
			expectPluginRules: false,
		},
		{
			name:              "plugins enabled - plugin rules loaded",
			pluginsEnabled:    true,
			expectPluginRules: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Generate YAML config with absolute plugin directory path
			yaml := fmt.Sprintf(`
version: 1
engines:
  style:
    enabled: true
plugins:
  enabled: %t
  directories:
    - %s
`, tc.pluginsEnabled, pluginDir)

			// Write config file
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = ""

			// Load config
			cfg, err := loadConfig()
			require.NoError(t, err)

			// Verify plugins.enabled is correctly loaded
			assert.Equal(t, tc.pluginsEnabled, cfg.Plugins.Enabled,
				"plugins.enabled should match YAML config")

			// Load plugin rules
			pluginRules, err := loadPluginRules(cfg)
			require.NoError(t, err)

			if tc.expectPluginRules {
				assert.NotEmpty(t, pluginRules, "plugin rules should be loaded when plugins.enabled=true")
				// Verify the specific rule was loaded
				found := false
				for _, r := range pluginRules {
					if r.Name() == "test-yaml-plugin" {
						found = true
						break
					}
				}
				assert.True(t, found, "test-yaml-plugin should be loaded")
			} else {
				assert.Empty(t, pluginRules, "no plugin rules should be loaded when plugins.enabled=false")
			}
		})
	}
}

// TestPluginsDirectories_FromYAMLConfig verifies that plugins.directories from YAML config
// correctly specifies where plugin rules are loaded from.
func TestPluginsDirectories_FromYAMLConfig(t *testing.T) {
	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
	})

	dir := t.TempDir()

	// Create two plugin directories with different rules
	pluginDir1 := filepath.Join(dir, "plugins1")
	pluginDir2 := filepath.Join(dir, "plugins2")
	require.NoError(t, os.MkdirAll(pluginDir1, 0o755))
	require.NoError(t, os.MkdirAll(pluginDir2, 0o755))

	// Rule in first directory
	yamlRule1 := `name: rule-from-dir1
description: Rule from directory 1
severity: warning
enabled: true
message: "Finding from dir1"
patterns:
  required_attributes:
    - attr1
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir1, "rule1.yaml"), []byte(yamlRule1), 0o644))

	// Rule in second directory
	yamlRule2 := `name: rule-from-dir2
description: Rule from directory 2
severity: warning
enabled: true
message: "Finding from dir2"
patterns:
  required_attributes:
    - attr2
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir2, "rule2.yaml"), []byte(yamlRule2), 0o644))

	tests := []struct {
		name          string
		directories   []string
		expectedRules []string
	}{
		{
			name:          "single directory loads its rules",
			directories:   []string{pluginDir1},
			expectedRules: []string{"rule-from-dir1"},
		},
		{
			name:          "multiple directories load all rules",
			directories:   []string{pluginDir1, pluginDir2},
			expectedRules: []string{"rule-from-dir1", "rule-from-dir2"},
		},
		{
			name:          "empty directories list loads no rules",
			directories:   []string{},
			expectedRules: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build directories YAML list
			dirsYAML := ""
			for _, d := range tc.directories {
				dirsYAML += fmt.Sprintf("    - %s\n", d)
			}

			yaml := fmt.Sprintf(`
version: 1
engines:
  style:
    enabled: true
plugins:
  enabled: true
  directories:
%s`, dirsYAML)

			// Write config file
			cfgPath := filepath.Join(dir, ".terratidy.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0o600))

			// Set globals to simulate CLI flags
			cfgFile = cfgPath
			profile = ""

			// Load config
			cfg, err := loadConfig()
			require.NoError(t, err)

			// Verify directories are correctly loaded
			// Note: empty slice becomes nil in YAML parsing
			if len(tc.directories) == 0 {
				assert.Empty(t, cfg.Plugins.Directories,
					"plugins.directories should be empty")
			} else {
				assert.Equal(t, tc.directories, cfg.Plugins.Directories,
					"plugins.directories should match YAML config")
			}

			// Load plugin rules
			pluginRules, err := loadPluginRules(cfg)
			require.NoError(t, err)

			// Verify expected rules are loaded
			loadedRuleNames := make([]string, 0, len(pluginRules))
			for _, r := range pluginRules {
				loadedRuleNames = append(loadedRuleNames, r.Name())
			}

			assert.Len(t, pluginRules, len(tc.expectedRules),
				"should load expected number of rules")

			for _, expectedRule := range tc.expectedRules {
				assert.Contains(t, loadedRuleNames, expectedRule,
					"rule %s should be loaded", expectedRule)
			}
		})
	}
}
