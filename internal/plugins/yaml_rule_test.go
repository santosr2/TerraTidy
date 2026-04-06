package plugins

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
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

func TestYAMLRule_Check_ForbiddenAttribute(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:     "no-acl",
		Enabled:  true,
		Severity: "error",
		Patterns: YAMLPatterns{
			ResourceTypes:       []string{"aws_s3_bucket"},
			ForbiddenAttributes: []string{"acl"},
		},
	}}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "no-acl", findings[0].Rule)
	assert.Contains(t, findings[0].Message, "Forbidden attribute present: acl")
	assert.Equal(t, sdk.SeverityError, findings[0].Severity)
}

func TestYAMLRule_Check_ForbiddenAttributeNotPresent(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:     "no-acl",
		Enabled:  true,
		Severity: "error",
		Patterns: YAMLPatterns{
			ResourceTypes:       []string{"aws_s3_bucket"},
			ForbiddenAttributes: []string{"acl"},
		},
	}}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestYAMLRule_Check_ForbiddenAttributeCustomMessage(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:     "no-acl",
		Enabled:  true,
		Severity: "warning",
		Message:  "Use aws_s3_bucket_acl resource instead of acl argument",
		Patterns: YAMLPatterns{
			ForbiddenAttributes: []string{"acl"},
		},
	}}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "Use aws_s3_bucket_acl resource instead of acl argument", findings[0].Message)
}

func TestYAMLRule_Check_ForbiddenAttributeLocation(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "no-acl",
		Enabled: true,
		Patterns: YAMLPatterns{
			ForbiddenAttributes: []string{"acl"},
		},
	}}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	// Location should point to the acl attribute line, not the block
	assert.Equal(t, 3, findings[0].Location.StartLine)
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

func TestYAMLRule_Tags(t *testing.T) {
	t.Run("returns configured tags", func(t *testing.T) {
		rule := &YAMLRule{config: YAMLRuleConfig{
			Name: "tagged-rule",
			Tags: []string{"security", "compliance"},
		}}
		assert.Equal(t, []string{"security", "compliance"}, rule.Tags())
	})

	t.Run("returns nil for rule with no tags", func(t *testing.T) {
		rule := &YAMLRule{config: YAMLRuleConfig{Name: "no-tags"}}
		assert.Nil(t, rule.Tags())
	})
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

func TestYAMLRule_Check_BlockTypeFiltering(t *testing.T) {
	// With explicit block_types, only matching blocks are checked
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "test",
		Enabled: true,
		Patterns: YAMLPatterns{
			BlockTypes:         []string{"resource"}, // Only check resources
			RequiredAttributes: []string{"tags"},
		},
	}}
	body := &hclsyntax.Body{
		Blocks: []*hclsyntax.Block{
			{Type: "variable", Labels: []string{"name"}}, // Should be skipped
		},
	}
	file := &hcl.File{Body: body}
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	assert.NoError(t, err)
	assert.Empty(t, findings) // Variable block skipped because block_types = ["resource"]
}

func TestYAMLRule_Check_BlockTypes_VariableOnly(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "require-variable-desc",
		Enabled: true,
		Patterns: YAMLPatterns{
			BlockTypes:         []string{"variable"},
			RequiredAttributes: []string{"description"},
		},
	}}

	src := []byte(`variable "name" {
  type = string
}

resource "aws_instance" "example" {
  ami = "ami-12345"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	// Only the variable should trigger, not the resource
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "description")
}

func TestYAMLRule_Check_BlockTypes_MultipleTypes(t *testing.T) {
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "require-desc",
		Enabled: true,
		Patterns: YAMLPatterns{
			BlockTypes:         []string{"resource", "data"},
			RequiredAttributes: []string{"tags"},
		},
	}}

	src := []byte(`resource "aws_instance" "example" {
  ami = "ami-12345"
}

data "aws_ami" "example" {
  most_recent = true
}

variable "name" {
  type = string
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	// Both resource and data should trigger, but not variable
	assert.Len(t, findings, 2)
}

func TestYAMLRule_Check_BlockTypes_EmptyMatchesAll(t *testing.T) {
	// When block_types is empty, should match all block types
	rule := &YAMLRule{config: YAMLRuleConfig{
		Name:    "require-tags",
		Enabled: true,
		Patterns: YAMLPatterns{
			// BlockTypes intentionally empty = match all
			RequiredAttributes: []string{"tags"},
		},
	}}

	src := []byte(`resource "aws_instance" "example" {
  ami = "ami-12345"
}

variable "name" {
  type = string
}

data "aws_ami" "example" {
  most_recent = true
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	// All blocks should trigger when block_types is empty
	assert.Len(t, findings, 3)
}

func TestYAMLRule_matchesBlockType(t *testing.T) {
	tests := []struct {
		name       string
		blockTypes []string
		blockType  string
		expected   bool
	}{
		{
			name:       "empty block_types matches resource",
			blockTypes: []string{},
			blockType:  "resource",
			expected:   true,
		},
		{
			name:       "empty block_types matches variable",
			blockTypes: []string{},
			blockType:  "variable",
			expected:   true,
		},
		{
			name:       "explicit variable matches variable",
			blockTypes: []string{"variable"},
			blockType:  "variable",
			expected:   true,
		},
		{
			name:       "explicit variable rejects resource",
			blockTypes: []string{"variable"},
			blockType:  "resource",
			expected:   false,
		},
		{
			name:       "multiple types match first",
			blockTypes: []string{"resource", "data", "module"},
			blockType:  "resource",
			expected:   true,
		},
		{
			name:       "multiple types match middle",
			blockTypes: []string{"resource", "data", "module"},
			blockType:  "data",
			expected:   true,
		},
		{
			name:       "multiple types reject unlisted",
			blockTypes: []string{"resource", "data"},
			blockType:  "variable",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &YAMLRule{config: YAMLRuleConfig{
				Patterns: YAMLPatterns{BlockTypes: tt.blockTypes},
			}}
			block := &hclsyntax.Block{Type: tt.blockType}
			assert.Equal(t, tt.expected, rule.matchesBlockType(block))
		})
	}
}

func TestLoadYAMLRule_WithBlockTypes(t *testing.T) {
	content := `name: block-type-rule
description: Test block types parsing
severity: warning
enabled: true
patterns:
  block_types:
    - variable
    - output
  required_attributes:
    - description
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "block-type-rule.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	rule, err := loadYAMLRule(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"variable", "output"}, rule.config.Patterns.BlockTypes)
}

func TestLoadYAMLRule_WithForbiddenAttributes(t *testing.T) {
	content := `name: no-deprecated-attrs
description: Forbid deprecated attributes
severity: error
enabled: true
patterns:
  resource_types:
    - aws_s3_bucket
  forbidden_attributes:
    - acl
    - website
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "no-deprecated-attrs.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	rule, err := loadYAMLRule(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"acl", "website"}, rule.config.Patterns.ForbiddenAttributes)
}

func TestLoadYAMLRule_WithAttributePatterns(t *testing.T) {
	content := `name: bucket-naming
description: Validate bucket naming
severity: warning
enabled: true
patterns:
  attribute_patterns:
    - attribute: bucket
      pattern: "^[a-z0-9-]+$"
      message: "Bucket name must be lowercase alphanumeric with hyphens"
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bucket-naming.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	rule, err := loadYAMLRule(path)
	require.NoError(t, err)
	require.Len(t, rule.config.Patterns.AttributePatterns, 1)
	assert.Equal(t, "bucket", rule.config.Patterns.AttributePatterns[0].Attribute)
	assert.Equal(t, "^[a-z0-9-]+$", rule.config.Patterns.AttributePatterns[0].Pattern)
	// Verify regex was compiled
	require.Len(t, rule.compiledPatterns, 1)
	assert.NotNil(t, rule.compiledPatterns[0].regex)
}

func TestLoadYAMLRule_InvalidRegex(t *testing.T) {
	content := `name: bad-regex
description: Invalid regex pattern
severity: warning
enabled: true
patterns:
  attribute_patterns:
    - attribute: bucket
      pattern: "[invalid"
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad-regex.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	_, err := loadYAMLRule(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestYAMLRule_Check_AttributePattern_Matches(t *testing.T) {
	rule := &YAMLRule{
		config: YAMLRuleConfig{
			Name:     "bucket-naming",
			Enabled:  true,
			Severity: "warning",
			Patterns: YAMLPatterns{
				ResourceTypes: []string{"aws_s3_bucket"},
			},
		},
		compiledPatterns: []compiledPattern{
			{
				AttributePattern: AttributePattern{
					Attribute: "bucket",
					Pattern:   "^[a-z0-9-]+$",
				},
				regex: regexp.MustCompile("^[a-z0-9-]+$"),
			},
		},
	}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-valid-bucket-name"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, findings, "valid bucket name should not produce findings")
}

func TestYAMLRule_Check_AttributePattern_NoMatch(t *testing.T) {
	rule := &YAMLRule{
		config: YAMLRuleConfig{
			Name:     "bucket-naming",
			Enabled:  true,
			Severity: "warning",
			Patterns: YAMLPatterns{
				ResourceTypes: []string{"aws_s3_bucket"},
			},
		},
		compiledPatterns: []compiledPattern{
			{
				AttributePattern: AttributePattern{
					Attribute: "bucket",
					Pattern:   "^[a-z0-9-]+$",
				},
				regex: regexp.MustCompile("^[a-z0-9-]+$"),
			},
		},
	}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "My_Invalid_Bucket"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "bucket")
	assert.Contains(t, findings[0].Message, "My_Invalid_Bucket")
}

func TestYAMLRule_Check_AttributePattern_CustomMessage(t *testing.T) {
	rule := &YAMLRule{
		config: YAMLRuleConfig{
			Name:     "bucket-naming",
			Enabled:  true,
			Severity: "error",
			Patterns: YAMLPatterns{},
		},
		compiledPatterns: []compiledPattern{
			{
				AttributePattern: AttributePattern{
					Attribute: "bucket",
					Pattern:   "^[a-z0-9-]+$",
					Message:   "Bucket names must be lowercase with hyphens only",
				},
				regex: regexp.MustCompile("^[a-z0-9-]+$"),
			},
		},
	}

	src := []byte(`resource "aws_s3_bucket" "example" {
  bucket = "UPPERCASE"
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "Bucket names must be lowercase with hyphens only", findings[0].Message)
}

func TestYAMLRule_Check_AttributePattern_MissingAttribute(t *testing.T) {
	rule := &YAMLRule{
		config: YAMLRuleConfig{
			Name:     "bucket-naming",
			Enabled:  true,
			Patterns: YAMLPatterns{},
		},
		compiledPatterns: []compiledPattern{
			{
				AttributePattern: AttributePattern{
					Attribute: "bucket",
					Pattern:   "^[a-z0-9-]+$",
				},
				regex: regexp.MustCompile("^[a-z0-9-]+$"),
			},
		},
	}

	// No bucket attribute
	src := []byte(`resource "aws_s3_bucket" "example" {
  tags = {}
}
`)
	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)
	assert.Empty(t, findings, "missing attribute should be skipped gracefully")
}

func TestManager_LoadAll_WithInvalidYAMLRule(t *testing.T) {
	tmpDir := t.TempDir()

	// YAML file missing required 'name' field
	content := "description: no name\nseverity: warning\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte(content), 0o644))

	manager := NewManager([]string{tmpDir}, false)
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

	manager := NewManager([]string{tmpDir}, false)
	err := manager.LoadAll()
	require.NoError(t, err)

	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "yaml-test-rule", plugins[0].Metadata.Name)

	rule, ok := manager.GetRule("yaml-test-rule")
	assert.True(t, ok)
	assert.Equal(t, "yaml-test-rule", rule.Name())
}

// Integration test: load YAML rule with block_types from file and verify findings
func TestYAMLRule_BlockTypes_Integration(t *testing.T) {
	// Create a YAML rule file with block_types
	pluginDir := t.TempDir()
	ruleContent := `name: require-variable-desc
description: Variables must have description
severity: warning
enabled: true
patterns:
  block_types:
    - variable
  required_attributes:
    - description
`
	rulePath := filepath.Join(pluginDir, "require-variable-desc.yaml")
	require.NoError(t, os.WriteFile(rulePath, []byte(ruleContent), 0o644))

	// Load the rule via the plugin manager
	manager := NewManager([]string{pluginDir}, false)
	err := manager.LoadAll()
	require.NoError(t, err)

	rule, ok := manager.GetRule("require-variable-desc")
	require.True(t, ok, "rule should be loaded")

	// Create test Terraform content with variables and resources
	tfContent := []byte(`variable "name" {
  type = string
}

variable "with_desc" {
  type        = string
  description = "This variable has a description"
}

resource "aws_instance" "example" {
  ami = "ami-12345"
}

output "result" {
  value = "test"
}
`)
	file, diags := hclsyntax.ParseConfig(tfContent, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())

	// Check the file
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, file)
	require.NoError(t, err)

	// Should only report the variable without description
	// The resource and output should be ignored due to block_types filter
	assert.Len(t, findings, 1, "should only find 1 variable missing description")
	assert.Equal(t, "require-variable-desc", findings[0].Rule)
	assert.Contains(t, findings[0].Message, "description")
}
