package main

import (
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

		rootCmd.SetArgs([]string{"fmt", "--check", "--diff", tmpDir})
		_ = rootCmd.Execute() // May return error for unformatted file

		// Verify file was NOT modified (check mode)
		afterContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "File should not be modified in check mode")
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

	t.Run("relative path display", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create unformatted file
		unformattedContent := `resource "null_resource" "x"   {}`
		filePath := filepath.Join(tmpDir, "rel_path.tf")
		require.NoError(t, os.WriteFile(filePath, []byte(unformattedContent), 0o644))

		// Reset flags and ensure absolute paths is off
		fmtCheck = false
		fmtDiff = false
		fmtAll = false
		changed = false
		absolutePaths = false

		rootCmd.SetArgs([]string{"fmt", tmpDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
		// Output uses DisplayPath which converts to relative paths by default
		// The test verifies the code path works without errors
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
