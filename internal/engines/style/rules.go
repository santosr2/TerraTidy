// Package style provides the style engine and rules for TerraTidy.
// It enforces consistent code style and formatting conventions in Terraform
// configurations, such as attribute ordering, block spacing, and naming conventions.
package style

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// getOrderedAttrNames returns attribute names from hclsyntax sorted by line number.
func getOrderedAttrNames(syntaxBody *hclsyntax.Body) []string {
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

// reorderBlockAttrs reorders attributes in a block according to the specified order.
// firstAttrs are placed at the start, lastAttrs at the end, others maintain relative order.
// orderedNames should be the original order of attributes (from hclsyntax parsing).
func reorderBlockAttrs(body *hclwrite.Body, orderedNames, firstAttrs, lastAttrs []string) {
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

	// Collect attribute info preserving expression tokens
	attrTokens := make(map[string]hclwrite.Tokens)
	for name, attr := range body.Attributes() {
		attrTokens[name] = attr.Expr().BuildTokens(nil)
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

// formatAndCleanBlankLines applies hclwrite.Format and removes extra blank lines inside blocks.
func formatAndCleanBlankLines(content []byte) []byte {
	// First apply hclwrite.Format
	formatted := hclwrite.Format(content)

	// Remove all blank lines inside blocks
	lines := splitLines(formatted)
	var result []byte
	insideBlock := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isBlank := len(trimmed) == 0
		hasOpenBrace := strings.Contains(line, "{")
		hasCloseBrace := strings.Contains(line, "}")

		// Track block depth (check before updating depth)
		// For closing braces, we're still "inside" when we see the brace
		if hasCloseBrace && !hasOpenBrace {
			insideBlock--
		}

		// Skip all blank lines inside blocks
		if insideBlock > 0 && isBlank {
			// Opening brace line updates depth after this check
			if hasOpenBrace {
				insideBlock++
			}
			continue
		}

		// Track block depth for opening braces
		if hasOpenBrace {
			insideBlock++
		}

		result = append(result, line...)
		result = append(result, '\n')
	}

	return result
}

// snakeCaseRegex matches valid snake_case identifiers
var snakeCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// NoBlankLinesInsideBlocksRule ensures no blank lines inside blocks.
type NoBlankLinesInsideBlocksRule struct{}

// Name returns the rule identifier.
func (r *NoBlankLinesInsideBlocksRule) Name() string {
	return "style.no-blank-lines-inside-blocks"
}

// Description returns a human-readable description of the rule.
func (r *NoBlankLinesInsideBlocksRule) Description() string {
	return "Ensures there are no blank lines inside blocks"
}

// Check examines the file for blank lines inside blocks.
func (r *NoBlankLinesInsideBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	lines := splitLines(content)

	// Check each block for internal blank lines
	for _, block := range hclFile.Blocks {
		blockFindings := r.checkBlock(ctx, block, lines)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *NoBlankLinesInsideBlocksRule) checkBlock(ctx *sdk.Context, block *hclsyntax.Block, lines []string) []sdk.Finding {
	var findings []sdk.Finding

	startLine := block.Range().Start.Line
	endLine := block.Range().End.Line

	// Skip if block is a single line
	if endLine <= startLine+1 {
		return findings
	}

	// Check lines inside the block (excluding start/end lines)
	filePath := ctx.File
	for lineNum := startLine + 1; lineNum < endLine; lineNum++ {
		if lineNum-1 >= len(lines) {
			continue
		}
		line := lines[lineNum-1]
		trimmed := trimLeftWhitespace(line)

		if len(trimmed) == 0 {
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: "Blank line inside block",
				File:    ctx.File,
				Location: hcl.Range{
					Filename: ctx.File,
					Start:    hcl.Pos{Line: lineNum, Column: 1},
					End:      hcl.Pos{Line: lineNum, Column: 1},
				},
				Severity: sdk.SeverityInfo,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					return r.fixFile(filePath)
				},
			})
		}
	}

	// Also check nested blocks recursively
	for _, nested := range block.Body.Blocks {
		nestedFindings := r.checkBlock(ctx, nested, lines)
		findings = append(findings, nestedFindings...)
	}

	return findings
}

// fixFile removes blank lines inside blocks.
func (r *NoBlankLinesInsideBlocksRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return formatAndCleanBlankLines(content), nil
}

// Fix removes blank lines inside blocks.
func (r *NoBlankLinesInsideBlocksRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	return r.fixFile(ctx.File)
}

// BlankLineBetweenBlocksRule ensures blank lines between top-level blocks.
type BlankLineBetweenBlocksRule struct{}

// Name returns the rule identifier.
func (r *BlankLineBetweenBlocksRule) Name() string {
	return "style.blank-line-between-blocks"
}

// Description returns a human-readable description of the rule.
func (r *BlankLineBetweenBlocksRule) Description() string {
	return "Ensures there is exactly one blank line between top-level blocks"
}

// Check examines the file for blank line violations between blocks.
func (r *BlankLineBetweenBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Read file content to check for comments and blank lines
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	lines := splitLines(content)

	blocks := hclFile.Blocks
	for i := 0; i < len(blocks)-1; i++ {
		currentBlock := blocks[i]
		nextBlock := blocks[i+1]

		endLine := currentBlock.Range().End.Line
		startLine := nextBlock.Range().Start.Line

		// Count actual blank lines (excluding comments) between blocks
		blankLines := countBlankLinesBetween(lines, endLine, startLine)

		if blankLines < 1 {
			// Capture values for closure
			filePath := ctx.File
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Missing blank line between blocks",
				File:     ctx.File,
				Location: nextBlock.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					return r.fixFile(filePath)
				},
			})
		} else if blankLines > 1 {
			// Capture values for closure
			filePath := ctx.File
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Too many blank lines between blocks (should be exactly 1)",
				File:     ctx.File,
				Location: nextBlock.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					return r.fixFile(filePath)
				},
			})
		}
	}

	return findings, nil
}

// splitLines splits content into lines.
func splitLines(content []byte) []string {
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

// countBlankLinesBetween counts actual blank lines (not comments) between two line numbers.
// Line numbers are 1-indexed (HCL convention).
func countBlankLinesBetween(lines []string, endLine, startLine int) int {
	blankCount := 0

	// Lines between endLine and startLine (exclusive of both)
	for lineNum := endLine + 1; lineNum < startLine; lineNum++ {
		if lineNum-1 >= len(lines) {
			continue
		}
		line := lines[lineNum-1] // Convert to 0-indexed
		trimmed := trimLeftWhitespace(line)

		// Count as blank if empty or whitespace-only
		// Don't count comment lines as blank lines
		if len(trimmed) == 0 {
			blankCount++
		}
	}

	return blankCount
}

// trimLeftWhitespace trims leading whitespace from a string.
func trimLeftWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[i:]
		}
	}
	return ""
}

// fixFile fixes blank line issues in the file
func (r *BlankLineBetweenBlocksRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse to get block positions
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	hclFile, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return content, nil
	}

	// Get original lines
	lines := splitLines(content)

	// Build a map of line numbers that need adjustments
	// We'll track where to add/remove blank lines
	type lineAdjustment struct {
		afterLine int
		action    string // "add" or "remove"
	}
	var adjustments []lineAdjustment

	blocks := hclFile.Blocks
	for i := 0; i < len(blocks)-1; i++ {
		currentBlock := blocks[i]
		nextBlock := blocks[i+1]

		endLine := currentBlock.Range().End.Line
		startLine := nextBlock.Range().Start.Line

		// Count actual blank lines between blocks
		blankLines := countBlankLinesBetween(lines, endLine, startLine)

		if blankLines < 1 {
			adjustments = append(adjustments, lineAdjustment{
				afterLine: endLine,
				action:    "add",
			})
		} else if blankLines > 1 {
			// Need to remove extra blank lines
			adjustments = append(adjustments, lineAdjustment{
				afterLine: endLine,
				action:    "remove",
			})
		}
	}

	if len(adjustments) == 0 {
		return content, nil
	}

	// Apply adjustments (process from end to start to keep line numbers valid)
	var result []string
	lineNum := 1
	for lineNum <= len(lines) {
		line := lines[lineNum-1]
		result = append(result, line)

		// Check for adjustments at this line
		for _, adj := range adjustments {
			if adj.afterLine == lineNum {
				if adj.action == "add" {
					// Add a blank line
					result = append(result, "")
				} else if adj.action == "remove" {
					// Skip extra blank lines (but keep one)
					blankKept := false
					for lineNum < len(lines) {
						nextLine := lines[lineNum] // 0-indexed, so lineNum is next
						trimmed := trimLeftWhitespace(nextLine)
						if len(trimmed) > 0 {
							break // Found non-blank line
						}
						if !blankKept {
							result = append(result, "")
							blankKept = true
						}
						lineNum++
					}
				}
			}
		}

		lineNum++
	}

	// Join lines with newlines
	return []byte(strings.Join(result, "\n") + "\n"), nil
}

// Fix corrects blank line issues between blocks.
func (r *BlankLineBetweenBlocksRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	return r.fixFile(ctx.File)
}

// BlockLabelCaseRule ensures block labels follow naming conventions.
type BlockLabelCaseRule struct{}

// Name returns the rule identifier.
func (r *BlockLabelCaseRule) Name() string {
	return "style.block-label-case"
}

// Description returns a human-readable description of the rule.
func (r *BlockLabelCaseRule) Description() string {
	return "Ensures block labels follow naming conventions (snake_case for resources/data)"
}

// Check examines block labels for naming convention violations.
func (r *BlockLabelCaseRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		blockType := block.Type

		if blockType != "resource" && blockType != "data" && blockType != "module" {
			continue
		}

		if len(block.Labels) < 2 {
			continue
		}

		name := block.Labels[1]

		if name == "" {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Block label cannot be empty",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityError,
				Fixable:  false,
			})
			continue
		}

		// Validate snake_case for resources and data sources
		if (blockType == "resource" || blockType == "data") && !snakeCaseRegex.MatchString(name) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Block label should be snake_case: " + name,
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as block label renaming requires manual review.
func (r *BlockLabelCaseRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// ForEachCountFirstRule ensures for_each/count is the first attribute in resource/module blocks.
type ForEachCountFirstRule struct{}

// Name returns the rule identifier.
func (r *ForEachCountFirstRule) Name() string {
	return "style.for-each-count-first"
}

// Description returns a human-readable description of the rule.
func (r *ForEachCountFirstRule) Description() string {
	return "Ensures for_each or count is the first attribute in resource/module blocks"
}

// Check examines blocks for for_each/count attribute positioning.
func (r *ForEachCountFirstRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" {
			continue
		}

		body := block.Body

		// Find for_each or count attributes
		var forEachAttr, countAttr *hclsyntax.Attribute
		var firstAttr *hclsyntax.Attribute
		firstAttrLine := int(^uint(0) >> 1) // max int

		for name, attr := range body.Attributes {
			if attr.Range().Start.Line < firstAttrLine {
				firstAttrLine = attr.Range().Start.Line
				firstAttr = attr
			}
			if name == "for_each" {
				forEachAttr = attr
			}
			if name == "count" {
				countAttr = attr
			}
		}

		// Check if for_each/count exists but is not first
		if forEachAttr != nil && firstAttr != nil && forEachAttr != firstAttr {
			filePath := ctx.File
			blockType := block.Type
			blockLabels := block.Labels
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "for_each should be the first attribute in the block",
				File:     ctx.File,
				Location: forEachAttr.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					return r.fixBlock(filePath, blockType, blockLabels, "for_each")
				},
			})
		}

		if countAttr != nil && firstAttr != nil && countAttr != firstAttr && forEachAttr == nil {
			filePath := ctx.File
			blockType := block.Type
			blockLabels := block.Labels
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "count should be the first attribute in the block",
				File:     ctx.File,
				Location: countAttr.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					return r.fixBlock(filePath, blockType, blockLabels, "count")
				},
			})
		}
	}

	return findings, nil
}

// fixBlock moves for_each or count to be the first attribute in the block.
func (r *ForEachCountFirstRule) fixBlock(
	filePath, blockType string,
	blockLabels []string,
	_ string, // attrName - unused, we handle both for_each and count
) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse with hclsyntax to get attribute ordering
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Parse with hclwrite for modifications
	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Find syntax body for this block
	var syntaxBody *hclsyntax.Body
	for _, block := range syntaxFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type != blockType {
			continue
		}
		if len(block.Labels) != len(blockLabels) {
			continue
		}
		match := true
		for i, l := range block.Labels {
			if l != blockLabels[i] {
				match = false
				break
			}
		}
		if match {
			syntaxBody = block.Body
			break
		}
	}

	if syntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}
		labels := block.Labels()
		if len(labels) != len(blockLabels) {
			continue
		}
		match := true
		for i, l := range labels {
			if l != blockLabels[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		orderedNames := getOrderedAttrNames(syntaxBody)
		firstAttrs := []string{"for_each", "count"}
		reorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
		break
	}

	return formatAndCleanBlankLines(writeFile.Bytes()), nil
}

// Fix moves for_each/count to be first attribute in each block.
func (r *ForEachCountFirstRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	// Parse with hclsyntax to get attribute ordering
	syntaxFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	// Parse with hclwrite for modifications
	writeFile, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Build a map of block identifiers to their syntax bodies for ordering
	syntaxBlocks := make(map[string]*hclsyntax.Body)
	for _, block := range syntaxFile.Blocks {
		key := blockKey(block.Type, block.Labels)
		syntaxBlocks[key] = block.Body
	}

	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "resource" && block.Type() != "module" && block.Type() != "data" {
			continue
		}

		// Check if block has for_each or count
		hasForEach := block.Body().GetAttribute("for_each") != nil
		hasCount := block.Body().GetAttribute("count") != nil
		if !hasForEach && !hasCount {
			continue
		}

		// Get ordering from syntax body
		key := blockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := getOrderedAttrNames(syntaxBody)
		firstAttrs := []string{"for_each", "count"}
		reorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
	}

	return formatAndCleanBlankLines(writeFile.Bytes()), nil
}

// blockKey creates a unique key for a block based on type and labels.
func blockKey(blockType string, labels []string) string {
	key := blockType
	for _, l := range labels {
		key += "." + l
	}
	return key
}

// LifecycleAtEndRule ensures lifecycle block is at the end of resource blocks.
type LifecycleAtEndRule struct{}

// Name returns the rule identifier.
func (r *LifecycleAtEndRule) Name() string {
	return "style.lifecycle-at-end"
}

// Description returns a human-readable description of the rule.
func (r *LifecycleAtEndRule) Description() string {
	return "Ensures lifecycle block is at the end of resource blocks"
}

// Check examines resource blocks for lifecycle block positioning.
func (r *LifecycleAtEndRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" {
			continue
		}

		body := block.Body

		// Find lifecycle block and the last element
		var lifecycleBlock *hclsyntax.Block
		var lastLine int

		for _, nested := range body.Blocks {
			if nested.Range().End.Line > lastLine {
				lastLine = nested.Range().End.Line
			}
			if nested.Type == "lifecycle" {
				lifecycleBlock = nested
			}
		}

		for _, attr := range body.Attributes {
			if attr.Range().End.Line > lastLine {
				lastLine = attr.Range().End.Line
			}
		}

		// If lifecycle exists and is not at the end
		if lifecycleBlock != nil && lifecycleBlock.Range().End.Line < lastLine {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "lifecycle block should be at the end of the resource block",
				File:     ctx.File,
				Location: lifecycleBlock.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as lifecycle reordering requires manual review.
func (r *LifecycleAtEndRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// TagsAtEndRule ensures tags/labels are at the end of resource blocks (before lifecycle).
type TagsAtEndRule struct{}

// Name returns the rule identifier.
func (r *TagsAtEndRule) Name() string {
	return "style.tags-at-end"
}

// Description returns a human-readable description of the rule.
func (r *TagsAtEndRule) Description() string {
	return "Ensures tags/labels are near the end of resource blocks (before lifecycle)"
}

// Check examines resource blocks for tags/labels positioning.
func (r *TagsAtEndRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "module" {
			continue
		}
		blockFindings := r.checkTagsBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *TagsAtEndRule) checkTagsBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding
	body := block.Body

	tagsAttr := findTagsAttribute(body.Attributes)
	if tagsAttr == nil {
		return findings
	}

	// Create fix function for this block
	filePath := ctx.File
	blockType := block.Type
	blockLabels := block.Labels
	fixFunc := func() ([]byte, error) {
		return r.fixTagsBlock(filePath, blockType, blockLabels)
	}

	lifecycleBlock := findNestedBlock(body.Blocks, "lifecycle")
	tagsLine := tagsAttr.Range().Start.Line

	if lifecycleBlock != nil && tagsLine > lifecycleBlock.Range().Start.Line {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "tags should be before lifecycle block",
			File:     ctx.File,
			Location: tagsAttr.Range(),
			Severity: sdk.SeverityWarning,
			Fixable:  true,
			FixFunc:  fixFunc,
		})
	}

	if countAttrsAfterTags(body.Attributes, tagsLine) > 2 {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "tags should be near the end of the block",
			File:     ctx.File,
			Location: tagsAttr.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  true,
			FixFunc:  fixFunc,
		})
	}

	return findings
}

// fixTagsBlock reorders a specific block's tags attribute to be at the end.
func (r *TagsAtEndRule) fixTagsBlock(filePath, blockType string, blockLabels []string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse with hclsyntax to get attribute ordering
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Parse with hclwrite for modifications
	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Find syntax body for this block
	var syntaxBody *hclsyntax.Body
	for _, block := range syntaxFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type != blockType {
			continue
		}
		if len(block.Labels) != len(blockLabels) {
			continue
		}
		match := true
		for i, l := range block.Labels {
			if l != blockLabels[i] {
				match = false
				break
			}
		}
		if match {
			syntaxBody = block.Body
			break
		}
	}

	if syntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}
		labels := block.Labels()
		if len(labels) != len(blockLabels) {
			continue
		}
		match := true
		for i, l := range labels {
			if l != blockLabels[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		orderedNames := getOrderedAttrNames(syntaxBody)
		// tags/labels should be at the end
		lastAttrs := []string{"tags", "labels", "tags_all"}
		reorderBlockAttrs(block.Body(), orderedNames, nil, lastAttrs)
		break
	}

	return formatAndCleanBlankLines(writeFile.Bytes()), nil
}

func findTagsAttribute(attrs hclsyntax.Attributes) *hclsyntax.Attribute {
	for name, attr := range attrs {
		if name == "tags" || name == "labels" || name == "tags_all" {
			return attr
		}
	}
	return nil
}

func countAttrsAfterTags(attrs hclsyntax.Attributes, tagsLine int) int {
	count := 0
	for name, attr := range attrs {
		if attr.Range().Start.Line > tagsLine && name != "tags_all" {
			count++
		}
	}
	return count
}

// Fix moves tags/labels to the end of blocks (before lifecycle if present).
func (r *TagsAtEndRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	syntaxFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	writeFile, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Build a map of block identifiers to their syntax bodies
	syntaxBlocks := make(map[string]*hclsyntax.Body)
	for _, block := range syntaxFile.Blocks {
		if block.Type == "resource" || block.Type == "module" {
			key := blockKey(block.Type, block.Labels)
			syntaxBlocks[key] = block.Body
		}
	}

	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "resource" && block.Type() != "module" {
			continue
		}

		// Check if block has tags
		hasTags := block.Body().GetAttribute("tags") != nil ||
			block.Body().GetAttribute("labels") != nil ||
			block.Body().GetAttribute("tags_all") != nil
		if !hasTags {
			continue
		}

		key := blockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := getOrderedAttrNames(syntaxBody)
		// tags/labels should be at the end
		lastAttrs := []string{"tags", "labels", "tags_all"}
		reorderBlockAttrs(block.Body(), orderedNames, nil, lastAttrs)
	}

	return formatAndCleanBlankLines(writeFile.Bytes()), nil
}

// DependsOnOrderRule ensures depends_on is at the end of blocks.
type DependsOnOrderRule struct{}

// Name returns the rule identifier.
func (r *DependsOnOrderRule) Name() string {
	return "style.depends-on-order"
}

// Description returns a human-readable description of the rule.
func (r *DependsOnOrderRule) Description() string {
	return "Ensures depends_on is at the end of resource/module blocks"
}

// Check examines blocks for depends_on attribute positioning.
func (r *DependsOnOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if !isDependsOnRelevantBlock(block.Type) {
			continue
		}
		blockFindings := r.checkDependsOnBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func isDependsOnRelevantBlock(blockType string) bool {
	return blockType == "resource" || blockType == "module" || blockType == "data"
}

func (r *DependsOnOrderRule) checkDependsOnBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding
	body := block.Body

	dependsOnAttr := findAttribute(body.Attributes, "depends_on")
	if dependsOnAttr == nil {
		return findings
	}

	lifecycleBlock := findNestedBlock(body.Blocks, "lifecycle")
	dependsOnLine := dependsOnAttr.Range().Start.Line

	if lifecycleBlock != nil && dependsOnLine > lifecycleBlock.Range().Start.Line {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "depends_on should be before lifecycle block",
			File:     ctx.File,
			Location: dependsOnAttr.Range(),
			Severity: sdk.SeverityWarning,
			Fixable:  false,
		})
	}

	if r.hasAttributesAfterDependsOn(body.Attributes, dependsOnLine) {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "depends_on should be near the end of the block",
			File:     ctx.File,
			Location: dependsOnAttr.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  false,
		})
	}

	return findings
}

func findAttribute(attrs hclsyntax.Attributes, name string) *hclsyntax.Attribute {
	for n, attr := range attrs {
		if n == name {
			return attr
		}
	}
	return nil
}

func findNestedBlock(blocks hclsyntax.Blocks, blockType string) *hclsyntax.Block {
	for _, b := range blocks {
		if b.Type == blockType {
			return b
		}
	}
	return nil
}

func (r *DependsOnOrderRule) hasAttributesAfterDependsOn(attrs hclsyntax.Attributes, dependsOnLine int) bool {
	endAttrs := map[string]bool{"depends_on": true, "tags": true, "tags_all": true, "labels": true}
	for name, attr := range attrs {
		if !endAttrs[name] && attr.Range().Start.Line > dependsOnLine {
			return true
		}
	}
	return false
}

// Fix is a no-op for this rule as depends_on reordering requires manual review.
func (r *DependsOnOrderRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// SourceVersionGroupedRule ensures source and version are grouped together in module blocks.
type SourceVersionGroupedRule struct{}

// Name returns the rule identifier.
func (r *SourceVersionGroupedRule) Name() string {
	return "style.source-version-grouped"
}

// Description returns a human-readable description of the rule.
func (r *SourceVersionGroupedRule) Description() string {
	return "Ensures source and version are grouped at the start of module blocks"
}

// Check examines module blocks for source/version attribute grouping.
func (r *SourceVersionGroupedRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "module" {
			continue
		}
		blockFindings := r.checkModuleBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *SourceVersionGroupedRule) checkModuleBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding
	body := block.Body

	sourceAttr := findAttribute(body.Attributes, "source")
	versionAttr := findAttribute(body.Attributes, "version")

	// Create fix function that handles all source/version ordering for this block
	filePath := ctx.File
	blockLabels := block.Labels
	fixFunc := func() ([]byte, error) {
		return r.fixModuleBlock(filePath, blockLabels)
	}

	if sourceAttr != nil {
		if finding := r.checkSourcePosition(ctx, body.Attributes, sourceAttr, fixFunc); finding != nil {
			findings = append(findings, *finding)
		}
	}

	if sourceAttr != nil && versionAttr != nil {
		if finding := r.checkVersionFollowsSource(ctx, body.Attributes, sourceAttr, versionAttr, fixFunc); finding != nil {
			findings = append(findings, *finding)
		}
	}

	return findings
}

// fixModuleBlock reorders a specific module block's attributes.
func (r *SourceVersionGroupedRule) fixModuleBlock(filePath string, blockLabels []string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse with hclsyntax to get attribute ordering
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Parse with hclwrite for modifications
	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Find syntax body for this block
	var syntaxBody *hclsyntax.Body
	for _, block := range syntaxFile.Body.(*hclsyntax.Body).Blocks {
		if block.Type != "module" {
			continue
		}
		if len(block.Labels) != len(blockLabels) {
			continue
		}
		match := true
		for i, l := range block.Labels {
			if l != blockLabels[i] {
				match = false
				break
			}
		}
		if match {
			syntaxBody = block.Body
			break
		}
	}

	if syntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "module" {
			continue
		}
		labels := block.Labels()
		if len(labels) != len(blockLabels) {
			continue
		}
		match := true
		for i, l := range labels {
			if l != blockLabels[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}

		orderedNames := getOrderedAttrNames(syntaxBody)
		// source and version should come after for_each/count but before everything else
		firstAttrs := []string{"for_each", "count", "source", "version"}
		reorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
		break
	}

	return formatAndCleanBlankLines(writeFile.Bytes()), nil
}

func (r *SourceVersionGroupedRule) checkSourcePosition(
	ctx *sdk.Context, attrs hclsyntax.Attributes, sourceAttr *hclsyntax.Attribute,
	fixFunc func() ([]byte, error),
) *sdk.Finding {
	sourceLine := sourceAttr.Range().Start.Line
	allowedBefore := map[string]bool{"source": true, "for_each": true, "count": true}

	for name, attr := range attrs {
		if !allowedBefore[name] && attr.Range().Start.Line < sourceLine {
			return &sdk.Finding{
				Rule:     r.Name(),
				Message:  "source should be at the start of module block (after for_each/count if present)",
				File:     ctx.File,
				Location: sourceAttr.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc:  fixFunc,
			}
		}
	}
	return nil
}

func (r *SourceVersionGroupedRule) checkVersionFollowsSource(
	ctx *sdk.Context, attrs hclsyntax.Attributes,
	sourceAttr, versionAttr *hclsyntax.Attribute,
	fixFunc func() ([]byte, error),
) *sdk.Finding {
	sourceLine := sourceAttr.Range().End.Line
	versionLine := versionAttr.Range().Start.Line

	for name, attr := range attrs {
		attrLine := attr.Range().Start.Line
		if name != "source" && name != "version" &&
			attrLine > sourceLine && attrLine < versionLine {
			return &sdk.Finding{
				Rule:     r.Name(),
				Message:  "version should immediately follow source in module block",
				File:     ctx.File,
				Location: versionAttr.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc:  fixFunc,
			}
		}
	}
	return nil
}

// Fix reorders source/version to be at the start of module blocks (after for_each/count).
func (r *SourceVersionGroupedRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	syntaxFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	writeFile, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Build a map of block identifiers to their syntax bodies
	syntaxBlocks := make(map[string]*hclsyntax.Body)
	for _, block := range syntaxFile.Blocks {
		if block.Type == "module" {
			key := blockKey(block.Type, block.Labels)
			syntaxBlocks[key] = block.Body
		}
	}

	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "module" {
			continue
		}

		// Check if block has source
		if block.Body().GetAttribute("source") == nil {
			continue
		}

		key := blockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := getOrderedAttrNames(syntaxBody)
		// source and version should come after for_each/count but before everything else
		firstAttrs := []string{"for_each", "count", "source", "version"}
		reorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
	}

	return formatAndCleanBlankLines(writeFile.Bytes()), nil
}

// VariableOrderRule ensures variable blocks follow standard ordering.
type VariableOrderRule struct{}

// Name returns the rule identifier.
func (r *VariableOrderRule) Name() string {
	return "style.variable-order"
}

// Description returns a human-readable description of the rule.
func (r *VariableOrderRule) Description() string {
	return "Ensures variable blocks follow standard ordering: description, type, default, validation"
}

// varAttrPos represents an attribute position for ordering checks.
type varAttrPos struct {
	name  string
	line  int
	order int
}

// varAttrOrder defines the expected order for variable attributes.
var varAttrOrder = map[string]int{
	"description": 1,
	"type":        2,
	"default":     3,
	"sensitive":   4,
	"nullable":    5,
}

// Check examines variable blocks for attribute ordering.
func (r *VariableOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "variable" {
			continue
		}
		blockFindings := r.checkVariableBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *VariableOrderRule) checkVariableBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	attrs := r.collectVariableAttrs(block.Body)
	return r.findOrderViolations(ctx, block, attrs)
}

func (r *VariableOrderRule) collectVariableAttrs(body *hclsyntax.Body) []varAttrPos {
	var attrs []varAttrPos

	for name, attr := range body.Attributes {
		if order, ok := varAttrOrder[name]; ok {
			attrs = append(attrs, varAttrPos{
				name:  name,
				line:  attr.Range().Start.Line,
				order: order,
			})
		}
	}

	for _, nested := range body.Blocks {
		if nested.Type == "validation" {
			attrs = append(attrs, varAttrPos{
				name:  "validation",
				line:  nested.Range().Start.Line,
				order: 6,
			})
		}
	}

	return attrs
}

func (r *VariableOrderRule) findOrderViolations(
	ctx *sdk.Context, block *hclsyntax.Block, attrs []varAttrPos,
) []sdk.Finding {
	var findings []sdk.Finding
	if len(attrs) < 2 {
		return findings
	}

	for i := 0; i < len(attrs)-1; i++ {
		for j := i + 1; j < len(attrs); j++ {
			if finding := r.checkAttrPair(ctx, block, attrs[i], attrs[j]); finding != nil {
				findings = append(findings, *finding)
			}
		}
	}
	return findings
}

func (r *VariableOrderRule) checkAttrPair(ctx *sdk.Context, block *hclsyntax.Block, a, b varAttrPos) *sdk.Finding {
	filePath := ctx.File

	if b.line < a.line && b.order > a.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  b.name + " should come after " + a.name + " in variable block",
			File:     ctx.File,
			Location: block.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  true,
			FixFunc: func() ([]byte, error) {
				return r.Fix(&sdk.Context{File: filePath}, nil)
			},
		}
	}

	if a.line < b.line && a.order > b.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  a.name + " should come after " + b.name + " in variable block",
			File:     ctx.File,
			Location: block.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  true,
			FixFunc: func() ([]byte, error) {
				return r.Fix(&sdk.Context{File: filePath}, nil)
			},
		}
	}

	return nil
}

// Fix reorders variable attributes to match the standard order.
func (r *VariableOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	f, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Expected order for variable attributes
	attrOrder := []string{"description", "type", "default", "sensitive", "nullable"}

	for _, block := range f.Body().Blocks() {
		if block.Type() != "variable" {
			continue
		}

		body := block.Body()

		// Collect all attributes with their expressions
		attrExprs := make(map[string]hclwrite.Tokens)
		for name, attr := range body.Attributes() {
			attrExprs[name] = attr.Expr().BuildTokens(nil)
		}

		// Remove all known-order attributes
		for _, name := range attrOrder {
			body.RemoveAttribute(name)
		}

		// Re-add in correct order
		for _, name := range attrOrder {
			if tokens, ok := attrExprs[name]; ok {
				body.SetAttributeRaw(name, tokens)
			}
		}

		// Add back any other attributes that weren't in the order list
		for name, tokens := range attrExprs {
			found := false
			for _, orderedName := range attrOrder {
				if name == orderedName {
					found = true
					break
				}
			}
			if !found {
				body.SetAttributeRaw(name, tokens)
			}
		}
	}

	return f.Bytes(), nil
}

// OutputOrderRule ensures output blocks follow standard ordering.
type OutputOrderRule struct{}

// Name returns the rule identifier.
func (r *OutputOrderRule) Name() string {
	return "style.output-order"
}

// Description returns a human-readable description of the rule.
func (r *OutputOrderRule) Description() string {
	return "Ensures output blocks follow standard ordering: description, value, sensitive"
}

// outputAttrOrder defines the expected order for output attributes.
var outputAttrOrder = map[string]int{
	"description": 1,
	"value":       2,
	"sensitive":   3,
	"depends_on":  4,
}

// Check examines output blocks for attribute ordering.
func (r *OutputOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "output" {
			continue
		}
		blockFindings := r.checkOutputBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *OutputOrderRule) checkOutputBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	attrs := r.collectOutputAttrs(block.Body)
	return r.findOutputOrderViolations(ctx, block, attrs)
}

func (r *OutputOrderRule) collectOutputAttrs(body *hclsyntax.Body) []varAttrPos {
	var attrs []varAttrPos

	for name, attr := range body.Attributes {
		if order, ok := outputAttrOrder[name]; ok {
			attrs = append(attrs, varAttrPos{
				name:  name,
				line:  attr.Range().Start.Line,
				order: order,
			})
		}
	}

	return attrs
}

func (r *OutputOrderRule) findOutputOrderViolations(
	ctx *sdk.Context, block *hclsyntax.Block, attrs []varAttrPos,
) []sdk.Finding {
	var findings []sdk.Finding
	if len(attrs) < 2 {
		return findings
	}

	for i := 0; i < len(attrs)-1; i++ {
		for j := i + 1; j < len(attrs); j++ {
			if finding := r.checkOutputAttrPair(ctx, block, attrs[i], attrs[j]); finding != nil {
				findings = append(findings, *finding)
			}
		}
	}
	return findings
}

func (r *OutputOrderRule) checkOutputAttrPair(ctx *sdk.Context, block *hclsyntax.Block, a, b varAttrPos) *sdk.Finding {
	filePath := ctx.File

	if b.line < a.line && b.order > a.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  b.name + " should come after " + a.name + " in output block",
			File:     ctx.File,
			Location: block.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  true,
			FixFunc: func() ([]byte, error) {
				return r.Fix(&sdk.Context{File: filePath}, nil)
			},
		}
	}

	if a.line < b.line && a.order > b.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  a.name + " should come after " + b.name + " in output block",
			File:     ctx.File,
			Location: block.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  true,
			FixFunc: func() ([]byte, error) {
				return r.Fix(&sdk.Context{File: filePath}, nil)
			},
		}
	}

	return nil
}

// Fix reorders output attributes to match the standard order.
func (r *OutputOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	f, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Expected order for output attributes
	attrOrder := []string{"description", "value", "sensitive", "depends_on"}

	for _, block := range f.Body().Blocks() {
		if block.Type() != "output" {
			continue
		}

		body := block.Body()

		// Collect all attributes with their expressions
		attrExprs := make(map[string]hclwrite.Tokens)
		for name, attr := range body.Attributes() {
			attrExprs[name] = attr.Expr().BuildTokens(nil)
		}

		// Remove all known-order attributes
		for _, name := range attrOrder {
			body.RemoveAttribute(name)
		}

		// Re-add in correct order
		for _, name := range attrOrder {
			if tokens, ok := attrExprs[name]; ok {
				body.SetAttributeRaw(name, tokens)
			}
		}

		// Add back any other attributes that weren't in the order list
		for name, tokens := range attrExprs {
			found := false
			for _, orderedName := range attrOrder {
				if name == orderedName {
					found = true
					break
				}
			}
			if !found {
				body.SetAttributeRaw(name, tokens)
			}
		}
	}

	return f.Bytes(), nil
}

// TerraformBlockFirstRule ensures terraform block is first in the file.
type TerraformBlockFirstRule struct{}

// Name returns the rule identifier.
func (r *TerraformBlockFirstRule) Name() string {
	return "style.terraform-block-first"
}

// Description returns a human-readable description of the rule.
func (r *TerraformBlockFirstRule) Description() string {
	return "Ensures terraform block is the first block in the file"
}

// Check examines the file for terraform block positioning.
func (r *TerraformBlockFirstRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	if len(hclFile.Blocks) == 0 {
		return findings, nil
	}

	var terraformBlock *hclsyntax.Block
	firstBlock := hclFile.Blocks[0]

	for _, block := range hclFile.Blocks {
		if block.Type == "terraform" {
			terraformBlock = block
			break
		}
	}

	if terraformBlock != nil && terraformBlock != firstBlock {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "terraform block should be the first block in the file",
			File:     ctx.File,
			Location: terraformBlock.Range(),
			Severity: sdk.SeverityWarning,
			Fixable:  false,
		})
	}

	return findings, nil
}

// Fix is a no-op for this rule as block reordering requires manual review.
func (r *TerraformBlockFirstRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// ProviderBlockOrderRule ensures provider blocks come after terraform block.
type ProviderBlockOrderRule struct{}

// Name returns the rule identifier.
func (r *ProviderBlockOrderRule) Name() string {
	return "style.provider-block-order"
}

// Description returns a human-readable description of the rule.
func (r *ProviderBlockOrderRule) Description() string {
	return "Ensures provider blocks come after terraform block"
}

// Check examines the file for provider block positioning.
func (r *ProviderBlockOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	var terraformEndLine int
	firstResourceLine := int(^uint(0) >> 1)

	for _, block := range hclFile.Blocks {
		if block.Type == "terraform" {
			terraformEndLine = block.Range().End.Line
		}
		if block.Type == "resource" || block.Type == "data" || block.Type == "module" {
			if block.Range().Start.Line < firstResourceLine {
				firstResourceLine = block.Range().Start.Line
			}
		}
	}

	for _, block := range hclFile.Blocks {
		if block.Type == "provider" {
			providerLine := block.Range().Start.Line

			// Provider should be after terraform block
			if terraformEndLine > 0 && providerLine < terraformEndLine {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "provider block should come after terraform block",
					File:     ctx.File,
					Location: block.Range(),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}

			// Provider should be before resources/data/modules
			if providerLine > firstResourceLine {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "provider block should come before resource/data/module blocks",
					File:     ctx.File,
					Location: block.Range(),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as provider block reordering requires manual review.
func (r *ProviderBlockOrderRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// NoEmptyBlocksRule ensures blocks are not empty.
type NoEmptyBlocksRule struct{}

// Name returns the rule identifier.
func (r *NoEmptyBlocksRule) Name() string {
	return "style.no-empty-blocks"
}

// Description returns a human-readable description of the rule.
func (r *NoEmptyBlocksRule) Description() string {
	return "Ensures blocks are not empty without content"
}

// Check examines blocks for empty content.
func (r *NoEmptyBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		body := block.Body

		if len(body.Attributes) == 0 && len(body.Blocks) == 0 {
			// Some blocks are allowed to be empty
			if block.Type == "lifecycle" || block.Type == "provisioner" {
				continue
			}

			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Block is empty: " + block.Type,
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as empty block removal requires manual review.
func (r *NoEmptyBlocksRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// VariablesInFileRule ensures variables are defined in variables.tf.
type VariablesInFileRule struct{}

// Name returns the rule identifier.
func (r *VariablesInFileRule) Name() string {
	return "style.variables-in-file"
}

// Description returns a human-readable description of the rule.
func (r *VariablesInFileRule) Description() string {
	return "Variables should be defined in variables.tf"
}

// Check examines if variables are in the correct file.
func (r *VariablesInFileRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Skip if this is variables.tf
	basename := extractBasename(ctx.File)
	if basename == "variables.tf" {
		return findings, nil
	}

	// Check for variable blocks in non-variables.tf files
	for _, block := range hclFile.Blocks {
		if block.Type == "variable" {
			varName := ""
			if len(block.Labels) > 0 {
				varName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Variable '" + varName + "' should be defined in variables.tf",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as moving blocks requires manual review.
func (r *VariablesInFileRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// OutputsInFileRule ensures outputs are defined in outputs.tf.
type OutputsInFileRule struct{}

// Name returns the rule identifier.
func (r *OutputsInFileRule) Name() string {
	return "style.outputs-in-file"
}

// Description returns a human-readable description of the rule.
func (r *OutputsInFileRule) Description() string {
	return "Outputs should be defined in outputs.tf"
}

// Check examines if outputs are in the correct file.
func (r *OutputsInFileRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Skip if this is outputs.tf
	basename := extractBasename(ctx.File)
	if basename == "outputs.tf" {
		return findings, nil
	}

	// Check for output blocks in non-outputs.tf files
	for _, block := range hclFile.Blocks {
		if block.Type == "output" {
			outputName := ""
			if len(block.Labels) > 0 {
				outputName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Output '" + outputName + "' should be defined in outputs.tf",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as moving blocks requires manual review.
func (r *OutputsInFileRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// ProvidersInFileRule ensures providers are defined in providers.tf or versions.tf.
type ProvidersInFileRule struct{}

// Name returns the rule identifier.
func (r *ProvidersInFileRule) Name() string {
	return "style.providers-in-file"
}

// Description returns a human-readable description of the rule.
func (r *ProvidersInFileRule) Description() string {
	return "Provider configurations should be in providers.tf or versions.tf"
}

// Check examines if providers are in the correct file.
func (r *ProvidersInFileRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Skip if this is providers.tf or versions.tf
	basename := extractBasename(ctx.File)
	if basename == "providers.tf" || basename == "versions.tf" {
		return findings, nil
	}

	// Check for provider blocks in other files
	for _, block := range hclFile.Blocks {
		if block.Type == "provider" {
			providerName := ""
			if len(block.Labels) > 0 {
				providerName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Provider '" + providerName + "' should be defined in providers.tf or versions.tf",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as moving blocks requires manual review.
func (r *ProvidersInFileRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// extractBasename extracts the base filename from a path.
func extractBasename(path string) string {
	// Find the last path separator
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// VariableNamingRule ensures variable names follow naming conventions.
type VariableNamingRule struct{}

// Name returns the rule identifier.
func (r *VariableNamingRule) Name() string {
	return "style.variable-naming"
}

// Description returns a human-readable description of the rule.
func (r *VariableNamingRule) Description() string {
	return "Variable names should use snake_case"
}

// Check examines variable names for naming convention compliance.
func (r *VariableNamingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "variable" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		name := block.Labels[0]
		if !snakeCaseRegex.MatchString(name) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Variable name should be snake_case: " + name,
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming requires manual review.
func (r *VariableNamingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// OutputNamingRule ensures output names follow naming conventions.
type OutputNamingRule struct{}

// Name returns the rule identifier.
func (r *OutputNamingRule) Name() string {
	return "style.output-naming"
}

// Description returns a human-readable description of the rule.
func (r *OutputNamingRule) Description() string {
	return "Output names should use snake_case"
}

// Check examines output names for naming convention compliance.
func (r *OutputNamingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "output" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		name := block.Labels[0]
		if !snakeCaseRegex.MatchString(name) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Output name should be snake_case: " + name,
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming requires manual review.
func (r *OutputNamingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// LocalNamingRule ensures local value names follow naming conventions.
type LocalNamingRule struct{}

// Name returns the rule identifier.
func (r *LocalNamingRule) Name() string {
	return "style.local-naming"
}

// Description returns a human-readable description of the rule.
func (r *LocalNamingRule) Description() string {
	return "Local value names should use snake_case"
}

// Check examines local value names for naming convention compliance.
func (r *LocalNamingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "locals" {
			continue
		}

		// Check each attribute in the locals block
		for name, attr := range block.Body.Attributes {
			if !snakeCaseRegex.MatchString(name) {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "Local value name should be snake_case: " + name,
					File:     ctx.File,
					Location: attr.Range(),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming requires manual review.
func (r *LocalNamingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
