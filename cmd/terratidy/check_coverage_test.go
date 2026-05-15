package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// TestOutputStyleResults_CheckMode verifies that outputStyleResults returns an
// ExitError with ExitFindings code in check mode when findings are present.
func TestOutputStyleResults_CheckMode(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	findings := []sdk.Finding{
		{Rule: "style.blank-lines", Message: "test", Severity: sdk.SeverityWarning, File: "main.tf"},
	}

	err := outputStyleResults(findings, findings, true, nil)
	require.Error(t, err, "check mode with findings should return error")

	// Should be an ExitError with findings code
	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "should be an ExitError")
	assert.Equal(t, sdk.ExitFindings, exitErr.Code, "should have findings exit code")
}

func TestOutputStyleResults_NoCheckMode(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	findings := []sdk.Finding{
		{Rule: "style.blank-lines", Message: "test", Severity: sdk.SeverityWarning, File: "main.tf"},
	}

	err := outputStyleResults(findings, findings, false, nil)
	assert.NoError(t, err, "non-check mode should not return error for warnings")
}

func TestOutputStyleResults_NoFindings(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	err := outputStyleResults(nil, nil, true, nil)
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

	err := outputLintResults(findings, nil)
	assert.NoError(t, err)
}

func TestOutputLintResults_NoFindings(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	err := outputLintResults(nil, nil)
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
			"style.blank-line-between-blocks": {
				Enabled:  config.BoolPtr(true),
				Severity: "warning",
			},
		}

		cfg := buildStyleConfig(appCfg, false)
		require.Contains(t, cfg.Rules, "style.blank-line-between-blocks")
		assert.True(t, *cfg.Rules["style.blank-line-between-blocks"].Enabled)
		assert.Equal(t, "warning", cfg.Rules["style.blank-line-between-blocks"].Severity)
	})

	t.Run("with engine rules", func(t *testing.T) {
		appCfg := config.DefaultConfig()
		appCfg.Engines.Style.Rules = map[string]config.RuleConfig{
			"my-rule": {Enabled: config.BoolPtr(true), Severity: "error", Config: map[string]any{"key": "val"}},
		}

		cfg := buildStyleConfig(appCfg, false)
		require.Contains(t, cfg.Rules, "my-rule")
		assert.True(t, *cfg.Rules["my-rule"].Enabled)
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
			Enabled:  config.BoolPtr(true),
			Severity: "error",
			Config:   map[string]any{"max_lines": 100},
		},
	}

	cfg := buildStyleConfig(appCfg, false)
	rc := cfg.Rules["rule-with-options"]
	assert.True(t, *rc.Enabled)
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

func TestHasMatchingTag(t *testing.T) {
	tests := []struct {
		name       string
		ruleTags   []string
		filterTags []string
		want       bool
	}{
		{"exact match", []string{"security"}, []string{"security"}, true},
		{"one of many matches", []string{"security", "compliance"}, []string{"compliance"}, true},
		{"no match", []string{"security"}, []string{"lint"}, false},
		{"empty rule tags", []string{}, []string{"security"}, false},
		{"empty filter tags", []string{"security"}, []string{}, false},
		{"both empty", []string{}, []string{}, false},
		{"multiple filter tags one matches", []string{"security"}, []string{"lint", "security"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasMatchingTag(tt.ruleTags, tt.filterTags)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadPluginRules_NilConfig(t *testing.T) {
	rules, err := loadPluginRules(nil)
	require.NoError(t, err)
	assert.Nil(t, rules)
}

func TestLoadPluginRules_PluginsDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = false

	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	assert.Nil(t, rules)
}

func TestLoadPluginRules_DisabledRule(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	yamlRule := `name: disabled-rule
description: Should be filtered out
severity: warning
enabled: true
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "disabled-rule.yaml"), []byte(yamlRule), 0o644))

	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"disabled-rule": {Enabled: config.BoolPtr(false)},
	}

	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	assert.Empty(t, rules, "disabled plugin rule should be filtered out")
}

func TestLoadPluginRules_TagFilter_MatchingTag(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	yamlRule := `name: tagged-rule
description: Has matching tag
severity: warning
enabled: true
tags:
  - security
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "tagged-rule.yaml"), []byte(yamlRule), 0o644))

	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}
	cfg.Plugins.Tags = []string{"security"}

	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	assert.Len(t, rules, 1, "rule with matching tag should be included")
}

func TestLoadPluginRules_TagFilter_NoMatch(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	yamlRule := `name: untagged-rule
description: Does not match tag filter
severity: warning
enabled: true
tags:
  - compliance
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "untagged-rule.yaml"), []byte(yamlRule), 0o644))

	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}
	cfg.Plugins.Tags = []string{"security"}

	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	assert.Empty(t, rules, "rule with non-matching tag should be excluded")
}

func TestLoadPluginRules_TagFilter_RuleWithoutTagsSkipped(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	// YAML rule with no tags field — implements TaggedRule but returns nil
	yamlRule := `name: no-tags-rule
description: Rule with no tags
severity: warning
enabled: true
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "no-tags-rule.yaml"), []byte(yamlRule), 0o644))

	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = true
	cfg.Plugins.Directories = []string{pluginDir}
	cfg.Plugins.Tags = []string{"security"}

	rules, err := loadPluginRules(cfg)
	require.NoError(t, err)
	assert.Empty(t, rules, "rule with no tags should be excluded when tag filter is active")
}

func TestBuildStyleConfig_WithPluginRules(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"my-plugin-rule": {
			Enabled:  config.BoolPtr(true),
			Severity: "error",
			Config:   map[string]any{"threshold": 5},
		},
	}

	styleCfg := buildStyleConfig(cfg, false)
	require.Contains(t, styleCfg.Rules, "my-plugin-rule")
	rc := styleCfg.Rules["my-plugin-rule"]
	assert.True(t, *rc.Enabled)
	assert.Equal(t, "error", rc.Severity)
	assert.Equal(t, 5, rc.Options["threshold"])
}

func TestBuildLintConfig_WithEngineAndPluginRules(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Engines.Lint.Rules = map[string]config.RuleConfig{
		"engine-rule": {Enabled: config.BoolPtr(true), Severity: "warning"},
	}
	cfg.Plugins.Rules = map[string]config.RuleConfig{
		"plugin-rule": {Enabled: config.BoolPtr(true), Severity: "error", Config: map[string]any{"key": "val"}},
	}

	lintCfg := buildLintConfig(cfg)
	require.Contains(t, lintCfg.Rules, "engine-rule")
	assert.Equal(t, "warning", lintCfg.Rules["engine-rule"].Severity)

	require.Contains(t, lintCfg.Rules, "plugin-rule")
	assert.Equal(t, "error", lintCfg.Rules["plugin-rule"].Severity)
	assert.Equal(t, "val", lintCfg.Rules["plugin-rule"].Options["key"])
}

// TestChangedFlagConsistency verifies that --changed flag behaves consistently
// across fmt, style, and lint commands: all three only process git-modified files.
// NOTE: This test uses os.Chdir; do not call t.Parallel().
func TestChangedFlagConsistency(t *testing.T) {
	dir := t.TempDir()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	// Create directory structure with two files
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "modules"), 0o755))

	// Initial committed content (properly formatted)
	committedContent := `resource "aws_instance" "committed" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	// Changed content (properly formatted but different)
	changedContent := `resource "aws_instance" "modified" {
  ami           = "ami-456"
  instance_type = "t2.small"
}
`

	// Write initial files and commit them
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unchanged.tf"), []byte(committedContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "changed.tf"), []byte(committedContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modules", "nested.tf"), []byte(committedContent), 0o644))

	runGit("init", "-b", "main")
	runGit("config", "commit.gpgsign", "false")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "test")
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	// Modify one file to create uncommitted change
	require.NoError(t, os.WriteFile(filepath.Join(dir, "changed.tf"), []byte(changedContent), 0o644))

	// Save and restore working directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	require.NoError(t, os.Chdir(dir))

	// Save and restore global state that affects getTargetFiles behavior
	oldNoRecurse := noRecurse
	oldExclude := excludePatterns
	t.Cleanup(func() {
		noRecurse = oldNoRecurse
		excludePatterns = oldExclude
	})
	noRecurse = false
	excludePatterns = nil

	t.Run("getChangedFiles returns only modified file", func(t *testing.T) {
		files, err := getChangedFiles([]string{"."}, true)
		require.NoError(t, err)

		require.Len(t, files, 1, "should only find the one changed file")
		assert.Contains(t, files[0], "changed.tf", "should be the changed.tf file")
	})

	t.Run("getTargetFiles with changedOnly=true respects --changed", func(t *testing.T) {
		files, err := getTargetFiles([]string{"."}, true)
		require.NoError(t, err)

		require.Len(t, files, 1, "changedOnly=true should return only changed files")
		assert.Contains(t, files[0], "changed.tf")
	})

	t.Run("getTargetFiles with changedOnly=false returns all files", func(t *testing.T) {
		files, err := getTargetFiles([]string{"."}, false)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(files), 3, "changedOnly=false should return all .tf files")
	})

	t.Run("getTargetFilesWithExcludes respects changedOnly", func(t *testing.T) {
		cfg := config.DefaultConfig()

		filesWithChanged, err := getTargetFilesWithExcludes([]string{"."}, true, nil, cfg)
		require.NoError(t, err)
		require.Len(t, filesWithChanged, 1, "changedOnly=true should return only changed file")

		filesWithoutChanged, err := getTargetFilesWithExcludes([]string{"."}, false, nil, cfg)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(filesWithoutChanged), 3, "changedOnly=false should return all files")
	})
}

// TestChangedFlagErrorsOutsideGitRepo verifies that --changed returns an error
// when run outside a git repository for all commands using getTargetFiles.
// NOTE: This test uses os.Chdir; do not call t.Parallel().
func TestChangedFlagErrorsOutsideGitRepo(t *testing.T) {
	nonGitDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(nonGitDir, "main.tf"), []byte("# test"), 0o644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	require.NoError(t, os.Chdir(nonGitDir))

	_, err = getTargetFiles([]string{"."}, true)
	require.Error(t, err, "changedOnly=true outside git repo should error")
	assert.Contains(t, err.Error(), "not a git repository", "error should mention git repo requirement")
}

// TestChangedFlagWithNestedDirectories verifies --changed works with nested paths.
// NOTE: This test uses os.Chdir; do not call t.Parallel().
func TestChangedFlagWithNestedDirectories(t *testing.T) {
	dir := t.TempDir()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	// Create nested directory structure
	modulesDir := filepath.Join(dir, "modules")
	envDir := filepath.Join(dir, "environments")
	require.NoError(t, os.MkdirAll(modulesDir, 0o755))
	require.NoError(t, os.MkdirAll(envDir, 0o755))

	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	modifiedContent := `resource "aws_instance" "modified" {
  ami           = "ami-456"
  instance_type = "t2.small"
}
`

	// Create files in both directories
	require.NoError(t, os.WriteFile(filepath.Join(modulesDir, "mod.tf"), []byte(content), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(envDir, "env.tf"), []byte(content), 0o644))

	runGit("init", "-b", "main")
	runGit("config", "commit.gpgsign", "false")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "test")
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	// Modify only the modules file
	require.NoError(t, os.WriteFile(filepath.Join(modulesDir, "mod.tf"), []byte(modifiedContent), 0o644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	require.NoError(t, os.Chdir(dir))

	t.Run("changed flag from root finds nested changed file", func(t *testing.T) {
		files, err := getChangedFiles([]string{"."}, true)
		require.NoError(t, err)

		require.Len(t, files, 1)
		assert.Contains(t, files[0], "mod.tf")
	})

	t.Run("changed flag with specific path filters correctly", func(t *testing.T) {
		// Ask for changes in modules - should find the changed file
		filesInModules, err := getChangedFiles([]string{"modules"}, true)
		require.NoError(t, err)
		require.Len(t, filesInModules, 1, "modules dir has one changed file")

		// Ask for changes in environments - should find nothing (no changes there)
		filesInEnv, err := getChangedFiles([]string{"environments"}, true)
		require.NoError(t, err)
		assert.Empty(t, filesInEnv, "environments dir has no changes")
	})
}
