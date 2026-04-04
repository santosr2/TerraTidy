package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockLabelCaseRule(t *testing.T) {
	rule := &BlockLabelCaseRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.block-label-case", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case resource",
			content: `resource "aws_instance" "my_server" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase resource",
			content: `resource "aws_instance" "myServer" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "invalid PascalCase resource",
			content: `resource "aws_instance" "MyServer" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "valid snake_case data source",
			content: `data "aws_ami" "latest_ubuntu" {
  most_recent = true
}`,
			wantFindings: 0,
		},
		{
			name: "invalid data source name",
			content: `data "aws_ami" "latestUbuntu" {
  most_recent = true
}`,
			wantFindings: 1,
		},
		{
			name: "module with any name is valid",
			content: `module "myModule" {
  source = "./module"
}`,
			wantFindings: 0,
		},
		{
			name: "empty label",
			content: `resource "aws_instance" "" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "variable block is ignored",
			content: `variable "MyVariable" {
  type = string
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix returns nil", func(t *testing.T) {
		result, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestVariableNamingRule(t *testing.T) {
	rule := &VariableNamingRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.variable-naming", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case variable",
			content: `variable "my_variable" {
  type = string
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase variable",
			content: `variable "myVariable" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name: "invalid PascalCase variable",
			content: `variable "MyVariable" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name: "resource is ignored",
			content: `resource "aws_instance" "myServer" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix returns nil", func(t *testing.T) {
		result, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestOutputNamingRule(t *testing.T) {
	rule := &OutputNamingRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.output-naming", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case output",
			content: `output "my_output" {
  value = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase output",
			content: `output "myOutput" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "invalid PascalCase output",
			content: `output "MyOutput" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "variable is ignored",
			content: `variable "myVariable" {
  type = string
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix returns nil", func(t *testing.T) {
		result, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestLocalNamingRule(t *testing.T) {
	rule := &LocalNamingRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.local-naming", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case local",
			content: `locals {
  my_local = "value"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase local",
			content: `locals {
  myLocal = "value"
}`,
			wantFindings: 1,
		},
		{
			name: "multiple locals mixed",
			content: `locals {
  valid_name   = "ok"
  invalidName  = "bad"
  another_good = "ok"
}`,
			wantFindings: 1,
		},
		{
			name: "variable block is ignored",
			content: `variable "myVariable" {
  type = string
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix returns nil", func(t *testing.T) {
		result, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

// TestNamingRulesWithConfig tests naming rules with different naming conventions from config.
func TestNamingRulesWithConfig(t *testing.T) {
	t.Run("VariableNamingRule with camelCase config", func(t *testing.T) {
		rule := &VariableNamingRule{}
		content := `variable "myVariable" {
  type = string
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{
			File: "test.tf",
			Options: map[string]any{
				"options": map[string]any{
					"case": "camelCase",
				},
			},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 0, "camelCase should be valid with camelCase config")
	})

	t.Run("VariableNamingRule snake_case invalid with camelCase config", func(t *testing.T) {
		rule := &VariableNamingRule{}
		content := `variable "my_variable" {
  type = string
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{
			File: "test.tf",
			Options: map[string]any{
				"options": map[string]any{
					"case": "camelCase",
				},
			},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 1, "snake_case should be invalid with camelCase config")
	})

	t.Run("LocalNamingRule with kebab-case config", func(t *testing.T) {
		rule := &LocalNamingRule{}
		content := `locals {
  my-local = "value"
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{
			File: "test.tf",
			Options: map[string]any{
				"options": map[string]any{
					"case": "kebab-case",
				},
			},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 0, "kebab-case should be valid with kebab-case config")
	})

	t.Run("OutputNamingRule with PascalCase config", func(t *testing.T) {
		rule := &OutputNamingRule{}
		content := `output "MyOutput" {
  value = "test"
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{
			File: "test.tf",
			Options: map[string]any{
				"options": map[string]any{
					"case": "PascalCase",
				},
			},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 0, "PascalCase should be valid with PascalCase config")
	})

	t.Run("BlockLabelCaseRule with custom pattern config", func(t *testing.T) {
		rule := &BlockLabelCaseRule{}
		content := `resource "aws_instance" "prefix_server" {
  ami = "ami-123"
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{
			File: "test.tf",
			Options: map[string]any{
				"options": map[string]any{
					"case":    "custom",
					"pattern": "^prefix_",
				},
			},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 0, "prefix_server should match custom pattern ^prefix_")
	})

	t.Run("BlockLabelCaseRule custom pattern invalid", func(t *testing.T) {
		rule := &BlockLabelCaseRule{}
		content := `resource "aws_instance" "server" {
  ami = "ami-123"
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{
			File: "test.tf",
			Options: map[string]any{
				"options": map[string]any{
					"case":    "custom",
					"pattern": "^prefix_",
				},
			},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 1, "server should not match custom pattern ^prefix_")
	})
}
