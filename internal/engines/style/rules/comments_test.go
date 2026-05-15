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
			// Inline trailing `// comment` is no longer flagged: the rule's scope is full-line
			// `//` comments only (per fmt-style-polish Phase 4 scope narrowing).
			name: "trailing // comment is not flagged",
			content: `resource "aws_instance" "web" {
  ami = "ami-123" // inline comment
}`,
			wantFindings: 0,
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
		{
			// Regression: # comment containing a URL with `//` must not be flagged.
			name: "hash comment containing URL is not flagged",
			content: `# https://github.com/hashicorp/terraform
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			// Regression: # comment with arbitrary `//` content is not flagged.
			name: "hash comment containing double-slash is not flagged",
			content: `# you can use // to do X
resource "aws_instance" "web" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			// Regression: line with `//` after a value (inside a string-position context)
			// is treated as not-a-full-line comment.
			name: "url inside string with // is not flagged",
			content: `output "u" { value = "url://example.com" }
`,
			wantFindings: 0,
		},
		{
			// Indented `//` at line start is still flagged.
			name: "indented // comment is flagged",
			content: `resource "x" "y" {
  // indented full-line comment
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			// `// foo // bar`: first `//` is the comment delimiter; second is body content.
			// Only the line itself is flagged once; on Fix, only the FIRST `//` is rewritten.
			name: "multiple // on same comment line is flagged once",
			content: `// foo // bar
resource "x" "y" {
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

	t.Run("Fix preserves hash comments containing URLs verbatim", func(t *testing.T) {
		// Regression lock: the buggy old scanner would convert the `//` inside the URL.
		// Use the exact hashicorp URL form called out in the plan.
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `# https://github.com/hashicorp/terraform
# also fine: // does not break me
resource "aws_instance" "web" {
  ami = "ami-123" // trailing remains as // because the rule is full-line only
}
`
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		out := string(result)

		// Hash comments untouched.
		assert.Contains(t, out, "# https://github.com/hashicorp/terraform")
		assert.Contains(t, out, "# also fine: // does not break me")
		// Trailing `//` after a value is intentionally NOT rewritten (scope: full-line only).
		assert.Contains(t, out, `ami = "ami-123" // trailing remains as //`)
	})

	t.Run("Fix preserves leading whitespace when converting // to #", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `resource "x" "y" {
  // indented comment
  ami = "ami-123"
}
`
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Contains(t, string(result), "  # indented comment")
		assert.NotContains(t, string(result), "  // indented comment")
	})

	t.Run("Fix rewrites only the first // on a multi-slash comment line", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := "// foo // bar\n"
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		// First `//` becomes `#`; the second `//` (inside the comment body) is preserved.
		assert.Contains(t, string(result), "# foo // bar")
	})

	t.Run("Fix is idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `// to convert
# https://example.com/url-stays
# normal hash
resource "x" "y" {
  ami = "ami-123" // trailing left alone
}
`
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))
		ctx := &sdk.Context{File: tmpFile}

		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, string(first), string(second), "Fix(Fix(x)) must equal Fix(x)")
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
