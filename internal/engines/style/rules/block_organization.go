package rules

import (
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// MetaArgumentsOrderRule ensures meta-arguments (for_each, count, depends_on, provider) are ordered correctly.
type MetaArgumentsOrderRule struct{}

// Name returns the rule identifier.
func (r *MetaArgumentsOrderRule) Name() string {
	return "style.meta-arguments-order"
}

// Description returns a human-readable description of the rule.
func (r *MetaArgumentsOrderRule) Description() string {
	return "Ensures meta-arguments are ordered: for_each/count first, provider, depends_on last"
}

// metaArgOrder defines the expected order for meta-arguments
var metaArgOrder = map[string]int{
	"for_each":   1, // First (mutually exclusive with count)
	"count":      1, // First (mutually exclusive with for_each)
	"provider":   2, // After for_each/count
	"depends_on": 3, // Last meta-argument
}

// Check examines blocks for meta-argument ordering.
func (r *MetaArgumentsOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "data" && block.Type != "module" {
			continue
		}

		blockFindings := r.checkBlock(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *MetaArgumentsOrderRule) checkBlock(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding

	// Collect meta-arguments with their positions
	type metaArg struct {
		name  string
		line  int
		order int
	}
	var metaArgs []metaArg

	for name, attr := range block.Body.Attributes {
		if order, ok := metaArgOrder[name]; ok {
			metaArgs = append(metaArgs, metaArg{
				name:  name,
				line:  attr.Range().Start.Line,
				order: order,
			})
		}
	}

	if len(metaArgs) < 2 {
		return findings
	}

	// Check ordering
	for i := 0; i < len(metaArgs)-1; i++ {
		for j := i + 1; j < len(metaArgs); j++ {
			a, b := metaArgs[i], metaArgs[j]
			// If a appears before b in file but should come after
			if a.line < b.line && a.order > b.order {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  a.name + " should come after " + b.name,
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityInfo,
				})
			}
			// If b appears before a in file but should come after
			if b.line < a.line && b.order > a.order {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  b.name + " should come after " + a.name,
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityInfo,
				})
			}
		}
	}

	return findings
}

// Fix reorders meta-arguments in all blocks.
func (r *MetaArgumentsOrderRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
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

	syntaxBlocks := make(map[string]*hclsyntax.Body)
	for _, block := range syntaxFile.Blocks {
		if block.Type == "resource" || block.Type == "data" || block.Type == "module" {
			key := BlockKey(block.Type, block.Labels)
			syntaxBlocks[key] = block.Body
		}
	}

	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "resource" && block.Type() != "data" && block.Type() != "module" {
			continue
		}

		key := BlockKey(block.Type(), block.Labels())
		syntaxBody, ok := syntaxBlocks[key]
		if !ok {
			continue
		}

		orderedNames := GetOrderedAttrNames(syntaxBody)
		firstAttrs := []string{"for_each", "count", "provider"}
		lastAttrs := []string{"depends_on"}
		ReorderBlockAttrs(block.Body(), orderedNames, firstAttrs, lastAttrs)
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
}

// LifecycleAttributeOrderRule ensures lifecycle block attributes are ordered correctly.
type LifecycleAttributeOrderRule struct{}

// Name returns the rule identifier.
func (r *LifecycleAttributeOrderRule) Name() string {
	return "style.lifecycle-attribute-order"
}

// Description returns a human-readable description of the rule.
func (r *LifecycleAttributeOrderRule) Description() string {
	return "Ensures lifecycle block attributes are ordered: create_before_destroy, prevent_destroy, ignore_changes, replace_triggered_by"
}

// lifecycleAttrOrder defines the expected order for lifecycle attributes
var lifecycleAttrOrder = map[string]int{
	"create_before_destroy": 1,
	"prevent_destroy":       2,
	"ignore_changes":        3,
	"replace_triggered_by":  4,
	"precondition":          5,
	"postcondition":         6,
}

// Check examines lifecycle blocks for attribute ordering.
func (r *LifecycleAttributeOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" {
			continue
		}

		for _, nested := range block.Body.Blocks {
			if nested.Type != "lifecycle" {
				continue
			}

			blockFindings := r.checkLifecycleBlock(ctx, nested)
			findings = append(findings, blockFindings...)
		}
	}

	return findings, nil
}

func (r *LifecycleAttributeOrderRule) checkLifecycleBlock(ctx *sdk.Context, lifecycleBlock *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding

	// Collect lifecycle attributes with their positions
	type lcAttr struct {
		name  string
		line  int
		order int
	}
	var attrs []lcAttr

	for name, attr := range lifecycleBlock.Body.Attributes {
		if order, ok := lifecycleAttrOrder[name]; ok {
			attrs = append(attrs, lcAttr{
				name:  name,
				line:  attr.Range().Start.Line,
				order: order,
			})
		}
	}

	if len(attrs) < 2 {
		return findings
	}

	// Check ordering
	for i := 0; i < len(attrs)-1; i++ {
		for j := i + 1; j < len(attrs); j++ {
			a, b := attrs[i], attrs[j]
			if a.line < b.line && a.order > b.order {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  a.name + " should come after " + b.name + " in lifecycle block",
					File:     ctx.File,
					Location: sdk.LocationFromRange(lifecycleBlock.Range()),
					Severity: sdk.SeverityInfo,
				})
			}
			if b.line < a.line && b.order > a.order {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  b.name + " should come after " + a.name + " in lifecycle block",
					File:     ctx.File,
					Location: sdk.LocationFromRange(lifecycleBlock.Range()),
					Severity: sdk.SeverityInfo,
				})
			}
		}
	}

	return findings
}

// Fix reorders lifecycle attributes in all blocks.
func (r *LifecycleAttributeOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	writeFile, diags := hclwrite.ParseConfig(content, ctx.File, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	for _, block := range writeFile.Body().Blocks() {
		if block.Type() != "resource" {
			continue
		}

		for _, nested := range block.Body().Blocks() {
			if nested.Type() != "lifecycle" {
				continue
			}

			attrOrder := []string{"create_before_destroy", "prevent_destroy", "ignore_changes", "replace_triggered_by"}

			attrExprs := make(map[string]hclwrite.Tokens)
			for name, attr := range nested.Body().Attributes() {
				attrExprs[name] = getExprTokensWithTrailingComment(attr)
			}

			for _, name := range attrOrder {
				nested.Body().RemoveAttribute(name)
			}

			for _, name := range attrOrder {
				if tokens, ok := attrExprs[name]; ok {
					nested.Body().SetAttributeRaw(name, tokens)
				}
			}

			for name, tokens := range attrExprs {
				found := false
				for _, orderedName := range attrOrder {
					if name == orderedName {
						found = true
						break
					}
				}
				if !found {
					nested.Body().SetAttributeRaw(name, tokens)
				}
			}
		}
	}

	return FormatAndCleanBlankLines(writeFile.Bytes()), nil
}

// NestedBlockOrderRule ensures nested blocks follow consistent ordering.
type NestedBlockOrderRule struct{}

// Name returns the rule identifier.
func (r *NestedBlockOrderRule) Name() string {
	return "style.nested-block-order"
}

// Description returns a human-readable description of the rule.
func (r *NestedBlockOrderRule) Description() string {
	return "Ensures nested blocks are ordered consistently (timeouts, connection, provisioner, lifecycle)"
}

// nestedBlockOrder defines the expected order for common nested blocks
var nestedBlockOrder = map[string]int{
	"timeouts":    1,
	"connection":  2,
	"provisioner": 3,
	"lifecycle":   99, // Always last
}

// Check examines blocks for nested block ordering.
func (r *NestedBlockOrderRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "data" {
			continue
		}

		blockFindings := r.checkNestedBlocks(ctx, block)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *NestedBlockOrderRule) checkNestedBlocks(ctx *sdk.Context, block *hclsyntax.Block) []sdk.Finding {
	var findings []sdk.Finding

	// Collect nested blocks with their positions
	type nestedInfo struct {
		blockType string
		line      int
		order     int
	}
	var nestedBlocks []nestedInfo

	for _, nested := range block.Body.Blocks {
		order, ok := nestedBlockOrder[nested.Type]
		if !ok {
			order = 50 // Default order for unknown blocks
		}
		nestedBlocks = append(nestedBlocks, nestedInfo{
			blockType: nested.Type,
			line:      nested.Range().Start.Line,
			order:     order,
		})
	}

	if len(nestedBlocks) < 2 {
		return findings
	}

	// Check ordering
	for i := 0; i < len(nestedBlocks)-1; i++ {
		for j := i + 1; j < len(nestedBlocks); j++ {
			a, b := nestedBlocks[i], nestedBlocks[j]
			if a.line < b.line && a.order > b.order {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  a.blockType + " block should come after " + b.blockType + " block",
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityInfo,
				})
			}
		}
	}

	return findings
}

// Fix is a no-op for this rule as reordering nested blocks is complex.
func (r *NestedBlockOrderRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// OneLineAttributeSpacingRule ensures one-line attributes are grouped together.
// This is a variant of attribute-group-spacing specifically for one-line vs block attributes.
type OneLineAttributeSpacingRule struct{}

// Name returns the rule identifier.
func (r *OneLineAttributeSpacingRule) Name() string {
	return "style.one-line-attribute-spacing"
}

// Description returns a human-readable description of the rule.
func (r *OneLineAttributeSpacingRule) Description() string {
	return "Ensures one-line attributes are grouped together, separated from block-valued attributes"
}

// Check examines blocks for proper grouping of one-line vs block attributes.
func (r *OneLineAttributeSpacingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
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
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" {
			continue
		}

		blockFindings := r.checkBlock(ctx, block, lines)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *OneLineAttributeSpacingRule) checkBlock(ctx *sdk.Context, block *hclsyntax.Block, lines []string) []sdk.Finding {
	var findings []sdk.Finding

	// Collect attributes with their line info
	type attrInfo struct {
		name        string
		startLine   int
		endLine     int
		isMultiLine bool
	}
	var attrs []attrInfo

	for name, attr := range block.Body.Attributes {
		isMulti := attr.Range().End.Line > attr.Range().Start.Line
		attrs = append(attrs, attrInfo{
			name:        name,
			startLine:   attr.Range().Start.Line,
			endLine:     attr.Range().End.Line,
			isMultiLine: isMulti,
		})
	}

	// Sort by line number
	for i := 0; i < len(attrs)-1; i++ {
		for j := i + 1; j < len(attrs); j++ {
			if attrs[j].startLine < attrs[i].startLine {
				attrs[i], attrs[j] = attrs[j], attrs[i]
			}
		}
	}

	// Check for one-line attrs after multi-line attrs without proper separation
	lastMultiLineEnd := 0
	for i, attr := range attrs {
		if attr.isMultiLine {
			lastMultiLineEnd = attr.endLine
		} else if lastMultiLineEnd > 0 && i > 0 {
			// This is a one-line attr after a multi-line attr
			prevAttr := attrs[i-1]
			if prevAttr.isMultiLine {
				// Check if there's a blank line between them
				hasBlank := false
				for lineNum := prevAttr.endLine + 1; lineNum < attr.startLine; lineNum++ {
					if lineNum-1 < len(lines) && strings.TrimSpace(lines[lineNum-1]) == "" {
						hasBlank = true
						break
					}
				}
				if !hasBlank && attr.startLine-prevAttr.endLine == 1 {
					findings = append(findings, sdk.Finding{
						Rule:    r.Name(),
						Message: "Consider adding a blank line between block-valued and one-line attributes",
						File:    ctx.File,
						Location: sdk.Location{
							Filename:    ctx.File,
							StartLine:   attr.startLine,
							StartColumn: 1,
							EndLine:     attr.startLine,
							EndColumn:   1,
						},
						Severity: sdk.SeverityInfo,
					})
				}
			}
		}
	}

	return findings
}

// Fix is a no-op for this rule as it provides informational suggestions.
func (r *OneLineAttributeSpacingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
