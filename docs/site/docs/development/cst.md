# CST (Concrete Syntax Tree)

Reference for the `internal/cst` package, the structural tree TerraTidy's
style-engine rules use to implement auto-fixes that move blocks and attributes
without disturbing surrounding source.

Package: `github.com/santosr2/TerraTidy/internal/cst`

!!! note
    `internal/cst` is private to TerraTidy. It is not part of [`pkg/sdk`](sdk.md)
    and its API may change between minor releases. External rule authors should
    keep using the SDK types ([`hcl.File`](sdk.md#rule-interface),
    [`TextEdit`](sdk.md#textedit), [`FixResult`](sdk.md#fixresult)) and rely on
    built-in rules to do structural surgery on their behalf.

## Mental model

The CST is a **structural** tree over HCL files: it knows where blocks,
attributes, blank lines, and standalone comments live, and in what order. It
deliberately does **not** model expression syntax. The bytes between an
attribute's `=` and its terminating newline are held verbatim in
`Attribute.ExpressionBytes`. That keeps heredocs, interpolations, template
directives, multi-line function calls, and every other expression shape
byte-faithful with zero parsing effort.

This split is the point. The hclwrite-based helpers that preceded CST
preserved expression fidelity by reshuffling tokens; that mostly worked but
silently lost comments, mangled blank lines, and converted CRLF to LF on
fix. The CST preserves expression fidelity by **not touching expressions**,
and preserves surrounding trivia by holding each item's source bytes as a
fast path.

```text
HCL source ──Build──▶ CST (File + Body of items)
                         │
                         │  Move / MoveBefore / MoveAfter / Insert / Remove
                         │  Find* (Attribute / Block / BlocksByType / BlockByTypeAndLabels)
                         ▼
                      Mutated CST
                         │
                         │  File.Bytes()
                         ▼
HCL source (round-trip; unchanged items byte-identical)
```

## Node taxonomy

`BodyItem` is a **sealed sum type**: the four concrete implementations are
`*Attribute`, `*Block`, `*BlankLine`, and `*StandaloneComment`. Sealing is
enforced by an unexported `bodyItem()` method on the interface, so external
code cannot add a fifth variant. A `switch` over `BodyItem` should cover
all four (Go does not enforce exhaustiveness, so reviewers are the gate).

| Variant | What it represents | Round-trip mechanism |
|---|---|---|
| `*Attribute` | `name = expression` line, plus leading and inline comments | `raw` bytes when unmodified; regenerated from fields otherwise |
| `*Block` | `type "label" { body }` block, plus leading comments and brace-line comments | Header + nested `Body.writeTo` + footer; nested-body mutations flow through automatically. Exception: inline blocks (single-line `type { body }`) emit via `wholeRaw` and do **not** reflect nested-body mutations. No structural rule targets inline blocks today. |
| `*BlankLine` | Run of one or more consecutive empty source lines (Count ≥ 1) | `raw` when unmodified; emitted as `Count × lineSep` otherwise |
| `*StandaloneComment` | Comment run separated from its neighbors by blank lines on both sides; does **not** attach to any attribute or block | `raw` when unmodified; emitted as `comment + lineSep` per line otherwise |

`*StandaloneComment` is the variant that exists for `style.terraform-block-first`.
Floating section headers like `### SNS Notifications` that sit between
top-level blocks attach to neither neighbor; treating them as their own
body item is what makes "move the terraform block to position 0" not sweep
the header along with it.

Every variant exposes `RawBytes() []byte`. A non-nil return means "use these
exact bytes on serialization." A `nil` return means "the item was mutated;
regenerate from fields." `*Block` always returns `nil` because Block always
goes through the regeneration path (header + body + footer); any
nested-Body mutation is therefore visible at the file root by construction,
without a dirty-marking walk.

## Build policy

HCL gives no syntactic signal for "leading comment of X" when blank lines
sit between the comment and the next item. The CST exposes that heuristic
as a `Policy` toggle applied per `Body` at `Build` time:

```go
type Policy struct {
    StrictAdjacency bool
}
```

| Policy | Where to use | Behavior |
|---|---|---|
| `DefaultTopLevelPolicy()` (`StrictAdjacency: true`) | **Top-level** body items | A comment separated from the next item by ≥ 1 blank line becomes a `StandaloneComment` (survives reorders in place). Zero-blank-line comments still attach as `LeadingComments`. |
| `DefaultBlockBodyPolicy()` (`StrictAdjacency: false`) | **Nested** block bodies | Blank lines are tolerated; the comment still attaches as a leading comment of the next item. |

Build applies the policy argument to the top-level Body only; nested block
bodies always use `DefaultBlockBodyPolicy()`. That asymmetry matches user
intent: inside a block body, a comment above an attribute almost always
belongs to that attribute even with cosmetic blank lines between, while
at the top level a floating header comment between two blocks is the
section divider, not a label for whichever block happens to follow.

Rules that perform top-level structural fixes should call
`Build(content, filename, cst.DefaultTopLevelPolicy())`. Rules that
restructure inside a block body (none today) would build with the same
top-level policy and let the per-Body nested policy take care of itself.

## Worked example: writing a CST-based structural rule

`style.tags-at-end` (the post-migration `TagsAtEndRule.Fix`) is small enough
to read end-to-end as a template for new structural rules:

```go
func (r *TagsAtEndRule) Fix(ctx *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
    originalContent, err := os.ReadFile(ctx.File)
    if err != nil {
        return nil, err
    }

    file, parseErr := cst.Build(originalContent, ctx.File, cst.DefaultTopLevelPolicy())
    if parseErr != nil {
        // No-op on parse error: do not mutate a partial tree, and do not
        // surface the diagnostic as a Fix error (Check already produced it).
        return nil, nil //nolint:nilerr
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
```

The pattern every CST-based Fix follows:

1. **Read the file** through `os.ReadFile(ctx.File)`. The CST works on bytes;
   the `*hcl.File` argument is unused by Fix (Check already consumed it).
2. **Build with `DefaultTopLevelPolicy`** so floating top-level comments
   become `StandaloneComment` items.
3. **No-op on parse error** with `return nil, nil //nolint:nilerr`. Check
   has already surfaced the diagnostic; Fix must not mutate a partial tree
   and must not double-report.
4. **Walk `file.Body.Items`** and type-switch to `*cst.Block` / `*cst.Attribute`.
   Find targets via the `Find*` helpers (`FindBlock`, `FindAttribute`,
   `FindBlocksByType`, `FindBlockByTypeAndLabels`).
5. **Mutate via Body operations** (`Move`, `MoveBefore`, `MoveAfter`,
   `Insert`, `Remove`). Each operation returns `bool` (`false` when the
   target item is not in the body or the index is out of range); rules
   typically ignore the return because the `Find*` lookup just before it
   established that the item exists.
6. **Return a `WholeFileEdit`** comparing `originalContent` to `file.Bytes()`.
   `WholeFileEdit` returns `nil` when the two are byte-equal, so a Check
   finding that ends up needing no edit (e.g. the attribute was already
   canonical) collapses to a no-op without extra branching in the rule.

The engine collects every `*sdk.FixResult` across all findings for the file
and applies them in a single splice pass; see [`FixResult`](sdk.md#fixresult)
for the apply order and the whole-file exclusivity rule.

### When NOT to use the CST

The CST is for moves and inserts. It is not for token-level surgery inside
an expression; there are no `cst.Expression` nodes by design. Rules that
need to rewrite an expression (e.g. case conversion, regex replacement)
should keep using `hclwrite` or the byte-range `TextEdit` mechanism directly.

## Terragrunt awareness

`cst.IsTerragruntFile(f *File, path string) bool` is the heuristic for
Terragrunt-aware rules. It checks the filename first (basename
`terragrunt.hcl`, recognizing both `/` and `\` separators), then scans the
top-level body for any non-`terraform` block in `cst.TerragruntBlockTypes`:

```go
var TerragruntBlockTypes = map[string]bool{
    "include":      true,
    "dependency":   true,
    "dependencies": true,
    "remote_state": true,
    "generate":     true,
    "catalog":      true,
    "terraform":    true,
}
```

`terraform` is in the set because Terragrunt overloads the block name with
extra fields (`source`, `extra_arguments`, `before_hook`, `after_hook`). The
same block name is also legal in standard Terraform, so `IsTerragruntFile`
excludes a top-level `terraform` block from triggering detection by itself.
Otherwise every `.tf` file with a `terraform` block would falsely register.

A `nil` file or one with a `nil` Body is tolerated: detection falls back to
the filename check alone. That lets callers query path-based intent before
or independently of Build.

Scope: covers Terragrunt blocks stable through v0.50. The v0.67+ blocks
(`errors`, `exclude`, `feature`, `engine`) and the v0.66+
`terragrunt.stack.hcl` filename are out of scope until a consumer rule
needs them.

## Round-trip guarantee

`Build → File.Bytes` is **byte-identical** for any file that passes through
without mutation. Items unmodified since Build write through their `RawBytes()`
fast path; gap items (BlankLine, StandaloneComment) carry their own raw
bytes too; nothing regenerates.

After a `Move` / `Insert` / `Remove` that **reshuffles items without
mutating any item itself**, the round-trip is still byte-faithful for every
moved item. Each item keeps its own raw bytes, and reordering the slice
does not invalidate them.

### Caveat: regenerated regions are not byte-identical

Three operations cause an item to lose its raw fast path and regenerate
from fields on the next `File.Bytes()`:

- Mutating any field on an Attribute, Block, BlankLine, or StandaloneComment
  (today, no rule does this; structural fixes only reshuffle).
- Inserting a caller-constructed item (`raw` fields are nil on freshly
  built items).
- Building a `*File` literal manually instead of through `cst.Build`.

Regenerated output is **structurally** correct but may differ in trivia:
the line ending is whatever `f.LineSep()` detected (LF default), and
comment formatting reflects the field values, not the original source
positions. Mixed line endings in the input survive untouched as long as
no regeneration runs through them, but a regenerated region collapses to
the single detected `lineSep`.

`FuzzCSTRoundTrip` (in `internal/cst/cst_fuzz_test.go`) exercises the
structural round-trip on random valid HCL inputs (syntactic equality: the
same top-level attribute names and block type-plus-labels). The stricter
byte-identical property is enforced against curated fixtures in
`serialize_test.go`. `FuzzCSTMutateRoundTrip` in the same file extends
fuzz coverage to `Move`, `Insert`, and `Remove`. CI runs each fuzz target
for 30s per push; the scheduled run is 5 minutes.

## See also

- [Architecture](architecture.md) for where `internal/cst` fits in the engine
  pipeline.
- [SDK Reference](sdk.md) for the public Rule / Fixer interfaces, and the
  byte-range `TextEdit` and `FixResult` types that wrap CST-based fixes.
- [Style rules](../rules/style-rules.md) for the user-facing list of rules
  whose Fix paths are built on the CST.
