package cst

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripAllowlist names repo-relative prefixes containing fixtures that
// are intentionally malformed HCL — files that fail hclsyntax.ParseConfig
// and therefore cannot be expected to round-trip. The staleness gate in
// TestCSTRoundTrip_AllInTreeFixtures asserts that every prefix here has at
// least one file that actually fails to parse; a stale prefix (no broken
// files) fails the test loudly so the next maintainer either removes it or
// confirms it's intentional.
//
// Empty: no actually-malformed `.tf` / `.hcl` corpus lives in-tree.
// Rule-invalid fixtures (missing tags, missing required_version) parse
// cleanly and therefore do NOT belong here. Future malformed corpora can
// be added with a comment explaining intent.
var roundTripAllowlist = []string{}

// TestSerialize_EmptyFile pins that an empty Build round-trips to an empty
// (non-nil) byte slice.
func TestSerialize_EmptyFile(t *testing.T) {
	t.Parallel()

	f, err := Build([]byte(""), "empty.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	got := f.Bytes()
	require.NotNil(t, got)
	assert.Empty(t, got)
}

// TestSerialize_RoundTripInline pins identity round-trip on synthetic HCL
// inputs covering the shapes the build_test.go table exercises. Each case
// asserts Bytes(Build(content)) == content byte-for-byte: if Build adds any
// items the source didn't have, or Serialize ever decides to "tidy up"
// whitespace, this fails.
func TestSerialize_RoundTripInline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		policy  Policy
	}{
		{name: "empty", content: "", policy: DefaultTopLevelPolicy()},
		{name: "only LF blank lines", content: "\n\n\n", policy: DefaultTopLevelPolicy()},
		{name: "single hash comment", content: "# header\n", policy: DefaultTopLevelPolicy()},
		{name: "comments separated by blank", content: "# first\n\n# second\n", policy: DefaultTopLevelPolicy()},
		{name: "single attribute", content: "name = \"alice\"\n", policy: DefaultTopLevelPolicy()},
		{
			name:    "attribute with leading comment",
			content: "# why\nname = \"alice\"\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "attribute with inline comment",
			content: "name = \"alice\" # eyo\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "single empty block",
			content: "resource \"aws_instance\" \"web\" {\n}\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "block with attribute",
			content: "resource \"aws_instance\" \"web\" {\n  ami = \"ami-1\"\n}\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name: "nested blocks three deep",
			content: "resource \"aws_instance\" \"web\" {\n" +
				"  network_interface {\n" +
				"    security_groups {\n" +
				"      ids = []\n" +
				"    }\n" +
				"  }\n" +
				"}\n",
			policy: DefaultTopLevelPolicy(),
		},
		{
			name: "terragrunt include + dependency + locals",
			content: "include \"root\" {\n  path = find_in_parent_folders()\n}\n\n" +
				"dependency \"vpc\" {\n  config_path = \"../vpc\"\n}\n\n" +
				"locals {\n  region = \"us-east-1\"\n}\n",
			policy: DefaultTopLevelPolicy(),
		},
		{
			name:    "CRLF file",
			content: "resource \"aws_instance\" \"web\" {\r\n  ami = \"ami-1\"\r\n}\r\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "no final newline",
			content: "name = \"alice\"",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "attribute followed by blank line then attribute",
			content: "alpha = 1\n\nbeta = 2\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "block with opening-brace inline comment",
			content: "resource \"aws_instance\" \"web\" { # primary\n  ami = \"ami-1\"\n}\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "block with closing-brace inline comment",
			content: "resource \"aws_instance\" \"web\" {\n  ami = \"ami-1\"\n} # end\n",
			policy:  DefaultTopLevelPolicy(),
		},
		{
			name:    "passthrough policy attaches comment past blank line",
			content: "# why\n\nname = \"alice\"\n",
			policy:  DefaultBlockBodyPolicy(),
		},
		{
			// Heredocs are a distinct HCL token shape; their entire body
			// must travel verbatim in Attribute.ExpressionBytes. A regression
			// in the tokenizer / expression-range handling would corrupt
			// the inner lines here in a way the curated-fixture gate
			// wouldn't pinpoint to "heredoc".
			name:    "attribute with indented heredoc",
			content: "policy = <<-EOT\n  line one\n  line two\nEOT\n",
			policy:  DefaultTopLevelPolicy(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, err := Build([]byte(tc.content), tc.name+".tf", tc.policy)
			require.NoError(t, err)

			got := f.Bytes()
			assert.Equal(t, tc.content, string(got),
				"round-trip mismatch:\n--- want ---\n%q\n--- got ---\n%q",
				tc.content, string(got))
		})
	}
}

// TestSerialize_RoundTripCuratedFixtures pins identity round-trip on a
// hand-curated list of in-tree fixtures. Stable paths so a CI failure
// here points at a specific file. Compare against the broader walkdir
// gate below, which sweeps every `.tf`/`.hcl` file.
func TestSerialize_RoundTripCuratedFixtures(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	fixtures := []string{
		"examples/rule-test-files/valid.tf",
		"examples/rule-test-files/missing-tags.tf",
		"examples/rule-test-files/missing-description.tf",
		"examples/rule-test-files/hardcoded-account.tf",
		"examples/rule-test-files/suppression-annotations.tf",
		"vscode/src/test/fixtures/sample.tf",
	}

	for _, rel := range fixtures {
		path := filepath.Join(root, rel)
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			assertRoundTrip(t, path)
		})
	}
}

// TestCSTRoundTrip_AllInTreeFixtures sweeps every `.tf` and `.hcl` file
// reachable from the repo root and asserts round-trip identity. New
// fixtures added by future rules auto-join the gate — there's no per-file
// allowlist to forget to update.
//
// Files under a `roundTripAllowlist` prefix are skipped because they are
// intentionally malformed HCL. The companion subtest asserts the allowlist
// stays honest: every allowlisted prefix must contain at least one file
// that actually fails hclsyntax.ParseConfig, otherwise the prefix is stale.
func TestCSTRoundTrip_AllInTreeFixtures(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)

	t.Run("round_trip_identity", func(t *testing.T) {
		t.Parallel()

		var checked int
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if shouldSkipDir(root, path) {
					return fs.SkipDir
				}
				return nil
			}
			if !sdk.IsHCLFile(path) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if isAllowlisted(rel) {
				return nil
			}
			assertRoundTrip(t, path)
			checked++
			return nil
		})
		require.NoError(t, err)
		require.Positive(t, checked,
			"walkdir matched zero Terraform/HCL files — fixture discovery is broken")
	})

	t.Run("allowlist_staleness", func(t *testing.T) {
		t.Parallel()

		for _, prefix := range roundTripAllowlist {
			t.Run(prefix, func(t *testing.T) {
				t.Parallel()

				prefixPath := filepath.Join(root, prefix)
				var broken []string
				var found []string
				err := filepath.WalkDir(prefixPath, func(path string, d fs.DirEntry, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if d.IsDir() || !sdk.IsHCLFile(path) {
						return nil
					}
					rel, _ := filepath.Rel(root, path)
					found = append(found, rel)
					content, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					_, diags := hclsyntax.ParseConfig(content, path, hcl.InitialPos)
					if diags.HasErrors() {
						broken = append(broken, rel)
					}
					return nil
				})
				require.NoError(t, err)
				require.NotEmpty(t, broken,
					"allowlist prefix %q has no parse-failing files (found %d HCL files); "+
						"either remove the prefix or add a malformed fixture under it",
					prefix, len(found))
			})
		}
	})
}

// TestSerialize_CRLFPreservation pins that a CRLF source round-trips with
// every \r preserved, even when one item's raw is invalidated (simulating
// a rule that mutates one block in place). Unchanged-region CRLFs come
// from the items' raw bytes; the mutated region regenerates using
// File.lineSep, which Build sets to "\r\n" for a CRLF input.
func TestSerialize_CRLFPreservation(t *testing.T) {
	t.Parallel()

	const content = "alpha = 1\r\n\r\nbeta = 2\r\n"
	f, err := Build([]byte(content), "crlf.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	require.Equal(t, []byte("\r\n"), f.LineSep())

	// No-mutation: byte-identical round-trip.
	assert.Equal(t, content, string(f.Bytes()))

	// Mutated path: drop raw on the first attribute, forcing the regenerated
	// branch through writeRegenerated. Untouched items keep their raw bytes
	// (and their CRLF terminators); the mutated attribute regenerates using
	// f.lineSep, which is "\r\n", so its terminator is also CRLF. The
	// expected value is exact — a hypothetical duplication or normalization
	// bug would change the byte count and a Contains-style assertion would
	// still pass; Equal won't.
	first, ok := f.Body.Items[0].(*Attribute)
	require.True(t, ok, "expected first item to be Attribute, got %T", f.Body.Items[0])
	first.raw = nil

	assert.Equal(t, content, string(f.Bytes()),
		"regenerated CRLF attr must produce the same byte stream as the input")
}

// TestSerialize_MixedLineEndings pins the first-occurrence-wins
// detection: lineSep locks to the first newline shape seen. Round-trip
// on the un-mutated file is still byte-identical because unchanged items
// write their raw bytes, which already encode whichever ending they had —
// the detected lineSep only matters when an item regenerates.
func TestSerialize_MixedLineEndings(t *testing.T) {
	t.Parallel()

	// First newline is CRLF; subsequent are LF. lineSep should be CRLF.
	const content = "alpha = 1\r\nbeta = 2\ngamma = 3\n"
	f, err := Build([]byte(content), "mixed.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)

	assert.Equal(t, []byte("\r\n"), f.LineSep(),
		"first-occurrence-wins: first newline is CRLF so lineSep should be CRLF")

	// No-mutation round-trip preserves the mixed input verbatim.
	assert.Equal(t, content, string(f.Bytes()))
}

// TestSerialize_NoFinalNewline pins that a file ending without a final
// newline round-trips without one — Serialize never normalizes.
func TestSerialize_NoFinalNewline(t *testing.T) {
	t.Parallel()

	const content = "name = \"alice\""
	f, err := Build([]byte(content), "no-newline.tf", DefaultTopLevelPolicy())
	require.NoError(t, err)
	assert.Equal(t, content, string(f.Bytes()))
}

// TestSerialize_BlankLineRegeneratesUsingLineSep exercises the regenerated
// path for BlankLine. Construct a BlankLine with raw == nil and verify
// writeTo emits Count × lineSep. This is the path callers hit when they
// insert a fresh BlankLine between items.
func TestSerialize_BlankLineRegeneratesUsingLineSep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		lineSep []byte
		count   int
		want    string
	}{
		{name: "LF count 1", lineSep: []byte("\n"), count: 1, want: "\n"},
		{name: "LF count 3", lineSep: []byte("\n"), count: 3, want: "\n\n\n"},
		{name: "CRLF count 2", lineSep: []byte("\r\n"), count: 2, want: "\r\n\r\n"},
		{name: "nil lineSep defaults to LF", lineSep: nil, count: 2, want: "\n\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &File{
				Body: &Body{
					Items: []BodyItem{&BlankLine{Count: tc.count}},
				},
				lineSep: tc.lineSep,
			}
			assert.Equal(t, tc.want, string(f.Bytes()))
		})
	}
}

// TestSerialize_StandaloneCommentRegeneratesUsingLineSep exercises the
// regenerated path for StandaloneComment. Each comment's Raw is written
// followed by lineSep. No existing structural rule produces a
// StandaloneComment with nil raw, but the contract is here for future
// callers and for catching regressions in the regeneration shape.
func TestSerialize_StandaloneCommentRegeneratesUsingLineSep(t *testing.T) {
	t.Parallel()

	sc := &StandaloneComment{
		Comments: []Comment{
			{Style: CommentHash, Raw: []byte("# first")},
			{Style: CommentHash, Raw: []byte("# second")},
		},
	}
	f := &File{
		Body:    &Body{Items: []BodyItem{sc}},
		lineSep: []byte("\r\n"),
	}
	assert.Equal(t, "# first\r\n# second\r\n", string(f.Bytes()))
}

// TestSerialize_AttributeRegeneratesCanonically pins the writeRegenerated
// shape for an Attribute mutated since Build: `name = expression` plus
// optional leading and inline comments. No existing structural rule drives
// this path, but the canonical shape is the contract for future Insert.
func TestSerialize_AttributeRegeneratesCanonically(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr *Attribute
		want string
	}{
		{
			name: "bare",
			attr: &Attribute{Name: "name", ExpressionBytes: []byte("\"alice\"")},
			want: "name = \"alice\"\n",
		},
		{
			name: "with leading comment",
			attr: &Attribute{
				LeadingComments: []Comment{{Raw: []byte("# why")}},
				Name:            "name",
				ExpressionBytes: []byte("\"alice\""),
			},
			want: "# why\nname = \"alice\"\n",
		},
		{
			name: "with inline comment",
			attr: &Attribute{
				Name:            "name",
				ExpressionBytes: []byte("\"alice\""),
				InlineComment:   &Comment{Raw: []byte("# eyo")},
			},
			want: "name = \"alice\" # eyo\n",
		},
		{
			name: "with both leading and inline comments",
			attr: &Attribute{
				LeadingComments: []Comment{{Raw: []byte("# why")}, {Raw: []byte("# again")}},
				Name:            "name",
				ExpressionBytes: []byte("\"alice\""),
				InlineComment:   &Comment{Raw: []byte("# eyo")},
			},
			want: "# why\n# again\nname = \"alice\" # eyo\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &File{
				Body:    &Body{Items: []BodyItem{tc.attr}},
				lineSep: []byte("\n"),
			}
			assert.Equal(t, tc.want, string(f.Bytes()))
		})
	}
}

// TestSerialize_BlockRegeneratesCanonically pins the writeRegenerated shape
// for a Block mutated since Build. Labels, opening/closing brace inline
// comments, and nested body items all round-trip through the regenerated
// path. Body items inside a regenerated Block still use their own raw fast
// path or recurse into writeRegenerated as appropriate.
func TestSerialize_BlockRegeneratesCanonically(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		block *Block
		want  string
	}{
		{
			name:  "empty block no labels",
			block: &Block{Type: "locals", Body: &Body{}},
			want:  "locals {\n}\n",
		},
		{
			name: "block with labels",
			block: &Block{
				Type:   "resource",
				Labels: []Label{{Raw: []byte("\"aws_instance\"")}, {Raw: []byte("\"web\"")}},
				Body:   &Body{},
			},
			want: "resource \"aws_instance\" \"web\" {\n}\n",
		},
		{
			name: "block with leading and opening-brace comments",
			block: &Block{
				LeadingComments:     []Comment{{Raw: []byte("# header")}},
				Type:                "locals",
				OpeningBraceComment: &Comment{Raw: []byte("# inline")},
				Body:                &Body{},
			},
			want: "# header\nlocals { # inline\n}\n",
		},
		{
			name: "block with closing-brace comment",
			block: &Block{
				Type:                "locals",
				Body:                &Body{},
				ClosingBraceComment: &Comment{Raw: []byte("# end")},
			},
			want: "locals {\n} # end\n",
		},
		{
			name: "block with one body item via raw",
			block: &Block{
				Type: "locals",
				Body: &Body{Items: []BodyItem{
					&Attribute{raw: []byte("  region = \"us-east-1\"\n")},
				}},
			},
			want: "locals {\n  region = \"us-east-1\"\n}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &File{
				Body:    &Body{Items: []BodyItem{tc.block}},
				lineSep: []byte("\n"),
			}
			assert.Equal(t, tc.want, string(f.Bytes()))
		})
	}
}

// assertRoundTrip reads path and asserts Bytes(Build(content)) == content.
// The error message includes the first diverging byte offset and ±32 bytes
// around it, so CI logs point at the regression rather than dumping the
// full file diff.
func assertRoundTrip(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)

	f, err := Build(content, path, DefaultTopLevelPolicy())
	require.NoError(t, err, "build %s", path)

	got := f.Bytes()
	if bytes.Equal(content, got) {
		return
	}

	diff := firstDiffWindow(content, got, 32)
	t.Fatalf("round-trip mismatch for %s:\n%s", path, diff)
}

// firstDiffWindow renders a small context window around the first byte
// where want and got differ. Keeps CI logs short while making the failure
// site obvious.
func firstDiffWindow(want, got []byte, ctx int) string {
	n := len(want)
	if len(got) < n {
		n = len(got)
	}
	at := -1
	for i := 0; i < n; i++ {
		if want[i] != got[i] {
			at = i
			break
		}
	}
	if at == -1 {
		return strings.Join([]string{
			"length mismatch",
			"  want len: " + strconv.Itoa(len(want)),
			"  got  len: " + strconv.Itoa(len(got)),
			"  want tail: " + quoteSnippet(tail(want, n, 64)),
			"  got  tail: " + quoteSnippet(tail(got, n, 64)),
		}, "\n")
	}
	start := at - ctx
	if start < 0 {
		start = 0
	}
	endWant := at + ctx
	if endWant > len(want) {
		endWant = len(want)
	}
	endGot := at + ctx
	if endGot > len(got) {
		endGot = len(got)
	}
	return strings.Join([]string{
		"  diverge at byte: " + strconv.Itoa(at),
		"  want: " + quoteSnippet(want[start:endWant]),
		"  got:  " + quoteSnippet(got[start:endGot]),
	}, "\n")
}

// tail returns up to limit bytes starting at offset from. Used by
// firstDiffWindow to show the trailing bytes when the prefix matches but
// the lengths differ.
func tail(b []byte, from, limit int) []byte {
	if from >= len(b) {
		return nil
	}
	end := from + limit
	if end > len(b) {
		end = len(b)
	}
	return b[from:end]
}

func quoteSnippet(b []byte) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, c := range b {
		switch c {
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		default:
			out.WriteByte(c)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// repoRoot walks up from the test's working directory until it finds the
// repository's `go.mod`. Tests use this to anchor walkdir from the repo
// root regardless of which package's `go test` invoked them. Failures here
// are fatal because every caller anchors a walkdir gate on the result.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := findRepoRoot()
	require.NoError(t, err)
	return dir
}

// shouldSkipDir prunes directory trees the walkdir doesn't need to descend
// into. node_modules is the heavyweight one (vscode/ has thousands of
// files); `.git` is excluded so the walker doesn't trip on packed objects;
// build outputs and worktrees are skipped because they hold derived copies
// of in-tree fixtures and would inflate the gate without adding coverage.
func shouldSkipDir(root, path string) bool {
	base := filepath.Base(path)
	switch base {
	case ".git", "node_modules", ".history", ".worktree", "bin", "dist", "coverage":
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return strings.Contains(rel, string(filepath.Separator)+"node_modules"+string(filepath.Separator)) ||
		strings.HasPrefix(rel, "node_modules"+string(filepath.Separator))
}

// isAllowlisted reports whether a repo-relative file path falls under any
// roundTripAllowlist prefix.
func isAllowlisted(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, prefix := range roundTripAllowlist {
		p := filepath.ToSlash(prefix)
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}
