package rules

import (
	"strings"
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

func TestIsCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid camelCase", "myVariable", true},
		{"single word lowercase", "variable", true},
		{"with numbers", "var123Test", true},
		{"snake_case", "my_variable", false},
		{"PascalCase", "MyVariable", false},
		{"with hyphens", "my-variable", false},
		{"starts with number", "123var", false},
		{"starts with uppercase", "MyVariable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsKebabCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid kebab-case", "my-variable", true},
		{"single word", "variable", true},
		{"with numbers", "var-123", true},
		{"snake_case", "my_variable", false},
		{"camelCase", "myVariable", false},
		{"PascalCase", "MyVariable", false},
		{"starts with number", "123-var", false},
		{"uppercase", "MY-VARIABLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsKebabCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid PascalCase", "MyVariable", true},
		{"single uppercase word", "Variable", true},
		{"with numbers", "Var123Test", true},
		{"snake_case", "my_variable", false},
		{"camelCase", "myVariable", false},
		{"with hyphens", "My-Variable", false},
		{"starts with number", "123Variable", false},
		{"all lowercase", "myvariable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesCustomPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pattern  string
		expected bool
	}{
		{"matches simple pattern", "test123", "^test[0-9]+$", true},
		{"doesn't match pattern", "test", "^test[0-9]+$", false},
		{"empty pattern returns true", "anything", "", true},
		{"invalid regex returns false", "test", "[invalid", false},
		{"prefix pattern", "prefix_name", "^prefix_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesCustomPattern(tt.input, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNaming(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		convention    NamingCase
		customPattern string
		expectValid   bool
		expectCase    string
	}{
		{"snake_case valid", "my_var", SnakeCase, "", true, "snake_case"},
		{"snake_case invalid", "myVar", SnakeCase, "", false, "snake_case"},
		{"camelCase valid", "myVar", CamelCase, "", true, "camelCase"},
		{"camelCase invalid", "my_var", CamelCase, "", false, "camelCase"},
		{"kebab-case valid", "my-var", KebabCase, "", true, "kebab-case"},
		{"kebab-case invalid", "my_var", KebabCase, "", false, "kebab-case"},
		{"PascalCase valid", "MyVar", PascalCase, "", true, "PascalCase"},
		{"PascalCase invalid", "myVar", PascalCase, "", false, "PascalCase"},
		{"custom pattern valid", "prefix_name", CustomCase, "^prefix_", true, "custom pattern"},
		{"custom pattern invalid", "name", CustomCase, "^prefix_", false, "custom pattern"},
		{"custom empty pattern", "anything", CustomCase, "", true, "custom"},
		{"default to snake_case", "my_var", "", "", true, "snake_case"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, caseName := ValidateNaming(tt.input, tt.convention, tt.customPattern)
			assert.Equal(t, tt.expectValid, valid)
			assert.Equal(t, tt.expectCase, caseName)
		})
	}
}

func TestGetNamingConventionFromConfig(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]any
		expectConvention NamingCase
		expectPattern    string
	}{
		{
			name:             "nil config returns snake_case",
			config:           nil,
			expectConvention: SnakeCase,
			expectPattern:    "",
		},
		{
			name:             "empty config returns snake_case",
			config:           map[string]any{},
			expectConvention: SnakeCase,
			expectPattern:    "",
		},
		{
			name: "snake_case from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "snake_case",
				},
			},
			expectConvention: SnakeCase,
			expectPattern:    "",
		},
		{
			name: "camelCase from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "camelCase",
				},
			},
			expectConvention: CamelCase,
			expectPattern:    "",
		},
		{
			name: "kebab-case from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "kebab-case",
				},
			},
			expectConvention: KebabCase,
			expectPattern:    "",
		},
		{
			name: "PascalCase from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "PascalCase",
				},
			},
			expectConvention: PascalCase,
			expectPattern:    "",
		},
		{
			name: "custom with pattern",
			config: map[string]any{
				"options": map[string]any{
					"case":    "custom",
					"pattern": "^prefix_",
				},
			},
			expectConvention: CustomCase,
			expectPattern:    "^prefix_",
		},
		{
			name: "unknown case defaults to snake_case",
			config: map[string]any{
				"options": map[string]any{
					"case": "UNKNOWN",
				},
			},
			expectConvention: SnakeCase,
			expectPattern:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convention, pattern := GetNamingConventionFromConfig(tt.config)
			assert.Equal(t, tt.expectConvention, convention)
			assert.Equal(t, tt.expectPattern, pattern)
		})
	}
}

func TestGetAttributeOrderFromConfig(t *testing.T) {
	defaultOrder := map[string]int{
		"description": 1,
		"type":        2,
		"default":     3,
	}

	tests := []struct {
		name          string
		config        map[string]any
		expectedOrder map[string]int
	}{
		{
			name:          "nil config returns default",
			config:        nil,
			expectedOrder: defaultOrder,
		},
		{
			name:          "empty config returns default",
			config:        map[string]any{},
			expectedOrder: defaultOrder,
		},
		{
			name: "custom order from config",
			config: map[string]any{
				"options": map[string]any{
					"order": []any{"type", "description", "default"},
				},
			},
			expectedOrder: map[string]int{
				"type":        1,
				"description": 2,
				"default":     3,
			},
		},
		{
			name: "custom order with additional attributes",
			config: map[string]any{
				"options": map[string]any{
					"order": []any{"description", "value", "sensitive", "depends_on"},
				},
			},
			expectedOrder: map[string]int{
				"description": 1,
				"value":       2,
				"sensitive":   3,
				"depends_on":  4,
			},
		},
		{
			name: "missing options returns default",
			config: map[string]any{
				"other": "value",
			},
			expectedOrder: defaultOrder,
		},
		{
			name: "wrong options type returns default",
			config: map[string]any{
				"options": "not a map",
			},
			expectedOrder: defaultOrder,
		},
		{
			name: "wrong order type returns default",
			config: map[string]any{
				"options": map[string]any{
					"order": "not a list",
				},
			},
			expectedOrder: defaultOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAttributeOrderFromConfig(tt.config, defaultOrder)
			assert.Equal(t, tt.expectedOrder, result)
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

func TestGetExprTokensWithTrailingComment(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		attrName        string
		expectComment   bool
		commentContains string
	}{
		{
			name: "attribute with inline comment",
			content: `resource "test" "example" {
  ami = "ami-123" # this is a comment
}`,
			attrName:        "ami",
			expectComment:   true,
			commentContains: "this is a comment",
		},
		{
			name: "attribute without comment",
			content: `resource "test" "example" {
  ami = "ami-123"
}`,
			attrName:      "ami",
			expectComment: false,
		},
		{
			name: "attribute with // style comment",
			content: `resource "test" "example" {
  ami = "ami-123" // another comment style
}`,
			attrName:        "ami",
			expectComment:   true,
			commentContains: "another comment style",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFile, diags := hclwrite.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			for _, block := range writeFile.Body().Blocks() {
				attr := block.Body().GetAttribute(tt.attrName)
				require.NotNil(t, attr, "attribute %s not found", tt.attrName)

				tokens := getExprTokensWithTrailingComment(attr)
				assert.NotEmpty(t, tokens)

				// Check if there's a comment token
				hasComment := false
				for _, tok := range tokens {
					if tok.Type.String() == "TokenComment" {
						hasComment = true
						if tt.commentContains != "" {
							assert.Contains(t, string(tok.Bytes), tt.commentContains)
						}
						break
					}
				}

				assert.Equal(t, tt.expectComment, hasComment, "comment presence mismatch")
			}
		})
	}
}

func TestReorderBlockAttrsPreservesComments(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		firstAttrs []string
		lastAttrs  []string
		checkFirst string
		checkLast  string
		// Check that these comments are preserved in the output
		expectComments []string
	}{
		{
			name: "preserves inline comment when reordering",
			content: `resource "test" "example" {
  ami      = "ami-123"
  for_each = var.instances # loop over instances
  name     = "test"
}`,
			firstAttrs:     []string{"for_each", "count"},
			lastAttrs:      nil,
			checkFirst:     "for_each",
			expectComments: []string{"loop over instances"},
		},
		{
			name: "preserves multiple inline comments",
			content: `resource "test" "example" {
  tags = { Name = "test" } # resource tags
  ami  = "ami-123" # the ami id
  name = "test"
}`,
			firstAttrs:     nil,
			lastAttrs:      []string{"tags", "labels"},
			checkLast:      "tags",
			expectComments: []string{"resource tags", "the ami id"},
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

				// Check that all expected comments are preserved
				for _, comment := range tt.expectComments {
					assert.Contains(t, result, comment, "comment should be preserved: %s", comment)
				}
			}
		})
	}
}

func TestExtractAttrRegions(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectRegions  []string
		expectComment  map[string]string // attr name -> expected comment content
		expectMinLines map[string]int    // attr name -> minimum number of lines
	}{
		{
			name: "attributes with leading comments",
			//nolint:dupword // HCL content intentionally contains repeated identifiers
			content: `resource "test" "example" {
  # This is a comment for ami
  ami = "ami-123"

  # Comment for name
  name = "test"
}`,
			expectRegions: []string{"ami", "name"},
			expectComment: map[string]string{
				"ami":  "# This is a comment for ami",
				"name": "# Comment for name",
			},
		},
		{
			name: "attribute without comment",
			content: `resource "test" "example" {
  ami = "ami-123"
}`,
			expectRegions: []string{"ami"},
			expectComment: map[string]string{},
		},
		{
			name: "multi-line attribute spans all lines",
			content: `resource "test" "example" {
  tags = {
    Name = "test"
    Env  = "prod"
  }
}`,
			expectRegions:  []string{"tags"},
			expectMinLines: map[string]int{"tags": 4},
		},
		{
			name: "empty block returns no regions",
			content: `resource "test" "example" {
}`,
			expectRegions: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body.(*hclsyntax.Body)
			require.NotEmpty(t, body.Blocks, "test content should have at least one block")

			regions := ExtractAttrRegions([]byte(tt.content), body.Blocks[0].Body)

			// Verify exact count of regions (catches spurious extra regions)
			assert.Len(t, regions, len(tt.expectRegions),
				"expected %d regions, got %d", len(tt.expectRegions), len(regions))

			// Check expected regions exist
			for _, name := range tt.expectRegions {
				assert.Contains(t, regions, name, "region %s should exist", name)
			}

			// Check comments if specified
			for name, expectedComment := range tt.expectComment {
				region, ok := regions[name]
				require.True(t, ok, "region %s should exist for comment check", name)
				assert.Contains(t, region.LeadingComment, expectedComment,
					"attribute %s should have comment containing: %s", name, expectedComment)
			}

			// Check multi-line attributes span expected lines
			for name, minLines := range tt.expectMinLines {
				region, ok := regions[name]
				require.True(t, ok, "region %s should exist for line count check", name)
				assert.GreaterOrEqual(t, len(region.Lines), minLines,
					"attribute %s should span at least %d lines, got %d", name, minLines, len(region.Lines))
			}
		})
	}
}

func TestReorderBlockAttrsPreservingComments(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		firstAttrs   []string
		lastAttrs    []string
		checkFirst   string // attribute that should appear first
		checkLast    string // attribute that should appear last
		expectInline []string
	}{
		{
			name: "for_each moves to first with comment preserved",
			//nolint:dupword // HCL content intentionally contains repeated identifiers
			content: `resource "test" "example" {
  # Comment for ami
  ami = "ami-123"

  # Comment for for_each
  for_each = var.items

  name = "test"
}`,
			firstAttrs:   []string{"for_each"},
			lastAttrs:    nil,
			checkFirst:   "for_each",
			expectInline: []string{"# Comment for for_each", "# Comment for ami"},
		},
		{
			name: "tags moves to end with comment preserved",
			content: `resource "test" "example" {
  # Tags comment
  tags = { Name = "test" }
  ami  = "ami-123"
}`,
			firstAttrs:   nil,
			lastAttrs:    []string{"tags"},
			checkLast:    "tags",
			expectInline: []string{"# Tags comment"},
		},
		{
			name: "firstAttrs targeting absent attribute leaves block unchanged",
			content: `resource "test" "example" {
  ami = "ami-123"
}`,
			firstAttrs:   []string{"for_each"},
			lastAttrs:    nil,
			expectInline: []string{`ami = "ami-123"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body.(*hclsyntax.Body)
			require.NotEmpty(t, body.Blocks, "test content should have at least one block")

			block := body.Blocks[0]
			orderedNames := GetOrderedAttrNames(block.Body)

			result := ReorderBlockAttrsPreservingComments(
				[]byte(tt.content),
				block.Body,
				block.Range().Start.Line,
				block.Range().End.Line,
				orderedNames,
				tt.firstAttrs,
				tt.lastAttrs,
			)

			resultStr := string(result)
			assert.NotEmpty(t, resultStr)

			// Verify positional ordering if checkFirst is specified
			if tt.checkFirst != "" {
				firstIdx := strings.Index(resultStr, tt.checkFirst)
				require.NotEqual(t, -1, firstIdx, "%s should be in result", tt.checkFirst)

				// Check it comes before other attributes
				for _, name := range orderedNames {
					if name != tt.checkFirst {
						otherIdx := strings.Index(resultStr, name+" =")
						if otherIdx != -1 {
							assert.Less(t, firstIdx, otherIdx,
								"%s should appear before %s", tt.checkFirst, name)
						}
					}
				}
			}

			// Verify positional ordering if checkLast is specified
			if tt.checkLast != "" {
				lastIdx := strings.Index(resultStr, tt.checkLast)
				require.NotEqual(t, -1, lastIdx, "%s should be in result", tt.checkLast)

				// Check it comes after other attributes
				for _, name := range orderedNames {
					if name != tt.checkLast {
						otherIdx := strings.Index(resultStr, name+" =")
						if otherIdx != -1 {
							assert.Greater(t, lastIdx, otherIdx,
								"%s should appear after %s", tt.checkLast, name)
						}
					}
				}
			}

			// Check expected content is preserved
			for _, expected := range tt.expectInline {
				assert.Contains(t, resultStr, expected, "should contain: %s", expected)
			}
		})
	}
}

func TestReorderBlockAttrs_EdgeCases(t *testing.T) {
	t.Run("empty orderedNames leaves block unchanged", func(t *testing.T) {
		content := `resource "test" "example" {
  ami = "ami-123"
}`
		writeFile, diags := hclwrite.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		block := writeFile.Body().Blocks()[0]
		ReorderBlockAttrs(block.Body(), []string{}, []string{"for_each"}, nil)

		// Should be structurally unchanged
		result := string(writeFile.Bytes())
		assert.Contains(t, result, `ami = "ami-123"`, "original attribute assignment should be preserved")
	})

	t.Run("nonexistent firstAttrs are skipped without panic", func(t *testing.T) {
		content := `resource "test" "example" {
  ami = "ami-123"
}`
		syntaxFile, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		writeFile, diags := hclwrite.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		syntaxBody := syntaxFile.Body.(*hclsyntax.Body)
		orderedNames := GetOrderedAttrNames(syntaxBody.Blocks[0].Body)

		// Add a name that doesn't exist in the body
		orderedNames = append(orderedNames, "nonexistent_attr")

		writeBlock := writeFile.Body().Blocks()[0]
		ReorderBlockAttrs(writeBlock.Body(), orderedNames, []string{"nonexistent_attr"}, nil)

		// Should not panic and original content preserved
		result := string(writeFile.Bytes())
		assert.Contains(t, result, `ami = "ami-123"`, "original attribute should be preserved")
	})

	t.Run("for_each moves first even when other attrs not in priority list", func(t *testing.T) {
		content := `resource "test" "example" {
  z_attr = "z"
  a_attr = "a"
  for_each = var.items
}`
		syntaxFile, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		writeFile, diags := hclwrite.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		syntaxBody := syntaxFile.Body.(*hclsyntax.Body)
		orderedNames := GetOrderedAttrNames(syntaxBody.Blocks[0].Body)

		writeBlock := writeFile.Body().Blocks()[0]
		ReorderBlockAttrs(writeBlock.Body(), orderedNames, []string{"for_each"}, nil)

		result := string(writeFile.Bytes())

		// Verify for_each appears before other attributes
		forEachIdx := strings.Index(result, "for_each")
		zAttrIdx := strings.Index(result, "z_attr")
		aAttrIdx := strings.Index(result, "a_attr")

		require.NotEqual(t, -1, forEachIdx, "for_each should be in result")
		require.NotEqual(t, -1, zAttrIdx, "z_attr should be in result")
		require.NotEqual(t, -1, aAttrIdx, "a_attr should be in result")

		assert.Less(t, forEachIdx, zAttrIdx, "for_each should appear before z_attr")
		assert.Less(t, forEachIdx, aAttrIdx, "for_each should appear before a_attr")
	})

	t.Run("lastAttrs moves attribute to end", func(t *testing.T) {
		content := `resource "test" "example" {
  tags = { Name = "test" }
  ami  = "ami-123"
  name = "example"
}`
		syntaxFile, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		writeFile, diags := hclwrite.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())

		syntaxBody := syntaxFile.Body.(*hclsyntax.Body)
		orderedNames := GetOrderedAttrNames(syntaxBody.Blocks[0].Body)

		writeBlock := writeFile.Body().Blocks()[0]
		ReorderBlockAttrs(writeBlock.Body(), orderedNames, nil, []string{"tags"})

		result := string(writeFile.Bytes())

		// Verify tags appears after other attributes
		tagsIdx := strings.Index(result, "tags")
		amiIdx := strings.Index(result, "ami")
		nameIdx := strings.Index(result, "name")

		require.NotEqual(t, -1, tagsIdx, "tags should be in result")
		require.NotEqual(t, -1, amiIdx, "ami should be in result")
		require.NotEqual(t, -1, nameIdx, "name should be in result")

		assert.Greater(t, tagsIdx, amiIdx, "tags should appear after ami")
		assert.Greater(t, tagsIdx, nameIdx, "tags should appear after name")
	})
}
