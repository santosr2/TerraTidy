package cst

// Body mutation operations: Move, MoveBefore, MoveAfter, Insert, Remove,
// and the four Find* lookups. Each Find* lookup walks Body.Items in source
// order and matches by pointer-equality-relevant fields (Attribute.Name,
// Block.Type, Block.Labels). Mutations operate on the Body.Items slice
// in place; items keep their own raw bytes, so reshuffling round-trips
// through File.Bytes byte-for-byte without touching item contents.
//
// Concurrency: these methods are not safe to call concurrently with any
// reader or writer of the same Body. moveSlice and insertSlice rewrite
// b.Items in place via append, so a reader holding a stale slice header
// may observe a truncated or corrupted window during a Move. All existing
// call sites — single-pass rule Fix() invocations — are single-threaded,
// so this is documentation, not a regression risk; the invariant must be
// preserved if rule execution ever fans out across goroutines per file.

// Move repositions item to a new index in body.Items. Returns false if item
// is not in this body or newIndex is out of range [0, len(Items)).
//
// The item keeps its leading comments — they live on the item itself
// (Attribute.LeadingComments / Block.LeadingComments, encoded into the
// item's raw bytes on items unmodified since Build), so they travel without
// any extra work here. StandaloneComment items in the same body do NOT
// reshuffle: their slice position changes only as a side-effect of the
// moved item passing over them, not because Move targets them. That is the
// fix mechanism for the floating section-header bug in
// `style.terraform-block-first` — when the terraform block moves to position
// 0, a `### SNS Notifications` StandaloneComment between resources stays a
// StandaloneComment with the same content.
//
// A Move to the item's current index is a successful no-op.
func (b *Body) Move(item BodyItem, newIndex int) bool {
	src := indexOf(b.Items, item)
	if src < 0 {
		return false
	}
	if newIndex < 0 || newIndex >= len(b.Items) {
		return false
	}
	if src == newIndex {
		return true
	}
	b.Items = moveSlice(b.Items, src, newIndex)
	b.markDirty()
	return true
}

// MoveBefore positions src immediately before dst. Returns false if either
// item is not in this body or src and dst are the same item.
//
// The new index of src is dst's index in the post-removal slice: dstIdx-1
// when srcIdx < dstIdx (because dst shifts left by one when src is removed
// from before it), dstIdx otherwise.
func (b *Body) MoveBefore(src, dst BodyItem) bool {
	srcIdx := indexOf(b.Items, src)
	dstIdx := indexOf(b.Items, dst)
	if srcIdx < 0 || dstIdx < 0 || srcIdx == dstIdx {
		return false
	}
	target := dstIdx
	if srcIdx < dstIdx {
		target = dstIdx - 1
	}
	return b.Move(src, target)
}

// MoveAfter positions src immediately after dst. Returns false if either
// item is not in this body or src and dst are the same item.
//
// The new index of src is one past dst's index in the post-removal slice.
// When src is originally before dst, removing src shifts dst left by one
// so the target collapses to dstIdx (which is dst's new index + 1). When
// src is after dst, removing src leaves dst's index unchanged, so the
// target is dstIdx + 1.
func (b *Body) MoveAfter(src, dst BodyItem) bool {
	srcIdx := indexOf(b.Items, src)
	dstIdx := indexOf(b.Items, dst)
	if srcIdx < 0 || dstIdx < 0 || srcIdx == dstIdx {
		return false
	}
	target := dstIdx + 1
	if srcIdx < dstIdx {
		target = dstIdx
	}
	return b.Move(src, target)
}

// Insert puts item at index in body.Items. Items at >= index shift right
// by one. Returns false if item is nil or index is out of [0, len(Items)].
//
// index == 0 prepends; index == len(Items) appends. Insert does not check
// whether item is already in body — callers should use Move to relocate an
// existing item rather than Insert-after-Remove, which is two passes for
// no benefit.
//
// When item is a *Block, Insert wires its parentBody pointer so future
// mutations on the inserted block's body propagate dirty-marking back up
// the tree.
func (b *Body) Insert(item BodyItem, index int) bool {
	if item == nil || index < 0 || index > len(b.Items) {
		return false
	}
	b.Items = insertSlice(b.Items, index, item)
	if blk, ok := item.(*Block); ok {
		blk.parentBody = b
		if blk.Body != nil {
			blk.Body.parentBlock = blk
		}
	}
	b.markDirty()
	return true
}

// Remove deletes item from body.Items. Returns false if item is nil or not
// in this body. Adjacent BlankLines and StandaloneComments are NOT collapsed;
// they stay at their original positions. The item's leading comments go with
// it because they live on the item (in its raw bytes or LeadingComments
// field).
//
// Future: a BlankLinePolicy hook could let callers opt into collapsing
// double-blank-line runs created by Remove. No existing structural rule
// needs it, so the hook is deliberately not added yet.
//
// When item is a *Block, Remove clears its parentBody pointer so a stale
// back-link doesn't trip up dirty-marking if the removed block is later
// re-inserted into a different body.
func (b *Body) Remove(item BodyItem) bool {
	idx := indexOf(b.Items, item)
	if idx < 0 {
		return false
	}
	b.Items = append(b.Items[:idx], b.Items[idx+1:]...)
	if blk, ok := item.(*Block); ok {
		blk.parentBody = nil
	}
	b.markDirty()
	return true
}

// FindAttribute returns the first Attribute in body.Items whose Name equals
// name, or nil. Iteration is in source order — Body.Items preserves it.
func (b *Body) FindAttribute(name string) *Attribute {
	for _, item := range b.Items {
		if a, ok := item.(*Attribute); ok && a.Name == name {
			return a
		}
	}
	return nil
}

// FindBlock returns the first Block in body.Items whose Type equals
// blockType, or nil. Use FindBlockByTypeAndLabels when the labels matter.
func (b *Body) FindBlock(blockType string) *Block {
	for _, item := range b.Items {
		if blk, ok := item.(*Block); ok && blk.Type == blockType {
			return blk
		}
	}
	return nil
}

// FindBlocksByType returns every Block in body.Items whose Type equals
// blockType, in source order. Returns nil (not an empty slice) when no
// matches exist — a no-allocation fast path for the common case.
func (b *Body) FindBlocksByType(blockType string) []*Block {
	var out []*Block
	for _, item := range b.Items {
		if blk, ok := item.(*Block); ok && blk.Type == blockType {
			out = append(out, blk)
		}
	}
	return out
}

// FindBlockByTypeAndLabels returns the first Block whose Type equals
// blockType and whose Labels match the given labels position-by-position,
// or nil. The match is exact: a block with a different label count never
// matches — labels is the full tuple, not a prefix.
//
// A nil or empty labels slice matches only blocks with zero labels.
func (b *Body) FindBlockByTypeAndLabels(blockType string, labels []string) *Block {
	for _, item := range b.Items {
		blk, ok := item.(*Block)
		if !ok || blk.Type != blockType || len(blk.Labels) != len(labels) {
			continue
		}
		match := true
		for i, lbl := range labels {
			if blk.Labels[i].Text != lbl {
				match = false
				break
			}
		}
		if match {
			return blk
		}
	}
	return nil
}

// indexOf returns the position of item in items by pointer-equality, or -1
// when item is nil or absent. Go interface comparison compares both dynamic
// type and value; every BodyItem implementation is a pointer type, so the
// comparison reduces to pointer identity. A duplicate item in the slice
// (which Insert can produce — see its doc) would only be findable at the
// first position, not the later one; that asymmetry is intentional and
// makes Move/Remove deterministic in the face of caller misuse.
func indexOf(items []BodyItem, item BodyItem) int {
	if item == nil {
		return -1
	}
	for i, it := range items {
		if it == item {
			return i
		}
	}
	return -1
}

// moveSlice repositions items[src] to dst. Both indices must already be
// validated in [0, len(items)); the caller (Move) does so. Implemented as
// remove-then-insert so the same arithmetic works whether src < dst or
// src > dst — splitting the cases adds branches without saving work, and
// the slice ops here are already amortized O(N).
func moveSlice(items []BodyItem, src, dst int) []BodyItem {
	item := items[src]
	items = append(items[:src], items[src+1:]...)
	return insertSlice(items, dst, item)
}

// insertSlice puts item at index in items, shifting subsequent items right
// by one. index must be in [0, len(items)] — the caller (Insert / moveSlice)
// validates this.
//
// The "append nil then copy backwards" idiom is safe under the Go
// language spec: "The source and destination [of copy] may overlap." The
// builtin handles the direction correctly so source elements are read
// before they are overwritten in the shift-right pass.
func insertSlice(items []BodyItem, index int, item BodyItem) []BodyItem {
	items = append(items, nil)
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

// markDirty invalidates the raw bytes of every Block on the path from this
// Body up to the file root. Without this walk, a mutation deep inside a
// nested body would be invisible: Serialize sees the unchanged Block.raw
// of every ancestor, writes it verbatim, and never descends to discover
// the mutation. Each invalidated ancestor's RawBytes() now returns nil,
// forcing Serialize to take its writeRegenerated path, which iterates the
// (mutated) Body.
//
// Top-level Body has parentBlock == nil, so markDirty is a no-op there —
// the top-level Body.writeTo iterates items directly and reflects any
// mutation without needing invalidation.
//
// The walk terminates on the first ancestor with no parent — either a
// Block whose parentBody is nil (a detached subtree, possibly mid-Insert
// elsewhere) or the file root.
func (b *Body) markDirty() {
	blk := b.parentBlock
	for blk != nil {
		blk.raw = nil
		if blk.parentBody == nil {
			return
		}
		blk = blk.parentBody.parentBlock
	}
}
