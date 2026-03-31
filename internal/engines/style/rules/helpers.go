// Package rules provides style rules for TerraTidy.
package rules

import (
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

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

// getExprTokensWithTrailingComment extracts expression tokens and any trailing inline comment.
// This preserves comments like: description = "foo" # this comment
func getExprTokensWithTrailingComment(attr *hclwrite.Attribute) hclwrite.Tokens {
	// Get expression tokens (the value)
	exprTokens := attr.Expr().BuildTokens(nil)

	// Get full attribute tokens to find trailing comments
	fullTokens := attr.BuildTokens(nil)

	// The full tokens structure is: name = expr [comment] [newline]
	// We need to find comment tokens after the expression
	var trailingTokens hclwrite.Tokens
	for i := len(fullTokens) - 1; i >= 0; i-- {
		tok := fullTokens[i]
		// Include comment tokens
		if tok.Type.String() == "TokenComment" {
			trailingTokens = append(hclwrite.Tokens{tok}, trailingTokens...)
		} else if tok.Type.String() == "TokenNewline" {
			// Skip newlines at the very end
			continue
		} else {
			// Stop when we hit something that's not a comment or newline
			break
		}
	}

	// Append trailing tokens (comments) to expression tokens
	if len(trailingTokens) > 0 {
		result := make(hclwrite.Tokens, len(exprTokens)+len(trailingTokens))
		copy(result, exprTokens)
		copy(result[len(exprTokens):], trailingTokens)
		return result
	}

	return exprTokens
}

// ReorderBlockAttrs reorders attributes in a block according to the specified order.
// firstAttrs are placed at the start, lastAttrs at the end, others maintain relative order.
// orderedNames should be the original order of attributes (from hclsyntax parsing).
func ReorderBlockAttrs(body *hclwrite.Body, orderedNames, firstAttrs, lastAttrs []string) {
	if len(orderedNames) == 0 {
		return
	}

	// Build sets for quick lookup
	firstSet := make(map[string]bool)
	for _, name := range firstAttrs {
		firstSet[name] = true
	}
	lastSet := make(map[string]bool)
	for _, name := range lastAttrs {
		lastSet[name] = true
	}

	// Collect attribute info preserving expression tokens AND trailing comments
	attrTokens := make(map[string]hclwrite.Tokens)
	for name, attr := range body.Attributes() {
		attrTokens[name] = getExprTokensWithTrailingComment(attr)
	}

	// Categorize attribute names
	var first, middle, last []string
	for _, name := range orderedNames {
		if _, exists := attrTokens[name]; !exists {
			continue // Skip if attribute doesn't exist in hclwrite body
		}
		if firstSet[name] {
			first = append(first, name)
		} else if lastSet[name] {
			last = append(last, name)
		} else {
			middle = append(middle, name)
		}
	}

	// Sort first and last attributes by priority order
	sortByPriority := func(names []string, priority []string) {
		prioMap := make(map[string]int)
		for i, name := range priority {
			prioMap[name] = i
		}
		sort.SliceStable(names, func(i, j int) bool {
			pi, oki := prioMap[names[i]]
			pj, okj := prioMap[names[j]]
			if oki && okj {
				return pi < pj
			}
			if oki {
				return true
			}
			return !okj
		})
	}

	sortByPriority(first, firstAttrs)
	sortByPriority(last, lastAttrs)

	// Remove all attributes
	for name := range attrTokens {
		body.RemoveAttribute(name)
	}

	// Re-add in the correct order
	for _, name := range first {
		body.SetAttributeRaw(name, attrTokens[name])
	}
	for _, name := range middle {
		body.SetAttributeRaw(name, attrTokens[name])
	}
	for _, name := range last {
		body.SetAttributeRaw(name, attrTokens[name])
	}
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

// ReorderTopLevelBlocks reorders top-level blocks in a file according to best practices:
// 1. terraform blocks first
// 2. provider blocks second
// 3. variable blocks
// 4. locals blocks
// 5. data blocks
// 6. resource blocks
// 7. module blocks
// 8. output blocks
func ReorderTopLevelBlocks(writeFile *hclwrite.File) []byte {
	blocks := writeFile.Body().Blocks()
	if len(blocks) == 0 {
		return writeFile.Bytes()
	}

	// Categorize blocks by type
	var terraformBlocks []*hclwrite.Block
	var providerBlocks []*hclwrite.Block
	var variableBlocks []*hclwrite.Block
	var localsBlocks []*hclwrite.Block
	var dataBlocks []*hclwrite.Block
	var resourceBlocks []*hclwrite.Block
	var moduleBlocks []*hclwrite.Block
	var outputBlocks []*hclwrite.Block
	var otherBlocks []*hclwrite.Block

	for _, block := range blocks {
		switch block.Type() {
		case "terraform":
			terraformBlocks = append(terraformBlocks, block)
		case "provider":
			providerBlocks = append(providerBlocks, block)
		case "variable":
			variableBlocks = append(variableBlocks, block)
		case "locals":
			localsBlocks = append(localsBlocks, block)
		case "data":
			dataBlocks = append(dataBlocks, block)
		case "resource":
			resourceBlocks = append(resourceBlocks, block)
		case "module":
			moduleBlocks = append(moduleBlocks, block)
		case "output":
			outputBlocks = append(outputBlocks, block)
		default:
			otherBlocks = append(otherBlocks, block)
		}
	}

	// Clear all blocks from the body
	for _, block := range blocks {
		writeFile.Body().RemoveBlock(block)
	}

	// Re-add blocks in the desired order
	addBlocksWithSpacing(writeFile.Body(), terraformBlocks)
	addBlocksWithSpacing(writeFile.Body(), providerBlocks)
	addBlocksWithSpacing(writeFile.Body(), variableBlocks)
	addBlocksWithSpacing(writeFile.Body(), localsBlocks)
	addBlocksWithSpacing(writeFile.Body(), dataBlocks)
	addBlocksWithSpacing(writeFile.Body(), resourceBlocks)
	addBlocksWithSpacing(writeFile.Body(), moduleBlocks)
	addBlocksWithSpacing(writeFile.Body(), outputBlocks)
	addBlocksWithSpacing(writeFile.Body(), otherBlocks)

	return FormatAndCleanBlankLines(writeFile.Bytes())
}

// addBlocksWithSpacing adds blocks to a body, preserving their content including inline comments.
func addBlocksWithSpacing(body *hclwrite.Body, blocks []*hclwrite.Block) {
	for _, block := range blocks {
		newBlock := body.AppendNewBlock(block.Type(), block.Labels())
		// Copy attributes with inline comments preserved
		for name, attr := range block.Body().Attributes() {
			newBlock.Body().SetAttributeRaw(name, getExprTokensWithTrailingComment(attr))
		}
		// Copy nested blocks
		for _, nested := range block.Body().Blocks() {
			copyNestedBlock(newBlock.Body(), nested)
		}
		body.AppendNewline()
	}
}

// copyNestedBlock recursively copies a nested block to a new body, preserving inline comments.
func copyNestedBlock(body *hclwrite.Body, block *hclwrite.Block) {
	newBlock := body.AppendNewBlock(block.Type(), block.Labels())
	for name, attr := range block.Body().Attributes() {
		newBlock.Body().SetAttributeRaw(name, getExprTokensWithTrailingComment(attr))
	}
	for _, nested := range block.Body().Blocks() {
		copyNestedBlock(newBlock.Body(), nested)
	}
}
