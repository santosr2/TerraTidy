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
	return "Ensures full-line comments use # syntax instead of // (trailing // after a value is not flagged)"
}

// Check examines the file for // style comments.
func (r *CommentSyntaxRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	lines := SplitLines(content)

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
			})
		}
	}

	return findings, nil
}

// hasDoubleSlashComment reports whether a line is a full-line `//` comment (after
// trimming leading whitespace). Lines starting with `#` are never flagged, even if
// their body contains `//` (e.g. URLs like `# https://example.com/path`). Trailing
// `//` after a value (e.g. `key = "x" // note`) is not flagged either; the rule's
// scope is full-line comments only.
func (r *CommentSyntaxRule) hasDoubleSlashComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "//")
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

// fixLine converts a full-line `//` comment to `#` (preserving leading whitespace).
// Lines that are not full-line `//` comments are returned unchanged — `#` comments
// containing `//` (e.g. URLs) and inline `//` after a value both pass through.
//
// hasDoubleSlashComment guarantees the first `//` in the raw line is the comment
// delimiter (the trimmed line starts with `//`, so leading whitespace is the only
// thing that can precede it), so strings.Index is safe to use here without a guard.
func (r *CommentSyntaxRule) fixLine(line string) string {
	if !r.hasDoubleSlashComment(line) {
		return line
	}
	idx := strings.Index(line, "//")
	return line[:idx] + "#" + line[idx+2:]
}

// Fix replaces // comments with # comments.
func (r *CommentSyntaxRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	return WholeFileEdit(content, r.fixContent(content)), nil
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
			})
		}
	}

	return findings, nil
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
func (r *NoTrailingWhitespaceRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	return WholeFileEdit(content, r.fixContent(content)), nil
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
				})
			}
		} else {
			consecutiveBlank = 0
		}
	}

	return findings, nil
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
func (r *NoConsecutiveBlankLinesRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	return WholeFileEdit(content, r.fixContent(content)), nil
}
