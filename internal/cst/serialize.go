package cst

import (
	"bytes"
	"fmt"
)

// Bytes serializes the CST back to source bytes.
//
// Items unmodified since Build (RawBytes() != nil) write through verbatim;
// mutated items regenerate from their fields. Items are joined in their
// current Body.Items order, so a Move / Insert / Remove that reshuffles items
// without mutating any item itself round-trips byte-faithfully — each moved
// item keeps its own raw bytes and the surrounding gap items (BlankLine,
// StandaloneComment) carry their own raw too.
//
// On an empty File (Body == nil), Bytes returns an empty, non-nil slice.
// Files produced by Build always have Body set, including for empty input.
func (f *File) Bytes() []byte {
	var buf bytes.Buffer
	if f.Body != nil {
		f.Body.writeTo(&buf, lineSepOr(f.lineSep))
	}
	if out := buf.Bytes(); out != nil {
		return out
	}
	return []byte{}
}

// lineSepOr returns ls when non-empty, otherwise the LF default. Defensive
// fallback for File values not produced by Build — manually constructed
// File{} literals get a usable line ending for BlankLine and regenerated-item
// terminators without the caller having to remember to set lineSep.
func lineSepOr(ls []byte) []byte {
	if len(ls) == 0 {
		return []byte("\n")
	}
	return ls
}

// writeTo appends Body.Items to buf in their current order. lineSep is the
// terminator emitted by items that regenerate (blank-line runs, freshly
// constructed Attributes/Blocks, regenerated StandaloneComment runs).
//
// Unchanged items ignore lineSep: their raw bytes already encode whatever
// line endings they were built from, so mixed-ending source survives the
// round-trip even though lineSep itself is a single shape.
func (b *Body) writeTo(buf *bytes.Buffer, lineSep []byte) {
	for _, item := range b.Items {
		if rb := item.RawBytes(); rb != nil {
			buf.Write(rb)
			continue
		}
		// BodyItem is sealed (see types.go) — these four cases are exhaustive.
		switch v := item.(type) {
		case *Attribute:
			v.writeRegenerated(buf, lineSep)
		case *Block:
			v.writeRegenerated(buf, lineSep)
		case *BlankLine:
			for i := 0; i < v.Count; i++ {
				buf.Write(lineSep)
			}
		case *StandaloneComment:
			for _, c := range v.Comments {
				buf.Write(c.Raw)
				buf.Write(lineSep)
			}
		default:
			// BodyItem is sealed (unexported bodyItem method); Go can't enforce
			// exhaustiveness, so this guards against a fifth variant landing
			// upstream without a matching case here — silent fall-through
			// would drop the item, which is a worse failure mode than a panic.
			panic(fmt.Sprintf("cst: unhandled BodyItem type %T", item))
		}
	}
}

// writeRegenerated emits a freshly-formatted attribute line:
//
//	<LeadingComments...>name = expression[ inlineComment]<lineSep>
//
// Used only when an Attribute has been mutated since Build (raw == nil).
// Existing structural rules reshuffle Body items rather than mutating
// individual Attribute fields, so production rules don't drive this path —
// it exists for newly-Inserted attributes and for future rules that rename
// or rewrite expressions. Formatting is canonical (`name = expression`);
// original alignment whitespace is not preserved because there is no signal
// to reconstruct it from.
func (a *Attribute) writeRegenerated(buf *bytes.Buffer, lineSep []byte) {
	for _, c := range a.LeadingComments {
		buf.Write(c.Raw)
		buf.Write(lineSep)
	}
	buf.WriteString(a.Name)
	buf.WriteString(" = ")
	buf.Write(a.ExpressionBytes)
	if a.InlineComment != nil {
		buf.WriteByte(' ')
		buf.Write(a.InlineComment.Raw)
	}
	buf.Write(lineSep)
}

// writeRegenerated emits a block by combining its captured header / footer
// raw bytes (or a canonical regenerated header / footer for caller-built
// blocks) with the body items written via Body.writeTo:
//
//	<headerRaw or regenerated header>
//	<body items>
//	<footerRaw or regenerated footer>
//
// Block.RawBytes() returns nil unconditionally, so this is the only path
// Serialize uses for blocks. Body items emit their own line terminators,
// so no extra separator is needed between the header and the first body
// item beyond what headerRaw or the regenerated lineSep already carries.
//
// Inline blocks (opening and closing brace on the same source line) take
// a fast path: their entire source bytes live in wholeRaw and are emitted
// as a single unit, with Body.writeTo and the footer path skipped. Body
// mutations are not reflected for inline blocks; the wholeRaw fast path
// wins. No structural rule currently targets inline blocks.
//
// When headerRaw is nil (a Block constructed from scratch by an Insert
// caller, or by a test), the regenerated header has the canonical shape
// `<leading comments...>\ntype "label" ... { <opening-brace comment>\n`
// with no leading indentation — there is no signal to reconstruct
// indentation from. Same shape applies to footerRaw == nil.
func (b *Block) writeRegenerated(buf *bytes.Buffer, lineSep []byte) {
	if b.wholeRaw != nil {
		buf.Write(b.wholeRaw)
		return
	}
	switch {
	case b.headerRaw != nil:
		buf.Write(b.headerRaw)
	default:
		for _, c := range b.LeadingComments {
			buf.Write(c.Raw)
			buf.Write(lineSep)
		}
		buf.WriteString(b.Type)
		for _, l := range b.Labels {
			buf.WriteByte(' ')
			buf.Write(l.Raw)
		}
		buf.WriteString(" {")
		if b.OpeningBraceComment != nil {
			buf.WriteByte(' ')
			buf.Write(b.OpeningBraceComment.Raw)
		}
		buf.Write(lineSep)
	}
	if b.Body != nil {
		b.Body.writeTo(buf, lineSep)
	}
	switch {
	case b.footerRaw != nil:
		buf.Write(b.footerRaw)
	default:
		buf.WriteByte('}')
		if b.ClosingBraceComment != nil {
			buf.WriteByte(' ')
			buf.Write(b.ClosingBraceComment.Raw)
		}
		buf.Write(lineSep)
	}
}
