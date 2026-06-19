package cst

// Policy controls how comments separated from the next attribute or block by
// blank lines are attached when Build constructs a Body.
//
// HCL gives no syntactic signal for "leading comment of X". Heuristics decide.
// The legacy line-based helpers in internal/engines/style/rules/helpers.go
// contain three separate heuristics for this question; the CST collapses them
// into one toggle, applied per Body at Build time.
type Policy struct {
	// StrictAdjacency, when true, requires zero blank lines between a comment
	// and the next attribute or block for the comment to attach as a leading
	// comment. A comment separated from the next item by one or more blank
	// lines becomes a StandaloneComment that survives reorder in place.
	//
	// When false (passthrough), blank lines are tolerated and the comment
	// still attaches as a leading comment. This matches the pre-CST
	// block-body behavior of the legacy `collectLeadingComments` helper.
	StrictAdjacency bool
}

// DefaultTopLevelPolicy returns the policy for top-level Body items. Strict
// adjacency keeps floating section-header comments (e.g. `### SNS
// Notifications`) as StandaloneComment. They survive reorder in place rather
// than being silently dropped or attached to the wrong block — the fix
// mechanism for the floating-header bug in `style.terraform-block-first`.
func DefaultTopLevelPolicy() Policy { return Policy{StrictAdjacency: true} }

// DefaultBlockBodyPolicy returns the policy for nested-block Body items.
// Passthrough preserves the pre-CST behavior where blank-line-separated
// comments inside a block body still attach as leading comments — the user
// intent inside a block body is overwhelmingly "this comment belongs to the
// next attribute" even with cosmetic blank lines between.
func DefaultBlockBodyPolicy() Policy { return Policy{} }
