package cst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuild_Empty pins the empty-content case: Build returns a non-nil File
// with an empty Body, sentinel brace offsets, and the default LF line
// separator.
func TestBuild_Empty(t *testing.T) {
	t.Parallel()

	f, err := Build([]byte(""), "empty.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.NotNil(t, f)
	require.NotNil(t, f.Body)

	assert.Empty(t, f.Body.Items)
	assert.Equal(t, -1, f.Body.OpenByte)
	assert.Equal(t, -1, f.Body.CloseByte)
	assert.Equal(t, []byte("\n"), f.LineSep())
	assert.Equal(t, []byte(""), f.Source)
}

// TestBuild_OnlyComments covers files whose only body items are comments.
// Adjacent comment lines collapse into one StandaloneComment with multiple
// Comments; comments separated by a blank line produce distinct
// StandaloneComment items with a BlankLine between them.
//
// wantStyles is flattened across all StandaloneComment items in order. It
// pins per-comment Style so a regression that flips `#` to `//` or
// misclassifies `/* */` fails loudly.
func TestBuild_OnlyComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantTypes  []string
		wantStyles []CommentStyle
	}{
		{
			name:       "single comment",
			content:    "# header\n",
			wantTypes:  []string{"StandaloneComment"},
			wantStyles: []CommentStyle{CommentHash},
		},
		{
			name:       "two adjacent comments form one run",
			content:    "# first\n# second\n",
			wantTypes:  []string{"StandaloneComment"},
			wantStyles: []CommentStyle{CommentHash, CommentHash},
		},
		{
			name:       "two comments split by blank line",
			content:    "# first\n\n# second\n",
			wantTypes:  []string{"StandaloneComment", "BlankLine", "StandaloneComment"},
			wantStyles: []CommentStyle{CommentHash, CommentHash},
		},
		{
			name:       "slash-style comments",
			content:    "// first\n// second\n",
			wantTypes:  []string{"StandaloneComment"},
			wantStyles: []CommentStyle{CommentSlash, CommentSlash},
		},
		{
			// hclsyntax stops a block comment at `*/`, not past the newline.
			// extendPastLineTerminator pulls the newline in so the next
			// blank line is detected as its own BlankLine. Regression guard:
			// without the fix in commentsInRange, the trailing newline would
			// be missed and chunk boundaries would not abut.
			name:       "block comment followed by blank then comment",
			content:    "/* one */\n\n# two\n",
			wantTypes:  []string{"StandaloneComment", "BlankLine", "StandaloneComment"},
			wantStyles: []CommentStyle{CommentBlock, CommentHash},
		},
		{
			// CRLF arm of extendPastLineTerminator. Same shape as above,
			// `\r\n` instead of `\n`. Pins the block-comment-with-CRLF
			// branch so a future regression there fails loudly.
			name:       "block comment followed by blank then comment (CRLF)",
			content:    "/* one */\r\n\r\n# two\r\n",
			wantTypes:  []string{"StandaloneComment", "BlankLine", "StandaloneComment"},
			wantStyles: []CommentStyle{CommentBlock, CommentHash},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte(tc.content), "x.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)

			require.Equal(t, len(tc.wantTypes), len(f.Body.Items),
				"item count mismatch: got %d, want %d", len(f.Body.Items), len(tc.wantTypes))

			var gotStyles []CommentStyle
			for i, item := range f.Body.Items {
				assert.Equal(t, tc.wantTypes[i], itemKind(item),
					"item %d kind mismatch", i)
				if sc, ok := item.(*StandaloneComment); ok {
					for _, c := range sc.Comments {
						gotStyles = append(gotStyles, c.Style)
					}
				}
			}
			assert.Equal(t, tc.wantStyles, gotStyles, "flattened comment styles")
		})
	}
}

// TestBuild_BlockCommentText pins the `/* */` trim behavior in
// tokenToComment. The text omits the markers but preserves the inner
// bytes verbatim, while Raw keeps the full token including markers.
func TestBuild_BlockCommentText(t *testing.T) {
	t.Parallel()

	f, err := Build([]byte("/* one */\n"), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.Len(t, f.Body.Items, 1)

	sc, ok := f.Body.Items[0].(*StandaloneComment)
	require.True(t, ok)
	require.Len(t, sc.Comments, 1)

	c := sc.Comments[0]
	assert.Equal(t, CommentBlock, c.Style)
	assert.Equal(t, " one ", c.Text, "block-comment text strips /* and */, preserves spacing")
	assert.Equal(t, []byte("/* one */"), c.Raw)
}

// TestBuild_OnlyBlankLines verifies that contiguous blank lines collapse
// into a single BlankLine with Count == number of newlines.
func TestBuild_OnlyBlankLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantCount int
	}{
		{"one newline", "\n", 1},
		{"two newlines", "\n\n", 2},
		{"three newlines", "\n\n\n", 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte(tc.content), "blanks.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)
			require.Len(t, f.Body.Items, 1)

			bl, ok := f.Body.Items[0].(*BlankLine)
			require.True(t, ok, "expected *BlankLine, got %T", f.Body.Items[0])
			assert.Equal(t, tc.wantCount, bl.Count)
			assert.Equal(t, []byte(tc.content), bl.RawBytes())
		})
	}
}

// TestBuild_SingleAttribute covers a single attribute at the top level,
// with and without a directly-adjacent leading comment.
func TestBuild_SingleAttribute(t *testing.T) {
	t.Parallel()

	t.Run("bare attribute", func(t *testing.T) {
		t.Parallel()
		content := []byte("name = \"value\"\n")
		f, err := Build(content, "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		attr, ok := f.Body.Items[0].(*Attribute)
		require.True(t, ok, "expected *Attribute, got %T", f.Body.Items[0])
		assert.Equal(t, "name", attr.Name)
		assert.Empty(t, attr.LeadingComments)
		assert.Nil(t, attr.InlineComment)
		assert.Equal(t, content, attr.RawBytes())
		// EqualsByte must point at `=` inside Source. Structural rules
		// read it to splice byte-range edits; a regression that zeros it
		// silently corrupts Fix output.
		require.Less(t, attr.EqualsByte, len(content))
		assert.Equal(t, byte('='), content[attr.EqualsByte])
	})

	t.Run("attribute with adjacent leading comment", func(t *testing.T) {
		t.Parallel()
		content := "# leading\nname = \"value\"\n"
		f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		attr, ok := f.Body.Items[0].(*Attribute)
		require.True(t, ok)
		require.Len(t, attr.LeadingComments, 1)
		assert.Equal(t, CommentHash, attr.LeadingComments[0].Style)
		// Whole leading + body line round-trips via raw.
		assert.Equal(t, []byte(content), attr.RawBytes())
	})

	t.Run("attribute with inline trailing comment", func(t *testing.T) {
		t.Parallel()
		content := "name = \"value\" # why\n"
		f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		attr, ok := f.Body.Items[0].(*Attribute)
		require.True(t, ok)
		require.NotNil(t, attr.InlineComment)
		assert.Equal(t, CommentHash, attr.InlineComment.Style)
	})
}

// TestBuild_SingleBlock covers a single block at the top level, with and
// without a leading comment. Both labeled and label-less variants run.
func TestBuild_SingleBlock(t *testing.T) {
	t.Parallel()

	t.Run("labeled empty block", func(t *testing.T) {
		t.Parallel()
		content := "resource \"aws_instance\" \"x\" {}\n"
		f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		blk, ok := f.Body.Items[0].(*Block)
		require.True(t, ok, "expected *Block, got %T", f.Body.Items[0])
		assert.Equal(t, "resource", blk.Type)
		require.Len(t, blk.Labels, 2)
		assert.Equal(t, "aws_instance", blk.Labels[0].Text)
		assert.Equal(t, "x", blk.Labels[1].Text)
		assert.Empty(t, blk.LeadingComments)
		require.NotNil(t, blk.Body)
		assert.Empty(t, blk.Body.Items)
	})

	t.Run("labeless locals block with attribute", func(t *testing.T) {
		t.Parallel()
		content := "locals {\n  x = 1\n}\n"
		f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		blk, ok := f.Body.Items[0].(*Block)
		require.True(t, ok)
		assert.Equal(t, "locals", blk.Type)
		assert.Empty(t, blk.Labels)
		require.NotNil(t, blk.Body)
		require.Len(t, blk.Body.Items, 1)

		_, ok = blk.Body.Items[0].(*Attribute)
		assert.True(t, ok, "block body should hold a single *Attribute")
	})

	t.Run("block with adjacent leading comment", func(t *testing.T) {
		t.Parallel()
		content := "# header\nresource \"aws_instance\" \"x\" {}\n"
		f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		blk, ok := f.Body.Items[0].(*Block)
		require.True(t, ok)
		require.Len(t, blk.LeadingComments, 1)
		assert.Equal(t, CommentHash, blk.LeadingComments[0].Style)
		assert.Equal(t, []byte(content), blk.RawBytes())
	})
}

// TestBuild_NestedBlocks verifies the CST recurses into nested block bodies
// 3+ levels deep, and that inner Body brace offsets are populated.
func TestBuild_NestedBlocks(t *testing.T) {
	t.Parallel()

	content := "a {\n  b {\n    c {\n      x = 1\n    }\n  }\n}\n"
	f, err := Build([]byte(content), "nested.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.Len(t, f.Body.Items, 1)

	a, ok := f.Body.Items[0].(*Block)
	require.True(t, ok)
	assert.Equal(t, "a", a.Type)
	require.NotNil(t, a.Body)
	assert.NotEqual(t, -1, a.Body.OpenByte)
	assert.NotEqual(t, -1, a.Body.CloseByte)
	require.Len(t, a.Body.Items, 1)

	b, ok := a.Body.Items[0].(*Block)
	require.True(t, ok)
	assert.Equal(t, "b", b.Type)
	require.NotNil(t, b.Body)
	require.Len(t, b.Body.Items, 1)

	c, ok := b.Body.Items[0].(*Block)
	require.True(t, ok)
	assert.Equal(t, "c", c.Type)
	require.NotNil(t, c.Body)
	require.Len(t, c.Body.Items, 1)

	attr, ok := c.Body.Items[0].(*Attribute)
	require.True(t, ok)
	assert.Equal(t, "x", attr.Name)
}

// TestBuild_TerragruntBlocks confirms that Terragrunt-flavored block types
// (include, dependency, locals) are recognized as first-class Blocks with
// their bodies parsed as normal. The CST does not dispatch on block type —
// these are just Blocks.
func TestBuild_TerragruntBlocks(t *testing.T) {
	t.Parallel()

	content := "" +
		"include \"root\" {\n" +
		"  path = \"../\"\n" +
		"}\n" +
		"\n" +
		"dependency \"vpc\" {\n" +
		"  config_path = \"../vpc\"\n" +
		"}\n" +
		"\n" +
		"locals {\n" +
		"  x = 1\n" +
		"}\n"

	f, err := Build([]byte(content), "terragrunt.hcl", DefaultTopLevelPolicy())
	require.NoError(t, err)
	// 3 blocks + 2 separating BlankLines.
	require.Len(t, f.Body.Items, 5)

	wantTypes := []string{"include", "dependency", "locals"}
	blockIdx := 0
	for i, item := range f.Body.Items {
		switch i {
		case 1, 3:
			_, ok := item.(*BlankLine)
			assert.True(t, ok, "item %d should be *BlankLine, got %T", i, item)
		default:
			blk, ok := item.(*Block)
			require.True(t, ok, "item %d should be *Block, got %T", i, item)
			assert.Equal(t, wantTypes[blockIdx], blk.Type)
			blockIdx++
		}
	}
}

// TestBuild_FilenameDoesNotAffectShape verifies that Build dispatches on
// content, not file extension — mixed .tf and .hcl files in the same
// fixture set parse to identical CSTs given identical content. Each call
// asserts against literals so a hypothetical `.hcl`-only divergence fails
// even though both calls produce identical output.
func TestBuild_FilenameDoesNotAffectShape(t *testing.T) {
	t.Parallel()

	content := []byte("name = \"value\"\n")

	for _, filename := range []string{"x.tf", "x.hcl", "terragrunt.hcl"} {
		t.Run(filename, func(t *testing.T) {
			t.Parallel()
			f, err := Build(content, filename, DefaultTopLevelPolicy())
			require.NoError(t, err)
			require.Len(t, f.Body.Items, 1)

			attr, ok := f.Body.Items[0].(*Attribute)
			require.True(t, ok)
			assert.Equal(t, "name", attr.Name)
			assert.Equal(t, content, attr.RawBytes())
		})
	}
}

// TestBuild_PolicyAttachment exercises the StrictAdjacency vs passthrough
// distinction explicitly at the top level: when a comment is separated
// from the next item by a blank line, strict policy yields a
// StandaloneComment, passthrough policy attaches it as a LeadingComment.
func TestBuild_PolicyAttachment(t *testing.T) {
	t.Parallel()

	// Content: leading blank, then "# header", then attr. The blank is
	// ABOVE the comment (gap order: blank, comment, attr), which is the
	// only configuration where policy choice matters.
	content := []byte("\n# header\nattr = 1\n")

	t.Run("strict makes the comment standalone", func(t *testing.T) {
		t.Parallel()
		f, err := Build(content, "x.tf", Policy{StrictAdjacency: true})
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 3)

		_, ok := f.Body.Items[0].(*BlankLine)
		assert.True(t, ok, "item 0 should be *BlankLine, got %T", f.Body.Items[0])

		sc, ok := f.Body.Items[1].(*StandaloneComment)
		require.True(t, ok, "item 1 should be *StandaloneComment, got %T", f.Body.Items[1])
		assert.Len(t, sc.Comments, 1)

		attr, ok := f.Body.Items[2].(*Attribute)
		require.True(t, ok)
		assert.Empty(t, attr.LeadingComments)
	})

	t.Run("passthrough attaches the comment as leading", func(t *testing.T) {
		t.Parallel()
		f, err := Build(content, "x.tf", Policy{StrictAdjacency: false})
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 2)

		_, ok := f.Body.Items[0].(*BlankLine)
		assert.True(t, ok)

		attr, ok := f.Body.Items[1].(*Attribute)
		require.True(t, ok)
		require.Len(t, attr.LeadingComments, 1)
		assert.Equal(t, CommentHash, attr.LeadingComments[0].Style)
	})
}

// TestBuild_BlockBodyUsesPassthroughByDefault confirms the policy split:
// regardless of the top-level policy, nested block bodies always use
// DefaultBlockBodyPolicy() (passthrough). A blank-separated comment
// inside a block body therefore attaches as LeadingComment on the next
// attribute even when the top-level was built with the strict default.
func TestBuild_BlockBodyUsesPassthroughByDefault(t *testing.T) {
	t.Parallel()

	t.Run("blank-separated comment attaches to next attr", func(t *testing.T) {
		t.Parallel()

		content := []byte(
			"" +
				"parent {\n" +
				"  attr1 = 1\n" +
				"\n" +
				"  # header\n" +
				"  attr2 = 2\n" +
				"}\n",
		)

		f, err := Build(content, "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		parent, ok := f.Body.Items[0].(*Block)
		require.True(t, ok)
		require.NotNil(t, parent.Body)
		// attr1, BlankLine, attr2-with-leading-comment.
		require.Len(t, parent.Body.Items, 3)

		attr1, ok := parent.Body.Items[0].(*Attribute)
		require.True(t, ok)
		assert.Equal(t, "attr1", attr1.Name)
		assert.Empty(t, attr1.LeadingComments)

		_, ok = parent.Body.Items[1].(*BlankLine)
		assert.True(t, ok, "item 1 should be *BlankLine")

		attr2, ok := parent.Body.Items[2].(*Attribute)
		require.True(t, ok)
		assert.Equal(t, "attr2", attr2.Name)
		require.Len(t, attr2.LeadingComments, 1,
			"passthrough should attach the blank-separated comment to attr2")
	})

	t.Run("trailing comment before closing brace stays standalone", func(t *testing.T) {
		t.Parallel()

		// Comment after the last attribute has no next item inside the
		// body, so classifyGap's hasNextItem=false branch fires and the
		// comment becomes a StandaloneComment regardless of policy. This
		// exercises the block-body trailing-gap sweep path in buildBody
		// (cursor < bodyEndByte after the items loop).
		content := []byte(
			"" +
				"parent {\n" +
				"  attr = 1\n" +
				"\n" +
				"  # trailing\n" +
				"}\n",
		)

		f, err := Build(content, "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		require.Len(t, f.Body.Items, 1)

		parent, ok := f.Body.Items[0].(*Block)
		require.True(t, ok)
		require.NotNil(t, parent.Body)
		require.Len(t, parent.Body.Items, 3,
			"expected attr + BlankLine + StandaloneComment")

		_, ok = parent.Body.Items[0].(*Attribute)
		assert.True(t, ok, "item 0 should be *Attribute")
		_, ok = parent.Body.Items[1].(*BlankLine)
		assert.True(t, ok, "item 1 should be *BlankLine")
		sc, ok := parent.Body.Items[2].(*StandaloneComment)
		require.True(t, ok, "item 2 should be *StandaloneComment")
		assert.Len(t, sc.Comments, 1)
	})
}

// TestBuild_CRLFDetection asserts that line-ending detection picks CRLF
// when the first newline is preceded by a CR. Locks the contract that
// replaces the legacy SplitLines-based path (which stripped CRLF and broke
// round-trip on Windows-authored fixtures).
func TestBuild_CRLFDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []byte
	}{
		{"crlf throughout", "name = \"value\"\r\n", []byte("\r\n")},
		{"lf throughout", "name = \"value\"\n", []byte("\n")},
		{"crlf first wins over later lf", "a = 1\r\nb = 2\n", []byte("\r\n")},
		{"lf first wins over later crlf", "a = 1\nb = 2\r\n", []byte("\n")},
		{"no newline defaults to lf", "name = \"value\"", []byte("\n")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte(tc.content), "x.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)
			assert.Equal(t, tc.want, f.LineSep())
		})
	}
}

// TestBuild_BlockBraceComments verifies that an inline comment on a
// block's opening or closing brace line is captured on the Block, not
// lost into the surrounding gaps.
func TestBuild_BlockBraceComments(t *testing.T) {
	t.Parallel()

	content := "" +
		"resource \"aws_instance\" \"x\" { # why\n" +
		"  name = \"value\"\n" +
		"} # end\n"

	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.Len(t, f.Body.Items, 1)

	blk, ok := f.Body.Items[0].(*Block)
	require.True(t, ok)

	require.NotNil(t, blk.OpeningBraceComment, "opening-brace inline comment should be captured")
	assert.Equal(t, CommentHash, blk.OpeningBraceComment.Style)

	require.NotNil(t, blk.ClosingBraceComment, "closing-brace inline comment should be captured")
	assert.Equal(t, CommentHash, blk.ClosingBraceComment.Style)
}

// TestBuild_ParseError_ReturnsPartialAndError pins the partial-tree
// contract: on parse failure, Build returns a non-nil File plus the
// hclsyntax diagnostics. Rule Fix migrations rely on this to preserve
// the no-op-on-parse-error pattern.
func TestBuild_ParseError_ReturnsPartialAndError(t *testing.T) {
	t.Parallel()

	// Unterminated block: `{` opens, EOF before `}`.
	content := []byte("resource \"x\" \"y\" {\n  name = \"value\"\n")

	f, err := Build(content, "broken.tf", DefaultTopLevelPolicy())
	require.Error(t, err, "expected parse error on unterminated block")
	require.NotNil(t, f, "Build must return a non-nil File even on parse error")
	require.NotNil(t, f.Body, "f.Body must be non-nil so rule Fix can pattern-match on it")
	assert.Equal(t, content, f.Source,
		"f.Source must round-trip the original bytes even on parse error")
	// Sentinel brace offsets pin the file-level body shape. The Items
	// length depends on what hclsyntax salvages from a malformed input
	// and is intentionally not asserted (per-version variance).
	assert.Equal(t, -1, f.Body.OpenByte)
	assert.Equal(t, -1, f.Body.CloseByte)
}

// TestBuild_NoFinalNewline confirms that a file missing its trailing
// newline round-trips via the captured raw bytes — Build does not
// normalize the input.
func TestBuild_NoFinalNewline(t *testing.T) {
	t.Parallel()

	content := []byte("name = \"value\"")
	f, err := Build(content, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.Len(t, f.Body.Items, 1)

	attr, ok := f.Body.Items[0].(*Attribute)
	require.True(t, ok)
	assert.Equal(t, content, attr.RawBytes(),
		"raw bytes must include every input byte verbatim, with no synthesized newline")
}

// itemKind returns a human-readable name for a BodyItem, used by the
// table-driven OnlyComments test to express expected sequences as plain
// strings rather than typed sentinels.
func itemKind(item BodyItem) string {
	switch item.(type) {
	case *Attribute:
		return "Attribute"
	case *Block:
		return "Block"
	case *BlankLine:
		return "BlankLine"
	case *StandaloneComment:
		return "StandaloneComment"
	default:
		return "Unknown"
	}
}
