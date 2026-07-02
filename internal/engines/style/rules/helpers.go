// Package rules provides style rules for TerraTidy.
package rules

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// WholeFileEdit wraps a whole-file rewrite as a single [sdk.TextEdit] that
// covers the original content's byte range. Rules that rewrite the entire file
// (the dominant pattern currently) use this helper to migrate from the legacy
// []byte return to the byte-range [sdk.FixResult] contract without changing
// their algorithm.
//
// Returns nil when newContent is byte-identical to original (no-op fix),
// matching the prior contract where rules returned nil bytes to signal "no
// change". Otherwise returns a single [sdk.TextEdit] spanning the full original
// byte range with newContent as the replacement.
//
// A nil or empty original is treated as length 0 and produces an insertion at
// offset 0 — only valid when the on-disk file is also empty. Passing nil for a
// non-empty file would emit a zero-width insertion (Start=0, End=0) that does
// NOT qualify as a whole-file edit under the engine's apply path (which checks
// End == len(file.Bytes)); callers that want a whole-file rewrite must pass
// the actual file bytes as original.
//
// Exclusive-this-pass semantic: per [sdk.FixResult], when the engine collects a
// whole-file edit (Start == 0 && End == len(content)) in the same pass as
// narrow edits, the whole-file edit is applied alone and the narrow edits are
// discarded for that pass (they re-emit against the rewritten content on the
// next pass). Rules using this helper therefore suppress co-applied narrow
// edits in the same pass — this is intentional and matches the current
// one-fix-per-pass behavior.
func WholeFileEdit(original, newContent []byte) *sdk.FixResult {
	if bytes.Equal(original, newContent) {
		return nil
	}
	return &sdk.FixResult{
		Edits: []sdk.TextEdit{{
			Start:       0,
			End:         len(original),
			Replacement: newContent,
		}},
	}
}

// Case convention regexes for naming validation.
var (
	// snakeCaseRegex matches valid snake_case identifiers.
	snakeCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	// camelCaseRegex matches valid camelCase identifiers.
	camelCaseRegex = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	// kebabCaseRegex matches valid kebab-case identifiers.
	kebabCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// pascalCaseRegex matches valid PascalCase identifiers.
	pascalCaseRegex = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
)

// NamingCase represents supported naming conventions.
type NamingCase string

// Supported naming convention constants.
const (
	SnakeCase  NamingCase = "snake_case"
	CamelCase  NamingCase = "camelCase"
	KebabCase  NamingCase = "kebab-case"
	PascalCase NamingCase = "PascalCase"
	CustomCase NamingCase = "custom"
)

// GetOrderedAttrNames returns attribute names from hclsyntax sorted by line number.
func GetOrderedAttrNames(syntaxBody *hclsyntax.Body) []string {
	type attrPos struct {
		name string
		line int
	}

	attrs := make([]attrPos, 0, len(syntaxBody.Attributes))
	for name, attr := range syntaxBody.Attributes {
		attrs = append(attrs, attrPos{
			name: name,
			line: attr.Range().Start.Line,
		})
	}

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].line < attrs[j].line
	})

	result := make([]string, len(attrs))
	for i, a := range attrs {
		result[i] = a.name
	}
	return result
}

// FormatAndCleanBlankLines applies hclwrite.Format and removes leading/trailing blank lines inside blocks.
// It preserves internal blank lines for readability.
func FormatAndCleanBlankLines(content []byte) []byte {
	// First apply hclwrite.Format
	formatted := hclwrite.Format(content)

	// Only remove leading/trailing blank lines inside blocks, preserve internal ones
	lines := SplitLines(formatted)
	var result []byte

	// First pass: identify block boundaries
	blockStarts := make([]int, 0)
	blockEnds := make(map[int]int) // maps start line to end line

	for i, line := range lines {
		hasOpenBrace := strings.Contains(line, "{")
		hasCloseBrace := strings.Contains(line, "}")

		if hasOpenBrace && !hasCloseBrace {
			blockStarts = append(blockStarts, i)
		} else if hasCloseBrace && !hasOpenBrace {
			if len(blockStarts) > 0 {
				start := blockStarts[len(blockStarts)-1]
				blockStarts = blockStarts[:len(blockStarts)-1]
				blockEnds[start] = i
			}
		}
		// Single line blocks like `tags = {}` have no internal lines to process
	}

	// Second pass: remove only leading/trailing blank lines inside blocks
	// Mark lines to skip
	skipLines := make(map[int]bool)

	for startLine, endLine := range blockEnds {
		// Remove leading blank lines (lines immediately after opening brace)
		for i := startLine + 1; i < endLine; i++ {
			trimmed := strings.TrimSpace(lines[i])
			if len(trimmed) == 0 {
				skipLines[i] = true
			} else {
				break
			}
		}

		// Remove trailing blank lines (lines immediately before closing brace)
		for i := endLine - 1; i > startLine; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if len(trimmed) == 0 {
				skipLines[i] = true
			} else {
				break
			}
		}
	}

	// Build result
	for i, line := range lines {
		if skipLines[i] {
			continue
		}
		result = append(result, line...)
		result = append(result, '\n')
	}

	return result
}

// SplitLines splits content into lines.
func SplitLines(content []byte) []string {
	var lines []string
	start := 0
	for i, b := range content {
		if b == '\n' {
			lines = append(lines, string(content[start:i]))
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, string(content[start:]))
	}
	return lines
}

// TrimLeftWhitespace trims leading whitespace from a string.
func TrimLeftWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[i:]
		}
	}
	return ""
}

// CountBlankLinesBetween counts actual blank lines (not comments) between two line numbers.
// Line numbers are 1-indexed (HCL convention).
func CountBlankLinesBetween(lines []string, endLine, startLine int) int {
	blankCount := 0

	// Lines between endLine and startLine (exclusive of both)
	for lineNum := endLine + 1; lineNum < startLine; lineNum++ {
		if lineNum-1 >= len(lines) {
			continue
		}
		line := lines[lineNum-1] // Convert to 0-indexed
		trimmed := TrimLeftWhitespace(line)

		// Count as blank if empty or whitespace-only
		// Don't count comment lines as blank lines
		if len(trimmed) == 0 {
			blankCount++
		}
	}

	return blankCount
}

// HasCommentBetween checks if there's a comment line between two line numbers.
// Line numbers are 1-indexed (HCL convention).
func HasCommentBetween(lines []string, endLine, startLine int) bool {
	for lineNum := endLine + 1; lineNum < startLine; lineNum++ {
		if lineNum-1 >= len(lines) {
			continue
		}
		line := lines[lineNum-1]
		trimmed := strings.TrimSpace(line)

		// Check if line is a comment
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			return true
		}
	}
	return false
}

// BlockKey creates a unique key for a block based on type and labels.
func BlockKey(blockType string, labels []string) string {
	key := blockType
	for _, l := range labels {
		key += "." + l
	}
	return key
}

// FindAttribute finds an attribute by name in the attributes map.
func FindAttribute(attrs hclsyntax.Attributes, name string) *hclsyntax.Attribute {
	if attr, ok := attrs[name]; ok {
		return attr
	}
	return nil
}

// FindNestedBlock finds a nested block by type in the blocks slice.
func FindNestedBlock(blocks hclsyntax.Blocks, blockType string) *hclsyntax.Block {
	for _, b := range blocks {
		if b.Type == blockType {
			return b
		}
	}
	return nil
}

// IsSnakeCase checks if a string is valid snake_case.
func IsSnakeCase(s string) bool {
	return snakeCaseRegex.MatchString(s)
}

// IsCamelCase checks if a string is valid camelCase.
func IsCamelCase(s string) bool {
	return camelCaseRegex.MatchString(s)
}

// IsKebabCase checks if a string is valid kebab-case.
func IsKebabCase(s string) bool {
	return kebabCaseRegex.MatchString(s)
}

// IsPascalCase checks if a string is valid PascalCase.
func IsPascalCase(s string) bool {
	return pascalCaseRegex.MatchString(s)
}

// MatchesCustomPattern checks if a string matches a custom regex pattern.
func MatchesCustomPattern(s, pattern string) bool {
	if pattern == "" {
		return true
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

// ValidateNaming checks if a name matches the specified naming convention.
// Returns (isValid, caseName) where caseName is the human-readable convention name.
func ValidateNaming(name string, convention NamingCase, customPattern string) (bool, string) {
	switch convention {
	case CamelCase:
		return IsCamelCase(name), "camelCase"
	case KebabCase:
		return IsKebabCase(name), "kebab-case"
	case PascalCase:
		return IsPascalCase(name), "PascalCase"
	case CustomCase:
		if customPattern == "" {
			return true, "custom"
		}
		return MatchesCustomPattern(name, customPattern), "custom pattern"
	default:
		// Default to snake_case
		return IsSnakeCase(name), "snake_case"
	}
}

// GetNamingConventionFromConfig extracts naming convention settings from rule config.
// Returns (convention, customPattern).
func GetNamingConventionFromConfig(config map[string]any) (NamingCase, string) {
	if config == nil {
		return SnakeCase, ""
	}

	options, ok := config["options"].(map[string]any)
	if !ok {
		return SnakeCase, ""
	}

	convention := SnakeCase
	if caseStr, ok := options["case"].(string); ok {
		switch caseStr {
		case "snake_case":
			convention = SnakeCase
		case "camelCase":
			convention = CamelCase
		case "kebab-case":
			convention = KebabCase
		case "PascalCase":
			convention = PascalCase
		case "custom":
			convention = CustomCase
		default:
			convention = SnakeCase
		}
	}

	customPattern := ""
	if pattern, ok := options["pattern"].(string); ok {
		customPattern = pattern
	}

	return convention, customPattern
}

// GetAttributeOrderFromConfig extracts attribute ordering configuration from rule config.
// Returns a map of attribute name to position, and the default order if not configured.
func GetAttributeOrderFromConfig(config map[string]any, defaultOrder map[string]int) map[string]int {
	if config == nil {
		return defaultOrder
	}

	options, ok := config["options"].(map[string]any)
	if !ok {
		return defaultOrder
	}

	orderList, ok := options["order"].([]any)
	if !ok {
		return defaultOrder
	}

	// Build order map from the list
	customOrder := make(map[string]int)
	for i, item := range orderList {
		if name, ok := item.(string); ok {
			customOrder[name] = i + 1
		}
	}

	if len(customOrder) == 0 {
		return defaultOrder
	}

	return customOrder
}

// MatchBlockLabels checks if block labels match expected labels.
func MatchBlockLabels(labels, expectedLabels []string) bool {
	if len(labels) != len(expectedLabels) {
		return false
	}
	for i, l := range labels {
		if l != expectedLabels[i] {
			return false
		}
	}
	return true
}

// FindSyntaxBody finds the syntax body for a block matching the given type and labels.
func FindSyntaxBody(syntaxFile *hclsyntax.Body, blockType string, blockLabels []string) *hclsyntax.Body {
	for _, block := range syntaxFile.Blocks {
		if block.Type != blockType {
			continue
		}
		if MatchBlockLabels(block.Labels, blockLabels) {
			return block.Body
		}
	}
	return nil
}

// FindWriteBlock finds a block in hclwrite file matching the given type and labels.
func FindWriteBlock(writeFile *hclwrite.File, blockType string, blockLabels []string) *hclwrite.Block {
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}
		if MatchBlockLabels(block.Labels(), blockLabels) {
			return block
		}
	}
	return nil
}

// ParseBothFormats parses content with both hclsyntax (for positions) and hclwrite (for modifications).
func ParseBothFormats(content []byte, filePath string) (*hclsyntax.Body, *hclwrite.File, error) {
	// Parse with hclsyntax to get attribute ordering
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	syntaxBody, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil, nil
	}

	// Parse with hclwrite for modifications
	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	return syntaxBody, writeFile, nil
}
