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

func TestNoBlankLinesInsideBlocksRule(t *testing.T) {
	rule := &NoBlankLinesInsideBlocksRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.no-blank-lines-inside-blocks", rule.Name())
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
			name: "no blank lines inside block",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "blank line inside block",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"

  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name: "multiple blank lines inside block",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"


  instance_type = "t2.micro"
}`,
			wantFindings: 2,
		},
		{
			name: "nested block with blank line",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true

    ignore_changes = [tags]
  }
}`,
			wantFindings: 2, // One for outer block (blank line within) and one for nested lifecycle block
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)

			// Verify findings are fixable
			for _, f := range findings {
				assert.True(t, f.Fixable)
				assert.NotNil(t, f.FixFunc)
			}
		})
	}

	t.Run("Fix removes blank lines", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  ami = "ami-123"

  instance_type = "t2.micro"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotContains(t, string(result), "\n\n  instance_type")
	})
}

func TestBlankLineBetweenBlocksRule(t *testing.T) {
	rule := &BlankLineBetweenBlocksRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.blank-line-between-blocks", rule.Name())
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
			name: "proper spacing between blocks",
			content: `resource "aws_instance" "a" {
  ami = "ami-123"
}

resource "aws_instance" "b" {
  ami = "ami-456"
}`,
			wantFindings: 0,
		},
		{
			name: "missing blank line",
			content: `resource "aws_instance" "a" {
  ami = "ami-123"
}
resource "aws_instance" "b" {
  ami = "ami-456"
}`,
			wantFindings: 1,
		},
		{
			name: "too many blank lines",
			content: `resource "aws_instance" "a" {
  ami = "ami-123"
}


resource "aws_instance" "b" {
  ami = "ami-456"
}`,
			wantFindings: 1,
		},
		{
			name:         "single block",
			content:      `resource "aws_instance" "a" { ami = "ami-123" }`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
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

	t.Run("Fix adds missing blank line", func(t *testing.T) {
		content := `resource "aws_instance" "a" {
  ami = "ami-123"
}
resource "aws_instance" "b" {
  ami = "ami-456"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		// Should have blank line between blocks
		assert.Contains(t, string(result), "}\n\nresource")
	})

	t.Run("Fix removes extra blank lines", func(t *testing.T) {
		content := `resource "aws_instance" "a" {
  ami = "ami-123"
}



resource "aws_instance" "b" {
  ami = "ami-456"
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

func TestNoEmptyBlocksRule(t *testing.T) {
	rule := &NoEmptyBlocksRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.no-empty-blocks", rule.Name())
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
			name: "non-empty block",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name:         "empty resource block",
			content:      `resource "aws_instance" "example" {}`,
			wantFindings: 1,
		},
		{
			name: "empty terraform block is allowed",
			content: `terraform {
}`,
			wantFindings: 0,
		},
		{
			name: "empty required_providers block is allowed",
			content: `terraform {
  required_providers {}
}`,
			wantFindings: 0,
		},
		{
			name:         "empty variable block",
			content:      `variable "example" {}`,
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
