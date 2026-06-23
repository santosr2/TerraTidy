package rules

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Byte-exact goldens for MetaArgumentsOrderRule.Fix.
//
// The Fix walks the CST and relocates leading meta-args (for_each/count,
// provider) to the start of each block body, then depends_on to the last
// slot. Each item keeps its original source bytes, so column alignment
// and intra-block blank lines round-trip verbatim — only the slice order
// changes. Any future refactor of the Fix path can be evaluated against
// these goldens with a byte-exact diff: whitespace-only divergence in
// regenerated regions is an expected outcome of changing the structural
// algorithm and may justify a golden update; a semantic divergence
// (different attribute order, dropped comment, changed value, etc.)
// indicates a correctness regression.
//
// Fixtures live alongside TestMetaArgumentsOrderRule in
// block_organization_test.go (metaArgsReorderInput,
// metaArgsProviderBeforeCountInput) so the semantic assertion and the
// byte-exact snapshot share the same input — a fixture change cannot
// silently desync the two.
//
// Capture / re-capture: UPDATE_GOLDEN=1 go test -run TestMetaArgumentsOrderRule_FixGoldens ./internal/engines/style/rules/
func TestMetaArgumentsOrderRule_FixGoldens(t *testing.T) {
	rule := &MetaArgumentsOrderRule{}

	fixtures := []struct {
		name   string
		golden string
		input  string
	}{
		{
			name:   "reorder depends_on after for_each",
			golden: "meta_arguments_order/reorder_depends_on_after_for_each",
			input:  metaArgsReorderInput,
		},
		{
			name:   "reorder count before provider",
			golden: "meta_arguments_order/reorder_count_before_provider",
			input:  metaArgsProviderBeforeCountInput,
		},
		{
			// Pins the lifecycle-block interaction: depends_on goes to the
			// last body slot, which is after the lifecycle nested block.
			// Sibling rules own any further refinement of that position.
			name:   "reorder depends_on after lifecycle",
			golden: "meta_arguments_order/reorder_depends_on_after_lifecycle",
			input:  metaArgsWithLifecycleInput,
		},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tc.input), 0o644))

			file, diags := hclsyntax.ParseConfig([]byte(tc.input), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors(), "fixture failed to parse: %v", diags)

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			result, err := rule.Fix(ctx, hclFile)
			require.NoError(t, err)
			require.NotNil(t, result, "fixture must produce a fix; the no-op case belongs in the main test")
			require.Len(t, result.Edits, 1)

			assertRuleGolden(t, tc.golden, result.Edits[0].Replacement)
		})
	}
}

// assertRuleGolden compares actual bytes to testdata/goldens/<name>.golden.
// Set UPDATE_GOLDEN=1 to write the golden instead of asserting.
//
// Line endings are normalized to \n on read so the comparison is stable on
// Windows checkouts; writes preserve the bytes the rule produced.
func assertRuleGolden(t *testing.T, name string, actual []byte) {
	t.Helper()

	path := filepath.Join("testdata", "goldens", name+".golden")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755), "failed to create golden directory for %s", path)
		require.NoError(t, os.WriteFile(path, actual, 0o644), "failed to write golden %s", path)
		t.Logf("wrote golden %s (%d bytes)", path, len(actual))
		return
	}

	expected, err := os.ReadFile(path)
	require.NoError(t, err, "missing golden %s — re-run with UPDATE_GOLDEN=1 to create", path)

	normalizedExpected := bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n"))
	normalizedActual := bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n"))

	assert.Equal(
		t, string(normalizedExpected), string(normalizedActual),
		"output diverged from golden %s — whitespace-only diffs in realigned attribute columns are acceptable with reviewer sign-off and an UPDATE_GOLDEN=1 refresh; any semantic diff (reordered attributes, dropped comments, changed values) indicates a correctness regression",
		path,
	)
}
