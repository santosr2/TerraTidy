package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFmtCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "fmt [paths...]", fmtCmd.Use)
		assert.Equal(t, "Format Terraform and Terragrunt files", fmtCmd.Short)
		assert.NotEmpty(t, fmtCmd.Long)
		assert.NotEmpty(t, fmtCmd.Example)
	})

	t.Run("has check flag", func(t *testing.T) {
		flag := fmtCmd.Flags().Lookup("check")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has diff flag", func(t *testing.T) {
		flag := fmtCmd.Flags().Lookup("diff")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has all flag", func(t *testing.T) {
		flag := fmtCmd.Flags().Lookup("all")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestFmtCmdExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a properly formatted file
	formattedContent := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "formatted.tf"), []byte(formattedContent), 0o644))

	t.Run("check mode with formatted files", func(t *testing.T) {
		// Reset flags
		fmtCheck = true
		fmtDiff = false
		changed = false

		rootCmd.SetArgs([]string{"fmt", "--check", tmpDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("no files found", func(t *testing.T) {
		emptyDir := t.TempDir()
		fmtCheck = false
		fmtDiff = false
		changed = false

		rootCmd.SetArgs([]string{"fmt", emptyDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("all flag with style issues", func(t *testing.T) {
		styleDir := t.TempDir()
		// Content with style issue (missing blank line between blocks)
		styleContent := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(styleDir, "style.tf"), []byte(styleContent), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = true
		changed = false

		rootCmd.SetArgs([]string{"fmt", "--all", styleDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Reset flag
		fmtAll = false
	})
}

// TestBuildFmtConfig_CLICheckOverridesConfig verifies CLI --check flag overrides config.
func TestBuildFmtConfig_CLICheckOverridesConfig(t *testing.T) {
	tests := []struct {
		name          string
		configCheck   bool
		cliCheckSet   bool
		cliCheckValue bool
		expectedCheck bool
	}{
		{
			name:          "config check=true, CLI not set -> uses config (true)",
			configCheck:   true,
			cliCheckSet:   false,
			cliCheckValue: false,
			expectedCheck: true,
		},
		{
			name:          "config check=false, CLI not set -> uses config (false)",
			configCheck:   false,
			cliCheckSet:   false,
			cliCheckValue: false,
			expectedCheck: false,
		},
		{
			name:          "config check=false, CLI --check -> CLI overrides (true)",
			configCheck:   false,
			cliCheckSet:   true,
			cliCheckValue: true,
			expectedCheck: true,
		},
		{
			name:          "config check=true, CLI --check=false -> CLI overrides (false)",
			configCheck:   true,
			cliCheckSet:   true,
			cliCheckValue: false,
			expectedCheck: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh cobra command to test flag changes
			cmd := &cobra.Command{}
			cmd.Flags().Bool("check", false, "")
			cmd.Flags().Bool("diff", false, "")

			// Set CLI flag if needed (sets both the flag value and marks it as changed)
			if tt.cliCheckSet {
				err := cmd.Flags().Set("check", strconv.FormatBool(tt.cliCheckValue))
				require.NoError(t, err)
			}

			// Create config with the test value
			cfg := &config.Config{
				Engines: config.Engines{
					Fmt: config.FmtEngineConfig{
						Check: tt.configCheck,
					},
				},
			}

			result := buildFmtConfig(cmd, cfg)
			assert.Equal(t, tt.expectedCheck, result.Check, "Check mode mismatch")
		})
	}
}

// TestBuildFmtConfig_CLIDiffOverridesConfig verifies CLI --diff flag overrides config.
func TestBuildFmtConfig_CLIDiffOverridesConfig(t *testing.T) {
	tests := []struct {
		name         string
		configDiff   bool
		cliDiffSet   bool
		cliDiffValue bool
		expectedDiff bool
	}{
		{
			name:         "config diff=true, CLI not set -> uses config (true)",
			configDiff:   true,
			cliDiffSet:   false,
			cliDiffValue: false,
			expectedDiff: true,
		},
		{
			name:         "config diff=false, CLI not set -> uses config (false)",
			configDiff:   false,
			cliDiffSet:   false,
			cliDiffValue: false,
			expectedDiff: false,
		},
		{
			name:         "config diff=false, CLI --diff -> CLI overrides (true)",
			configDiff:   false,
			cliDiffSet:   true,
			cliDiffValue: true,
			expectedDiff: true,
		},
		{
			name:         "config diff=true, CLI --diff=false -> CLI overrides (false)",
			configDiff:   true,
			cliDiffSet:   true,
			cliDiffValue: false,
			expectedDiff: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh cobra command to test flag changes
			cmd := &cobra.Command{}
			cmd.Flags().Bool("check", false, "")
			cmd.Flags().Bool("diff", false, "")

			// Set CLI flag if needed (sets both the flag value and marks it as changed)
			if tt.cliDiffSet {
				err := cmd.Flags().Set("diff", strconv.FormatBool(tt.cliDiffValue))
				require.NoError(t, err)
			}

			// Create config with the test value
			cfg := &config.Config{
				Engines: config.Engines{
					Fmt: config.FmtEngineConfig{
						Diff: tt.configDiff,
					},
				},
			}

			result := buildFmtConfig(cmd, cfg)
			assert.Equal(t, tt.expectedDiff, result.Diff, "Diff mode mismatch")
		})
	}
}

// TestBuildFmtConfig_NilConfig verifies behavior when config is nil
func TestBuildFmtConfig_NilConfig(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("check", false, "")
	cmd.Flags().Bool("diff", false, "")

	result := buildFmtConfig(cmd, nil)
	assert.False(t, result.Check, "Check should default to false")
	assert.False(t, result.Diff, "Diff should default to false")
}

// TestFmtCheckMode_ReturnsError_UnformattedFiles verifies exit code 1 on unformatted files.
// The complementary passing case (formatted files) is covered by TestFmtCmdExecution.
func TestFmtCheckMode_ReturnsError_UnformattedFiles(t *testing.T) {
	// Save and restore globals to avoid test pollution
	oldFmtCheck := fmtCheck
	oldFmtDiff := fmtDiff
	oldFmtAll := fmtAll
	oldChanged := changed

	// Reset Cobra flag Changed state (prevents pollution from prior tests)
	resetFmtFlags := func() {
		for _, name := range []string{"check", "diff", "all"} {
			if f := fmtCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
	resetFmtFlags()

	t.Cleanup(func() {
		fmtCheck = oldFmtCheck
		fmtDiff = oldFmtDiff
		fmtAll = oldFmtAll
		changed = oldChanged
		rootCmd.SetArgs(nil)
		resetFmtFlags()
	})

	// Create an unformatted Terraform file (inconsistent spacing, missing alignment)
	dir := t.TempDir()
	unformattedContent := `resource "aws_instance" "bad"   {
ami="ami-123"
instance_type="t2.micro"
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unformatted.tf"), []byte(unformattedContent), 0o644))

	// Reset flags for clean state
	fmtCheck = false
	fmtDiff = false
	fmtAll = false
	changed = false

	// Run fmt --check on unformatted file
	rootCmd.SetArgs([]string{"fmt", "--check", dir})
	err := rootCmd.Execute()

	// Should return ExitError with findings code (exit code 1)
	require.Error(t, err, "fmt --check should return error for unformatted files")

	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "should be an ExitError, got: %v", err)
	assert.Equal(t, sdk.ExitFindings, exitErr.Code, "should have findings exit code")
}

// TestBuildFmtConfig_BothFlagsOverride verifies both flags can be overridden simultaneously
func TestBuildFmtConfig_BothFlagsOverride(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("check", false, "")
	cmd.Flags().Bool("diff", false, "")

	// Config has both set to true
	cfg := &config.Config{
		Engines: config.Engines{
			Fmt: config.FmtEngineConfig{
				Check: true,
				Diff:  true,
			},
		},
	}

	// CLI sets both to false
	require.NoError(t, cmd.Flags().Set("check", "false"))
	require.NoError(t, cmd.Flags().Set("diff", "false"))

	result := buildFmtConfig(cmd, cfg)
	assert.False(t, result.Check, "CLI --check=false should override config check=true")
	assert.False(t, result.Diff, "CLI --diff=false should override config diff=true")
}

// TestFmtCmd_ErrorPaths tests error handling in fmt command
func TestFmtCmd_ErrorPaths(t *testing.T) {
	t.Run("invalid config returns ExitConfig", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create invalid config
		invalidConfig := "invalid: yaml: ["
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".terratidy.yaml"), []byte(invalidConfig), 0o600))

		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(oldWd) }()

		// Reset global flags
		cfgFile = ""
		changed = false
		format = "text"
		fmtCheck = false
		fmtDiff = false
		fmtAll = false

		rootCmd.SetArgs([]string{"fmt", "."})
		err := rootCmd.Execute()
		require.Error(t, err)

		var exitErr *sdk.ExitError
		if errors.As(err, &exitErr) {
			assert.Equal(t, sdk.ExitConfig, exitErr.Code, "invalid config should return ExitConfig")
		}
	})

	t.Run("no files found", func(t *testing.T) {
		emptyDir := t.TempDir()
		changed = false
		format = "text"
		fmtCheck = false
		fmtDiff = false
		fmtAll = false

		rootCmd.SetArgs([]string{"fmt", emptyDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("fmt with --all flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := `resource "null_resource" "test" {
  triggers = {
    a = "b"
  }
}
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0o644))

		// Reset flags
		for _, name := range []string{"check", "diff", "all"} {
			if f := fmtCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}

		changed = false
		format = "text"
		fmtCheck = false
		fmtDiff = false
		fmtAll = false

		rootCmd.SetArgs([]string{"fmt", "--all", tmpDir})
		err := rootCmd.Execute()
		// May have findings but should not fail
		_ = err
	})

	t.Run("fmt with changed flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := `resource "null_resource" "test" {}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0o644))

		// Reset flags
		changed = true
		format = "text"
		fmtCheck = false
		fmtDiff = false
		fmtAll = false

		rootCmd.SetArgs([]string{"fmt", "--changed", tmpDir})
		err := rootCmd.Execute()
		// Should handle gracefully even if not a git repo
		_ = err

		changed = false // Reset
	})
}

// TestFmtDiffFlag tests that --diff flag shows diff content in output
func TestFmtDiffFlag(t *testing.T) {
	// Save and restore globals
	oldFmtCheck := fmtCheck
	oldFmtDiff := fmtDiff
	oldFmtAll := fmtAll
	oldChanged := changed
	oldColor := color

	resetFmtFlags := func() {
		for _, name := range []string{"check", "diff", "all"} {
			if f := fmtCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
	resetFmtFlags()

	t.Cleanup(func() {
		fmtCheck = oldFmtCheck
		fmtDiff = oldFmtDiff
		fmtAll = oldFmtAll
		changed = oldChanged
		color = oldColor
		rootCmd.SetArgs(nil)
		resetFmtFlags()
	})

	t.Run("diff flag shows diff without modifying file in check mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create unformatted file
		unformattedContent := `resource "aws_instance" "bad"   {
ami="ami-123"
instance_type="t2.micro"
}`
		filePath := filepath.Join(tmpDir, "unformatted.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(unformattedContent), 0o644))

		// Get original content for comparison
		originalContent, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Reset flags
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		color = false // Disable color for predictable output

		// Capture stdout so we can assert the diff is actually printed.
		oldStdout := os.Stdout
		t.Cleanup(func() { os.Stdout = oldStdout })
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		os.Stdout = w
		rootCmd.SetArgs([]string{"fmt", "--check", "--diff", tmpDir})
		_ = rootCmd.Execute() // May return error for unformatted file
		_ = w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		out := buf.String()

		// Verify file was NOT modified (check mode)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "File should not be modified in check mode")

		// Verify the diff was actually printed: per-file marker, unified-diff
		// hunk header, both a removed and an added line, and the summary.
		// The added-line regex tolerates hclwrite's column-alignment changes
		// (the literal removed line is unambiguous because it's our input).
		assert.Contains(t, out, "[!] ", "expected per-file needs-formatting marker")
		assert.Contains(t, out, "unformatted.tf", "expected filename in output")
		assert.Contains(t, out, "@@", "expected unified diff hunk header")
		assert.Contains(t, out, `-ami="ami-123"`, "expected original line in diff")
		assert.Regexp(t, `\+\s+ami\s+=\s+"ami-123"`, out, "expected formatted line in diff")
		assert.Contains(t, out, "Found 1 file(s) that can be formatted", "expected --check --diff summary line")
	})

	t.Run("diff flag with normal mode shows diff and formats file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create unformatted file (missing spaces around =, extra spaces after resource type)
		unformattedContent := `resource "aws_instance" "test"   {
ami="ami-123"
}`
		filePath := filepath.Join(tmpDir, "format_me.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(unformattedContent), 0o644))

		// Reset flags
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		color = false

		rootCmd.SetArgs([]string{"fmt", "--diff", tmpDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Verify file WAS modified (normal mode)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.NotEqual(t, unformattedContent, string(afterContent), "File should be formatted")
		// HCL formatter adds spaces around = and proper indentation
		assert.Contains(t, string(afterContent), "ami = ", "File should have spaces around =")
		assert.Contains(t, string(afterContent), "  ami", "File should have proper indentation")
	})

	t.Run("all and diff flags together show style diffs without modifying file in check mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create file with style issue (extra blank lines between blocks)
		contentWithStyleIssue := `resource "aws_instance" "test" {
  ami = "ami-123"
}


resource "null_resource" "test2" {
  triggers = {}
}
`
		filePath := filepath.Join(tmpDir, "main.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(contentWithStyleIssue), 0o644))

		originalContent, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Reset flags
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		color = false

		rootCmd.SetArgs([]string{"fmt", "--all", "--diff", "--check", tmpDir})
		_ = rootCmd.Execute() // May return ExitFindings due to style issues

		// File should NOT be modified (check mode)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "File should not be modified in check mode")
	})
}

// TestFmtAllCheckMode verifies that fmt --all --check reports style issues
func TestFmtAllCheckMode(t *testing.T) {
	// Save and restore globals
	oldFmtCheck := fmtCheck
	oldFmtDiff := fmtDiff
	oldFmtAll := fmtAll
	oldChanged := changed
	oldColor := color

	resetFmtFlags := func() {
		for _, name := range []string{"check", "diff", "all"} {
			if f := fmtCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
	resetFmtFlags()

	t.Cleanup(func() {
		fmtCheck = oldFmtCheck
		fmtDiff = oldFmtDiff
		fmtAll = oldFmtAll
		changed = oldChanged
		color = oldColor
		rootCmd.SetArgs(nil)
		resetFmtFlags()
	})

	t.Run("all check mode returns error when style issues exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create file with style issues (tags not at end, depends_on not at end)
		contentWithStyleIssues := `resource "aws_instance" "test" {
  tags = {
    Name = "test"
  }

  ami           = "ami-123"
  instance_type = "t2.micro"

  depends_on = [aws_vpc.main]
}
`
		filePath := filepath.Join(tmpDir, "style_issues.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(contentWithStyleIssues), 0o644))

		// Get original content to verify file is not modified
		originalContent, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Reset flags
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		color = false

		rootCmd.SetArgs([]string{"fmt", "--all", "--check", tmpDir})
		err = rootCmd.Execute()

		// Should return error due to style issues
		require.Error(t, err, "fmt --all --check should return error when style issues exist")

		var exitErr *sdk.ExitError
		require.True(t, errors.As(err, &exitErr), "should be an ExitError")
		assert.Equal(t, sdk.ExitFindings, exitErr.Code, "should have findings exit code")

		// Verify file was NOT modified (check mode)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "File should not be modified in check mode")
	})

	t.Run("all check mode succeeds when no issues", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create well-formatted file with no style issues
		cleanContent := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  tags = {
    Name = "test"
  }

  depends_on = [aws_vpc.main]
}
`
		filePath := filepath.Join(tmpDir, "clean.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(cleanContent), 0o644))

		// Reset flags
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		color = false

		rootCmd.SetArgs([]string{"fmt", "--all", "--check", tmpDir})
		err := rootCmd.Execute()

		// Should succeed with no issues
		assert.NoError(t, err, "fmt --all --check should succeed when no issues exist")
	})
}

// TestFmtTextSummarySeparator verifies that `---` appears before fmt and style
// summary totals in text mode (matching lint/style/policy cadence), and does
// NOT appear when there are no findings to summarize.
func TestFmtTextSummarySeparator(t *testing.T) {
	oldFmtCheck := fmtCheck
	oldFmtDiff := fmtDiff
	oldFmtAll := fmtAll
	oldChanged := changed
	oldFormat := format
	oldColor := color

	resetFmtFlags := func() {
		for _, name := range []string{"check", "diff", "all"} {
			if f := fmtCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}

	t.Cleanup(func() {
		fmtCheck = oldFmtCheck
		fmtDiff = oldFmtDiff
		fmtAll = oldFmtAll
		changed = oldChanged
		format = oldFormat
		color = oldColor
		rootCmd.SetArgs(nil)
		resetFmtFlags()
	})

	captureStdout := func(t *testing.T, run func()) string {
		t.Helper()
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		run()
		_ = w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		return buf.String()
	}

	t.Run("fmt with formatted files prints --- before Formatted totals", func(t *testing.T) {
		resetFmtFlags()
		dir := t.TempDir()
		unformatted := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.tf"), []byte(unformatted), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "text"
		color = false

		out := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"fmt", dir})
			_ = rootCmd.Execute()
		})

		assert.Contains(t, out, "---\nFormatted ", "expected --- separator before Formatted totals")
	})

	t.Run("fmt with all-formatted files does NOT print --- separator", func(t *testing.T) {
		resetFmtFlags()
		dir := t.TempDir()
		formatted := `resource "aws_instance" "good" {
  ami = "ami-123"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "good.tf"), []byte(formatted), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "text"
		color = false

		out := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"fmt", dir})
			_ = rootCmd.Execute()
		})

		assert.NotContains(t, out, "---", "no --- separator when there are no totals to print")
		assert.Contains(t, out, "All files are properly formatted")
	})

	t.Run("fmt --check --diff with unformatted files prints Found summary", func(t *testing.T) {
		resetFmtFlags()
		dir := t.TempDir()
		unformatted := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.tf"), []byte(unformatted), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "text"
		color = false

		out := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"fmt", "--check", "--diff", dir})
			_ = rootCmd.Execute()
		})

		assert.Contains(t, out, "---\nFound 1 file(s) that can be formatted with 'terratidy fmt'\n",
			"expected --- separator and singular file summary in check+diff mode")
	})

	t.Run("fmt --check (no diff) does NOT print Found summary", func(t *testing.T) {
		resetFmtFlags()
		dir := t.TempDir()
		unformatted := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.tf"), []byte(unformatted), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "text"
		color = false

		out := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"fmt", "--check", dir})
			_ = rootCmd.Execute()
		})

		assert.NotContains(t, out, "Found ",
			"Found summary is scoped to --check --diff; bare --check keeps the per-file lines as the sole signal")
	})

	t.Run("fmt --check --diff with multiple files uses plural file(s) marker", func(t *testing.T) {
		resetFmtFlags()
		dir := t.TempDir()
		unformatted := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.tf"), []byte(unformatted), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.tf"), []byte(unformatted), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "text"
		color = false

		out := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"fmt", "--check", "--diff", dir})
			_ = rootCmd.Execute()
		})

		assert.Contains(t, out, "Found 2 file(s) that can be formatted with 'terratidy fmt'\n",
			"expected aggregate count for both unformatted files")
	})

	t.Run("fmt --all --check with style issues prints --- before Found totals", func(t *testing.T) {
		resetFmtFlags()
		dir := t.TempDir()
		// Style issue: tags should be at end of resource block.
		content := `resource "aws_instance" "test" {
  tags = {
    Name = "test"
  }

  ami = "ami-123"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "text"
		color = false

		out := captureStdout(t, func() {
			rootCmd.SetArgs([]string{"fmt", "--all", "--check", dir})
			_ = rootCmd.Execute()
		})

		assert.Contains(t, out, "---\nFound ", "expected --- separator before Found style totals")
	})
}

// TestFmtStructuredOutput verifies that `fmt --format json` (and other
// structured formats) emit valid JSON, suppress banners/per-file lines, and
// surface exit codes through the shared findings-error path.
func TestFmtStructuredOutput(t *testing.T) {
	// Save and restore globals to avoid test pollution
	oldFmtCheck := fmtCheck
	oldFmtDiff := fmtDiff
	oldFmtAll := fmtAll
	oldChanged := changed
	oldFormat := format
	oldColor := color

	resetFmtFlags := func() {
		for _, name := range []string{"check", "diff", "all"} {
			if f := fmtCmd.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}

	t.Cleanup(func() {
		fmtCheck = oldFmtCheck
		fmtDiff = oldFmtDiff
		fmtAll = oldFmtAll
		changed = oldChanged
		format = oldFormat
		color = oldColor
		rootCmd.SetArgs(nil)
		resetFmtFlags()
	})

	t.Run("check mode with unformatted file produces valid JSON and no banner", func(t *testing.T) {
		resetFmtFlags()

		dir := t.TempDir()
		unformattedContent := `resource "aws_instance" "bad"   {
ami="ami-123"
}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.tf"), []byte(unformattedContent), 0o644))

		// Reset flags
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "json"
		color = false

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		rootCmd.SetArgs([]string{"fmt", "--check", "--format", "json", dir})
		runErr := rootCmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		out := buf.String()

		// fmt --check on an unformatted file must surface a findings exit error.
		require.Error(t, runErr, "fmt --check on unformatted file should return ExitError")
		var exitErr *sdk.ExitError
		require.True(t, errors.As(runErr, &exitErr), "expected ExitError, got: %v", runErr)
		assert.Equal(t, sdk.ExitFindings, exitErr.Code)

		// Output must be valid JSON.
		var payload map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &payload), "output should be valid JSON, got: %s", out)

		// Banner text and per-file markers must not appear on stdout in structured mode.
		assert.NotContains(t, out, "Formatting ")
		assert.NotContains(t, out, "[!] ")
		assert.NotContains(t, out, "[+] ")
		assert.NotContains(t, out, "Re-aligning")
		assert.NotContains(t, out, "All files are properly formatted")

		// JSON payload should include the fmt finding.
		assert.Contains(t, out, "fmt.needs-formatting")
	})

	t.Run("all and check with style issue emits valid JSON with no banner", func(t *testing.T) {
		resetFmtFlags()

		dir := t.TempDir()
		// Style issue (tags should be at end of resource block).
		contentWithStyleIssue := `resource "aws_instance" "test" {
  tags = {
    Name = "test"
  }

  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(contentWithStyleIssue), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "json"
		color = false

		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		rootCmd.SetArgs([]string{"fmt", "--all", "--check", "--format", "json", dir})
		runErr := rootCmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		out := buf.String()

		// --all --check on a file with a style issue must return ExitFindings.
		require.Error(t, runErr, "fmt --all --check should error on style issue")
		var exitErr *sdk.ExitError
		require.True(t, errors.As(runErr, &exitErr), "expected ExitError, got: %v", runErr)
		assert.Equal(t, sdk.ExitFindings, exitErr.Code)

		// Output must be valid JSON.
		var payload map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &payload), "output should be valid JSON, got: %s", out)

		// No human-readable banners or sub-banners on stdout.
		assert.NotContains(t, out, "Checking formatting and style")
		assert.NotContains(t, out, "Checking style...")
		assert.NotContains(t, out, "Applying style fixes...")
		assert.NotContains(t, out, "Re-aligning")
		assert.NotContains(t, out, "Found ")
	})

	t.Run("formatted file in check mode emits valid JSON with no findings and no banner", func(t *testing.T) {
		resetFmtFlags()

		dir := t.TempDir()
		formattedContent := `resource "aws_instance" "good" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "good.tf"), []byte(formattedContent), 0o644))

		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		format = "json"
		color = false

		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		rootCmd.SetArgs([]string{"fmt", "--check", "--format", "json", dir})
		runErr := rootCmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		out := buf.String()

		assert.NoError(t, runErr, "fmt --check should succeed on formatted files")

		// Output must be valid JSON even when there are no findings.
		var payload map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &payload), "output should be valid JSON, got: %s", out)

		assert.NotContains(t, out, "Formatting ")
		assert.NotContains(t, out, "All files are properly formatted")
		assert.NotContains(t, out, "---")
	})
}
