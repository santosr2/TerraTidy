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

// writeRegenerated emits a freshly-formatted block:
//
//	<LeadingComments...>
//	type "label" ... {[ openingBraceComment]<lineSep>
//	  <body items>
//	}[ closingBraceComment]<lineSep>
//
// Triggered when Block.raw is nil — either a Block constructed from scratch
// by an Insert caller, or an existing Block whose ancestor chain was
// invalidated by markDirty when a nested Body was mutated (see ops.go
// markDirty). Body items emit their own line terminators (raw bytes already
// include them, or regeneration appends lineSep), so no extra separator is
// needed between `{` and the first body item beyond the opening lineSep
// written here.
//
// Known limitation: when an ancestor Block.raw is nilled by markDirty, this
// path emits structurally correct but flush-left HCL — indentation lives in
// the nilled raw and has no separate storage yet. Splitting Block.raw into
// headerRaw + footerRaw with explicit indentation capture is the planned
// follow-up.
func (b *Block) writeRegenerated(buf *bytes.Buffer, lineSep []byte) {
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
	if b.Body != nil {
		b.Body.writeTo(buf, lineSep)
	}
	buf.WriteByte('}')
	if b.ClosingBraceComment != nil {
		buf.WriteByte(' ')
		buf.Write(b.ClosingBraceComment.Raw)
	}
	buf.Write(lineSep)
}
