package main

import (
	"context"
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/pkg/sdk"
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
		// Get target files (respecting --changed flag)
		files, err := getTargetFiles(args, changed)
		if err != nil {
			return fmt.Errorf("finding files: %w", err)
		}

		if len(files) == 0 {
			if changed {
				fmt.Println("No changed HCL files found")
			} else {
				fmt.Println("No HCL files found")
			}
			return nil
		}

		// Create style engine
		engine := style.New(&style.Config{
			Fix:   styleFix,
			Rules: make(map[string]style.RuleConfig),
		})

		modeMsg := ""
		if changed {
			modeMsg = " (changed files only)"
		}
		fmt.Printf("Checking style on %s%s...\n\n", formatFileCount(len(files)), modeMsg)

		// Run style checks
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("checking style: %w", err)
		}

		// Display results
		if len(findings) == 0 {
			fmt.Println("No style issues found")
			return nil
		}

		// Count by severity
		var errors, warnings, info int
		for _, finding := range findings {
			switch finding.Severity {
			case sdk.SeverityError:
				errors++
			case sdk.SeverityWarning:
				warnings++
			case sdk.SeverityInfo:
				info++
			}
		}

		// Display findings
		for _, finding := range findings {
			icon := "i"
			switch finding.Severity {
			case sdk.SeverityError:
				icon = "!"
			case sdk.SeverityWarning:
				icon = "!"
			}

			location := finding.File
			if finding.Location.Start.Line > 0 {
				location = fmt.Sprintf("%s:%d", finding.File, finding.Location.Start.Line)
			}
			fmt.Printf("  [%s] %s: %s (%s)\n", icon, location, finding.Message, finding.Rule)
		}

		// Summary
		fmt.Println()
		fmt.Println("---")
		fmt.Printf("Style check summary: %d error(s), %d warning(s), %d info\n", errors, warnings, info)

		if errors > 0 {
			os.Exit(1)
		}

		if styleCheck && len(findings) > 0 {
			return fmt.Errorf("found %d style issue(s)", len(findings))
		}

		return nil
	},
}

func init() {
	styleCmd.Flags().BoolVar(&styleFix, "fix", false, "automatically fix style issues")
	styleCmd.Flags().BoolVar(&styleCheck, "check", false, "check only, exit with error if issues found")
	styleCmd.Flags().BoolVar(&styleDiff, "diff", false, "show diff of style changes")
	rootCmd.AddCommand(styleCmd)
}
