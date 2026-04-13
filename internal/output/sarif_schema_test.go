package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sarifDocument represents the parsed SARIF structure for field presence validation.
// Using a concrete type instead of map[string]any to make field checks explicit.
type sarifDocument struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []struct {
		Tool struct {
			Driver struct {
				Name           string `json:"name"`
				Version        string `json:"version"`
				InformationURI string `json:"informationUri"`
				Rules          []struct {
					ID               string `json:"id"`
					ShortDescription struct {
						Text string `json:"text"`
					} `json:"shortDescription"`
				} `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []struct {
			RuleID  string `json:"ruleId"`
			Level   string `json:"level"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation struct {
						URI       string `json:"uri"`
						URIBaseID string `json:"uriBaseId"`
					} `json:"artifactLocation"`
					Region struct {
						StartLine   int `json:"startLine"`
						StartColumn int `json:"startColumn"`
						EndLine     int `json:"endLine"`
						EndColumn   int `json:"endColumn"`
					} `json:"region"`
				} `json:"physicalLocation"`
			} `json:"locations"`
		} `json:"results"`
	} `json:"runs"`
}

// TestSARIF_ValidatesAgainstSchema validates that SARIF output conforms to the official SARIF 2.1.0 schema.
func TestSARIF_ValidatesAgainstSchema(t *testing.T) {
	// Load the official SARIF 2.1.0 schema
	schemaData, err := os.ReadFile("testdata/sarif-schema-2.1.0.json")
	require.NoError(t, err, "failed to read SARIF schema")

	// Parse the schema JSON first
	var schemaDoc any
	err = json.Unmarshal(schemaData, &schemaDoc)
	require.NoError(t, err, "failed to parse SARIF schema JSON")

	compiler := jsonschema.NewCompiler()
	err = compiler.AddResource("sarif-schema-2.1.0.json", schemaDoc)
	require.NoError(t, err, "failed to add schema resource")

	schema, err := compiler.Compile("sarif-schema-2.1.0.json")
	require.NoError(t, err, "failed to compile SARIF schema")

	tests := []struct {
		name     string
		findings []sdk.Finding
	}{
		{
			name:     "empty findings",
			findings: []sdk.Finding{},
		},
		{
			name: "single finding",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Test message",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
		},
		{
			name: "multiple findings with different severities",
			findings: []sdk.Finding{
				{
					Rule:     "error.rule",
					Message:  "Error message",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{StartLine: 5, StartColumn: 3, EndLine: 5, EndColumn: 20},
				},
				{
					Rule:     "warning.rule",
					Message:  "Warning message",
					File:     "variables.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 10, StartColumn: 1, EndLine: 12, EndColumn: 5},
				},
				{
					Rule:     "info.rule",
					Message:  "Info message",
					File:     "outputs.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 3, EndColumn: 2},
				},
			},
		},
		{
			name: "finding with fix",
			findings: []sdk.Finding{
				{
					Rule:     "fixable.rule",
					Message:  "This can be auto-fixed",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
					Fix:      &sdk.FixResult{Content: []byte("fixed content")},
				},
			},
		},
		{
			name: "finding with zero location values",
			findings: []sdk.Finding{
				{
					Rule:     "zero.location",
					Message:  "Location has zeros",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 0, StartColumn: 0, EndLine: 0, EndColumn: 0},
				},
			},
		},
		{
			name: "unknown severity falls back to warning",
			findings: []sdk.Finding{
				{
					Rule:     "unknown.severity",
					Message:  "Unknown severity value",
					File:     "test.tf",
					Severity: sdk.Severity("custom"),
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
			},
		},
		{
			name: "file path with spaces",
			findings: []sdk.Finding{
				{
					Rule:     "path.spaces",
					Message:  "File has spaces in path",
					File:     "modules/my module/main.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &SARIFFormatter{Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err, "failed to format SARIF")

			// Parse the output as JSON
			var sarifDoc any
			err = json.Unmarshal(buf.Bytes(), &sarifDoc)
			require.NoError(t, err, "failed to parse SARIF JSON")

			// Validate against schema
			err = schema.Validate(sarifDoc)
			assert.NoError(t, err, "SARIF output does not validate against schema")
		})
	}
}

// TestSARIF_RequiredFieldsPresent verifies all mandatory SARIF 2.1.0 fields exist in output.
// This complements schema validation by making field requirements explicit and debuggable.
func TestSARIF_RequiredFieldsPresent(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
	}{
		{
			name:     "empty findings has all root fields",
			findings: []sdk.Finding{},
		},
		{
			name: "single finding has all nested fields",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Test message",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   10,
						StartColumn: 5,
						EndLine:     10,
						EndColumn:   20,
					},
				},
			},
		},
		{
			name: "multiple findings with rules",
			findings: []sdk.Finding{
				{
					Rule:     "rule.one",
					Message:  "First finding",
					File:     "first.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
				{
					Rule:     "rule.two",
					Message:  "Second finding",
					File:     "second.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{StartLine: 5, StartColumn: 3, EndLine: 7, EndColumn: 15},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &SARIFFormatter{Version: "1.2.3"}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err, "failed to format SARIF")

			var doc sarifDocument
			err = json.Unmarshal(buf.Bytes(), &doc)
			require.NoError(t, err, "failed to parse SARIF JSON")

			// Root-level required fields (SARIF 2.1.0 spec section 3.13)
			assert.Equal(t, "https://json.schemastore.org/sarif-2.1.0.json", doc.Schema, "$schema must be SARIF 2.1.0 schema URL")
			assert.Equal(t, "2.1.0", doc.Version, "version must be 2.1.0")
			require.NotEmpty(t, doc.Runs, "runs array is required and must not be empty")

			// Run-level required fields (SARIF 2.1.0 spec section 3.14)
			run := doc.Runs[0]
			assert.NotEmpty(t, run.Tool.Driver.Name, "tool.driver.name is required")
			assert.NotEmpty(t, run.Tool.Driver.Version, "tool.driver.version should be present")
			assert.NotEmpty(t, run.Tool.Driver.InformationURI, "tool.driver.informationUri should be present")

			// Results array must exist (can be empty but not null)
			assert.NotNil(t, run.Results, "results array must exist (not null)")

			// If there are findings, verify result-level required fields
			if len(tt.findings) > 0 {
				require.Len(t, run.Results, len(tt.findings), "results count must match findings count")

				// Valid SARIF 2.1.0 levels (spec section 3.27.10)
				validLevels := map[string]bool{"none": true, "note": true, "warning": true, "error": true}

				for i, result := range run.Results {
					// Result-level required fields (SARIF 2.1.0 spec section 3.27)
					assert.NotEmpty(t, result.RuleID, "result[%d].ruleId is required", i)
					assert.NotEmpty(t, result.Level, "result[%d].level is required", i)
					assert.True(t, validLevels[result.Level], "result[%d].level must be a valid SARIF level (got %q)", i, result.Level)
					assert.NotEmpty(t, result.Message.Text, "result[%d].message.text is required", i)

					// Location validation (SARIF 2.1.0 spec section 3.28)
					require.NotEmpty(t, result.Locations, "result[%d].locations should not be empty", i)
					loc := result.Locations[0]
					assert.NotEmpty(t, loc.PhysicalLocation.ArtifactLocation.URI, "result[%d].locations[0].physicalLocation.artifactLocation.uri is required", i)
					assert.GreaterOrEqual(t, loc.PhysicalLocation.Region.StartLine, 1, "result[%d].locations[0].physicalLocation.region.startLine must be >= 1", i)
				}

				// Rules should be present when there are findings
				require.NotEmpty(t, run.Tool.Driver.Rules, "rules array should be populated when there are findings")
				for i, rule := range run.Tool.Driver.Rules {
					assert.NotEmpty(t, rule.ID, "rules[%d].id is required", i)
					assert.NotEmpty(t, rule.ShortDescription.Text, "rules[%d].shortDescription.text is required", i)
				}
			}
		})
	}
}

// TestSARIF_RunsArrayNonEmpty verifies the runs array is never empty and contains valid run objects.
// SARIF 2.1.0 spec section 3.13.4 requires at least one run in the runs array.
// This test specifically covers nil vs empty findings to ensure nil-safety in buildSARIFResults.
func TestSARIF_RunsArrayNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
	}{
		{
			name:     "empty slice findings produces non-empty runs",
			findings: []sdk.Finding{},
		},
		{
			// nil findings exercises a different Go code path from empty slice.
			// buildSARIFResults must handle nil gracefully with make([]SARIFResult, 0, 0).
			name:     "nil findings produces non-empty runs",
			findings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &SARIFFormatter{Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err, "Format should not fail")

			var doc sarifDocument
			err = json.Unmarshal(buf.Bytes(), &doc)
			require.NoError(t, err, "JSON unmarshal should not fail")

			// SARIF 2.1.0 spec section 3.13.4: runs array must have at least one element
			require.Len(t, doc.Runs, 1, "runs array must have exactly one run")

			// Verify the run has required structure
			run := doc.Runs[0]
			assert.NotEmpty(t, run.Tool.Driver.Name, "run must have tool.driver.name")
			assert.NotNil(t, run.Results, "run must have results array (can be empty, not null)")
		})
	}
}

// TestSARIF_ResultLocationsValid verifies all result locations have valid region values.
// SARIF 2.1.0 spec section 3.30 requires startLine >= 1, startColumn >= 1 for valid regions.
// The formatter coerces zero/negative values to 1 to ensure spec compliance.
func TestSARIF_ResultLocationsValid(t *testing.T) {
	tests := []struct {
		name          string
		input         sdk.Location
		wantStartLine int
		wantStartCol  int
		wantEndLine   int
		wantEndCol    int
	}{
		{
			name:          "valid values preserved",
			input:         sdk.Location{StartLine: 10, StartColumn: 5, EndLine: 15, EndColumn: 20},
			wantStartLine: 10,
			wantStartCol:  5,
			wantEndLine:   15,
			wantEndCol:    20,
		},
		{
			name:          "zero values coerced to 1",
			input:         sdk.Location{StartLine: 0, StartColumn: 0, EndLine: 0, EndColumn: 0},
			wantStartLine: 1,
			wantStartCol:  1,
			wantEndLine:   1,
			wantEndCol:    1,
		},
		{
			name:          "negative values coerced to 1",
			input:         sdk.Location{StartLine: -5, StartColumn: -3, EndLine: -10, EndColumn: -1},
			wantStartLine: 1,
			wantStartCol:  1,
			wantEndLine:   1,
			wantEndCol:    1,
		},
		{
			name:          "mixed valid and invalid values",
			input:         sdk.Location{StartLine: 5, StartColumn: 0, EndLine: -1, EndColumn: 10},
			wantStartLine: 5,
			wantStartCol:  1,
			wantEndLine:   1,
			wantEndCol:    10,
		},
		{
			// Boundary: minimum valid values pass through unchanged (no coercion needed)
			name:          "minimum valid values pass through unchanged",
			input:         sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 1},
			wantStartLine: 1,
			wantStartCol:  1,
			wantEndLine:   1,
			wantEndCol:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finding := sdk.Finding{
				Rule:     "test.rule",
				Message:  "Test message",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
				Location: tt.input,
			}

			formatter := &SARIFFormatter{Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format([]sdk.Finding{finding}, &buf)
			require.NoError(t, err, "Format should not fail")

			var doc sarifDocument
			err = json.Unmarshal(buf.Bytes(), &doc)
			require.NoError(t, err, "JSON unmarshal should not fail")

			require.Len(t, doc.Runs, 1, "must have one run")
			require.Len(t, doc.Runs[0].Results, 1, "must have one result")
			require.Len(t, doc.Runs[0].Results[0].Locations, 1, "must have one location")

			region := doc.Runs[0].Results[0].Locations[0].PhysicalLocation.Region
			assert.Equal(t, tt.wantStartLine, region.StartLine, "startLine")
			assert.Equal(t, tt.wantStartCol, region.StartColumn, "startColumn")
			assert.Equal(t, tt.wantEndLine, region.EndLine, "endLine")
			assert.Equal(t, tt.wantEndCol, region.EndColumn, "endColumn")
		})
	}
}
