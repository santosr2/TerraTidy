package output

import (
	"bytes"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// generateFuzzFindings creates a slice of findings from fuzz input bytes.
// Uses byte values to construct realistic Finding structs with edge cases.
func generateFuzzFindings(data []byte) []sdk.Finding {
	if len(data) == 0 {
		return nil
	}

	// First byte determines count (0-15 findings)
	count := int(data[0] & 0x0F)
	if count == 0 {
		return []sdk.Finding{}
	}

	findings := make([]sdk.Finding, 0, count)
	offset := 1

	for i := 0; i < count && offset < len(data); i++ {
		f := sdk.Finding{}

		// Use bytes to populate fields
		if offset < len(data) {
			// Severity from byte
			switch data[offset] % 4 {
			case 0:
				f.Severity = sdk.SeverityError
			case 1:
				f.Severity = sdk.SeverityWarning
			case 2:
				f.Severity = sdk.SeverityInfo
			case 3:
				f.Severity = "" // Unknown severity
			}
			offset++
		}

		// Rule name
		if offset < len(data) {
			ruleLen := int(data[offset]&0x1F) + 1 // 1-32 chars
			offset++
			if offset+ruleLen <= len(data) {
				f.Rule = string(data[offset : offset+ruleLen])
				offset += ruleLen
			} else if offset < len(data) {
				f.Rule = string(data[offset:])
				offset = len(data)
			}
		}

		// Message
		if offset < len(data) {
			msgLen := int(data[offset]&0x3F) + 1 // 1-64 chars
			offset++
			if offset+msgLen <= len(data) {
				f.Message = string(data[offset : offset+msgLen])
				offset += msgLen
			} else if offset < len(data) {
				f.Message = string(data[offset:])
				offset = len(data)
			}
		}

		// File path
		if offset < len(data) {
			fileLen := int(data[offset]&0x1F) + 1 // 1-32 chars
			offset++
			if offset+fileLen <= len(data) {
				f.File = string(data[offset : offset+fileLen])
				offset += fileLen
			} else if offset < len(data) {
				f.File = string(data[offset:])
				offset = len(data)
			}
		}

		// Location (4 bytes: startLine, startCol, endLine, endCol)
		if offset+4 <= len(data) {
			f.Location = sdk.Location{
				StartLine:   int(data[offset]),
				StartColumn: int(data[offset+1]),
				EndLine:     int(data[offset+2]),
				EndColumn:   int(data[offset+3]),
			}
			offset += 4
		}

		// Fix (1 byte determines if fixable)
		if offset < len(data) && data[offset]%2 == 0 {
			f.Fix = &sdk.FixResult{Content: []byte("fixed content")}
		}
		if offset < len(data) {
			offset++
		}

		findings = append(findings, f)
	}

	return findings
}

// FuzzOutputJSON tests the JSON formatter with arbitrary findings.
func FuzzOutputJSON(f *testing.F) {
	// Seed corpus with various patterns
	f.Add([]byte{})  // Empty
	f.Add([]byte{0}) // Zero findings
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	f.Add([]byte{3, 0, 1, 'a', 1, 'b', 1, 'c', 1, 1, 1, 1, 0, 1, 1, 'd', 1, 'e', 1, 'f', 2, 2, 2, 2, 1})
	// Special characters (HTML, JSON escapes)
	f.Add([]byte{1, 0, 5, '<', '>', '&', '"', '\'', 10, '\\', 'n', '\\', 't', '"', '<', '>', '\n', '\r', '\t', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Unicode
	f.Add([]byte{1, 0, 6, 0xe4, 0xb8, 0xad, 0xe6, 0x96, 0x87, 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Large line numbers (via byte 255)
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 255, 255, 255, 255, 0})
	// Zero line numbers
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 0, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer

		// Test pretty JSON
		formatter := &JSONFormatter{Pretty: true}
		_ = formatter.Format(findings, &buf)

		// Test compact JSON
		buf.Reset()
		formatter = &JSONFormatter{Pretty: false}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputSARIF tests the SARIF formatter with arbitrary findings.
func FuzzOutputSARIF(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	f.Add([]byte{5, 0, 1, 'a', 1, 'b', 1, 'c', 1, 1, 1, 1, 0})
	// Special chars for JSON
	f.Add([]byte{1, 0, 5, '"', '\\', '\n', '\r', '\t', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// File path with special chars
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 4, 't', 'e', 's', 't', 10, '/', 'p', 'a', 't', 'h', '/', 'a', ' ', 'b', '/', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer
		formatter := &SARIFFormatter{Version: "1.0.0"}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputJUnit tests the JUnit XML formatter with arbitrary findings.
func FuzzOutputJUnit(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// XML special chars
	f.Add([]byte{1, 0, 5, '<', '>', '&', '"', '\'', 10, '<', 's', 'c', 'r', 'i', 'p', 't', '>', '<', '/', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// CDATA edge cases
	f.Add([]byte{1, 0, 8, ']', ']', '>', '<', '!', '[', 'C', 'D', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer
		formatter := &JUnitFormatter{Version: "1.0.0"}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputText tests the text formatter with arbitrary findings.
func FuzzOutputText(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// ANSI escape sequences in content
	f.Add([]byte{1, 0, 7, 0x1b, '[', '3', '1', 'm', 'x', 0x1b, 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Unicode icons
	f.Add([]byte{1, 0, 4, 0xe2, 0x9c, 0x97, 'x', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer

		// Test with color
		formatter := &TextFormatter{Color: true}
		_ = formatter.Format(findings, &buf)

		// Test without color
		buf.Reset()
		formatter = &TextFormatter{Color: false}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputHTML tests the HTML formatter with arbitrary findings.
func FuzzOutputHTML(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// HTML injection attempts
	f.Add([]byte{1, 0, 10, '<', 's', 'c', 'r', 'i', 'p', 't', '>', 'x', '<', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	f.Add([]byte{1, 0, 6, '&', 'a', 'm', 'p', ';', 'x', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Style injection
	f.Add([]byte{1, 0, 10, '<', '/', 's', 't', 'y', 'l', 'e', '>', 'x', '<', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer
		formatter := &HTMLFormatter{Title: "Test Report", Version: "1.0.0"}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputTable tests the table formatter with arbitrary findings.
func FuzzOutputTable(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Very long file path (tests truncation)
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 4, 't', 'e', 's', 't', 31, 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 1, 1, 1, 1, 0})
	// Box drawing chars
	f.Add([]byte{1, 0, 4, 0xe2, 0x94, 0x80, 'x', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer

		// Test with color
		formatter := &TableFormatter{Color: true, Verbose: true}
		_ = formatter.Format(findings, &buf)

		// Test without color
		buf.Reset()
		formatter = &TableFormatter{Color: false, Verbose: false}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputMarkdown tests the markdown formatter with arbitrary findings.
func FuzzOutputMarkdown(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Markdown special chars
	f.Add([]byte{1, 0, 8, '*', '*', 'b', 'o', 'l', 'd', '*', '*', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	f.Add([]byte{1, 0, 5, '|', '-', '|', '-', '|', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Code block injection
	f.Add([]byte{1, 0, 6, '`', '`', '`', '\n', 'x', '`', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer
		formatter := &MarkdownFormatter{Title: "Test Report", Version: "1.0.0"}
		_ = formatter.Format(findings, &buf)
	})
}

// FuzzOutputGitHubActions tests the GitHub Actions formatter with arbitrary findings.
func FuzzOutputGitHubActions(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 4, 't', 'e', 's', 't', 5, 'h', 'e', 'l', 'l', 'o', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Workflow command injection attempts
	f.Add([]byte{1, 0, 10, ':', ':', 'e', 'r', 'r', 'o', 'r', ':', ':', 'x', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Percent encoding edge cases
	f.Add([]byte{1, 0, 5, '%', '0', 'A', '%', 'x', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})
	// Newlines in message
	f.Add([]byte{1, 0, 5, 'a', '\n', 'b', '\r', 'c', 4, 't', 'e', 's', 't', 4, 'a', '.', 't', 'f', 1, 1, 1, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		findings := generateFuzzFindings(data)
		var buf bytes.Buffer
		formatter := &GitHubActionsFormatter{}
		_ = formatter.Format(findings, &buf)
	})
}
