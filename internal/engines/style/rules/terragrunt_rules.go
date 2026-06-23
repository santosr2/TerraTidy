package rules

import (
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/internal/cst"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// TerragruntIncludeFirstRule ensures top-level Terragrunt include blocks come
// first, followed by dependency blocks, then everything else. This matches the
// canonical layout of a terragrunt.hcl: parent wiring (include) precedes
// upstream wiring (dependency) precedes module configuration (inputs, locals,
// remote_state, etc.).
type TerragruntIncludeFirstRule struct{}

// Name returns the rule identifier.
func (r *TerragruntIncludeFirstRule) Name() string {
	return "style.terragrunt-include-first"
}

// Description returns a human-readable description of the rule.
func (r *TerragruntIncludeFirstRule) Description() string {
	return "Ensures top-level include blocks come first, then dependency blocks, then everything else"
}

// Check walks the top-level blocks and emits a finding for every include block
// that follows a non-include block, and every dependency block that follows
// any block other than include or dependency. The rule is structural and
// applies to any file whose top-level body contains include or dependency
// blocks; pure Terraform files produce no findings naturally.
//
// Only HCL blocks are evaluated. Top-level attributes (e.g. terragrunt's
// `inputs = { ... }`, `iam_role = "..."`) are invisible to this rule because
// they are not part of hclsyntax.Body.Blocks. A file that places `inputs =
// {}` above `include "root" { ... }` therefore produces zero findings even
// though "everything else" appears before the include — the rule scopes
// "everything else" to other top-level blocks.
func (r *TerragruntIncludeFirstRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	sawNonInclude := false
	sawOther := false
	for _, block := range hclFile.Blocks {
		switch block.Type {
		case "include":
			if sawNonInclude {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "include block should come before non-include blocks",
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityWarning,
				})
			}
		case "dependency":
			sawNonInclude = true
			if sawOther {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "dependency block should come after include blocks and before everything else",
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityWarning,
				})
			}
		default:
			sawNonInclude = true
			sawOther = true
		}
	}

	return findings, nil
}

// Fix reorders top-level include and dependency blocks into canonical position
// using the CST. The fix is gated on cst.IsTerragruntFile so it never mutates
// pure Terraform files; a hand-written .tf containing top-level include or
// dependency blocks does satisfy IsTerragruntFile (the heuristic recognizes
// the block names regardless of extension) and is fixed in the same way.
//
// Under DefaultTopLevelPolicy (StrictAdjacency=true), StandaloneComment items
// separated from a block by a blank line stay where they are when the block
// moves — section headers (the `### Inputs`-style comments common in
// terragrunt.hcl) survive the reorder intact.
func (r *TerragruntIncludeFirstRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	if !cst.IsTerragruntFile(file, ctx.File) {
		return nil, nil
	}

	if terragruntBlocksAreCanonical(file.Body) {
		return nil, nil
	}

	firstBlockIdx := -1
	for i, item := range file.Body.Items {
		if _, ok := item.(*cst.Block); ok {
			firstBlockIdx = i
			break
		}
	}
	if firstBlockIdx < 0 {
		return nil, nil
	}

	var anchor cst.BodyItem
	place := func(item cst.BodyItem) {
		if anchor == nil {
			file.Body.Move(item, firstBlockIdx)
		} else {
			file.Body.MoveAfter(item, anchor)
		}
		anchor = item
	}
	for _, blk := range file.Body.FindBlocksByType("include") {
		place(blk)
	}
	for _, blk := range file.Body.FindBlocksByType("dependency") {
		place(blk)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// terragruntBlocksAreCanonical reports whether the *cst.Block items in body
// already appear in canonical order: include blocks first, then dependency
// blocks, then everything else. BlankLine and StandaloneComment items are
// ignored. When true, Fix would only redistribute BlankLines — return early
// to keep the input byte-identical.
func terragruntBlocksAreCanonical(body *cst.Body) bool {
	prevPrio := 0
	for _, item := range body.Items {
		blk, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		var p int
		switch blk.Type {
		case "include":
			p = 1
		case "dependency":
			p = 2
		default:
			p = 3
		}
		if p < prevPrio {
			return false
		}
		prevPrio = p
	}
	return true
}
