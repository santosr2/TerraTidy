package style

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

func FuzzStyleCheck(f *testing.F) {
	// Use benchmark configs as seeds (comprehensive and realistic)
	f.Add([]byte(mediumConfig))  // provider, count, lifecycle, tags
	f.Add([]byte(complexConfig)) // for_each, depends_on, multiple providers

	// Additional basic seeds
	f.Add([]byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  tags = {
    Name = "test"
  }
}
`))
	f.Add([]byte(`variable "name" {
  type    = string
  default = "hello"
}
`))
	f.Add([]byte(`output "id" {
  value       = aws_s3_bucket.example.id
  description = "The bucket ID"
}
`))
	f.Add([]byte(`data "aws_caller_identity" "current" {}
`))

	// Edge cases (invalid/malformed HCL)
	f.Add([]byte(``))              // Empty
	f.Add([]byte(`{`))             // Invalid HCL
	f.Add([]byte(`}`))             // Invalid HCL
	f.Add([]byte(`= value`))       // Invalid HCL
	f.Add([]byte(`resource {}`))   // Missing labels
	f.Add([]byte(`"just string"`)) // Just a string

	// Deep nesting
	f.Add([]byte(`resource "test" "nested" {
  block1 {
    block2 {
      block3 {
        attr = "deep"
      }
    }
  }
}
`))

	// Unicode
	f.Add([]byte(`variable "日本語" {
  default = "テスト"
}
`))

	// Comments
	f.Add([]byte(`# Comment
resource "test" "example" {} // inline
/* block */
`))

	// Heredoc
	f.Add([]byte(`resource "test" "heredoc" {
  content = <<-EOF
    line1
    line2
  EOF
}
`))

	// Complex expressions
	f.Add([]byte(`locals {
  complex = merge(
    { for k, v in var.map : k => v if v != null },
    try(var.other, {})
  )
}
`))

	// Create engine once outside fuzz function for performance
	engine := New(&Config{Rules: make(map[string]RuleConfig)})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Write to temp file (style engine works on files)
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Run style checks - should not panic regardless of input
		_, _ = engine.Run(context.Background(), []string{tmpFile})
	})
}

// FuzzStyleFix exercises the fix-mode code path which involves:
// - Multi-pass loop (up to 3 passes)
// - File write-back via applyFixes
// - Re-read and re-parse after each fix
// - Diff generation
func FuzzStyleFix(f *testing.F) {
	// Use same seeds as FuzzStyleCheck
	f.Add([]byte(mediumConfig))
	f.Add([]byte(complexConfig))

	// Seeds with known style issues that trigger fixes
	f.Add([]byte(`resource "aws_instance" "web" {
tags = {
Name = "test"
}
ami = "ami-123"
}
`)) // tags not at end

	f.Add([]byte(`resource "aws_instance" "web" {
ami = "ami-123"


instance_type = "t2.micro"
}
`)) // consecutive blank lines

	f.Add([]byte(`


resource "test" "example" {}


`)) // leading/trailing blank lines

	f.Add([]byte(`variable "a" {}
variable "b" {}
`)) // no blank line between blocks

	// Edge cases
	f.Add([]byte(``))
	f.Add([]byte(`{`))
	f.Add([]byte(`resource {}`))

	// Create engine with Fix and Diff enabled
	engine := New(&Config{
		Fix:   true,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Write to temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Run style checks with fix mode - should not panic
		_, _ = engine.Run(context.Background(), []string{tmpFile})
	})
}

// FuzzApplyFixesSorting cross-checks applyFixes against an independent
// reference that filters and applies the same edits with an ascending-Start
// splice plus explicit offset shift tracking. After identical bounds-check,
// whole-file-exclusivity, and overlap filtering, the engine's descending-Start
// splice must produce byte-identical content and the same multiset of applied
// rule names. A divergence means either the sort order matters for retained
// edits (a real bug, since the overlap filter is supposed to guarantee the
// remaining set commutes) or the filter itself differs from the spec.
//
// The seed corpus encodes the 10 documented edge cases from the
// byte-range-textedits plan (lines 304-313): all-disjoint, same-range
// conflict, partial overlap, whole-file alongside narrow, stacked
// zero-width insertions, same-offset zero-width conflict, adjacent touching
// ranges, empty content, end-of-file insertion, and single full-content
// replacement. Random fuzz inputs mutate these and exercise content + edit
// combinations the seeds do not enumerate.
func FuzzApplyFixesSorting(f *testing.F) {
	seed := func(content []byte, edits ...sdk.TextEdit) {
		f.Add(content, encodeFuzzEdits(edits))
	}

	// 1. All-disjoint: three independent narrow replacements.
	seed(
		[]byte("hello world foo"),
		sdk.TextEdit{Start: 0, End: 5, Replacement: []byte("HELLO")},
		sdk.TextEdit{Start: 6, End: 11, Replacement: []byte("WORLD")},
		sdk.TextEdit{Start: 12, End: 15, Replacement: []byte("FOO")},
	)
	// 2. All-overlapping at the same range — second deferred.
	seed(
		[]byte("abc"),
		sdk.TextEdit{Start: 0, End: 3, Replacement: []byte("X")},
		sdk.TextEdit{Start: 0, End: 3, Replacement: []byte("Y")},
	)
	// 3. Partial overlap — half-open ranges share positions; second deferred.
	seed(
		[]byte("abcdef"),
		sdk.TextEdit{Start: 0, End: 3, Replacement: []byte("Z")},
		sdk.TextEdit{Start: 1, End: 4, Replacement: []byte("W")},
	)
	// 4. Whole-file alongside narrow — whole-file wins; narrow deferred.
	seed(
		[]byte("foo bar"),
		sdk.TextEdit{Start: 0, End: 7, Replacement: []byte("BAZ")},
		sdk.TextEdit{Start: 4, End: 7, Replacement: []byte("QUX")},
	)
	// 5. Stacked zero-width insertions at distinct offsets (all apply).
	seed(
		[]byte("ab"),
		sdk.TextEdit{Start: 0, End: 0, Replacement: []byte("X")},
		sdk.TextEdit{Start: 1, End: 1, Replacement: []byte("Y")},
		sdk.TextEdit{Start: 2, End: 2, Replacement: []byte("Z")},
	)
	// 6. Same-offset zero-width insertions — same-Start short-circuit; second deferred.
	seed(
		[]byte("ab"),
		sdk.TextEdit{Start: 1, End: 1, Replacement: []byte("X")},
		sdk.TextEdit{Start: 1, End: 1, Replacement: []byte("Y")},
	)
	// 7. Adjacent touching ranges — a.End == b.Start; both apply.
	seed(
		[]byte("abcdef"),
		sdk.TextEdit{Start: 0, End: 3, Replacement: []byte("X")},
		sdk.TextEdit{Start: 3, End: 6, Replacement: []byte("Y")},
	)
	// 8. Empty content with no edits.
	seed([]byte(""))
	// 9. Insertion at end of file (Start == End == len(content)).
	seed(
		[]byte("abc"),
		sdk.TextEdit{Start: 3, End: 3, Replacement: []byte("\n")},
	)
	// 10. Single edit replacing entire content (whole-file exclusivity sole edit).
	seed(
		[]byte("abc"),
		sdk.TextEdit{Start: 0, End: 3, Replacement: []byte("xyz")},
	)
	// 11. Empty file + zero-width insertion at offset 0 — this also satisfies
	// the whole-file exclusivity predicate (Start == 0 && End == len(content)
	// with len == 0), so the edit lands as a whole-file replacement on an
	// empty original. Locks the empty-file path that's otherwise only
	// reachable via fuzzer mutation of seed #8.
	seed(
		[]byte(""),
		sdk.TextEdit{Start: 0, End: 0, Replacement: []byte("hello")},
	)

	f.Fuzz(func(t *testing.T, content, editsRaw []byte) {
		// Keep iterations cheap: huge inputs would dwarf the algorithmic
		// coverage we want without finding additional bugs.
		if len(content) > 4096 {
			t.Skip()
		}
		edits := decodeFuzzEdits(editsRaw)
		if len(edits) > 32 {
			t.Skip()
		}
		runFuzzApplyFixesCase(t, content, edits)
	})
}

// encodeFuzzEdits serializes a slice of TextEdits into a self-delimiting byte
// stream used as the fuzz target's second argument. Each record is laid out
// as: Start uint16 (BE), End uint16 (BE), len(Replacement) uint8, then the
// replacement bytes. Edits whose fields do not fit (negative offsets, offsets
// > 0xFFFF, replacements > 0xFF bytes) are skipped — the seed corpus stays
// within those limits.
func encodeFuzzEdits(edits []sdk.TextEdit) []byte {
	var buf bytes.Buffer
	for _, e := range edits {
		if e.Start < 0 || e.Start > 0xFFFF {
			continue
		}
		if e.End < 0 || e.End > 0xFFFF {
			continue
		}
		if len(e.Replacement) > 0xFF {
			continue
		}
		_ = binary.Write(&buf, binary.BigEndian, uint16(e.Start))
		_ = binary.Write(&buf, binary.BigEndian, uint16(e.End))
		_ = buf.WriteByte(byte(len(e.Replacement)))
		buf.Write(e.Replacement)
	}
	return buf.Bytes()
}

// decodeFuzzEdits is the inverse of encodeFuzzEdits, except it accepts any
// trailing junk: parsing stops at the first record header that runs past the
// end of the input. This makes the encoding robust to fuzzer mutations that
// truncate or extend the byte stream.
func decodeFuzzEdits(raw []byte) []sdk.TextEdit {
	var edits []sdk.TextEdit
	for len(raw) >= 5 {
		start := int(binary.BigEndian.Uint16(raw[0:2]))
		end := int(binary.BigEndian.Uint16(raw[2:4]))
		replLen := int(raw[4])
		raw = raw[5:]
		if len(raw) < replLen {
			return edits
		}
		repl := append([]byte(nil), raw[:replLen]...)
		raw = raw[replLen:]
		edits = append(edits, sdk.TextEdit{Start: start, End: end, Replacement: repl})
	}
	return edits
}

// fuzzEditStub is a one-edit Fixer whose Fix always returns its preconfigured
// TextEdit. It deliberately coexists with stubNarrowEditRule (style_test.go)
// rather than replacing it: stubNarrowEditRule takes start/end/replacement as
// three separate fields, which reads more naturally inside the table-driven
// unit tests; fuzzEditStub holds a whole sdk.TextEdit value because the fuzz
// harness has already constructed one from the decoded byte stream and there
// is nothing to gain from re-splitting it. The two types are otherwise
// behaviorally identical.
type fuzzEditStub struct {
	name string
	edit sdk.TextEdit
}

func (s *fuzzEditStub) Name() string        { return s.name }
func (s *fuzzEditStub) Description() string { return "fuzz stub: emits one configured edit" }

func (s *fuzzEditStub) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return []sdk.Finding{{
		Rule:     s.name,
		File:     ctx.File,
		Severity: sdk.SeverityWarning,
		Fixable:  true,
	}}, nil
}

func (s *fuzzEditStub) Fix(_ *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	return &sdk.FixResult{Edits: []sdk.TextEdit{s.edit}}, nil
}

// runFuzzApplyFixesCase wires up the engine, stubs, and findings, runs
// applyFixes, then compares the result against referenceApplyFixes. Both
// must agree on (a) error-vs-success, (b) the multiset of applied rule names,
// (c) the written content. The single-write-per-pass invariant is also
// enforced when any edit lands.
func runFuzzApplyFixesCase(t *testing.T, content []byte, edits []sdk.TextEdit) {
	t.Helper()

	stubs := make([]sdk.Rule, len(edits))
	for i, e := range edits {
		stubs[i] = &fuzzEditStub{
			name: fmt.Sprintf("test.fuzz-%d", i),
			edit: e,
		}
	}
	engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)}, stubs...)

	var captured []byte
	var writeCalls int
	engine.writeFn = func(_ string, data []byte, _ os.FileMode) error {
		writeCalls++
		captured = append([]byte(nil), data...)
		return nil
	}

	// applyFixes calls os.Chmod after writeFn, so the file must exist on disk.
	tmpFile := filepath.Join(t.TempDir(), "fuzz.tf")
	if err := os.WriteFile(tmpFile, content, 0o600); err != nil {
		t.Fatalf("setup write: %v", err)
	}

	// hcl.File only needs Bytes; the stubs ignore Body so we can skip parsing
	// and feed arbitrary fuzzer bytes directly into applyFixes.
	file := &hcl.File{Bytes: content}
	ruleCtx := &sdk.Context{
		Context: context.Background(),
		Options: make(map[string]any),
		WorkDir: ".",
		File:    tmpFile,
	}
	findings := make([]sdk.Finding, len(edits))
	for i := range edits {
		findings[i] = sdk.Finding{
			Rule:     stubs[i].Name(),
			File:     tmpFile,
			Fixable:  true,
			Severity: sdk.SeverityWarning,
		}
	}

	applied, engineErr := engine.applyFixes(ruleCtx, file, findings, 0o644)
	refContent, refApplied, refErr := referenceApplyFixes(content, stubs, edits)

	if (engineErr == nil) != (refErr == nil) {
		t.Fatalf("error disagreement: engine err = %v; reference err = %v", engineErr, refErr)
	}
	if engineErr != nil {
		// Both errored: error message wording is engine-specific. Don't
		// compare strings, just confirm both rejected the input.
		return
	}

	if !sameRuleMultiset(applied, refApplied) {
		t.Errorf("applied rules differ: engine=%v reference=%v", applied, refApplied)
	}

	if len(applied) == 0 {
		if writeCalls != 0 {
			t.Errorf("expected zero writes when no edits applied, got %d", writeCalls)
		}
		return
	}
	if writeCalls != 1 {
		t.Errorf("expected exactly one write per pass, got %d", writeCalls)
	}
	if !bytes.Equal(captured, refContent) {
		t.Errorf("content differs:\n  engine:    %q\n  reference: %q\n  inputs:    content=%q edits=%+v",
			captured, refContent, content, edits)
	}
}

// referenceApplyFixes mirrors applyFixes step-by-step so the only material
// difference between the two implementations is the splice direction. The
// pipeline is:
//
//  1. Bounds-check every collected edit (must run before any splice path —
//     including the whole-file shortcut — so out-of-bounds inputs error
//     consistently with the engine).
//  2. Whole-file exclusivity: linear scan in source order; return the first
//     match alone with its rule name. Identical to applyFixes lines
//     ~492-497.
//  3. Overlap filter: walk in source order, retain edits that do not
//     conflict with any already-kept edit (shares editsConflict with the
//     engine, by design — the fuzz target's job is to catch sort/splice
//     bugs, not conflict-detection bugs).
//  4. Splice retained edits in ascending-Start order, tracking the
//     cumulative shift introduced by each replacement's length delta. The
//     engine instead sorts descending and splices on original offsets;
//     ascending-with-shift and descending-without-shift are equivalent
//     formulations of the same splice problem for non-overlapping ranges,
//     so they must produce byte-identical content.
//
// The mismatch between step 4 (different splice direction) is the only
// source of differential information; steps 1-3 are intentionally identical
// to avoid false-positive divergences on conflict-detection edge cases.
func referenceApplyFixes(content []byte, rules []sdk.Rule, edits []sdk.TextEdit) ([]byte, []string, error) {
	type pair struct {
		rule string
		edit sdk.TextEdit
	}
	collected := make([]pair, len(edits))
	for i, e := range edits {
		collected[i] = pair{rule: rules[i].Name(), edit: e}
	}

	for _, p := range collected {
		if p.edit.Start < 0 {
			return nil, nil, errors.New("start negative")
		}
		if p.edit.End < p.edit.Start {
			return nil, nil, errors.New("end precedes start")
		}
		if p.edit.End > len(content) {
			return nil, nil, errors.New("end exceeds content length")
		}
	}

	if len(collected) == 0 {
		return nil, nil, nil
	}

	for _, p := range collected {
		if p.edit.Start == 0 && p.edit.End == len(content) {
			return append([]byte(nil), p.edit.Replacement...), []string{p.rule}, nil
		}
	}

	var retained []pair
	for _, p := range collected {
		conflict := false
		for _, k := range retained {
			if editsConflict(p.edit, k.edit) {
				conflict = true
				break
			}
		}
		if !conflict {
			retained = append(retained, p)
		}
	}

	sort.SliceStable(retained, func(i, j int) bool {
		return retained[i].edit.Start < retained[j].edit.Start
	})

	newContent := append([]byte(nil), content...)
	shift := 0
	for _, p := range retained {
		start := p.edit.Start + shift
		end := p.edit.End + shift
		spliced := make([]byte, 0, len(newContent)-(end-start)+len(p.edit.Replacement))
		spliced = append(spliced, newContent[:start]...)
		spliced = append(spliced, p.edit.Replacement...)
		spliced = append(spliced, newContent[end:]...)
		newContent = spliced
		shift += len(p.edit.Replacement) - (p.edit.End - p.edit.Start)
	}

	var applied []string
	seen := make(map[string]struct{}, len(retained))
	for _, p := range retained {
		if _, dup := seen[p.rule]; dup {
			continue
		}
		seen[p.rule] = struct{}{}
		applied = append(applied, p.rule)
	}
	return newContent, applied, nil
}

// sameRuleMultiset reports whether a and b contain the same rule names with
// the same multiplicities, regardless of order. applyFixes returns names in
// retained-descending-Start order; referenceApplyFixes returns names in
// retained-source order. The contract is unordered, so a multiset comparison
// is the right relation.
func sameRuleMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}
