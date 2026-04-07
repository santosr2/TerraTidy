package annotations

import (
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_NextBlock(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Suppression
	}{
		{
			name: "simple next block suppression",
			content: `# terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 2,
					Type:       NextBlock,
				},
			},
		},
		{
			name: "next block with blank lines",
			content: `# terratidy:ignore:style.block-label-case

resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 3,
					Type:       NextBlock,
				},
			},
		},
		{
			name: "double-slash comment style",
			content: `// terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 2,
					Type:       NextBlock,
				},
			},
		},
		{
			name: "with leading whitespace",
			content: `  # terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 2,
					Type:       NextBlock,
				},
			},
		},
		{
			name: "skips comment-only lines",
			content: `# terratidy:ignore:style.block-label-case
# This is another comment
resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 3,
					Type:       NextBlock,
				},
			},
		},
		{
			name: "lint rule suppression",
			content: `# terratidy:ignore:lint.deprecated-resource
resource "aws_instance" "example" { }`,
			expected: []Suppression{
				{
					Rule:       "lint.deprecated-resource",
					Line:       1,
					TargetLine: 2,
					Type:       NextBlock,
				},
			},
		},
		{
			name: "policy rule suppression",
			content: `# terratidy:ignore:policy.require-tags
resource "aws_instance" "example" { }`,
			expected: []Suppression{
				{
					Rule:       "policy.require-tags",
					Line:       1,
					TargetLine: 2,
					Type:       NextBlock,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suppressions := Parse([]byte(tt.content))

			require.Len(t, suppressions, len(tt.expected))
			for i, s := range suppressions {
				assert.Equal(t, tt.expected[i].Rule, s.Rule, "rule mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].Line, s.Line, "line mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].TargetLine, s.TargetLine, "targetLine mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].Type, s.Type, "type mismatch at index %d", i)
			}
		})
	}
}

func TestParse_Inline(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Suppression
	}{
		{
			name:    "inline suppression with code before comment",
			content: `resource "aws_instance" "MyServer" { } # terratidy:ignore:style.block-label-case`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 1,
					Type:       Inline,
				},
			},
		},
		{
			name:    "inline suppression with double-slash",
			content: `resource "aws_instance" "MyServer" { } // terratidy:ignore:style.block-label-case`,
			expected: []Suppression{
				{
					Rule:       "style.block-label-case",
					Line:       1,
					TargetLine: 1,
					Type:       Inline,
				},
			},
		},
		{
			name:    "inline lint suppression",
			content: `resource "aws_instance" "example" { } # terratidy:ignore:lint.deprecated-resource`,
			expected: []Suppression{
				{
					Rule:       "lint.deprecated-resource",
					Line:       1,
					TargetLine: 1,
					Type:       Inline,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suppressions := Parse([]byte(tt.content))

			require.Len(t, suppressions, len(tt.expected))
			for i, s := range suppressions {
				assert.Equal(t, tt.expected[i].Rule, s.Rule, "rule mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].Line, s.Line, "line mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].TargetLine, s.TargetLine, "targetLine mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].Type, s.Type, "type mismatch at index %d", i)
			}
		})
	}
}

func TestParse_FileLevel(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Suppression
	}{
		{
			name: "file level suppression at top",
			content: `# terratidy:ignore-file:style.block-label-case
resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule: "style.block-label-case",
					Line: 1,
					Type: File,
				},
			},
		},
		{
			name: "file level suppression anywhere in file",
			content: `resource "aws_instance" "first" { }
# terratidy:ignore-file:style.block-label-case
resource "aws_instance" "MyServer" { }`,
			expected: []Suppression{
				{
					Rule: "style.block-label-case",
					Line: 2,
					Type: File,
				},
			},
		},
		{
			name: "file level lint suppression",
			content: `# terratidy:ignore-file:lint.deprecated-resource
resource "aws_instance" "example" { }`,
			expected: []Suppression{
				{
					Rule: "lint.deprecated-resource",
					Line: 1,
					Type: File,
				},
			},
		},
		{
			name: "file level policy suppression",
			content: `# terratidy:ignore-file:policy.require-tags
resource "aws_instance" "example" { }`,
			expected: []Suppression{
				{
					Rule: "policy.require-tags",
					Line: 1,
					Type: File,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suppressions := Parse([]byte(tt.content))

			require.Len(t, suppressions, len(tt.expected))
			for i, s := range suppressions {
				assert.Equal(t, tt.expected[i].Rule, s.Rule, "rule mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].Line, s.Line, "line mismatch at index %d", i)
				assert.Equal(t, tt.expected[i].Type, s.Type, "type mismatch at index %d", i)
			}
		})
	}
}

func TestParse_Multiple(t *testing.T) {
	content := `# terratidy:ignore-file:style.variable-naming
# terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }

resource "aws_s3_bucket" "Test" { } # terratidy:ignore:style.block-label-case`

	suppressions := Parse([]byte(content))

	require.Len(t, suppressions, 3)

	// File-level suppression
	assert.Equal(t, "style.variable-naming", suppressions[0].Rule)
	assert.Equal(t, File, suppressions[0].Type)

	// Next-block suppression
	assert.Equal(t, "style.block-label-case", suppressions[1].Rule)
	assert.Equal(t, NextBlock, suppressions[1].Type)
	assert.Equal(t, 3, suppressions[1].TargetLine)

	// Inline suppression
	assert.Equal(t, "style.block-label-case", suppressions[2].Rule)
	assert.Equal(t, Inline, suppressions[2].Type)
	assert.Equal(t, 5, suppressions[2].TargetLine)
}

func TestParse_MixedEngines(t *testing.T) {
	content := `# terratidy:ignore-file:style.*
# terratidy:ignore:lint.deprecated-resource
resource "aws_instance" "example" { } # terratidy:ignore:policy.require-tags`

	suppressions := Parse([]byte(content))

	require.Len(t, suppressions, 3)
	assert.Equal(t, "style.*", suppressions[0].Rule)
	assert.Equal(t, "lint.deprecated-resource", suppressions[1].Rule)
	assert.Equal(t, "policy.require-tags", suppressions[2].Rule)
}

func TestRuleMatches(t *testing.T) {
	tests := []struct {
		findingRule     string
		suppressionRule string
		expected        bool
	}{
		// Exact matches
		{"style.block-label-case", "style.block-label-case", true},
		{"style.block-label-case", "style.variable-naming", false},
		{"lint.some-rule", "style.some-rule", false},
		{"policy.require-tags", "policy.require-tags", true},

		// Wildcard matches
		{"style.block-label-case", "style.*", true},
		{"style.variable-naming", "style.*", true},
		{"lint.some-rule", "style.*", false},
		{"lint.some-rule", "lint.*", true},
		{"policy.require-tags", "policy.*", true},
		{"policy.cost-limit", "policy.*", true},

		// Edge cases
		{"style.block-label-case", "style", false},
		{"style", "style.*", false}, // "style" doesn't have a dot after prefix
	}

	for _, tt := range tests {
		t.Run(tt.findingRule+"_"+tt.suppressionRule, func(t *testing.T) {
			result := RuleMatches(tt.findingRule, tt.suppressionRule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSuppressed(t *testing.T) {
	tests := []struct {
		name         string
		finding      sdk.Finding
		suppressions []Suppression
		expected     bool
	}{
		{
			name: "file suppression matches",
			finding: sdk.Finding{
				Rule:     "style.block-label-case",
				Location: sdk.Location{StartLine: 10},
			},
			suppressions: []Suppression{
				{Rule: "style.block-label-case", Type: File},
			},
			expected: true,
		},
		{
			name: "file suppression with wildcard",
			finding: sdk.Finding{
				Rule:     "style.block-label-case",
				Location: sdk.Location{StartLine: 10},
			},
			suppressions: []Suppression{
				{Rule: "style.*", Type: File},
			},
			expected: true,
		},
		{
			name: "next-block suppression matches line",
			finding: sdk.Finding{
				Rule:     "style.block-label-case",
				Location: sdk.Location{StartLine: 5},
			},
			suppressions: []Suppression{
				{Rule: "style.block-label-case", TargetLine: 5, Type: NextBlock},
			},
			expected: true,
		},
		{
			name: "next-block suppression wrong line",
			finding: sdk.Finding{
				Rule:     "style.block-label-case",
				Location: sdk.Location{StartLine: 10},
			},
			suppressions: []Suppression{
				{Rule: "style.block-label-case", TargetLine: 5, Type: NextBlock},
			},
			expected: false,
		},
		{
			name: "inline suppression matches line",
			finding: sdk.Finding{
				Rule:     "style.block-label-case",
				Location: sdk.Location{StartLine: 3},
			},
			suppressions: []Suppression{
				{Rule: "style.block-label-case", TargetLine: 3, Type: Inline},
			},
			expected: true,
		},
		{
			name: "no suppression matches",
			finding: sdk.Finding{
				Rule:     "style.block-label-case",
				Location: sdk.Location{StartLine: 5},
			},
			suppressions: []Suppression{
				{Rule: "style.variable-naming", TargetLine: 5, Type: NextBlock},
			},
			expected: false,
		},
		{
			name:         "empty suppressions",
			finding:      sdk.Finding{Rule: "style.block-label-case"},
			suppressions: nil,
			expected:     false,
		},
		{
			name: "lint rule suppression",
			finding: sdk.Finding{
				Rule:     "lint.deprecated-resource",
				Location: sdk.Location{StartLine: 5},
			},
			suppressions: []Suppression{
				{Rule: "lint.*", Type: File},
			},
			expected: true,
		},
		{
			name: "policy rule suppression",
			finding: sdk.Finding{
				Rule:     "policy.require-tags",
				Location: sdk.Location{StartLine: 5},
			},
			suppressions: []Suppression{
				{Rule: "policy.require-tags", TargetLine: 5, Type: NextBlock},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSuppressed(tt.finding, tt.suppressions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterFindings(t *testing.T) {
	findings := []sdk.Finding{
		{Rule: "style.block-label-case", Location: sdk.Location{StartLine: 2}},
		{Rule: "style.variable-naming", Location: sdk.Location{StartLine: 5}},
		{Rule: "style.block-label-case", Location: sdk.Location{StartLine: 10}},
		{Rule: "lint.deprecated-resource", Location: sdk.Location{StartLine: 15}},
		{Rule: "policy.require-tags", Location: sdk.Location{StartLine: 20}},
	}

	t.Run("file-level suppression removes all matching", func(t *testing.T) {
		suppressions := []Suppression{
			{Rule: "style.block-label-case", Type: File},
		}
		filtered := FilterFindings(findings, suppressions)
		require.Len(t, filtered, 3)
		assert.Equal(t, "style.variable-naming", filtered[0].Rule)
		assert.Equal(t, "lint.deprecated-resource", filtered[1].Rule)
		assert.Equal(t, "policy.require-tags", filtered[2].Rule)
	})

	t.Run("line-specific suppression removes only that line", func(t *testing.T) {
		suppressions := []Suppression{
			{Rule: "style.block-label-case", TargetLine: 2, Type: NextBlock},
		}
		filtered := FilterFindings(findings, suppressions)
		require.Len(t, filtered, 4)
		assert.Equal(t, "style.variable-naming", filtered[0].Rule)
		assert.Equal(t, "style.block-label-case", filtered[1].Rule)
		assert.Equal(t, 10, filtered[1].Location.StartLine)
	})

	t.Run("wildcard suppresses all rules for engine", func(t *testing.T) {
		suppressions := []Suppression{
			{Rule: "style.*", Type: File},
		}
		filtered := FilterFindings(findings, suppressions)
		require.Len(t, filtered, 2)
		assert.Equal(t, "lint.deprecated-resource", filtered[0].Rule)
		assert.Equal(t, "policy.require-tags", filtered[1].Rule)
	})

	t.Run("empty suppressions returns all findings", func(t *testing.T) {
		filtered := FilterFindings(findings, nil)
		require.Len(t, filtered, 5)
	})

	t.Run("multiple engine suppressions", func(t *testing.T) {
		suppressions := []Suppression{
			{Rule: "style.*", Type: File},
			{Rule: "lint.*", Type: File},
		}
		filtered := FilterFindings(findings, suppressions)
		require.Len(t, filtered, 1)
		assert.Equal(t, "policy.require-tags", filtered[0].Rule)
	})
}
