package main

import (
	"context"
	"fmt"

	"github.com/santosr2/TerraTidy/internal/config"
	fmtengine "github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	fmtCheck bool
	fmtDiff  bool
	fmtAll   bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [paths...]",
	Short: "Format Terraform and Terragrunt files",
	Long: `Format .tf and .hcl files using the HCL formatter.

Use --changed to only format files that have been modified in git.
Use --check to verify formatting without making changes.
Use --all to also apply style fixes (equivalent to running fmt + style --fix).`,
	Example: `  # Format all files in current directory
  terratidy fmt

  # Format specific directory
  terratidy fmt ./modules

  # Check formatting without modifying
  terratidy fmt --check

  # Only format changed files (git)
  terratidy fmt --changed

  # Format and apply style fixes
  terratidy fmt --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := loadConfig()
		if err != nil {
			return err // Already wrapped as ExitConfig by loadConfig
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

		// Create formatter engine with config+CLI merge
		// CLI flags override config values when explicitly set
		fmtCfg := buildFmtConfig(cmd, cfg)
		engine := fmtengine.New(fmtCfg)

		modeMsg := ""
		if changed {
			modeMsg = " (changed files only)"
		}
		if fmtAll {
			fmt.Printf("Formatting and applying style fixes to %s%s...\n\n", formatFileCount(len(files)), modeMsg)
		} else {
			fmt.Printf("Formatting %s%s...\n\n", formatFileCount(len(files)), modeMsg)
		}

		// Run formatter
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return sdk.NewInternalError(fmt.Errorf("formatting files: %w", err))
		}

		// Apply severity threshold filtering
		threshold := getEffectiveSeverityThreshold(cfg)
		findings = filterFindingsBySeverity(findings, threshold)

		// Display formatting results
		needsFormatting := 0
		formatted := 0
		for _, finding := range findings {
			switch finding.Rule {
			case "fmt.needs-formatting":
				fmt.Printf("  [!] %s: needs formatting\n", finding.File)
				needsFormatting++
			case "fmt.formatted":
				fmt.Printf("  [+] %s: formatted\n", finding.File)
				formatted++
			}
		}

		// Summary for formatting
		if len(findings) == 0 {
			fmt.Println("All files are properly formatted")
		} else if formatted > 0 {
			fmt.Println()
			fmt.Printf("Formatted %s\n", formatFileCount(formatted))
		}

		// In check mode, return findings error if any file needs formatting
		if fmtCfg.Check && needsFormatting > 0 {
			return sdk.NewFindingsError()
		}

		// Run style fixes if --all flag is set
		if fmtAll && !fmtCfg.Check {
			fmt.Println()
			fmt.Println("Applying style fixes...")
			fmt.Println()

			// Load plugin rules if plugins are enabled
			pluginRules, err := loadPluginRules(cfg)
			if err != nil {
				return sdk.NewConfigError(fmt.Errorf("loading plugins: %w", err))
			}

			// Use config-based style engine with plugin rules
			styleEngine := style.New(buildStyleConfig(cfg, true), pluginRules...)

			styleFindings, err := styleEngine.Run(context.Background(), files)
			if err != nil {
				return sdk.NewInternalError(fmt.Errorf("applying style fixes: %w", err))
			}

			styleFixed := 0
			for _, finding := range styleFindings {
				if finding.Fix != nil {
					styleFixed++
				}
			}

			if styleFixed > 0 {
				fmt.Printf("Fixed %d style issue(s)\n", styleFixed)

				// Re-run formatter after style fixes to restore proper HCL formatting
				// (style fixes may disrupt equal sign alignment)
				fmt.Println()
				fmt.Println("Re-formatting files...")
				rerunEngine := fmtengine.New(&fmtengine.Config{
					Check: false,
					Diff:  false,
				})
				if _, err := rerunEngine.Run(context.Background(), files); err != nil {
					return sdk.NewInternalError(fmt.Errorf("re-formatting files: %w", err))
				}
				fmt.Println("Done")
			} else {
				fmt.Println("No style issues to fix")
			}
		}

		return nil
	},
}

func init() {
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "check if files are formatted without modifying")
	fmtCmd.Flags().BoolVar(&fmtDiff, "diff", false, "show diff of formatting changes")
	fmtCmd.Flags().BoolVar(&fmtAll, "all", false, "also apply style fixes (equivalent to fmt + style --fix)")
	rootCmd.AddCommand(fmtCmd)
}

// buildFmtConfig creates a format engine config by merging config file values
// with CLI flags. CLI flags take precedence when explicitly set.
func buildFmtConfig(cmd *cobra.Command, cfg *config.Config) *fmtengine.Config {
	// Start with config values using the engine's ConfigFromEngine
	var result *fmtengine.Config
	if cfg != nil {
		result = fmtengine.ConfigFromEngine(cfg.Engines.Fmt)
	} else {
		result = &fmtengine.Config{}
	}

	// CLI flags override config when explicitly set
	if cmd.Flags().Changed("check") {
		checkVal, _ := cmd.Flags().GetBool("check")
		result.Check = checkVal
	}
	if cmd.Flags().Changed("diff") {
		diffVal, _ := cmd.Flags().GetBool("diff")
		result.Diff = diffVal
	}

	return result
}
