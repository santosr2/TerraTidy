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
