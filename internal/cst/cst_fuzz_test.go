package cst

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// FuzzCSTRoundTrip exercises the Build → Bytes syntactic-equality
// invariant under random VALID HCL inputs.
//
// Contract: for input that hclsyntax.ParseConfig accepts cleanly,
// re-parsing the Bytes() output of Build must yield a top-level body whose
// Attribute.Names and Block.Type+Labels match the pre-serialization set.
// Byte-for-byte identity is the STRONGER invariant tested in
// serialize_test.go on curated fixtures; here we use syntactic equality
// because Build legally normalizes some shapes (e.g., trailing-only
// whitespace files collapse to empty), and the round-trip we care about
// for rule Fix safety is structural identity, not byte identity.
//
// Parse-failed inputs are filtered upstream because this target is scoped
// to "random valid HCL". Build's robustness on malformed inputs is tracked
// separately — there is a known panic on partial-tree expressions with
// inverted byte ranges (e.g. `A=A(`).
func FuzzCSTRoundTrip(f *testing.F) {
	seedFuzzCorpus(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		if !isValidHCL(data) {
			return
		}
		file, err := Build(data, "fuzz.tf", DefaultTopLevelPolicy())
		if file == nil {
			t.Fatal("Build returned nil File")
		}
		if err != nil {
			// Defense-in-depth: isValidHCL filtered parse errors, but
			// Build may still return a wrapped diag — exempt here.
			return
		}
		out := file.Bytes()

		expected := collectTopLevelNames(file.Body)

		reparsed, diags := hclsyntax.ParseConfig(out, "fuzz.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("post-serialize re-parse failed: %v\n--- output ---\n%s", diags, out)
		}
		body, ok := reparsed.Body.(*hclsyntax.Body)
		if !ok {
			t.Fatalf("re-parsed Body is %T, want *hclsyntax.Body", reparsed.Body)
		}
		actual := collectHCLSyntaxTopLevelNames(body)

		if !sameNameSet(expected, actual) {
			t.Fatalf("syntactic equality violated\nexpected: %v\nactual:   %v\n--- output ---\n%s",
				sortedKeys(expected), sortedKeys(actual), out)
		}
	})
}

// isValidHCL reports whether data is well-formed UTF-8 HCL that hclsyntax
// parses without diagnostics. Used to filter random fuzz inputs down to
// "valid HCL" before exercising Build / mutate, matching the spec scope of
// FuzzCSTRoundTrip / FuzzCSTMutateRoundTrip.
//
// The UTF-8 check is load-bearing: hclsyntax accepts isolated continuation
// bytes (e.g. a lone 0xC9 inside an expression) on the FIRST pass but
// changes its lexer state on the SECOND pass when content is appended
// after the malformed bytes — splicing an inserted attribute trips
// "Missing newline after argument" because the lexer drifts on the
// invalid byte. The HCL spec requires UTF-8 source, so this filter pins
// the contract rather than papering over a hclsyntax leniency.
func isValidHCL(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	_, diags := hclsyntax.ParseConfig(data, "fuzz.tf", hcl.InitialPos)
	return !diags.HasErrors()
}

// sameNameSet reports whether a and b contain the same set of keys.
func sameNameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// sortedKeys returns the keys of m sorted for stable diff output.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// FuzzCSTMutateRoundTrip exercises the structural mutation invariant: a
// Move/Insert/Remove on the top-level body produces output that (a) parses
// cleanly and (b) preserves the set of original top-level identifiers minus
// any item explicitly removed.
//
// Contract:
//   - (a) hclsyntax.ParseConfig succeeds on the post-mutation Bytes().
//   - (b) Every top-level Attribute.Name and Block.Type+Labels present
//     before the mutation is still present in the re-parsed AST — except
//     the one the Remove operation targeted, which is excluded from the
//     expected set.
//
// Scope: only top-level Body mutations are exercised. Nested-body
// mutations are covered by unit tests in ops_test.go that pin byte-exact
// round-trip identity through the writeRegenerated path. The fuzz also
// requires the input to end with `\n` so every CST item has a
// newline-terminated raw — without that, a reshuffle could juxtapose a
// no-newline raw with the following item and produce a deliberately
// broken concatenation, which is a serialize-layer bug separate from
// the structural mutation invariant we are testing here.
func FuzzCSTMutateRoundTrip(f *testing.F) {
	seedFuzzCorpus(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Target scope is "random valid HCL"; filter out parse failures
		// so Build doesn't trip the known panic on malformed partial-tree
		// byte ranges.
		if !isValidHCL(data) {
			return
		}
		// Precondition: input must end with `\n` so every Build-produced
		// item.raw ends with a line terminator. See function doc for the
		// rationale.
		if len(data) == 0 || data[len(data)-1] != '\n' {
			return
		}

		file, err := Build(data, "fuzz.tf", DefaultTopLevelPolicy())
		if err != nil || file == nil || file.Body == nil || len(file.Body.Items) == 0 {
			return
		}
		body := file.Body

		expected := collectTopLevelNames(body)

		seed := fuzzSeed(data)
		op := int(seed % 3)

		switch op {
		case 0: // Move — single-item bodies exercise the same-index no-op path
			src := int(seed % uint64(len(body.Items)))
			dst := int((seed / 7) % uint64(len(body.Items)))
			if !body.Move(body.Items[src], dst) {
				t.Fatalf("Move(%d → %d) on %d-item body returned false", src, dst, len(body.Items))
			}
		case 1: // Insert
			inserted := &Attribute{
				Name:            "__fuzz_inserted_attr",
				NameBytes:       []byte("__fuzz_inserted_attr"),
				ExpressionBytes: []byte("0"),
			}
			pos := int(seed % uint64(len(body.Items)+1))
			if !body.Insert(inserted, pos) {
				t.Fatalf("Insert at %d on %d-item body returned false", pos, len(body.Items))
			}
			// Positive assertion on the inserted identifier — without
			// adding it to expected, a silent drop of the inserted item
			// during serialize would slip past the subset check below.
			expected["attr:__fuzz_inserted_attr"] = true
		case 2: // Remove
			idx := int(seed % uint64(len(body.Items)))
			item := body.Items[idx]
			removeTopLevelName(expected, item)
			if !body.Remove(item) {
				t.Fatalf("Remove of items[%d] on %d-item body returned false", idx, len(body.Items))
			}
		}

		out := file.Bytes()
		reparsed, diags := hclsyntax.ParseConfig(out, "fuzz.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("(a) post-mutation re-parse failed: %v\n--- output ---\n%s", diags, out)
		}

		body2, ok := reparsed.Body.(*hclsyntax.Body)
		if !ok {
			t.Fatalf("re-parsed Body is %T, want *hclsyntax.Body", reparsed.Body)
		}
		actual := collectHCLSyntaxTopLevelNames(body2)

		for k := range expected {
			if !actual[k] {
				t.Fatalf("(b) identifier %q missing after op=%d\n--- output ---\n%s",
					k, op, out)
			}
		}
	})
}

// fuzzSeed derives a stable 64-bit value from the input bytes. Used by
// FuzzCSTMutateRoundTrip to pick a deterministic mutation per corpus entry —
// same input always exercises the same op, so a reproducer file maps to one
// failure mode rather than three.
//
// FNV-1a chosen for simplicity and avoidance of hash/maphash, which requires
// a seed and is not deterministic across runs.
func fuzzSeed(data []byte) uint64 {
	const (
		fnvOffset uint64 = 14695981039346656037
		fnvPrime  uint64 = 1099511628211
	)
	h := fnvOffset
	for _, b := range data {
		h ^= uint64(b)
		h *= fnvPrime
	}
	// Mix length so an empty input and a non-empty input that happens to
	// terminate the FNV loop on the offset basis still hash distinctly.
	h ^= uint64(len(data))
	return h
}

// collectTopLevelNames returns the set of structural identifiers in the
// top-level body, keyed so attribute and block identifiers cannot collide:
//
//	attr:<name>
//	block:<type>:<label1>:<label2>:...
//
// Body items that are not Attribute or Block (BlankLine, StandaloneComment)
// have no addressable identifier and are excluded.
func collectTopLevelNames(body *Body) map[string]bool {
	out := make(map[string]bool, len(body.Items))
	for _, item := range body.Items {
		switch v := item.(type) {
		case *Attribute:
			out["attr:"+v.Name] = true
		case *Block:
			out[blockKey(v.Type, blockLabelTexts(v.Labels))] = true
		}
	}
	return out
}

// removeTopLevelName deletes the identifier corresponding to item from m.
// No-op for BlankLine and StandaloneComment, which are not tracked by
// collectTopLevelNames.
func removeTopLevelName(m map[string]bool, item BodyItem) {
	switch v := item.(type) {
	case *Attribute:
		delete(m, "attr:"+v.Name)
	case *Block:
		delete(m, blockKey(v.Type, blockLabelTexts(v.Labels)))
	}
}

// collectHCLSyntaxTopLevelNames is the hclsyntax-side mirror of
// collectTopLevelNames, used to compare against the post-mutation re-parsed
// AST.
func collectHCLSyntaxTopLevelNames(body *hclsyntax.Body) map[string]bool {
	out := make(map[string]bool, len(body.Attributes)+len(body.Blocks))
	for name := range body.Attributes {
		out["attr:"+name] = true
	}
	for _, blk := range body.Blocks {
		out[blockKey(blk.Type, blk.Labels)] = true
	}
	return out
}

func blockLabelTexts(labels []Label) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.Text
	}
	return out
}

func blockKey(blockType string, labels []string) string {
	key := "block:" + blockType
	for _, l := range labels {
		key += ":" + l
	}
	return key
}

// seedFuzzCorpus loads inline seeds covering every BodyItem variant +
// Terragrunt blocks + CRLF + nested depth ≥ 3 + comments-only + empty,
// then adds the in-tree .tf/.hcl fixtures so the fuzzer starts from
// known-good round-trip targets rather than only random bytes. Seed-corpus
// adequacy is checked by running both fuzz targets 5 consecutive 30s
// rounds locally and confirming no new crash files appear under
// testdata/fuzz/ before the change ships.
func seedFuzzCorpus(f *testing.F) {
	for _, s := range inlineFuzzSeeds {
		f.Add([]byte(s))
	}

	root, err := findRepoRoot()
	if err != nil {
		return // running outside the repo — skip fixture seeds gracefully
	}
	for _, rel := range fuzzFixtureFiles {
		content, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		f.Add(content)
	}
}

// inlineFuzzSeeds are byte-string seeds covering the shapes the
// adequacy-gate calls out: Terragrunt blocks, CRLF, nested blocks ≥ 3
// deep, comments-only, empty. Each shape is one seed so a fuzzer crash
// reproducer points at a category, not at random mutations of a giant
// monolith seed.
var inlineFuzzSeeds = []string{
	"",
	"\n",
	"\n\n\n",
	"# only comment\n",
	"# c1\n# c2\n",
	"# c1\n\n# c2\n",
	"// slash comment\n",
	"/* block comment */\n",
	"name = \"alice\"\n",
	"name = \"alice\" # inline\n",
	"# leading\nname = \"alice\"\n",
	"alpha = 1\n\nbeta = 2\n",
	"resource \"aws_s3_bucket\" \"b\" {\n}\n",
	"resource \"aws_s3_bucket\" \"b\" {\n  bucket = \"x\"\n}\n",
	"resource \"aws_s3_bucket\" \"b\" {\n  bucket = \"x\"\n  tags = {\n    Name = \"x\"\n  }\n}\n",
	"terraform {\n  required_version = \">= 1.0\"\n  required_providers {\n    aws = {\n      source  = \"hashicorp/aws\"\n      version = \"~> 5.0\"\n    }\n  }\n}\n",
	// CRLF
	"alpha = 1\r\n\r\nbeta = 2\r\n",
	"resource \"x\" \"y\" {\r\n  a = 1\r\n}\r\n",
	// Terragrunt
	"include \"root\" {\n  path = find_in_parent_folders()\n}\n\ndependency \"vpc\" {\n  config_path = \"../vpc\"\n}\n\nlocals {\n  region = \"us-east-1\"\n}\n",
	"remote_state {\n  backend = \"s3\"\n  config = {\n    bucket = \"x\"\n  }\n}\n",
	// 3-deep nesting — exercises buildBody recursion + nested-Body raw paths
	"resource \"aws_instance\" \"web\" {\n  network_interface {\n    security_groups {\n      ids = []\n    }\n  }\n}\n",
	// Heredoc — distinct expression-byte handling
	"policy = <<-EOT\n  line one\n  line two\nEOT\n",
	"policy = <<EOT\nliteral\nEOT\n",
	// Multiple block types
	"variable \"name\" {\n  type    = string\n  default = \"hello\"\n}\n\noutput \"o\" {\n  value = var.name\n}\n",
	// Floating section-header — exercises StandaloneComment between blocks
	"resource \"x\" \"y\" {\n  a = 1\n}\n\n### SNS Notifications\n\nresource \"x\" \"z\" {\n  a = 2\n}\n",
	// Duplicate block type+labels — exercises the key-collision path
	// through collectTopLevelNames (two distinct CST Block items
	// collapse to one key on both sides of the comparison).
	"resource \"aws_instance\" \"web\" {\n  ami = \"a\"\n}\n\nresource \"aws_instance\" \"web\" {\n  ami = \"b\"\n}\n",
	// Block with brace-line comments
	"resource \"x\" \"y\" { # primary\n  a = 1\n} # end\n",
	// No final newline (parseable, but the mutate fuzz precondition skips)
	"name = \"alice\"",
}

// fuzzFixtureFiles are the in-tree round-trip targets — the same curated
// list serialize_test.go uses. Loaded by seedFuzzCorpus at fuzz init,
// after the inline seeds. Each entry is a repo-relative path; entries
// that cannot be read are silently skipped so the harness still runs in
// environments where the repo layout is incomplete (e.g., dist tarballs,
// sandboxed CI shards).
var fuzzFixtureFiles = []string{
	"examples/rule-test-files/valid.tf",
	"examples/rule-test-files/missing-tags.tf",
	"examples/rule-test-files/missing-description.tf",
	"examples/rule-test-files/hardcoded-account.tf",
	"examples/rule-test-files/suppression-annotations.tf",
	"vscode/src/test/fixtures/sample.tf",
}

// findRepoRoot walks up from the test's working directory until it finds
// the repository's go.mod. Returns an error rather than failing a *testing.T
// so callers (notably seedFuzzCorpus, which only has *testing.F at hand)
// can choose between fatal and graceful-skip handling. repoRoot in
// serialize_test.go wraps this with require.NoError.
func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate go.mod")
		}
		dir = parent
	}
}
