package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceNameMatchesTypeRule(t *testing.T) {
	rule := &ResourceNameMatchesTypeRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.resource-name-matches-type", rule.Name())
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
			name: "good descriptive name",
			content: `resource "aws_instance" "web_server" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "good descriptive name with purpose",
			content: `resource "aws_s3_bucket" "logs_bucket" {
  bucket = "my-logs"
}`,
			wantFindings: 0,
		},
		{
			name: "generic name 'this'",
			content: `resource "aws_instance" "this" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "generic name 'main'",
			content: `resource "aws_instance" "main" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "generic name 'foo'",
			content: `resource "aws_instance" "foo" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "generic name 'example'",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "name just repeats type",
			content: `resource "aws_instance" "instance" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "data source with generic name",
			content: `data "aws_ami" "this" {
  most_recent = true
}`,
			wantFindings: 1,
		},
		{
			name: "data source with good name",
			content: `data "aws_ami" "ubuntu_latest" {
  most_recent = true
}`,
			wantFindings: 0,
		},
		{
			name: "module block is not checked",
			content: `module "this" {
  source = "./module"
}`,
			wantFindings: 0,
		},
		{
			name: "variable block is not checked",
			content: `variable "this" {
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

func TestResourceNameMatchesTypeRule_ExtractTypeWords(t *testing.T) {
	rule := &ResourceNameMatchesTypeRule{}

	tests := []struct {
		resourceType string
		expected     []string
	}{
		{"aws_instance", []string{"instance"}},
		{"aws_s3_bucket", []string{"s3", "bucket"}},
		{"google_compute_instance", []string{"compute", "instance"}},
		{"azurerm_virtual_machine", []string{"virtual", "machine"}},
		{"null_resource", []string{"resource"}},
		{"random_string", []string{"string"}},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := rule.extractTypeWords(tt.resourceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResourceNameMatchesTypeRule_NameJustRepeatsType(t *testing.T) {
	rule := &ResourceNameMatchesTypeRule{}

	tests := []struct {
		name      string
		typeWords []string
		expected  bool
	}{
		{"instance", []string{"instance"}, true},
		{"web_server", []string{"instance"}, false},
		{"bucket", []string{"s3", "bucket"}, true},
		{"s3_bucket", []string{"s3", "bucket"}, true},
		{"logs_bucket", []string{"s3", "bucket"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.nameJustRepeatsType(tt.name, tt.typeWords)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputPrefixRule(t *testing.T) {
	rule := &OutputPrefixRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.output-prefix", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		config       map[string]any
		wantFindings int
	}{
		{
			name: "no config - any name is valid",
			content: `output "anything" {
  value = "test"
}`,
			config:       nil,
			wantFindings: 0,
		},
		{
			name: "prefix matches",
			content: `output "vpc_id" {
  value = module.vpc.id
}`,
			config: map[string]any{
				"options": map[string]any{
					"prefix": "vpc_",
				},
			},
			wantFindings: 0,
		},
		{
			name: "prefix missing",
			content: `output "id" {
  value = module.vpc.id
}`,
			config: map[string]any{
				"options": map[string]any{
					"prefix": "vpc_",
				},
			},
			wantFindings: 1,
		},
		{
			name: "suffix matches",
			content: `output "instance_id" {
  value = aws_instance.web.id
}`,
			config: map[string]any{
				"options": map[string]any{
					"suffix": "_id",
				},
			},
			wantFindings: 0,
		},
		{
			name: "suffix missing",
			content: `output "instance" {
  value = aws_instance.web.id
}`,
			config: map[string]any{
				"options": map[string]any{
					"suffix": "_id",
				},
			},
			wantFindings: 1,
		},
		{
			name: "both prefix and suffix match",
			content: `output "out_instance_id" {
  value = aws_instance.web.id
}`,
			config: map[string]any{
				"options": map[string]any{
					"prefix": "out_",
					"suffix": "_id",
				},
			},
			wantFindings: 0,
		},
		{
			name: "both prefix and suffix missing",
			content: `output "instance" {
  value = aws_instance.web.id
}`,
			config: map[string]any{
				"options": map[string]any{
					"prefix": "out_",
					"suffix": "_id",
				},
			},
			wantFindings: 2,
		},
		{
			name: "variable block is ignored",
			content: `variable "no_prefix" {
  type = string
}`,
			config: map[string]any{
				"options": map[string]any{
					"prefix": "out_",
				},
			},
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{
				File:   "test.tf",
				Config: tt.config,
			}

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

func TestModuleNameConventionRule(t *testing.T) {
	rule := &ModuleNameConventionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.module-name-convention", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		config       map[string]any
		wantFindings int
	}{
		{
			name: "valid snake_case module name",
			content: `module "my_vpc" {
  source = "./modules/vpc"
}`,
			config:       nil,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase module name",
			content: `module "myVpc" {
  source = "./modules/vpc"
}`,
			config:       nil,
			wantFindings: 1,
		},
		{
			name: "generic name 'this'",
			content: `module "this" {
  source = "./modules/vpc"
}`,
			config:       nil,
			wantFindings: 1,
		},
		{
			name: "generic name 'main'",
			content: `module "main" {
  source = "./modules/vpc"
}`,
			config:       nil,
			wantFindings: 1,
		},
		{
			name: "generic name 'module'",
			content: `module "module" {
  source = "./modules/vpc"
}`,
			config:       nil,
			wantFindings: 1,
		},
		{
			name: "camelCase valid with camelCase config",
			content: `module "myVpc" {
  source = "./modules/vpc"
}`,
			config: map[string]any{
				"options": map[string]any{
					"case": "camelCase",
				},
			},
			wantFindings: 0,
		},
		{
			name: "resource block is ignored",
			content: `resource "aws_instance" "this" {
  ami = "ami-123"
}`,
			config:       nil,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{
				File:   "test.tf",
				Config: tt.config,
			}

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
