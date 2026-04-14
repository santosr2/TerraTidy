package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
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

// TestInitRuleCmd_InvalidType verifies that init-rule rejects unsupported rule types
// BEFORE any side effects (directory creation, progress output).
func TestInitRuleCmd_InvalidType(t *testing.T) {
	// Use non-existent subdirectory to detect if MkdirAll is called
	dir := filepath.Join(t.TempDir(), "should-not-exist")
	setupInitRuleTest(t)

	// Capture stdout to verify no progress message is printed
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"init-rule", "--name", "test-rule", "--type", "bash", "--output", dir})
	err := rootCmd.Execute()

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported rule type")
	assert.Contains(t, err.Error(), "bash")

	// Verify exit code is ConfigError (user-correctable input)
	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "error should be ExitError")
	assert.Equal(t, sdk.ExitConfig, exitErr.Code, "invalid type should return ExitConfig (code 2)")

	// Verify no side effects: directory should not exist
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "output directory should not be created for invalid type")

	// Verify no "Creating" progress message was printed
	assert.NotContains(t, output, "Creating", "no progress message should be printed for invalid type")
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

func TestReadLine_Success(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("hello world\n"))
	line, err := readLine(reader)
	require.NoError(t, err)
	assert.Equal(t, "hello world", line)
}

func TestReadLine_TrimsWhitespace(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  trimmed  \n"))
	line, err := readLine(reader)
	require.NoError(t, err)
	assert.Equal(t, "trimmed", line)
}

func TestReadLine_EmptyInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	line, err := readLine(reader)
	require.NoError(t, err)
	assert.Equal(t, "", line)
}

func TestReadLine_PartialAtEOF(t *testing.T) {
	// Input without trailing newline (e.g., "echo -n y | terratidy init -i")
	// Should succeed and return the partial data
	reader := bufio.NewReader(strings.NewReader("partial"))
	line, err := readLine(reader)
	require.NoError(t, err)
	assert.Equal(t, "partial", line)
}

func TestReadLine_Error(t *testing.T) {
	// Completely empty input returns EOF error (no data at all)
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := readLine(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading input")
	assert.ErrorIs(t, err, io.EOF)
}

func TestReadYesNo_Yes(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"Yes\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := readYesNo(reader, false)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadYesNo_No(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"NO\n", false},
		{"anything\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := readYesNo(reader, true) // default true but input says no
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadYesNo_DefaultYes(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	got, err := readYesNo(reader, true)
	require.NoError(t, err)
	assert.True(t, got)
}

func TestReadYesNo_DefaultNo(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	got, err := readYesNo(reader, false)
	require.NoError(t, err)
	assert.False(t, got)
}

func TestReadYesNo_Error(t *testing.T) {
	// Empty reader returns EOF on ReadString
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := readYesNo(reader, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, io.EOF)
}
