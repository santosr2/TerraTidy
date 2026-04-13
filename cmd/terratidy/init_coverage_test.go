package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateDefaultConfig(t *testing.T) {
	cfg := generateDefaultConfig()
	assert.NotEmpty(t, cfg)
	assert.Contains(t, cfg, "version: 1")
	assert.Contains(t, cfg, "engines:")
	assert.Contains(t, cfg, "fmt:")
	assert.Contains(t, cfg, "style:")
	assert.Contains(t, cfg, "lint:")

	// Verify it's valid YAML
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(cfg), &parsed))
}

func TestGenerateMonorepoConfig(t *testing.T) {
	cfg := generateMonorepoConfig()
	assert.NotEmpty(t, cfg)
	assert.Contains(t, cfg, "version: 1")
	assert.Contains(t, cfg, "engines:")

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(cfg), &parsed))
}

func TestGenerateCustomConfig(t *testing.T) {
	cfg := generateCustomConfig(customConfigOptions{
		fmtEnabled:    true,
		styleEnabled:  true,
		lintEnabled:   false,
		policyEnabled: false,
		severity:      "warning",
		failFast:      true,
	})
	assert.NotEmpty(t, cfg)
	assert.Contains(t, cfg, "version: 1")
	assert.Contains(t, cfg, "fail_fast: true")

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(cfg), &parsed))
}

func TestToGoPackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-rule", "my_rule"},
		{"simple", "simple"},
		{"with_underscore", "with_underscore"},
		{"MixedCase", "mixedcase"}, // uppercase letters lowercased
		{"special!chars@here", "specialcharshere"},
		{"123numeric", "123numeric"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toGoPackageName(tt.input))
		})
	}
}

func TestGoRuleTemplate(t *testing.T) {
	tmpl := goRuleTemplate("my-rule")
	assert.Contains(t, tmpl, "my-rule")
	assert.Contains(t, tmpl, "func (r *Rule) Name()")
}

func TestGoTestTemplate(t *testing.T) {
	tmpl := goTestTemplate("my-rule")
	assert.Contains(t, tmpl, "my-rule")
	assert.Contains(t, tmpl, "func Test")
}

func TestCreateGoRule(t *testing.T) {
	dir := t.TempDir()
	oldOutput := initRuleOutput
	initRuleOutput = dir
	defer func() { initRuleOutput = oldOutput }()

	err := createGoRule("test-rule")
	require.NoError(t, err)

	// Creates initRuleOutput/rules/test-rule/
	ruleDir := filepath.Join(dir, "rules", "test-rule")
	_, err = os.Stat(ruleDir)
	require.NoError(t, err)

	ruleFile := filepath.Join(ruleDir, "rule.go")
	content, err := os.ReadFile(ruleFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-rule")
}

func TestCreateRegoRule(t *testing.T) {
	dir := t.TempDir()
	oldOutput := initRuleOutput
	initRuleOutput = dir
	defer func() { initRuleOutput = oldOutput }()

	err := createRegoRule("test-policy")
	require.NoError(t, err)

	// Creates initRuleOutput/policies/test-policy.rego
	regoFile := filepath.Join(dir, "policies", "test-policy.rego")
	content, err := os.ReadFile(regoFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-policy")
}

func TestCreateYAMLRule(t *testing.T) {
	dir := t.TempDir()
	oldOutput := initRuleOutput
	initRuleOutput = dir
	defer func() { initRuleOutput = oldOutput }()

	err := createYAMLRule("test-yaml-rule")
	require.NoError(t, err)

	// Creates initRuleOutput/rules/test-yaml-rule.yaml
	yamlFile := filepath.Join(dir, "rules", "test-yaml-rule.yaml")
	content, err := os.ReadFile(yamlFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-yaml-rule")

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(content, &parsed))
}

// setupInitRuleTest resets init-rule flags and registers cleanup.
func setupInitRuleTest(t *testing.T) {
	t.Helper()

	// Reset Cobra flag "changed" state before test.
	// Pre-reset guards against prior test leaving dirty state (tests may run out of
	// declaration order when parallel or across files). Cleanup handles tests that follow.
	if f := initRuleCmd.Flags().Lookup("name"); f != nil {
		f.Changed = false
	}
	if f := initRuleCmd.Flags().Lookup("type"); f != nil {
		f.Changed = false
	}
	if f := initRuleCmd.Flags().Lookup("output"); f != nil {
		f.Changed = false
	}

	t.Cleanup(func() {
		initRuleName = ""
		initRuleType = "rego"
		initRuleOutput = "."
		rootCmd.SetArgs(nil)

		if f := initRuleCmd.Flags().Lookup("name"); f != nil {
			f.Changed = false
		}
		if f := initRuleCmd.Flags().Lookup("type"); f != nil {
			f.Changed = false
		}
		if f := initRuleCmd.Flags().Lookup("output"); f != nil {
			f.Changed = false
		}
	})
}

// TestInitRuleCmd_InvalidType verifies that init-rule rejects unsupported rule types.
func TestInitRuleCmd_InvalidType(t *testing.T) {
	dir := t.TempDir()
	setupInitRuleTest(t)

	rootCmd.SetArgs([]string{"init-rule", "--name", "test-rule", "--type", "bash", "--output", dir})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported rule type")
	assert.Contains(t, err.Error(), "bash")
}

// TestInitRuleCmd_ExistingFile verifies that init-rule overwrites existing files.
// Note: Current behavior is to silently overwrite; this test documents that behavior.
func TestInitRuleCmd_ExistingFile(t *testing.T) {
	dir := t.TempDir()

	// Create existing file
	rulesDir := filepath.Join(dir, "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o750))
	existingFile := filepath.Join(rulesDir, "existing-rule.yaml")
	existingContent := "# existing content\nname: old\n"
	require.NoError(t, os.WriteFile(existingFile, []byte(existingContent), 0o600))

	setupInitRuleTest(t)

	rootCmd.SetArgs([]string{"init-rule", "--name", "existing-rule", "--type", "yaml", "--output", dir})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Verify file was overwritten with new content
	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "existing-rule")
	assert.NotEqual(t, existingContent, string(content)) // Content was replaced
	assert.NotContains(t, string(content), "name: old")  // Specific old value is gone
}

// TestInitRuleCmd_InvalidName verifies that init-rule requires a name via Cobra's
// MarkFlagRequired. Testing empty name via CLI args.
func TestInitRuleCmd_InvalidName(t *testing.T) {
	dir := t.TempDir()
	setupInitRuleTest(t)

	// Missing --name flag should trigger required flag error
	rootCmd.SetArgs([]string{"init-rule", "--type", "yaml", "--output", dir})
	err := rootCmd.Execute()

	require.Error(t, err)
	// Cobra's MarkFlagRequired produces: required flag(s) "name" not set
	assert.Contains(t, err.Error(), `"name" not set`)
}
