package output

import (
	"bytes"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func TestSARIFFormatter_WithFixableFindings(t *testing.T) {
	findings := []sdk.Finding{
		{
			Rule:     "test.fixable",
			Message:  "Can be fixed",
			File:     "main.tf",
			Severity: sdk.SeverityWarning,
			Fixable:  true,
			Location: hcl.Range{
				Start: hcl.Pos{Line: 5, Column: 1},
				End:   hcl.Pos{Line: 5, Column: 20},
			},
		},
	}

	formatter := &SARIFFormatter{Version: "1.0.0"}
	var buf bytes.Buffer
	err := formatter.Format(findings, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test.fixable")
	assert.Contains(t, output, "Can be fixed")
}

func TestSarifLevel_AllSeverities(t *testing.T) {
	tests := []struct {
		severity sdk.Severity
		want     string
	}{
		{sdk.SeverityError, "error"},
		{sdk.SeverityWarning, "warning"},
		{sdk.SeverityInfo, "note"},
		{sdk.Severity("unknown"), "warning"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.want, sarifLevel(tt.severity))
		})
	}
}

func TestSarifLine_ZeroAndNonZero(t *testing.T) {
	assert.Equal(t, 1, sarifLine(0))  // 0 becomes 1
	assert.Equal(t, 5, sarifLine(5))  // non-zero stays
	assert.Equal(t, 1, sarifLine(-1)) // negative becomes 1
}

func TestSarifColumn_ZeroAndNonZero(t *testing.T) {
	assert.Equal(t, 1, sarifColumn(0))  // 0 becomes 1
	assert.Equal(t, 3, sarifColumn(3))  // non-zero stays
	assert.Equal(t, 1, sarifColumn(-1)) // negative becomes 1
}

func TestMarkdownFormatter_AllSeverities(t *testing.T) {
	findings := []sdk.Finding{
		{Rule: "r1", Message: "error", File: "a.tf", Severity: sdk.SeverityError},
		{Rule: "r2", Message: "warn", File: "a.tf", Severity: sdk.SeverityWarning},
		{Rule: "r3", Message: "info", File: "a.tf", Severity: sdk.SeverityInfo},
	}

	formatter := &MarkdownFormatter{}
	var buf bytes.Buffer
	err := formatter.Format(findings, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "warn")
	assert.Contains(t, output, "info")
}

func TestGitHubActionsFormatter_AllSeverities(t *testing.T) {
	findings := []sdk.Finding{
		{
			Rule: "r1", Message: "error msg", File: "a.tf", Severity: sdk.SeverityError,
			Location: hcl.Range{Start: hcl.Pos{Line: 1, Column: 1}},
		},
		{
			Rule: "r2", Message: "warn msg", File: "a.tf", Severity: sdk.SeverityWarning,
			Location: hcl.Range{Start: hcl.Pos{Line: 2, Column: 1}},
		},
		{
			Rule: "r3", Message: "info msg", File: "a.tf", Severity: sdk.SeverityInfo,
			Location: hcl.Range{Start: hcl.Pos{Line: 3, Column: 1}},
		},
	}

	formatter := &GitHubActionsFormatter{}
	var buf bytes.Buffer
	err := formatter.Format(findings, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "::warning")
	assert.Contains(t, output, "::notice")
}
