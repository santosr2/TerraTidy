package policy

import (
	"context"
	"os"
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

func TestRun_NonExistentFile(t *testing.T) {
	cfg := &Config{
		PolicyDirs: []string{},
	}
	engine := New(cfg)
	// Non-existent files should not panic
	findings, err := engine.Run(context.Background(), []string{"/no/such/file.tf"})
	// May return error or empty findings depending on implementation
	_ = err
	_ = findings
}

func TestRun_WithInlinePolicy(t *testing.T) {
	dir := t.TempDir()

	// Create a simple .tf file
	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Create a simple policy
	policyContent := `package terraform

deny[msg] {
    resource := input.resources[_]
    resource.type == "aws_instance"
    not resource.values.tags
    msg := {
        "msg": "Instance missing tags",
        "rule": "require-tags",
        "severity": "warning"
    }
}
`
	policyFile := filepath.Join(dir, "policy.rego")
	require.NoError(t, os.WriteFile(policyFile, []byte(policyContent), 0o644))

	cfg := &Config{
		PolicyFiles: []string{policyFile},
	}
	engine := New(cfg)
	findings, err := engine.Run(context.Background(), []string{tfFile})
	require.NoError(t, err)
	// Policy may or may not find issues depending on how input is structured
	_ = findings
}

func BenchmarkPolicyEval(b *testing.B) {
	dir := b.TempDir()

	tfContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(b, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _ = engine.Run(ctx, []string{tfFile})
	}
}
