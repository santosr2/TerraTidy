package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/TerraTidy/pkg/sdk"
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
				result := getOrderedAttrNames(body.Blocks[0].Body)
				assert.Equal(t, tt.expected, result)
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
			result := formatAndCleanBlankLines([]byte(tt.input))
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
			result := splitLines([]byte(tt.input))
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
			result := trimLeftWhitespace(tt.input)
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
			result := countBlankLinesBetween(tt.lines, tt.endLine, tt.startLine)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasCommentBetween(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		endLine   int
		startLine int
		expected  bool
	}{
		{
			name:      "hash comment between",
			lines:     []string{"line1", "# comment", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  true,
		},
		{
			name:      "double-slash comment between",
			lines:     []string{"line1", "// comment", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  true,
		},
		{
			name:      "indented hash comment still counts",
			lines:     []string{"line1", "    # indented", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  true,
		},
		{
			name:      "indented double-slash comment still counts",
			lines:     []string{"line1", "    // indented", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  true,
		},
		{
			name:      "no comment between",
			lines:     []string{"line1", "line2", "line3"},
			endLine:   1,
			startLine: 3,
			expected:  false,
		},
		{
			name:      "blank lines only",
			lines:     []string{"line1", "", "  ", "line4"},
			endLine:   1,
			startLine: 4,
			expected:  false,
		},
		{
			name:      "comment present but outside scan window",
			lines:     []string{"# header", "line2", "line3", "# footer"},
			endLine:   2,
			startLine: 3,
			expected:  false,
		},
		{
			name:      "comment at startLine (exclusive boundary) is ignored",
			lines:     []string{"line1", "line2", "# at start", "line4"},
			endLine:   1,
			startLine: 3,
			expected:  false,
		},
		{
			name:      "comment at endLine (exclusive boundary) is ignored",
			lines:     []string{"# at end", "line2", "line3", "line4"},
			endLine:   1,
			startLine: 4,
			expected:  false,
		},
		{
			name:      "first comment wins among many",
			lines:     []string{"line1", "// first", "# second", "line4"},
			endLine:   1,
			startLine: 4,
			expected:  true,
		},
		{
			name:      "adjacent lines have no interior",
			lines:     []string{"line1", "line2"},
			endLine:   1,
			startLine: 2,
			expected:  false,
		},
		{
			name:      "out-of-range startLine does not panic",
			lines:     []string{"line1", "// comment"},
			endLine:   1,
			startLine: 10,
			expected:  true,
		},
		{
			name:      "empty lines slice returns false",
			lines:     []string{},
			endLine:   1,
			startLine: 5,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCommentBetween(tt.lines, tt.endLine, tt.startLine)
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
			result := blockKey(tt.blockType, tt.labels)
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
		attr := findAttribute(attrs, "ami")
		assert.NotNil(t, attr)
	})

	t.Run("returns nil for missing attribute", func(t *testing.T) {
		attr := findAttribute(attrs, "nonexistent")
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
		block := findNestedBlock(blocks, "lifecycle")
		assert.NotNil(t, block)
	})

	t.Run("returns nil for missing block", func(t *testing.T) {
		block := findNestedBlock(blocks, "provisioner")
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
			result := isSnakeCase(tt.input)
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
			result := isCamelCase(tt.input)
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
			result := isKebabCase(tt.input)
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
			result := isPascalCase(tt.input)
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
			result := matchesCustomPattern(tt.input, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNaming(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		convention    namingCase
		customPattern string
		expectValid   bool
		expectCase    string
	}{
		{"snake_case valid", "my_var", snakeCase, "", true, "snake_case"},
		{"snake_case invalid", "myVar", snakeCase, "", false, "snake_case"},
		{"camelCase valid", "myVar", camelCase, "", true, "camelCase"},
		{"camelCase invalid", "my_var", camelCase, "", false, "camelCase"},
		{"kebab-case valid", "my-var", kebabCase, "", true, "kebab-case"},
		{"kebab-case invalid", "my_var", kebabCase, "", false, "kebab-case"},
		{"PascalCase valid", "MyVar", pascalCase, "", true, "PascalCase"},
		{"PascalCase invalid", "myVar", pascalCase, "", false, "PascalCase"},
		{"custom pattern valid", "prefix_name", customCase, "^prefix_", true, "custom pattern"},
		{"custom pattern invalid", "name", customCase, "^prefix_", false, "custom pattern"},
		{"custom empty pattern", "anything", customCase, "", true, "custom"},
		{"default to snake_case", "my_var", "", "", true, "snake_case"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, caseName := validateNaming(tt.input, tt.convention, tt.customPattern)
			assert.Equal(t, tt.expectValid, valid)
			assert.Equal(t, tt.expectCase, caseName)
		})
	}
}

func TestGetNamingConventionFromConfig(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]any
		expectConvention namingCase
		expectPattern    string
	}{
		{
			name:             "nil config returns snake_case",
			config:           nil,
			expectConvention: snakeCase,
			expectPattern:    "",
		},
		{
			name:             "empty config returns snake_case",
			config:           map[string]any{},
			expectConvention: snakeCase,
			expectPattern:    "",
		},
		{
			name: "snake_case from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "snake_case",
				},
			},
			expectConvention: snakeCase,
			expectPattern:    "",
		},
		{
			name: "camelCase from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "camelCase",
				},
			},
			expectConvention: camelCase,
			expectPattern:    "",
		},
		{
			name: "kebab-case from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "kebab-case",
				},
			},
			expectConvention: kebabCase,
			expectPattern:    "",
		},
		{
			name: "PascalCase from config",
			config: map[string]any{
				"options": map[string]any{
					"case": "PascalCase",
				},
			},
			expectConvention: pascalCase,
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
			expectConvention: customCase,
			expectPattern:    "^prefix_",
		},
		{
			name: "unknown case defaults to snake_case",
			config: map[string]any{
				"options": map[string]any{
					"case": "UNKNOWN",
				},
			},
			expectConvention: snakeCase,
			expectPattern:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convention, pattern := getNamingConventionFromConfig(tt.config)
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
			result := getAttributeOrderFromConfig(tt.config, defaultOrder)
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
			result := matchBlockLabels(tt.labels, tt.expected)
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
		result := findSyntaxBody(body, "resource", []string{"aws_instance", "example"})
		assert.NotNil(t, result)
	})

	t.Run("returns nil for non-matching block", func(t *testing.T) {
		result := findSyntaxBody(body, "resource", []string{"aws_instance", "nonexistent"})
		assert.Nil(t, result)
	})

	t.Run("returns nil for wrong block type", func(t *testing.T) {
		result := findSyntaxBody(body, "data", []string{"aws_instance", "example"})
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
		result := findWriteBlock(file, "resource", []string{"aws_instance", "example"})
		assert.NotNil(t, result)
	})

	t.Run("returns nil for non-matching block", func(t *testing.T) {
		result := findWriteBlock(file, "resource", []string{"aws_instance", "nonexistent"})
		assert.Nil(t, result)
	})
}

func TestParseBothFormats(t *testing.T) {
	content := `resource "aws_instance" "example" {
  ami = "ami-123"
}`

	t.Run("parses valid HCL", func(t *testing.T) {
		syntaxBody, writeFile, err := parseBothFormats([]byte(content), "test.tf")
		assert.NoError(t, err)
		assert.NotNil(t, syntaxBody)
		assert.NotNil(t, writeFile)
	})

	t.Run("returns error for invalid HCL", func(t *testing.T) {
		_, _, err := parseBothFormats([]byte("invalid { content"), "test.tf")
		assert.Error(t, err)
	})
}

func TestWholeFileEdit(t *testing.T) {
	t.Run("no-op when content is identical", func(t *testing.T) {
		original := []byte("resource \"x\" \"y\" {}\n")
		result := WholeFileEdit(original, original)
		assert.Nil(t, result, "byte-equal inputs must produce a nil result (no-op)")
	})

	t.Run("no-op when both are nil", func(t *testing.T) {
		assert.Nil(t, WholeFileEdit(nil, nil))
	})

	t.Run("no-op when both are empty but different identities", func(t *testing.T) {
		assert.Nil(t, WholeFileEdit([]byte{}, []byte{}))
		assert.Nil(t, WholeFileEdit(nil, []byte{}))
		assert.Nil(t, WholeFileEdit([]byte{}, nil))
	})

	t.Run("single covering edit when content changed", func(t *testing.T) {
		original := []byte("foo")
		newContent := []byte("bar baz")

		result := WholeFileEdit(original, newContent)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		edit := result.Edits[0]
		assert.Equal(t, 0, edit.Start, "Start must be 0 for whole-file edit")
		assert.Equal(t, len(original), edit.End, "End must equal len(original) for whole-file edit")
		assert.Equal(t, newContent, edit.Replacement, "Replacement must be the new content")
	})

	t.Run("pure insertion when original is nil", func(t *testing.T) {
		newContent := []byte("inserted")

		result := WholeFileEdit(nil, newContent)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		edit := result.Edits[0]
		assert.Equal(t, 0, edit.Start)
		assert.Equal(t, 0, edit.End, "End must be 0 when original is nil (pure insertion)")
		assert.Equal(t, newContent, edit.Replacement)
	})

	t.Run("pure deletion when new content is nil", func(t *testing.T) {
		original := []byte("delete me")

		result := WholeFileEdit(original, nil)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		edit := result.Edits[0]
		assert.Equal(t, 0, edit.Start)
		assert.Equal(t, len(original), edit.End)
		assert.Empty(t, edit.Replacement, "nil replacement signals pure deletion")
	})

	t.Run("returned shape matches sdk.FixResult contract", func(t *testing.T) {
		original := []byte("a")
		newContent := []byte("b")

		got := WholeFileEdit(original, newContent)
		require.NotNil(t, got)
		assert.IsType(t, &sdk.FixResult{}, got)
		assert.IsType(t, []sdk.TextEdit{}, got.Edits)
	})
}
