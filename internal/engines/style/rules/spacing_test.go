package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoLeadingTrailingBlankLinesRule(t *testing.T) {
	rule := &NoLeadingTrailingBlankLinesRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.no-leading-trailing-blank-lines", rule.Name())
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
			name: "no leading/trailing blank lines inside block",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "internal blank line is allowed",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"

  instance_type = "t2.micro"
}`,
			wantFindings: 0, // Internal blank lines are now allowed
		},
		{
			name: "leading blank line inside block",
			content: `resource "aws_instance" "example" {

  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 1, // Leading blank line after opening brace
		},
		{
			name: "trailing blank line inside block",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"

}`,
			wantFindings: 1, // Trailing blank line before closing brace
		},
		{
			name: "both leading and trailing blank lines",
			content: `resource "aws_instance" "example" {

  ami = "ami-123"

}`,
			wantFindings: 2, // One leading, one trailing
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

	t.Run("Fix removes leading/trailing blank lines but preserves internal", func(t *testing.T) {
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
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)
		fixed := string(result.Edits[0].Replacement)
		// Should remove leading blank line (after {)
		assert.NotContains(t, fixed, "{\n\n  ami")
		// Should remove trailing blank line (before })
		assert.NotContains(t, fixed, "micro\"\n\n}")
		// Internal blank line should be preserved
		assert.Contains(t, fixed, "ami-123\"\n\n  instance_type")
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
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)
		// Should have blank line between blocks
		assert.Contains(t, string(result.Edits[0].Replacement), "}\n\nresource")
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

	t.Run("Fix is a no-op on already-correctly-spaced blocks", func(t *testing.T) {
		// Already-canonical input: exactly one blank line between blocks.
		// Fix must return nil FixResult so the engine writes nothing back.
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
		assert.Nil(t, result, "already-canonical input must produce no edits")
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
}

func TestBlankLineBetweenBlocksRule_FixMultipleConsecutiveBlanks(t *testing.T) {
	// Test for BUG-3: inner loop was mutating outer loop counter
	// This caused issues when removing multiple consecutive blank lines
	rule := &BlankLineBetweenBlocksRule{}

	// Content with 4 consecutive blank lines between blocks
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
	require.NotNil(t, result)
	require.Len(t, result.Edits, 1)
	fixed := string(result.Edits[0].Replacement)

	// Should have exactly one blank line between blocks
	lines := strings.Split(fixed, "\n")
	blankCount := 0
	inBlanks := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if inBlanks {
				blankCount++
			}
			inBlanks = true
		} else {
			inBlanks = false
		}
	}

	// The fix should collapse multiple blanks to one
	assert.Contains(t, fixed, "}\n\nresource", "should have exactly one blank line")
	assert.NotContains(t, fixed, "}\n\n\nresource", "should not have multiple consecutive blanks")
}
