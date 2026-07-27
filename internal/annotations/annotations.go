// Package annotations provides suppression annotation parsing for TerraTidy engines.
// It supports style, lint, and policy engines with a common annotation syntax:
//
//   - # terratidy:ignore:<rule>      - suppress rule on next block
//   - # terratidy:ignore:<rule>      - suppress rule on same line (inline)
//   - # terratidy:ignore-file:<rule> - suppress rule for entire file
//
// Both # and // comment styles are supported.
package annotations

import (
	"regexp"
	"strings"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// Type indicates how a suppression applies.
type Type int

const (
	// NextBlock suppresses the rule for the block on the next non-comment line.
	NextBlock Type = iota
	// Inline suppresses the rule for the block on the same line.
	Inline
	// File suppresses the rule for the entire file.
	File
)

// Suppression represents a single suppression annotation.
type Suppression struct {
	Rule       string // Rule name to suppress (e.g., "style.resource-name-convention")
	Line       int    // Line number where the annotation appears (1-based)
	TargetLine int    // Line number the suppression applies to (for NextBlock/Inline)
	Type       Type   // Type of suppression
}

// Regex patterns for suppression annotations.
var (
	// Matches: # terratidy:ignore:<rule> or // terratidy:ignore:<rule>
	ignorePattern = regexp.MustCompile(`(?:#|//)\s*terratidy:ignore:(\S+)`)
	// Matches: # terratidy:ignore-file:<rule> or // terratidy:ignore-file:<rule>
	ignoreFilePattern = regexp.MustCompile(`(?:#|//)\s*terratidy:ignore-file:(\S+)`)
)

// Parse extracts all suppression annotations from HCL file content.
func Parse(content []byte) []Suppression {
	lines := strings.Split(string(content), "\n")
	var suppressions []Suppression

	for lineNum, line := range lines {
		lineNumber := lineNum + 1 // Convert to 1-based

		// Check for file-level suppression
		if matches := ignoreFilePattern.FindStringSubmatch(line); matches != nil {
			suppressions = append(suppressions, Suppression{
				Rule: matches[1],
				Line: lineNumber,
				Type: File,
			})
			continue
		}

		// Check for ignore annotation (could be inline or next-block)
		if matches := ignorePattern.FindStringSubmatch(line); matches != nil {
			rule := matches[1]

			// Determine if this is inline or next-block
			// Inline: has code before the comment on the same line
			// Next-block: comment is the only content on the line
			trimmedLine := strings.TrimSpace(line)
			commentStart := strings.Index(trimmedLine, "#")
			if commentStart == -1 {
				commentStart = strings.Index(trimmedLine, "//")
			}

			if commentStart > 0 {
				// Has code before comment - inline suppression
				suppressions = append(suppressions, Suppression{
					Rule:       rule,
					Line:       lineNumber,
					TargetLine: lineNumber,
					Type:       Inline,
				})
			} else {
				// Comment at start of line - next-block suppression
				targetLine := findNextCodeLine(lines, lineNum+1)
				suppressions = append(suppressions, Suppression{
					Rule:       rule,
					Line:       lineNumber,
					TargetLine: targetLine,
					Type:       NextBlock,
				})
			}
		}
	}

	return suppressions
}

// findNextCodeLine finds the line number of the next non-blank, non-comment line.
// Returns -1 if no such line exists.
func findNextCodeLine(lines []string, startIdx int) int {
	for i := startIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		// Skip comment-only lines
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		return i + 1 // Convert to 1-based line number
	}
	return -1
}

// FilterFindings removes findings that are suppressed by annotations.
func FilterFindings(findings []sdk.Finding, suppressions []Suppression) []sdk.Finding {
	if len(suppressions) == 0 {
		return findings
	}

	var filtered []sdk.Finding
	for _, f := range findings {
		if !IsSuppressed(f, suppressions) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// IsSuppressed checks if a finding should be suppressed based on annotations.
func IsSuppressed(finding sdk.Finding, suppressions []Suppression) bool {
	for _, s := range suppressions {
		if !RuleMatches(finding.Rule, s.Rule) {
			continue
		}

		switch s.Type {
		case File:
			return true
		case NextBlock, Inline:
			if finding.Location.StartLine == s.TargetLine {
				return true
			}
		}
	}
	return false
}

// RuleMatches checks if a finding rule matches a suppression rule.
// Supports exact matches and wildcards (e.g., "style.*" matches all style rules).
func RuleMatches(findingRule, suppressionRule string) bool {
	if findingRule == suppressionRule {
		return true
	}

	// Wildcard match (e.g., "style.*", "lint.*", "policy.*")
	if prefix, ok := strings.CutSuffix(suppressionRule, ".*"); ok {
		return strings.HasPrefix(findingRule, prefix+".")
	}

	return false
}
