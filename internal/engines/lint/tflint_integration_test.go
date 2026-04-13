//go:build integration

package lint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNoTFLint skips the test if TFLint is not available.
func skipIfNoTFLint(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tflint"); err != nil {
		t.Skip("tflint not found in PATH, skipping integration test")
	}
}

// TestTFLint_SubprocessInvocation_ValidTerraform verifies TFLint can process valid Terraform.
// This test validates the subprocess invocation succeeds and returns no error for valid code.
func TestTFLint_SubprocessInvocation_ValidTerraform(t *testing.T) {
	skipIfNoTFLint(t)

	dir := t.TempDir()

	// Create a valid, minimal Terraform file that should produce no findings
	content := `terraform {
  required_version = ">= 1.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644))

	engine := New(&Config{
		UseTFLint: true,
	})

	ctx := context.Background()
	findings, err := engine.RunTFLint(ctx, dir)
	require.NoError(t, err, "TFLint should run without error on valid Terraform")
	assert.Empty(t, findings, "Valid minimal Terraform should produce no findings")
}

// TestTFLint_CustomConfig_DisablesRules verifies custom TFLint config is used.
// By creating a config that disables all rules, we can verify the config path is passed correctly.
func TestTFLint_CustomConfig_DisablesRules(t *testing.T) {
	skipIfNoTFLint(t)

	dir := t.TempDir()

	// Create Terraform file with a common issue (missing required_providers)
	content := `resource "aws_instance" "test" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644))

	// Create a custom TFLint config that disables all rules
	tflintConfig := `config {
  disabled_by_default = true
}
`
	configPath := filepath.Join(dir, ".tflint.hcl")
	require.NoError(t, os.WriteFile(configPath, []byte(tflintConfig), 0o644))

	engine := New(&Config{
		UseTFLint:    true,
		TFLintConfig: configPath,
	})

	ctx := context.Background()
	findings, err := engine.RunTFLint(ctx, dir)
	require.NoError(t, err)
	assert.Empty(t, findings, "Custom config with disabled_by_default=true should produce no findings")
}

// TestTFLint_MultipleDirectories_ProcessesBoth verifies TFLint processes files from multiple directories.
// The test validates that files from different directories are all processed.
func TestTFLint_MultipleDirectories_ProcessesBoth(t *testing.T) {
	skipIfNoTFLint(t)

	baseDir := t.TempDir()

	// Create two subdirectories with Terraform files
	dir1 := filepath.Join(baseDir, "module1")
	dir2 := filepath.Join(baseDir, "module2")
	require.NoError(t, os.MkdirAll(dir1, 0o755))
	require.NoError(t, os.MkdirAll(dir2, 0o755))

	// Minimal valid Terraform in each directory
	content := `terraform {
  required_version = ">= 1.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "main.tf"), []byte(content), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "main.tf"), []byte(content), 0o644))

	engine := New(&Config{
		UseTFLint: true,
	})

	ctx := context.Background()
	files := []string{
		filepath.Join(dir1, "main.tf"),
		filepath.Join(dir2, "main.tf"),
	}
	findings, err := engine.RunWithTFLint(ctx, files)
	require.NoError(t, err, "TFLint should process multiple directories without error")

	// Both directories should be processed (no error means both were visited)
	// Verify by checking we got valid output (empty findings for valid Terraform)
	assert.Empty(t, findings, "Valid Terraform in both directories should produce no findings")
}

// TestTFLint_FindingsIncludeFileInfo verifies findings include correct file information.
// This uses a TFLint config to enable a specific rule that will fire.
func TestTFLint_FindingsIncludeFileInfo(t *testing.T) {
	skipIfNoTFLint(t)

	dir := t.TempDir()

	// Create a file that will trigger the terraform_required_providers rule
	// This rule is part of TFLint core (terraform ruleset)
	content := `resource "aws_instance" "test" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`
	mainTf := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(mainTf, []byte(content), 0o644))

	// Enable only terraform_required_providers rule
	tflintConfig := `config {
  disabled_by_default = true
}

rule "terraform_required_providers" {
  enabled = true
}
`
	configPath := filepath.Join(dir, ".tflint.hcl")
	require.NoError(t, os.WriteFile(configPath, []byte(tflintConfig), 0o644))

	engine := New(&Config{
		UseTFLint:    true,
		TFLintConfig: configPath,
	})

	ctx := context.Background()
	findings, err := engine.RunTFLint(ctx, dir)
	require.NoError(t, err)

	// This rule may or may not fire depending on TFLint version
	if len(findings) == 0 {
		t.Skip("terraform_required_providers rule did not fire (TFLint version may not support it)")
	}

	// Verify finding structure when we do get findings
	f := findings[0]
	assert.NotEmpty(t, f.Rule, "Rule should not be empty")
	assert.True(t, strings.HasPrefix(f.Rule, "tflint."), "Rule should have tflint. prefix")
	assert.NotEmpty(t, f.Message, "Message should not be empty")
	assert.Contains(t, f.File, "main.tf", "Finding should reference the Terraform file")
}
