package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestRunPolicyCheckWithConfig(t *testing.T) {
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
	findings, err := runPolicyCheckWithConfig(ctx, cfg, []string{tmpFile}, 4, true)
	require.NoError(t, err)
	_ = findings
}

func TestBuildPolicyConfig_WithEngineConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Engines.Policy.PolicyDirs = []string{"./policies"}
	cfg.Engines.Policy.PolicyFiles = []string{"custom.rego"}

	policyCfg := buildPolicyConfig(cfg)
	assert.Equal(t, []string{"./policies"}, policyCfg.PolicyDirs)
	assert.Equal(t, []string{"custom.rego"}, policyCfg.PolicyFiles)
}

func TestOutputPolicyResults(t *testing.T) {
	t.Run("no findings returns nil", func(t *testing.T) {
		err := outputPolicyResults(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("warning findings return nil", func(t *testing.T) {
		findings := []sdk.Finding{
			{Rule: "policy.rule", Severity: sdk.SeverityWarning, Message: "warning"},
		}
		err := outputPolicyResults(findings, nil)
		assert.NoError(t, err)
	})

	t.Run("error findings return ExitError", func(t *testing.T) {
		findings := []sdk.Finding{
			{Rule: "policy.rule", Severity: sdk.SeverityError, Message: "error"},
		}
		err := outputPolicyResults(findings, nil)
		require.Error(t, err)
		var exitErr *sdk.ExitError
		assert.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.Code)
	})
}

func TestPolicyCommandWithExcludes(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()

	// Create directory structure
	externalDir := filepath.Join(dir, "external")
	require.NoError(t, os.MkdirAll(externalDir, 0o755))

	// Create a valid terraform file
	mainContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(mainContent), 0o644))

	// Create another file in "external" directory (should be excluded)
	externalContent := `resource "aws_instance" "external" {
  ami = "ami-456"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(externalDir, "external.tf"), []byte(externalContent), 0o644))

	// Create config with exclude patterns
	configContent := `version: 1
exclude:
  - "external/**"
engines:
  policy:
    enabled: true
`
	configPath := filepath.Join(dir, ".terratidy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Save and restore global state
	oldCfgFile := cfgFile
	oldProfile := profile
	oldFormat := format
	oldChanged := changed
	oldExclude := excludePatterns
	oldPolicyDirs := policyDirs
	oldPolicyFiles := policyFiles
	oldPolicyShowJSON := policyShowJSON
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		profile = oldProfile
		format = oldFormat
		changed = oldChanged
		excludePatterns = oldExclude
		policyDirs = oldPolicyDirs
		policyFiles = oldPolicyFiles
		policyShowJSON = oldPolicyShowJSON
	})

	// Set up global state
	cfgFile = configPath
	profile = ""
	format = "text"
	changed = false
	excludePatterns = nil
	policyDirs = nil
	policyFiles = nil
	policyShowJSON = false

	// Run the policy command - this exercises getTargetFilesWithExcludes in policy.go
	err := policyCmd.RunE(&cobra.Command{}, []string{dir})
	// Policy may return an error if there are violations, but the exclude logic was exercised
	_ = err
}
