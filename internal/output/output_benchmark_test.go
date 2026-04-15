package output

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func generateFindings(n int) []sdk.Finding {
	findings := make([]sdk.Finding, n)
	for i := range n {
		sev := sdk.SeverityWarning
		switch i % 3 {
		case 0:
			sev = sdk.SeverityError
		case 2:
			sev = sdk.SeverityInfo
		}
		findings[i] = sdk.Finding{
			Rule:     fmt.Sprintf("test.rule-%d", i%10),
			Message:  fmt.Sprintf("Issue %d in resource block", i),
			File:     fmt.Sprintf("modules/service-%d/main.tf", i%5),
			Severity: sev,
			Location: sdk.Location{
				StartLine:   i + 1,
				StartColumn: 1,
				EndLine:     i + 1,
				EndColumn:   40,
			},
		}
	}
	return findings
}

func BenchmarkSARIFOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &SARIFFormatter{Version: "1.0.0"}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkHTMLOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &HTMLFormatter{Version: "1.0.0"}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkJSONOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &JSONFormatter{Pretty: true}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkTextOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &TextFormatter{}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkJUnitOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &JUnitFormatter{Version: "1.0.0"}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkMarkdownOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &MarkdownFormatter{Version: "1.0.0", Title: "Benchmark Report"}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkTableOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &TableFormatter{Color: false, Verbose: true}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

func BenchmarkGitHubActionsOutput(b *testing.B) {
	findings := generateFindings(1000)
	formatter := &GitHubActionsFormatter{}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}

// BenchmarkOutputManyFindings stress tests all formatters with 5000+ findings.
func BenchmarkOutputManyFindings(b *testing.B) {
	findings := generateFindings(5000)

	formatters := []struct {
		name      string
		formatter Formatter
	}{
		{"Text", &TextFormatter{}},
		{"JSON", &JSONFormatter{Pretty: true}},
		{"SARIF", &SARIFFormatter{Version: "1.0.0"}},
		{"JUnit", &JUnitFormatter{Version: "1.0.0"}},
		{"Markdown", &MarkdownFormatter{Version: "1.0.0", Title: "Benchmark Report"}},
		{"HTML", &HTMLFormatter{Version: "1.0.0"}},
		{"Table", &TableFormatter{Color: false, Verbose: true}},
		{"GitHubActions", &GitHubActionsFormatter{}},
	}

	for _, tc := range formatters {
		b.Run(tc.name, func(b *testing.B) {
			var buf bytes.Buffer

			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				buf.Reset()
				err := tc.formatter.Format(findings, &buf)
				require.NoError(b, err)
			}
		})
	}
}
