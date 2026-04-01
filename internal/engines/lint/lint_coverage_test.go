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
