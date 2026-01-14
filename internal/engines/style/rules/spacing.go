package rules

import (
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// NoLeadingTrailingBlankLinesRule ensures no leading/trailing blank lines inside blocks.
// Internal blank lines are allowed for readability.
type NoLeadingTrailingBlankLinesRule struct{}

// Name returns the rule identifier.
func (r *NoLeadingTrailingBlankLinesRule) Name() string {
	return "style.no-leading-trailing-blank-lines"
}

// Description returns a human-readable description of the rule.
func (r *NoLeadingTrailingBlankLinesRule) Description() string {
	return "Ensures there are no leading or trailing blank lines inside blocks"
}

// Check examines the file for leading/trailing blank lines inside blocks.
func (r *NoLeadingTrailingBlankLinesRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
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

	// Check each block for leading/trailing blank lines
	for _, block := range hclFile.Blocks {
		blockFindings := r.checkBlock(ctx, block, lines)
		findings = append(findings, blockFindings...)
	}

	return findings, nil
}

func (r *NoLeadingTrailingBlankLinesRule) checkBlock(ctx *sdk.Context, block *hclsyntax.Block, lines []string) []sdk.Finding {
	var findings []sdk.Finding

	startLine := block.Range().Start.Line
	endLine := block.Range().End.Line

	// Skip if block is a single line or too small
	if endLine <= startLine+1 {
		return findings
	}

	filePath := ctx.File

	// Check for leading blank lines (lines immediately after opening brace)
	for lineNum := startLine + 1; lineNum < endLine; lineNum++ {
		if lineNum-1 >= len(lines) {
			continue
		}
		line := lines[lineNum-1]
		trimmed := TrimLeftWhitespace(line)

		if len(trimmed) == 0 {
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: "Leading blank line inside block",
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
		} else {
			break // Stop at first non-blank line
		}
	}

	// Check for trailing blank lines (lines immediately before closing brace)
	for lineNum := endLine - 1; lineNum > startLine; lineNum-- {
		if lineNum-1 >= len(lines) {
			continue
		}
		line := lines[lineNum-1]
		trimmed := TrimLeftWhitespace(line)

		if len(trimmed) == 0 {
			findings = append(findings, sdk.Finding{
				Rule:    r.Name(),
				Message: "Trailing blank line inside block",
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
		} else {
			break // Stop at first non-blank line
		}
	}

	// Also check nested blocks recursively
	for _, nested := range block.Body.Blocks {
		nestedFindings := r.checkBlock(ctx, nested, lines)
		findings = append(findings, nestedFindings...)
	}

	return findings
}

// fixFile removes leading/trailing blank lines inside blocks.
func (r *NoLeadingTrailingBlankLinesRule) fixFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return FormatAndCleanBlankLines(content), nil
}

// Fix removes leading/trailing blank lines inside blocks.
func (r *NoLeadingTrailingBlankLinesRule) Fix(ctx *sdk.Context, _ *hcl.File) ([]byte, error) {
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

// getBlankLineConfig extracts min_lines and max_lines from config.
func (r *BlankLineBetweenBlocksRule) getBlankLineConfig(config map[string]interface{}) (minLines, maxLines int) {
	minLines = 1 // Default: at least 1 blank line
	maxLines = 1 // Default: at most 1 blank line

	if config == nil {
		return minLines, maxLines
	}

	options, ok := config["options"].(map[string]interface{})
	if !ok {
		return minLines, maxLines
	}

	if min, ok := options["min_lines"].(int); ok {
		minLines = min
	} else if min, ok := options["min_lines"].(float64); ok {
		minLines = int(min)
	}

	if max, ok := options["max_lines"].(int); ok {
		maxLines = max
	} else if max, ok := options["max_lines"].(float64); ok {
		maxLines = int(max)
	}

	// Ensure min <= max
	if minLines > maxLines {
		minLines = maxLines
	}

	return minLines, maxLines
}

// Check examines the file for blank line violations between blocks.
func (r *BlankLineBetweenBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get configuration
	minLines, maxLines := r.getBlankLineConfig(ctx.Config)

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

		// Check if there's a comment between blocks
		hasComment := HasCommentBetween(lines, endLine, startLine)

		// Adjust max allowed blank lines when there's a comment
		effectiveMax := maxLines
		if hasComment {
			effectiveMax = maxLines + 1 // Allow 1 extra blank line for comments
		}

		if blankLines < minLines {
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
		} else if blankLines > effectiveMax {
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

// defaultAllowedEmptyBlocks are block types that are allowed to be empty by default.
var defaultAllowedEmptyBlocks = map[string]bool{
	"terraform":          true,
	"required_providers": true,
}

// getAllowedEmptyBlocks returns the set of block types allowed to be empty.
func (r *NoEmptyBlocksRule) getAllowedEmptyBlocks(config map[string]interface{}) map[string]bool {
	allowed := make(map[string]bool)

	// Start with defaults
	for k, v := range defaultAllowedEmptyBlocks {
		allowed[k] = v
	}

	if config == nil {
		return allowed
	}

	options, ok := config["options"].(map[string]interface{})
	if !ok {
		return allowed
	}

	// Get additional allowed blocks from config
	if allowedList, ok := options["allowed_blocks"].([]interface{}); ok {
		for _, item := range allowedList {
			if blockType, ok := item.(string); ok {
				allowed[blockType] = true
			}
		}
	}

	// Check if defaults should be overridden
	if override, ok := options["override_defaults"].(bool); ok && override {
		// Clear defaults, only use config-specified blocks
		allowed = make(map[string]bool)
		if allowedList, ok := options["allowed_blocks"].([]interface{}); ok {
			for _, item := range allowedList {
				if blockType, ok := item.(string); ok {
					allowed[blockType] = true
				}
			}
		}
	}

	return allowed
}

// Check examines the file for empty blocks.
func (r *NoEmptyBlocksRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get allowed empty blocks from config
	allowedEmpty := r.getAllowedEmptyBlocks(ctx.Config)

	for _, block := range hclFile.Blocks {
		// Skip allowed empty block types
		if allowedEmpty[block.Type] {
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
