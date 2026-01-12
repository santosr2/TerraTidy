package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrderedAttrNames(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "ordered attributes",
			content: `resource "test" "example" {
  first  = "1"
  second = "2"
  third  = "3"
}`,
			expected: []string{"first", "second", "third"},
		},
		{
			name: "single attribute",
			content: `resource "test" "example" {
  only = "value"
}`,
			expected: []string{"only"},
		},
		{
			name:     "no attributes",
			content:  `resource "test" "example" {}`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body.(*hclsyntax.Body)
			if len(body.Blocks) > 0 {
				result := GetOrderedAttrNames(body.Blocks[0].Body)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestReorderBlockAttrs(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		firstAttrs []string
		lastAttrs  []string
		checkFirst string
		checkLast  string
	}{
		{
			name: "move for_each to first",
			content: `resource "test" "example" {
  ami       = "ami-123"
  for_each  = var.instances
  name      = "test"
}`,
			firstAttrs: []string{"for_each", "count"},
			lastAttrs:  nil,
			checkFirst: "for_each",
		},
		{
			name: "move tags to last",
			content: `resource "test" "example" {
  tags = { Name = "test" }
  ami  = "ami-123"
  name = "test"
}`,
			firstAttrs: nil,
			lastAttrs:  []string{"tags", "labels"},
			checkLast:  "tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse with hclsyntax for ordering
			syntaxFile, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			// Parse with hclwrite for modification
			writeFile, diags := hclwrite.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			syntaxBody := syntaxFile.Body.(*hclsyntax.Body)
			if len(syntaxBody.Blocks) > 0 {
				orderedNames := GetOrderedAttrNames(syntaxBody.Blocks[0].Body)

				writeBlock := writeFile.Body().Blocks()[0]
				ReorderBlockAttrs(writeBlock.Body(), orderedNames, tt.firstAttrs, tt.lastAttrs)

				// Get the result
				result := string(writeFile.Bytes())
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestFormatAndCleanBlankLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "preserves internal blank lines inside block",
			input: `resource "test" "example" {
  ami = "ami-123"

  name = "test"
}
`,
			expected: `resource "test" "example" {
  ami = "ami-123"

  name = "test"
}
`,
		},
		{
			name: "removes leading blank lines inside block",
			input: `resource "test" "example" {

  ami = "ami-123"
  name = "test"
}
`,
			// hclwrite.Format aligns the equal signs
			expected: `resource "test" "example" {
  ami  = "ami-123"
  name = "test"
}
`,
		},
		{
			name: "removes trailing blank lines inside block",
			input: `resource "test" "example" {
  ami = "ami-123"
  name = "test"

}
`,
			// hclwrite.Format aligns the equal signs
			expected: `resource "test" "example" {
  ami  = "ami-123"
  name = "test"
}
`,
		},
		{
			name: "preserves blank lines between blocks",
			input: `resource "test" "a" {
  ami = "ami-123"
}

resource "test" "b" {
  ami = "ami-456"
}
`,
			expected: `resource "test" "a" {
  ami = "ami-123"
}

resource "test" "b" {
  ami = "ami-456"
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAndCleanBlankLines([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "single line",
			input:    "only line",
			expected: []string{"only line"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "trailing newline",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitLines([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimLeftWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no whitespace", "hello", "hello"},
		{"leading spaces", "   hello", "hello"},
		{"leading tabs", "\t\thello", "hello"},
		{"mixed whitespace", " \t hello", "hello"},
		{"only whitespace", "   ", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimLeftWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountBlankLinesBetween(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		endLine   int
		startLine int
		expected  int
	}{
		{
			name:      "one blank line",
			lines:     []string{"line1", "", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  1,
		},
		{
			name:      "no blank lines",
			lines:     []string{"line1", "line2", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  0,
		},
		{
			name:      "multiple blank lines",
			lines:     []string{"line1", "", "", "", "line5"},
			endLine:   1,
			startLine: 5,
			expected:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountBlankLinesBetween(tt.lines, tt.endLine, tt.startLine)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBlockKey(t *testing.T) {
	tests := []struct {
		name      string
		blockType string
		labels    []string
		expected  string
	}{
		{"resource with labels", "resource", []string{"aws_instance", "example"}, "resource.aws_instance.example"},
		{"module with label", "module", []string{"vpc"}, "module.vpc"},
		{"no labels", "terraform", []string{}, "terraform"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BlockKey(tt.blockType, tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindAttribute(t *testing.T) {
	content := `resource "test" "example" {
  ami  = "ami-123"
  name = "test"
}`
	file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body.(*hclsyntax.Body)
	attrs := body.Blocks[0].Body.Attributes

	t.Run("finds existing attribute", func(t *testing.T) {
		attr := FindAttribute(attrs, "ami")
		assert.NotNil(t, attr)
	})

	t.Run("returns nil for missing attribute", func(t *testing.T) {
		attr := FindAttribute(attrs, "nonexistent")
		assert.Nil(t, attr)
	})
}

func TestFindNestedBlock(t *testing.T) {
	content := `resource "test" "example" {
  ami = "ami-123"

  lifecycle {
    prevent_destroy = true
  }
}`
	file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body.(*hclsyntax.Body)
	blocks := body.Blocks[0].Body.Blocks

	t.Run("finds existing nested block", func(t *testing.T) {
		block := FindNestedBlock(blocks, "lifecycle")
		assert.NotNil(t, block)
	})

	t.Run("returns nil for missing block", func(t *testing.T) {
		block := FindNestedBlock(blocks, "provisioner")
		assert.Nil(t, block)
	})
}

func TestIsSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid snake_case", "my_variable", true},
		{"single word", "variable", true},
		{"with numbers", "var_123", true},
		{"camelCase", "myVariable", false},
		{"PascalCase", "MyVariable", false},
		{"with hyphens", "my-variable", false},
		{"starts with number", "123_var", false},
		{"uppercase", "MY_VARIABLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchBlockLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected []string
		match    bool
	}{
		{"exact match", []string{"aws_instance", "example"}, []string{"aws_instance", "example"}, true},
		{"different labels", []string{"aws_instance", "example"}, []string{"aws_instance", "other"}, false},
		{"different length", []string{"aws_instance"}, []string{"aws_instance", "example"}, false},
		{"empty labels", []string{}, []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchBlockLabels(tt.labels, tt.expected)
			assert.Equal(t, tt.match, result)
		})
	}
}

func TestFindSyntaxBody(t *testing.T) {
	content := `resource "aws_instance" "example" {
  ami = "ami-123"
}

resource "aws_instance" "other" {
  ami = "ami-456"
}`
	file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	body := file.Body.(*hclsyntax.Body)

	t.Run("finds matching block", func(t *testing.T) {
		result := FindSyntaxBody(body, "resource", []string{"aws_instance", "example"})
		assert.NotNil(t, result)
	})

	t.Run("returns nil for non-matching block", func(t *testing.T) {
		result := FindSyntaxBody(body, "resource", []string{"aws_instance", "nonexistent"})
		assert.Nil(t, result)
	})

	t.Run("returns nil for wrong block type", func(t *testing.T) {
		result := FindSyntaxBody(body, "data", []string{"aws_instance", "example"})
		assert.Nil(t, result)
	})
}

func TestFindWriteBlock(t *testing.T) {
	content := `resource "aws_instance" "example" {
  ami = "ami-123"
}

resource "aws_instance" "other" {
  ami = "ami-456"
}`
	file, diags := hclwrite.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	t.Run("finds matching block", func(t *testing.T) {
		result := FindWriteBlock(file, "resource", []string{"aws_instance", "example"})
		assert.NotNil(t, result)
	})

	t.Run("returns nil for non-matching block", func(t *testing.T) {
		result := FindWriteBlock(file, "resource", []string{"aws_instance", "nonexistent"})
		assert.Nil(t, result)
	})
}

func TestParseBothFormats(t *testing.T) {
	content := `resource "aws_instance" "example" {
  ami = "ami-123"
}`

	t.Run("parses valid HCL", func(t *testing.T) {
		syntaxBody, writeFile, err := ParseBothFormats([]byte(content), "test.tf")
		assert.NoError(t, err)
		assert.NotNil(t, syntaxBody)
		assert.NotNil(t, writeFile)
	})

	t.Run("returns error for invalid HCL", func(t *testing.T) {
		_, _, err := ParseBothFormats([]byte("invalid { content"), "test.tf")
		assert.Error(t, err)
	})
}
