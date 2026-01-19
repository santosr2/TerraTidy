package rules

import (
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/terratidy/pkg/sdk"
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

	syntaxBody, writeFile, err := ParseBothFormats(content, filePath)
	if err != nil {
		return nil, err
	}
	if syntaxBody == nil {
		return content, nil
	}

	// Find syntax body for this block
	var targetSyntaxBody *hclsyntax.Body
	for _, block := range syntaxBody.Blocks {
		if block.Type != blockType {
			continue
		}
		if MatchBlockLabels(block.Labels, blockLabels) {
			targetSyntaxBody = block.Body
			break
		}
	}

	if targetSyntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}
		if !MatchBlockLabels(block.Labels(), blockLabels) {
			continue
		}

		orderedNames := GetOrderedAttrNames(targetSyntaxBody)
		firstAttrs := []string{"for_each", "count"}
		ReorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
		break
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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
		key := BlockKey(block.Type, block.Labels)
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
		key := BlockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := GetOrderedAttrNames(syntaxBody)
		firstAttrs := []string{"for_each", "count"}
		ReorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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
			filePath := ctx.File
			blockLabels := block.Labels
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "lifecycle block should be at the end of the resource block",
				File:     ctx.File,
				Location: lifecycleBlock.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					return r.fixLifecyclePosition(filePath, blockLabels)
				},
			})
		}
	}

	return findings, nil
}

// fixLifecyclePosition moves lifecycle block to the end of the resource.
func (r *LifecycleAtEndRule) fixLifecyclePosition(filePath string, blockLabels []string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Find the resource block
	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "resource" {
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

// Fix moves lifecycle blocks to the end of resource blocks.
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

	// Process each resource block
	for _, block := range hclFile.Blocks {
		if block.Type != "resource" {
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
			content, err = r.fixLifecyclePosition(ctx.File, block.Labels)
			if err != nil {
				return nil, err
			}
			// Write intermediate result
			if err := os.WriteFile(ctx.File, content, 0o644); err != nil {
				return nil, err
			}
		}
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

	lifecycleBlock := FindNestedBlock(body.Blocks, "lifecycle")
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

	syntaxBody, writeFile, err := ParseBothFormats(content, filePath)
	if err != nil {
		return nil, err
	}
	if syntaxBody == nil {
		return content, nil
	}

	// Find syntax body for this block
	targetSyntaxBody := FindSyntaxBody(syntaxBody, blockType, blockLabels)
	if targetSyntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	targetBlock := FindWriteBlock(writeFile, blockType, blockLabels)
	if targetBlock == nil {
		return content, nil
	}

	orderedNames := GetOrderedAttrNames(targetSyntaxBody)
	// tags/labels should be at the end
	lastAttrs := []string{"tags", "labels", "tags_all"}
	ReorderBlockAttrs(targetBlock.Body(), orderedNames, nil, lastAttrs)

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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
			key := BlockKey(block.Type, block.Labels)
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

		key := BlockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := GetOrderedAttrNames(syntaxBody)
		// tags/labels should be at the end
		lastAttrs := []string{"tags", "labels", "tags_all"}
		ReorderBlockAttrs(block.Body(), orderedNames, nil, lastAttrs)
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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

	filePath := ctx.File
	blockType := block.Type
	blockLabels := block.Labels
	fixFunc := func() ([]byte, error) {
		return r.fixDependsOnOrder(filePath, blockType, blockLabels)
	}

	lifecycleBlock := FindNestedBlock(body.Blocks, "lifecycle")
	dependsOnLine := dependsOnAttr.Range().Start.Line

	if lifecycleBlock != nil && dependsOnLine > lifecycleBlock.Range().Start.Line {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "depends_on should be before lifecycle block",
			File:     ctx.File,
			Location: dependsOnAttr.Range(),
			Severity: sdk.SeverityWarning,
			Fixable:  true,
			FixFunc:  fixFunc,
		})
	}

	if r.hasAttributesAfterDependsOn(body.Attributes, dependsOnLine) {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "depends_on should be near the end of the block",
			File:     ctx.File,
			Location: dependsOnAttr.Range(),
			Severity: sdk.SeverityInfo,
			Fixable:  true,
			FixFunc:  fixFunc,
		})
	}

	return findings
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

// fixDependsOnOrder moves depends_on to be near the end (before tags/lifecycle).
func (r *DependsOnOrderRule) fixDependsOnOrder(filePath, blockType string, blockLabels []string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	syntaxBody, writeFile, err := ParseBothFormats(content, filePath)
	if err != nil {
		return nil, err
	}
	if syntaxBody == nil {
		return content, nil
	}

	// Find syntax body for this block
	targetSyntaxBody := FindSyntaxBody(syntaxBody, blockType, blockLabels)
	if targetSyntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	targetBlock := FindWriteBlock(writeFile, blockType, blockLabels)
	if targetBlock == nil {
		return content, nil
	}

	orderedNames := GetOrderedAttrNames(targetSyntaxBody)
	// depends_on should be near the end (before tags)
	lastAttrs := []string{"depends_on", "tags", "labels", "tags_all"}
	ReorderBlockAttrs(targetBlock.Body(), orderedNames, nil, lastAttrs)

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
}

// Fix moves depends_on to be near the end of blocks.
func (r *DependsOnOrderRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	if ctx == nil || file == nil {
		return nil, nil
	}

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
		if IsDependsOnRelevantBlock(block.Type) {
			key := BlockKey(block.Type, block.Labels)
			syntaxBlocks[key] = block.Body
		}
	}

	for _, block := range writeFile.Body().Blocks() {
		if !IsDependsOnRelevantBlock(block.Type()) {
			continue
		}

		// Check if block has depends_on
		if block.Body().GetAttribute("depends_on") == nil {
			continue
		}

		key := BlockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := GetOrderedAttrNames(syntaxBody)
		lastAttrs := []string{"depends_on", "tags", "labels", "tags_all"}
		ReorderBlockAttrs(block.Body(), orderedNames, nil, lastAttrs)
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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

	syntaxBody, writeFile, err := ParseBothFormats(content, filePath)
	if err != nil {
		return nil, err
	}
	if syntaxBody == nil {
		return content, nil
	}

	// Find syntax body for this block
	targetSyntaxBody := FindSyntaxBody(syntaxBody, "module", blockLabels)
	if targetSyntaxBody == nil {
		return content, nil
	}

	// Find the matching block in hclwrite
	targetBlock := FindWriteBlock(writeFile, "module", blockLabels)
	if targetBlock == nil {
		return content, nil
	}

	orderedNames := GetOrderedAttrNames(targetSyntaxBody)
	// source and version should come after for_each/count but before everything else
	firstAttrs := []string{"for_each", "count", "source", "version"}
	ReorderBlockAttrs(targetBlock.Body(), orderedNames, firstAttrs, nil)

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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
			key := BlockKey(block.Type, block.Labels)
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

		key := BlockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := GetOrderedAttrNames(syntaxBody)
		// source and version should come after for_each/count but before everything else
		firstAttrs := []string{"for_each", "count", "source", "version"}
		ReorderBlockAttrs(block.Body(), orderedNames, firstAttrs, nil)
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
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
	attrOrder := GetAttributeOrderFromConfig(ctx.Config, varAttrOrder)

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

		// Collect all attributes with their expressions and trailing comments
		attrExprs := make(map[string]hclwrite.Tokens)
		for name, attr := range body.Attributes() {
			attrExprs[name] = getExprTokensWithTrailingComment(attr)
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

	// Clean up any leading/trailing blank lines that may have been introduced
	return FormatAndCleanBlankLines(f.Bytes()), nil
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
	attrOrder := GetAttributeOrderFromConfig(ctx.Config, outputAttrOrder)

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

		// Collect all attributes with their expressions and trailing comments
		attrExprs := make(map[string]hclwrite.Tokens)
		for name, attr := range body.Attributes() {
			attrExprs[name] = getExprTokensWithTrailingComment(attr)
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

	// Clean up any leading/trailing blank lines that may have been introduced
	return FormatAndCleanBlankLines(f.Bytes()), nil
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
		filePath := ctx.File
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "terraform block should be the first block in the file",
			File:     ctx.File,
			Location: terraformBlock.Range(),
			Severity: sdk.SeverityWarning,
			Fixable:  true,
			FixFunc: func() ([]byte, error) {
				return r.fixFile(filePath)
			},
		})
	}

	return findings, nil
}

// fixFile moves the terraform block to the beginning of the file.
func (r *TerraformBlockFirstRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	return ReorderTopLevelBlocks(writeFile), nil
}

// Fix moves terraform block to the first position.
func (r *TerraformBlockFirstRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	if ctx == nil {
		return nil, nil
	}
	return r.fixFile(ctx.File)
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

	filePath := ctx.File
	fixFunc := func() ([]byte, error) {
		return r.fixFile(filePath)
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
					Fixable:  true,
					FixFunc:  fixFunc,
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
					Fixable:  true,
					FixFunc:  fixFunc,
				})
			}
		}
	}

	return findings, nil
}

// fixFile reorders provider blocks to come after terraform and before resources.
func (r *ProviderBlockOrderRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	writeFile, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	return ReorderTopLevelBlocks(writeFile), nil
}

// Fix reorders provider blocks to proper position.
func (r *ProviderBlockOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	if ctx == nil {
		return nil, nil
	}
	return r.fixFile(ctx.File)
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
	filePath := ctx.File

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
				Location: hcl.Range{
					Filename: ctx.File,
					Start:    hcl.Pos{Line: next.line, Column: 1},
					End:      hcl.Pos{Line: next.line, Column: 1},
				},
				Severity: sdk.SeverityInfo,
				Fixable:  true,
				FixFunc: func() ([]byte, error) {
					// Fix all blocks in one pass (deduplication means only first finding's FixFunc runs)
					return r.fixAllBlocks(filePath)
				},
			})
		}
	}

	return findings
}

// fixBlock adds blank lines between attribute groups in a block.
func (r *AttributeGroupSpacingRule) fixBlock(filePath, blockType string, blockLabels []string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

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

// fixAllBlocks fixes attribute group spacing in all blocks in the file.
func (r *AttributeGroupSpacingRule) fixAllBlocks(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse to find all blocks
	syntaxFile, diags := hclsyntax.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return content, nil
	}

	syntaxBody, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return content, nil
	}

	// Process each block that needs spacing
	for _, block := range syntaxBody.Blocks {
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" &&
			block.Type != "variable" && block.Type != "output" {
			continue
		}

		// Fix this block
		content, err = r.fixBlock(filePath, block.Type, block.Labels)
		if err != nil {
			return nil, err
		}

		// Write intermediate result so next fixBlock sees updated line numbers
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			return nil, err
		}
	}

	return content, nil
}

// Fix adds blank lines between attribute groups.
func (r *AttributeGroupSpacingRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return content, nil
	}

	// Process each block
	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" &&
			block.Type != "variable" && block.Type != "output" {
			continue
		}

		// Fix this block
		content, err = r.fixBlock(ctx.File, block.Type, block.Labels)
		if err != nil {
			return nil, err
		}

		// Write intermediate result for next iteration
		if err := os.WriteFile(ctx.File, content, 0o644); err != nil {
			return nil, err
		}
	}

	return content, nil
}
