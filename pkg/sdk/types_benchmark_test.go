// Benchmarks for Severity and Location operations in the sdk package.
package sdk

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func BenchmarkParseSeverity(b *testing.B) {
	tests := []struct {
		name       string
		input      string
		defaultSev Severity
	}{
		{"ValidError", "error", SeverityWarning},
		{"ValidWarning", "warning", SeverityError},
		{"ValidInfo", "info", SeverityWarning},
		{"Uppercase", "ERROR", SeverityWarning},
		{"MixedCase", "Warning", SeverityError},
		{"Unknown", "critical", SeverityWarning},
		{"Empty", "", SeverityWarning},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := ParseSeverity(tc.input, tc.defaultSev)
				_ = result
			}
		})
	}
}

func BenchmarkSeverityLevel(b *testing.B) {
	tests := []struct {
		name     string
		severity Severity
	}{
		{"Error", SeverityError},
		{"Warning", SeverityWarning},
		{"Info", SeverityInfo},
		{"Unknown", Severity("unknown")},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := tc.severity.Level()
				_ = result
			}
		})
	}
}

// generateSeverities creates a slice of severities with realistic distribution:
// 20% error, 40% warning, 40% info.
func generateSeverities(count int) []Severity {
	severities := make([]Severity, count)
	for i := range count {
		switch i % 5 {
		case 0:
			severities[i] = SeverityError
		case 1, 2:
			severities[i] = SeverityWarning
		default:
			severities[i] = SeverityInfo
		}
	}
	return severities
}

// BenchmarkSeverityCompare benchmarks realistic severity filtering workflows.
// This measures the full cost of filtering findings by severity threshold.
func BenchmarkSeverityCompare(b *testing.B) {
	threshold := SeverityWarning

	tests := []struct {
		name  string
		count int
	}{
		{"3Items", 3},
		{"100Items", 100},
		{"1000Items", 1000},
		{"5000Items", 5000},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			severities := generateSeverities(tc.count)
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				thresholdLevel := threshold.Level()
				count := 0
				for _, sev := range severities {
					if sev.Level() >= thresholdLevel {
						count++
					}
				}
				_ = count
			}
		})
	}
}

// BenchmarkLocationFromRange benchmarks the HCL Range to Location conversion.
func BenchmarkLocationFromRange(b *testing.B) {
	r := hcl.Range{
		Filename: "modules/service/main.tf",
		Start:    hcl.Pos{Line: 10, Column: 5, Byte: 150},
		End:      hcl.Pos{Line: 15, Column: 1, Byte: 250},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		loc := LocationFromRange(r)
		_ = loc
	}
}
