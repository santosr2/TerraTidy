package cst

import (
	"bytes"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Build constructs a CST from HCL source bytes. The returned File holds
// content verbatim so unchanged body items round-trip byte-for-byte through
// File.Bytes.
//
// On parse error, Build returns the partial CST it managed to build along
// with the first error from hclsyntax. The partial tree is safe to traverse,
// but rule Fix migrations preserve the no-op-on-parse-error pattern: if
// err != nil, do not attempt structural mutation.
//
// policy controls how comments separated from the next body item by blank
// lines are attached for the TOP-LEVEL Body. Nested block bodies always use
// DefaultBlockBodyPolicy() — the policy argument is for the top-level only.
func Build(content []byte, filename string, policy Policy) (*File, error) {
	tokens, _ := hclsyntax.LexConfig(content, filename, hcl.InitialPos)

	syntaxFile, diags := hclsyntax.ParseConfig(content, filename, hcl.InitialPos)
	var parseErr error
	if diags.HasErrors() {
		parseErr = diags
	}

	f := &File{
		Source:  content,
		lineSep: detectLineSep(content),
	}

	if syntaxFile == nil || syntaxFile.Body == nil {
		f.Body = newEmptyBody()
		return f, parseErr
	}

	syntaxBody, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		f.Body = newEmptyBody()
		return f, parseErr
	}

	f.Body = buildBody(content, tokens, syntaxBody, policy, 0, len(content), -1, -1)
	return f, parseErr
}

func newEmptyBody() *Body { return &Body{OpenByte: -1, CloseByte: -1} }

// detectLineSep returns the line ending used by content. CRLF is selected
// when the first newline is preceded by a CR; LF otherwise. Defaults to LF
// for files with no newline at all.
//
// Old-Mac CR-only line endings (no LF following) are not handled — HCL
// tooling in the wild does not produce them, and hclsyntax treats them as
// regular bytes inside expressions. commentsInRange and findInlineComment
// share this assumption when they probe for a trailing newline after a
// block comment.
func detectLineSep(content []byte) []byte {
	for i, b := range content {
		if b != '\n' {
			continue
		}
		if i > 0 && content[i-1] == '\r' {
			return []byte("\r\n")
		}
		return []byte("\n")
	}
	return []byte("\n")
}

// structuralItem normalizes an attr-or-block for source-order iteration.
type structuralItem struct {
	startByte, endByte int
	startLine, endLine int
	attr               *hclsyntax.Attribute
	block              *hclsyntax.Block
}

// buildBody constructs a CST Body from a hclsyntax body, walking items in
// source order and filling gaps with BlankLines and StandaloneComments per
// policy.
//
// bodyStartByte/bodyEndByte bound the gap-scan region — for a block body,
// that is content between the braces; for the top-level body it is the full
// content. openByte/closeByte are the brace offsets stored on Body for
// downstream consumers; both are -1 at the top level.
func buildBody(
	content []byte,
	tokens hclsyntax.Tokens,
	syntaxBody *hclsyntax.Body,
	policy Policy,
	bodyStartByte, bodyEndByte int,
	openByte, closeByte int,
) *Body {
	body := &Body{OpenByte: openByte, CloseByte: closeByte}

	items := collectStructural(syntaxBody)

	cursor := bodyStartByte
	// For block bodies, the first newline after `{` is the terminator of
	// the opening-brace line, not a blank line. Step past it so subsequent
	// newline counts measure real blank lines. The top-level body has no
	// such opening to skip.
	if openByte != -1 {
		cursor = consumeLineTerminator(content, cursor, bodyEndByte)
	}
	for i := range items {
		item := &items[i]

		gapItems, leading, leadingStart := classifyGap(
			content, tokens, cursor, item.startByte, policy, true,
		)
		body.Items = append(body.Items, gapItems...)

		if item.attr != nil {
			cstAttr, newCursor := buildAttribute(
				content, tokens, item.attr, leading, leadingStart, bodyEndByte,
			)
			body.Items = append(body.Items, cstAttr)
			cursor = newCursor
		} else {
			cstBlock, newCursor := buildBlock(
				content, tokens, item.block, leading, leadingStart, bodyEndByte,
			)
			cstBlock.parentBody = body
			body.Items = append(body.Items, cstBlock)
			cursor = newCursor
		}
	}

	if cursor < bodyEndByte {
		gapItems, _, _ := classifyGap(
			content, tokens, cursor, bodyEndByte, policy, false,
		)
		body.Items = append(body.Items, gapItems...)
	}

	return body
}

// collectStructural returns the body's attrs and blocks merged into one
// slice and sorted by source-order start byte. body.Attributes is an
// unordered map; this is the same trick GetOrderedAttrNames uses in
// internal/engines/style/rules/helpers.go.
func collectStructural(b *hclsyntax.Body) []structuralItem {
	items := make([]structuralItem, 0, len(b.Attributes)+len(b.Blocks))
	for _, a := range b.Attributes {
		items = append(items, structuralItem{
			startByte: a.Range().Start.Byte,
			endByte:   a.Range().End.Byte,
			startLine: a.Range().Start.Line,
			endLine:   a.Range().End.Line,
			attr:      a,
		})
	}
	for _, blk := range b.Blocks {
		items = append(items, structuralItem{
			startByte: blk.Range().Start.Byte,
			endByte:   blk.Range().End.Byte,
			startLine: blk.Range().Start.Line,
			endLine:   blk.Range().End.Line,
			block:     blk,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].startByte < items[j].startByte
	})
	return items
}

// chunkKind categorizes a contiguous run inside a gap.
type chunkKind int

const (
	chunkBlank chunkKind = iota
	chunkComments
)

// gapChunk is one segment of the gap between two body items (or between a
// body boundary and a body item). It is either a run of blank lines or a
// run of consecutive comment lines.
type gapChunk struct {
	kind     chunkKind
	startBy  int       // first source byte of the chunk
	endBy    int       // one past the last source byte
	blanks   int       // for chunkBlank: number of blank lines
	comments []Comment // for chunkComments: the comments in source order
}

// classifyGap walks the bytes in [start, end) and partitions them into a
// sequence of chunks (blank-line runs and comment-line runs). Then it
// decides per policy which trailing comments attach as LeadingComments to
// the structural item that follows the gap.
//
// hasNextItem is true when a structural item follows this gap (i.e. inside
// a body during iteration); false at end-of-body, where every comment is
// standalone since there is nothing to attach to.
//
// Returns the gap items that go into the body (BlankLine + StandaloneComment
// in source order), the leading comments to attach to the next structural
// item, and the source-byte offset where those leading comments start (so
// the next item's `raw` includes them verbatim). leadingStart == -1 means
// "no leading comments attached".
func classifyGap(
	content []byte,
	tokens hclsyntax.Tokens,
	start, end int,
	policy Policy,
	hasNextItem bool,
) (items []BodyItem, leading []Comment, leadingStart int) {
	leadingStart = -1
	if start >= end {
		return nil, nil, -1
	}

	chunks := chunkGap(content, tokens, start, end)
	if len(chunks) == 0 {
		return nil, nil, -1
	}

	lastIdx := len(chunks) - 1
	last := chunks[lastIdx]

	// Decide whether the last comment-run attaches as leading on the next
	// item. The last chunk is necessarily comment-adjacent to the next item
	// (chunkGap guarantees alternating blank/comment runs, so if the last
	// chunk is chunkComments there is no trailing blank between it and the
	// gap end). The remaining question is whether there is a blank chunk
	// ABOVE this comment-run:
	//   - no blank above → always attach.
	//   - blank above → policy decides (strict standalone, passthrough attach).
	attach := false
	if hasNextItem && last.kind == chunkComments {
		hasBlankAbove := lastIdx > 0 && chunks[lastIdx-1].kind == chunkBlank
		if hasBlankAbove {
			attach = !policy.StrictAdjacency
		} else {
			attach = true
		}
	}

	for i, c := range chunks {
		if i == lastIdx && attach {
			leading = append([]Comment(nil), c.comments...)
			leadingStart = c.startBy
			break
		}
		switch c.kind {
		case chunkBlank:
			items = append(items, &BlankLine{
				Count: c.blanks,
				raw:   bytes.Clone(content[c.startBy:c.endBy]),
			})
		case chunkComments:
			items = append(items, &StandaloneComment{
				Comments: append([]Comment(nil), c.comments...),
				raw:      bytes.Clone(content[c.startBy:c.endBy]),
			})
		}
	}

	return items, leading, leadingStart
}

// chunkGap partitions content[start:end] into alternating blank-line runs
// and comment-line runs. A "blank line" is a fully blank source line that
// is NOT part of the line-terminator of a preceding structural item.
//
// The trick: the gap's first line may be the tail of a previous item's
// line (e.g., after `x = 1` ends mid-line at byte 5, the rest of that line
// `\n` is in the gap but is a TERMINATOR, not a blank). chunkGap detects
// this by checking whether the first newline in the gap closes a partial
// line.
func chunkGap(content []byte, tokens hclsyntax.Tokens, start, end int) []gapChunk {
	comments := commentsInRange(content, tokens, start, end)

	if len(comments) == 0 {
		return chunkAllBlanks(content, start, end)
	}

	var chunks []gapChunk
	cursor := start

	for i := 0; i < len(comments); {
		c := comments[i]
		commentStart := c.tokenStart
		if commentStart > cursor {
			b := countBlankLines(content, cursor, commentStart)
			if b > 0 {
				chunks = append(chunks, gapChunk{
					kind:    chunkBlank,
					startBy: cursor,
					endBy:   commentStart,
					blanks:  b,
				})
			}
		}

		runStart := commentStart
		runEnd := c.tokenEnd
		run := []Comment{c.comment}
		j := i + 1
		for j < len(comments) {
			next := comments[j]
			if countBlankLines(content, runEnd, next.tokenStart) > 0 {
				break
			}
			run = append(run, next.comment)
			runEnd = next.tokenEnd
			j++
		}
		chunks = append(chunks, gapChunk{
			kind:     chunkComments,
			startBy:  runStart,
			endBy:    runEnd,
			comments: run,
		})
		cursor = runEnd
		i = j
	}

	// Tail blank lines between last comment and gap end.
	if cursor < end {
		b := countBlankLines(content, cursor, end)
		if b > 0 {
			chunks = append(chunks, gapChunk{
				kind:    chunkBlank,
				startBy: cursor,
				endBy:   end,
				blanks:  b,
			})
		}
	}

	return chunks
}

// chunkAllBlanks returns a single blank chunk for a gap that contains no
// comments, or nothing when the gap has no blank lines (only a line
// terminator).
func chunkAllBlanks(content []byte, start, end int) []gapChunk {
	b := countBlankLines(content, start, end)
	if b == 0 {
		return nil
	}
	return []gapChunk{{
		kind:    chunkBlank,
		startBy: start,
		endBy:   end,
		blanks:  b,
	}}
}

// countBlankLines returns the number of newlines in content[start:end). The
// caller is responsible for advancing past any prior line terminator before
// calling — buildBody and buildAttribute do this via consumeLineTerminator.
// This invariant means every newline in the input range opens a new blank
// line, so the count is the answer directly.
func countBlankLines(content []byte, start, end int) int {
	newlines := 0
	for i := start; i < end; i++ {
		if content[i] == '\n' {
			newlines++
		}
	}
	return newlines
}

// commentInfo pairs a Comment with its source byte range.
type commentInfo struct {
	comment    Comment
	tokenStart int // inclusive — first byte of the comment marker
	tokenEnd   int // one past the last byte the gap should consume for this comment
}

// commentsInRange returns the comments whose source range falls fully
// inside [start, end), in source order. Token ranges that do not already
// include their trailing newline (block `/* */` comments) are extended
// past it so the chunk owns it and downstream blank-line counting does
// not double-count.
//
// Verified against hclsyntax v2 (github.com/hashicorp/hcl/v2 ≥ 2.0):
// TokenComment bytes for `#` and `//` include the trailing line
// terminator (LF or CRLF); block comments do not. extendPastLineTerminator
// relies on this asymmetry. If the upstream behavior flips, the swallow
// bug returns and a BlankLine count decrements by one per affected gap.
func commentsInRange(content []byte, tokens hclsyntax.Tokens, start, end int) []commentInfo {
	var out []commentInfo
	for _, tok := range tokens {
		if tok.Type != hclsyntax.TokenComment {
			continue
		}
		// Token must fit entirely inside the gap window; bounds are
		// inclusive on both ends because callers pass the full slice
		// they intend to own.
		if tok.Range.Start.Byte < start || tok.Range.End.Byte > end {
			continue
		}
		ts := tok.Range.Start.Byte
		te := extendPastLineTerminator(content, tok.Range.End.Byte, end)
		out = append(out, commentInfo{
			comment:    tokenToComment(tok),
			tokenStart: ts,
			tokenEnd:   te,
		})
	}
	return out
}

// extendPastLineTerminator advances tokenEnd past a trailing LF or CRLF
// when the token does not already include it. `#` and `//` line comments
// emitted by hclsyntax already include their terminator; extending again
// would advance tokenEnd past the next blank line's `\n`, silently
// decrementing the following BlankLine.Count by one and hiding the gap
// from the CST. `/* */` block comments stop short of any following
// newline and need the extension so chunk byte ranges abut cleanly.
//
// limit is the body bound — the function never advances past it.
//
// Precondition: tokenEnd >= 1. Every TokenComment hclsyntax produces has
// at least one byte (the comment marker), so the in-tree call sites
// satisfy this; the function would index out of range on tokenEnd == 0.
func extendPastLineTerminator(content []byte, tokenEnd, limit int) int {
	if tokenEnd > len(content) || content[tokenEnd-1] == '\n' {
		return tokenEnd
	}
	if tokenEnd < limit && tokenEnd < len(content) && content[tokenEnd] == '\n' {
		return tokenEnd + 1
	}
	if tokenEnd+1 < limit && tokenEnd+1 < len(content) && content[tokenEnd] == '\r' && content[tokenEnd+1] == '\n' {
		return tokenEnd + 2
	}
	return tokenEnd
}

// tokenToComment converts a TokenComment into a Comment, classifying the
// style from the leading marker.
func tokenToComment(tok hclsyntax.Token) Comment {
	raw := bytes.Clone(tok.Bytes)
	style := CommentHash
	text := ""
	switch {
	case bytes.HasPrefix(raw, []byte("//")):
		style = CommentSlash
		text = strings.TrimRight(string(raw[2:]), "\r\n")
	case bytes.HasPrefix(raw, []byte("/*")):
		style = CommentBlock
		t := raw
		t = bytes.TrimPrefix(t, []byte("/*"))
		t = bytes.TrimSuffix(t, []byte("*/"))
		text = string(t)
	case bytes.HasPrefix(raw, []byte("#")):
		style = CommentHash
		text = strings.TrimRight(string(raw[1:]), "\r\n")
	}
	return Comment{Style: style, Text: text, Raw: raw}
}

// buildAttribute constructs a CST Attribute from a hclsyntax Attribute,
// attaching any leading comments and detecting a same-line inline trailing
// comment.
//
// Returns the attribute and the byte cursor advanced past it. The cursor
// includes the line terminator (and any inline comment + its newline) so
// downstream gap processing starts on a clean line boundary.
func buildAttribute(
	content []byte,
	tokens hclsyntax.Tokens,
	a *hclsyntax.Attribute,
	leading []Comment,
	leadingStart int,
	bodyEndByte int,
) (*Attribute, int) {
	attrStart := a.SrcRange.Start.Byte
	attrEnd := a.SrcRange.End.Byte

	rawStart := attrStart
	if leadingStart >= 0 {
		rawStart = leadingStart
	}

	inline, afterInline := findInlineComment(content, tokens, attrEnd, a.SrcRange.End.Line, bodyEndByte)
	rawEnd := afterInline
	if inline == nil {
		rawEnd = consumeLineTerminator(content, attrEnd, bodyEndByte)
	}

	cstAttr := &Attribute{
		LeadingComments: leading,
		Name:            a.Name,
		NameBytes:       bytes.Clone(content[a.NameRange.Start.Byte:a.NameRange.End.Byte]),
		EqualsByte:      a.EqualsRange.Start.Byte,
		ExpressionBytes: bytes.Clone(content[a.Expr.Range().Start.Byte:a.Expr.Range().End.Byte]),
		InlineComment:   inline,
		raw:             bytes.Clone(content[rawStart:rawEnd]),
	}
	return cstAttr, rawEnd
}

// findInlineComment looks for a TokenComment that starts on the same source
// line as endLine and is the first non-whitespace token after fromByte.
// Returns the comment (or nil) and the byte position just past it.
func findInlineComment(
	content []byte,
	tokens hclsyntax.Tokens,
	fromByte, endLine int,
	bodyEndByte int,
) (*Comment, int) {
	for _, tok := range tokens {
		if tok.Range.Start.Byte < fromByte {
			continue
		}
		if tok.Range.Start.Byte >= bodyEndByte {
			return nil, fromByte
		}
		if tok.Type == hclsyntax.TokenNewline {
			return nil, fromByte
		}
		if tok.Type != hclsyntax.TokenComment {
			return nil, fromByte
		}
		if tok.Range.Start.Line != endLine {
			return nil, fromByte
		}
		c := tokenToComment(tok)
		te := extendPastLineTerminator(content, tok.Range.End.Byte, bodyEndByte)
		return &c, te
	}
	return nil, fromByte
}

// consumeLineTerminator returns the byte offset just past the next `\n`
// (or `\r\n`) at or after fromByte, bounded by bodyEndByte. If no newline
// is found before the bound, returns fromByte unchanged.
func consumeLineTerminator(content []byte, fromByte, bodyEndByte int) int {
	limit := bodyEndByte
	if limit > len(content) {
		limit = len(content)
	}
	for i := fromByte; i < limit; i++ {
		if content[i] == '\n' {
			return i + 1
		}
	}
	return fromByte
}

// buildBlock constructs a CST Block, recursing into the block body with
// DefaultBlockBodyPolicy() (nested-block default).
//
// Returns the block and the byte cursor advanced past the closing brace
// and its line terminator.
func buildBlock(
	content []byte,
	tokens hclsyntax.Tokens,
	b *hclsyntax.Block,
	leading []Comment,
	leadingStart int,
	bodyEndByte int,
) (*Block, int) {
	rawStart := b.Range().Start.Byte
	if leadingStart >= 0 {
		rawStart = leadingStart
	}
	closeEnd := b.CloseBraceRange.End.Byte
	rawEnd := consumeLineTerminator(content, closeEnd, bodyEndByte)

	labels := make([]Label, 0, len(b.Labels))
	for i, l := range b.Labels {
		lr := b.LabelRanges[i]
		labels = append(labels, Label{
			Text: l,
			Raw:  bytes.Clone(content[lr.Start.Byte:lr.End.Byte]),
		})
	}

	innerStart := b.OpenBraceRange.End.Byte
	innerEnd := b.CloseBraceRange.Start.Byte

	// Opening-brace inline comment (e.g. `resource "x" "y" { # why`). Scoped
	// inside the body so it doesn't bleed past `}`. findInlineComment stops
	// at the first non-comment / non-newline token, so it naturally returns
	// nil if `{` is followed by a newline (the common multi-line case).
	openComment, _ := findInlineComment(
		content, tokens, innerStart, b.OpenBraceRange.End.Line, innerEnd,
	)
	// Closing-brace inline comment (e.g. `} # end of resource`). Bound by
	// the post-block cursor — same bound buildBlock uses for rawEnd.
	closeComment, _ := findInlineComment(
		content, tokens, b.CloseBraceRange.End.Byte, b.CloseBraceRange.End.Line, bodyEndByte,
	)

	nestedBody := buildBody(
		content, tokens, b.Body, DefaultBlockBodyPolicy(),
		innerStart, innerEnd,
		b.OpenBraceRange.Start.Byte, b.CloseBraceRange.Start.Byte,
	)

	cstBlock := &Block{
		LeadingComments:     leading,
		Type:                b.Type,
		TypeBytes:           bytes.Clone(content[b.TypeRange.Start.Byte:b.TypeRange.End.Byte]),
		Labels:              labels,
		OpeningBraceComment: openComment,
		Body:                nestedBody,
		ClosingBraceComment: closeComment,
		raw:                 bytes.Clone(content[rawStart:rawEnd]),
	}
	nestedBody.parentBlock = cstBlock
	return cstBlock, rawEnd
}
