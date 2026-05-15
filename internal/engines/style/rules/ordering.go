package rules

import (
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

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
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "for_each should be the first attribute in the block",
				File:     ctx.File,
				Location: sdk.LocationFromRange(forEachAttr.Range()),
				Severity: sdk.SeverityWarning,
			})
		}

		if countAttr != nil && firstAttr != nil && countAttr != firstAttr && forEachAttr == nil {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "count should be the first attribute in the block",
				File:     ctx.File,
				Location: sdk.LocationFromRange(countAttr.Range()),
				Severity: sdk.SeverityWarning,
			})
		}
	}

	return findings, nil
}

// Fix moves for_each/count to be first attribute in each block.
// Uses line-based reordering to preserve leading comments.
func (r *ForEachCountFirstRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	// Process each block that has for_each or count
	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" {
			continue
		}

		// Check if block has for_each or count
		hasForEach := FindAttribute(block.Body.Attributes, "for_each") != nil
		hasCount := FindAttribute(block.Body.Attributes, "count") != nil
		if !hasForEach && !hasCount {
			continue
		}

		orderedNames := GetOrderedAttrNames(block.Body)
		firstAttrs := []string{"for_each", "count"}
		if block.Type == "module" {
			firstAttrs = []string{"for_each", "count", "source", "version"}
		}
		lastAttrs := []string{"tags", "labels", "tags_all", "depends_on"}

		// Use line-based reordering to preserve leading comments
		content = ReorderBlockAttrsPreservingComments(
			content,
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			orderedNames,
			firstAttrs,
			lastAttrs,
		)
	}

	return FormatAndCleanBlankLines(content), nil
}

// LifecycleAtEndRule ensures lifecycle block is at the end of resource, data,
// module, and check blocks.
type LifecycleAtEndRule struct{}

// lifecycleHostBlockTypes lists the block types that may contain a `lifecycle`
// nested block and that this rule polices. `check` is included for TF 1.5+
// assertion blocks; while they rarely embed a lifecycle today, including the
// type keeps the rule consistent and future-proof.
var lifecycleHostBlockTypes = map[string]struct{}{
	"resource": {},
	"data":     {},
	"module":   {},
	"check":    {},
}

func isLifecycleHostBlock(blockType string) bool {
	_, ok := lifecycleHostBlockTypes[blockType]
	return ok
}

// Name returns the rule identifier.
func (r *LifecycleAtEndRule) Name() string {
	return "style.lifecycle-at-end"
}

// Description returns a human-readable description of the rule.
func (r *LifecycleAtEndRule) Description() string {
	return "Ensures lifecycle block is at the end of resource, data, module, and check blocks"
}

// Check examines resource, data, module, and check blocks for lifecycle block positioning.
func (r *LifecycleAtEndRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if !isLifecycleHostBlock(block.Type) {
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
				Message:  "lifecycle block should be at the end of the " + block.Type + " block",
				File:     ctx.File,
				Location: sdk.LocationFromRange(lifecycleBlock.Range()),
				Severity: sdk.SeverityWarning,
			})
		}
	}

	return findings, nil
}

// fixLifecyclePositionContent moves the lifecycle block to the end of the
// matching host block (resource, data, module, or check). Works entirely in
// memory - takes content as input.
func (r *LifecycleAtEndRule) fixLifecyclePositionContent(content []byte, filePath, blockType string, blockLabels []string) ([]byte, error) {
	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Find the host block
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}
		if !MatchBlockLabels(block.Labels(), blockLabels) {
			continue
		}

		// Find and remove lifecycle block
		var lifecycleBlock *hclwrite.Block
		for _, nested := range block.Body().Blocks() {
			if nested.Type() == "lifecycle" {
				lifecycleBlock = nested
				break
			}
		}

		if lifecycleBlock == nil {
			continue
		}

		// Get lifecycle block tokens (preserving content)
		lifecycleTokens := lifecycleBlock.BuildTokens(nil)

		// Remove lifecycle block
		block.Body().RemoveBlock(lifecycleBlock)

		// Re-add lifecycle block at the end
		block.Body().AppendUnstructuredTokens(lifecycleTokens)

		break
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
}

// Fix moves lifecycle blocks to the end of resource, data, module, and check blocks.
// Works entirely in memory - does NOT write to disk.
func (r *LifecycleAtEndRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	if ctx == nil || file == nil {
		return nil, nil
	}

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	// Collect (block type, labels) pairs before modifying content
	type blockRef struct {
		blockType string
		labels    []string
	}
	var blocksToFix []blockRef
	for _, block := range hclFile.Blocks {
		if !isLifecycleHostBlock(block.Type) {
			continue
		}

		// Check if this block has a lifecycle that needs moving
		var lifecycleBlock *hclsyntax.Block
		var lastLine int

		for _, nested := range block.Body.Blocks {
			if nested.Range().End.Line > lastLine {
				lastLine = nested.Range().End.Line
			}
			if nested.Type == "lifecycle" {
				lifecycleBlock = nested
			}
		}

		for _, attr := range block.Body.Attributes {
			if attr.Range().End.Line > lastLine {
				lastLine = attr.Range().End.Line
			}
		}

		if lifecycleBlock != nil && lifecycleBlock.Range().End.Line < lastLine {
			blocksToFix = append(blocksToFix, blockRef{blockType: block.Type, labels: block.Labels})
		}
	}

	// Process each block (re-parse after each modification)
	for _, ref := range blocksToFix {
		content, err = r.fixLifecyclePositionContent(content, ctx.File, ref.blockType, ref.labels)
		if err != nil {
			return nil, err
		}
		// Do NOT write to disk - work entirely in memory
	}

	return content, nil
}

// TagsAtEndRule ensures tags/labels are at the end of resource blocks (before lifecycle).
type TagsAtEndRule struct{}

// Name returns the rule identifier.
func (r *TagsAtEndRule) Name() string {
	return "style.tags-at-end"
}

// Description returns a human-readable description of the rule.
func (r *TagsAtEndRule) Description() string {
	return "Ensures tags/labels appear just before lifecycle in resource/module blocks, after all other attributes and nested blocks"
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

	lifecycleBlock := FindNestedBlock(body.Blocks, "lifecycle")
	tagsLine := tagsAttr.Range().Start.Line

	// Single-finding policy: the "before lifecycle" and "near the end" conditions both
	// describe the same logical violation (tags is in the wrong spot). Emit the
	// "before lifecycle" message when applicable since it is more specific; otherwise
	// fall back to the more general "near the end" message. This avoids two findings
	// pointing at the same line for a single misplacement.
	if lifecycleBlock != nil && tagsLine > lifecycleBlock.Range().Start.Line {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "tags should be before lifecycle block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(tagsAttr.Range()),
			Severity: sdk.SeverityWarning,
		})
	} else if countItemsAfterTags(body, tagsLine) > 0 {
		// Flag any item (attr or non-lifecycle block) after tags. The earlier behavior
		// (> 2 attrs) tolerated trailing attrs and ignored trailing nested blocks entirely,
		// so a layout like `tags = {}; ingress {}; egress {}; lifecycle {}` went unflagged
		// even though tags was nowhere near the end of the block.
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "tags should be near the end of the block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(tagsAttr.Range()),
			Severity: sdk.SeverityInfo,
		})
	}

	return findings
}

// findTagsAttribute returns the tags-family attribute the rule operates on, with a
// deterministic priority. `tags` wins over `labels`, which wins over `tags_all`.
// `tags_all` is provider-managed (derived from inherited tags) and is included only
// as a fallback for resources that use it directly; otherwise the rule prefers the
// author-controlled `tags`/`labels`.
func findTagsAttribute(attrs hclsyntax.Attributes) *hclsyntax.Attribute {
	for _, name := range []string{"tags", "labels", "tags_all"} {
		if attr, ok := attrs[name]; ok {
			return attr
		}
	}
	return nil
}

// countItemsAfterTags counts attributes and nested blocks that appear after the tags
// attribute, excluding the items the rule deliberately treats as trailing:
//   - `tags_all` (the derived attribute paired with `tags`)
//   - `depends_on` (a meta-argument that conventionally goes last)
//   - `lifecycle` (the trailing nested block)
//
// The new threshold of > 0 (down from > 2) means a single trailing item flags the
// violation. The exclusions ensure the rule does not fire on canonical layouts where
// tags is followed only by depends_on / lifecycle.
func countItemsAfterTags(body *hclsyntax.Body, tagsLine int) int {
	count := 0
	for name, attr := range body.Attributes {
		if attr.Range().Start.Line > tagsLine && name != "tags_all" && name != "depends_on" {
			count++
		}
	}
	for _, b := range body.Blocks {
		if b.Range().Start.Line > tagsLine && b.Type != "lifecycle" {
			count++
		}
	}
	return count
}

// Fix moves tags/labels (and tags_all) to land just before any lifecycle block,
// or at the end of the block body if no lifecycle is present. Other attributes and
// nested blocks stay in their source positions; only the tags region (including its
// leading comment) moves. Operates bottom-up so line ranges remain valid across rewrites.
func (r *TagsAtEndRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	syntaxFile, diags := hclsyntax.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	hclFile, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	var targets []*hclsyntax.Block
	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "module" {
			continue
		}
		if findTagsAttribute(block.Body.Attributes) == nil {
			continue
		}
		targets = append(targets, block)
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Range().Start.Line > targets[j].Range().Start.Line
	})

	for _, block := range targets {
		content = moveTagsBeforeLifecycle(content, block)
	}

	return FormatAndCleanBlankLines(content), nil
}

// moveAttrBeforeLifecycle relocates the given attribute (with its leading comment)
// inside `block` to immediately before any lifecycle nested block, or to the end of
// the block body when no lifecycle is present. All other body lines remain at their
// source positions. The move is a no-op when the attribute is already adjacent to
// the insertion point. Returns content unchanged when attr is nil.
//
// Shared by both `style.tags-at-end` (via moveTagsBeforeLifecycle wrapper) and
// `style.depends-on-order` (via moveDependsOnBeforeLifecycle wrapper).
func moveAttrBeforeLifecycle(content []byte, block *hclsyntax.Block, attr *hclsyntax.Attribute) []byte {
	if attr == nil {
		return content
	}
	attrStart := attr.Range().Start.Line
	attrEnd := attr.Range().End.Line

	var insertBefore int
	if lifecycle := FindNestedBlock(block.Body.Blocks, "lifecycle"); lifecycle != nil {
		insertBefore = lifecycle.Range().Start.Line
	} else {
		insertBefore = block.Range().End.Line
	}

	lines := SplitLines(content)

	// No-op when attr is already effectively adjacent to the insertion point: nothing
	// but blank lines sits between attrEnd+1 and insertBefore-1. A strict `attrEnd+1
	// == insertBefore` check would miss the common case of one blank line separating
	// the attribute from `lifecycle`, causing Fix to mutate files that Check considers
	// clean (the splice runs, FormatAndCleanBlankLines normalises, idempotence holds
	// but a spurious diff is produced on the first pass).
	alreadyAdjacent := true
	for line := attrEnd + 1; line < insertBefore; line++ {
		if line-1 < 0 || line-1 >= len(lines) {
			continue
		}
		if strings.TrimSpace(lines[line-1]) != "" {
			alreadyAdjacent = false
			break
		}
	}
	if alreadyAdjacent {
		return content
	}

	// Compute prior boundary for the leading-comment scan: the largest body-item end-line
	// less than attrStart, falling back to the block's opening line.
	priorEnd := block.Range().Start.Line
	for _, a := range block.Body.Attributes {
		if a.Range().End.Line < attrStart && a.Range().End.Line > priorEnd {
			priorEnd = a.Range().End.Line
		}
	}
	for _, b := range block.Body.Blocks {
		if b.Range().End.Line < attrStart && b.Range().End.Line > priorEnd {
			priorEnd = b.Range().End.Line
		}
	}

	// Capture the leading-comment line numbers (not just strings) so we can both emit
	// the moved region and skip those exact lines from their original position.
	var commentLineNums []int
	for lineNum := attrStart - 1; lineNum >= priorEnd+1; lineNum-- {
		if lineNum-1 < 0 || lineNum-1 >= len(lines) {
			continue
		}
		trimmed := strings.TrimSpace(lines[lineNum-1])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			commentLineNums = append([]int{lineNum}, commentLineNums...)
			continue
		}
		break
	}

	moved := make([]string, 0, len(commentLineNums)+(attrEnd-attrStart+1))
	for _, n := range commentLineNums {
		moved = append(moved, lines[n-1])
	}
	for line := attrStart; line <= attrEnd && line-1 < len(lines); line++ {
		moved = append(moved, lines[line-1])
	}

	skipped := make(map[int]bool, len(commentLineNums)+(attrEnd-attrStart+1))
	for _, n := range commentLineNums {
		skipped[n] = true
	}
	for line := attrStart; line <= attrEnd; line++ {
		skipped[line] = true
	}

	result := make([]string, 0, len(lines)+len(moved))
	for i, line := range lines {
		lineNum := i + 1
		if lineNum == insertBefore {
			result = append(result, moved...)
		}
		if !skipped[lineNum] {
			result = append(result, line)
		}
	}

	return []byte(strings.Join(result, "\n") + "\n")
}

// moveTagsBeforeLifecycle is a thin wrapper around moveAttrBeforeLifecycle targeting
// the tags-family attribute selected by findTagsAttribute (tags > labels > tags_all).
func moveTagsBeforeLifecycle(content []byte, block *hclsyntax.Block) []byte {
	return moveAttrBeforeLifecycle(content, block, findTagsAttribute(block.Body.Attributes))
}

// DependsOnOrderRule ensures depends_on is at the end of blocks.
type DependsOnOrderRule struct{}

// Name returns the rule identifier.
func (r *DependsOnOrderRule) Name() string {
	return "style.depends-on-order"
}

// Description returns a human-readable description of the rule.
func (r *DependsOnOrderRule) Description() string {
	return "Ensures depends_on appears just before lifecycle in resource/module blocks, after all other attributes and non-lifecycle nested blocks"
}

// Check examines blocks for depends_on attribute positioning.
func (r *DependsOnOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if !IsDependsOnRelevantBlock(block.Type) {
			continue
		}
		blockFindings := r.checkDependsOnBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

// IsDependsOnRelevantBlock checks if a block type supports depends_on attribute.
func IsDependsOnRelevantBlock(blockType string) bool {
	return blockType == "resource" || blockType == "module" || blockType == "data"
}

func (r *DependsOnOrderRule) checkDependsOnBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding
	body := block.Body

	dependsOnAttr := FindAttribute(body.Attributes, "depends_on")
	if dependsOnAttr == nil {
		return findings
	}

	lifecycleBlock := FindNestedBlock(body.Blocks, "lifecycle")
	dependsOnLine := dependsOnAttr.Range().Start.Line

	// Single-finding policy: the "before lifecycle" and "near the end" conditions both
	// describe the same logical violation. Prefer the more specific "before lifecycle"
	// message; otherwise fall back to "near the end".
	if lifecycleBlock != nil && dependsOnLine > lifecycleBlock.Range().Start.Line {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "depends_on should be before lifecycle block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(dependsOnAttr.Range()),
			Severity: sdk.SeverityWarning,
		})
	} else if countItemsAfterDependsOn(body, dependsOnLine) > 0 {
		// Warning severity: a misplacement that --fix will actively rewrite deserves at
		// least Warning. Info would hide the finding under default `severity_threshold:
		// warning` configs, leaving users surprised when `--fix` mutates their file.
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "depends_on should come after non-lifecycle nested blocks and before lifecycle (or at the end of the block)",
			File:     ctx.File,
			Location: sdk.LocationFromRange(dependsOnAttr.Range()),
			Severity: sdk.SeverityWarning,
		})
	}

	return findings
}

// countItemsAfterDependsOn counts attributes and nested blocks that appear after the
// depends_on attribute, excluding the items conventionally placed at the very tail:
//   - `tags`, `tags_all`, `labels` (the tags family lives between depends_on and lifecycle)
//   - `lifecycle` (the trailing nested block)
//
// `dependsOnLine` is the depends_on attribute's start line; the `>` comparison ensures
// depends_on itself is never counted (its own start line is never strictly greater).
// A single trailing item flags the violation.
func countItemsAfterDependsOn(body *hclsyntax.Body, dependsOnLine int) int {
	endAttrs := map[string]bool{"tags": true, "tags_all": true, "labels": true}
	count := 0
	for name, attr := range body.Attributes {
		if attr.Range().Start.Line > dependsOnLine && !endAttrs[name] {
			count++
		}
	}
	for _, b := range body.Blocks {
		if b.Range().Start.Line > dependsOnLine && b.Type != "lifecycle" {
			count++
		}
	}
	return count
}

// Fix moves depends_on to land just before any lifecycle block, or at the end of
// the block body when no lifecycle is present. Other attributes and nested blocks
// (e.g. `ordered_placement_strategy` in `aws_ecs_service`) stay at their source
// positions; only the depends_on region (including its leading comment) moves.
func (r *DependsOnOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	if ctx == nil {
		return nil, nil
	}
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	syntaxFile, diags := hclsyntax.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	hclFile, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	var targets []*hclsyntax.Block
	for _, block := range hclFile.Blocks {
		if !IsDependsOnRelevantBlock(block.Type) {
			continue
		}
		if FindAttribute(block.Body.Attributes, "depends_on") == nil {
			continue
		}
		targets = append(targets, block)
	}
	// Bottom-up so each rewrite cannot shift the line ranges of blocks above it.
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Range().Start.Line > targets[j].Range().Start.Line
	})

	for _, block := range targets {
		dependsOn := FindAttribute(block.Body.Attributes, "depends_on")
		content = moveAttrBeforeLifecycle(content, block, dependsOn)
	}

	return FormatAndCleanBlankLines(content), nil
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

	sourceAttr := FindAttribute(body.Attributes, "source")
	versionAttr := FindAttribute(body.Attributes, "version")

	if sourceAttr != nil {
		if finding := r.checkSourcePosition(ctx, body.Attributes, sourceAttr); finding != nil {
			findings = append(findings, *finding)
		}
	}

	if sourceAttr != nil && versionAttr != nil {
		if finding := r.checkVersionFollowsSource(ctx, body.Attributes, sourceAttr, versionAttr); finding != nil {
			findings = append(findings, *finding)
		}
	}

	return findings
}

func (r *SourceVersionGroupedRule) checkSourcePosition(
	ctx *sdk.Context, attrs hclsyntax.Attributes, sourceAttr *hclsyntax.Attribute,
) *sdk.Finding {
	sourceLine := sourceAttr.Range().Start.Line
	allowedBefore := map[string]bool{"source": true, "for_each": true, "count": true}

	for name, attr := range attrs {
		if !allowedBefore[name] && attr.Range().Start.Line < sourceLine {
			return &sdk.Finding{
				Rule:     r.Name(),
				Message:  "source should be at the start of module block (after for_each/count if present)",
				File:     ctx.File,
				Location: sdk.LocationFromRange(sourceAttr.Range()),
				Severity: sdk.SeverityWarning,
			}
		}
	}
	return nil
}

func (r *SourceVersionGroupedRule) checkVersionFollowsSource(
	ctx *sdk.Context, attrs hclsyntax.Attributes,
	sourceAttr, versionAttr *hclsyntax.Attribute,
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
				Location: sdk.LocationFromRange(versionAttr.Range()),
				Severity: sdk.SeverityWarning,
			}
		}
	}
	return nil
}

// Fix reorders source/version to be at the start of module blocks (after for_each/count).
// Uses line-based reordering to preserve leading comments.
func (r *SourceVersionGroupedRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	// Process each module block that has source
	for _, block := range hclFile.Blocks {
		if block.Type != "module" {
			continue
		}

		// Check if block has source
		if FindAttribute(block.Body.Attributes, "source") == nil {
			continue
		}

		orderedNames := GetOrderedAttrNames(block.Body)
		// source and version should come after for_each/count but before everything else
		firstAttrs := []string{"for_each", "count", "source", "version"}
		lastAttrs := []string{"tags", "labels", "tags_all", "depends_on"}

		// Use line-based reordering to preserve leading comments
		content = ReorderBlockAttrsPreservingComments(
			content,
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			orderedNames,
			firstAttrs,
			lastAttrs,
		)
	}

	return FormatAndCleanBlankLines(content), nil
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

	// Get attribute order from config (defaults to varAttrOrder)
	attrOrder := GetAttributeOrderFromConfig(ctx.Options, varAttrOrder)

	for _, block := range hclFile.Blocks {
		if block.Type != "variable" {
			continue
		}
		blockFindings := r.checkVariableBlock(ctx, block, attrOrder)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *VariableOrderRule) checkVariableBlock(ctx *sdk.Context, block *hclsyntax.Block, attrOrder map[string]int) []sdk.Finding {
	attrs := r.collectVariableAttrs(block.Body, attrOrder)
	return r.findOrderViolations(ctx, block, attrs)
}

func (r *VariableOrderRule) collectVariableAttrs(body *hclsyntax.Body, attrOrder map[string]int) []varAttrPos {
	var attrs []varAttrPos

	for name, attr := range body.Attributes {
		if order, ok := attrOrder[name]; ok {
			attrs = append(attrs, varAttrPos{
				name:  name,
				line:  attr.Range().Start.Line,
				order: order,
			})
		}
	}

	// Handle validation block - use order from config or default to after all attributes
	validationOrder := len(attrOrder) + 1
	if order, ok := attrOrder["validation"]; ok {
		validationOrder = order
	}

	for _, nested := range body.Blocks {
		if nested.Type == "validation" {
			attrs = append(attrs, varAttrPos{
				name:  "validation",
				line:  nested.Range().Start.Line,
				order: validationOrder,
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
	if b.line < a.line && b.order > a.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  b.name + " should come after " + a.name + " in variable block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(block.Range()),
			Severity: sdk.SeverityInfo,
		}
	}

	if a.line < b.line && a.order > b.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  a.name + " should come after " + b.name + " in variable block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(block.Range()),
			Severity: sdk.SeverityInfo,
		}
	}

	return nil
}

// Fix reorders variable attributes and nested blocks to match the canonical order:
// description, type, default, sensitive, nullable, validation blocks, then everything else.
// Heredoc bodies live within an attribute's line range and are carried along intact.
func (r *VariableOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	syntaxFile, diags := hclsyntax.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	hclFile, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	attrOrder := []string{"description", "type", "default", "sensitive", "nullable"}
	nestedBlockOrder := []string{"validation"}

	// Process variable blocks bottom-up: rewriting block N only mutates lines at or below
	// blockStartLine of N, so blocks above N keep accurate line ranges from the single parse.
	var variables []*hclsyntax.Block
	for _, block := range hclFile.Blocks {
		if block.Type == "variable" {
			variables = append(variables, block)
		}
	}
	sort.Slice(variables, func(i, j int) bool {
		return variables[i].Range().Start.Line > variables[j].Range().Start.Line
	})

	for _, block := range variables {
		content = ReorderBlockBodyPreservingAll(
			content,
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			attrOrder,
			nestedBlockOrder,
		)
	}

	return FormatAndCleanBlankLines(content), nil
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

	// Get attribute order from config (defaults to outputAttrOrder)
	attrOrder := GetAttributeOrderFromConfig(ctx.Options, outputAttrOrder)

	for _, block := range hclFile.Blocks {
		if block.Type != "output" {
			continue
		}
		blockFindings := r.checkOutputBlock(ctx, block, attrOrder)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *OutputOrderRule) checkOutputBlock(ctx *sdk.Context, block *hclsyntax.Block, attrOrder map[string]int) []sdk.Finding {
	attrs := r.collectOutputAttrs(block.Body, attrOrder)
	return r.findOutputOrderViolations(ctx, block, attrs)
}

func (r *OutputOrderRule) collectOutputAttrs(body *hclsyntax.Body, attrOrder map[string]int) []varAttrPos {
	var attrs []varAttrPos

	for name, attr := range body.Attributes {
		if order, ok := attrOrder[name]; ok {
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
	if b.line < a.line && b.order > a.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  b.name + " should come after " + a.name + " in output block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(block.Range()),
			Severity: sdk.SeverityInfo,
		}
	}

	if a.line < b.line && a.order > b.order {
		return &sdk.Finding{
			Rule:     r.Name(),
			Message:  a.name + " should come after " + b.name + " in output block",
			File:     ctx.File,
			Location: sdk.LocationFromRange(block.Range()),
			Severity: sdk.SeverityInfo,
		}
	}

	return nil
}

// Fix reorders output attributes and preserves nested precondition blocks (TF 1.2+).
//
// Layout buckets, in order:
//  1. Known attrs (description, value, sensitive, depends_on) in that order.
//  2. precondition blocks (in source order if multiple).
//  3. Any remaining attributes in source order.
//  4. Any remaining nested blocks in source order.
func (r *OutputOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	syntaxFile, diags := hclsyntax.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	hclFile, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	attrOrder := []string{"description", "value", "sensitive", "depends_on"}
	nestedBlockOrder := []string{"precondition"}

	// Process output blocks bottom-up: rewriting block N only mutates lines at or below
	// blockStartLine of N, so blocks above N keep accurate line ranges from the single parse.
	var outputs []*hclsyntax.Block
	for _, block := range hclFile.Blocks {
		if block.Type == "output" {
			outputs = append(outputs, block)
		}
	}
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].Range().Start.Line > outputs[j].Range().Start.Line
	})

	for _, block := range outputs {
		content = ReorderBlockBodyPreservingAll(
			content,
			block.Body,
			block.Range().Start.Line,
			block.Range().End.Line,
			attrOrder,
			nestedBlockOrder,
		)
	}

	return FormatAndCleanBlankLines(content), nil
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
			Location: sdk.LocationFromRange(terraformBlock.Range()),
			Severity: sdk.SeverityWarning,
		})
	}

	return findings, nil
}

// Fix moves the terraform block to the first position via line-range reorder.
// Attribute order and comments inside every untouched block are byte-for-byte preserved.
// Shares ReorderTopLevelBlocksByLineRange with ProviderBlockOrderRule.Fix intentionally —
// both rules consume the same canonical priority order.
func (r *TerraformBlockFirstRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	if ctx == nil {
		return nil, nil
	}
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	return ReorderTopLevelBlocksByLineRange(content)
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
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityWarning,
				})
			}

			// Provider should be before resources/data/modules
			if providerLine > firstResourceLine {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "provider block should come before resource/data/module blocks",
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityWarning,
				})
			}
		}
	}

	return findings, nil
}

// Fix reorders provider blocks to the canonical position via line-range reorder.
// Both rules share the same line-based helper because they consume the same canonical
// priority order (terraform, provider, variable, locals, data, resource, module, output).
func (r *ProviderBlockOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	if ctx == nil {
		return nil, nil
	}
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	return ReorderTopLevelBlocksByLineRange(content)
}

// AttributeGroupSpacingRule ensures blank lines between attribute groups in blocks.
// Groups are: meta-args (for_each/count) | source/version | main attrs | depends_on | tags/lifecycle
type AttributeGroupSpacingRule struct{}

// Name returns the rule identifier.
func (r *AttributeGroupSpacingRule) Name() string {
	return "style.attribute-group-spacing"
}

// Description returns a human-readable description of the rule.
func (r *AttributeGroupSpacingRule) Description() string {
	return "Ensures blank lines between attribute groups (meta-args, source/version, one-line attrs, block attrs, depends_on, tags) and between block-valued attributes"
}

// attrGroup represents which group an attribute belongs to.
type attrGroup int

const (
	groupMeta        attrGroup = iota // for_each, count, provider
	groupSource                       // source, version (modules only)
	groupMainOneLine                  // regular one-line attributes (key = value)
	groupMainBlock                    // attributes with block/multi-line values
	groupDependsOn                    // depends_on
	groupTags                         // tags, labels, tags_all
	groupUnknown
)

// getAttrGroup returns the group for an attribute name.
// isMultiLine indicates if the attribute value spans multiple lines.
func getAttrGroup(name string, blockType string, isMultiLine bool) attrGroup {
	// Handle variable block attributes
	if blockType == "variable" {
		// Variable blocks don't use the same grouping as resources
		// Just separate one-line from multi-line attributes
		if isMultiLine {
			return groupMainBlock
		}
		return groupMainOneLine
	}

	// Handle output block attributes
	if blockType == "output" {
		// Output blocks don't use the same grouping as resources
		// Just separate one-line from multi-line attributes
		if isMultiLine {
			return groupMainBlock
		}
		return groupMainOneLine
	}

	// Handle resource, module, and data blocks
	switch name {
	case "for_each", "count", "provider":
		return groupMeta
	case "source", "version":
		if blockType == "module" {
			return groupSource
		}
		if isMultiLine {
			return groupMainBlock
		}
		return groupMainOneLine
	case "depends_on":
		return groupDependsOn
	case "tags", "labels", "tags_all":
		return groupTags
	default:
		if isMultiLine {
			return groupMainBlock
		}
		return groupMainOneLine
	}
}

// isAttrMultiLine checks if an attribute spans multiple lines.
func isAttrMultiLine(attr *hclsyntax.Attribute) bool {
	return attr.Range().End.Line > attr.Range().Start.Line
}

// Check examines blocks for missing blank lines between attribute groups.
func (r *AttributeGroupSpacingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}
	lines := SplitLines(content)

	for _, block := range hclFile.Blocks {
		// Check resource, module, data, variable, and output blocks
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" &&
			block.Type != "variable" && block.Type != "output" {
			continue
		}

		blockFindings := r.checkBlock(ctx, block, lines)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *AttributeGroupSpacingRule) checkBlock(ctx *sdk.Context, block *hclsyntax.Block, lines []string) []sdk.Finding {
	var findings []sdk.Finding

	// Get attributes sorted by line number
	type attrInfo struct {
		name  string
		line  int
		group attrGroup
	}
	var attrs []attrInfo

	for name, attr := range block.Body.Attributes {
		isMultiLine := isAttrMultiLine(attr)
		attrs = append(attrs, attrInfo{
			name:  name,
			line:  attr.Range().Start.Line,
			group: getAttrGroup(name, block.Type, isMultiLine),
		})
	}

	// Sort by line number
	for i := 0; i < len(attrs)-1; i++ {
		for j := i + 1; j < len(attrs); j++ {
			if attrs[j].line < attrs[i].line {
				attrs[i], attrs[j] = attrs[j], attrs[i]
			}
		}
	}

	// Check for missing blank lines between groups
	for i := 0; i < len(attrs)-1; i++ {
		curr := attrs[i]
		next := attrs[i+1]

		// Determine if we need a blank line between these attributes:
		// 1. Different groups always need blank lines
		// 2. Consecutive block-valued attributes (groupMainBlock) need blank lines between them
		needsBlankLine := curr.group != next.group ||
			(curr.group == groupMainBlock && next.group == groupMainBlock)

		if !needsBlankLine {
			continue
		}

		// Check if there's a blank line between them
		hasBlankLine := false
		for lineNum := curr.line + 1; lineNum < next.line; lineNum++ {
			if lineNum-1 < len(lines) {
				trimmed := TrimLeftWhitespace(lines[lineNum-1])
				if len(trimmed) == 0 {
					hasBlankLine = true
					break
				}
			}
		}

		if !hasBlankLine {
			message := "Missing blank line between " + curr.name + " and " + next.name
			if curr.group == next.group {
				message += " (block-valued attributes should be separated)"
			} else {
				message += " (different attribute groups)"
			}
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: message,
				File:    ctx.File,
				Location: sdk.Location{
					Filename:    ctx.File,
					StartLine:   next.line,
					StartColumn: 1,
					EndLine:     next.line,
					EndColumn:   1,
				},
				Severity: sdk.SeverityInfo,
			})
		}
	}

	return findings
}

// fixBlockContent adds blank lines between attribute groups in a block.
// Works entirely in memory - takes content as input and returns modified content.
func (r *AttributeGroupSpacingRule) fixBlockContent(content []byte, filePath, blockType string, blockLabels []string) ([]byte, error) {
	lines := SplitLines(content)

	// Parse to find the block and its attributes
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	syntaxBody, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return content, nil
	}

	// Find the target block
	var targetBlock *hclsyntax.Block
	for _, block := range syntaxBody.Blocks {
		if block.Type != blockType {
			continue
		}
		if MatchBlockLabels(block.Labels, blockLabels) {
			targetBlock = block
			break
		}
	}

	if targetBlock == nil {
		return content, nil
	}

	// Get attributes sorted by line number
	type attrInfo struct {
		name    string
		line    int
		endLine int
		group   attrGroup
	}
	var attrs []attrInfo

	for name, attr := range targetBlock.Body.Attributes {
		isMultiLine := isAttrMultiLine(attr)
		attrs = append(attrs, attrInfo{
			name:    name,
			line:    attr.Range().Start.Line,
			endLine: attr.Range().End.Line,
			group:   getAttrGroup(name, blockType, isMultiLine),
		})
	}

	// Sort by line number
	for i := 0; i < len(attrs)-1; i++ {
		for j := i + 1; j < len(attrs); j++ {
			if attrs[j].line < attrs[i].line {
				attrs[i], attrs[j] = attrs[j], attrs[i]
			}
		}
	}

	// Find lines where we need to insert blank lines (after the attribute, before next)
	insertAfterLines := make(map[int]bool)

	for i := 0; i < len(attrs)-1; i++ {
		curr := attrs[i]
		next := attrs[i+1]

		// Determine if we need a blank line between these attributes:
		// 1. Different groups always need blank lines
		// 2. Consecutive block-valued attributes (groupMainBlock) need blank lines between them
		needsBlankLine := curr.group != next.group ||
			(curr.group == groupMainBlock && next.group == groupMainBlock)

		if !needsBlankLine {
			continue
		}

		// Check if there's already a blank line between them
		hasBlankLine := false
		for lineNum := curr.endLine + 1; lineNum < next.line; lineNum++ {
			if lineNum-1 < len(lines) {
				trimmed := TrimLeftWhitespace(lines[lineNum-1])
				if len(trimmed) == 0 {
					hasBlankLine = true
					break
				}
			}
		}

		if !hasBlankLine {
			// Insert blank line after the current attribute's end line
			insertAfterLines[curr.endLine] = true
		}
	}

	if len(insertAfterLines) == 0 {
		return content, nil
	}

	// Build result with inserted blank lines
	var result []string
	for i, line := range lines {
		lineNum := i + 1 // 1-indexed
		result = append(result, line)

		if insertAfterLines[lineNum] {
			result = append(result, "")
		}
	}

	return []byte(strings.Join(result, "\n") + "\n"), nil
}

// Fix adds blank lines between attribute groups.
// Works entirely in memory - does NOT write to disk.
func (r *AttributeGroupSpacingRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return content, nil
	}

	// Collect block info before modifying content (line numbers will change)
	type blockInfo struct {
		blockType string
		labels    []string
	}
	var blocks []blockInfo
	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" &&
			block.Type != "variable" && block.Type != "output" {
			continue
		}
		blocks = append(blocks, blockInfo{blockType: block.Type, labels: block.Labels})
	}

	// Process each block (re-parse after each modification)
	for _, bi := range blocks {
		content, err = r.fixBlockContent(content, ctx.File, bi.blockType, bi.labels)
		if err != nil {
			return nil, err
		}
		// Do NOT write to disk - work entirely in memory
	}

	return content, nil
}
