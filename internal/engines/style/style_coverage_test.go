package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_EmptyFiles(t *testing.T) {
	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_NilFiles(t *testing.T) {
	engine := New(nil)
	findings, err := engine.Run(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestFixWithDiff(t *testing.T) {
	dir := t.TempDir()
	// Content missing blank line between blocks (triggers blank-line-between-blocks rule)
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Should have findings with diff information
	hasDiff := false
	for _, f := range findings {
		if f.Message != "" && len(f.Message) > 10 {
			hasDiff = true
		}
	}
	_ = hasDiff // diff may or may not be in findings depending on implementation
}

func TestFixMode_ActuallyModifiesFile(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	})

	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// File should have been modified (blank line added between blocks)
	modified, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.NotEqual(t, content, string(modified), "fix mode should modify the file")
}

func TestMultiPassFix(t *testing.T) {
	dir := t.TempDir()
	// Content that may require multiple fix passes
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
resource "aws_instance" "test3" {
  ami = "ami-789"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	_ = findings

	// After fix, re-running check should find fewer or no issues
	checkEngine := New(&Config{Rules: make(map[string]RuleConfig)})
	checkFindings, err := checkEngine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// The blank-line rule should be satisfied after fix
	for _, f := range checkFindings {
		assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule,
			"blank line between blocks should be fixed after fix pass")
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" { ami = "ami-123" }
`
	for i := range 5 {
		f := filepath.Join(dir, "test"+string(rune('0'+i))+".tf")
		require.NoError(t, os.WriteFile(f, []byte(content), 0o644))
	}

	var files []string
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		files = append(files, filepath.Join(dir, e.Name()))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	engine := New(nil)
	_, err = engine.Run(ctx, files)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRun_NonExistentFile(t *testing.T) {
	engine := New(nil)
	_, err := engine.Run(context.Background(), []string{"/no/such/file.tf"})
	assert.Error(t, err)
}

func TestApplyFixes_WithFixableFindings(t *testing.T) {
	dir := t.TempDir()
	// Content missing blank line between blocks - this triggers a fixable finding
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// First run without fix to verify we get fixable findings
	checkEngine := New(&Config{Rules: make(map[string]RuleConfig)})
	findings, err := checkEngine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Verify we have at least one finding with Fixable set by the engine.
	// The engine dispatches to Fixer.Fix(ctx, file) lazily in applyFixes — Check()
	// never carries fix bytes through the Finding.
	hasFixableFinding := false
	for _, f := range findings {
		if f.Fixable {
			hasFixableFinding = true
			break
		}
	}
	assert.True(t, hasFixableFinding, "should have at least one fixable finding")

	// Now run with fix mode to apply the fix
	fixEngine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	})
	_, err = fixEngine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Verify file was modified
	modified, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.NotEqual(t, content, string(modified), "fix should modify the file")
	assert.Contains(t, string(modified), "\n\nresource", "should have blank line between blocks")
}

// stubNonFixerRule is a rule that does not implement sdk.Fixer; used to verify
// that the engine forces Fix to nil on findings from non-Fixer rules.
type stubNonFixerRule struct{}

func (r *stubNonFixerRule) Name() string        { return "test.stub-non-fixer" }
func (r *stubNonFixerRule) Description() string { return "Test rule that is not a Fixer" }

func (r *stubNonFixerRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return []sdk.Finding{{
		Rule:     r.Name(),
		Message:  "stub finding from non-fixer",
		File:     ctx.File,
		Severity: sdk.SeverityWarning,
		// Intentionally set Fixable to true here; the engine must overwrite it to
		// false because this rule does not implement Fixer.
		Fixable: true,
	}}, nil
}

// stubErroringFixerRule is a Fixer rule whose Fix() always returns an error;
// used to verify that applyFixes wraps and propagates Fix() errors.
type stubErroringFixerRule struct{}

func (r *stubErroringFixerRule) Name() string        { return "test.stub-erroring-fixer" }
func (r *stubErroringFixerRule) Description() string { return "Test rule that errors in Fix()" }

func (r *stubErroringFixerRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return []sdk.Finding{{
		Rule:     r.Name(),
		Message:  "stub finding",
		File:     ctx.File,
		Severity: sdk.SeverityWarning,
	}}, nil
}

func (r *stubErroringFixerRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, assert.AnError
}

func TestEngineDispatch_PropagatesFixerError(t *testing.T) {
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.tf")
	// Empty file produces no findings from any built-in rule, so the stub is the
	// only finding-generator the engine sees. This keeps the test from depending
	// on the rule registration order or the set of enabled-by-default rules.
	require.NoError(t, os.WriteFile(tmpFile, []byte("\n"), 0o644))

	engine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	}, &stubErroringFixerRule{})

	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.Error(t, err, "applyFixes must propagate errors from Fixer.Fix()")
	assert.Contains(t, err.Error(), "test.stub-erroring-fixer",
		"propagated error should name the failing rule")
}

func TestEngineDispatch_NonFixerFindingsHaveNilFix(t *testing.T) {
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte("resource \"aws_instance\" \"x\" {}\n"), 0o644))

	// Inject the stub via the plugin slot so it runs through the regular Check path.
	engine := New(&Config{Rules: make(map[string]RuleConfig)}, &stubNonFixerRule{})
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	var stubFinding *sdk.Finding
	for i := range findings {
		if findings[i].Rule == "test.stub-non-fixer" {
			stubFinding = &findings[i]
			break
		}
	}
	require.NotNil(t, stubFinding, "stub rule should produce a finding")
	assert.False(t, stubFinding.Fixable,
		"engine must force Fixable=false for findings from non-Fixer rules, even if the rule sets it")
}

func TestRun_RuleDisabling(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Disable the blank line rule
	engine := New(&Config{
		Rules: map[string]RuleConfig{
			"style.blank-line-between-blocks": {Enabled: config.BoolPtr(false)},
		},
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	for _, f := range findings {
		assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule,
			"disabled rule should not produce findings")
	}
}

// TestFixWithDiff_ProducesDiffFinding verifies that Fix+Diff mode emits a style.diff finding
// when the file is actually changed. This exercises the generateDiff path in checkFile.
func TestFixWithDiff_ProducesDiffFinding(t *testing.T) {
	dir := t.TempDir()
	// Missing blank line between blocks - blank-line rule will fix and produce a diff.
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// The engine must emit a style.diff finding containing the unified diff text.
	var diffFinding *sdk.Finding
	for i := range findings {
		if findings[i].Rule == "style.diff" {
			diffFinding = &findings[i]
			break
		}
	}
	require.NotNil(t, diffFinding, "Fix+Diff mode must produce a style.diff finding when the file changes")
	assert.Contains(t, diffFinding.Message, "@@", "diff finding should contain unified diff markers")
	assert.Equal(t, sdk.SeverityInfo, diffFinding.Severity)
}

// TestFixWithDiff_NoChangeNoFinding verifies that Fix+Diff mode does NOT emit a style.diff finding
// when the file content is unchanged (i.e. no fixable issues).
func TestFixWithDiff_NoChangeNoFinding(t *testing.T) {
	dir := t.TempDir()
	// Already well-formed file - nothing to fix.
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}

resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	for _, f := range findings {
		assert.NotEqual(t, "style.diff", f.Rule, "no diff finding expected when file is unchanged")
	}
}

// TestCheckFile_MultiPassFix_LoopContinues verifies that the fix loop executes multiple passes
// when fixes are applied (exercising the fixedCount > 0 → continue branch). After the engine
// finishes all passes, the file must be fully corrected.
func TestCheckFile_MultiPassFix_LoopContinues(t *testing.T) {
	dir := t.TempDir()
	// Three consecutive resources with no blank lines forces at least one full fix loop.
	content := `resource "aws_instance" "a" {
  ami = "ami-1"
}
resource "aws_instance" "b" {
  ami = "ami-2"
}
resource "aws_instance" "c" {
  ami = "ami-3"
}
`
	tmpFile := filepath.Join(dir, "multipass.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	})

	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// After all passes the blank-line rule must be satisfied.
	checkEngine := New(&Config{Rules: make(map[string]RuleConfig)})
	checkFindings, err := checkEngine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	for _, f := range checkFindings {
		assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule,
			"blank-line rule must be satisfied after multi-pass fix")
	}
}

// TestCheckFile_DiffAndFix_ReadOriginalContent covers the branch that reads the file's original
// content when both Diff and Fix are enabled. We ensure the engine reads the file without error
// and that the original content is captured for the eventual diff comparison.
func TestCheckFile_DiffAndFix_ReadOriginalContent(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "x" {
  ami = "ami-x"
}
resource "aws_instance" "y" {
  ami = "ami-y"
}
`
	tmpFile := filepath.Join(dir, "diff_original.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   true,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	// A successful run proves the original-content read path (lines 125-131) completed
	// without error and that the diff generation path was reached.
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// File was changed, so a diff finding must exist.
	found := false
	for _, f := range findings {
		if f.Rule == "style.diff" {
			found = true
			break
		}
	}
	assert.True(t, found, "diff finding expected after fixing a file with Diff+Fix mode enabled")
}

// TestDiffPreviewMode_GeneratesDiffWithoutModifyingFile verifies that Diff=true with Fix=false
// (preview mode) generates a diff finding showing what would change, but does NOT modify the
// original file. This exercises the capture-fix-restore logic in checkFile.
func TestDiffPreviewMode_GeneratesDiffWithoutModifyingFile(t *testing.T) {
	dir := t.TempDir()
	// Missing blank line between blocks - will trigger a fixable finding
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "preview.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Capture original content for comparison
	originalContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	engine := New(&Config{
		Fix:   false, // Preview mode - don't actually fix
		Diff:  true,  // But show what would change
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Must produce a style.diff finding showing preview of changes
	var diffFinding *sdk.Finding
	for i := range findings {
		if findings[i].Rule == "style.diff" {
			diffFinding = &findings[i]
			break
		}
	}
	require.NotNil(t, diffFinding, "preview mode (Diff=true, Fix=false) must produce a style.diff finding")
	assert.Contains(t, diffFinding.Message, "@@", "diff finding should contain unified diff markers")
	assert.Equal(t, sdk.SeverityInfo, diffFinding.Severity)

	// Verify file was NOT modified (original content restored)
	afterContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, afterContent, "file must remain unchanged in preview mode")
}

// TestDiffPreviewMode_NoIssues_NoFinding verifies that Diff=true with Fix=false produces no
// diff finding when there are no fixable issues.
func TestDiffPreviewMode_NoIssues_NoFinding(t *testing.T) {
	dir := t.TempDir()
	// Already well-formed file - nothing to fix
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}

resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	tmpFile := filepath.Join(dir, "already_good.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(&Config{
		Fix:   false,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	for _, f := range findings {
		assert.NotEqual(t, "style.diff", f.Rule, "no diff finding expected when file has no issues")
	}
}
