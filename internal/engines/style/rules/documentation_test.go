package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireVariableDescriptionRule(t *testing.T) {
	rule := &RequireVariableDescriptionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.require-variable-description", rule.Name())
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
			name: "variable with description is valid",
			content: `variable "my_var" {
  description = "A test variable"
  type        = string
}`,
			wantFindings: 0,
		},
		{
			name: "variable without description reports finding",
			content: `variable "my_var" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name: "multiple variables mixed",
			content: `variable "with_desc" {
  description = "Has description"
  type        = string
}

variable "without_desc" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name: "resource is ignored",
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

func TestRequireOutputDescriptionRule(t *testing.T) {
	rule := &RequireOutputDescriptionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.require-output-description", rule.Name())
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
			name: "output with description is valid",
			content: `output "my_output" {
  description = "A test output"
  value       = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "output without description reports finding",
			content: `output "my_output" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "multiple outputs mixed",
			content: `output "with_desc" {
  description = "Has description"
  value       = "test"
}

output "without_desc" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "variable is ignored",
			content: `variable "my_var" {
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

func TestRequireVariableTypeRule(t *testing.T) {
	rule := &RequireVariableTypeRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.require-variable-type", rule.Name())
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
			name: "variable with type is valid",
			content: `variable "my_var" {
  type = string
}`,
			wantFindings: 0,
		},
		{
			name: "variable without type reports finding",
			content: `variable "my_var" {
  description = "A variable"
}`,
			wantFindings: 1,
		},
		{
			name: "variable with only default reports finding",
			content: `variable "my_var" {
  default = "value"
}`,
			wantFindings: 1,
		},
		{
			name: "multiple variables mixed",
			content: `variable "with_type" {
  type = string
}

variable "without_type" {
  description = "No type"
}`,
			wantFindings: 1,
		},
		{
			name: "output is ignored",
			content: `output "my_output" {
  value = "test"
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
