package rules

import (
	"bytes"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/internal/cst"
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

// Fix moves for_each/count to the first position in each resource/module/data
// block body. The move is narrow: only the meta-argument relocates; other
// attributes and nested blocks stay at their source positions. Sibling rules
// (source-version-grouped, tags-at-end, depends-on-order) handle the other
// canonical-ordering pieces in the engine pipeline.
//
// `for_each` wins over `count` when both are somehow present (Terraform itself
// rejects the combination at validate-time, so the precedence is academic;
// matched here to the historical Check policy that flags `for_each` first).
func (r *ForEachCountFirstRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		// No-op on parse error: do not mutate a partial tree, and do not
		// surface the diagnostic as a Fix error (Check already produced it).
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		if block.Type != "resource" && block.Type != "module" && block.Type != "data" {
			continue
		}
		moveForEachOrCountToFront(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// moveForEachOrCountToFront relocates `for_each` (or `count` when `for_each`
// is absent) to index 0 of body.Items. A no-op when body is nil, when neither
// attribute exists, or when the attribute is already at index 0.
func moveForEachOrCountToFront(body *cst.Body) {
	if body == nil {
		return
	}
	if forEach := body.FindAttribute("for_each"); forEach != nil {
		body.Move(forEach, 0)
		return
	}
	if count := body.FindAttribute("count"); count != nil {
		body.Move(count, 0)
	}
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

// Fix moves the lifecycle nested block to the last position inside each
// host block (resource/data/module/check) that contains one. Other body
// items stay at their source positions; only the lifecycle region moves.
// No-op when no host block contains a lifecycle or when lifecycle is
// already in the canonical position.
func (r *LifecycleAtEndRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		// No-op on parse error: do not mutate a partial tree, and do not
		// surface the diagnostic as a Fix error (Check already produced it).
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		if !isLifecycleHostBlock(block.Type) {
			continue
		}
		moveLifecycleToEnd(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// moveLifecycleToEnd relocates the lifecycle nested block in body to the
// last position in body.Items. A no-op when body is nil, no lifecycle
// block exists, or lifecycle is already at the last index.
func moveLifecycleToEnd(body *cst.Body) {
	if body == nil {
		return
	}
	lifecycle := body.FindBlock("lifecycle")
	if lifecycle == nil {
		return
	}
	body.Move(lifecycle, len(body.Items)-1)
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
// or at the last position in the block body when no lifecycle is present. Other
// attributes and nested blocks stay in their source positions; only the tags
// region (including its leading comment) moves. Direction-agnostic: tags below
// lifecycle is relocated above lifecycle in the same call.
func (r *TagsAtEndRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		// No-op on parse error: do not mutate a partial tree, and do not
		// surface the diagnostic as a Fix error (Check already produced it).
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		if block.Type != "resource" && block.Type != "module" {
			continue
		}
		moveTagsAttrToEnd(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// moveTagsAttrToEnd relocates the tags-family attribute in body (priority:
// tags > labels > tags_all) to immediately before any lifecycle nested block,
// or to the last position in the body when no lifecycle is present. A no-op
// when no tags-family attribute exists, when the attribute is already in the
// canonical position, or when body is nil.
func moveTagsAttrToEnd(body *cst.Body) {
	if body == nil {
		return
	}
	tagsAttr := findTagsCSTAttribute(body)
	if tagsAttr == nil {
		return
	}
	if lifecycle := body.FindBlock("lifecycle"); lifecycle != nil {
		body.MoveBefore(tagsAttr, lifecycle)
		return
	}
	body.Move(tagsAttr, len(body.Items)-1)
}

// findTagsCSTAttribute returns the tags-family attribute in body with a
// deterministic priority. `tags` wins over `labels`, which wins over
// `tags_all`. `tags_all` is provider-managed (derived from inherited tags)
// and is included only as a fallback for resources that use it directly;
// otherwise the rule prefers the author-controlled `tags`/`labels`.
func findTagsCSTAttribute(body *cst.Body) *cst.Attribute {
	for _, name := range []string{"tags", "labels", "tags_all"} {
		if attr := body.FindAttribute(name); attr != nil {
			return attr
		}
	}
	return nil
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

// Fix moves depends_on to land just before any lifecycle block, or to the
// last position in the block body when no lifecycle is present. Other
// attributes and nested blocks (e.g. `ordered_placement_strategy` in
// `aws_ecs_service`) stay at their source positions; only the depends_on
// region moves. Leading comments on depends_on travel with it (carried in
// the attribute's raw bytes).
//
// Already-canonical layouts return a nil FixResult. Canonical means
// depends_on is followed only by blank lines, standalone comments, and/or
// tags-family attributes, ending in either a lifecycle block or end-of-body.
// See isDependsOnCanonicallyPlaced.
func (r *DependsOnOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		// No-op on parse error: do not mutate a partial tree, and do not
		// surface the diagnostic as a Fix error (Check already produced it).
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		if !IsDependsOnRelevantBlock(block.Type) {
			continue
		}
		moveDependsOnToEnd(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// moveDependsOnToEnd relocates depends_on in body to immediately before any
// lifecycle nested block, or to the last position in the body when no
// lifecycle is present. A no-op when no depends_on attribute exists, when
// depends_on is already canonically placed (see isDependsOnCanonicallyPlaced),
// or when body is nil.
func moveDependsOnToEnd(body *cst.Body) {
	if body == nil {
		return
	}
	dependsOn := body.FindAttribute("depends_on")
	if dependsOn == nil {
		return
	}
	if isDependsOnCanonicallyPlaced(body, dependsOn) {
		return
	}
	if lifecycle := body.FindBlock("lifecycle"); lifecycle != nil {
		body.MoveBefore(dependsOn, lifecycle)
		return
	}
	body.Move(dependsOn, len(body.Items)-1)
}

// isDependsOnCanonicallyPlaced returns true when depends_on already sits in
// a position the Check rule considers non-violating. Mirrors the policy
// encoded in countItemsAfterDependsOn (line-based check), adapted to CST
// item-list shape: depends_on may be followed only by blank lines, standalone
// comments, and/or tags-family attributes (tags, tags_all, labels), ending in
// either a lifecycle block or end-of-body. Anything else — a non-tags attribute,
// a non-lifecycle nested block — after depends_on flags a Check finding, so
// Fix must rewrite.
//
// Without this guard, MoveBefore would shuffle an intervening BlankLine
// between depends_on and lifecycle to the wrong side of depends_on, producing
// a non-trivial diff on input that Check considers already clean — and
// breaking the documented Fix/Check semantic-gap contract pinned by
// `Fix is a no-op (no diff) when depends_on is already adjacent to lifecycle
// with a blank gap` in ordering_test.go.
func isDependsOnCanonicallyPlaced(body *cst.Body, dependsOn *cst.Attribute) bool {
	idx := -1
	for i, item := range body.Items {
		if attr, ok := item.(*cst.Attribute); ok && attr == dependsOn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return true
	}
	for i := idx + 1; i < len(body.Items); i++ {
		switch v := body.Items[i].(type) {
		case *cst.BlankLine, *cst.StandaloneComment:
			continue
		case *cst.Block:
			return v.Type == "lifecycle"
		case *cst.Attribute:
			if v.Name == "tags" || v.Name == "tags_all" || v.Name == "labels" {
				continue
			}
			return false
		default:
			// Fail-safe: a future BodyItem variant defaults to non-canonical so
			// Fix rewrites rather than silently treating unknown items as
			// passthrough.
			return false
		}
	}
	return true
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

// Fix relocates `source` to the start of each module block (after any
// `for_each`/`count` meta-argument) and pulls `version` to immediately follow
// `source`. The move is narrow: only those two attributes relocate; the rest
// of the body stays at its source positions. Sibling rules
// (for-each-count-first, tags-at-end, depends-on-order) handle the other
// canonical-ordering pieces in the engine pipeline.
func (r *SourceVersionGroupedRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		if block.Type != "module" {
			continue
		}
		groupSourceVersionInModule(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// groupSourceVersionInModule places `source` at the start of body (after any
// `for_each` or `count` meta-argument) and, when present, pulls `version` to
// immediately follow `source`. A no-op when body is nil or when `source` is
// absent. `for_each` and `count` cannot coexist in valid Terraform; whichever
// is present is used as the anchor.
func groupSourceVersionInModule(body *cst.Body) {
	if body == nil {
		return
	}
	source := body.FindAttribute("source")
	if source == nil {
		return
	}

	var anchor cst.BodyItem
	if forEach := body.FindAttribute("for_each"); forEach != nil {
		anchor = forEach
	} else if count := body.FindAttribute("count"); count != nil {
		anchor = count
	}
	if anchor != nil {
		body.MoveAfter(source, anchor)
	} else {
		body.Move(source, 0)
	}

	if version := body.FindAttribute("version"); version != nil {
		body.MoveAfter(version, source)
	}
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

// Fix reorders variable block bodies to the canonical sequence:
//
//  1. Known attributes in the order: description, type, default, sensitive, nullable.
//  2. validation blocks (in their original relative order when multiple).
//  3. Everything else — non-canonical attributes and unknown nested blocks — stays
//     where it was in the body's item list, shifting only as a side-effect of
//     the canonical items moving past it.
//
// Leading comments on each attribute/block travel with that item (they live
// in the item's raw bytes). Heredoc bodies and inline trailing comments are
// likewise carried intact.
//
// The Fix-time attribute order is fixed; the Check method honors
// config-provided overrides for finding emission, but Fix always rewrites to
// the canonical sequence (matching the historical hclwrite-helper behavior).
func (r *VariableOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok || block.Type != "variable" {
			continue
		}
		reorderVariableBody(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// variableAttrFixOrder is the canonical Fix-time attribute order for variable
// blocks. Sibling: variableNestedBlockOrder.
var variableAttrFixOrder = []string{"description", "type", "default", "sensitive", "nullable"}

// variableNestedBlockOrder is the canonical Fix-time nested-block order for
// variable blocks. Sibling: variableAttrFixOrder.
var variableNestedBlockOrder = []string{"validation"}

// reorderVariableBody walks variableAttrFixOrder then variableNestedBlockOrder
// and threads each found item into a contiguous canonical prefix at the head
// of body.Items via Move/MoveAfter. Items not in either list stay where they
// were and end up after the canonical prefix (since the prefix moves past
// them).
func reorderVariableBody(body *cst.Body) {
	if body == nil {
		return
	}
	var anchor cst.BodyItem
	place := func(item cst.BodyItem) {
		if anchor == nil {
			body.Move(item, 0)
		} else {
			body.MoveAfter(item, anchor)
		}
		anchor = item
	}
	for _, name := range variableAttrFixOrder {
		if attr := body.FindAttribute(name); attr != nil {
			place(attr)
		}
	}
	for _, blockType := range variableNestedBlockOrder {
		for _, blk := range body.FindBlocksByType(blockType) {
			place(blk)
		}
	}
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

// Fix reorders output block bodies to the canonical sequence:
//
//  1. Known attributes in the order: description, value, sensitive, depends_on.
//  2. precondition blocks (in their original relative order when multiple).
//  3. Everything else stays where it was, shifting only as a side-effect of
//     the canonical items moving past it.
//
// As in VariableOrderRule, the Fix-time order is fixed regardless of any
// config override that the Check method honors for emission.
func (r *OutputOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok || block.Type != "output" {
			continue
		}
		reorderOutputBody(block.Body)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// outputAttrFixOrder is the canonical Fix-time attribute order for output
// blocks. Sibling: outputNestedBlockOrder.
var outputAttrFixOrder = []string{"description", "value", "sensitive", "depends_on"}

// outputNestedBlockOrder is the canonical Fix-time nested-block order for
// output blocks. Sibling: outputAttrFixOrder.
var outputNestedBlockOrder = []string{"precondition"}

// reorderOutputBody threads outputAttrFixOrder and outputNestedBlockOrder
// into a contiguous canonical prefix at the head of body.Items via
// Move/MoveAfter, leaving non-canonical items where they were.
func reorderOutputBody(body *cst.Body) {
	if body == nil {
		return
	}
	var anchor cst.BodyItem
	place := func(item cst.BodyItem) {
		if anchor == nil {
			body.Move(item, 0)
		} else {
			body.MoveAfter(item, anchor)
		}
		anchor = item
	}
	for _, name := range outputAttrFixOrder {
		if attr := body.FindAttribute(name); attr != nil {
			place(attr)
		}
	}
	for _, blockType := range outputNestedBlockOrder {
		for _, blk := range body.FindBlocksByType(blockType) {
			place(blk)
		}
	}
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

// Fix moves the terraform block to the first position in the file body. The
// move is item-aware: StandaloneComment items between resource blocks (the
// `### SNS Notifications`-style section headers) stay where they are in the
// items slice, so they survive intact with their flanking blank lines rather
// than being swept along with the moved block. Block bodies are untouched —
// only the top-level item order changes.
func (r *TerraformBlockFirstRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	terraformBlock := file.Body.FindBlock("terraform")
	if terraformBlock == nil {
		return nil, nil
	}
	file.Body.Move(terraformBlock, 0)

	return WholeFileEdit(originalContent, file.Bytes()), nil
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

// Fix reorders top-level blocks into the canonical priority order
// (terraform, provider, variable, locals, data, resource, module, output)
// via cst.Body.Move / MoveAfter. Blocks of unknown type stay in source
// order after the canonical prefix. The canonical prefix anchors at the
// first existing Block in body.Items, so file-header StandaloneComments
// (separated from the first block by a blank line) stay in their slot.
//
// Under DefaultTopLevelPolicy (StrictAdjacency=true), comments with a blank
// line above them are StandaloneComments and do NOT travel with the block
// they were adjacent to in source. This is the same mechanism that fixes
// the floating section-header bug in style.terraform-block-first.
func (r *ProviderBlockOrderRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
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

	if topLevelBlocksAreCanonical(file.Body) {
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
	for _, blockType := range topLevelCanonicalOrder {
		for _, blk := range file.Body.FindBlocksByType(blockType) {
			place(blk)
		}
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// topLevelBlocksAreCanonical reports whether the *cst.Block items in body
// already appear in topLevelCanonicalOrder, with any unknown-type blocks
// trailing the canonical prefix. BlankLine and StandaloneComment items are
// ignored. When true, the Fix would only redistribute BlankLines (since
// blocks are already canonically ordered) — return early to keep the input
// byte-identical.
func topLevelBlocksAreCanonical(body *cst.Body) bool {
	priority := make(map[string]int, len(topLevelCanonicalOrder))
	for i, t := range topLevelCanonicalOrder {
		priority[t] = i + 1
	}
	prevPrio := 0
	sawUnknown := false
	for _, item := range body.Items {
		blk, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		p, known := priority[blk.Type]
		if !known {
			sawUnknown = true
			continue
		}
		if sawUnknown || p < prevPrio {
			return false
		}
		prevPrio = p
	}
	return true
}

// topLevelCanonicalOrder is the canonical Fix-time order for top-level blocks.
// Blocks of unknown type retain their relative source order after the
// canonical prefix.
var topLevelCanonicalOrder = []string{
	"terraform", "provider", "variable", "locals", "data", "resource", "module", "output",
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

// Fix inserts a blank-line item before any attribute whose group differs from
// the preceding attribute (or before any second block-valued attribute when
// both neighbors are block-valued) via cst.Body.Insert. Insert-only: blank
// lines are never removed, matching the pre-CST semantics. Other body items
// stay at their source positions.
//
// Resource, module, data, variable, and output blocks are processed; other
// top-level block types pass through untouched. Pre-CST FormatAndCleanBlankLines
// wrapping is intentionally dropped — the CST mutation produces byte-exact
// output for unmodified regions.
func (r *AttributeGroupSpacingRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	originalContent, err := os.ReadFile(ctx.File)
	if err != nil {
		return nil, err
	}

	file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
	if parseErr != nil {
		return nil, nil //nolint:nilerr // parse error already surfaced by Check; Fix preserves no-op contract on partial trees
	}

	for _, item := range file.Body.Items {
		block, ok := item.(*cst.Block)
		if !ok {
			continue
		}
		if !isAttrGroupSpacingHostBlock(block.Type) {
			continue
		}
		insertAttributeGroupSpacing(block.Body, block.Type)
	}

	return WholeFileEdit(originalContent, file.Bytes()), nil
}

// isAttrGroupSpacingHostBlock reports whether the rule polices block bodies of
// blockType. Matches the Check predicate at the top-level Blocks iteration.
func isAttrGroupSpacingHostBlock(blockType string) bool {
	return blockType == "resource" || blockType == "module" || blockType == "data" ||
		blockType == "variable" || blockType == "output"
}

// insertAttributeGroupSpacing walks body.Items in source order, classifies each
// attribute via getAttrGroup, and inserts a *cst.BlankLine immediately before
// any attribute whose group differs from the preceding attribute's group (or
// before any second block-valued attribute when both neighbors are
// groupMainBlock). A blank line is inserted only when none already exists
// between the pair in body.Items.
//
// Iteration walks pairs from the last source-order attribute backward so each
// Insert lands at a position strictly greater than every earlier-snapshot
// attribute index — earlier-snapshot indices stay valid without recomputation,
// and the snapshotted attribute pointers keep resolving to the same items they
// did at the start of the function.
func insertAttributeGroupSpacing(body *cst.Body, blockType string) {
	if body == nil {
		return
	}
	type attrEntry struct {
		attr  *cst.Attribute
		idx   int
		group attrGroup
	}
	var attrs []attrEntry
	for i, item := range body.Items {
		a, ok := item.(*cst.Attribute)
		if !ok {
			continue
		}
		isMultiLine := bytes.IndexByte(a.ExpressionBytes, '\n') >= 0
		attrs = append(attrs, attrEntry{
			attr:  a,
			idx:   i,
			group: getAttrGroup(a.Name, blockType, isMultiLine),
		})
	}
	if len(attrs) < 2 {
		return
	}
	for i := len(attrs) - 1; i >= 1; i-- {
		earlier, later := attrs[i-1], attrs[i]
		needsBlankLine := earlier.group != later.group ||
			(earlier.group == groupMainBlock && later.group == groupMainBlock)
		if !needsBlankLine {
			continue
		}
		if hasBlankLineBetween(body.Items, earlier.idx, later.idx) {
			continue
		}
		body.Insert(&cst.BlankLine{Count: 1}, later.idx)
	}
}

// hasBlankLineBetween reports whether any *cst.BlankLine appears in
// items[earlierIdx+1:laterIdx]. Indices are guaranteed by construction to
// satisfy laterIdx > earlierIdx and to land in [0, len(items)) — they come
// from a forward scan of items, so no bounds check is needed.
func hasBlankLineBetween(items []cst.BodyItem, earlierIdx, laterIdx int) bool {
	for k := earlierIdx + 1; k < laterIdx; k++ {
		if _, ok := items[k].(*cst.BlankLine); ok {
			return true
		}
	}
	return false
}
