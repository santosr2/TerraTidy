package main

import (
	"context"
	"os"
	"path/filepath"
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

func TestPrintCheckHints(t *testing.T) {
	// Verify it doesn't panic
	printCheckHints()
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

		err := outputCheckResults(nil, false)
		assert.NoError(t, err)
	})

	t.Run("with error findings text format", func(t *testing.T) {
		old := format
		format = "text"
		defer func() { format = old }()

		findings := []sdk.Finding{
			{Rule: "test.rule", Message: "test", Severity: sdk.SeverityError, File: "test.tf"},
		}
		err := outputCheckResults(findings, false)
		assert.Error(t, err, "should return exit error for errors")
	})

	t.Run("with error findings json format", func(t *testing.T) {
		old := format
		format = "json"
		defer func() { format = old }()

		findings := []sdk.Finding{
			{Rule: "test.rule", Message: "test", Severity: sdk.SeverityError, File: "test.tf"},
		}
		err := outputCheckResults(findings, true)
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
			"lint-disabled-rule": {Enabled: false},
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
			"lint-severity-rule": {Enabled: true, Severity: "error"},
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
		"disabled-rule": {Enabled: false},
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
		"enabled-rule": {Enabled: true},
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
		"severity-test-rule": {Enabled: true, Severity: "error"},
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
