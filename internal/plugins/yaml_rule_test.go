package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRule_Name(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{Name: "test-rule"}}
	assert.Equal(t, "test-rule", rule.Name())
}

func TestYAMLRule_Description(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{Description: "A test rule"}}
	assert.Equal(t, "A test rule", rule.Description())
}

func TestYAMLRule_Check_Disabled(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{Enabled: false}}
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, &hcl.File{})
	require.NoError(t, err)
	assert.Nil(t, findings)
}

func TestYAMLRule_Check_RequiredAttribute(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:     "require-tags",
		Enabled:  true,
		Severity: "warning",
		Message:  "Resource must have tags",
		Patterns: YAMLPatterns{
			RequiredAttributes: []string{"tags"},
		},
	}}

	src := []byte(`resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "require-tags", findings[0].Rule)
	assert.Equal(t, "Resource must have tags", findings[0].Message)
	assert.Equal(t, sdk.SeverityWarning, findings[0].Severity)
}

func TestYAMLRule_Check_RequiredAttributePresent(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:     "require-tags",
		Enabled:  true,
		Severity: "warning",
		Patterns: YAMLPatterns{
			RequiredAttributes: []string{"tags"},
		},
	}}

	src := []byte(`resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  tags = {
    Name = "example"
  }
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestYAMLRule_Check_ResourceTypeFilter(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:     "s3-encryption",
		Enabled:  true,
		Severity: "error",
		Patterns: YAMLPatterns{
			ResourceTypes:      []string{"aws_s3_bucket"},
			RequiredAttributes: []string{"server_side_encryption_configuration"},
		},
	}}

	src := []byte(`resource "aws_instance" "example" {
  ami = "ami-12345"
}

resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	// Only the s3 bucket should trigger, not the instance
	assert.Len(t, findings, 1)
	assert.Equal(t, sdk.SeverityError, findings[0].Severity)
}

func TestYAMLRule_Check_DefaultMessage(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "require-desc",
		Enabled: true,
		Patterns: YAMLPatterns{
			RequiredAttributes: []string{"description"},
		},
	}}

	src := []byte(`resource "aws_instance" "example" {
  ami = "ami-12345"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "Missing required attribute: description", findings[0].Message)
}

func TestYAMLRule_Fix_ReturnsNil(t *testing.T) {
	rule := &YAMLRule{}
	result, err := rule.Fix(&sdk.Context{}, &hcl.File{})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadYAMLRule(t *testing.T) {
	content := `name: test-rule
description: A test rule
severity: error
enabled: true
message: "Test violation"
patterns:
  resource_types:
    - aws_instance
  required_attributes:
    - tags
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-rule.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	rule, err := loadYAMLRule(path)
	require.NoError(t, err)
	assert.Equal(t, "test-rule", rule.Name())
	assert.Equal(t, "A test rule", rule.Description())
	assert.True(t, rule.config.Enabled)
	assert.Equal(t, "error", rule.config.Severity)
}

func TestLoadYAMLRule_MissingName(t *testing.T) {
	content := `description: No name field
severity: warning
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	_, err := loadYAMLRule(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required 'name' field")
}

func TestLoadYAMLRule_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(":::invalid"), 0o644))

	_, err := loadYAMLRule(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing YAML rule")
}

func TestLoadYAMLRule_FileNotFound(t *testing.T) {
	_, err := loadYAMLRule("/nonexistent/rule.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading YAML rule")
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected sdk.Severity
	}{
		{"error", sdk.SeverityError},
		{"Error", sdk.SeverityError},
		{"ERROR", sdk.SeverityError},
		{"warning", sdk.SeverityWarning},
		{"info", sdk.SeverityInfo},
		{"unknown", sdk.SeverityWarning},
		{"", sdk.SeverityWarning},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseSeverity(tt.input))
		})
	}
}

func TestYAMLRule_Check_NonHclsyntaxBody(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "test",
		Enabled: true,
	}}
	// hcl.File with nil Body (not *hclsyntax.Body)
	file := &hcl.File{Body: hcl.EmptyBody()}
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	assert.NoError(t, err)
	assert.Nil(t, findings)
}

func TestYAMLRule_Check_NonResourceBlock(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "test",
		Enabled: true,
		Patterns: YAMLPatterns{
			RequiredAttributes: []string{"tags"},
		},
	}}
	body := &hclsyntax.Body{
		Blocks: []*hclsyntax.Block{
			{Type: "variable", Labels: []string{"name"}},
		},
	}
	file := &hcl.File{Body: body}
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	assert.NoError(t, err)
	assert.Empty(t, findings)
}

func TestManager_LoadAll_WithInvalidYAMLRule(t *testing.T) {
	tmpDir := t.TempDir()

	// YAML file missing required 'name' field
	content := "description: no name\nseverity: warning\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte(content), 0o644))

	manager := NewManager([]string{tmpDir})
	err := manager.LoadAll()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading YAML rule")
}

func TestManager_LoadAll_WithYAMLRule(t *testing.T) {
	tmpDir := t.TempDir()

	content := `name: yaml-test-rule
description: Test YAML rule loading
severity: warning
enabled: true
patterns:
  required_attributes:
    - tags
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte(content), 0o644))

	manager := NewManager([]string{tmpDir})
	err := manager.LoadAll()
	require.NoError(t, err)

	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "yaml-test-rule", plugins[0].Metadata.Name)

	rule, ok := manager.GetRule("yaml-test-rule")
	assert.True(t, ok)
	assert.Equal(t, "yaml-test-rule", rule.Name())
}
