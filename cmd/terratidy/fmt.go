package main

import (
	"context"
	"fmt"

	fmtengine "github.com/santosr2/terratidy/internal/engines/format"
	"github.com/santosr2/terratidy/internal/engines/style"
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

		// Create formatter engine
		engine := fmtengine.New(&fmtengine.Config{
			Check: fmtCheck,
			Diff:  fmtDiff,
		})

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
			return fmt.Errorf("formatting files: %w", err)
		}

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

		// In check mode, return error if any file needs formatting
		if fmtCheck && needsFormatting > 0 {
			return fmt.Errorf("%d file(s) need formatting", needsFormatting)
		}

		// Run style fixes if --all flag is set
		if fmtAll && !fmtCheck {
			fmt.Println()
			fmt.Println("Applying style fixes...")
			fmt.Println()

			styleEngine := style.New(&style.Config{
				Fix:   true,
				Rules: make(map[string]style.RuleConfig),
			})

			styleFindings, err := styleEngine.Run(context.Background(), files)
			if err != nil {
				return fmt.Errorf("applying style fixes: %w", err)
			}

			styleFixed := 0
			for _, finding := range styleFindings {
				if finding.Fixable && finding.FixFunc != nil {
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
					return fmt.Errorf("re-formatting files: %w", err)
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
