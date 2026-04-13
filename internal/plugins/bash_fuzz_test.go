//go:build !windows

// Bash rules are Unix-only; see bash_rule.go.

package plugins

import (
	"encoding/json"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// FuzzBashRuleOutput tests the JSON parsing path in BashRule.Check.
// It exercises json.Unmarshal of bashRuleOutput and conversion to sdk.Finding.
// The goal is to find panics or crashes when parsing arbitrary JSON output
// from bash rule scripts.
func FuzzBashRuleOutput(f *testing.F) {
	// Valid outputs
	f.Add([]byte(`{"findings":[]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","line":1,"message":"test"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","line":1,"column":5,"message":"test","severity":"error","rule":"my-rule"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","line":1,"message":"m1"},{"file":"b.tf","line":2,"message":"m2"}]}`))

	// Edge cases - empty/null
	f.Add([]byte(``))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"findings":null}`))

	// Malformed JSON
	f.Add([]byte(`{`))
	f.Add([]byte(`{"findings":`))
	f.Add([]byte(`{"findings":[}`))
	f.Add([]byte(`not json at all`))
	f.Add([]byte{0x00, 0x01, 0x02}) // Binary garbage

	// Edge case values
	f.Add([]byte(`{"findings":[{"file":"","line":0,"column":0,"message":"","severity":"","rule":""}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","line":-1,"column":-1,"message":"negative"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","line":9999999999,"message":"huge line"}]}`))

	// Special characters
	f.Add([]byte(`{"findings":[{"file":"path/with spaces/file.tf","message":"has\nnewlines\tand\ttabs"}]}`))
	f.Add([]byte(`{"findings":[{"file":"日本語.tf","message":"unicode: 中文 русский العربية"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"quote\"inside"}]}`))

	// JSON escapes
	f.Add([]byte(`{"findings":[{"file":"a\\b.tf","message":"\\n\\t\\r"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"\u0000\u001f"}]}`))

	// Type mismatches (JSON allows different types)
	f.Add([]byte(`{"findings":[{"file":123,"line":"string","message":true}]}`))
	f.Add([]byte(`{"findings":[{"file":null,"line":null,"message":null}]}`))
	f.Add([]byte(`{"findings":"not an array"}`))
	f.Add([]byte(`{"findings":123}`))

	// Deeply nested
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"test","extra":{"nested":{"deep":{"value":1}}}}]}`))

	// All canonical severity values plus unknown
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"t","severity":"error"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"t","severity":"warning"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"t","severity":"info"}]}`))
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"t","severity":"critical"}]}`))  // Unknown
	f.Add([]byte(`{"findings":[{"file":"a.tf","message":"t","severity":"UPPERCASE"}]}`)) // Case variation

	// Large findings array - exercises loop path under memory pressure
	f.Add([]byte(`{"findings":[{"file":"a.tf","line":1,"message":"m"},{"file":"a.tf","line":2,"message":"m"},{"file":"a.tf","line":3,"message":"m"},{"file":"a.tf","line":4,"message":"m"},{"file":"a.tf","line":5,"message":"m"},{"file":"a.tf","line":6,"message":"m"},{"file":"a.tf","line":7,"message":"m"},{"file":"a.tf","line":8,"message":"m"},{"file":"a.tf","line":9,"message":"m"},{"file":"a.tf","line":10,"message":"m"}]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse JSON the same way BashRule.Check does (lines 132-135 of bash_rule.go).
		// Note: the empty-stdout early return in Check is not exercised here since
		// we're testing the parsing path, not the exec path.
		var output bashRuleOutput
		if err := json.Unmarshal(data, &output); err != nil {
			// Invalid JSON is expected, just ensure no panic
			return
		}

		// Exercise the finding conversion logic (mirrors Check lines 137-155).
		// Uses "fuzz-rule" as a stand-in for r.name since we're not constructing
		// a real BashRule here.
		for _, finding := range output.Findings {
			ruleName := finding.Rule
			if ruleName == "" {
				ruleName = "fuzz-rule"
			}

			// Construct to exercise sdk.ParseSeverity and field assignment
			_ = sdk.Finding{
				Rule:    ruleName,
				Message: finding.Message,
				File:    finding.File,
				Location: sdk.Location{
					Filename:    finding.File,
					StartLine:   finding.Line,
					StartColumn: finding.Column,
					EndLine:     finding.Line,
					EndColumn:   finding.Column,
				},
				Severity: sdk.ParseSeverity(finding.Severity, sdk.SeverityWarning),
			}
		}
	})
}
