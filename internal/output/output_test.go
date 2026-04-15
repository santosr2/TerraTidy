package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		want     string
	}{
		{
			name:     "no findings",
			findings: []sdk.Finding{},
			want:     "✓ No issues found\n",
		},
		{
			name: "single error",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Test error message",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
			want: "✗ test.tf:1:1: Test error message (test.error)\n",
		},
		{
			name: "multiple findings",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{StartLine: 1, StartColumn: 1},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{StartLine: 5, StartColumn: 3},
				},
				{
					Rule:     "test.info",
					Message:  "Info",
					File:     "test.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{StartLine: 10, StartColumn: 1},
				},
			},
			want: "✗ test.tf:1:1: Error (test.error)\n⚠ test.tf:5:3: Warning (test.warning)\nℹ test.tf:10:1: Info (test.info)\n",
		},
		{
			name: "finding without location (file-level)",
			findings: []sdk.Finding{
				{
					Rule:     "policy.required-tags",
					Message:  "File is missing required tags",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					// No Location set - StartLine defaults to 0
				},
			},
			want: "✗ main.tf: File is missing required tags (policy.required-tags)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TextFormatter{}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		pretty   bool
	}{
		{
			name:     "no findings",
			findings: []sdk.Finding{},
			pretty:   true,
		},
		{
			name: "single finding",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Test message",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Fix:      &sdk.FixResult{Content: []byte("fixed")},
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
			pretty: true,
		},
		{
			name: "multiple findings",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Fix:      &sdk.FixResult{Content: []byte("fixed")},
					Location: sdk.Location{
						StartLine:   2,
						StartColumn: 1,
						EndLine:     2,
						EndColumn:   10,
					},
				},
			},
			pretty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JSONFormatter{Pretty: tt.pretty}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			var output JSONOutput
			require.NoError(t, json.Unmarshal(buf.Bytes(), &output), "invalid JSON output")

			assert.Equal(t, len(tt.findings), output.Summary.Total)
			assert.Len(t, output.Findings, len(tt.findings))

			expectedErrors := 0
			expectedWarnings := 0
			expectedInfo := 0
			for _, f := range tt.findings {
				switch f.Severity {
				case sdk.SeverityError:
					expectedErrors++
				case sdk.SeverityWarning:
					expectedWarnings++
				case sdk.SeverityInfo:
					expectedInfo++
				}
			}

			assert.Equal(t, expectedErrors, output.Summary.Errors)
			assert.Equal(t, expectedWarnings, output.Summary.Warnings)
			assert.Equal(t, expectedInfo, output.Summary.Info)
		})
	}
}

func TestGetFormatterWithColor_Formats(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		verbose  bool
		wantErr  bool
		wantType string
	}{
		{
			name:     "text format",
			format:   "text",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.TextFormatter",
		},
		{
			name:     "json format",
			format:   "json",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.JSONFormatter",
		},
		{
			name:     "json-compact format",
			format:   "json-compact",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.JSONFormatter",
		},
		{
			name:     "sarif format",
			format:   "sarif",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.SARIFFormatter",
		},
		{
			name:     "html format",
			format:   "html",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.HTMLFormatter",
		},
		{
			name:     "github format",
			format:   "github",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.GitHubActionsFormatter",
		},
		{
			name:     "gha format alias",
			format:   "gha",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.GitHubActionsFormatter",
		},
		{
			name:     "empty format (defaults to text)",
			format:   "",
			verbose:  false,
			wantErr:  false,
			wantType: "*output.TextFormatter",
		},
		{
			name:    "unsupported format",
			format:  "xml",
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := GetFormatterWithColor(tt.format, tt.verbose, "1.0.0", true, false)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			typeName := fmt.Sprintf("%T", formatter)
			assert.Contains(t, typeName, tt.wantType)
		})
	}
}

func TestSARIFFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		version  string
	}{
		{
			name:     "no findings",
			findings: []sdk.Finding{},
			version:  "1.0.0",
		},
		{
			name: "single finding",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Test message",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Fix:      &sdk.FixResult{Content: []byte("fixed")},
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
			version: "1.0.0",
		},
		{
			name: "multiple findings",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Fix:      &sdk.FixResult{Content: []byte("fixed")},
					Location: sdk.Location{
						StartLine:   2,
						StartColumn: 1,
						EndLine:     2,
						EndColumn:   10,
					},
				},
			},
			version: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &SARIFFormatter{Version: tt.version}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			// Verify it's valid JSON
			var sarif SARIF
			require.NoError(t, json.Unmarshal(buf.Bytes(), &sarif), "invalid SARIF JSON")

			// Verify schema
			assert.Equal(t, "https://json.schemastore.org/sarif-2.1.0.json", sarif.Schema)

			// Verify version
			assert.Equal(t, "2.1.0", sarif.Version)

			// Verify runs
			require.Len(t, sarif.Runs, 1)

			run := sarif.Runs[0]

			// Verify tool
			assert.Equal(t, "TerraTidy", run.Tool.Driver.Name)

			// Verify results count
			assert.Len(t, run.Results, len(tt.findings))
		})
	}
}

func TestHTMLFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		version  string
	}{
		{
			name:     "no findings",
			findings: []sdk.Finding{},
			version:  "1.0.0",
		},
		{
			name: "single finding",
			findings: []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Test message",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Fix:      &sdk.FixResult{Content: []byte("fixed")},
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
			version: "1.0.0",
		},
		{
			name: "multiple findings different files",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error finding",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning finding",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Fix:      &sdk.FixResult{Content: []byte("fixed")},
					Location: sdk.Location{
						StartLine:   5,
						StartColumn: 1,
						EndLine:     5,
						EndColumn:   10,
					},
				},
				{
					Rule:     "test.info",
					Message:  "Info finding",
					File:     "variables.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{
						StartLine:   10,
						StartColumn: 1,
						EndLine:     10,
						EndColumn:   10,
					},
				},
			},
			version: "2.0.0",
		},
		{
			name: "special characters in message",
			findings: []sdk.Finding{
				{
					Rule:     "test.xss",
					Message:  "<script>alert('xss')</script>",
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
			version: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &HTMLFormatter{Title: "TerraTidy Report", Version: tt.version}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			output := buf.String()

			// Verify it's HTML
			assert.Contains(t, output, "<!DOCTYPE html>", "Output should start with DOCTYPE")
			assert.Contains(t, output, "<html", "Output should contain <html> tag")
			assert.Contains(t, output, "</html>", "Output should contain closing </html> tag")

			// Verify title
			assert.Contains(t, output, "<title>TerraTidy Report</title>", "Output should contain title")

			// Verify version in footer
			assert.Contains(t, output, tt.version, "Output should contain version %s", tt.version)

			// Verify summary cards
			assert.Contains(t, output, "Total Issues", "Output should contain summary cards")

			if len(tt.findings) == 0 {
				// Verify no issues message
				assert.Contains(t, output, "All checks passed", "Output should show 'All checks passed' for no findings")
			} else {
				// Verify findings are present
				for _, f := range tt.findings {
					assert.Contains(t, output, escapeHTML(f.Rule), "Output should contain rule %s", f.Rule)
				}
			}

			// Verify XSS protection for special characters test
			if tt.name == "special characters in message" {
				assert.NotContains(t, output, "<script>", "Output should escape HTML special characters")
				assert.Contains(t, output, "&lt;script&gt;", "Output should contain escaped script tag")
			}

			// Verify fixable badge appears for fixable findings
			hasFixable := false
			for _, f := range tt.findings {
				if f.Fix != nil {
					hasFixable = true
					break
				}
			}
			if hasFixable {
				assert.Contains(t, output, "Fixable", "Output should contain Fixable badge for fixable findings")
			}
		})
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{"\"quoted\"", "&quot;quoted&quot;"},
		{"it's", "it&#39;s"},
		{"<div class=\"foo\">bar</div>", "&lt;div class=&quot;foo&quot;&gt;bar&lt;/div&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitHubActionsFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		want     []string
	}{
		{
			name:     "no findings",
			findings: []sdk.Finding{},
			want:     []string{},
		},
		{
			name: "single error",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Test error message",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   10,
						StartColumn: 5,
						EndLine:     10,
						EndColumn:   20,
					},
				},
			},
			want: []string{"::error file=test.tf,line=10,col=5,title=test.error::Test error message"},
		},
		{
			name: "single warning",
			findings: []sdk.Finding{
				{
					Rule:     "test.warning",
					Message:  "Test warning",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
			want: []string{"::warning file=main.tf,line=1,col=1,title=test.warning::Test warning"},
		},
		{
			name: "single notice (info)",
			findings: []sdk.Finding{
				{
					Rule:     "test.info",
					Message:  "Test info",
					File:     "vars.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{
						StartLine:   5,
						StartColumn: 3,
						EndLine:     5,
						EndColumn:   10,
					},
				},
			},
			want: []string{"::notice file=vars.tf,line=5,col=3,title=test.info::Test info"},
		},
		{
			name: "multiple findings",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error 1",
					File:     "main.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning 1",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   5,
						StartColumn: 1,
						EndLine:     5,
						EndColumn:   10,
					},
				},
			},
			want: []string{
				"::error file=main.tf,line=1,col=1,title=test.error::Error 1",
				"::warning file=main.tf,line=5,col=1,title=test.warning::Warning 1",
			},
		},
		{
			name: "multiline finding with endLine",
			findings: []sdk.Finding{
				{
					Rule:     "test.multiline",
					Message:  "Spans multiple lines",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     5,
						EndColumn:   10,
					},
				},
			},
			want: []string{"::warning file=test.tf,line=1,col=1,endLine=5,endColumn=10,title=test.multiline::Spans multiple lines"},
		},
		{
			name: "message with special characters",
			findings: []sdk.Finding{
				{
					Rule:     "test.special",
					Message:  "Line 1\nLine 2\rLine 3",
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
			want: []string{"::warning file=test.tf,line=1,col=1,title=test.special::Line 1%0ALine 2%0DLine 3"},
		},
		{
			name: "message with percent sign",
			findings: []sdk.Finding{
				{
					Rule:     "test.percent",
					Message:  "100% coverage",
					File:     "test.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   10,
					},
				},
			},
			want: []string{"::notice file=test.tf,line=1,col=1,title=test.percent::100%25 coverage"},
		},
		{
			name: "zero line/column defaults to 1",
			findings: []sdk.Finding{
				{
					Rule:     "test.zero",
					Message:  "Zero position",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   0,
						StartColumn: 0,
						EndLine:     0,
						EndColumn:   0,
					},
				},
			},
			want: []string{"::warning file=test.tf,line=1,col=1,title=test.zero::Zero position"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &GitHubActionsFormatter{}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			output := buf.String()
			lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

			// Handle empty output case
			if len(tt.want) == 0 {
				assert.Empty(t, output)
				return
			}

			require.Len(t, lines, len(tt.want), "got: %v\nwant: %v", lines, tt.want)

			for i, want := range tt.want {
				assert.Equal(t, want, lines[i], "line %d", i)
			}
		})
	}
}

func TestEscapeGitHubMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"line1\nline2", "line1%0Aline2"},
		{"line1\rline2", "line1%0Dline2"},
		{"100%", "100%25"},
		{"a\nb\rc%d", "a%0Ab%0Dc%25d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeGitHubMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTableFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		color    bool
		verbose  bool
	}{
		{
			name:     "no findings with color",
			findings: []sdk.Finding{},
			color:    true,
			verbose:  false,
		},
		{
			name:     "no findings without color",
			findings: []sdk.Finding{},
			color:    false,
			verbose:  false,
		},
		{
			name: "single error with color",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Test error message",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   10,
						StartColumn: 5,
						EndLine:     10,
						EndColumn:   20,
					},
				},
			},
			color:   true,
			verbose: false,
		},
		{
			name: "multiple findings without color verbose",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error",
					File:     "test.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   1,
						StartColumn: 1,
						EndLine:     1,
						EndColumn:   1,
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: sdk.Location{
						StartLine:   2,
						StartColumn: 1,
						EndLine:     2,
						EndColumn:   1,
					},
				},
				{
					Rule:     "test.info",
					Message:  "Info",
					File:     "test.tf",
					Severity: sdk.SeverityInfo,
					Location: sdk.Location{
						StartLine:   3,
						StartColumn: 1,
						EndLine:     3,
						EndColumn:   1,
					},
				},
			},
			color:   false,
			verbose: true,
		},
		{
			name: "long location gets truncated",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error",
					File:     "/very/long/path/to/some/deeply/nested/terraform/module/file.tf",
					Severity: sdk.SeverityError,
					Location: sdk.Location{
						StartLine:   100,
						StartColumn: 50,
						EndLine:     100,
						EndColumn:   50,
					},
				},
			},
			color:   false,
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TableFormatter{Color: tt.color, Verbose: tt.verbose}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			require.NoError(t, err)

			output := buf.String()

			if len(tt.findings) == 0 {
				assert.Contains(t, output, "No issues found", "Output should show 'No issues found' for empty findings")
			} else {
				// Verify header is present
				assert.Contains(t, output, "SEVERITY", "Output should contain SEVERITY header")
				assert.Contains(t, output, "LOCATION", "Output should contain LOCATION header")
				assert.Contains(t, output, "MESSAGE", "Output should contain MESSAGE header")

				// Verify summary
				assert.Contains(t, output, "Summary", "Output should contain Summary")
			}
		})
	}
}

func TestGetFormatterWithColor(t *testing.T) {
	t.Run("table format with color enabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("table", true, "1.0.0", true, false)
		require.NoError(t, err)
		table, ok := f.(*TableFormatter)
		require.True(t, ok, "expected TableFormatter")
		assert.True(t, table.Color)
	})

	t.Run("table format with color disabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("table", true, "1.0.0", false, false)
		require.NoError(t, err)
		table, ok := f.(*TableFormatter)
		require.True(t, ok, "expected TableFormatter")
		assert.False(t, table.Color)
	})

	t.Run("text format with absolutePaths enabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("text", true, "1.0.0", true, true)
		require.NoError(t, err)
		text, ok := f.(*TextFormatter)
		require.True(t, ok, "expected TextFormatter")
		assert.True(t, text.AbsolutePaths)
	})

	t.Run("text format with absolutePaths disabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("text", true, "1.0.0", true, false)
		require.NoError(t, err)
		text, ok := f.(*TextFormatter)
		require.True(t, ok, "expected TextFormatter")
		assert.False(t, text.AbsolutePaths)
	})

	t.Run("json format with absolutePaths enabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("json", true, "1.0.0", true, true)
		require.NoError(t, err)
		jsonFmt, ok := f.(*JSONFormatter)
		require.True(t, ok, "expected JSONFormatter")
		assert.True(t, jsonFmt.AbsolutePaths)
	})

	t.Run("sarif format with absolutePaths enabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("sarif", true, "1.0.0", true, true)
		require.NoError(t, err)
		sarifFmt, ok := f.(*SARIFFormatter)
		require.True(t, ok, "expected SARIFFormatter")
		assert.True(t, sarifFmt.AbsolutePaths)
	})
}

func TestTextFormatterPathBehavior(t *testing.T) {
	// Use an absolute path for testing
	absPath := "/absolute/path/to/test.tf"

	finding := sdk.Finding{
		Rule:     "test.rule",
		Message:  "Test message",
		File:     absPath,
		Severity: sdk.SeverityError,
	}

	t.Run("relative paths by default", func(t *testing.T) {
		formatter := &TextFormatter{AbsolutePaths: false}
		var buf bytes.Buffer
		err := formatter.Format([]sdk.Finding{finding}, &buf)
		require.NoError(t, err)
		// When AbsolutePaths is false and path is absolute, displayPath attempts
		// to make it relative. Since the path doesn't exist under cwd, it stays absolute.
		// But the key is that AbsolutePaths=false is set.
		assert.Contains(t, buf.String(), "test.tf")
	})

	t.Run("absolute paths when enabled", func(t *testing.T) {
		formatter := &TextFormatter{AbsolutePaths: true}
		var buf bytes.Buffer
		err := formatter.Format([]sdk.Finding{finding}, &buf)
		require.NoError(t, err)
		// When AbsolutePaths is true, the full path is preserved
		assert.Contains(t, buf.String(), absPath)
	})
}

func TestDisplayPathRelativeConversion(t *testing.T) {
	// Test that displayPath converts absolute paths under cwd to relative paths
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Create an absolute path under the current working directory
	absPathUnderCwd := filepath.Join(cwd, "subdir", "test.tf")

	finding := sdk.Finding{
		Rule:     "test.rule",
		Message:  "Test message",
		File:     absPathUnderCwd,
		Severity: sdk.SeverityError,
	}

	t.Run("converts to relative when under cwd", func(t *testing.T) {
		formatter := &TextFormatter{AbsolutePaths: false}
		var buf bytes.Buffer
		err := formatter.Format([]sdk.Finding{finding}, &buf)
		require.NoError(t, err)
		output := buf.String()
		// Should contain the relative path, not the full absolute path
		// Use filepath.Join to get platform-correct separator
		expectedRelPath := filepath.Join("subdir", "test.tf")
		assert.Contains(t, output, expectedRelPath)
		assert.NotContains(t, output, cwd)
	})

	t.Run("keeps absolute when flag enabled", func(t *testing.T) {
		formatter := &TextFormatter{AbsolutePaths: true}
		var buf bytes.Buffer
		err := formatter.Format([]sdk.Finding{finding}, &buf)
		require.NoError(t, err)
		output := buf.String()
		// Should contain the full absolute path
		assert.Contains(t, output, absPathUnderCwd)
	})
}

func TestJSONFormatterPathBehavior(t *testing.T) {
	absPath := "/absolute/path/to/test.tf"

	finding := sdk.Finding{
		Rule:     "test.rule",
		Message:  "Test message",
		File:     absPath,
		Severity: sdk.SeverityError,
	}

	t.Run("absolute paths when enabled", func(t *testing.T) {
		formatter := &JSONFormatter{Pretty: false, AbsolutePaths: true}
		var buf bytes.Buffer
		err := formatter.Format([]sdk.Finding{finding}, &buf)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		findings := result["findings"].([]any)
		require.Len(t, findings, 1)
		firstFinding := findings[0].(map[string]any)
		assert.Equal(t, absPath, firstFinding["file"])
	})
}

func TestSARIFFormatterPathBehavior(t *testing.T) {
	absPath := "/absolute/path/to/test.tf"

	finding := sdk.Finding{
		Rule:     "test.rule",
		Message:  "Test message",
		File:     absPath,
		Severity: sdk.SeverityError,
	}

	t.Run("absolute paths when enabled", func(t *testing.T) {
		formatter := &SARIFFormatter{Version: "1.0.0", AbsolutePaths: true}
		var buf bytes.Buffer
		err := formatter.Format([]sdk.Finding{finding}, &buf)
		require.NoError(t, err)

		// Parse SARIF output and check the artifact location contains the absolute path
		assert.Contains(t, buf.String(), absPath)
	})
}

func TestHTMLFormatterSeverityIcons(t *testing.T) {
	tests := []struct {
		name     string
		severity sdk.Severity
		wantIcon string
	}{
		{"error shows ✗", sdk.SeverityError, "✗"},
		{"warning shows ⚠", sdk.SeverityWarning, "⚠"},
		{"info shows ℹ", sdk.SeverityInfo, "ℹ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := []sdk.Finding{
				{
					Rule:     "test.rule",
					Message:  "Test message",
					File:     "test.tf",
					Severity: tt.severity,
					Location: sdk.Location{StartLine: 1, StartColumn: 1},
				},
			}

			formatter := &HTMLFormatter{Title: "Test", Version: "1.0.0"}
			var buf bytes.Buffer
			err := formatter.Format(findings, &buf)
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.wantIcon,
				"HTML output for %s severity should contain %s icon", tt.severity, tt.wantIcon)
		})
	}
}

// TestHTMLFormatterDeterministic verifies that HTML output is deterministic across runs
func TestHTMLFormatterDeterministic(t *testing.T) {
	// Create findings with files in non-sorted order
	findings := []sdk.Finding{
		{Rule: "rule1", Message: "msg1", File: "z_file.tf", Severity: sdk.SeverityError},
		{Rule: "rule2", Message: "msg2", File: "a_file.tf", Severity: sdk.SeverityWarning},
		{Rule: "rule3", Message: "msg3", File: "m_file.tf", Severity: sdk.SeverityInfo},
	}

	formatter := &HTMLFormatter{Version: "1.0.0"}

	// Run format multiple times and verify output is identical
	var outputs []string
	for i := 0; i < 5; i++ {
		var buf bytes.Buffer
		err := formatter.Format(findings, &buf)
		require.NoError(t, err)
		outputs = append(outputs, buf.String())
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		assert.Equal(t, outputs[0], outputs[i], "HTML output should be deterministic across runs")
	}

	// Verify files appear in sorted order
	output := outputs[0]
	aIdx := strings.Index(output, "a_file.tf")
	mIdx := strings.Index(output, "m_file.tf")
	zIdx := strings.Index(output, "z_file.tf")
	assert.True(t, aIdx < mIdx && mIdx < zIdx, "Files should appear in sorted order: a < m < z")
}

// TestJUnitFormatterDeterministic verifies that JUnit output is deterministic across runs
func TestJUnitFormatterDeterministic(t *testing.T) {
	// Create findings with files in non-sorted order
	findings := []sdk.Finding{
		{Rule: "rule1", Message: "msg1", File: "z_file.tf", Severity: sdk.SeverityError},
		{Rule: "rule2", Message: "msg2", File: "a_file.tf", Severity: sdk.SeverityWarning},
		{Rule: "rule3", Message: "msg3", File: "m_file.tf", Severity: sdk.SeverityInfo},
	}

	formatter := &JUnitFormatter{Version: "1.0.0"}

	// Run format multiple times and verify output is identical
	var outputs []string
	for i := 0; i < 5; i++ {
		var buf bytes.Buffer
		err := formatter.Format(findings, &buf)
		require.NoError(t, err)
		outputs = append(outputs, buf.String())
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		assert.Equal(t, outputs[0], outputs[i], "JUnit output should be deterministic across runs")
	}

	// Verify test suites appear in sorted order by file
	output := outputs[0]
	aIdx := strings.Index(output, "a_file.tf")
	mIdx := strings.Index(output, "m_file.tf")
	zIdx := strings.Index(output, "z_file.tf")
	assert.True(t, aIdx < mIdx && mIdx < zIdx, "Test suites should appear in sorted order: a < m < z")
}

// TestSARIFFormatterDeterministic verifies that SARIF output is deterministic across runs
func TestSARIFFormatterDeterministic(t *testing.T) {
	// Create findings with rules in non-sorted order
	findings := []sdk.Finding{
		{Rule: "z_rule", Message: "msg1", File: "test.tf", Severity: sdk.SeverityError},
		{Rule: "a_rule", Message: "msg2", File: "test.tf", Severity: sdk.SeverityWarning},
		{Rule: "m_rule", Message: "msg3", File: "test.tf", Severity: sdk.SeverityInfo},
	}

	formatter := &SARIFFormatter{Version: "1.0.0"}

	// Run format multiple times and verify output is identical
	var outputs []string
	for i := 0; i < 5; i++ {
		var buf bytes.Buffer
		err := formatter.Format(findings, &buf)
		require.NoError(t, err)
		outputs = append(outputs, buf.String())
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		assert.Equal(t, outputs[0], outputs[i], "SARIF output should be deterministic across runs")
	}

	// Verify rules appear in sorted order in the rules array
	output := outputs[0]

	// Parse JSON to check rules array order
	var sarif SARIF
	err := json.Unmarshal([]byte(output), &sarif)
	require.NoError(t, err)
	require.Len(t, sarif.Runs, 1)

	rules := sarif.Runs[0].Tool.Driver.Rules
	require.Len(t, rules, 3)
	assert.Equal(t, "a_rule", rules[0].ID, "Rules should be sorted alphabetically")
	assert.Equal(t, "m_rule", rules[1].ID, "Rules should be sorted alphabetically")
	assert.Equal(t, "z_rule", rules[2].ID, "Rules should be sorted alphabetically")
}
