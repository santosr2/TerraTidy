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
			require.NotEmpty(t, syntaxBody.Blocks, "test content should have at least one block")

			orderedNames := GetOrderedAttrNames(syntaxBody.Blocks[0].Body)

			writeBlock := writeFile.Body().Blocks()[0]
			ReorderBlockAttrs(writeBlock.Body(), orderedNames, tt.firstAttrs, tt.lastAttrs)

			result := string(writeFile.Bytes())

			// Verify checkFirst attribute appears before others
			// Use "\n  " prefix to match line start (immune to hclwrite alignment padding)
			if tt.checkFirst != "" {
				firstIdx := strings.Index(result, "\n  "+tt.checkFirst)
				require.NotEqual(t, -1, firstIdx, "%s should be in result", tt.checkFirst)

				for _, name := range orderedNames {
					if name != tt.checkFirst {
						otherIdx := strings.Index(result, "\n  "+name)
						require.NotEqual(t, -1, otherIdx, "%s should be in result", name)
						assert.Less(t, firstIdx, otherIdx,
							"%s should appear before %s", tt.checkFirst, name)
					}
				}
			}

			// Verify checkLast attribute appears after others
			if tt.checkLast != "" {
				lastIdx := strings.LastIndex(result, "\n  "+tt.checkLast)
				require.NotEqual(t, -1, lastIdx, "%s should be in result", tt.checkLast)

				for _, name := range orderedNames {
					if name != tt.checkLast {
						otherIdx := strings.LastIndex(result, "\n  "+name)
						require.NotEqual(t, -1, otherIdx, "%s should be in result", name)
						assert.Greater(t, lastIdx, otherIdx,
							"%s should appear after %s", tt.checkLast, name)
					}
				}
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
		name            string
		content         string
		expectRegions   []string
		expectComment   map[string]string // attr name -> expected comment content
		expectLineCount map[string]int    // attr name -> exact number of lines expected
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
			expectRegions:   []string{"tags"},
			expectLineCount: map[string]int{"tags": 4},
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

			// Check multi-line attributes span exact expected lines
			for name, expectedLines := range tt.expectLineCount {
				region, ok := regions[name]
				require.True(t, ok, "region %s should exist for line count check", name)
				assert.Equal(t, expectedLines, len(region.Lines),
					"attribute %s should span exactly %d lines, got %d", name, expectedLines, len(region.Lines))
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
			// Use "\n  " prefix to match line start (immune to alignment padding, avoids comment matches)
			if tt.checkFirst != "" {
				firstIdx := strings.Index(resultStr, "\n  "+tt.checkFirst)
				require.NotEqual(t, -1, firstIdx, "%s should be in result", tt.checkFirst)

				// Check it comes before other attributes
				for _, name := range orderedNames {
					if name != tt.checkFirst {
						otherIdx := strings.Index(resultStr, "\n  "+name)
						require.NotEqual(t, -1, otherIdx, "%s should be in result", name)
						assert.Less(t, firstIdx, otherIdx,
							"%s should appear before %s", tt.checkFirst, name)
					}
				}
			}

			// Verify positional ordering if checkLast is specified
			if tt.checkLast != "" {
				lastIdx := strings.LastIndex(resultStr, "\n  "+tt.checkLast)
				require.NotEqual(t, -1, lastIdx, "%s should be in result", tt.checkLast)

				// Check it comes after other attributes
				for _, name := range orderedNames {
					if name != tt.checkLast {
						otherIdx := strings.LastIndex(resultStr, "\n  "+name)
						require.NotEqual(t, -1, otherIdx, "%s should be in result", name)
						assert.Greater(t, lastIdx, otherIdx,
							"%s should appear after %s", tt.checkLast, name)
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
		// Note: hclwrite adds alignment padding, so search for "\n  <attr>" to match line start
		forEachIdx := strings.Index(result, "\n  for_each")
		zAttrIdx := strings.Index(result, "\n  z_attr")
		aAttrIdx := strings.Index(result, "\n  a_attr")

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
		// Note: hclwrite adds alignment padding, so search for "\n  <attr>" to match line start
		tagsIdx := strings.LastIndex(result, "\n  tags")
		amiIdx := strings.LastIndex(result, "\n  ami")
		nameIdx := strings.LastIndex(result, "\n  name")

		require.NotEqual(t, -1, tagsIdx, "tags should be in result")
		require.NotEqual(t, -1, amiIdx, "ami should be in result")
		require.NotEqual(t, -1, nameIdx, "name should be in result")

		assert.Greater(t, tagsIdx, amiIdx, "tags should appear after ami")
		assert.Greater(t, tagsIdx, nameIdx, "tags should appear after name")
	})
}

func TestExtractBlockRegions(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		expectTypes       []string
		expectComments    map[int]string // block index -> expected comment substring
		expectLineCounts  map[int]int    // block index -> expected number of lines
		expectFirstLineAt map[int]int    // block index -> expected first line of region (1-indexed, comments-included)
	}{
		{
			name: "single nested block without comment",
			content: `variable "x" {
  type = string
  validation {
    condition     = length(var.x) > 0
    error_message = "required"
  }
}`,
			expectTypes:      []string{"validation"},
			expectLineCounts: map[int]int{0: 4},
		},
		{
			name: "block with leading comment",
			content: `variable "x" {
  type = string
  # validation comment
  validation {
    condition     = length(var.x) > 0
    error_message = "required"
  }
}`,
			expectTypes:    []string{"validation"},
			expectComments: map[int]string{0: "# validation comment"},
		},
		{
			name: "block sandwiched between attributes captures only its own comment",
			//nolint:dupword // HCL content intentionally contains repeated identifiers
			content: `variable "x" {
  description = "x"
  # only this comment belongs to validation
  validation {
    condition     = length(var.x) > 0
    error_message = "required"
  }
  type = string
}`,
			expectTypes: []string{"validation"},
			expectComments: map[int]string{
				0: "# only this comment belongs to validation",
			},
		},
		{
			name: "multiple validation blocks preserved in source order",
			content: `variable "x" {
  type = string
  validation {
    condition     = length(var.x) > 0
    error_message = "first"
  }
  validation {
    condition     = length(var.x) < 64
    error_message = "second"
  }
}`,
			expectTypes: []string{"validation", "validation"},
		},
		{
			name: "body with no nested blocks returns nil",
			content: `variable "x" {
  type = string
}`,
			expectTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			body := file.Body.(*hclsyntax.Body)
			require.NotEmpty(t, body.Blocks, "test content should have at least one block")

			regions := ExtractBlockRegions([]byte(tt.content), body.Blocks[0].Body)
			require.Equal(t, len(tt.expectTypes), len(regions),
				"expected %d regions, got %d", len(tt.expectTypes), len(regions))

			for i, want := range tt.expectTypes {
				assert.Equal(t, want, regions[i].Type, "region %d type", i)
			}
			for idx, want := range tt.expectComments {
				assert.Contains(t, regions[idx].LeadingComment, want,
					"region %d should have comment %q", idx, want)
			}
			for idx, want := range tt.expectLineCounts {
				assert.Equal(t, want, len(regions[idx].Lines),
					"region %d should span %d lines, got %d", idx, want, len(regions[idx].Lines))
			}
			for idx, want := range tt.expectFirstLineAt {
				assert.Equal(t, want, regions[idx].StartLine, "region %d StartLine", idx)
			}
		})
	}
}

func TestReorderBlockBodyPreservingAll(t *testing.T) {
	t.Run("attrs and nested blocks reorder to canonical layout", func(t *testing.T) {
		content := `variable "x" {
  validation {
    condition     = length(var.x) > 0
    error_message = "required"
  }
  type        = string
  description = "x"
}
`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		body := file.Body.(*hclsyntax.Body)
		require.NotEmpty(t, body.Blocks)
		block := body.Blocks[0]

		result := ReorderBlockBodyPreservingAll(
			[]byte(content),
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			[]string{"description", "type"},
			[]string{"validation"},
		)
		out := string(result)

		descIdx := strings.Index(out, `description = "x"`)
		typeIdx := strings.Index(out, `type        = string`)
		valIdx := strings.Index(out, `validation {`)
		require.NotEqual(t, -1, descIdx)
		require.NotEqual(t, -1, typeIdx)
		require.NotEqual(t, -1, valIdx)
		assert.Less(t, descIdx, typeIdx, "description before type")
		assert.Less(t, typeIdx, valIdx, "type before validation")
	})

	t.Run("unknown attrs and blocks land after prioritized ones in source order", func(t *testing.T) {
		content := `variable "x" {
  custom_block {
    field = 1
  }
  unknown_attr = "u"
  validation {
    condition     = length(var.x) > 0
    error_message = "required"
  }
  type        = string
  description = "x"
}
`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		body := file.Body.(*hclsyntax.Body)
		block := body.Blocks[0]

		result := ReorderBlockBodyPreservingAll(
			[]byte(content),
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			[]string{"description", "type"},
			[]string{"validation"},
		)
		out := string(result)

		descIdx := strings.Index(out, `description = "x"`)
		typeIdx := strings.Index(out, `type        = string`)
		valIdx := strings.Index(out, `validation {`)
		unknownIdx := strings.Index(out, `unknown_attr = "u"`)
		customIdx := strings.Index(out, `custom_block {`)

		assert.Less(t, descIdx, typeIdx, "description before type")
		assert.Less(t, typeIdx, valIdx, "type before validation")
		assert.Less(t, valIdx, unknownIdx, "validation before unknown_attr")
		assert.Less(t, unknownIdx, customIdx, "unknown_attr before custom_block")
	})

	t.Run("empty body is left unchanged", func(t *testing.T) {
		content := "variable \"x\" {}\n"
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		body := file.Body.(*hclsyntax.Body)
		block := body.Blocks[0]

		result := ReorderBlockBodyPreservingAll(
			[]byte(content),
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			[]string{"description"},
			[]string{"validation"},
		)
		assert.Equal(t, content, string(result))
	})

	t.Run("trailing orphan comment is preserved before closing brace", func(t *testing.T) {
		content := `variable "x" {
  type        = string
  description = "x"

  # forgotten note pinned to the bottom
}
`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		body := file.Body.(*hclsyntax.Body)
		block := body.Blocks[0]

		result := ReorderBlockBodyPreservingAll(
			[]byte(content),
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			[]string{"description", "type"},
			nil,
		)
		out := string(result)

		assert.Contains(t, out, `# forgotten note pinned to the bottom`,
			"orphan comment with no following region must not be dropped")

		// Orphan should appear after the last region and before the closing brace.
		descIdx := strings.Index(out, `description = "x"`)
		orphanIdx := strings.Index(out, `# forgotten note pinned to the bottom`)
		closeIdx := strings.LastIndex(out, "}")
		assert.Greater(t, orphanIdx, descIdx, "orphan should appear after all regions")
		assert.Less(t, orphanIdx, closeIdx, "orphan should appear before closing brace")
	})
}

func TestCollectOrphanLines(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		blockStart     int
		blockEnd       int
		consumed       map[int]bool
		expectedOrphan []string
	}{
		{
			name: "trailing comment with no following region",
			content: `block {
  attr = "x"

  # orphan
}`,
			blockStart:     1,
			blockEnd:       5,
			consumed:       map[int]bool{2: true},
			expectedOrphan: []string{"  # orphan"},
		},
		{
			name: "no orphans when everything is consumed",
			content: `block {
  a = 1
  b = 2
}`,
			blockStart:     1,
			blockEnd:       4,
			consumed:       map[int]bool{2: true, 3: true},
			expectedOrphan: nil,
		},
		{
			name: "blank lines are dropped, not treated as orphans",
			content: `block {
  a = 1


}`,
			blockStart:     1,
			blockEnd:       5,
			consumed:       map[int]bool{2: true},
			expectedOrphan: nil,
		},
		{
			// Helper-level contract test. In real pipelines mid-block orphans do not
			// arise because collectLeadingComments would claim line 3 as line 4's leading
			// comment and the caller would mark line 3 consumed. We construct a `consumed`
			// map that omits line 3 to verify the helper's contract: any non-consumed,
			// non-blank line is returned regardless of position.
			name: "non-consumed interior line is returned (helper contract)",
			content: `block {
  a = 1
  # interior line
  b = 2
}`,
			blockStart:     1,
			blockEnd:       5,
			consumed:       map[int]bool{2: true, 4: true},
			expectedOrphan: []string{"  # interior line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := SplitLines([]byte(tt.content))
			got := collectOrphanLines(lines, tt.blockStart, tt.blockEnd, tt.consumed)
			assert.Equal(t, tt.expectedOrphan, got)
		})
	}
}

func TestReorderTopLevelBlocksByLineRange(t *testing.T) {
	t.Run("canonical priority order", func(t *testing.T) {
		input := `output "x" { value = "x" }
module "m" { source = "./m" }
resource "r" "x" { x = 1 }
data "d" "x" { x = 1 }
locals { x = 1 }
variable "v" { default = "x" }
provider "p" { region = "x" }
terraform { required_version = ">= 1.0" }
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)

		assertOrderedSubstrings(t, string(out), []string{
			"terraform {",
			"provider ",
			"variable ",
			"locals {",
			"data ",
			"resource ",
			"module ",
			"output ",
		})
	})

	t.Run("preserves block-internal content byte-for-byte", func(t *testing.T) {
		input := `resource "aws_instance" "x" {
  # important comment
  ami           = "ami-123"
  instance_type = "t3.medium" # trailing
  tags = {
    Name = "test"
  }
}

terraform {
  required_version = ">= 1.0"
}
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)
		outStr := string(out)

		// Resource body must be byte-for-byte preserved; the buggy old helper would
		// reshuffle attrs via map iteration.
		expectedResourceBody := `resource "aws_instance" "x" {
  # important comment
  ami           = "ami-123"
  instance_type = "t3.medium" # trailing
  tags = {
    Name = "test"
  }
}`
		assert.Contains(t, outStr, expectedResourceBody)
	})

	t.Run("file header preserved before reordered blocks", func(t *testing.T) {
		input := `# Copyright notice
# License terms

resource "x" "x" { x = 1 }

terraform { required_version = ">= 1.0" }
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)
		outStr := string(out)

		assertOrderedSubstrings(t, outStr, []string{
			"# Copyright notice",
			"# License terms",
			"terraform {",
			"resource ",
		})
	})

	t.Run("file footer preserved after reordered blocks", func(t *testing.T) {
		input := `resource "x" "x" { x = 1 }

terraform { required_version = ">= 1.0" }

# trailing footer note
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)
		outStr := string(out)

		// Footer must appear after the last reordered block.
		footerIdx := strings.Index(outStr, "# trailing footer note")
		resourceIdx := strings.Index(outStr, "resource ")
		require.NotEqual(t, -1, footerIdx, "footer comment must survive")
		require.NotEqual(t, -1, resourceIdx)
		assert.Greater(t, footerIdx, resourceIdx, "footer must appear after last block")
	})

	t.Run("unknown block types sort to bottom in source order", func(t *testing.T) {
		input := `custom_block "x" { x = 1 }
terraform { required_version = ">= 1.0" }
another_custom "y" { y = 2 }
resource "r" "x" { x = 1 }
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)
		assertOrderedSubstrings(t, string(out), []string{
			"terraform {",
			`resource `,
			`custom_block `,
			`another_custom `,
		})
	})

	t.Run("empty file is returned unchanged", func(t *testing.T) {
		input := []byte("")
		out, err := ReorderTopLevelBlocksByLineRange(input)
		require.NoError(t, err)
		assert.Equal(t, input, out)
	})

	t.Run("comment directly above first block travels with that block", func(t *testing.T) {
		// No blank line between the comment and the resource means the comment is the
		// resource's adjacent leading comment, not file-level header content. After
		// reorder, the comment must move with the resource.
		input := `# directly attached comment
resource "x" "x" { x = 1 }

terraform { required_version = ">= 1.0" }
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)
		outStr := string(out)

		// terraform is now first; the directly attached comment must precede resource
		// (not appear at the top of the file as if it were a file header).
		assertOrderedSubstrings(t, outStr, []string{
			"terraform {",
			"# directly attached comment",
			"resource ",
		})
		// The directly attached comment must not appear before terraform.
		tfIdx := strings.Index(outStr, "terraform {")
		commentIdx := strings.Index(outStr, "# directly attached comment")
		assert.Greater(t, commentIdx, tfIdx, "adjacent comment must travel with its block, not stay at file top")
	})

	t.Run("multi-blank-line gaps between blocks are normalised to single blank", func(t *testing.T) {
		// Documents (does not enforce as new) behavior: the reorder emits exactly one
		// blank line between reordered blocks. The old hclwrite-based reorder did the
		// same through FormatAndCleanBlankLines. Extra blank lines between source blocks
		// are collapsed; this is an intentional formatting normalization.
		input := "terraform { required_version = \">= 1.0\" }\n\n\n\nresource \"x\" \"x\" { x = 1 }\n"
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)
		outStr := string(out)

		// Count blank-line runs between terraform and resource — there must be exactly one.
		tfEnd := strings.Index(outStr, "}\n")
		require.NotEqual(t, -1, tfEnd)
		between := outStr[tfEnd+2:]
		resIdx := strings.Index(between, "resource ")
		require.NotEqual(t, -1, resIdx)
		gap := between[:resIdx]
		assert.Equal(t, "\n", gap, "expected exactly one blank line between blocks, got %q", gap)
	})

	t.Run("invalid HCL returns an error", func(t *testing.T) {
		input := []byte("this is not valid hcl {{{\n")
		_, err := ReorderTopLevelBlocksByLineRange(input)
		assert.Error(t, err)
	})

	t.Run("stable order within same block type", func(t *testing.T) {
		input := `variable "b" { default = "b" }
variable "a" { default = "a" }
terraform { required_version = ">= 1.0" }
`
		out, err := ReorderTopLevelBlocksByLineRange([]byte(input))
		require.NoError(t, err)

		// terraform must come first; variables preserve their source order (b then a).
		assertOrderedSubstrings(t, string(out), []string{
			"terraform {",
			`variable "b"`,
			`variable "a"`,
		})
	})
}
