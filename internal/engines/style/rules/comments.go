package rules

import (
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// CommentSyntaxRule ensures comments use # instead of //.
type CommentSyntaxRule struct{}

// Name returns the rule identifier.
func (r *CommentSyntaxRule) Name() string {
	return "style.comment-syntax"
}

// Description returns a human-readable description of the rule.
func (r *CommentSyntaxRule) Description() string {
	return "Ensures comments use # syntax instead of //"
}

// Check examines the file for // style comments.
func (r *CommentSyntaxRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	lines := SplitLines(content)

	// Pre-compute fix once for all findings in this file
	fixedContent := r.fixContent(content)
	fixResult := &sdk.FixResult{Content: fixedContent}

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Check for // style comments (but not in strings)
		if r.hasDoubleSlashComment(line) {
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: "Use # for comments instead of //",
				File:    ctx.File,
				Location: sdk.Location{
					Filename:    ctx.File,
					StartLine:   lineNum,
					StartColumn: 1,
					EndLine:     lineNum,
					EndColumn:   len(trimmed),
				},
				Severity: sdk.SeverityInfo,
				Fix:      fixResult,
			})
		}
	}

	return findings, nil
}

// hasDoubleSlashComment checks if a line has a // comment outside of strings
func (r *CommentSyntaxRule) hasDoubleSlashComment(line string) bool {
	inString := false
	stringChar := rune(0)

	for i := 0; i < len(line)-1; i++ {
		c := rune(line[i])
		next := rune(line[i+1])

		// Handle string boundaries
		if (c == '"' || c == '\'') && (i == 0 || line[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = c
			} else if c == stringChar {
				inString = false
			}
			continue
		}

		// Check for // outside strings
		if !inString && c == '/' && next == '/' {
			return true
		}
	}

	return false
}

func (r *CommentSyntaxRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return r.fixContent(content), nil
}

func (r *CommentSyntaxRule) fixContent(content []byte) []byte {
	lines := SplitLines(content)
	var result []string

	for _, line := range lines {
		fixed := r.fixLine(line)
		result = append(result, fixed)
	}

	return []byte(strings.Join(result, "\n") + "\n")
}

func (r *CommentSyntaxRule) fixLine(line string) string {
	// Only replace // with # when it's at the start of a comment (not inside strings)
	inString := false
	stringChar := rune(0)
	result := []byte(line)

	for i := 0; i < len(line)-1; i++ {
		c := rune(line[i])
		next := rune(line[i+1])

		if (c == '"' || c == '\'') && (i == 0 || line[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = c
			} else if c == stringChar {
				inString = false
			}
			continue
		}

		if !inString && c == '/' && next == '/' {
			// Replace // with #
			result[i] = '#'
			// Remove the second /
			result = append(result[:i+1], result[i+2:]...)
			break
		}
	}

	return string(result)
}

// Fix replaces // comments with # comments.
func (r *CommentSyntaxRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	return r.fixFile(ctx.File)
}

// NoTrailingWhitespaceRule ensures no trailing whitespace on lines.
type NoTrailingWhitespaceRule struct{}

// Name returns the rule identifier.
func (r *NoTrailingWhitespaceRule) Name() string {
	return "style.no-trailing-whitespace"
}

// Description returns a human-readable description of the rule.
func (r *NoTrailingWhitespaceRule) Description() string {
	return "Ensures no trailing whitespace on lines"
}

// trailingWhitespaceRegex matches trailing whitespace
var trailingWhitespaceRegex = regexp.MustCompile(`[ \t]+$`)

// Check examines the file for trailing whitespace.
func (r *NoTrailingWhitespaceRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	lines := SplitLines(content)

	// Pre-compute fix once for all findings in this file
	fixedContent := r.fixContent(content)
	fixResult := &sdk.FixResult{Content: fixedContent}

	for i, line := range lines {
		lineNum := i + 1

		if trailingWhitespaceRegex.MatchString(line) {
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: "Trailing whitespace",
				File:    ctx.File,
				Location: sdk.Location{
					Filename:    ctx.File,
					StartLine:   lineNum,
					StartColumn: len(strings.TrimRight(line, " \t")) + 1,
					EndLine:     lineNum,
					EndColumn:   len(line),
				},
				Severity: sdk.SeverityInfo,
				Fix:      fixResult,
			})
		}
	}

	return findings, nil
}

func (r *NoTrailingWhitespaceRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return r.fixContent(content), nil
}

func (r *NoTrailingWhitespaceRule) fixContent(content []byte) []byte {
	lines := SplitLines(content)
	var result []string

	for _, line := range lines {
		result = append(result, strings.TrimRight(line, " \t"))
	}

	return []byte(strings.Join(result, "\n") + "\n")
}

// Fix removes trailing whitespace from all lines.
func (r *NoTrailingWhitespaceRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	return r.fixFile(ctx.File)
}

// ConsistentQuotesRule ensures consistent quote style.
type ConsistentQuotesRule struct{}

// Name returns the rule identifier.
func (r *ConsistentQuotesRule) Name() string {
	return "style.consistent-quotes"
}

// Description returns a human-readable description of the rule.
func (r *ConsistentQuotesRule) Description() string {
	return "Ensures consistent use of double quotes (Terraform standard)"
}

// Check examines the file for inconsistent quote usage.
// Note: HCL/Terraform always uses double quotes, so this mainly catches
// issues where single quotes might appear in unexpected places.
func (r *ConsistentQuotesRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	lines := SplitLines(content)

	for i, line := range lines {
		lineNum := i + 1

		// Check for single-quoted strings used as values
		// This is technically invalid HCL, but might occur in malformed files
		if r.hasSingleQuotedValue(line) {
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: "Use double quotes for strings (Terraform convention)",
				File:    ctx.File,
				Location: sdk.Location{
					Filename:    ctx.File,
					StartLine:   lineNum,
					StartColumn: 1,
					EndLine:     lineNum,
					EndColumn:   len(line),
				},
				Severity: sdk.SeverityWarning,
			})
		}
	}

	return findings, nil
}

// hasSingleQuotedValue checks if a line has a single-quoted string value
func (r *ConsistentQuotesRule) hasSingleQuotedValue(line string) bool {
	// Look for patterns like: key = 'value' or key = ['value']
	// Skip comment lines
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
		return false
	}

	// Simple heuristic: look for = followed by single quote
	idx := strings.Index(line, "=")
	if idx == -1 {
		return false
	}

	afterEquals := strings.TrimSpace(line[idx+1:])
	// Check if value starts with single quote (not in heredoc or template)
	if strings.HasPrefix(afterEquals, "'") {
		return true
	}

	// Check for single quotes in lists/maps
	if strings.Contains(afterEquals, "['") || strings.Contains(afterEquals, "{'") {
		return true
	}

	return false
}

// Fix is a no-op for this rule as quote style fixing is complex.
func (r *ConsistentQuotesRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// NoConsecutiveBlankLinesRule ensures no more than one consecutive blank line.
type NoConsecutiveBlankLinesRule struct{}

// Name returns the rule identifier.
func (r *NoConsecutiveBlankLinesRule) Name() string {
	return "style.no-consecutive-blank-lines"
}

// Description returns a human-readable description of the rule.
func (r *NoConsecutiveBlankLinesRule) Description() string {
	return "Ensures no more than one consecutive blank line"
}

// Check examines the file for consecutive blank lines.
func (r *NoConsecutiveBlankLinesRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	lines := SplitLines(content)

	// Pre-compute fix once for all findings in this file
	fixedContent := r.fixContent(content)
	fixResult := &sdk.FixResult{Content: fixedContent}

	consecutiveBlank := 0

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if len(trimmed) == 0 {
			consecutiveBlank++
			if consecutiveBlank > 1 {
				findings = append(findings, sdk.Finding{
					Rule:    r.Name(),
					Message: "More than one consecutive blank line",
					File:    ctx.File,
					Location: sdk.Location{
						Filename:    ctx.File,
						StartLine:   lineNum,
						StartColumn: 1,
						EndLine:     lineNum,
						EndColumn:   1,
					},
					Severity: sdk.SeverityInfo,
					Fix:      fixResult,
				})
			}
		} else {
			consecutiveBlank = 0
		}
	}

	return findings, nil
}

func (r *NoConsecutiveBlankLinesRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return r.fixContent(content), nil
}

func (r *NoConsecutiveBlankLinesRule) fixContent(content []byte) []byte {
	lines := SplitLines(content)
	var result []string
	lastWasBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isBlank := len(trimmed) == 0

		if isBlank {
			if !lastWasBlank {
				result = append(result, line)
			}
			lastWasBlank = true
		} else {
			result = append(result, line)
			lastWasBlank = false
		}
	}

	return []byte(strings.Join(result, "\n") + "\n")
}

// Fix removes consecutive blank lines, keeping only one.
func (r *NoConsecutiveBlankLinesRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	return r.fixFile(ctx.File)
}
