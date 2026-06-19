package cst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// itemKinds returns a compact discriminator-prefixed name for each item in
// items, in order: `A:<name>` for Attribute, `B:<type>` for Block, `L` for
// BlankLine, `C` for StandaloneComment. Lets assertions read as the source
// structure without unwrapping each variant by hand.
func itemKinds(items []BodyItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case *Attribute:
			out = append(out, "A:"+v.Name)
		case *Block:
			out = append(out, "B:"+v.Type)
		case *BlankLine:
			out = append(out, "L")
		case *StandaloneComment:
			out = append(out, "C")
		}
	}
	return out
}

// TestMove_AttrToFirstMiddleLast covers the three positional cases for Move
// on attribute items in a flat body: src moves to first (downshift below),
// to middle (cross over neighbors), and to last (upshift above). Body
// composition is homogeneous so the test isolates slice math from
// item-type interactions; the BlankLine / StandaloneComment cases are
// covered in TestMove_StandaloneCommentStaysInPlace.
func TestMove_AttrToFirstMiddleLast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		moveName  string
		newIdx    int
		wantOrder []string
		wantBytes string
	}{
		{
			name:      "move c to first",
			moveName:  "c",
			newIdx:    0,
			wantOrder: []string{"A:c", "A:a", "A:b"},
			wantBytes: "c = 3\na = 1\nb = 2\n",
		},
		{
			name:      "move a to middle",
			moveName:  "a",
			newIdx:    1,
			wantOrder: []string{"A:b", "A:a", "A:c"},
			wantBytes: "b = 2\na = 1\nc = 3\n",
		},
		{
			name:      "move a to last",
			moveName:  "a",
			newIdx:    2,
			wantOrder: []string{"A:b", "A:c", "A:a"},
			wantBytes: "b = 2\nc = 3\na = 1\n",
		},
		{
			name:      "move to same index is no-op success",
			moveName:  "b",
			newIdx:    1,
			wantOrder: []string{"A:a", "A:b", "A:c"},
			wantBytes: "a = 1\nb = 2\nc = 3\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte("a = 1\nb = 2\nc = 3\n"), "x.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)

			target := f.Body.FindAttribute(tc.moveName)
			require.NotNil(t, target)
			ok := f.Body.Move(target, tc.newIdx)
			require.True(t, ok)

			assert.Equal(t, tc.wantOrder, itemKinds(f.Body.Items))
			assert.Equal(t, tc.wantBytes, string(f.Bytes()))
		})
	}
}

// TestMove_LeadingCommentsTravel pins that a moved attribute carries its
// leading comment with it. Leading comments live in the Attribute's raw
// bytes for items unmodified since Build, so this is mechanical — but the
// test guards against a future regression where someone splits leading
// comments from raw and forgets to move them with the item.
func TestMove_LeadingCommentsTravel(t *testing.T) {
	t.Parallel()

	// Strings concatenated to break textual "x\nx" patterns dupword
	// flags as duplicate words across the newline.
	content := "# describes apple\n" + "apple = 1\n" + "# describes banana\n" + "banana = 2\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	banana := f.Body.FindAttribute("banana")
	require.NotNil(t, banana)
	require.True(t, f.Body.Move(banana, 0))

	want := "# describes banana\n" + "banana = 2\n" + "# describes apple\n" + "apple = 1\n"
	assert.Equal(t, want, string(f.Bytes()))
}

// TestMove_StandaloneCommentStaysInPlace pins the floating-section-header
// preservation that fixes the bug in `style.terraform-block-first` where
// reordering a block swept along the preceding standalone comment. When
// a top-level block moves, StandaloneComment items in the same body are
// NOT reshuffled: the same pointer survives in body.Items with the same
// content. Their numeric index may shift as the moved item passes over
// them, but they are never re-attached as a LeadingComment on the moved
// block.
func TestMove_StandaloneCommentStaysInPlace(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_sns_topic\" \"alerts\" {\n  name = \"x\"\n}\n\n" +
		"### SNS Notifications\n\n" +
		"resource \"aws_sns_subscription\" \"email\" {\n  topic = \"y\"\n}\n\n" +
		"terraform {\n  required_version = \">= 1.0\"\n}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	var sc *StandaloneComment
	for _, item := range f.Body.Items {
		if s, ok := item.(*StandaloneComment); ok {
			sc = s
			break
		}
	}
	require.NotNil(t, sc, "expected a StandaloneComment in items")
	require.Equal(t, "### SNS Notifications\n", string(sc.RawBytes()))

	tf := f.Body.FindBlock("terraform")
	require.NotNil(t, tf)
	require.True(t, f.Body.Move(tf, 0))

	survived := false
	for _, item := range f.Body.Items {
		if item == sc {
			survived = true
		}
	}
	assert.True(t, survived, "StandaloneComment must still be present after Move")
	assert.Equal(t, tf, f.Body.Items[0], "terraform block must be at index 0")

	// The SNS section header must remain between the two resource blocks
	// in the serialized output, not slide up to follow the terraform block.
	// Pre-Move layout is [B:resource, L, C, L, B:resource, L, B:terraform].
	// Post-Move layout is [B:terraform, B:resource, L, C, L, B:resource, L].
	assert.Equal(
		t,
		[]string{"B:terraform", "B:resource", "L", "C", "L", "B:resource", "L"},
		itemKinds(f.Body.Items),
		"Move(tf, 0) shifts items right of insertion; standalone comment keeps its slot relative to the resource blocks",
	)
	assert.Contains(t, string(f.Bytes()), "### SNS Notifications")
}

// TestMove_FailureModes pins the contract that Move returns false (without
// mutating body.Items) on out-of-range indices, items not in the body, and
// nil item — a defensive set of guards that prevent silent corruption on
// caller misuse.
func TestMove_FailureModes(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) *File {
		t.Helper()
		f, err := Build([]byte("a = 1\nb = 2\nc = 3\n"), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		return f
	}

	t.Run("newIndex negative", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		ok := f.Body.Move(f.Body.FindAttribute("a"), -1)
		assert.False(t, ok)
		assert.Equal(t, []string{"A:a", "A:b", "A:c"}, itemKinds(f.Body.Items))
	})

	t.Run("newIndex == len", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		ok := f.Body.Move(f.Body.FindAttribute("a"), len(f.Body.Items))
		assert.False(t, ok)
		assert.Equal(t, []string{"A:a", "A:b", "A:c"}, itemKinds(f.Body.Items))
	})

	t.Run("item not in body", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		other := &Attribute{Name: "x", ExpressionBytes: []byte("1")}
		ok := f.Body.Move(other, 0)
		assert.False(t, ok)
		assert.Equal(t, []string{"A:a", "A:b", "A:c"}, itemKinds(f.Body.Items))
	})

	t.Run("nil item", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		ok := f.Body.Move(nil, 0)
		assert.False(t, ok)
	})

	t.Run("empty body", func(t *testing.T) {
		t.Parallel()
		f, err := Build([]byte(""), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		ok := f.Body.Move(&Attribute{Name: "x"}, 0)
		assert.False(t, ok)
	})
}

// TestMoveBefore_PositionsImmediatelyBeforeDst covers MoveBefore's two
// arithmetic branches: src is originally below dst (dst index shifts left
// on src removal, so target collapses to dstIdx-1), and src is originally
// above dst (dst stays put, target is dstIdx). Plus the adjacent-no-op
// case where the source already sits where MoveBefore would place it.
func TestMoveBefore_PositionsImmediatelyBeforeDst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		srcName   string
		dstName   string
		wantOrder []string
	}{
		{
			name:      "src above dst",
			srcName:   "a",
			dstName:   "d",
			wantOrder: []string{"A:b", "A:c", "A:a", "A:d"},
		},
		{
			name:      "src below dst",
			srcName:   "d",
			dstName:   "b",
			wantOrder: []string{"A:a", "A:d", "A:b", "A:c"},
		},
		{
			name:      "src immediately above dst is no-op",
			srcName:   "b",
			dstName:   "c",
			wantOrder: []string{"A:a", "A:b", "A:c", "A:d"},
		},
		{
			name:      "src to before first item",
			srcName:   "d",
			dstName:   "a",
			wantOrder: []string{"A:d", "A:a", "A:b", "A:c"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte("a = 1\nb = 2\nc = 3\nd = 4\n"), "x.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)

			src := f.Body.FindAttribute(tc.srcName)
			dst := f.Body.FindAttribute(tc.dstName)
			require.NotNil(t, src)
			require.NotNil(t, dst)
			require.True(t, f.Body.MoveBefore(src, dst))

			assert.Equal(t, tc.wantOrder, itemKinds(f.Body.Items))
		})
	}
}

// TestMoveAfter_PositionsImmediatelyAfterDst mirrors MoveBefore but for the
// MoveAfter direction. Both arithmetic branches and the adjacent-no-op case
// are exercised.
func TestMoveAfter_PositionsImmediatelyAfterDst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		srcName   string
		dstName   string
		wantOrder []string
	}{
		{
			name:      "src above dst",
			srcName:   "a",
			dstName:   "c",
			wantOrder: []string{"A:b", "A:c", "A:a", "A:d"},
		},
		{
			name:      "src below dst",
			srcName:   "d",
			dstName:   "b",
			wantOrder: []string{"A:a", "A:b", "A:d", "A:c"},
		},
		{
			name:      "src immediately below dst is no-op",
			srcName:   "c",
			dstName:   "b",
			wantOrder: []string{"A:a", "A:b", "A:c", "A:d"},
		},
		{
			name:      "src to after last item",
			srcName:   "a",
			dstName:   "d",
			wantOrder: []string{"A:b", "A:c", "A:d", "A:a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte("a = 1\nb = 2\nc = 3\nd = 4\n"), "x.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)

			src := f.Body.FindAttribute(tc.srcName)
			dst := f.Body.FindAttribute(tc.dstName)
			require.NotNil(t, src)
			require.NotNil(t, dst)
			require.True(t, f.Body.MoveAfter(src, dst))

			assert.Equal(t, tc.wantOrder, itemKinds(f.Body.Items))
		})
	}
}

// TestMoveBeforeAfter_FailureModes covers the rejection paths for both
// MoveBefore and MoveAfter: missing src, missing dst, and src == dst.
func TestMoveBeforeAfter_FailureModes(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) (*File, *Attribute) {
		t.Helper()
		f, err := Build([]byte("a = 1\nb = 2\n"), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		return f, f.Body.FindAttribute("a")
	}

	t.Run("MoveBefore src missing", func(t *testing.T) {
		t.Parallel()
		f, _ := build(t)
		dst := f.Body.FindAttribute("b")
		ok := f.Body.MoveBefore(&Attribute{Name: "ghost"}, dst)
		assert.False(t, ok)
	})

	t.Run("MoveBefore dst missing", func(t *testing.T) {
		t.Parallel()
		f, src := build(t)
		ok := f.Body.MoveBefore(src, &Attribute{Name: "ghost"})
		assert.False(t, ok)
	})

	t.Run("MoveBefore src == dst", func(t *testing.T) {
		t.Parallel()
		f, src := build(t)
		ok := f.Body.MoveBefore(src, src)
		assert.False(t, ok)
	})

	t.Run("MoveAfter src missing", func(t *testing.T) {
		t.Parallel()
		f, _ := build(t)
		dst := f.Body.FindAttribute("b")
		ok := f.Body.MoveAfter(&Attribute{Name: "ghost"}, dst)
		assert.False(t, ok)
	})

	t.Run("MoveAfter dst missing", func(t *testing.T) {
		t.Parallel()
		f, src := build(t)
		ok := f.Body.MoveAfter(src, &Attribute{Name: "ghost"})
		assert.False(t, ok)
	})

	t.Run("MoveAfter src == dst", func(t *testing.T) {
		t.Parallel()
		f, src := build(t)
		ok := f.Body.MoveAfter(src, src)
		assert.False(t, ok)
	})
}

// TestInsert_AtBoundaries covers Insert at index 0 (prepend), middle, and
// index == len(Items) (append). Each case verifies the inserted item lands
// at the right index and surrounding items don't reshuffle except for the
// required right-shift.
func TestInsert_AtBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		index     int
		wantOrder []string
	}{
		{name: "prepend", index: 0, wantOrder: []string{"A:x", "A:a", "A:b", "A:c"}},
		{name: "middle", index: 2, wantOrder: []string{"A:a", "A:b", "A:x", "A:c"}},
		{name: "append", index: 3, wantOrder: []string{"A:a", "A:b", "A:c", "A:x"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := Build([]byte("a = 1\nb = 2\nc = 3\n"), "x.tf", DefaultTopLevelPolicy())
			require.NoError(t, err)

			// Fresh attribute, regenerated path on serialize (raw is nil).
			x := &Attribute{
				Name:            "x",
				NameBytes:       []byte("x"),
				ExpressionBytes: []byte("42"),
			}
			require.True(t, f.Body.Insert(x, tc.index))
			assert.Equal(t, tc.wantOrder, itemKinds(f.Body.Items))
		})
	}
}

// TestInsert_FailureModes pins the rejection paths: nil item, negative
// index, index past end-of-slice. Each case must leave body.Items unchanged.
func TestInsert_FailureModes(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) *File {
		t.Helper()
		f, err := Build([]byte("a = 1\nb = 2\n"), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		return f
	}

	t.Run("nil item", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		assert.False(t, f.Body.Insert(nil, 0))
		assert.Equal(t, []string{"A:a", "A:b"}, itemKinds(f.Body.Items))
	})

	t.Run("negative index", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		assert.False(t, f.Body.Insert(&Attribute{Name: "x"}, -1))
		assert.Equal(t, []string{"A:a", "A:b"}, itemKinds(f.Body.Items))
	})

	t.Run("index past end", func(t *testing.T) {
		t.Parallel()
		f := build(t)
		assert.False(t, f.Body.Insert(&Attribute{Name: "x"}, len(f.Body.Items)+1))
		assert.Equal(t, []string{"A:a", "A:b"}, itemKinds(f.Body.Items))
	})
}

// TestRemove_PreservesAdjacentBlankLines pins that Remove does NOT
// collapse surrounding whitespace: adjacent BlankLines stay put.
// A future BlankLinePolicy hook could opt into collapsing; the present
// default semantics are deliberately minimal.
func TestRemove_PreservesAdjacentBlankLines(t *testing.T) {
	t.Parallel()

	content := "a = 1\n\nb = 2\n\nc = 3\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.Equal(t, []string{"A:a", "L", "A:b", "L", "A:c"}, itemKinds(f.Body.Items))

	b := f.Body.FindAttribute("b")
	require.NotNil(t, b)
	require.True(t, f.Body.Remove(b))

	assert.Equal(t, []string{"A:a", "L", "L", "A:c"}, itemKinds(f.Body.Items),
		"two BlankLines must remain — Remove does not collapse adjacent whitespace")
	assert.Equal(t, "a = 1\n\n\nc = 3\n", string(f.Bytes()))
}

// TestRemove_FailureModes covers item-not-in-body and nil-item.
func TestRemove_FailureModes(t *testing.T) {
	t.Parallel()

	f, err := Build([]byte("a = 1\n"), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	t.Run("item not in body", func(t *testing.T) {
		t.Parallel()
		assert.False(t, f.Body.Remove(&Attribute{Name: "ghost"}))
	})

	t.Run("nil item", func(t *testing.T) {
		t.Parallel()
		assert.False(t, f.Body.Remove(nil))
	})
}

// TestFindAttribute covers hit, miss, and pick-first-on-duplicate (which
// HCL would actually flag as a parse error, but the CST shouldn't assume —
// it returns the first match in source order regardless).
func TestFindAttribute(t *testing.T) {
	t.Parallel()

	f, err := Build([]byte("a = 1\nb = 2\nc = 3\n"), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	t.Run("hit returns first match", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindAttribute("b")
		require.NotNil(t, got)
		assert.Equal(t, "b", got.Name)
	})

	t.Run("miss returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, f.Body.FindAttribute("nope"))
	})

	t.Run("empty body returns nil", func(t *testing.T) {
		t.Parallel()
		empty, err := Build([]byte(""), "x.tf", DefaultTopLevelPolicy())
		require.NoError(t, err)
		assert.Nil(t, empty.Body.FindAttribute("anything"))
	})
}

// TestFindBlock covers hit, miss, and first-of-multiple-matches.
func TestFindBlock(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"a\" {\n}\n" +
		"resource \"aws_instance\" \"b\" {\n}\n" +
		"variable \"x\" {\n}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	t.Run("hit returns first source-order match", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlock("resource")
		require.NotNil(t, got)
		require.Len(t, got.Labels, 2)
		assert.Equal(t, "a", got.Labels[1].Text, "first source-order match should win")
	})

	t.Run("miss returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, f.Body.FindBlock("module"))
	})
}

// TestFindBlocksByType verifies all matches are returned in source order
// and a no-match query returns nil (not an empty slice).
func TestFindBlocksByType(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"a\" {\n}\n" +
		"variable \"v\" {\n}\n" +
		"resource \"aws_instance\" \"b\" {\n}\n" +
		"resource \"aws_s3_bucket\" \"c\" {\n}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	t.Run("multiple matches in source order", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlocksByType("resource")
		require.Len(t, got, 3)
		assert.Equal(t, "a", got[0].Labels[1].Text)
		assert.Equal(t, "b", got[1].Labels[1].Text)
		assert.Equal(t, "c", got[2].Labels[1].Text)
	})

	t.Run("single match", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlocksByType("variable")
		require.Len(t, got, 1)
		assert.Equal(t, "v", got[0].Labels[0].Text)
	})

	t.Run("no match returns nil", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlocksByType("module")
		assert.Nil(t, got, "no-match path returns nil, not empty slice")
	})
}

// TestFindBlockByTypeAndLabels pins exact-label matching: a block with a
// different label count never matches, label position matters, and an
// empty/nil labels slice matches only zero-label blocks.
func TestFindBlockByTypeAndLabels(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"web\" {\n}\n" +
		"resource \"aws_instance\" \"db\" {\n}\n" +
		"terraform {\n}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	t.Run("exact two-label match", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlockByTypeAndLabels("resource", []string{"aws_instance", "db"})
		require.NotNil(t, got)
		assert.Equal(t, "db", got.Labels[1].Text)
	})

	t.Run("label position matters", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlockByTypeAndLabels("resource", []string{"db", "aws_instance"})
		assert.Nil(t, got, "labels in wrong order must not match")
	})

	t.Run("label count mismatch never matches", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlockByTypeAndLabels("resource", []string{"aws_instance"})
		assert.Nil(t, got, "one-label query must not match two-label block")
	})

	t.Run("zero-label block matches nil labels", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlockByTypeAndLabels("terraform", nil)
		require.NotNil(t, got)
		assert.Equal(t, "terraform", got.Type)
	})

	t.Run("zero-label block matches empty labels", func(t *testing.T) {
		t.Parallel()
		got := f.Body.FindBlockByTypeAndLabels("terraform", []string{})
		require.NotNil(t, got)
		assert.Equal(t, "terraform", got.Type)
	})

	t.Run("miss returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, f.Body.FindBlockByTypeAndLabels("module", []string{"x"}))
	})
}

// TestFind_NestedBodyIsolated pins that Find* on a body sees only that
// body's items — not items in nested block bodies. A `tags = {}` attribute
// inside a `lifecycle { ... }` block is not visible to the top-level
// FindAttribute("tags") lookup. Sibling-of-current-body lookups are exactly
// what the structural rules need; deep recursion would be wrong.
func TestFind_NestedBodyIsolated(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"web\" {\n" +
		"  lifecycle {\n" +
		"    create_before_destroy = true\n" +
		"  }\n" +
		"}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	assert.Nil(t, f.Body.FindAttribute("create_before_destroy"),
		"top-level FindAttribute must not descend into block bodies")

	res := f.Body.FindBlock("resource")
	require.NotNil(t, res)
	assert.Nil(t, res.Body.FindAttribute("create_before_destroy"),
		"resource Body's FindAttribute must not descend into lifecycle either")

	lc := res.Body.FindBlock("lifecycle")
	require.NotNil(t, lc)
	assert.NotNil(t, lc.Body.FindAttribute("create_before_destroy"),
		"lifecycle Body's FindAttribute must find its own attribute")
}

// TestRoundTrip_AfterMove pins the contract that Move → Bytes → re-Build
// produces an item list matching the post-mutation state. Ops only
// reshuffle existing items (no creation or destruction), so re-parsing the
// serialized output must recover the same structural sequence.
//
// This is the safety contract for structural rules built on the CST:
// rules build a CST, mutate it, serialize, and emit a WholeFileEdit. The
// downstream engine layer never sees a half-broken intermediate state.
func TestRoundTrip_AfterMove(t *testing.T) {
	t.Parallel()

	content := []byte("resource \"aws_sns_topic\" \"alerts\" {\n  name = \"x\"\n}\n\n" +
		"resource \"aws_sns_subscription\" \"email\" {\n  topic = \"y\"\n}\n\n" +
		"terraform {\n  required_version = \">= 1.0\"\n}\n")
	f, err := Build(content, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	tf := f.Body.FindBlock("terraform")
	require.NotNil(t, tf)
	require.True(t, f.Body.Move(tf, 0))

	out := f.Bytes()
	f2, err := Build(out, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	assert.Equal(t, itemKinds(f.Body.Items), itemKinds(f2.Body.Items),
		"re-Build of serialized output must yield the same item sequence")
}

// TestRoundTrip_AfterInsertAndRemove pins the same contract under Insert
// and Remove. A freshly-built BlankLine round-trips because Serialize emits
// a regenerated `\n` for it; an existing item round-trips because its raw
// bytes write through verbatim.
func TestRoundTrip_AfterInsertAndRemove(t *testing.T) {
	t.Parallel()

	content := []byte("a = 1\nb = 2\nc = 3\n")
	f, err := Build(content, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	require.True(t, f.Body.Insert(&BlankLine{Count: 1}, 1))
	require.True(t, f.Body.Remove(f.Body.FindAttribute("c")))

	out := f.Bytes()
	assert.Equal(t, "a = 1\n\nb = 2\n", string(out))

	f2, err := Build(out, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	assert.Equal(t, itemKinds(f.Body.Items), itemKinds(f2.Body.Items))
}

// TestInsert_BlockWiresParentage pins that Insert of a *Block sets up the
// back-pointer chain so subsequent mutations on the inserted block's body
// invalidate ancestor raw bytes and reflect in serialize output. Without
// the parentage wiring at ops.go:97-104, mutations on the inserted block's
// body would be invisible — Serialize would write the surrounding bytes
// unaware of the change.
func TestInsert_BlockWiresParentage(t *testing.T) {
	t.Parallel()

	f, err := Build([]byte("a = 1\n"), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	// Fresh block constructed from scratch with two attributes in its body.
	freshBlk := &Block{
		Type:      "resource",
		TypeBytes: []byte("resource"),
		Labels: []Label{
			{Text: "aws_instance", Raw: []byte(`"aws_instance"`)},
			{Text: "web", Raw: []byte(`"web"`)},
		},
		Body: &Body{Items: []BodyItem{
			&Attribute{Name: "ami", NameBytes: []byte("ami"), ExpressionBytes: []byte(`"ami-1"`)},
			&Attribute{Name: "instance_type", NameBytes: []byte("instance_type"), ExpressionBytes: []byte(`"t3.micro"`)},
		}},
	}
	require.True(t, f.Body.Insert(freshBlk, 1))

	// Mutate the inserted block's body: reorder so instance_type comes first.
	it := freshBlk.Body.FindAttribute("instance_type")
	require.NotNil(t, it)
	require.True(t, freshBlk.Body.Move(it, 0))

	// Serialize and re-Build. If parentage wiring is intact, markDirty
	// propagated through freshBlk's parentBody (= f.Body), but f.Body has
	// no parentBlock so the walk stops at the top — meanwhile freshBlk's
	// own raw is nil from construction, so it always regenerates. The
	// post-move order must round-trip.
	out := f.Bytes()
	f2, err := Build(out, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	res := f2.Body.FindBlock("resource")
	require.NotNil(t, res)
	assert.Equal(t, []string{"A:instance_type", "A:ami"}, itemKinds(res.Body.Items),
		"mutation on inserted block's body must reflect in serialize output")
}

// TestRemove_BlockClearsParentage pins the safety contract that a Remove'd
// *Block has its parentBody cleared, so a stale back-link does not trip
// markDirty into invalidating a body the block is no longer part of.
func TestRemove_BlockClearsParentage(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"web\" {\n  ami = \"ami-1\"\n}\n" +
		"resource \"aws_instance\" \"db\" {\n  ami = \"ami-2\"\n}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	blocks := f.Body.FindBlocksByType("resource")
	require.Len(t, blocks, 2)
	removed := blocks[0]

	require.True(t, f.Body.Remove(removed))
	assert.Nil(t, removed.parentBody,
		"removed block must have parentBody cleared to prevent stale back-link")

	// Round-trip: the removed block is absent from the re-Built CST.
	out := f.Bytes()
	f2, err := Build(out, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	remaining := f2.Body.FindBlocksByType("resource")
	require.Len(t, remaining, 1)
	assert.Equal(t, "db", remaining[0].Labels[1].Text,
		"only the second resource block must remain after Remove")
}

// TestRemove_DetachedSubtreeStopsMarkDirtyWalk pins the detached-subtree
// guard in markDirty: when a Block is Removed from its parent body, its
// parentBody is cleared, but the block's own nested Body still has its
// parentBlock pointing back at the (now-detached) block. A subsequent
// mutation on the detached subtree's inner body must terminate the
// dirty-marking walk at the detached block — there is no live ancestor
// above it, so propagating further would corrupt an unrelated tree if the
// detached block were ever Insert-ed elsewhere later.
func TestRemove_DetachedSubtreeStopsMarkDirtyWalk(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"web\" {\n  ami = \"ami-1\"\n  instance_type = \"t3.micro\"\n}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	res := f.Body.FindBlock("resource")
	require.NotNil(t, res)
	require.True(t, f.Body.Remove(res))
	require.Nil(t, res.parentBody, "Remove must clear parentBody on the detached block")

	// Mutate the detached subtree's body. markDirty walks:
	// res.Body.parentBlock = res (raw cleared); res.parentBody = nil → return.
	// No crash, no nil-deref, the walk terminates cleanly.
	it := res.Body.FindAttribute("instance_type")
	require.NotNil(t, it)
	require.True(t, res.Body.Move(it, 0))
	assert.Nil(t, res.raw, "detached block's own raw is still invalidated")
}

// TestMove_ThreeLevelDeepInvalidatesAllAncestors pins that markDirty walks
// to the file root through every Block on the path. A mutation three levels
// deep — module → resource → lifecycle — must propagate through both
// ancestor blocks. This exercises the loop iteration path in markDirty
// (ops.go markDirty walk) that two-level tests do not reach.
func TestMove_ThreeLevelDeepInvalidatesAllAncestors(t *testing.T) {
	t.Parallel()

	content := "module \"outer\" {\n" +
		"  source = \"./m\"\n" +
		"  resource_block {\n" +
		"    ami = \"ami-1\"\n" +
		"    lifecycle {\n" +
		"      create_before_destroy = true\n" +
		"      prevent_destroy       = false\n" +
		"    }\n" +
		"  }\n" +
		"}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	outer := f.Body.FindBlock("module")
	require.NotNil(t, outer)
	inner := outer.Body.FindBlock("resource_block")
	require.NotNil(t, inner)
	lifecycle := inner.Body.FindBlock("lifecycle")
	require.NotNil(t, lifecycle)

	// Mutation at the deepest body must invalidate raw on both outer and
	// inner blocks — otherwise Serialize at file root would write outer.raw
	// verbatim and the mutation would be invisible.
	prevent := lifecycle.Body.FindAttribute("prevent_destroy")
	require.NotNil(t, prevent)
	require.True(t, lifecycle.Body.Move(prevent, 0))

	assert.Nil(t, outer.raw, "outermost ancestor raw must be invalidated by markDirty walk")
	assert.Nil(t, inner.raw, "intermediate ancestor raw must be invalidated by markDirty walk")

	out := f.Bytes()
	f2, err := Build(out, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	outer2 := f2.Body.FindBlock("module")
	require.NotNil(t, outer2)
	inner2 := outer2.Body.FindBlock("resource_block")
	require.NotNil(t, inner2)
	lifecycle2 := inner2.Body.FindBlock("lifecycle")
	require.NotNil(t, lifecycle2)

	assert.Equal(
		t,
		[]string{"A:prevent_destroy", "A:create_before_destroy"},
		itemKinds(lifecycle2.Body.Items),
		"deep-level mutation must round-trip through every ancestor",
	)
}

// TestMove_InsideBlockBody pins that the same Move operation works on a
// nested block body — the lifecycle-at-end fix pattern used by the
// lifecycle-ordering rule. Body.Move is type-agnostic; this is just
// confidence that block-body bodies have the same semantics as the
// top-level body.
func TestMove_InsideBlockBody(t *testing.T) {
	t.Parallel()

	content := "resource \"aws_instance\" \"web\" {\n" +
		"  ami           = \"ami-1\"\n" +
		"  instance_type = \"t3.micro\"\n" +
		"  lifecycle {\n" +
		"    create_before_destroy = true\n" +
		"  }\n" +
		"  tags = {\n" +
		"    Name = \"web\"\n" +
		"  }\n" +
		"}\n"
	f, err := Build([]byte(content), "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	res := f.Body.FindBlock("resource")
	require.NotNil(t, res)

	lifecycle := res.Body.FindBlock("lifecycle")
	require.NotNil(t, lifecycle)
	// Move lifecycle to the last position inside the resource body.
	require.True(t, res.Body.Move(lifecycle, len(res.Body.Items)-1))

	// Round-trip via re-Build to confirm the mutated CST parses and
	// preserves the same structural sequence.
	out := f.Bytes()
	f2, err := Build(out, "x.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	res2 := f2.Body.FindBlock("resource")
	require.NotNil(t, res2)
	assert.Equal(t, itemKinds(res.Body.Items), itemKinds(res2.Body.Items))

	// lifecycle is the last block in the resource body.
	blocks := res2.Body.FindBlocksByType("lifecycle")
	require.Len(t, blocks, 1)
	for i := len(res2.Body.Items) - 1; i >= 0; i-- {
		if _, ok := res2.Body.Items[i].(*Block); ok {
			assert.Equal(t, blocks[0], res2.Body.Items[i],
				"lifecycle should be the last block in the body")
			break
		}
	}
}
