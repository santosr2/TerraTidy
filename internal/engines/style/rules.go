// Package style provides the style engine and rules for TerraTidy.
// It enforces consistent code style and formatting conventions in Terraform
// configurations, such as attribute ordering, block spacing, and naming conventions.
package style

import (
	"os"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// snakeCaseRegex matches valid snake_case identifiers
var snakeCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

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

	blocks := hclFile.Blocks
	for i := 0; i < len(blocks)-1; i++ {
		currentBlock := blocks[i]
		nextBlock := blocks[i+1]

		endLine := currentBlock.Range().End.Line
		startLine := nextBlock.Range().Start.Line
		linesBetween := startLine - endLine - 1

		if linesBetween < 1 {
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
		} else if linesBetween > 1 {
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

// fixFile fixes blank line issues in the file
func (r *BlankLineBetweenBlocksRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse with hclwrite to work with the structure
	f, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Get the formatted output which handles spacing properly
	return f.Bytes(), nil
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
	attrName string,
) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	f, diags := hclwrite.ParseConfig(content, filePath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// Find the matching block
	for _, block := range f.Body().Blocks() {
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

		// Get the attribute
		attr := block.Body().GetAttribute(attrName)
		if attr == nil {
			continue
		}

		// Get the expression tokens
		exprTokens := attr.Expr().BuildTokens(nil)

		// Remove the attribute
		block.Body().RemoveAttribute(attrName)

		// Re-add at the beginning by setting and relying on hclwrite's ordering
		// hclwrite doesn't have a "prepend" method, so we rebuild the body
		// Get all current attributes
		oldAttrs := make(map[string]*hclwrite.Attribute)
		for name, a := range block.Body().Attributes() {
			oldAttrs[name] = a
		}

		// Clear all attributes
		for name := range oldAttrs {
			block.Body().RemoveAttribute(name)
		}

		// Add the target attribute first
		block.Body().SetAttributeRaw(attrName, exprTokens)

		// Re-add other attributes
		for name, a := range oldAttrs {
			block.Body().SetAttributeRaw(name, a.Expr().BuildTokens(nil))
		}
	}

	return f.Bytes(), nil
}

// Fix moves for_each/count to be first attribute in each block.
func (r *ForEachCountFirstRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	// Fix all blocks in the file
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	f, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	for _, block := range f.Body().Blocks() {
		if block.Type() != "resource" && block.Type() != "module" && block.Type() != "data" {
			continue
		}

		// Check for for_each first, then count
		for _, attrName := range []string{"for_each", "count"} {
			attr := block.Body().GetAttribute(attrName)
			if attr == nil {
				continue
			}

			// Get expression tokens
			exprTokens := attr.Expr().BuildTokens(nil)

			// Get all current attributes
			oldAttrs := make(map[string]*hclwrite.Attribute)
			for name, a := range block.Body().Attributes() {
				oldAttrs[name] = a
			}

			// Clear and rebuild with attrName first
			for name := range oldAttrs {
				block.Body().RemoveAttribute(name)
			}

			block.Body().SetAttributeRaw(attrName, exprTokens)

			for name, a := range oldAttrs {
				if name != attrName {
					block.Body().SetAttributeRaw(name, a.Expr().BuildTokens(nil))
				}
			}

			break // Only process for_each OR count, not both
		}
	}

	return f.Bytes(), nil
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

		body := block.Body

		var tagsAttr *hclsyntax.Attribute
		var lifecycleLine int
		var lastAttrLine int

		// Find tags and lifecycle positions
		for name, attr := range body.Attributes {
			if name == "tags" || name == "labels" || name == "tags_all" {
				tagsAttr = attr
			}
			if attr.Range().End.Line > lastAttrLine {
				lastAttrLine = attr.Range().End.Line
			}
		}

		for _, nested := range body.Blocks {
			if nested.Type == "lifecycle" {
				lifecycleLine = nested.Range().Start.Line
			}
		}

		// If tags exists, check it's positioned correctly
		if tagsAttr != nil {
			tagsLine := tagsAttr.Range().Start.Line

			// Tags should be after most attributes but before lifecycle
			if lifecycleLine > 0 && tagsLine > lifecycleLine {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "tags should be before lifecycle block",
					File:     ctx.File,
					Location: tagsAttr.Range(),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}

			// Check if there are many attributes after tags (excluding lifecycle-related)
			attrsAfterTags := 0
			for name, attr := range body.Attributes {
				if attr.Range().Start.Line > tagsLine && name != "tags_all" {
					attrsAfterTags++
				}
			}

			if attrsAfterTags > 2 { // Allow a couple of attributes after
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "tags should be near the end of the block",
					File:     ctx.File,
					Location: tagsAttr.Range(),
					Severity: sdk.SeverityInfo,
					Fixable:  false,
				})
			}
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as tags reordering requires manual review.
func (r *TagsAtEndRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
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

		body := block.Body

		var sourceAttr, versionAttr *hclsyntax.Attribute
		firstAttrLine := int(^uint(0) >> 1)

		for name, attr := range body.Attributes {
			if attr.Range().Start.Line < firstAttrLine {
				firstAttrLine = attr.Range().Start.Line
			}
			if name == "source" {
				sourceAttr = attr
			}
			if name == "version" {
				versionAttr = attr
			}
		}

		// Check if source is first (after for_each/count)
		if sourceAttr != nil {
			sourceLine := sourceAttr.Range().Start.Line

			// Allow for_each/count to be before source
			for name, attr := range body.Attributes {
				if name != "source" && name != "for_each" && name != "count" &&
					attr.Range().Start.Line < sourceLine {
					findings = append(findings, sdk.Finding{
						Rule:     r.Name(),
						Message:  "source should be at the start of module block (after for_each/count if present)",
						File:     ctx.File,
						Location: sourceAttr.Range(),
						Severity: sdk.SeverityWarning,
						Fixable:  false,
					})
					break
				}
			}
		}

		// Check if version immediately follows source
		if sourceAttr != nil && versionAttr != nil {
			sourceLine := sourceAttr.Range().End.Line
			versionLine := versionAttr.Range().Start.Line

			// Check for attributes between source and version
			for name, attr := range body.Attributes {
				attrLine := attr.Range().Start.Line
				if name != "source" && name != "version" &&
					attrLine > sourceLine && attrLine < versionLine {
					findings = append(findings, sdk.Finding{
						Rule:     r.Name(),
						Message:  "version should immediately follow source in module block",
						File:     ctx.File,
						Location: versionAttr.Range(),
						Severity: sdk.SeverityWarning,
						Fixable:  false,
					})
					break
				}
			}
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as source/version reordering requires manual review.
func (r *SourceVersionGroupedRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
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
