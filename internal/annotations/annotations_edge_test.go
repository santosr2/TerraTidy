package annotations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse_MultipleRulesOneLine documents current behavior: comma-separated
// rules like "rule1,rule2" are captured as a single token. Suppression only works
// if the finding rule exactly matches "rule1,rule2" (which never happens).
func TestParse_MultipleRulesOneLine(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantRule     string
		wantSuppress bool // Does it suppress "style.rule1"?
	}{
		{
			name:         "comma separated rules captured as single token",
			content:      "# terratidy:ignore:style.rule1,style.rule2\nresource \"aws_instance\" \"test\" {}",
			wantRule:     "style.rule1,style.rule2", // Entire string is captured
			wantSuppress: false,                     // Does NOT suppress "style.rule1"
		},
		{
			name:         "single rule works normally",
			content:      "# terratidy:ignore:style.rule1\nresource \"aws_instance\" \"test\" {}",
			wantRule:     "style.rule1",
			wantSuppress: true, // DOES suppress "style.rule1"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suppressions := Parse([]byte(tt.content))

			require.Len(t, suppressions, 1)
			assert.Equal(t, tt.wantRule, suppressions[0].Rule, "rule should be captured as-is")

			// Verify suppression behavior against individual rule "style.rule1"
			matches := RuleMatches("style.rule1", suppressions[0].Rule)
			assert.Equal(t, tt.wantSuppress, matches,
				"suppression match result for 'style.rule1'")
		})
	}
}

// TestParse_CaseInsensitivity documents current behavior: annotations are
// case-sensitive. TERRATIDY:IGNORE and Terratidy:Ignore do NOT work.
func TestParse_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name:    "lowercase works",
			content: "# terratidy:ignore:style.rule\nresource \"aws_instance\" \"test\" {}",
			wantLen: 1,
		},
		{
			name:    "UPPERCASE does not match",
			content: "# TERRATIDY:IGNORE:style.rule\nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "mixed case Terratidy does not match",
			content: "# Terratidy:Ignore:style.rule\nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "uppercase IGNORE only does not match",
			content: "# terratidy:IGNORE:style.rule\nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "ignore-file uppercase does not match",
			content: "# TERRATIDY:IGNORE-FILE:style.rule\nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suppressions := Parse([]byte(tt.content))
			assert.Len(t, suppressions, tt.wantLen,
				"annotations are case-sensitive; only lowercase 'terratidy:ignore' is recognized")
		})
	}
}

// TestParse_InvalidSyntax documents behavior for malformed annotations.
// Empty rules (nothing after the colon) are not parsed as suppressions.
func TestParse_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name:    "empty rule after colon",
			content: "# terratidy:ignore:\nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "empty rule with trailing space",
			content: "# terratidy:ignore: \nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "ignore-file empty rule",
			content: "# terratidy:ignore-file:\nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "only whitespace after colon",
			content: "# terratidy:ignore:   \nresource \"aws_instance\" \"test\" {}",
			wantLen: 0,
		},
		{
			name:    "valid rule for comparison",
			content: "# terratidy:ignore:style.rule\nresource \"aws_instance\" \"test\" {}",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suppressions := Parse([]byte(tt.content))
			assert.Len(t, suppressions, tt.wantLen,
				"empty rules should not be parsed as suppressions")
		})
	}
}
