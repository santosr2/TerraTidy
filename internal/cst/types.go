// Package cst provides the CST node model for HCL files used by TerraTidy
// structural rule fixes. Nodes carry verbatim byte slices so unmodified items
// can round-trip byte-for-byte via [File.Bytes].
//
// Build (HCL → CST), [File.Bytes] (CST → HCL), Policy, the Body mutation
// operations (Move, MoveBefore, MoveAfter, Insert, Remove, and the four
// Find* lookups), Terragrunt awareness, and fuzz coverage are in place.
package cst

// BodyItem is the sealed sum type for items appearing in a Body —
// implementations: Attribute, Block, BlankLine, StandaloneComment. The
// unexported bodyItem method prevents external packages from extending the
// sum, so a switch over the concrete types stays exhaustive. Any code that
// switches on BodyItem must cover every implementation; Go does not enforce
// exhaustiveness, so reviewers are the gate.
//
// RawBytes returns the original source bytes for an unmodified item, used by
// Serialize for the fast path. Mutated items return nil so the serializer
// regenerates output from their fields. nil is the only sentinel for
// "regenerate"; implementations must not return an empty non-nil slice for
// the fast path.
type BodyItem interface {
	bodyItem()
	RawBytes() []byte
}

// File is the root of a parsed CST. Source owns the original bytes so that
// descendant nodes can hold verbatim slices into the same backing array.
//
// lineSep is the line ending Build detected in Source (`\n` or `\r\n`), used
// by Serialize when it has to emit a regenerated region. Unchanged items
// round-trip via their own raw bytes and ignore lineSep, so mixed line
// endings in the input survive untouched as long as no regeneration runs
// through them. Detection is first-occurrence-wins; defaults to `\n` for
// files with no newline at all. Manually constructed File values with a
// zero-value lineSep are treated as LF by Serialize.
type File struct {
	Source  []byte
	Body    *Body
	lineSep []byte
}

// LineSep returns the line ending Build detected. Exposed for tests and for
// future external consumers of the CST; production rules do not read it
// directly — Serialize uses it internally on the regeneration path.
func (f *File) LineSep() []byte { return f.lineSep }

// Body is an ordered sequence of items: attributes, blocks, blank lines, and
// standalone comments. Item order is preserved exactly as in the source.
// OpenByte and CloseByte are byte offsets into File.Source for the surrounding
// `{` and `}`; both are -1 for the file-root body.
type Body struct {
	Items     []BodyItem
	OpenByte  int
	CloseByte int
}

// CommentStyle identifies the syntax used for a Comment.
type CommentStyle int

// Comment styles supported by HCL.
const (
	// CommentHash is a `#`-prefixed line comment.
	CommentHash CommentStyle = iota
	// CommentSlash is a `//`-prefixed line comment.
	CommentSlash
	// CommentBlock is a `/* ... */` block comment.
	CommentBlock
)

// Comment is a single comment token. Raw holds the verbatim source bytes,
// including the leading comment marker.
type Comment struct {
	Style CommentStyle
	Text  string
	Raw   []byte
}

// Label is one of the label tokens following a block type — e.g.,
// `resource "aws_instance" "this"` has labels `aws_instance` and `this`.
// Raw holds the verbatim bytes including any surrounding quotes.
type Label struct {
	Text string
	Raw  []byte
}

// Attribute is a `name = expression` body item, with optional leading and
// inline comments. ExpressionBytes is held verbatim so heredocs,
// interpolations, and template directives round-trip exactly. EqualsByte is
// the byte offset of `=` within File.Source.
type Attribute struct {
	LeadingComments []Comment
	Name            string
	NameBytes       []byte
	EqualsByte      int
	ExpressionBytes []byte
	InlineComment   *Comment
	raw             []byte
}

// RawBytes returns the original source bytes for this attribute, or nil when
// the attribute has been mutated since Build.
func (a *Attribute) RawBytes() []byte { return a.raw }

func (*Attribute) bodyItem() {}

// Block is a block-type body item — resource, module, locals, etc. — with
// optional labels and a nested Body.
//
// For multi-line blocks (the common shape `type "label" {\n  body\n}\n`),
// headerRaw and footerRaw hold the verbatim source bytes around the body:
// headerRaw spans line-start of the leading content (or of the block-type
// identifier when there are no leading comments) through the opening brace,
// any inline opening-brace comment, and its line terminator; footerRaw spans
// line-start of the closing brace through the closing brace, any inline
// closing-brace comment, and its line terminator. Build populates both;
// writeRegenerated emits headerRaw + Body.writeTo + footerRaw so any
// nested-Body mutation is visible at the file root by construction.
//
// For inline blocks where opening and closing braces share a line
// (`type {body}`), wholeRaw holds the entire block's source bytes verbatim
// and writeRegenerated emits it as a single unit, skipping Body.writeTo
// and footerRaw. Body items are still parsed for Find* lookups; mutations
// on an inline block's Body do not propagate to the serialized output
// because the wholeRaw fast path takes precedence. No structural rule
// today targets inline blocks.
//
// Caller-constructed Blocks leave all three raw fields nil; writeRegenerated
// falls back to a canonical regenerated header / footer.
type Block struct {
	LeadingComments     []Comment
	Type                string
	TypeBytes           []byte
	Labels              []Label
	OpeningBraceComment *Comment
	Body                *Body
	ClosingBraceComment *Comment
	wholeRaw            []byte
	headerRaw           []byte
	footerRaw           []byte
}

// RawBytes always returns nil for *Block. Blocks are written via the
// regeneration path (headerRaw + Body.writeTo + footerRaw), so any mutation
// on a nested Body is visible at the file root by construction — no
// dirty-marking walk required, because there is no fast-path block.raw
// emission that could shadow it. Body items each keep their own raw fast
// path; only the surrounding block header/footer come from raw bytes here.
func (*Block) RawBytes() []byte { return nil }

func (*Block) bodyItem() {}

// BlankLine represents one or more consecutive empty source lines. Count is
// preserved so re-serialization keeps user-intended spacing exact and must be
// >= 1; a BlankLine with Count == 0 is malformed and Build will not produce
// one.
type BlankLine struct {
	Count int
	raw   []byte
}

// RawBytes returns the original source bytes for this run of blank lines, or
// nil when the run has been mutated since Build.
func (bl *BlankLine) RawBytes() []byte { return bl.raw }

func (*BlankLine) bodyItem() {}

// StandaloneComment is a comment (or run of consecutive comment lines) that
// is NOT attached to any attribute or block — separated from neighboring
// items by blank lines on both sides. Standalone comments survive reorder
// operations in place, which is the fix mechanism for the floating
// section-header bug in `style.terraform-block-first` (block reorders that
// swept along the preceding standalone header comment).
type StandaloneComment struct {
	Comments []Comment
	raw      []byte
}

// RawBytes returns the original source bytes for this comment run, or nil
// when the run has been mutated since Build.
func (sc *StandaloneComment) RawBytes() []byte { return sc.raw }

func (*StandaloneComment) bodyItem() {}
