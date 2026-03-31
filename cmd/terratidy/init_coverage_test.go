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
		{"MixedCase", "ixedase"}, // uppercase letters stripped
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
