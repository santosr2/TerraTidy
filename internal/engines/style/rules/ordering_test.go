package rules

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

func TestForEachCountFirstRule(t *testing.T) {
	rule := &ForEachCountFirstRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.for-each-count-first", rule.Name())
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
			name: "for_each is first",
			content: `resource "aws_instance" "example" {
  for_each      = var.instances
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "for_each is not first",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  for_each      = var.instances
  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name: "count is first",
			content: `resource "aws_instance" "example" {
  count         = 3
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "count is not first",
			content: `resource "aws_instance" "example" {
  ami   = "ami-123"
  count = 3
}`,
			wantFindings: 1,
		},
		{
			name: "no for_each or count",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "module with for_each not first",
			content: `module "example" {
  source   = "./module"
  for_each = var.items
}`,
			wantFindings: 1,
		},
		{
			name: "data source with count not first",
			content: `data "aws_ami" "example" {
  most_recent = true
  count       = 2
}`,
			wantFindings: 1,
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

	t.Run("Fix reorders for_each to first", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  ami      = "ami-123"
  for_each = var.instances
  name     = "test"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		assert.NotNil(t, result)
		// for_each should be near the beginning
		resultStr := string(result)
		forEachIdx := indexOf(resultStr, "for_each")
		amiIdx := indexOf(resultStr, "ami")
		assert.Less(t, forEachIdx, amiIdx)
	})
}

func TestLifecycleAtEndRule(t *testing.T) {
	rule := &LifecycleAtEndRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.lifecycle-at-end", rule.Name())
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
			name: "lifecycle at end",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "lifecycle not at end",
			content: `resource "aws_instance" "example" {
  lifecycle {
    prevent_destroy = true
  }
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "no lifecycle",
			content: `resource "aws_instance" "example" {
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

func TestTagsAtEndRule(t *testing.T) {
	rule := &TagsAtEndRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.tags-at-end", rule.Name())
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
			name: "tags at end",
			content: `resource "aws_instance" "example" {
  ami  = "ami-123"
  tags = { Name = "test" }
}`,
			wantFindings: 0,
		},
		{
			name: "tags before lifecycle",
			content: `resource "aws_instance" "example" {
  ami  = "ami-123"
  tags = { Name = "test" }
  lifecycle {
    prevent_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "tags after lifecycle",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true
  }
  tags = { Name = "test" }
}`,
			wantFindings: 1,
		},
		{
			name: "no tags",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "labels in module",
			content: `module "example" {
  source = "./module"
  labels = { env = "prod" }
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
}

func TestDependsOnOrderRule(t *testing.T) {
	rule := &DependsOnOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.depends-on-order", rule.Name())
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
			name: "depends_on at end",
			content: `resource "aws_instance" "example" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
}`,
			wantFindings: 0,
		},
		{
			name: "depends_on not at end",
			content: `resource "aws_instance" "example" {
  depends_on = [aws_vpc.main]
  ami        = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "no depends_on",
			content: `resource "aws_instance" "example" {
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

func TestSourceVersionGroupedRule(t *testing.T) {
	rule := &SourceVersionGroupedRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.source-version-grouped", rule.Name())
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
			name: "source and version grouped at start",
			content: `module "example" {
  source  = "./module"
  version = "1.0.0"
  name    = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "source not at start",
			content: `module "example" {
  name    = "test"
  source  = "./module"
  version = "1.0.0"
}`,
			wantFindings: 1,
		},
		{
			name: "version not immediately after source",
			content: `module "example" {
  source  = "./module"
  name    = "test"
  version = "1.0.0"
}`,
			wantFindings: 1,
		},
		{
			name: "source only",
			content: `module "example" {
  source = "./module"
  name   = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "for_each before source is ok",
			content: `module "example" {
  for_each = var.items
  source   = "./module"
  version  = "1.0.0"
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
}

func TestVariableOrderRule(t *testing.T) {
	rule := &VariableOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.variable-order", rule.Name())
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
			name: "correct order",
			content: `variable "example" {
  description = "Example variable"
  type        = string
  default     = "value"
}`,
			wantFindings: 0,
		},
		{
			name: "type before description",
			content: `variable "example" {
  type        = string
  description = "Example variable"
}`,
			wantFindings: 1,
		},
		{
			name: "default before type",
			content: `variable "example" {
  default = "value"
  type    = string
}`,
			wantFindings: 1,
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

	t.Run("Fix reorders attributes", func(t *testing.T) {
		content := `variable "example" {
  type        = string
  description = "Example variable"
  default     = "value"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestOutputOrderRule(t *testing.T) {
	rule := &OutputOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.output-order", rule.Name())
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
			name: "correct order",
			content: `output "example" {
  description = "Example output"
  value       = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "value before description",
			content: `output "example" {
  value       = "test"
  description = "Example output"
}`,
			wantFindings: 1,
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

	t.Run("Fix reorders attributes", func(t *testing.T) {
		content := `output "example" {
  value       = "test"
  description = "Example output"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestTerraformBlockFirstRule(t *testing.T) {
	rule := &TerraformBlockFirstRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.terraform-block-first", rule.Name())
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
			name: "terraform block first",
			content: `terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "terraform block not first",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}

terraform {
  required_version = ">= 1.0"
}`,
			wantFindings: 1,
		},
		{
			name: "no terraform block",
			content: `resource "aws_instance" "example" {
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

func TestProviderBlockOrderRule(t *testing.T) {
	rule := &ProviderBlockOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.provider-block-order", rule.Name())
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
			name: "provider after terraform, before resources",
			content: `terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "provider before terraform",
			content: `provider "aws" {
  region = "us-east-1"
}

terraform {
  required_version = ">= 1.0"
}`,
			wantFindings: 1,
		},
		{
			name: "provider after resources",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}

provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 1,
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

func TestIsDependsOnRelevantBlock(t *testing.T) {
	tests := []struct {
		blockType string
		expected  bool
	}{
		{"resource", true},
		{"module", true},
		{"data", true},
		{"variable", false},
		{"output", false},
		{"terraform", false},
		{"locals", false},
		{"provider", false},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			result := IsDependsOnRelevantBlock(tt.blockType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to find index of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
