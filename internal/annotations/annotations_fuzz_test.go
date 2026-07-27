package annotations

import (
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// FuzzAnnotationParse tests the Parse function with arbitrary comment strings.
// It exercises the suppression annotation regex parsing and ensures no panics
// occur with malformed or edge-case input.
func FuzzAnnotationParse(f *testing.F) {
	// Valid file-level suppressions
	f.Add([]byte(`# terratidy:ignore-file:style.rule`))
	f.Add([]byte(`// terratidy:ignore-file:style.rule`))
	f.Add([]byte(`# terratidy:ignore-file:lint.deprecated-resource`))
	f.Add([]byte(`# terratidy:ignore-file:policy.require-tags`))

	// Valid next-block suppressions
	f.Add([]byte("# terratidy:ignore:style.resource-name-convention\nresource \"aws_instance\" \"test\" {}"))
	f.Add([]byte("// terratidy:ignore:style.resource-name-convention\nresource \"aws_instance\" \"test\" {}"))
	f.Add([]byte("  # terratidy:ignore:style.rule\nresource \"x\" \"y\" {}"))

	// Valid inline suppressions
	f.Add([]byte(`resource "aws_instance" "test" {} # terratidy:ignore:style.rule`))
	f.Add([]byte(`resource "aws_instance" "test" {} // terratidy:ignore:style.rule`))
	f.Add([]byte(`output "value" { } # terratidy:ignore:lint.unused`))

	// Wildcard rules
	f.Add([]byte(`# terratidy:ignore-file:style.*`))
	f.Add([]byte(`# terratidy:ignore:lint.*`))
	f.Add([]byte(`# terratidy:ignore:policy.*`))

	// Multiple suppressions in one file
	f.Add([]byte(`# terratidy:ignore-file:style.*
# terratidy:ignore:lint.rule
resource "x" "y" {} # terratidy:ignore:policy.tags`))

	// Empty and whitespace
	f.Add([]byte(``))
	f.Add([]byte("\n"))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte("   "))
	f.Add([]byte("\t\t\t"))
	f.Add([]byte("   \n   \n   "))

	// Regular comments (not suppressions)
	f.Add([]byte(`# just a regular comment`))
	f.Add([]byte(`// another regular comment`))
	f.Add([]byte(`# terratidy is cool but this isn't an annotation`))

	// Partial/malformed annotations
	f.Add([]byte(`# terratidy:ignore`))
	f.Add([]byte(`# terratidy:ignore:`))
	f.Add([]byte(`# terratidy:ignore-file`))
	f.Add([]byte(`# terratidy:ignore-file:`))
	f.Add([]byte(`# terratidy:`))
	f.Add([]byte(`# terratidy`))
	f.Add([]byte(`#terratidy:ignore:rule`))
	f.Add([]byte(`//terratidy:ignore:rule`))
	f.Add([]byte(`#  terratidy:ignore:rule`))

	// Case variations (regex is case-sensitive)
	f.Add([]byte(`# TERRATIDY:ignore:rule`))
	f.Add([]byte(`# TerraTidy:ignore:rule`))
	f.Add([]byte(`# terratidy:IGNORE:rule`))
	f.Add([]byte(`# terratidy:ignore-FILE:rule`))

	// Special characters in rule names
	f.Add([]byte(`# terratidy:ignore:style.rule-with-dashes`))
	f.Add([]byte(`# terratidy:ignore:style.rule_with_underscores`))
	f.Add([]byte(`# terratidy:ignore:style.rule.with.dots`))
	f.Add([]byte(`# terratidy:ignore:123.456`))
	f.Add([]byte(`# terratidy:ignore:*`))

	// Unicode in comments
	f.Add([]byte(`# terratidy:ignore:style.日本語`))
	f.Add([]byte(`# terratidy:ignore:中文规则`))
	f.Add([]byte(`# terratidy:ignore:правило`))
	f.Add([]byte(`# terratidy:ignore:مقاعد`))
	f.Add([]byte(`# terratidy:ignore:style.émoji🎉`))

	// Very long rule names
	f.Add([]byte(`# terratidy:ignore:style.very-long-rule-name-that-goes-on-and-on-and-on-and-on-and-on`))
	f.Add([]byte(`# terratidy:ignore-file:this.is.an.extremely.deeply.nested.rule.name.with.many.segments`))

	// Newline variations
	f.Add([]byte("# terratidy:ignore:rule\r\nresource {}"))
	f.Add([]byte("# terratidy:ignore:rule\rresource {}"))
	f.Add([]byte("line1\r\n# terratidy:ignore:rule\r\nline3"))

	// Many annotations
	f.Add([]byte(`# terratidy:ignore-file:a
# terratidy:ignore-file:b
# terratidy:ignore-file:c
# terratidy:ignore:d
# terratidy:ignore:e
# terratidy:ignore:f
resource "x" "y" {}
resource "a" "b" {} # terratidy:ignore:g`))

	// Annotation at end of file with no following code
	f.Add([]byte(`# terratidy:ignore:style.rule`))
	f.Add([]byte("# terratidy:ignore:style.rule\n"))
	f.Add([]byte("# terratidy:ignore:style.rule\n# another comment"))
	f.Add([]byte("# terratidy:ignore:style.rule\n\n\n"))

	// Binary garbage
	f.Add([]byte{0x00, 0x01, 0x02, 0x03})
	f.Add([]byte{0xff, 0xfe, 0xfd})
	f.Add([]byte("normal text\x00with null\x00bytes"))

	// Long lines
	longLine := make([]byte, 10000)
	for i := range longLine {
		longLine[i] = 'x'
	}
	f.Add(longLine)
	f.Add(append([]byte("# terratidy:ignore:style.rule # "), longLine...))

	// HCL-like content with embedded annotations
	f.Add([]byte(`
terraform {
  required_version = ">= 1.0"
}

# terratidy:ignore:style.resource-name-convention
resource "aws_instance" "MyServer" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}

variable "name" {} # terratidy:ignore:lint.missing-description

# terratidy:ignore-file:policy.require-tags
`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse should never panic regardless of input
		suppressions := Parse(data)

		// Exercise the result to ensure returned data is valid
		for _, s := range suppressions {
			_ = s.Rule
			_ = s.Line
			_ = s.TargetLine
			_ = s.Type
		}
	})
}

// FuzzAnnotationFilter tests FilterFindings and IsSuppressed with fuzz input.
// It constructs findings and suppressions from arbitrary bytes and verifies
// no panics occur during filtering operations.
func FuzzAnnotationFilter(f *testing.F) {
	// Seeds that exercise different suppression types and matching
	f.Add([]byte{0x01, 0x05, 's', 't', 'y', 'l', 'e', 0x00, 0x05})
	f.Add([]byte{0x03, 0x05, 'l', 'i', 'n', 't', '.', 0x01, 0x10, 0x02, 0x20})
	f.Add([]byte{0x02, 0x06, 'p', 'o', 'l', 'i', 'c', 'y', 0x00, 0x01})

	// Edge cases
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xff})
	f.Add([]byte{0x01, 0x00})
	f.Add([]byte{0x01, 0xff})
	f.Add([]byte{0x0F, 0x01, 'x'})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}

		// Generate findings and suppressions from fuzz input
		findings := generateFuzzFindings(data)
		suppressions := generateFuzzSuppressions(data)

		// FilterFindings should never panic
		filtered := FilterFindings(findings, suppressions)
		_ = len(filtered)

		// IsSuppressed should never panic for any finding
		for _, finding := range findings {
			_ = IsSuppressed(finding, suppressions)
		}
	})
}

// FuzzRuleMatches tests the RuleMatches function with arbitrary rule strings.
func FuzzRuleMatches(f *testing.F) {
	// Valid rule patterns
	f.Add("style.resource-name-convention", "style.resource-name-convention")
	f.Add("style.resource-name-convention", "style.*")
	f.Add("lint.deprecated", "lint.*")
	f.Add("policy.require-tags", "policy.*")

	// Non-matching patterns
	f.Add("style.rule", "lint.rule")
	f.Add("style.rule", "style.other")
	f.Add("style", "style.*")

	// Edge cases
	f.Add("", "")
	f.Add("", "style.*")
	f.Add("style.rule", "")
	f.Add("*", "*")
	f.Add(".*", ".*")
	f.Add("style.*", "style.*")

	// Unicode
	f.Add("style.日本語", "style.*")
	f.Add("style.émoji", "style.émoji")

	// Long strings
	f.Add("very.long.rule.name.with.many.segments", "very.*")
	f.Add("a.b.c.d.e.f.g.h.i.j", "a.*")

	f.Fuzz(func(t *testing.T, findingRule, suppressionRule string) {
		// RuleMatches should never panic
		_ = RuleMatches(findingRule, suppressionRule)
	})
}

// generateFuzzFindings creates sdk.Finding slice from fuzz input bytes.
// Intentionally omits Severity, Message, and File fields because IsSuppressed
// only reads Rule and Location.StartLine. Similar to output_fuzz_test.go but
// simplified to focus on the fields that matter for suppression matching.
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

		// Rule name from bytes
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

		// Line number from byte
		if offset < len(data) {
			f.Location.StartLine = int(data[offset])
			offset++
		}

		findings = append(findings, f)
	}

	return findings
}

// generateFuzzSuppressions creates Suppression slice from fuzz input bytes.
func generateFuzzSuppressions(data []byte) []Suppression {
	if len(data) < 2 {
		return nil
	}

	// Use second byte for suppression count
	count := int(data[1] & 0x0F)
	if count == 0 {
		return []Suppression{}
	}

	suppressions := make([]Suppression, 0, count)
	offset := 2

	for i := 0; i < count && offset < len(data); i++ {
		s := Suppression{}

		// Type from byte
		if offset < len(data) {
			s.Type = Type(data[offset] % 3) // NextBlock, Inline, or File
			offset++
		}

		// Rule name
		if offset < len(data) {
			ruleLen := int(data[offset]&0x1F) + 1
			offset++
			if offset+ruleLen <= len(data) {
				s.Rule = string(data[offset : offset+ruleLen])
				offset += ruleLen
			} else if offset < len(data) {
				s.Rule = string(data[offset:])
				offset = len(data)
			}
		}

		// Target line
		if offset < len(data) {
			s.TargetLine = int(data[offset])
			offset++
		}

		suppressions = append(suppressions, s)
	}

	return suppressions
}
