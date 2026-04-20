package main

import (
	"context"
	"fmt"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/internal/output"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	styleFix   bool
	styleCheck bool
	styleDiff  bool
)

var styleCmd = &cobra.Command{
	Use:   "style [paths...]",
	Short: "Check and fix style issues",
	Long: `Run the Style Engine to check for style violations and optionally fix them.

Style checks ensure consistent formatting and organization of Terraform code,
including block ordering, naming conventions, and structural consistency.

Use --changed to only check files that have been modified in git.
Use --fix to automatically fix fixable style issues.`,
	Example: `  # Check style in current directory
  terratidy style

  # Check and fix style issues
  terratidy style --fix

  # Only check changed files
  terratidy style --changed

  # Check specific paths
  terratidy style ./modules ./environments`,
	RunE: func(_ *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := loadConfig()
		if err != nil {
			return err // Already wrapped as ExitConfig by loadConfig
		}

		// Load plugin rules if plugins are enabled
		pluginRules, err := loadPluginRules(cfg)
		if err != nil {
			return sdk.NewConfigError(fmt.Errorf("loading plugins: %w", err))
		}

		// Get target files (respecting --changed flag and excludes)
		files, err := getTargetFilesWithExcludes(args, changed, cfg.Exclude, cfg)
		if err != nil {
			return sdk.NewInternalError(fmt.Errorf("finding files: %w", err))
		}

		if len(files) == 0 {
			printNoFilesMessage()
			return nil
		}

		// Create style engine with config and plugin rules
		engine := style.New(buildStyleConfig(cfg, styleFix, styleDiff), pluginRules...)

		// For structured output formats, skip the progress messages
		useStructuredOutput := format != "" && format != "text"

		if !useStructuredOutput {
			modeMsg := ""
			if changed {
				modeMsg = " (changed files only)"
			}
			fmt.Printf("Checking style on %s%s...\n\n", formatFileCount(len(files)), modeMsg)
		}

		// Run style checks
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return sdk.NewInternalError(fmt.Errorf("checking style: %w", err))
		}

		// Extract diff findings BEFORE severity filtering (they have SeverityInfo and
		// would be filtered out by the default "warning" threshold)
		var diffFindings []sdk.Finding
		var regularFindings []sdk.Finding
		for _, finding := range findings {
			if finding.Rule == "style.diff" {
				diffFindings = append(diffFindings, finding)
			} else {
				regularFindings = append(regularFindings, finding)
			}
		}

		// Apply severity threshold filtering to regular findings only
		threshold := getEffectiveSeverityThreshold(cfg)
		regularFindings = filterFindingsBySeverity(regularFindings, threshold)

		// For text format, print diff findings with colored output (matching fmt.go pattern)
		if !useStructuredOutput {
			for _, finding := range diffFindings {
				// Print blank line before diff, then colored diff (same pattern as fmt.go)
				fmt.Println()
				fmt.Print(output.FormatDiff(finding.Message, color))
			}
		}

		// For structured output, include diff findings in the formatter output
		// (but NOT in the count used for check-mode exit code)
		outputFindings := regularFindings
		if useStructuredOutput {
			outputFindings = append(outputFindings, diffFindings...)
		}

		// Output results using formatter (regularFindings count for check-mode, not diff findings)
		return outputStyleResults(outputFindings, regularFindings, styleCheck, cfg)
	},
}

func outputStyleResults(outputFindings, checkModeFindings []sdk.Finding, checkMode bool, cfg *config.Config) error {
	if err := outputResults(outputFindings, "Style check summary", cfg); err != nil {
		return err
	}

	// In check mode, return findings error if any style issues found
	// (excludes style.diff findings which are informational)
	if checkMode && len(checkModeFindings) > 0 {
		return sdk.NewFindingsError()
	}

	return nil
}

func init() {
	styleCmd.Flags().BoolVar(&styleFix, "fix", false, "automatically fix style issues")
	styleCmd.Flags().BoolVar(&styleCheck, "check", false, "check only, exit with error if issues found")
	styleCmd.Flags().BoolVar(&styleDiff, "diff", false, "show diff of style changes")
	rootCmd.AddCommand(styleCmd)
}
