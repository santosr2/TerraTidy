package main

import (
	"context"
	"fmt"

	"github.com/santosr2/terratidy/internal/engines/style"
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
	Long:  `Run the Style Engine to check for style violations and optionally fix them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine paths to check
		targetPaths := args
		if len(targetPaths) == 0 {
			targetPaths = []string{"."}
		}

		// Find all HCL files
		files, err := findHCLFiles(targetPaths)
		if err != nil {
			return fmt.Errorf("finding files: %w", err)
		}

		if len(files) == 0 {
			fmt.Println("No HCL files found")
			return nil
		}

		// Create style engine
		engine := style.New(&style.Config{
			Fix:   styleFix,
			Rules: make(map[string]style.RuleConfig),
		})

		// Run style checks
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("checking style: %w", err)
		}

		// Display results
		if len(findings) == 0 {
			fmt.Println("✓ No style issues found")
			return nil
		}

		hasErrors := false
		for _, finding := range findings {
			icon := "⚠"
			if finding.Severity == "error" {
				icon = "✗"
				hasErrors = true
			}
			fmt.Printf("%s %s: %s (%s)\n", icon, finding.File, finding.Message, finding.Rule)
		}

		if hasErrors || styleCheck {
			return fmt.Errorf("found %d style issue(s)", len(findings))
		}

		return nil
	},
}

func init() {
	styleCmd.Flags().BoolVar(&styleFix, "fix", false, "automatically fix style issues")
	styleCmd.Flags().BoolVar(&styleCheck, "check", false, "check only, do not modify files")
	styleCmd.Flags().BoolVar(&styleDiff, "diff", false, "show diff of style changes")
	rootCmd.AddCommand(styleCmd)
}
