package rules

import (
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
)

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
	lines := SplitLines(content)

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
		trimmed := TrimLeftWhitespace(line)

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
	return FormatAndCleanBlankLines(content), nil
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
	lines := SplitLines(content)

	blocks := hclFile.Blocks
	for i := 0; i < len(blocks)-1; i++ {
		currentBlock := blocks[i]
		nextBlock := blocks[i+1]

		endLine := currentBlock.Range().End.Line
		startLine := nextBlock.Range().Start.Line

		// Count actual blank lines (excluding comments) between blocks
		blankLines := CountBlankLinesBetween(lines, endLine, startLine)

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

// fixFile fixes blank line issues in the file.
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
	lines := SplitLines(content)

	// Build a map of line numbers that need adjustments
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
		blankLines := CountBlankLinesBetween(lines, endLine, startLine)

		if blankLines < 1 {
			adjustments = append(adjustments, lineAdjustment{
				afterLine: endLine,
				action:    "add",
			})
		} else if blankLines > 1 {
			adjustments = append(adjustments, lineAdjustment{
				afterLine: endLine,
				action:    "remove",
			})
		}
	}

	if len(adjustments) == 0 {
		return content, nil
	}

	// Apply adjustments
	var result []string
	lineNum := 1
	for lineNum <= len(lines) {
		line := lines[lineNum-1]
		result = append(result, line)

		// Check for adjustments at this line
		for _, adj := range adjustments {
			if adj.afterLine == lineNum {
				if adj.action == "add" {
					result = append(result, "")
				} else if adj.action == "remove" {
					blankKept := false
					for lineNum < len(lines) {
						nextLine := lines[lineNum]
						trimmed := TrimLeftWhitespace(nextLine)
						if len(trimmed) > 0 {
							break
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

	return []byte(strings.Join(result, "\n") + "\n"), nil
}

// Fix corrects blank line issues between blocks.
func (r *BlankLineBetweenBlocksRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
	return r.fixFile(ctx.File)
}

// NoEmptyBlocksRule ensures blocks have content.
type NoEmptyBlocksRule struct{}

// Name returns the rule identifier.
func (r *NoEmptyBlocksRule) Name() string {
	return "style.no-empty-blocks"
}

// Description returns a human-readable description of the rule.
func (r *NoEmptyBlocksRule) Description() string {
	return "Ensures blocks are not empty (have at least one attribute or nested block)"
}

// Check examines the file for empty blocks.
func (r *NoEmptyBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		// Skip certain block types that are allowed to be empty
		if block.Type == "terraform" || block.Type == "required_providers" {
			continue
		}

		body := block.Body
		if len(body.Attributes) == 0 && len(body.Blocks) == 0 {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Empty block",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as empty blocks require manual review.
func (r *NoEmptyBlocksRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
