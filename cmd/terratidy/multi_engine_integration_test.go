//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
)

// TestMultiEngine_FailFastStopsEarly verifies that fail_fast: true stops processing
// after the first engine that produces error-severity findings.
// Note: fail_fast only triggers on error-severity findings, not warnings.
// When fmt produces errors (not warnings), style and lint should NOT run.
func TestMultiEngine_FailFastStopsEarly(t *testing.T) {
	resetCheckGlobals(t)

	// Create an unformatted file that triggers fmt errors (error severity).
	// The double blank line would trigger style.blank-line-between-blocks (warning)
	// if style ran, but it shouldn't because fmt errors trigger fail_fast first.
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
instance_type="t2.micro"
}


resource "aws_instance" "test2" {
ami="ami-456"
}`

	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Create a config with fail_fast enabled and multiple engines active.
	cfg := config.DefaultConfig()
	cfg.FailFast = config.BoolPtr(true)
	cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
	cfg.Engines.Style.Enabled = config.BoolPtr(true)
	cfg.Engines.Lint.Enabled = config.BoolPtr(true)
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)

	ctx := context.Background()

	// Run sequential checks (fail_fast only applies to sequential mode).
	findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tmpFile}, true, nil)
	require.NoError(t, err)

	// With fail_fast enabled, only fmt should run because fmt produces error-severity
	// findings. Style and lint should be skipped due to early exit.
	require.NotEmpty(t, findings, "should have at least one finding from fmt")

	// All findings should be from the fmt engine. If fail_fast worked correctly,
	// no style.* findings should be present (style engine was skipped).
	for _, f := range findings {
		assert.True(t, strings.HasPrefix(f.Rule, "fmt."),
			"all findings should be from fmt engine, got rule: %s", f.Rule)
	}
}

// TestMultiEngine_FailFastDisabled verifies that when fail_fast is false,
// all engines run even if one produces error-severity findings.
func TestMultiEngine_FailFastDisabled(t *testing.T) {
	resetCheckGlobals(t)

	// Create a file with both fmt and style issues.
	// Fmt will produce error-severity findings (unformatted code).
	// Style will produce warning-severity findings (double blank line).
	// With fail_fast disabled, both engines should run regardless of fmt errors.
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
instance_type="t2.micro"
}


resource "aws_instance" "test2" {
ami="ami-456"
}`

	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Create a config with fail_fast DISABLED.
	cfg := config.DefaultConfig()
	cfg.FailFast = config.BoolPtr(false)
	cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
	cfg.Engines.Style.Enabled = config.BoolPtr(true)
	cfg.Engines.Lint.Enabled = config.BoolPtr(false) // Skip lint for simplicity
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)

	ctx := context.Background()

	findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tmpFile}, true, nil)
	require.NoError(t, err)
	require.NotEmpty(t, findings, "expected findings from both engines")

	// With fail_fast disabled, both fmt and style should run.
	// We should have findings from both engines.
	var hasFmt, hasStyle bool
	for _, f := range findings {
		if strings.HasPrefix(f.Rule, "fmt.") {
			hasFmt = true
		}
		if strings.HasPrefix(f.Rule, "style.") {
			hasStyle = true
		}
	}

	assert.True(t, hasFmt, "should have fmt findings")
	assert.True(t, hasStyle, "should have style findings when fail_fast is disabled")
}

// TestMultiEngine_EngineErrorHandling verifies that when an engine returns an error
// (not findings, but an actual error like file not found), the error is propagated
// to the caller. This tests the error path, not the findings-based fail_fast logic.
func TestMultiEngine_EngineErrorHandling(t *testing.T) {
	resetCheckGlobals(t)

	dir := t.TempDir()

	// Reference a file that doesn't exist. The fmt engine will fail trying to read it.
	nonExistentFile := filepath.Join(dir, "does_not_exist.tf")

	// Create config with engines enabled.
	// Note: fail_fast is irrelevant here since engine errors (vs findings) always
	// cause runAllChecksSequentialWithConfig to return immediately.
	cfg := config.DefaultConfig()
	cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
	cfg.Engines.Style.Enabled = config.BoolPtr(true)
	cfg.Engines.Lint.Enabled = config.BoolPtr(false)
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)

	ctx := context.Background()

	// Run sequential checks - fmt engine should fail with file not found error.
	_, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{nonExistentFile}, true, nil)

	// The fmt engine should fail due to missing file, and the error should be propagated.
	require.Error(t, err, "expected error from fmt engine due to missing file")
	assert.Contains(t, err.Error(), "fmt", "error message should indicate fmt check failed")
}

// TestMultiEngine_LintOnly verifies that when only the lint engine is enabled,
// only lint findings are produced and no findings from other engines appear.
// This tests engine isolation: disabled engines must not produce findings.
func TestMultiEngine_LintOnly(t *testing.T) {
	resetCheckGlobals(t)

	// Create a file that triggers lint.terraform-unused-declarations.
	// The unused_var variable is declared but never referenced by any resource.
	// This also has unformatted code (would trigger fmt if enabled) and
	// missing blank lines (would trigger style if enabled).
	dir := t.TempDir()
	content := `variable "unused_var" {
  description = "This variable is declared but never used"
  type        = string
  default     = "unused"
}
resource "aws_instance" "test"   {
ami="ami-123"
instance_type="t2.micro"
}
resource "aws_instance" "test2" {
ami="ami-456"
}`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Configure only lint engine enabled with fallback to built-in rules.
	cfg := config.DefaultConfig()
	cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
	cfg.Engines.Style.Enabled = config.BoolPtr(false)
	cfg.Engines.Lint.Enabled = config.BoolPtr(true)
	cfg.Engines.Lint.UseTFLint = false
	cfg.Engines.Lint.FallbackBuiltin = true
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)

	ctx := context.Background()

	findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tmpFile}, true, nil)
	require.NoError(t, err)

	// Engine isolation check: verify no findings from disabled engines.
	// This assertion is unconditional - even with zero lint findings,
	// disabled engines must not produce any findings.
	for _, f := range findings {
		assert.False(t, strings.HasPrefix(f.Rule, "fmt."),
			"fmt engine is disabled but produced finding: %s", f.Rule)
		assert.False(t, strings.HasPrefix(f.Rule, "style."),
			"style engine is disabled but produced finding: %s", f.Rule)
		assert.False(t, strings.HasPrefix(f.Rule, "policy."),
			"policy engine is disabled but produced finding: %s", f.Rule)
	}

	// Verify lint engine ran and produced findings (lint.terraform-unused-declarations).
	// This is a liveness check to ensure the engine actually executed.
	var hasLint bool
	for _, f := range findings {
		if strings.HasPrefix(f.Rule, "lint.") {
			hasLint = true
			break
		}
	}
	assert.True(t, hasLint, "lint engine should produce at least one finding for unused variable")
}

// TestMultiEngine_PolicyOnly verifies that when only the policy engine is enabled,
// only policy findings are produced and no findings from other engines appear.
// This tests engine isolation: disabled engines must not produce findings.
func TestMultiEngine_PolicyOnly(t *testing.T) {
	resetCheckGlobals(t)

	// Create a file that triggers built-in policy.required-tags.
	// Resources without tags (aws_instance, aws_s3_bucket) trigger this policy.
	// This also has unformatted code (would trigger fmt if enabled).
	dir := t.TempDir()
	content := `resource "aws_instance" "test"   {
ami="ami-123"
instance_type="t2.micro"
}
resource "aws_s3_bucket" "test" {
bucket="my-test-bucket"
}`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Configure only policy engine enabled.
	// With no PolicyDirs/PolicyFiles, the engine uses built-in policies.
	cfg := config.DefaultConfig()
	cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
	cfg.Engines.Style.Enabled = config.BoolPtr(false)
	cfg.Engines.Lint.Enabled = config.BoolPtr(false)
	cfg.Engines.Policy.Enabled = config.BoolPtr(true)

	ctx := context.Background()

	findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tmpFile}, true, nil)
	require.NoError(t, err)

	// Engine isolation check: verify no findings from disabled engines.
	// This assertion is unconditional - even with zero policy findings,
	// disabled engines must not produce any findings.
	for _, f := range findings {
		assert.False(t, strings.HasPrefix(f.Rule, "fmt."),
			"fmt engine is disabled but produced finding: %s", f.Rule)
		assert.False(t, strings.HasPrefix(f.Rule, "style."),
			"style engine is disabled but produced finding: %s", f.Rule)
		assert.False(t, strings.HasPrefix(f.Rule, "lint."),
			"lint engine is disabled but produced finding: %s", f.Rule)
	}

	// Verify policy engine ran and produced findings (policy.required-tags).
	// This is a liveness check to ensure the engine actually executed.
	var hasPolicy bool
	for _, f := range findings {
		if strings.HasPrefix(f.Rule, "policy.") {
			hasPolicy = true
			break
		}
	}
	assert.True(t, hasPolicy, "policy engine should produce at least one finding for missing tags")
}

// TestMultiEngine_LintAndPolicyOnly verifies that when both lint and policy
// engines are enabled (but not fmt or style), only lint.* and policy.* findings
// are produced. This tests engine isolation with multiple enabled engines.
func TestMultiEngine_LintAndPolicyOnly(t *testing.T) {
	resetCheckGlobals(t)

	// Create a file that triggers both lint and policy engines:
	// - lint.terraform-unused-declarations: unused_var is never referenced
	// - policy.required-tags: aws_instance has no tags
	dir := t.TempDir()
	content := `variable "unused_var" {
  description = "Unused"
  type        = string
}
resource "aws_instance" "test"   {
ami="ami-123"
instance_type="t2.micro"
}`
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Configure lint and policy engines only.
	cfg := config.DefaultConfig()
	cfg.Engines.Fmt.Enabled = config.BoolPtr(false)
	cfg.Engines.Style.Enabled = config.BoolPtr(false)
	cfg.Engines.Lint.Enabled = config.BoolPtr(true)
	cfg.Engines.Lint.UseTFLint = false
	cfg.Engines.Lint.FallbackBuiltin = true
	cfg.Engines.Policy.Enabled = config.BoolPtr(true)

	ctx := context.Background()

	findings, err := runAllChecksSequentialWithConfig(ctx, cfg, []string{tmpFile}, true, nil)
	require.NoError(t, err)

	// Engine isolation check: verify no findings from disabled engines.
	// This is unconditional - even if lint/policy produce zero findings,
	// disabled engines must not produce any findings.
	for _, f := range findings {
		assert.False(t, strings.HasPrefix(f.Rule, "fmt."),
			"fmt engine is disabled but produced finding: %s", f.Rule)
		assert.False(t, strings.HasPrefix(f.Rule, "style."),
			"style engine is disabled but produced finding: %s", f.Rule)
	}

	// Track which enabled engines produced findings.
	var hasLint, hasPolicy bool
	for _, f := range findings {
		switch {
		case strings.HasPrefix(f.Rule, "lint."):
			hasLint = true
		case strings.HasPrefix(f.Rule, "policy."):
			hasPolicy = true
		}
	}

	// Liveness check: at least one enabled engine should produce findings.
	// The fixture is designed to trigger both lint and policy findings.
	assert.True(t, hasLint || hasPolicy,
		"at least one enabled engine (lint or policy) should produce findings")
}

// TestFmt_NoPhantomFixOnSecondRun locks in the spec success criterion:
// "Running fmt --all twice on the same tree results in zero changes the
// second time (no phantom-fix bug)." The original bug (reproduced 2026-04-30
// against workera-iac) had the engine reporting "Fixed N style issue(s)" on
// runs where the file content was actually unchanged — lifecycle-at-end and
// tags-at-end fixes cancelled each other out. The hash-based fixed-point
// detection in internal/engines/style/style.go:152-188 prevents this by
// refusing to write content the engine has already seen.
//
// The fixture is constructed so the FIRST run genuinely modifies the file
// (blank-line-between-blocks adds a blank line between the two resources)
// AND triggers the lifecycle-at-end + tags-at-end pair that historically
// drove the phantom-fix. The second run must produce zero file changes —
// even though the engine still emits Fixable findings for those rules (the
// tags-at-end Fix is a known no-op when tags already sit after lifecycle),
// the file content must not move.
func TestFmt_NoPhantomFixOnSecondRun(t *testing.T) {
	resetCheckGlobals(t) // also saves/restores `format`
	// Pin text output explicitly. In structured output mode (format="json"
	// and friends), the engine's hash-based fixed-point detection surfaces
	// a style.fix-loop error finding from the tags-at-end no-op Fix above,
	// which outputResults converts to an exit-1 error and would mask the
	// real bytes.Equal assertion. Text mode discards the fix-loop finding
	// silently, which is the user-facing behaviour this test cares about.
	format = "text"
	// resetCheckGlobals does not cover fmt-only flags; clean them up here.
	t.Cleanup(func() {
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
	})

	// Fixture:
	//   - Missing blank line between two top-level resources triggers
	//     blank-line-between-blocks, which the first run actually fixes.
	//   - Resource "b" has lifecycle in the middle and tags at the bottom —
	//     the original phantom-fix pattern. Rules fire on every pass but
	//     their fixes are no-ops on the canonical-ish post-first-run state.
	dir := t.TempDir()
	original := `resource "aws_instance" "a" {
  ami = "ami-1"
}
resource "aws_instance" "b" {
  ami = "ami-2"

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name = "b"
  }
}
`
	tfFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tfFile, []byte(original), 0o644))

	// First run: applies fmt + style fixes. Must produce real content change
	// so this test is not vacuously verifying "no-op on a clean file".
	fmtAll = true
	rootCmd.SetArgs([]string{"fmt", "--all", dir})
	require.NoError(t, rootCmd.Execute(), "first fmt --all run failed")

	afterFirst, err := os.ReadFile(tfFile)
	require.NoError(t, err, "reading file after first run")
	require.NotEqual(t, string(original), string(afterFirst),
		"first fmt --all run was a no-op — fixture does not exercise the regression "+
			"(check that blank-line-between-blocks is enabled by default and that the "+
			"two top-level resources are still missing a blank line in the fixture)")

	// Second run: must produce zero file changes. Even if rules still emit
	// findings whose Fix is a no-op (the tags-at-end "before lifecycle" fix
	// is a no-op for tags-after-lifecycle cases), the engine's hash-based
	// fixed-point detection MUST prevent the file from being rewritten with
	// identical content.
	rootCmd.SetArgs([]string{"fmt", "--all", dir})
	require.NoError(t, rootCmd.Execute(), "second fmt --all run failed")

	afterSecond, err := os.ReadFile(tfFile)
	require.NoError(t, err, "reading file after second run")

	if !bytes.Equal(afterFirst, afterSecond) {
		t.Fatalf(
			"phantom-fix regression: fmt --all second run modified the file\n"+
				"--- after first run (%d bytes) ---\n%s\n"+
				"--- after second run (%d bytes) ---\n%s",
			len(afterFirst), afterFirst, len(afterSecond), afterSecond,
		)
	}
}
