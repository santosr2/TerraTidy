package lint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_EmptyFiles(t *testing.T) {
	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_NilFiles(t *testing.T) {
	engine := New(nil)
	findings, err := engine.Run(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_NonTFFiles(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte("package main"), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{goFile})
	require.NoError(t, err)
	// Lint engine attempts to parse all files; non-TF files produce parse errors
	for _, f := range findings {
		assert.Equal(t, "lint.parse-error", f.Rule, "non-TF files should only produce parse errors")
	}
}

func TestNew_NilConfig(t *testing.T) {
	engine := New(nil)
	require.NotNil(t, engine)
	assert.Equal(t, "lint", engine.Name())
}

func TestNew_WithConfig(t *testing.T) {
	cfg := &Config{
		ConfigFile: "custom.hcl",
		Plugins:    []string{"aws"},
	}
	engine := New(cfg)
	require.NotNil(t, engine)
}

func TestRunTFLint_SkipIfUnavailable(t *testing.T) {
	if _, err := exec.LookPath("tflint"); err != nil {
		t.Skip("tflint not available")
	}

	dir := t.TempDir()
	content := `terraform {
  required_version = ">= 1.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644))

	engine := New(&Config{ConfigFile: ".tflint.hcl"})
	findings, err := engine.RunTFLint(context.Background(), dir)
	// May error if no .tflint.hcl config, that's acceptable
	_ = err
	_ = findings
}

func TestRun_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	engine := New(nil)
	_, err := engine.Run(ctx, []string{tmpFile})
	assert.ErrorIs(t, err, context.Canceled)
}

func BenchmarkLintLargeModule(b *testing.B) {
	dir := b.TempDir()

	// Create 50 .tf files simulating a large module
	for i := range 50 {
		content := fmt.Sprintf(`resource "aws_instance" "server_%d" {
  ami           = "ami-%06d"
  instance_type = "t2.micro"

  tags = {
    Name = "server-%d"
  }
}

variable "var_%d" {
  description = "Variable %d"
  type        = string
  default     = "value-%d"
}
`, i, i, i, i, i, i)
		f := filepath.Join(dir, fmt.Sprintf("file_%02d.tf", i))
		require.NoError(b, os.WriteFile(f, []byte(content), 0o644))
	}

	var files []string
	entries, err := os.ReadDir(dir)
	require.NoError(b, err)
	for _, e := range entries {
		files = append(files, filepath.Join(dir, e.Name()))
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _ = engine.Run(ctx, files)
	}
}

func BenchmarkLintModule(b *testing.B) {
	dir := b.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  tags = {
    Name = "test"
  }
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

output "instance_id" {
  description = "Instance ID"
  value       = aws_instance.test.id
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(b, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _ = engine.Run(ctx, []string{tmpFile})
	}
}

// TestRun_TFLintEnabled_NotAvailable_FallbackFalse verifies that when
// UseTFLint is true, TFLint is not in PATH, and FallbackBuiltin is false,
// Run returns an error instead of silently falling back.
func TestRun_TFLintEnabled_NotAvailable_FallbackFalse(t *testing.T) {
	if _, err := exec.LookPath("tflint"); err == nil {
		t.Skip("tflint is available on this system; this test requires it to be absent")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "aws_instance" "x" {}`), 0o644))

	engine := New(&Config{
		UseTFLint:       true,
		FallbackBuiltin: false,
	})

	_, err := engine.Run(context.Background(), []string{filepath.Join(dir, "main.tf")})
	assert.Error(t, err, "should error when TFLint requested but unavailable and FallbackBuiltin is false")
	assert.Contains(t, err.Error(), "tflint")
}

// TestRun_TFLintEnabled_NotAvailable_FallbackTrue verifies that when
// UseTFLint is true, TFLint is not in PATH, but FallbackBuiltin is true,
// the engine falls through to built-in rules without error.
func TestRun_TFLintEnabled_NotAvailable_FallbackTrue(t *testing.T) {
	if _, err := exec.LookPath("tflint"); err == nil {
		t.Skip("tflint is available on this system; this test requires it to be absent")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "aws_instance" "x" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`+"\n"), 0o644))

	engine := New(&Config{
		UseTFLint:       true,
		FallbackBuiltin: true,
	})

	findings, err := engine.Run(context.Background(), []string{filepath.Join(dir, "main.tf")})
	assert.NoError(t, err, "should fall back to built-in rules when TFLint unavailable")
	assert.NotNil(t, findings)
}

// TestRun_TFLintEnabled_CustomPath_NotFound verifies that a non-existent
// custom TFLintPath with FallbackBuiltin=false produces an error.
func TestRun_TFLintEnabled_CustomPath_NotFound(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "aws_instance" "x" {}`), 0o644))

	engine := New(&Config{
		UseTFLint:       true,
		FallbackBuiltin: false,
		TFLintPath:      "/nonexistent/path/to/tflint",
	})

	_, err := engine.Run(context.Background(), []string{filepath.Join(dir, "main.tf")})
	assert.Error(t, err, "should error when custom TFLintPath does not exist")
}

// TestValidateTFLintPath_CustomPath_IsDir verifies that a directory path is
// rejected with a clear error message.
func TestValidateTFLintPath_CustomPath_IsDir(t *testing.T) {
	dir := t.TempDir()

	engine := New(&Config{
		TFLintPath: dir, // a directory, not an executable
	})

	err := engine.validateTFLintPath()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory")
}

// TestRunWithTFLint_WhenAvailable tests the RunWithTFLint path when tflint is installed.
func TestRunWithTFLint_WhenAvailable(t *testing.T) {
	if _, err := exec.LookPath("tflint"); err != nil {
		t.Skip("tflint not available")
	}

	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		UseTFLint:       true,
		FallbackBuiltin: true,
	})

	// This will call RunWithTFLint
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	// TFLint may return findings or errors depending on config, but shouldn't panic
	_ = err
	_ = findings
}

// TestRunWithTFLint_ContextCancellation tests context cancellation in RunWithTFLint.
func TestRunWithTFLint_ContextCancellation(t *testing.T) {
	if _, err := exec.LookPath("tflint"); err != nil {
		t.Skip("tflint not available")
	}

	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami = "ami-123"
}
`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	engine := New(&Config{
		UseTFLint:       true,
		FallbackBuiltin: false,
	})

	_, err := engine.RunWithTFLint(ctx, []string{tmpFile})
	assert.ErrorIs(t, err, context.Canceled)
}

// TestRunWithTFLint_FallbackOnError tests fallback to built-in when TFLint fails.
func TestRunWithTFLint_FallbackOnError(t *testing.T) {
	// Create a fake tflint that outputs to stderr and fails
	dir := t.TempDir()
	fakeTFLint := filepath.Join(dir, "tflint")
	script := "#!/bin/sh\necho 'tflint error: config not found' >&2\nexit 1\n"
	require.NoError(t, os.WriteFile(fakeTFLint, []byte(script), 0o755))

	tfDir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(tfDir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		UseTFLint:       true,
		TFLintPath:      fakeTFLint,
		FallbackBuiltin: true,
	})

	// RunWithTFLint should fallback to built-in rules when TFLint fails
	findings, err := engine.RunWithTFLint(context.Background(), []string{tmpFile})
	// Should not error when fallback is enabled
	require.NoError(t, err)
	// findings may be empty slice, that's fine
	_ = findings
}

// TestRunWithTFLint_NoFallbackOnError tests error propagation when fallback is disabled.
func TestRunWithTFLint_NoFallbackOnError(t *testing.T) {
	// Create a fake tflint that outputs to stderr and fails
	dir := t.TempDir()
	fakeTFLint := filepath.Join(dir, "tflint")
	script := "#!/bin/sh\necho 'tflint error: config not found' >&2\nexit 1\n"
	require.NoError(t, os.WriteFile(fakeTFLint, []byte(script), 0o755))

	tfDir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami = "ami-123"
}
`
	tmpFile := filepath.Join(tfDir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		UseTFLint:       true,
		TFLintPath:      fakeTFLint,
		FallbackBuiltin: false,
	})

	// RunWithTFLint should return error when fallback is disabled
	_, err := engine.RunWithTFLint(context.Background(), []string{tmpFile})
	assert.Error(t, err)
}
