package output

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJUnit_ValidatesAgainstSchema validates JUnit output parses correctly as well-formed XML
// and matches the expected JUnit XML structure (testsuites → testsuite → testcase hierarchy).
// Since JUnit uses XSD rather than JSON schema, we validate by parsing into strongly-typed structs.
func TestJUnit_ValidatesAgainstSchema(t *testing.T) {
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
			name: "multiple findings in same file",
			findings: []sdk.Finding{
				{
					Rule:     "rule.one",
					Message:  "First issue",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
				{
					Rule:     "rule.two",
					Message:  "Second issue",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 5, StartColumn: 1, EndLine: 5, EndColumn: 10},
				},
			},
		},
		{
			name:     "nil findings",
			findings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JUnitFormatter{Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err, "failed to format JUnit")

			output := buf.String()

			// Validate XML declaration is present
			assert.True(t, strings.HasPrefix(output, "<?xml"), "must start with XML declaration")

			// Parse as well-formed XML into strongly-typed struct
			var testSuites JUnitTestSuites
			err = xml.Unmarshal(buf.Bytes(), &testSuites)
			assert.NoError(t, err, "JUnit output must be valid, parseable XML")

			// Validate root element name
			assert.Equal(t, "testsuites", testSuites.XMLName.Local, "root element must be 'testsuites'")
		})
	}
}

// TestJUnit_TestSuiteStructure validates the proper JUnit hierarchy:
// testsuites (root) → testsuite (per file) → testcase (per finding).
//
// Note: The root testsuites.tests attribute counts findings, not test cases.
// For empty findings, tests=0 even though a synthetic "all_checks_passed" test case exists.
func TestJUnit_TestSuiteStructure(t *testing.T) {
	tests := []struct {
		name          string
		findings      []sdk.Finding
		wantSuites    int
		wantCases     int
		wantErrors    int
		wantFailures  int
		wantFileNames []string
	}{
		{
			name:          "empty findings creates single passing suite",
			findings:      []sdk.Finding{},
			wantSuites:    1,
			wantCases:     1, // "all_checks_passed" test case
			wantErrors:    0,
			wantFailures:  0,
			wantFileNames: []string{"terratidy"},
		},
		{
			name: "single file with one error",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Error message",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
			},
			wantSuites:    1,
			wantCases:     1,
			wantErrors:    1,
			wantFailures:  0,
			wantFileNames: []string{"main.tf"},
		},
		{
			name: "single file with one warning",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Warning message",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
			},
			wantSuites:    1,
			wantCases:     1,
			wantErrors:    0,
			wantFailures:  1,
			wantFileNames: []string{"main.tf"},
		},
		{
			name: "multiple files create separate suites",
			findings: []sdk.Finding{
				{
					Rule:     "rule.one",
					Message:  "First",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
				{
					Rule:     "rule.two",
					Message:  "Second",
					File:     "variables.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
			},
			wantSuites:    2,
			wantCases:     2,
			wantErrors:    1,
			wantFailures:  1,
			wantFileNames: []string{"main.tf", "variables.tf"},
		},
		{
			name: "info severity is neither error nor failure",
			findings: []sdk.Finding{
				{
					Rule:     "info.rule",
					Message:  "Info message",
					File:     "test.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
				},
			},
			wantSuites:    1,
			wantCases:     1,
			wantErrors:    0,
			wantFailures:  0,
			wantFileNames: []string{"test.tf"},
		},
		{
			name: "multiple findings in same file grouped",
			findings: []sdk.Finding{
				{
					Rule:     "rule.one",
					Message:  "First",
					File:     "main.tf",
					Severity: sdk.SeverityError,
				},
				{
					Rule:     "rule.two",
					Message:  "Second",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
				},
				{
					Rule:     "rule.three",
					Message:  "Third",
					File:     "main.tf",
					Severity: sdk.SeverityInfo,
				},
			},
			wantSuites:    1,
			wantCases:     3,
			wantErrors:    1,
			wantFailures:  1,
			wantFileNames: []string{"main.tf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JUnitFormatter{Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err, "Format should not fail")

			var testSuites JUnitTestSuites
			err = xml.Unmarshal(buf.Bytes(), &testSuites)
			require.NoError(t, err, "XML unmarshal should not fail")

			// Validate root-level attributes
			assert.Equal(t, "TerraTidy", testSuites.Name, "testsuites name")
			// Note: testsuites.Tests counts findings, not test cases.
			// For empty findings, this is 0 even though there's 1 synthetic test case.
			assert.Equal(t, len(tt.findings), testSuites.Tests, "testsuites tests count")
			assert.Equal(t, tt.wantErrors, testSuites.Errors, "testsuites errors count")
			assert.Equal(t, tt.wantFailures, testSuites.Failures, "testsuites failures count")
			assert.NotEmpty(t, testSuites.Timestamp, "testsuites timestamp required")

			// Validate suite count
			require.Len(t, testSuites.Suites, tt.wantSuites, "number of test suites")

			// Validate suite names match expected files
			suiteNames := make([]string, len(testSuites.Suites))
			for i, suite := range testSuites.Suites {
				suiteNames[i] = suite.Name
				// Each suite must have required attributes
				assert.NotEmpty(t, suite.Name, "testsuite[%d].name required", i)
				assert.NotEmpty(t, suite.Timestamp, "testsuite[%d].timestamp required", i)
				assert.GreaterOrEqual(t, suite.Tests, 0, "testsuite[%d].tests must be non-negative", i)
				assert.GreaterOrEqual(t, suite.Errors, 0, "testsuite[%d].errors must be non-negative", i)
				assert.GreaterOrEqual(t, suite.Failures, 0, "testsuite[%d].failures must be non-negative", i)
				assert.GreaterOrEqual(t, suite.Skipped, 0, "testsuite[%d].skipped must be non-negative", i)
			}
			assert.ElementsMatch(t, tt.wantFileNames, suiteNames, "suite names should match expected files")

			// Validate total test cases across all suites
			var totalCases int
			for _, suite := range testSuites.Suites {
				totalCases += len(suite.TestCases)
				for j, tc := range suite.TestCases {
					// Each test case must have required attributes
					assert.NotEmpty(t, tc.Name, "testsuite.testcase[%d].name required", j)
					assert.NotEmpty(t, tc.ClassName, "testsuite.testcase[%d].classname required", j)
				}
			}
			assert.Equal(t, tt.wantCases, totalCases, "total test cases")
		})
	}
}

// TestJUnit_EscapesXMLSpecialChars verifies that XML special characters in messages, rules,
// and file paths are properly escaped to produce valid XML output.
func TestJUnit_EscapesXMLSpecialChars(t *testing.T) {
	tests := []struct {
		name      string
		finding   sdk.Finding
		wantInXML []string // Strings that should appear escaped in XML
	}{
		{
			name: "ampersand in message",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  "Expected A & B",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{"A &amp; B"},
		},
		{
			name: "less than in message",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  "Value < 10",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{"&lt; 10"},
		},
		{
			name: "greater than in message",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  "Value > 10",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{"&gt; 10"},
		},
		{
			// Tests that formatter preserves stdlib's XML encoding of quotes.
			// encoding/xml uses &#34; for double quotes in attributes.
			// This verifies the formatter doesn't break that encoding.
			name: "quotes in message",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  `Use "double" or 'single' quotes`,
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{`&#34;double&#34;`},
		},
		{
			name: "multiple special chars in message",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  "<script>alert('XSS & injection')</script>",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{"&lt;script&gt;", "&amp;"},
		},
		{
			name: "special chars in rule name",
			finding: sdk.Finding{
				Rule:     "rule<>test",
				Message:  "Test message",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{"rule&lt;&gt;test"},
		},
		{
			name: "special chars in file path",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  "Test message",
				File:     "path/with<special>&chars.tf",
				Severity: sdk.SeverityWarning,
			},
			wantInXML: []string{"with&lt;special&gt;&amp;chars"},
		},
		{
			name: "newlines in message encoded as XML entities",
			finding: sdk.Finding{
				Rule:     "test.rule",
				Message:  "Line 1\nLine 2\nLine 3",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
			// Newlines are encoded as &#xA; in XML attributes
			wantInXML: []string{"Line 1&#xA;Line 2&#xA;Line 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JUnitFormatter{Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format([]sdk.Finding{tt.finding}, &buf)
			require.NoError(t, err, "Format should not fail")

			output := buf.String()

			// Output must be valid XML (parseable)
			var testSuites JUnitTestSuites
			err = xml.Unmarshal(buf.Bytes(), &testSuites)
			assert.NoError(t, err, "output must be valid XML after escaping")

			// Check expected escaped strings appear
			for _, want := range tt.wantInXML {
				assert.Contains(t, output, want, "expected escaped string in output")
			}
		})
	}
}

// TestJUnit_NoInvalidXMLCharacters verifies that the output never contains
// XML-invalid characters (control chars except tab, CR, LF).
func TestJUnit_NoInvalidXMLCharacters(t *testing.T) {
	// Test with various control characters that are invalid in XML 1.0
	invalidChars := []struct {
		name string
		char string
	}{
		{"null byte", "\x00"},
		{"bell", "\x07"},
		{"backspace", "\x08"},
		{"vertical tab", "\x0B"},
		{"form feed", "\x0C"},
		{"unit separator", "\x1F"},
	}

	for _, tc := range invalidChars {
		t.Run(tc.name+" in message", func(t *testing.T) {
			formatter := &JUnitFormatter{Version: "1.0.0"}
			var buf bytes.Buffer

			// Create finding with invalid character in message
			finding := sdk.Finding{
				Rule:     "test.rule",
				Message:  "Invalid char: " + tc.char + " here",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			}

			err := formatter.Format([]sdk.Finding{finding}, &buf)
			require.NoError(t, err, "Format should not fail")

			// The XML should be parseable (Go's xml.Marshal escapes or removes invalid chars)
			var testSuites JUnitTestSuites
			err = xml.Unmarshal(buf.Bytes(), &testSuites)
			require.NoError(t, err, "output must be valid XML even with control chars in input")

			// Verify safe portions of the message survived
			// encoding/xml replaces invalid chars with the Unicode replacement char or strips them
			output := buf.String()
			assert.Contains(t, output, "Invalid char:", "safe prefix should survive")
			assert.Contains(t, output, "here", "safe suffix should survive")
		})
	}
}
