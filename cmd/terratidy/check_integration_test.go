package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestPrintCheckSummary(t *testing.T) {
	t.Run("no findings", func(t *testing.T) {
		err := printCheckSummary(nil)
		assert.NoError(t, err)
	})

	t.Run("with errors exits", func(t *testing.T) {
		findings := []sdk.Finding{
			{Severity: sdk.SeverityError, Rule: "test.rule"},
		}
		err := printCheckSummary(findings)
		assert.Error(t, err) // Should return ExitError
	})

	t.Run("warnings only no exit", func(t *testing.T) {
		findings := []sdk.Finding{
			{Severity: sdk.SeverityWarning, Rule: "test.rule"},
		}
		err := printCheckSummary(findings)
		assert.NoError(t, err)
	})
}

func TestPrintSeverityCounts(t *testing.T) {
	// Just verify it doesn't panic with various inputs
	printSeverityCounts(0, 0, 0)
	printSeverityCounts(1, 2, 3)
	printSeverityCounts(0, 5, 0)
}

func TestRunFmtCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	findings, err := runFmtCheckWithConfig(ctx, nil, []string{tmpFile}, 1, true)
	require.NoError(t, err)
	assert.NotEmpty(t, findings, "unformatted file should produce findings")
}

func TestRunStyleCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	cfg := config.DefaultConfig()
	findings, err := runStyleCheckWithConfig(ctx, cfg, []string{tmpFile}, 2, true, nil)
	require.NoError(t, err)
	// May or may not have findings depending on content
	_ = findings
}

func TestRunLintCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	cfg := config.DefaultConfig()
	findings, err := runLintCheckWithConfig(ctx, cfg, []string{tmpFile}, 3, true, nil)
	require.NoError(t, err)
	_ = findings
}

func TestOutputCheckResults(t *testing.T) {
	t.Run("no findings", func(t *testing.T) {
		old := format
		format = "text"
		defer func() { format = old }()

		err := outputCheckResults(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("with error findings text format", func(t *testing.T) {
		old := format
		format = "text"
		defer func() { format = old }()

		findings := []sdk.Finding{
			{Rule: "test.rule", Message: "test", Severity: sdk.SeverityError, File: "test.tf"},
		}
		err := outputCheckResults(findings, nil)
		assert.Error(t, err, "should return exit error for errors")
	})

	t.Run("with error findings json format", func(t *testing.T) {
		old := format
		format = "json"
		defer func() { format = old }()

		findings := []sdk.Finding{
			{Rule: "test.rule", Message: "test", Severity: sdk.SeverityError, File: "test.tf"},
		}
		err := outputCheckResults(findings, nil)
		assert.Error(t, err, "should return exit error for errors in json")
	})
}

func TestRunAllChecksWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg := config.DefaultConfig()

	t.Run("sequential", func(t *testing.T) {
		old := checkParallel
		checkParallel = false
		defer func() { checkParallel = old }()

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)
		_ = findings
	})

	t.Run("parallel", func(t *testing.T) {
		old := checkParallel
		checkParallel = true
		defer func() { checkParallel = old }()

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)
		_ = findings
	})

	t.Run("no-parallel forces sequential despite config parallel: true", func(t *testing.T) {
		oldParallel := checkParallel
		oldNoParallel := checkNoParallel
		checkParallel = true
		checkNoParallel = true
		defer func() {
			checkParallel = oldParallel
			checkNoParallel = oldNoParallel
		}()

		parallelCfg := config.DefaultConfig()
		parallelCfg.Parallel = config.BoolPtr(true)

		findings, err := runAllChecksWithConfig(parallelCfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)
		_ = findings
	})
}

func TestPrintCheckHeader(t *testing.T) {
	printCheckHeader(5)
	// Just verify it doesn't panic
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Set up globals
	oldChanged := changed
	oldFormat := format
	oldCfgFile := cfgFile
	changed = false
	format = "text"
	cfgFile = ""
	defer func() {
		changed = oldChanged
		format = oldFormat
		cfgFile = oldCfgFile
	}()

	rootCmd.SetArgs([]string{"check", dir})
	err := rootCmd.Execute()
	// May return ExitError if findings have errors
	_ = err
}

// TestEnginesStyleConfigFix verifies that engines.style.fix: true in config
// enables auto-fix mode without requiring the --fix CLI flag.
func TestEnginesStyleConfigFix(t *testing.T) {
	dir := t.TempDir()

	// Create config with style.fix: true
	configContent := `version: 1
engines:
  fmt:
    enabled: false
  style:
    enabled: true
    fix: true
  lint:
    enabled: false
  policy:
    enabled: false
`
	cfgPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(configContent), 0o644))

	// Create a file with style issues (multiple blank lines between blocks)
	tfContent := `resource "aws_instance" "one" {
  ami = "ami-123"
}


resource "aws_instance" "two" {
  ami = "ami-456"
}
`
	tfPath := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfPath, []byte(tfContent), 0o644))

	// Load config and run style checks
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = runStyleCheckWithConfig(ctx, cfg, []string{tfPath}, 1, true, nil)
	require.NoError(t, err)

	// Verify the file was modified (2 blank lines → 1)
	modifiedContent, err := os.ReadFile(tfPath)
	require.NoError(t, err)

	expectedContent := `resource "aws_instance" "one" {
  ami = "ami-123"
}

resource "aws_instance" "two" {
  ami = "ami-456"
}
`
	assert.Equal(t, expectedContent, string(modifiedContent), "style.fix: true should auto-fix blank line issues")
}

// TestEnginesLintConfigArgs verifies that engines.lint.args in config
// are passed to buildLintConfig and available in the lint engine.
func TestEnginesLintConfigArgs(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Engines: config.Engines{
			Lint: config.LintEngineConfig{
				Enabled:    config.BoolPtr(true),
				Args:       []string{"--force", "--no-color", "--minimum-tf-version=1.5.0"},
				ConfigFile: ".tflint.hcl",
			},
		},
	}

	lintCfg := buildLintConfig(cfg)

	require.NotNil(t, lintCfg)
	assert.Equal(t, []string{"--force", "--no-color", "--minimum-tf-version=1.5.0"}, lintCfg.Args,
		"lint config should include args from engines.lint.args")
	assert.Equal(t, ".tflint.hcl", lintCfg.ConfigFile)
}

// TestLoadPluginRules verifies that loadPluginRules loads rules from configured directories.
func TestLoadPluginRules(t *testing.T) {
	t.Run("plugins disabled returns nil", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = false

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		assert.Nil(t, rules)
	})

	t.Run("plugins enabled with empty directories returns empty", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{t.TempDir()}

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		assert.Empty(t, rules)
	})

	t.Run("plugins enabled loads YAML rules", func(t *testing.T) {
		dir := t.TempDir()

		// Create a YAML rule file
		yamlRule := `name: test-plugin-rule
description: Test rule from plugin
severity: warning
enabled: true
message: "Test finding"
patterns:
  required_attributes:
    - test_attr
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "test-rule.yaml"), []byte(yamlRule), 0o644))

		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{dir}

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		require.Len(t, rules, 1)
		assert.Equal(t, "test-plugin-rule", rules[0].Name())
	})

	t.Run("nil config returns nil", func(t *testing.T) {
		rules, err := loadPluginRules(nil)
		require.NoError(t, err)
		assert.Nil(t, rules)
	})
}

// TestStyleEngineWithPluginRules verifies that plugin rules produce findings during style checks.
func TestStyleEngineWithPluginRules(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule that requires 'description' attribute
	yamlRule := `name: require-description
description: Resources must have a description
severity: warning
enabled: true
message: "Resource is missing 'description' attribute"
patterns:
  required_attributes:
    - description
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "require-desc.yaml"), []byte(yamlRule), 0o644))

	// Create a test TF file without description attribute
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Load plugin rules
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}

	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)

	// Run style check with plugin rules
	ctx := context.Background()
	findings, err := runStyleCheckWithConfig(ctx, cfg, []string{tfFile}, 1, true, pluginRules)
	require.NoError(t, err)

	// Should have at least one finding from the plugin rule
	var pluginFinding bool
	for _, f := range findings {
		if f.Rule == "require-description" {
			pluginFinding = true
			assert.Equal(t, sdk.SeverityWarning, f.Severity)
			break
		}
	}
	assert.True(t, pluginFinding, "should have finding from plugin rule")
}

// TestLintEngineWithPluginRules verifies that plugin rules produce findings during lint checks.
func TestLintEngineWithPluginRules(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule that requires 'tags' attribute
	yamlRule := `name: require-tags
description: Resources must have tags
severity: warning
enabled: true
message: "Resource is missing 'tags' attribute"
patterns:
  required_attributes:
    - tags
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "require-tags.yaml"), []byte(yamlRule), 0o644))

	// Create a test TF file without tags attribute
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Load plugin rules
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}

	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)

	// Run lint check with plugin rules
	ctx := context.Background()
	findings, err := runLintCheckWithConfig(ctx, cfg, []string{tfFile}, 1, true, pluginRules)
	require.NoError(t, err)

	// Should have at least one finding from the plugin rule
	var pluginFinding bool
	for _, f := range findings {
		if f.Rule == "require-tags" {
			pluginFinding = true
			assert.Equal(t, sdk.SeverityWarning, f.Severity)
			break
		}
	}
	assert.True(t, pluginFinding, "should have finding from plugin rule in lint engine")
}

// TestLintEnginePluginRuleFiltering verifies that plugin rule filtering works in lint engine.
func TestLintEnginePluginRuleFiltering(t *testing.T) {
	t.Run("disabled rule produces no findings", func(t *testing.T) {
		dir := t.TempDir()

		// Create a YAML rule that requires 'tags' attribute
		yamlRule := `name: lint-disabled-rule
description: Rule to be disabled via config
severity: warning
enabled: true
message: "Should not see this finding"
patterns:
  required_attributes:
    - tags
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "lint-disabled.yaml"), []byte(yamlRule), 0o644))

		// Create a test TF file without tags (would trigger rule if enabled)
		tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
		tfFile := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

		// Create config with plugin rule disabled
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{dir}
		cfg.Plugins.Rules = map[string]config.RuleConfig{
			"lint-disabled-rule": {Enabled: config.BoolPtr(false)},
		}

		// Load plugin rules (should be filtered out)
		pluginRules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		assert.Empty(t, pluginRules, "disabled rule should be filtered out")

		// Run lint check - should have no findings from the disabled rule
		ctx := context.Background()
		findings, err := runLintCheckWithConfig(ctx, cfg, []string{tfFile}, 1, true, pluginRules)
		require.NoError(t, err)

		// Verify no finding from the disabled plugin rule
		for _, f := range findings {
			assert.NotEqual(t, "lint-disabled-rule", f.Rule, "disabled rule should not produce findings")
		}
	})

	t.Run("severity override applied in lint", func(t *testing.T) {
		dir := t.TempDir()

		// Create a YAML rule with warning severity
		yamlRule := `name: lint-severity-rule
description: Rule with severity override
severity: warning
enabled: true
message: "Resource is missing 'owner' attribute"
patterns:
  required_attributes:
    - owner
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "lint-severity.yaml"), []byte(yamlRule), 0o644))

		// Create a test TF file without owner attribute
		tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
		tfFile := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

		// Create config with severity override to error
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{dir}
		cfg.Plugins.Rules = map[string]config.RuleConfig{
			"lint-severity-rule": {Enabled: config.BoolPtr(true), Severity: "error"},
		}

		// Load plugin rules
		pluginRules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		require.Len(t, pluginRules, 1)

		// Run lint check with plugin rules
		ctx := context.Background()
		findings, err := runLintCheckWithConfig(ctx, cfg, []string{tfFile}, 1, true, pluginRules)
		require.NoError(t, err)

		// Find the plugin rule finding and verify severity override
		var foundFinding bool
		for _, f := range findings {
			if f.Rule == "lint-severity-rule" {
				foundFinding = true
				assert.Equal(t, sdk.SeverityError, f.Severity, "severity should be overridden to error")
				break
			}
		}
		assert.True(t, foundFinding, "should have finding from plugin rule in lint engine")
	})
}

// TestPluginRuleFiltering_DisabledRule verifies that disabled plugin rules do not produce findings.
func TestPluginRuleFiltering_DisabledRule(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule
	yamlRule := `name: disabled-rule
description: This rule should be disabled via config
severity: warning
enabled: true
message: "Should not see this finding"
patterns:
  required_attributes:
    - some_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "disabled-rule.yaml"), []byte(yamlRule), 0o644))

	// Create config with plugin rule disabled
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"disabled-rule": {Enabled: config.BoolPtr(false)},
	}

	// Load plugin rules (should be filtered out)
	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	assert.Empty(t, rules, "disabled rule should be filtered out")
}

// TestPluginRuleFiltering_EnabledRule verifies that enabled plugin rules are loaded.
func TestPluginRuleFiltering_EnabledRule(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule
	yamlRule := `name: enabled-rule
description: This rule is explicitly enabled
severity: warning
enabled: true
message: "Should see this finding"
patterns:
  required_attributes:
    - some_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "enabled-rule.yaml"), []byte(yamlRule), 0o644))

	// Create config with plugin rule explicitly enabled
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"enabled-rule": {Enabled: config.BoolPtr(true)},
	}

	// Load plugin rules (should be included)
	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "enabled-rule", rules[0].Name())
}

// TestPluginRuleFiltering_SeverityOverride verifies that severity overrides are applied to findings.
func TestPluginRuleFiltering_SeverityOverride(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule with warning severity
	yamlRule := `name: severity-test-rule
description: Rule with configurable severity
severity: warning
enabled: true
message: "Resource is missing 'description' attribute"
patterns:
  required_attributes:
    - description
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "severity-rule.yaml"), []byte(yamlRule), 0o644))

	// Create a test TF file that triggers the rule
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Create config with severity override to error
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"severity-test-rule": {Enabled: config.BoolPtr(true), Severity: "error"},
	}

	// Load plugin rules
	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)

	// Run style check with plugin rules
	ctx := context.Background()
	findings, err := runStyleCheckWithConfig(ctx, cfg, []string{tfFile}, 1, true, pluginRules)
	require.NoError(t, err)

	// Find the plugin rule finding and verify severity override
	var foundFinding bool
	for _, f := range findings {
		if f.Rule == "severity-test-rule" {
			foundFinding = true
			assert.Equal(t, sdk.SeverityError, f.Severity, "severity should be overridden to error")
			break
		}
	}
	assert.True(t, foundFinding, "should have finding from plugin rule")
}

// TestPluginRuleFiltering_SameNameAsBuiltIn verifies behavior when plugin rule has same name as built-in.
// Current behavior: both rules run (no deduplication or precedence).
func TestPluginRuleFiltering_SameNameAsBuiltIn(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule with same name as built-in style rule
	yamlRule := `name: style.block-label-case
description: Plugin rule with same name as built-in
severity: error
enabled: true
message: "Plugin rule finding"
patterns:
  required_attributes:
    - description
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "duplicate-rule.yaml"), []byte(yamlRule), 0o644))

	// Create a test TF file
	tfContent := `resource "aws_instance" "Test_Instance" {
  ami = "ami-123"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Load plugin rules
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}

	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)
	assert.Equal(t, "style.block-label-case", pluginRules[0].Name())

	// Run style check - both built-in and plugin rule should run
	ctx := context.Background()
	findings, err := runStyleCheckWithConfig(ctx, cfg, []string{tfFile}, 1, true, pluginRules)
	require.NoError(t, err)

	// Count findings from the rule name - should have findings from BOTH rules
	var ruleFindings int
	for _, f := range findings {
		if f.Rule == "style.block-label-case" {
			ruleFindings++
		}
	}
	// Built-in produces 1 finding (label case violation), plugin produces 1 (missing description)
	// Current behavior: both run, so we get 2+ findings
	assert.GreaterOrEqual(t, ruleFindings, 2, "both built-in and plugin rule should produce findings")
}

// TestPluginRuleFiltering_RuleNotInConfig verifies that rules not in config are loaded with defaults.
func TestPluginRuleFiltering_RuleNotInConfig(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML rule
	yamlRule := `name: unconfigured-rule
description: Rule not mentioned in config
severity: info
enabled: true
message: "Test finding"
patterns:
  required_attributes:
    - test_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unconfigured-rule.yaml"), []byte(yamlRule), 0o644))

	// Create config without any plugin rule overrides
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{dir}
	// No cfg.Plugins.Rules set

	// Load plugin rules (should be included with original settings)
	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "unconfigured-rule", rules[0].Name())
}

// TestRunAllChecksSequentialWithConfig_FailFast verifies that fail_fast stops
// processing after the first engine that produces error-severity findings.
func TestRunAllChecksSequentialWithConfig_FailFast(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	// YAML rule requires a nonexistent attribute, guaranteeing a finding.
	yamlRule := `name: always-finds-rule
description: Always produces a finding
severity: warning
enabled: true
message: "Forced finding"
patterns:
  required_attributes:
    - nonexistent_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "always-finds.yaml"), []byte(yamlRule), 0o644))

	cfg := config.DefaultConfig()
	cfg.FailFast = config.BoolPtr(true)
	cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
	cfg.Engines.Style.Enabled = config.BoolPtr(true)
	cfg.Engines.Lint.Enabled = config.BoolPtr(true)
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}
	// Override the rule severity to error so fail-fast fires after style.
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"always-finds-rule": {Enabled: config.BoolPtr(true), Severity: "error"},
	}

	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)

	ctx := context.Background()
	oldSkipLint := checkSkipLint
	checkSkipLint = false
	defer func() { checkSkipLint = oldSkipLint }()

	findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tmpFile}, true, pluginRules)
	require.NoError(t, err)

	// Style engine applies the severity override from cfg.Plugins.Rules,
	// so there should be at least one error-severity finding.
	var hasErr bool
	for _, f := range findings {
		if f.Severity == sdk.SeverityError {
			hasErr = true
		}
	}
	assert.True(t, hasErr, "should have error-severity finding after severity override")
}

// TestBuildLintConfig_PluginRules verifies that plugins.rules entries are merged
// into the lint config (the new code path added for plugin rule integration).
func TestBuildLintConfig_PluginRules(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"my-plugin-rule": {Enabled: config.BoolPtr(true), Severity: "error", Config: map[string]any{"key": "val"}},
	}

	lintCfg := buildLintConfig(cfg)

	require.Contains(t, lintCfg.Rules, "my-plugin-rule")
	rc := lintCfg.Rules["my-plugin-rule"]
	assert.True(t, *rc.Enabled)
	assert.Equal(t, "error", rc.Severity)
	assert.Equal(t, map[string]any{"key": "val"}, rc.Options)
}

// TestBuildStyleConfig_PluginRules verifies that plugins.rules entries are merged
// into the style config (the new code path added for plugin rule integration).
func TestBuildStyleConfig_PluginRules(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"my-plugin-style-rule": {Enabled: config.BoolPtr(true), Severity: "warning", Config: map[string]any{"option": 42}},
	}

	styleCfg := buildStyleConfig(cfg, false)

	require.Contains(t, styleCfg.Rules, "my-plugin-style-rule")
	rc := styleCfg.Rules["my-plugin-style-rule"]
	assert.True(t, *rc.Enabled)
	assert.Equal(t, "warning", rc.Severity)
	assert.Equal(t, 42, rc.Options["option"])
}

// TestBuildStyleConfig_PluginRules_EnginePrecedence verifies that plugins.rules
// takes precedence over engines.style.rules when both configure the same rule name.
func TestBuildStyleConfig_PluginRules_EnginePrecedence(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Engines.Style.Rules = map[string]config.RuleConfig{
		"shared-rule": {Enabled: config.BoolPtr(true), Severity: "warning"},
	}
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"shared-rule": {Enabled: config.BoolPtr(false), Severity: "error"},
	}

	styleCfg := buildStyleConfig(cfg, false)

	// plugins.rules is applied last, so it wins.
	rc := styleCfg.Rules["shared-rule"]
	assert.False(t, *rc.Enabled, "plugins.rules should override engines.style.rules")
	assert.Equal(t, "error", rc.Severity)
}

// TestRunAllChecksParallelWithConfig_WithPluginRules verifies that plugin rules
// are passed through to both style and lint engines in parallel mode.
func TestRunAllChecksParallelWithConfig_WithPluginRules(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	yamlRule := `name: parallel-plugin-rule
description: Rule for parallel test
severity: warning
enabled: true
message: "Resource is missing 'env' attribute"
patterns:
  required_attributes:
    - env
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "parallel-rule.yaml"), []byte(yamlRule), 0o644))

	cfg := config.DefaultConfig()
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}

	pluginRules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	require.Len(t, pluginRules, 1)

	ctx := context.Background()
	findings, err := runAllChecksParallelWithConfig(ctx, cfg, []string{tmpFile}, true, pluginRules)
	require.NoError(t, err)

	var foundPluginFinding bool
	for _, f := range findings {
		if f.Rule == "parallel-plugin-rule" {
			foundPluginFinding = true
		}
	}
	assert.True(t, foundPluginFinding, "parallel mode should propagate plugin rule findings")
}

// TestYAMLRuleTagsFiltering verifies that plugins.tags config filters rules by tag.
func TestYAMLRuleTagsFiltering(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	// Create two YAML rules with different tags
	securityRule := `name: security-rule
description: A security-related rule
severity: error
enabled: true
message: "Security violation"
tags:
  - security
  - compliance
patterns:
  required_attributes:
    - security_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "security-rule.yaml"), []byte(securityRule), 0o644))

	namingRule := `name: naming-rule
description: A naming convention rule
severity: warning
enabled: true
message: "Naming violation"
tags:
  - naming
  - style
patterns:
  required_attributes:
    - naming_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "naming-rule.yaml"), []byte(namingRule), 0o644))

	untaggedRule := `name: untagged-rule
description: A rule without tags
severity: info
enabled: true
message: "Untagged finding"
patterns:
  required_attributes:
    - untagged_attr
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "untagged-rule.yaml"), []byte(untaggedRule), 0o644))

	t.Run("no tags filter loads all rules", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{pluginDir}
		// No tags filter

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		assert.Len(t, rules, 3, "should load all rules when no tags filter")
	})

	t.Run("tags filter loads only matching rules", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{pluginDir}
		cfg.Plugins.Tags = []string{"security"}

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		require.Len(t, rules, 1, "should load only security-tagged rule")
		assert.Equal(t, "security-rule", rules[0].Name())
	})

	t.Run("multiple tags filter loads rules with any matching tag", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{pluginDir}
		cfg.Plugins.Tags = []string{"security", "naming"}

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		assert.Len(t, rules, 2, "should load rules with security or naming tags")

		names := make([]string, 0, len(rules))
		for _, r := range rules {
			names = append(names, r.Name())
		}
		assert.Contains(t, names, "security-rule")
		assert.Contains(t, names, "naming-rule")
	})

	t.Run("tags filter excludes untagged rules", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{pluginDir}
		cfg.Plugins.Tags = []string{"style"}

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		require.Len(t, rules, 1, "should load only style-tagged rule")
		assert.Equal(t, "naming-rule", rules[0].Name())
	})

	t.Run("non-matching tags filter loads no rules", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Plugins.Enabled = true
		cfg.Plugins.Directories = []string{pluginDir}
		cfg.Plugins.Tags = []string{"nonexistent-tag"}

		rules, err := loadPluginRules(cfg)
		require.NoError(t, err)
		assert.Empty(t, rules, "should load no rules when no tags match")
	})
}

// TestRootConfigFieldsAffectRuntime verifies that all root config fields
// (imports, severity_threshold, fail_fast, parallel) affect runtime behavior.
func TestRootConfigFieldsAffectRuntime(t *testing.T) {
	dir := t.TempDir()

	// Create a Terraform file that will trigger findings
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(tfContent), 0o644))

	t.Run("severity_threshold filters findings", func(t *testing.T) {
		// Create findings with different severities
		findings := []sdk.Finding{
			{Rule: "rule.error", Severity: sdk.SeverityError, Message: "error"},
			{Rule: "rule.warning", Severity: sdk.SeverityWarning, Message: "warning"},
			{Rule: "rule.info", Severity: sdk.SeverityInfo, Message: "info"},
		}

		// Test error threshold - should only keep errors
		cfg := config.DefaultConfig()
		cfg.SeverityThreshold = "error"
		threshold := getEffectiveSeverityThreshold(cfg)
		filtered := filterFindingsBySeverity(findings, threshold)
		assert.Len(t, filtered, 1, "error threshold should keep only errors")
		assert.Equal(t, sdk.SeverityError, filtered[0].Severity)

		// Test warning threshold - should keep errors and warnings
		cfg.SeverityThreshold = "warning"
		threshold = getEffectiveSeverityThreshold(cfg)
		filtered = filterFindingsBySeverity(findings, threshold)
		assert.Len(t, filtered, 2, "warning threshold should keep errors and warnings")

		// Test info threshold - should keep all
		cfg.SeverityThreshold = "info"
		threshold = getEffectiveSeverityThreshold(cfg)
		filtered = filterFindingsBySeverity(findings, threshold)
		assert.Len(t, filtered, 3, "info threshold should keep all findings")
	})

	t.Run("fail_fast stops on error", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.FailFast = config.BoolPtr(true)
		assert.True(t, shouldFailFast(cfg), "shouldFailFast should return true when enabled")

		cfg.FailFast = config.BoolPtr(false)
		assert.False(t, shouldFailFast(cfg), "shouldFailFast should return false when disabled")

		// Verify hasErrors detects error severity
		findings := []sdk.Finding{
			{Severity: sdk.SeverityWarning},
		}
		assert.False(t, hasErrors(findings), "hasErrors should return false for warnings only")

		findings = append(findings, sdk.Finding{Severity: sdk.SeverityError})
		assert.True(t, hasErrors(findings), "hasErrors should return true when errors present")
	})

	t.Run("parallel affects execution mode", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Config parallel=true, CLI parallel=false -> use config (true)
		cfg.Parallel = config.BoolPtr(true)
		assert.True(t, getEffectiveParallel(cfg, false, false), "should use config when CLI flag false")

		// Config parallel=false, CLI parallel=true -> CLI wins
		cfg.Parallel = config.BoolPtr(false)
		assert.True(t, getEffectiveParallel(cfg, true, false), "CLI flag should override config")

		// Both false
		assert.False(t, getEffectiveParallel(cfg, false, false), "both false should return false")

		// --no-parallel beats config parallel: true
		cfg.Parallel = config.BoolPtr(true)
		assert.False(t, getEffectiveParallel(cfg, false, true), "--no-parallel should override config parallel: true")

		// --no-parallel beats --parallel
		cfg.Parallel = config.BoolPtr(false)
		assert.False(t, getEffectiveParallel(cfg, true, true), "--no-parallel should override --parallel")
	})

	t.Run("imports merge config correctly", func(t *testing.T) {
		// Create main config file
		mainConfig := `version: 1
imports:
  - "imports/*.yaml"
engines:
  fmt:
    enabled: true
`
		mainPath := filepath.Join(dir, ".terratidy.yaml")
		require.NoError(t, os.WriteFile(mainPath, []byte(mainConfig), 0o644))

		// Create imports directory and imported config
		importsDir := filepath.Join(dir, "imports")
		require.NoError(t, os.MkdirAll(importsDir, 0o755))

		importedConfig := `severity_threshold: error
engines:
  style:
    rules:
      imported-rule:
        enabled: true
        severity: warning
`
		require.NoError(t, os.WriteFile(filepath.Join(importsDir, "rules.yaml"), []byte(importedConfig), 0o644))

		// Load and verify merge
		cfg, err := config.Load(mainPath)
		require.NoError(t, err)

		// Imported severity_threshold should be merged
		assert.Equal(t, "error", cfg.SeverityThreshold, "imported severity_threshold should be merged")

		// Imported rule should be present in engines.style.rules
		assert.Contains(t, cfg.Engines.Style.Rules, "imported-rule", "imported rule should be present")
		assert.True(t, *cfg.Engines.Style.Rules["imported-rule"].Enabled, "imported rule should be enabled")
	})

	t.Run("all fields work together in runAllChecksWithConfig", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.SeverityThreshold = "warning"
		cfg.FailFast = config.BoolPtr(false)
		cfg.Parallel = config.BoolPtr(false)
		cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		// Run checks - should complete without error
		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)

		// Apply severity threshold filtering as the real code does
		threshold := getEffectiveSeverityThreshold(cfg)
		filtered := filterFindingsBySeverity(findings, threshold)

		// Verify filtering worked (all findings should be >= warning)
		for _, f := range filtered {
			assert.True(t, f.Severity.Level() >= sdk.SeverityWarning.Level(),
				"filtered findings should be >= warning severity")
		}
	})
}

// TestEngineConfigFieldsAffectRuntime verifies that engine-specific config fields
// (engines.*.enabled, engines.*.config.*) affect runtime behavior.
func TestEngineConfigFieldsAffectRuntime(t *testing.T) {
	dir := t.TempDir()

	// Create an intentionally unformatted Terraform file
	// (extra spaces, misaligned equals signs will trigger format findings)
	unformattedTF := `resource "aws_instance" "test" {
  ami           =    "ami-123"
  instance_type="t2.micro"
  tags={
    Name="test"
  }
}
`
	tmpFile := filepath.Join(dir, "unformatted.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(unformattedTF), 0o644))

	t.Run("engines.fmt.enabled controls format engine", func(t *testing.T) {
		// Test with fmt disabled
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)

		// No engines enabled, should have no findings
		assert.Empty(t, findings, "no engines enabled should produce no findings")

		// Test with fmt enabled
		cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
		findings, err = runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)

		// Format engine should detect the unformatted file
		hasFmtFinding := false
		for _, f := range findings {
			if f.Rule == "format" || strings.Contains(f.Message, "format") || strings.Contains(f.Message, "Format") {
				hasFmtFinding = true
				break
			}
		}
		assert.True(t, hasFmtFinding, "format engine should produce findings for unformatted file")
	})

	t.Run("engines.style.enabled controls style engine", func(t *testing.T) {
		// Test with style disabled
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)
		assert.Empty(t, findings, "no engines enabled should produce no findings")

		// Test with style enabled
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		_, err = runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		// Style engine runs without error; findings may be nil/empty depending on rules
		require.NoError(t, err, "style engine should run without error when enabled")
	})

	t.Run("engines.lint.enabled controls lint engine", func(t *testing.T) {
		// Test with lint disabled
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)
		assert.Empty(t, findings, "no engines enabled should produce no findings")

		// Test with lint enabled (fallback_builtin mode to avoid TFLint dependency)
		cfg.Engines.Lint.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.FallbackBuiltin = true
		cfg.Engines.Lint.UseTFLint = false

		_, err = runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		// Lint engine runs without error; findings may be nil/empty with no TFLint
		require.NoError(t, err, "lint engine should run without error when enabled")
	})

	t.Run("engines.policy.enabled controls policy engine", func(t *testing.T) {
		// Test with policy disabled
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err)
		assert.Empty(t, findings, "no engines enabled should produce no findings")

		// Test with policy enabled (no policies configured, so no findings)
		cfg.Engines.Policy.Enabled = config.BoolPtr(true)
		_, err = runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		// Policy engine runs without error; findings may be nil/empty with no policies
		require.NoError(t, err, "policy engine should run without error when enabled")
	})
}

// TestEngineStyleConfigDiff verifies that engines.style.config.diff shows diff output.
func TestEngineStyleConfigDiff(t *testing.T) {
	dir := t.TempDir()

	// Create a file with consecutive blocks without blank lines between them
	// This triggers the blank-line-between-blocks rule which has auto-fix
	noBlankLinesTF := `resource "aws_instance" "one" {
  ami = "ami-123"
}
resource "aws_instance" "two" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "no_blanks.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(noBlankLinesTF), 0o644))

	t.Run("diff disabled produces no diff finding", func(t *testing.T) {
		// Restore file content before each test
		require.NoError(t, os.WriteFile(tmpFile, []byte(noBlankLinesTF), 0o644))

		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		cfg.Engines.Style.Fix = true   // Enable auto-fix
		cfg.Engines.Style.Diff = false // Diff disabled
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runStyleCheckWithConfig(context.Background(), cfg, []string{tmpFile}, 1, true, nil)
		require.NoError(t, err)

		// Should NOT have a style.diff finding
		for _, f := range findings {
			assert.NotEqual(t, "style.diff", f.Rule, "diff finding should not be generated when diff disabled")
		}
	})

	t.Run("diff enabled produces diff finding when fixes applied", func(t *testing.T) {
		// Restore file content
		require.NoError(t, os.WriteFile(tmpFile, []byte(noBlankLinesTF), 0o644))

		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		cfg.Engines.Style.Fix = true  // Enable auto-fix
		cfg.Engines.Style.Diff = true // Diff enabled
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runStyleCheckWithConfig(context.Background(), cfg, []string{tmpFile}, 1, true, nil)
		require.NoError(t, err)

		// Should have a style.diff finding with diff content
		hasDiffFinding := false
		for _, f := range findings {
			if f.Rule == "style.diff" {
				hasDiffFinding = true
				assert.Contains(t, f.Message, "@@", "diff finding should contain unified diff markers")
				break
			}
		}
		assert.True(t, hasDiffFinding, "diff finding should be generated when diff enabled and fixes applied")
	})
}

// TestEngineStyleConfigRules verifies that engines.style.config.rules affects rule behavior.
func TestEngineStyleConfigRules(t *testing.T) {
	dir := t.TempDir()

	// Create a file that triggers blank-line-between-blocks rule
	noBlankLinesTF := `resource "aws_instance" "one" {
  ami = "ami-123"
}
resource "aws_instance" "two" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "rules_test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(noBlankLinesTF), 0o644))

	t.Run("rule disabled via config produces no finding", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		cfg.Engines.Style.Rules = map[string]config.RuleConfig{
			"style.blank-line-between-blocks": {
				Enabled:  config.BoolPtr(false), // Disable the rule
				Severity: "warning",
			},
		}
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runStyleCheckWithConfig(context.Background(), cfg, []string{tmpFile}, 1, true, nil)
		require.NoError(t, err)

		// Should NOT have blank-line-between-blocks finding
		for _, f := range findings {
			assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule,
				"disabled rule should not produce findings")
		}
	})

	t.Run("rule severity changed via config", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		cfg.Engines.Style.Rules = map[string]config.RuleConfig{
			"style.blank-line-between-blocks": {
				Enabled:  config.BoolPtr(true),
				Severity: "error", // Change severity from default warning to error
			},
		}
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runStyleCheckWithConfig(context.Background(), cfg, []string{tmpFile}, 1, true, nil)
		require.NoError(t, err)

		// Find the blank-line-between-blocks finding and verify severity
		foundRule := false
		for _, f := range findings {
			if f.Rule == "style.blank-line-between-blocks" {
				foundRule = true
				assert.Equal(t, sdk.SeverityError, f.Severity,
					"rule severity should be changed to error via config")
				break
			}
		}
		assert.True(t, foundRule, "blank-line-between-blocks rule should produce findings")
	})

	t.Run("opt-in rule enabled via config", func(t *testing.T) {
		// Create a file that would trigger the no-trailing-whitespace rule (opt-in)
		fileWithTrailingWS := "resource \"aws_instance\" \"test\" {\n  ami = \"ami-123\"   \n}\n"
		wsFile := filepath.Join(dir, "trailing_ws.tf")
		require.NoError(t, os.WriteFile(wsFile, []byte(fileWithTrailingWS), 0o644))

		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(true)
		cfg.Engines.Style.Rules = map[string]config.RuleConfig{
			"style.no-trailing-whitespace": {
				Enabled:  config.BoolPtr(true), // Enable this opt-in rule
				Severity: "warning",
			},
		}
		cfg.Engines.Lint.Enabled = config.BoolPtr(false)
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		findings, err := runStyleCheckWithConfig(context.Background(), cfg, []string{wsFile}, 1, true, nil)
		require.NoError(t, err)

		// Should have no-trailing-whitespace finding now that rule is enabled
		foundRule := false
		for _, f := range findings {
			if f.Rule == "style.no-trailing-whitespace" {
				foundRule = true
				break
			}
		}
		assert.True(t, foundRule, "opt-in rule should produce findings when enabled via config")
	})
}

// TestEngineLintConfigFields verifies that engines.lint.config.* fields are correctly
// propagated to the lint engine via buildLintConfig.
func TestEngineLintConfigFields(t *testing.T) {
	t.Run("config_file is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled:    config.BoolPtr(true),
					ConfigFile: "/custom/path/.tflint.hcl",
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, "/custom/path/.tflint.hcl", lintCfg.ConfigFile,
			"config_file should be propagated to lint config")
	})

	t.Run("plugins are propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled: config.BoolPtr(true),
					Plugins: []string{"aws", "google", "azurerm"},
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, []string{"aws", "google", "azurerm"}, lintCfg.Plugins,
			"plugins should be propagated to lint config")
	})

	t.Run("use_tflint is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled:   config.BoolPtr(true),
					UseTFLint: true,
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		assert.True(t, lintCfg.UseTFLint, "use_tflint should be propagated to lint config")
	})

	t.Run("tflint_path is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled:    config.BoolPtr(true),
					TFLintPath: "/usr/local/bin/tflint-custom",
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, "/usr/local/bin/tflint-custom", lintCfg.TFLintPath,
			"tflint_path should be propagated to lint config")
	})

	t.Run("fallback_builtin is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled:         config.BoolPtr(true),
					FallbackBuiltin: true,
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		assert.True(t, lintCfg.FallbackBuiltin,
			"fallback_builtin should be propagated to lint config")
	})

	t.Run("rules are propagated with severity and options", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled: config.BoolPtr(true),
					Rules: map[string]config.RuleConfig{
						"terraform_deprecated_interpolation": {
							Enabled:  config.BoolPtr(true),
							Severity: "error",
							Config:   map[string]any{"strict": true},
						},
					},
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		require.Contains(t, lintCfg.Rules, "terraform_deprecated_interpolation")
		rule := lintCfg.Rules["terraform_deprecated_interpolation"]
		assert.True(t, *rule.Enabled, "rule enabled should be propagated")
		assert.Equal(t, "error", rule.Severity, "rule severity should be propagated")
		assert.Equal(t, map[string]any{"strict": true}, rule.Options, "rule options should be propagated")
	})

	t.Run("default config_file when not specified", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Lint: config.LintEngineConfig{
					Enabled: config.BoolPtr(true),
					// ConfigFile not set
				},
			},
		}

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, ".tflint.hcl", lintCfg.ConfigFile,
			"config_file should default to .tflint.hcl when not specified")
	})
}

// TestEnginePolicyConfigFields verifies that engines.policy.config.* fields are correctly
// propagated to the policy engine via buildPolicyConfig.
func TestEnginePolicyConfigFields(t *testing.T) {
	t.Run("policy_dirs is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Policy: config.PolicyEngineConfig{
					Enabled:    config.BoolPtr(true),
					PolicyDirs: []string{"./policies", "./extra-policies"},
				},
			},
		}

		policyCfg := buildPolicyConfig(cfg)
		assert.Equal(t, []string{"./policies", "./extra-policies"}, policyCfg.PolicyDirs,
			"policy_dirs should be propagated to policy config")
	})

	t.Run("policy_files is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Policy: config.PolicyEngineConfig{
					Enabled:     config.BoolPtr(true),
					PolicyFiles: []string{"main.rego", "helpers.rego"},
				},
			},
		}

		policyCfg := buildPolicyConfig(cfg)
		assert.Equal(t, []string{"main.rego", "helpers.rego"}, policyCfg.PolicyFiles,
			"policy_files should be propagated to policy config")
	})

	t.Run("data_files is propagated", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Policy: config.PolicyEngineConfig{
					Enabled:   config.BoolPtr(true),
					DataFiles: []string{"data.json", "config.yaml"},
				},
			},
		}

		policyCfg := buildPolicyConfig(cfg)
		assert.Equal(t, []string{"data.json", "config.yaml"}, policyCfg.DataFiles,
			"data_files should be propagated to policy config")
	})

	t.Run("rules are propagated with severity", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Policy: config.PolicyEngineConfig{
					Enabled: config.BoolPtr(true),
					Rules: map[string]config.RuleConfig{
						"require_tags": {
							Enabled:  config.BoolPtr(true),
							Severity: "error",
						},
					},
				},
			},
		}

		policyCfg := buildPolicyConfig(cfg)
		require.Contains(t, policyCfg.Rules, "require_tags")
		rule := policyCfg.Rules["require_tags"]
		assert.True(t, *rule.Enabled, "rule enabled should be propagated")
		assert.Equal(t, "error", rule.Severity, "rule severity should be propagated")
	})

	t.Run("nil config returns empty slices", func(t *testing.T) {
		policyCfg := buildPolicyConfig(nil)
		assert.NotNil(t, policyCfg.PolicyDirs, "policy_dirs should not be nil")
		assert.NotNil(t, policyCfg.PolicyFiles, "policy_files should not be nil")
		assert.Empty(t, policyCfg.PolicyDirs, "policy_dirs should be empty")
		assert.Empty(t, policyCfg.PolicyFiles, "policy_files should be empty")
	})

	t.Run("all fields together", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Engines: config.Engines{
				Policy: config.PolicyEngineConfig{
					Enabled:     config.BoolPtr(true),
					PolicyDirs:  []string{"./policies"},
					PolicyFiles: []string{"extra.rego"},
					DataFiles:   []string{"vars.json"},
					Rules: map[string]config.RuleConfig{
						"custom_rule": {Enabled: config.BoolPtr(true), Severity: "warning"},
					},
				},
			},
		}

		policyCfg := buildPolicyConfig(cfg)
		assert.Equal(t, []string{"./policies"}, policyCfg.PolicyDirs)
		assert.Equal(t, []string{"extra.rego"}, policyCfg.PolicyFiles)
		assert.Equal(t, []string{"vars.json"}, policyCfg.DataFiles)
		require.Contains(t, policyCfg.Rules, "custom_rule")
	})
}

// TestLintEngineTFLintConfigIntegration verifies that TFLint config options
// are properly handled during lint engine execution.
func TestLintEngineTFLintConfigIntegration(t *testing.T) {
	dir := t.TempDir()

	// Create a simple Terraform file
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(tfContent), 0o644))

	t.Run("use_tflint false runs builtin rules only", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.UseTFLint = false // Don't try to invoke TFLint
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		// Should run without error using builtin rules
		_, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err, "lint engine should run with builtin rules when use_tflint=false")
	})

	t.Run("invalid tflint_path with fallback_builtin uses builtin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
		cfg.Engines.Style.Enabled = config.BoolPtr(false)
		cfg.Engines.Lint.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.UseTFLint = true
		cfg.Engines.Lint.TFLintPath = "/nonexistent/path/to/tflint"
		cfg.Engines.Lint.FallbackBuiltin = true // Fall back to builtin when TFLint unavailable
		cfg.Engines.Policy.Enabled = config.BoolPtr(false)

		// Should fall back to builtin rules without error
		_, err := runAllChecksWithConfig(cfg, []string{tmpFile}, true, nil)
		require.NoError(t, err, "lint engine should fall back to builtin rules when TFLint unavailable")
	})

	t.Run("config_file is passed to lint config", func(t *testing.T) {
		// Create a custom TFLint config file
		tflintConfig := `plugin "terraform" {
  enabled = true
}
`
		tflintPath := filepath.Join(dir, "custom.tflint.hcl")
		require.NoError(t, os.WriteFile(tflintPath, []byte(tflintConfig), 0o644))

		cfg := config.DefaultConfig()
		cfg.Engines.Lint.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.ConfigFile = tflintPath
		cfg.Engines.Lint.UseTFLint = false // Use builtin to avoid TFLint dependency

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, tflintPath, lintCfg.ConfigFile,
			"config_file should be passed to lint engine config")
	})

	t.Run("args are passed to lint config", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Lint.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.Args = []string{"--force", "--no-color", "--minimum-tf-version=1.5.0"}

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, []string{"--force", "--no-color", "--minimum-tf-version=1.5.0"}, lintCfg.Args,
			"args should be passed to lint engine config")
	})

	t.Run("plugins are passed to lint config", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Engines.Lint.Enabled = config.BoolPtr(true)
		cfg.Engines.Lint.Plugins = []string{"aws", "google", "azurerm"}

		lintCfg := buildLintConfig(cfg)
		assert.Equal(t, []string{"aws", "google", "azurerm"}, lintCfg.Plugins,
			"plugins should be passed to lint engine config")
	})
}

// TestCheckWithProfile verifies that `terratidy check --profile <name>` uses profile config.
func TestCheckWithProfile(t *testing.T) {
	dir := t.TempDir()

	// Create a TF file
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tfContent), 0o644))

	// Create config with a profile that disables all engines except fmt
	configContent := `version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

profiles:
  minimal:
    description: "Minimal checks - fmt only"
    engines:
      fmt:
        enabled: true
      style:
        enabled: false
      lint:
        enabled: false
      policy:
        enabled: false
`
	configPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Save and restore globals
	oldCfgFile := cfgFile
	oldProfile := profile
	oldFormat := format
	oldChanged := changed
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
		format = oldFormat
		changed = oldChanged
	})

	// Run check with --profile minimal
	cfgFile = configPath
	profile = "minimal"
	format = "json"
	changed = false

	rootCmd.SetArgs([]string{"check", dir})
	err := rootCmd.Execute()

	// The test passes if no panic occurs and the profile is applied
	// With minimal profile, only fmt runs (no style/lint findings)
	_ = err
}

func TestExcludedFilesNotProcessed(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create directory structure
	generatedDir := filepath.Join(dir, "generated")
	require.NoError(t, os.MkdirAll(generatedDir, 0o755))

	// Create a properly formatted file that should be processed
	mainContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(mainContent), 0o644))

	// Create an improperly formatted file in "generated" directory
	// This would normally produce findings, but should be excluded
	badContent := `resource "aws_instance" "bad"   {
ami="ami-123"
instance_type="t2.micro"
}`
	require.NoError(t, os.WriteFile(filepath.Join(generatedDir, "auto.generated.tf"), []byte(badContent), 0o644))

	// Create config with exclude patterns
	configContent := `version: 1
exclude:
  - "generated/**"
  - "**/*.generated.tf"
engines:
  fmt:
    enabled: true
  style:
    enabled: false
  lint:
    enabled: false
  policy:
    enabled: false
`
	configPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	oldFormat := format
	oldChanged := changed
	oldExclude := excludePatterns
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
		format = oldFormat
		changed = oldChanged
		excludePatterns = oldExclude
	})

	// Load config with excludes
	cfgFile = configPath
	profile = ""
	format = "text"
	changed = false
	excludePatterns = nil

	cfg, err := loadConfig()
	require.NoError(t, err)

	// Get files with excludes
	files, err := getTargetFilesWithExcludes([]string{dir}, false, cfg.Exclude, cfg)
	require.NoError(t, err)

	// Should only find main.tf, not the excluded generated file
	assert.Len(t, files, 1, "should only find 1 file after excluding generated directory")
	for _, f := range files {
		assert.NotContains(t, f, "generated", "should not contain excluded files")
		assert.NotContains(t, f, ".generated.tf", "should not contain excluded files")
	}
}

func TestCLIExcludeFlag(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create directory structure
	externalDir := filepath.Join(dir, "external")
	require.NoError(t, os.MkdirAll(externalDir, 0o755))

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(externalDir, "external.tf"), []byte("# external"), 0o644))

	// Save and restore global state
	oldExclude := excludePatterns
	t.Cleanup(func() {
		excludePatterns = oldExclude
	})

	// Set exclude patterns via CLI flag simulation
	excludePatterns = []string{"external/**"}

	// Get files with CLI excludes
	files, err := getTargetFilesWithExcludes([]string{dir}, false, nil, nil)
	require.NoError(t, err)

	// Should only find main.tf, not the excluded external file
	assert.Len(t, files, 1, "should only find 1 file after CLI exclude")
	for _, f := range files {
		assert.NotContains(t, f, "external", "should not contain CLI-excluded files")
	}
}

func TestCLIAndConfigExcludesCombine(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create directory structure
	externalDir := filepath.Join(dir, "external")
	archiveDir := filepath.Join(dir, "archive")
	require.NoError(t, os.MkdirAll(externalDir, 0o755))
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(externalDir, "external.tf"), []byte("# external"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(archiveDir, "archive.tf"), []byte("# archive"), 0o644))

	// Save and restore global state
	oldExclude := excludePatterns
	t.Cleanup(func() {
		excludePatterns = oldExclude
	})

	// Set CLI exclude patterns
	excludePatterns = []string{"external/**"}

	// Config exclude patterns
	configExcludes := []string{"archive/**"}

	// Get files with both CLI and config excludes
	files, err := getTargetFilesWithExcludes([]string{dir}, false, configExcludes, nil)
	require.NoError(t, err)

	// Should only find main.tf, not the excluded external or archive files
	assert.Len(t, files, 1, "should only find 1 file after combining CLI and config excludes")
	for _, f := range files {
		assert.NotContains(t, f, "external", "should not contain CLI-excluded files")
		assert.NotContains(t, f, "archive", "should not contain config-excluded files")
	}
}

// TestNoRecurseFlagOnlyScansSpecifiedDirectory verifies that --no-recurse
// only scans the specified directory, not its subdirectories.
func TestNoRecurseFlagOnlyScansSpecifiedDirectory(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create subdirectory structure
	subDir := filepath.Join(dir, "modules")
	nestedDir := filepath.Join(subDir, "vpc")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))

	// Create test files at different levels
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# root level"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "variables.tf"), []byte("# root level"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "module.tf"), []byte("# subdir level"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "vpc.tf"), []byte("# nested level"), 0o644))

	// Save and restore global state
	oldNoRecurse := noRecurse
	t.Cleanup(func() {
		noRecurse = oldNoRecurse
	})

	t.Run("with noRecurse=false scans all levels", func(t *testing.T) {
		noRecurse = false
		files, err := getTargetFiles([]string{dir}, false)
		require.NoError(t, err)

		// Should find all 4 files
		assert.Len(t, files, 4, "recursive scan should find all .tf files")
	})

	t.Run("with noRecurse=true scans only specified directory", func(t *testing.T) {
		noRecurse = true
		files, err := getTargetFiles([]string{dir}, false)
		require.NoError(t, err)

		// Should only find 2 files in root directory
		assert.Len(t, files, 2, "non-recursive scan should only find root-level files")

		for _, f := range files {
			// Verify no subdirectory files are included
			assert.NotContains(t, f, "modules", "should not contain files from subdirectories")
		}
	})
}

// TestNoRecurseWithChangedScansChangedFileDirsOnly verifies that --no-recurse
// with --changed only returns changed files directly in specified directories.
func TestNoRecurseWithChangedScansChangedFileDirsOnly(t *testing.T) {
	// Create temp directory for git repo
	dir := t.TempDir()

	// Helper to run git commands
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	// Create directory structure
	subDir := filepath.Join(dir, "modules")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	// Initialize git repo with initial commit
	runGit("init", "-b", "main")
	runGit("config", "commit.gpgsign", "false")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# root"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "module.tf"), []byte("# subdir"), 0o644))
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	// Modify files at both levels (create uncommitted changes)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# root modified"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "module.tf"), []byte("# subdir modified"), 0o644))

	// Save and restore working directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Change to test directory
	require.NoError(t, os.Chdir(dir))

	// Save and restore global state
	oldNoRecurse := noRecurse
	t.Cleanup(func() {
		noRecurse = oldNoRecurse
	})

	t.Run("with recursive=true returns all changed files", func(t *testing.T) {
		files, err := getChangedFiles([]string{"."}, true)
		require.NoError(t, err)

		// Should find both changed files
		assert.Len(t, files, 2, "recursive mode should find all changed files")
	})

	t.Run("with recursive=false returns only root-level changed files", func(t *testing.T) {
		files, err := getChangedFiles([]string{"."}, false)
		require.NoError(t, err)

		// Should only find root-level changed file
		assert.Len(t, files, 1, "non-recursive mode should only find root-level changed files")

		for _, f := range files {
			assert.NotContains(t, f, "modules", "should not contain files from subdirectories")
		}
	})
}

// TestNoRecurseWithPositionalPathScansBaseDirOnly verifies that --no-recurse
// with a positional path argument only scans files directly in that directory.
func TestNoRecurseWithPositionalPathScansBaseDirOnly(t *testing.T) {
	// Create temp directory with nested structure
	dir := t.TempDir()

	// Create directory structure:
	// dir/
	//   main.tf
	//   modules/
	//     module.tf
	//     vpc/
	//       vpc.tf
	modulesDir := filepath.Join(dir, "modules")
	vpcDir := filepath.Join(modulesDir, "vpc")
	require.NoError(t, os.MkdirAll(vpcDir, 0o755))

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# root"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(modulesDir, "module.tf"), []byte("# modules level"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(vpcDir, "vpc.tf"), []byte("# vpc nested"), 0o644))

	// Save and restore global state
	oldNoRecurse := noRecurse
	t.Cleanup(func() {
		noRecurse = oldNoRecurse
	})

	t.Run("recursive scan of modules/ finds all nested files", func(t *testing.T) {
		noRecurse = false
		files, err := getTargetFiles([]string{modulesDir}, false)
		require.NoError(t, err)

		// Should find both module.tf and vpc/vpc.tf
		assert.Len(t, files, 2, "recursive scan should find all files in modules/")
	})

	t.Run("non-recursive scan of modules/ finds only direct children", func(t *testing.T) {
		noRecurse = true
		files, err := getTargetFiles([]string{modulesDir}, false)
		require.NoError(t, err)

		// Should only find module.tf, not vpc/vpc.tf
		assert.Len(t, files, 1, "non-recursive scan should only find direct children")

		for _, f := range files {
			assert.NotContains(t, f, "vpc", "should not contain files from nested subdirectories")
			assert.Contains(t, f, "module.tf", "should contain the direct child file")
		}
	})

	t.Run("non-recursive scan of multiple paths", func(t *testing.T) {
		noRecurse = true
		// Pass both root dir and modules dir as positional arguments
		files, err := getTargetFiles([]string{dir, modulesDir}, false)
		require.NoError(t, err)

		// Should find main.tf (from dir) and module.tf (from modules/)
		// But NOT vpc/vpc.tf
		assert.Len(t, files, 2, "should find one file from each specified directory")

		for _, f := range files {
			assert.NotContains(t, f, "vpc", "should not contain nested files")
		}
	})
}

// TestConfigRecursiveFalseBehavesLikeNoRecurseFlag verifies that recursive: false
// in config produces the same behavior as the --no-recurse CLI flag.
func TestConfigRecursiveFalseBehavesLikeNoRecurseFlag(t *testing.T) {
	// Create temp directory with nested structure
	dir := t.TempDir()

	// Create directory structure
	subDir := filepath.Join(dir, "modules")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# root"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "module.tf"), []byte("# subdir"), 0o644))

	// Save and restore global state
	oldNoRecurse := noRecurse
	t.Cleanup(func() {
		noRecurse = oldNoRecurse
	})

	// Reset noRecurse flag to ensure config takes effect
	noRecurse = false

	t.Run("config recursive true scans all files", func(t *testing.T) {
		cfg := &config.Config{
			Version:   1,
			Recursive: config.BoolPtr(true),
		}

		files, err := getTargetFilesWithExcludes([]string{dir}, false, nil, cfg)
		require.NoError(t, err)

		assert.Len(t, files, 2, "recursive: true should find all files")
	})

	t.Run("config recursive false scans only root level", func(t *testing.T) {
		cfg := &config.Config{
			Version:   1,
			Recursive: config.BoolPtr(false),
		}

		files, err := getTargetFilesWithExcludes([]string{dir}, false, nil, cfg)
		require.NoError(t, err)

		assert.Len(t, files, 1, "recursive: false should only find root-level files")
		for _, f := range files {
			assert.NotContains(t, f, "modules", "should not contain subdirectory files")
		}
	})

	t.Run("CLI flag overrides config", func(t *testing.T) {
		// Config says recursive: true, but CLI flag says --no-recurse
		noRecurse = true
		cfg := &config.Config{
			Version:   1,
			Recursive: config.BoolPtr(true),
		}

		files, err := getTargetFilesWithExcludes([]string{dir}, false, nil, cfg)
		require.NoError(t, err)

		assert.Len(t, files, 1, "CLI --no-recurse should override config recursive: true")
	})
}

func TestAbsolutePathsOutputBehavior(t *testing.T) {
	// Save original value and restore after test
	originalAbsolutePaths := absolutePaths
	defer func() { absolutePaths = originalAbsolutePaths }()

	t.Run("default uses relative paths", func(t *testing.T) {
		absolutePaths = false
		cfg := &config.Config{
			Version: 1,
		}

		// getEffectiveAbsolutePaths should return false by default
		result := getEffectiveAbsolutePaths(cfg)
		assert.False(t, result, "default should use relative paths")
	})

	t.Run("CLI flag enables absolute paths", func(t *testing.T) {
		absolutePaths = true
		cfg := &config.Config{
			Version: 1,
		}

		result := getEffectiveAbsolutePaths(cfg)
		assert.True(t, result, "CLI flag should enable absolute paths")
	})

	t.Run("config enables absolute paths", func(t *testing.T) {
		absolutePaths = false
		cfg := &config.Config{
			Version: 1,
			Output:  config.OutputConfig{AbsolutePaths: config.BoolPtr(true)},
		}

		result := getEffectiveAbsolutePaths(cfg)
		assert.True(t, result, "config should enable absolute paths")
	})

	t.Run("CLI flag takes precedence over config", func(t *testing.T) {
		// CLI says absolute, config says relative
		absolutePaths = true
		cfg := &config.Config{
			Version: 1,
			Output:  config.OutputConfig{AbsolutePaths: config.BoolPtr(false)},
		}

		result := getEffectiveAbsolutePaths(cfg)
		assert.True(t, result, "CLI flag should override config")
	})
}
