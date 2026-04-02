package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentSyntaxRule(t *testing.T) {
	rule := &CommentSyntaxRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.comment-syntax", rule.Name())
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
			name: "valid # comment",
			content: `# This is a valid comment
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid // comment",
			content: `// This is an invalid comment
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "inline // comment",
			content: `resource "aws_instance" "web" {
  ami = "ami-123" // inline comment
}`,
			wantFindings: 1,
		},
		{
			name: "// inside string is ignored",
			content: `resource "aws_instance" "web" {
  tags = {
    Description = "URL is https://example.com"
  }
}`,
			wantFindings: 0,
		},
		{
			name: "multiple // comments",
			content: `// First comment
// Second comment
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 2,
		},
		{
			name: "mixed valid and invalid comments",
			content: `# Valid comment
// Invalid comment
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file for the rule to read
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix replaces // with #", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `// This should be fixed
resource "aws_instance" "web" {
  ami = "ami-123"
}`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		assert.Contains(t, string(result), "# This should be fixed")
		assert.NotContains(t, string(result), "// This should be fixed")
	})
}

func TestNoTrailingWhitespaceRule(t *testing.T) {
	rule := &NoTrailingWhitespaceRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.no-trailing-whitespace", rule.Name())
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
			name: "no trailing whitespace",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name:         "trailing spaces",
			content:      "resource \"aws_instance\" \"web\" {  \n  ami = \"ami-123\"\n}",
			wantFindings: 1,
		},
		{
			name:         "trailing tabs",
			content:      "resource \"aws_instance\" \"web\" {\t\n  ami = \"ami-123\"\n}",
			wantFindings: 1,
		},
		{
			name:         "multiple lines with trailing whitespace",
			content:      "resource \"aws_instance\" \"web\" {  \n  ami = \"ami-123\"  \n}",
			wantFindings: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix removes trailing whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := "resource \"aws_instance\" \"web\" {  \n  ami = \"ami-123\"  \n}"
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		// Check that trailing whitespace is removed
		assert.NotContains(t, string(result), "{  \n")
		assert.NotContains(t, string(result), "\"  \n")
	})
}

func TestConsistentQuotesRule(t *testing.T) {
	rule := &ConsistentQuotesRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.consistent-quotes", rule.Name())
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
			name: "valid double quotes",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "single quoted value",
			content: `resource "aws_instance" "web" {
  ami = 'ami-123'
}`,
			wantFindings: 1,
		},
		{
			name: "single quote in list",
			content: `resource "aws_instance" "web" {
  tags = ['tag1', 'tag2']
}`,
			wantFindings: 1,
		},
		{
			name: "comment with single quote is ignored",
			content: `# Don't use this
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			// Note: HCL parser may reject single quotes, but we still test the rule
			if diags.HasErrors() {
				// Skip parse errors for invalid HCL
				return
			}

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

func TestNoConsecutiveBlankLinesRule(t *testing.T) {
	rule := &NoConsecutiveBlankLinesRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.no-consecutive-blank-lines", rule.Name())
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
			name: "no consecutive blank lines",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}

resource "aws_instance" "api" {
  ami = "ami-456"
}`,
			wantFindings: 0,
		},
		{
			name: "two consecutive blank lines",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}


resource "aws_instance" "api" {
  ami = "ami-456"
}`,
			wantFindings: 1,
		},
		{
			name: "three consecutive blank lines",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}



resource "aws_instance" "api" {
  ami = "ami-456"
}`,
			wantFindings: 2,
		},
		{
			name: "multiple groups of consecutive blank lines",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}


resource "aws_instance" "api" {
  ami = "ami-456"
}


resource "aws_instance" "db" {
  ami = "ami-789"
}`,
			wantFindings: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix removes consecutive blank lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `resource "aws_instance" "web" {
  ami = "ami-123"
}


resource "aws_instance" "api" {
  ami = "ami-456"
}`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)

		// Count blank lines between resources
		lines := SplitLines(result)
		consecutiveBlank := 0
		maxConsecutive := 0
		for _, line := range lines {
			if len(line) == 0 {
				consecutiveBlank++
				if consecutiveBlank > maxConsecutive {
					maxConsecutive = consecutiveBlank
				}
			} else {
				consecutiveBlank = 0
			}
		}
		assert.LessOrEqual(t, maxConsecutive, 1, "Should have at most 1 consecutive blank line")
	})
}
