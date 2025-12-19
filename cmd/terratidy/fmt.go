package main

import (
	"context"
	"fmt"

	fmtengine "github.com/santosr2/terratidy/internal/engines/format"
	"github.com/spf13/cobra"
)

var (
	fmtCheck bool
	fmtDiff  bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [paths...]",
	Short: "Format Terraform and Terragrunt files",
	Long: `Format .tf and .hcl files using the HCL formatter.

Use --changed to only format files that have been modified in git.
Use --check to verify formatting without making changes.`,
	Example: `  # Format all files in current directory
  terratidy fmt

  # Format specific directory
  terratidy fmt ./modules

  # Check formatting without modifying
  terratidy fmt --check

  # Only format changed files (git)
  terratidy fmt --changed`,
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
		fmt.Printf("Formatting %s%s...\n\n", formatFileCount(len(files)), modeMsg)

		// Run formatter
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("formatting files: %w", err)
		}

		// Display results
		if len(findings) == 0 {
			fmt.Println("All files are properly formatted")
			return nil
		}

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

		// Summary
		fmt.Println()
		if formatted > 0 {
			fmt.Printf("Formatted %s\n", formatFileCount(formatted))
		}

		// In check mode, return error if any file needs formatting
		if fmtCheck && needsFormatting > 0 {
			return fmt.Errorf("%d file(s) need formatting", needsFormatting)
		}

		return nil
	},
}

func init() {
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "check if files are formatted without modifying")
	fmtCmd.Flags().BoolVar(&fmtDiff, "diff", false, "show diff of formatting changes")
	rootCmd.AddCommand(fmtCmd)
}
