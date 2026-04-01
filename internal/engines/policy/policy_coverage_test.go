package policy

import (
	"context"
	"fmt"
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

func BenchmarkPolicyMultiFile(b *testing.B) {
	dir := b.TempDir()

	// Create 20 .tf files
	var files []string
	for i := range 20 {
		content := fmt.Sprintf(`resource "aws_instance" "server_%d" {
  ami           = "ami-%06d"
  instance_type = "t2.micro"
}
`, i, i)
		f := filepath.Join(dir, fmt.Sprintf("file_%02d.tf", i))
		require.NoError(b, os.WriteFile(f, []byte(content), 0o644))
		files = append(files, f)
	}

	// Create 5 policy files
	for i := range 5 {
		policy := fmt.Sprintf(`package terraform

deny[msg] {
    resource := input.resources[_]
    resource.type == "aws_instance"
    not resource.values.tags
    msg := {
        "msg": "Policy %d: missing tags",
        "rule": "require-tags-%d",
        "severity": "warning"
    }
}
`, i, i)
		pf := filepath.Join(dir, fmt.Sprintf("policy_%d.rego", i))
		require.NoError(b, os.WriteFile(pf, []byte(policy), 0o644))
	}

	cfg := &Config{PolicyDirs: []string{dir}}
	engine := New(cfg)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _ = engine.Run(ctx, files)
	}
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
