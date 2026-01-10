package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/terratidy/pkg/sdk"
)

func TestTextFormatter(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		verbose  bool
		want     string
	}{
		{
			name:     "no findings",
			findings: []sdk.Finding{},
			verbose:  false,
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
					},
				},
			},
			verbose: false,
			want:    "✗ test.tf: Test error message (test.error)\n",
		},
		{
			name: "multiple findings",
			findings: []sdk.Finding{
				{
					Rule:     "test.error",
					Message:  "Error",
					File:     "test.tf",
					Severity: sdk.SeverityError,
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
				},
				{
					Rule:     "test.info",
					Message:  "Info",
					File:     "test.tf",
					Severity: sdk.SeverityInfo,
				},
			},
			verbose: false,
			want:    "✗ test.tf: Error (test.error)\n⚠ test.tf: Warning (test.warning)\nℹ test.tf: Info (test.info)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TextFormatter{Verbose: tt.verbose}
			var buf bytes.Buffer
			err := formatter.Format(tt.findings, &buf)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("Format() output mismatch:\ngot:  %q\nwant: %q", got, tt.want)
			}
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
					Fixable:  true,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
					Fixable:  false,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Fixable:  true,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 2, Column: 1},
						End:   hcl.Pos{Line: 2, Column: 10},
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
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Verify it's valid JSON
			var output JSONOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				t.Fatalf("invalid JSON output: %v", err)
			}

			// Verify summary
			if output.Summary.Total != len(tt.findings) {
				t.Errorf("Summary.Total = %d, want %d", output.Summary.Total, len(tt.findings))
			}

			// Verify findings count
			if len(output.Findings) != len(tt.findings) {
				t.Errorf("len(Findings) = %d, want %d", len(output.Findings), len(tt.findings))
			}

			// Verify severity counts
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

			if output.Summary.Errors != expectedErrors {
				t.Errorf("Summary.Errors = %d, want %d", output.Summary.Errors, expectedErrors)
			}
			if output.Summary.Warnings != expectedWarnings {
				t.Errorf("Summary.Warnings = %d, want %d", output.Summary.Warnings, expectedWarnings)
			}
			if output.Summary.Info != expectedInfo {
				t.Errorf("Summary.Info = %d, want %d", output.Summary.Info, expectedInfo)
			}
		})
	}
}

func TestGetFormatter(t *testing.T) {
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
			formatter, err := GetFormatter(tt.format, tt.verbose, "1.0.0")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFormatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				typeName := fmt.Sprintf("%T", formatter)
				if !strings.Contains(typeName, tt.wantType) {
					t.Errorf("GetFormatter() type = %s, want %s", typeName, tt.wantType)
				}
			}
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
					Fixable:  true,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
					Fixable:  false,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Fixable:  true,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 2, Column: 1},
						End:   hcl.Pos{Line: 2, Column: 10},
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
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Verify it's valid JSON
			var sarif SARIF
			if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
				t.Fatalf("invalid SARIF JSON: %v", err)
			}

			// Verify schema
			if sarif.Schema != "https://json.schemastore.org/sarif-2.1.0.json" {
				t.Errorf("Schema = %s, want SARIF 2.1.0 schema", sarif.Schema)
			}

			// Verify version
			if sarif.Version != "2.1.0" {
				t.Errorf("Version = %s, want 2.1.0", sarif.Version)
			}

			// Verify runs
			if len(sarif.Runs) != 1 {
				t.Fatalf("len(Runs) = %d, want 1", len(sarif.Runs))
			}

			run := sarif.Runs[0]

			// Verify tool
			if run.Tool.Driver.Name != "TerraTidy" {
				t.Errorf("Tool name = %s, want TerraTidy", run.Tool.Driver.Name)
			}

			// Verify results count
			if len(run.Results) != len(tt.findings) {
				t.Errorf("len(Results) = %d, want %d", len(run.Results), len(tt.findings))
			}
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
					Fixable:  true,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
					Fixable:  false,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning finding",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Fixable:  true,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 5, Column: 1},
						End:   hcl.Pos{Line: 5, Column: 10},
					},
				},
				{
					Rule:     "test.info",
					Message:  "Info finding",
					File:     "variables.tf",
					Severity: sdk.SeverityInfo,
					Fixable:  false,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 10, Column: 1},
						End:   hcl.Pos{Line: 10, Column: 10},
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
					Fixable:  false,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			output := buf.String()

			// Verify it's HTML
			if !strings.Contains(output, "<!DOCTYPE html>") {
				t.Error("Output should start with DOCTYPE")
			}
			if !strings.Contains(output, "<html") {
				t.Error("Output should contain <html> tag")
			}
			if !strings.Contains(output, "</html>") {
				t.Error("Output should contain closing </html> tag")
			}

			// Verify title
			if !strings.Contains(output, "<title>TerraTidy Report</title>") {
				t.Error("Output should contain title")
			}

			// Verify version in footer
			if !strings.Contains(output, tt.version) {
				t.Errorf("Output should contain version %s", tt.version)
			}

			// Verify summary cards
			if !strings.Contains(output, "Total Issues") {
				t.Error("Output should contain summary cards")
			}

			if len(tt.findings) == 0 {
				// Verify no issues message
				if !strings.Contains(output, "All checks passed") {
					t.Error("Output should show 'All checks passed' for no findings")
				}
			} else {
				// Verify findings are present
				for _, f := range tt.findings {
					if !strings.Contains(output, escapeHTML(f.Rule)) {
						t.Errorf("Output should contain rule %s", f.Rule)
					}
				}
			}

			// Verify XSS protection for special characters test
			if tt.name == "special characters in message" {
				if strings.Contains(output, "<script>") {
					t.Error("Output should escape HTML special characters")
				}
				if !strings.Contains(output, "&lt;script&gt;") {
					t.Error("Output should contain escaped script tag")
				}
			}

			// Verify fixable badge appears for fixable findings
			hasFixable := false
			for _, f := range tt.findings {
				if f.Fixable {
					hasFixable = true
					break
				}
			}
			if hasFixable && !strings.Contains(output, "Fixable") {
				t.Error("Output should contain Fixable badge for fixable findings")
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
			if result != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 10, Column: 5},
						End:   hcl.Pos{Line: 10, Column: 20},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 5, Column: 3},
						End:   hcl.Pos{Line: 5, Column: 10},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning 1",
					File:     "main.tf",
					Severity: sdk.SeverityWarning,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 5, Column: 1},
						End:   hcl.Pos{Line: 5, Column: 10},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 5, Column: 10},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
						End:   hcl.Pos{Line: 1, Column: 10},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 0, Column: 0},
						End:   hcl.Pos{Line: 0, Column: 0},
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
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			output := buf.String()
			lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

			// Handle empty output case
			if len(tt.want) == 0 {
				if output != "" {
					t.Errorf("expected empty output, got: %q", output)
				}
				return
			}

			if len(lines) != len(tt.want) {
				t.Errorf("got %d lines, want %d lines\ngot: %v\nwant: %v", len(lines), len(tt.want), lines, tt.want)
				return
			}

			for i, want := range tt.want {
				if lines[i] != want {
					t.Errorf("line %d mismatch:\ngot:  %q\nwant: %q", i, lines[i], want)
				}
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
			if result != tt.expected {
				t.Errorf("escapeGitHubMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 10, Column: 5},
						End:   hcl.Pos{Line: 10, Column: 20},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1},
					},
				},
				{
					Rule:     "test.warning",
					Message:  "Warning",
					File:     "test.tf",
					Severity: sdk.SeverityWarning,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 2, Column: 1},
					},
				},
				{
					Rule:     "test.info",
					Message:  "Info",
					File:     "test.tf",
					Severity: sdk.SeverityInfo,
					Location: hcl.Range{
						Start: hcl.Pos{Line: 3, Column: 1},
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
					Location: hcl.Range{
						Start: hcl.Pos{Line: 100, Column: 50},
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
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			output := buf.String()

			if len(tt.findings) == 0 {
				if !strings.Contains(output, "No issues found") {
					t.Error("Output should show 'No issues found' for empty findings")
				}
			} else {
				// Verify header is present
				if !strings.Contains(output, "SEVERITY") {
					t.Error("Output should contain SEVERITY header")
				}
				if !strings.Contains(output, "LOCATION") {
					t.Error("Output should contain LOCATION header")
				}
				if !strings.Contains(output, "MESSAGE") {
					t.Error("Output should contain MESSAGE header")
				}

				// Verify summary
				if !strings.Contains(output, "Summary") {
					t.Error("Output should contain Summary")
				}
			}
		})
	}
}

func TestGetFormatterWithColor(t *testing.T) {
	t.Run("table format with color enabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("table", true, "1.0.0", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		table, ok := f.(*TableFormatter)
		if !ok {
			t.Fatal("expected TableFormatter")
		}
		if !table.Color {
			t.Error("expected Color to be true")
		}
	})

	t.Run("table format with color disabled", func(t *testing.T) {
		f, err := GetFormatterWithColor("table", true, "1.0.0", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		table, ok := f.(*TableFormatter)
		if !ok {
			t.Fatal("expected TableFormatter")
		}
		if table.Color {
			t.Error("expected Color to be false")
		}
	})
}
