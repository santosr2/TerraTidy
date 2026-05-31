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
// (the dominant pattern today) use this helper to migrate from the legacy
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
// edits in the same pass — this is intentional and matches today's
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

// AttrRegion represents an attribute with its leading comments.
type AttrRegion struct {
	Name           string
	LeadingComment string   // Comments on lines before the attribute
	Lines          []string // The attribute line(s) including trailing comment
	StartLine      int      // 1-indexed line number where region starts
	EndLine        int      // 1-indexed line number where region ends
}

// collectLeadingComments walks backwards from startLine-1 (exclusive) down to searchStart
// (inclusive), collecting consecutive `#` and `//` comment lines. Blank lines are passed
// through without being collected and without terminating the scan, matching long-standing
// behavior across other style rules. Non-comment content terminates the scan.
// Returns comment lines in source order (top-down).
//
// Note on blank-line passthrough: a comment separated from the next region by one or more
// blank lines is still claimed as that region's leading comment. The motivating reason is
// damage control: the reassembly path emits only content attached to a region or captured
// by collectOrphanLines, so a stricter "stop on blank" rule would silently drop comments
// the author placed visually above a target region. Section-header comments above blank
// lines therefore travel with the following region; users who want them stationary should
// place them outside the block or attach them inline.
func collectLeadingComments(lines []string, startLine, searchStart int) []string {
	var leading []string
	for lineNum := startLine - 1; lineNum >= searchStart; lineNum-- {
		if lineNum-1 >= len(lines) || lineNum-1 < 0 {
			continue
		}
		line := lines[lineNum-1]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			leading = append([]string{line}, leading...)
			continue
		}
		break
	}
	return leading
}

// ExtractAttrRegions extracts attribute regions (including leading comments) from content.
// syntaxBody provides accurate line numbers for attributes.
func ExtractAttrRegions(content []byte, syntaxBody *hclsyntax.Body) map[string]*AttrRegion {
	lines := SplitLines(content)
	regions := make(map[string]*AttrRegion)

	type attrPos struct {
		name      string
		startLine int
		endLine   int
	}
	var attrs []attrPos
	for name, attr := range syntaxBody.Attributes {
		attrs = append(attrs, attrPos{
			name:      name,
			startLine: attr.Range().Start.Line,
			endLine:   attr.Range().End.Line,
		})
	}
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].startLine < attrs[j].startLine
	})

	for i, attr := range attrs {
		region := &AttrRegion{
			Name:      attr.name,
			StartLine: attr.startLine,
			EndLine:   attr.endLine,
		}

		// Stop the comment scan at the previous attribute's end line (or block start).
		searchStart := 1
		if i > 0 {
			searchStart = attrs[i-1].endLine + 1
		}

		leadingCommentLines := collectLeadingComments(lines, attr.startLine, searchStart)
		if len(leadingCommentLines) > 0 {
			region.LeadingComment = strings.Join(leadingCommentLines, "\n") + "\n"
			region.StartLine = attr.startLine - len(leadingCommentLines)
		}

		for lineNum := attr.startLine; lineNum <= attr.endLine && lineNum-1 < len(lines); lineNum++ {
			region.Lines = append(region.Lines, lines[lineNum-1])
		}

		regions[attr.name] = region
	}

	return regions
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

// markConsumedAttrLines marks the source lines covered by the given attribute regions
// (including their captured leading comments) so the caller can later identify content
// that belongs to no region. Safe to call alongside markConsumedBlockLines on the same
// map: well-formed HCL guarantees attr and block ranges do not overlap, but writing the
// same true value twice for a line is idempotent regardless.
func markConsumedAttrLines(regions map[string]*AttrRegion, consumed map[int]bool) {
	for _, region := range regions {
		if region == nil {
			continue
		}
		for line := region.StartLine; line <= region.EndLine; line++ {
			consumed[line] = true
		}
	}
}

// markConsumedBlockLines marks the source lines covered by the given block regions.
func markConsumedBlockLines(regions []*BlockRegion, consumed map[int]bool) {
	for _, region := range regions {
		if region == nil {
			continue
		}
		for line := region.StartLine; line <= region.EndLine; line++ {
			consumed[line] = true
		}
	}
}

// collectOrphanLines returns non-blank lines in the (blockStart, blockEnd) interior whose
// source line number is not in `consumed`. These are comments or other content that no
// region claimed as part of its leading-comment block. Preserving them protects against
// silent data loss when reassembling a reordered body.
//
// Position semantics (trailing-orphan): callers emit these lines after all regions and
// before the closing brace. In real pipelines orphans only arise when a comment appears
// after the last region with no following sibling — comments interleaved between regions
// are claimed as leading-comments of the region that follows them by collectLeadingComments.
func collectOrphanLines(lines []string, blockStartLine, blockEndLine int, consumed map[int]bool) []string {
	var orphans []string
	for line := blockStartLine + 1; line < blockEndLine; line++ {
		if consumed[line] {
			continue
		}
		if line-1 < 0 || line-1 >= len(lines) {
			continue
		}
		if strings.TrimSpace(lines[line-1]) == "" {
			continue
		}
		orphans = append(orphans, lines[line-1])
	}
	return orphans
}

// ReorderBlockAttrsPreservingComments reorders attributes while preserving leading comments.
// Non-blank lines inside the block that no region claimed (orphan comments after the last
// attribute, for example) are emitted before the closing brace so they are never silently lost.
func ReorderBlockAttrsPreservingComments(
	content []byte,
	syntaxBody *hclsyntax.Body,
	blockStartLine, blockEndLine int,
	orderedNames, firstAttrs, lastAttrs []string,
) []byte {
	if len(orderedNames) == 0 {
		return content
	}

	regions := ExtractAttrRegions(content, syntaxBody)

	firstSet := make(map[string]bool)
	for _, name := range firstAttrs {
		firstSet[name] = true
	}
	lastSet := make(map[string]bool)
	for _, name := range lastAttrs {
		lastSet[name] = true
	}

	var first, middle, last []string
	for _, name := range orderedNames {
		if _, exists := regions[name]; !exists {
			continue
		}
		if firstSet[name] {
			first = append(first, name)
		} else if lastSet[name] {
			last = append(last, name)
		} else {
			middle = append(middle, name)
		}
	}

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

	lines := SplitLines(content)
	consumed := make(map[int]bool)
	markConsumedAttrLines(regions, consumed)
	orphans := collectOrphanLines(lines, blockStartLine, blockEndLine, consumed)

	var newBlockContent []string

	if blockStartLine-1 < len(lines) {
		newBlockContent = append(newBlockContent, lines[blockStartLine-1])
	}

	reorderedNames := append(append(first, middle...), last...)
	for _, name := range reorderedNames {
		region := regions[name]
		if region == nil {
			continue
		}
		if region.LeadingComment != "" {
			commentLines := strings.Split(strings.TrimSuffix(region.LeadingComment, "\n"), "\n")
			newBlockContent = append(newBlockContent, commentLines...)
		}
		newBlockContent = append(newBlockContent, region.Lines...)
	}

	newBlockContent = append(newBlockContent, orphans...)

	if blockEndLine-1 < len(lines) {
		newBlockContent = append(newBlockContent, lines[blockEndLine-1])
	}

	var result []string
	for i := 0; i < blockStartLine-1 && i < len(lines); i++ {
		result = append(result, lines[i])
	}
	result = append(result, newBlockContent...)
	for i := blockEndLine; i < len(lines); i++ {
		result = append(result, lines[i])
	}

	return []byte(strings.Join(result, "\n") + "\n")
}

// BlockRegion represents a nested block with its leading comments.
type BlockRegion struct {
	Type           string
	Labels         []string
	LeadingComment string   // Comments on lines before the block (may be empty)
	Lines          []string // The block's source lines (opening header through closing brace)
	StartLine      int      // 1-indexed line number where region starts (includes leading comments)
	EndLine        int      // 1-indexed line number where region ends
}

// ExtractBlockRegions extracts nested block regions (including leading comments) from content.
// syntaxBody provides accurate line numbers for blocks. Regions are returned in source order.
//
// Leading-comment scan stops at the closest prior body item (attribute or block), so blocks
// sandwiched between attributes pick up only comments that belong to them.
func ExtractBlockRegions(content []byte, syntaxBody *hclsyntax.Body) []*BlockRegion {
	if len(syntaxBody.Blocks) == 0 {
		return nil
	}
	lines := SplitLines(content)

	// Collect all body item end lines so each block can find its immediate prior boundary
	// regardless of body element kind.
	endLines := make([]int, 0, len(syntaxBody.Attributes)+len(syntaxBody.Blocks))
	for _, attr := range syntaxBody.Attributes {
		endLines = append(endLines, attr.Range().End.Line)
	}
	for _, b := range syntaxBody.Blocks {
		endLines = append(endLines, b.Range().End.Line)
	}
	sort.Ints(endLines)

	blocks := make([]*hclsyntax.Block, len(syntaxBody.Blocks))
	copy(blocks, syntaxBody.Blocks)
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Range().Start.Line < blocks[j].Range().Start.Line
	})

	regions := make([]*BlockRegion, 0, len(blocks))
	for _, blk := range blocks {
		startLine := blk.Range().Start.Line
		endLine := blk.Range().End.Line

		searchStart := priorBoundary(endLines, startLine)

		leadingCommentLines := collectLeadingComments(lines, startLine, searchStart)

		region := &BlockRegion{
			Type:      blk.Type,
			Labels:    append([]string(nil), blk.Labels...),
			StartLine: startLine,
			EndLine:   endLine,
		}
		if len(leadingCommentLines) > 0 {
			region.LeadingComment = strings.Join(leadingCommentLines, "\n") + "\n"
			region.StartLine = startLine - len(leadingCommentLines)
		}
		for lineNum := startLine; lineNum <= endLine && lineNum-1 < len(lines); lineNum++ {
			region.Lines = append(region.Lines, lines[lineNum-1])
		}
		regions = append(regions, region)
	}

	return regions
}

// priorBoundary returns the line immediately after the largest end-line in sortedEndLines
// that is strictly less than startLine. Returns 1 if no such end-line exists.
func priorBoundary(sortedEndLines []int, startLine int) int {
	boundary := 1
	for _, end := range sortedEndLines {
		if end >= startLine {
			break
		}
		if end+1 > boundary {
			boundary = end + 1
		}
	}
	return boundary
}

// ReorderBlockBodyPreservingAll reorders attributes AND nested blocks in a block body,
// preserving leading comments on each. The desired output structure is:
//
//  1. Attributes named in attrOrder (in that order).
//  2. Nested blocks whose type is in nestedBlockOrder (in that order; multiple blocks of the
//     same type keep their original relative order).
//  3. Remaining attributes (in their original source order).
//  4. Remaining nested blocks (in their original source order).
//
// Items not present in the body are silently skipped.
func ReorderBlockBodyPreservingAll(
	content []byte,
	syntaxBody *hclsyntax.Body,
	blockStartLine, blockEndLine int,
	attrOrder, nestedBlockOrder []string,
) []byte {
	attrRegions := ExtractAttrRegions(content, syntaxBody)
	blockRegions := ExtractBlockRegions(content, syntaxBody)

	if len(attrRegions) == 0 && len(blockRegions) == 0 {
		return content
	}

	attrPrio := make(map[string]int, len(attrOrder))
	for i, name := range attrOrder {
		attrPrio[name] = i
	}
	blockPrio := make(map[string]int, len(nestedBlockOrder))
	for i, t := range nestedBlockOrder {
		blockPrio[t] = i
	}

	var orderedAttrs, leftoverAttrs []*AttrRegion
	for _, name := range GetOrderedAttrNames(syntaxBody) {
		region, ok := attrRegions[name]
		if !ok || region == nil {
			continue
		}
		if _, prioritized := attrPrio[name]; prioritized {
			orderedAttrs = append(orderedAttrs, region)
		} else {
			leftoverAttrs = append(leftoverAttrs, region)
		}
	}
	sort.SliceStable(orderedAttrs, func(i, j int) bool {
		return attrPrio[orderedAttrs[i].Name] < attrPrio[orderedAttrs[j].Name]
	})

	var orderedBlocks, leftoverBlocks []*BlockRegion
	for _, region := range blockRegions {
		if _, prioritized := blockPrio[region.Type]; prioritized {
			orderedBlocks = append(orderedBlocks, region)
		} else {
			leftoverBlocks = append(leftoverBlocks, region)
		}
	}
	sort.SliceStable(orderedBlocks, func(i, j int) bool {
		return blockPrio[orderedBlocks[i].Type] < blockPrio[orderedBlocks[j].Type]
	})

	lines := SplitLines(content)

	consumed := make(map[int]bool)
	markConsumedAttrLines(attrRegions, consumed)
	markConsumedBlockLines(blockRegions, consumed)
	orphans := collectOrphanLines(lines, blockStartLine, blockEndLine, consumed)

	var newBlockContent []string
	if blockStartLine-1 < len(lines) {
		newBlockContent = append(newBlockContent, lines[blockStartLine-1])
	}

	appendRegion := func(comment string, regionLines []string) {
		if comment != "" {
			commentLines := strings.Split(strings.TrimSuffix(comment, "\n"), "\n")
			newBlockContent = append(newBlockContent, commentLines...)
		}
		newBlockContent = append(newBlockContent, regionLines...)
	}
	for _, r := range orderedAttrs {
		appendRegion(r.LeadingComment, r.Lines)
	}
	for _, r := range orderedBlocks {
		appendRegion(r.LeadingComment, r.Lines)
	}
	for _, r := range leftoverAttrs {
		appendRegion(r.LeadingComment, r.Lines)
	}
	for _, r := range leftoverBlocks {
		appendRegion(r.LeadingComment, r.Lines)
	}

	newBlockContent = append(newBlockContent, orphans...)

	if blockEndLine-1 < len(lines) {
		newBlockContent = append(newBlockContent, lines[blockEndLine-1])
	}

	var result []string
	for i := 0; i < blockStartLine-1 && i < len(lines); i++ {
		result = append(result, lines[i])
	}
	result = append(result, newBlockContent...)
	for i := blockEndLine; i < len(lines); i++ {
		result = append(result, lines[i])
	}

	return []byte(strings.Join(result, "\n") + "\n")
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

// topLevelBlockPriority defines the canonical source order for top-level blocks.
// Blocks of unknown type (e.g., user-defined extensions) sort to the bottom in source order.
var topLevelBlockPriority = map[string]int{
	"terraform": 1,
	"provider":  2,
	"variable":  3,
	"locals":    4,
	"data":      5,
	"resource":  6,
	"module":    7,
	"output":    8,
}

// topLevelOtherPriority is the sort key for top-level blocks of unknown type. The wide gap
// from the highest known priority (8) leaves room to add new canonical types without
// disturbing the relative ordering of pre-existing unknown blocks.
const topLevelOtherPriority = 99

// collectAdjacentLeadingComments walks backwards from startLine-1 (exclusive) down to
// searchStart (inclusive), collecting `#` and `//` comment lines that are DIRECTLY
// adjacent (no blank-line gap). Unlike collectLeadingComments, a blank line terminates
// the scan. This stricter semantics matches the conventional reading at the top level:
// file-header comments are separated from the first block by a blank line and should
// stay with the file, not travel with the block.
//
// Returns the captured comment lines in source order. regionStart is the 1-indexed line
// number of the first captured comment (or equal to startLine if no comments were captured).
func collectAdjacentLeadingComments(lines []string, startLine, searchStart int) (comments []string, regionStart int) {
	regionStart = startLine
	for lineNum := startLine - 1; lineNum >= searchStart; lineNum-- {
		if lineNum-1 < 0 || lineNum-1 >= len(lines) {
			break
		}
		line := lines[lineNum-1]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		if !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "//") {
			break
		}
		comments = append([]string{line}, comments...)
		regionStart = lineNum
	}
	return comments, regionStart
}

// ReorderTopLevelBlocksByLineRange reorders top-level blocks in a HCL file according to the
// canonical priority defined by topLevelBlockPriority (terraform, provider, variable, locals,
// data, resource, module, output, then anything else in source order).
//
// This is a line-based reorder: each block is emitted as its original source line range
// plus any captured leading comments, so attribute order, inline comments, nested-block
// layout, and heredoc bodies inside each block are byte-for-byte preserved.
//
// Stable within priority: blocks of the same type retain their original relative order.
// File-header content (anything before the first block's adjacent leading comments) and
// file-footer content (anything after the last block in source) is preserved verbatim.
// Reordered blocks are joined with exactly one blank line between them.
//
// Top-level leading-comment capture is strict (collectAdjacentLeadingComments): only comments
// directly above a block with no blank-line gap travel with it. This matches the convention
// that file-level headers (copyright, license) are visually separated from the first block
// by a blank line and should remain anchored to the file.
func ReorderTopLevelBlocksByLineRange(content []byte) ([]byte, error) {
	syntaxFile, diags := hclsyntax.ParseConfig(content, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	body, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok || len(body.Blocks) == 0 {
		return content, nil
	}

	lines := SplitLines(content)

	sourceBlocks := make([]*hclsyntax.Block, len(body.Blocks))
	copy(sourceBlocks, body.Blocks)
	sort.Slice(sourceBlocks, func(i, j int) bool {
		return sourceBlocks[i].Range().Start.Line < sourceBlocks[j].Range().Start.Line
	})

	type topRegion struct {
		priority     int
		sourceIdx    int
		commentLines []string
		regionStart  int // 1-indexed line where the comment+body region begins
		bodyStart    int
		bodyEnd      int
	}

	regions := make([]topRegion, 0, len(sourceBlocks))
	for i, blk := range sourceBlocks {
		searchStart := 1
		if i > 0 {
			searchStart = sourceBlocks[i-1].Range().End.Line + 1
		}
		commentLines, regionStart := collectAdjacentLeadingComments(lines, blk.Range().Start.Line, searchStart)

		prio, known := topLevelBlockPriority[blk.Type]
		if !known {
			prio = topLevelOtherPriority
		}
		regions = append(regions, topRegion{
			priority:     prio,
			sourceIdx:    i,
			commentLines: commentLines,
			regionStart:  regionStart,
			bodyStart:    blk.Range().Start.Line,
			bodyEnd:      blk.Range().End.Line,
		})
	}

	headerEnd := regions[0].regionStart
	footerStart := sourceBlocks[len(sourceBlocks)-1].Range().End.Line

	sort.SliceStable(regions, func(i, j int) bool {
		if regions[i].priority != regions[j].priority {
			return regions[i].priority < regions[j].priority
		}
		return regions[i].sourceIdx < regions[j].sourceIdx
	})

	var result []string

	// File header: everything before the first source block's adjacent-comment region.
	for i := 0; i < headerEnd-1 && i < len(lines); i++ {
		result = append(result, lines[i])
	}

	for i, r := range regions {
		if i > 0 {
			result = append(result, "")
		}
		result = append(result, r.commentLines...)
		for line := r.bodyStart; line <= r.bodyEnd && line-1 < len(lines); line++ {
			result = append(result, lines[line-1])
		}
	}

	// File footer: everything after the last source block, dropping a leading blank
	// so the separator we emit before the footer isn't doubled.
	var footer []string
	for i := footerStart; i < len(lines); i++ {
		footer = append(footer, lines[i])
	}
	for len(footer) > 0 && strings.TrimSpace(footer[0]) == "" {
		footer = footer[1:]
	}
	if len(footer) > 0 {
		result = append(result, "")
		result = append(result, footer...)
	}

	return []byte(strings.Join(result, "\n") + "\n"), nil
}
