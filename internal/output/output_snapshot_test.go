package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snapshotFindings returns a consistent set of findings for snapshot tests.
// This set exercises all severity levels, multiple files, fixes, and edge cases.
func snapshotFindings() []sdk.Finding {
	return []sdk.Finding{
		{
			Rule:     "style.resource-naming",
			Message:  "Resource name does not follow naming convention",
			File:     "modules/network/main.tf",
			Severity: sdk.SeverityError,
			Location: sdk.Location{
				StartLine:   15,
				StartColumn: 1,
				EndLine:     15,
				EndColumn:   42,
			},
		},
		{
			Rule:     "style.blank-lines",
			Message:  "Missing blank line between blocks",
			File:     "modules/network/main.tf",
			Severity: sdk.SeverityWarning,
			Location: sdk.Location{
				StartLine:   25,
				StartColumn: 1,
				EndLine:     25,
				EndColumn:   1,
			},
			Fix: &sdk.FixResult{Content: []byte("\n")},
		},
		{
			Rule:     "lint.deprecated-attribute",
			Message:  "Attribute 'instance_type' is deprecated, use 'size' instead",
			File:     "modules/compute/instances.tf",
			Severity: sdk.SeverityWarning,
			Location: sdk.Location{
				StartLine:   8,
				StartColumn: 3,
				EndLine:     8,
				EndColumn:   28,
			},
		},
		{
			Rule:     "policy.required-tags",
			Message:  "Resource is missing required tag: Environment",
			File:     "main.tf",
			Severity: sdk.SeverityError,
			Location: sdk.Location{
				StartLine:   1,
				StartColumn: 1,
				EndLine:     10,
				EndColumn:   2,
			},
		},
		{
			Rule:     "style.description-present",
			Message:  "Variable should have a description",
			File:     "variables.tf",
			Severity: sdk.SeverityInfo,
			Location: sdk.Location{
				StartLine:   5,
				StartColumn: 1,
				EndLine:     7,
				EndColumn:   2,
			},
			Fix: &sdk.FixResult{Content: []byte(`variable "name" {
  description = "TODO: add description"
  type        = string
}`)},
		},
	}
}

// updateGolden checks if UPDATE_GOLDEN env var is set to enable golden file updates.
func updateGolden() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

// goldenPath returns the path to a golden file in testdata/.
func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden")
}

// normalizeLineEndings converts \r\n to \n for cross-platform comparison.
func normalizeLineEndings(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// assertGolden compares actual output with golden file, optionally updating it.
// Line endings are normalized to \n for cross-platform compatibility.
func assertGolden(t *testing.T, name string, actual []byte) {
	t.Helper()

	path := goldenPath(name)

	if updateGolden() {
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		require.NoError(t, err, "failed to create golden file directory")
		err = os.WriteFile(path, actual, 0o644)
		require.NoError(t, err, "failed to update golden file %s", path)
		t.Logf("Updated golden file: %s", path)
		return
	}

	expected, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read golden file %s (run with UPDATE_GOLDEN=1 to create)", path)

	// Normalize line endings for cross-platform comparison (Windows uses \r\n, Unix uses \n)
	expectedNorm := normalizeLineEndings(expected)
	actualNorm := normalizeLineEndings(actual)

	assert.Equal(t, string(expectedNorm), string(actualNorm),
		"output does not match golden file %s (run with UPDATE_GOLDEN=1 to update)", path)
}

// TestOutput_TextSnapshot verifies text formatter output matches golden file.
func TestOutput_TextSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		golden   string
	}{
		{
			name:     "empty findings",
			findings: []sdk.Finding{},
			golden:   "text_empty",
		},
		{
			name:     "standard findings",
			findings: snapshotFindings(),
			golden:   "text_standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TextFormatter{Color: false}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			assertGolden(t, tt.golden, buf.Bytes())
		})
	}
}

// TestOutput_JSONSnapshot verifies JSON formatter output matches golden file.
func TestOutput_JSONSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		pretty   bool
		golden   string
	}{
		{
			name:     "empty findings",
			findings: []sdk.Finding{},
			pretty:   true,
			golden:   "json_empty",
		},
		{
			name:     "standard findings",
			findings: snapshotFindings(),
			pretty:   true,
			golden:   "json_standard",
		},
		{
			name:     "compact findings",
			findings: snapshotFindings(),
			pretty:   false,
			golden:   "json_compact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JSONFormatter{Pretty: tt.pretty}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			assertGolden(t, tt.golden, buf.Bytes())
		})
	}
}

// TestOutput_SARIFSnapshot verifies SARIF formatter output matches golden file.
func TestOutput_SARIFSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		version  string
		golden   string
	}{
		{
			name:     "empty findings",
			findings: []sdk.Finding{},
			version:  "1.0.0",
			golden:   "sarif_empty",
		},
		{
			name:     "standard findings",
			findings: snapshotFindings(),
			version:  "1.0.0",
			golden:   "sarif_standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &SARIFFormatter{Version: tt.version}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			assertGolden(t, tt.golden, buf.Bytes())
		})
	}
}

// TestOutput_TableSnapshot verifies table formatter output matches golden file.
func TestOutput_TableSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		verbose  bool
		golden   string
	}{
		{
			name:     "empty findings",
			findings: []sdk.Finding{},
			verbose:  false,
			golden:   "table_empty",
		},
		{
			name:     "standard findings",
			findings: snapshotFindings(),
			verbose:  false,
			golden:   "table_standard",
		},
		{
			name:     "verbose findings",
			findings: snapshotFindings(),
			verbose:  true,
			golden:   "table_verbose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TableFormatter{Color: false, Verbose: tt.verbose}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			assertGolden(t, tt.golden, buf.Bytes())
		})
	}
}
