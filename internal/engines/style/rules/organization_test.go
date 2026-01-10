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

func TestVariablesInFileRule(t *testing.T) {
	rule := &VariablesInFileRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.variables-in-file", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		filename     string
		content      string
		wantFindings int
	}{
		{
			name:     "variable in variables.tf is valid",
			filename: "variables.tf",
			content: `variable "example" {
  type = string
}`,
			wantFindings: 0,
		},
		{
			name:     "variable in main.tf reports finding",
			filename: "main.tf",
			content: `variable "example" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name:     "resource in main.tf is valid",
			filename: "main.tf",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name:     "multiple variables in wrong file",
			filename: "other.tf",
			content: `variable "var1" {
  type = string
}

variable "var2" {
  type = number
}`,
			wantFindings: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

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

func TestOutputsInFileRule(t *testing.T) {
	rule := &OutputsInFileRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.outputs-in-file", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		filename     string
		content      string
		wantFindings int
	}{
		{
			name:     "output in outputs.tf is valid",
			filename: "outputs.tf",
			content: `output "example" {
  value = "test"
}`,
			wantFindings: 0,
		},
		{
			name:     "output in main.tf reports finding",
			filename: "main.tf",
			content: `output "example" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name:     "resource in main.tf is valid",
			filename: "main.tf",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

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

func TestProvidersInFileRule(t *testing.T) {
	rule := &ProvidersInFileRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.providers-in-file", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		filename     string
		content      string
		wantFindings int
	}{
		{
			name:     "provider in providers.tf is valid",
			filename: "providers.tf",
			content: `provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 0,
		},
		{
			name:     "provider in versions.tf is valid",
			filename: "versions.tf",
			content: `provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 0,
		},
		{
			name:     "provider in main.tf reports finding",
			filename: "main.tf",
			content: `provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 1,
		},
		{
			name:     "resource in main.tf is valid",
			filename: "main.tf",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

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

func TestExtractBasename(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"unix path", "/home/user/project/main.tf", "main.tf"},
		{"windows path", "C:\\Users\\project\\main.tf", "main.tf"},
		{"just filename", "main.tf", "main.tf"},
		{"nested path", "/a/b/c/d/variables.tf", "variables.tf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBasename(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
