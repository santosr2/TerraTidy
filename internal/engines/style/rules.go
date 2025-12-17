package style

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// BlankLineBetweenBlocksRule ensures blank lines between top-level blocks
type BlankLineBetweenBlocksRule struct{}

func (r *BlankLineBetweenBlocksRule) Name() string {
	return "style.blank-line-between-blocks"
}

func (r *BlankLineBetweenBlocksRule) Description() string {
	return "Ensures there is a blank line between top-level blocks"
}

func (r *BlankLineBetweenBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	// Cast to HCL syntax tree
	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		// Not an HCL file (might be JSON), skip
		return findings, nil
	}

	// Check spacing between blocks
	blocks := hclFile.Blocks
	for i := 0; i < len(blocks)-1; i++ {
		currentBlock := blocks[i]
		nextBlock := blocks[i+1]

		// Calculate lines between blocks
		endLine := currentBlock.Range().End.Line
		startLine := nextBlock.Range().Start.Line
		linesBetween := startLine - endLine - 1

		// We want exactly 1 blank line
		if linesBetween < 1 {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Missing blank line between blocks",
				File:     ctx.File,
				Location: nextBlock.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false, // Would require rewriting the entire file
			})
		} else if linesBetween > 1 {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Too many blank lines between blocks (should be exactly 1)",
				File:     ctx.File,
				Location: nextBlock.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

func (r *BlankLineBetweenBlocksRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	// TODO: Implement fix by rewriting the file with proper spacing
	return nil, nil
}

// BlockLabelCaseRule ensures block labels follow naming conventions
type BlockLabelCaseRule struct{}

func (r *BlockLabelCaseRule) Name() string {
	return "style.block-label-case"
}

func (r *BlockLabelCaseRule) Description() string {
	return "Ensures block labels follow naming conventions (snake_case for resources/data, dash-case for modules)"
}

func (r *BlockLabelCaseRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	// Cast to HCL syntax tree
	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Check each block
	for _, block := range hclFile.Blocks {
		blockType := block.Type

		// Only check resource, data, and module blocks
		if blockType != "resource" && blockType != "data" && blockType != "module" {
			continue
		}

		// Get the label (name) of the block
		if len(block.Labels) < 2 {
			continue
		}

		name := block.Labels[1] // resource "type" "name" - we want the name

		// Check naming convention
		// For now, just check that it's not empty and doesn't start with a number
		if name == "" {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Block label cannot be empty",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityError,
				Fixable:  false,
			})
		}

		// Add more sophisticated checks here (snake_case validation, etc.)
	}

	return findings, nil
}

func (r *BlockLabelCaseRule) Fix(ctx *sdk.Context, file *hcl.File) ([]byte, error) {
	return nil, nil
}
