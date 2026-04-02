package output

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
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
			Location: hcl.Range{
				Start: hcl.Pos{Line: i + 1, Column: 1},
				End:   hcl.Pos{Line: i + 1, Column: 40},
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
	formatter := &TextFormatter{Verbose: true}
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		buf.Reset()
		err := formatter.Format(findings, &buf)
		require.NoError(b, err)
	}
}
