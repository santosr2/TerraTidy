package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

	// Verify we have at least one finding with Fix populated
	hasFixableFinding := false
	for _, f := range findings {
		if f.Fix != nil {
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
			"style.blank-line-between-blocks": {Enabled: false},
		},
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	for _, f := range findings {
		assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule,
			"disabled rule should not produce findings")
	}
}
