package main

import (
	"context"
	"fmt"

	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/spf13/cobra"
)

var (
	lintConfigFile string
)

var lintCmd = &cobra.Command{
	Use:   "lint [paths...]",
	Short: "Run linting checks",
	Long:  `Run TFLint to check for errors and best practice violations.`,
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

		// Create lint engine
		engine := lint.New(&lint.Config{
			ConfigFile: lintConfigFile,
		})

		// Run linter
		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("running linter: %w", err)
		}

		// Display results
		if len(findings) == 0 {
			fmt.Println("✓ No linting issues found")
			return nil
		}

		hasErrors := false
		for _, finding := range findings {
			icon := "ℹ"
			if finding.Severity == "warning" {
				icon = "⚠"
			} else if finding.Severity == "error" {
				icon = "✗"
				hasErrors = true
			}
			fmt.Printf("%s %s: %s (%s)\n", icon, finding.File, finding.Message, finding.Rule)
		}

		if hasErrors {
			return fmt.Errorf("found %d linting issue(s)", len(findings))
		}

		return nil
	},
}

func init() {
	lintCmd.Flags().StringVar(&lintConfigFile, "config-file", ".tflint.hcl", "path to TFLint config file")
	rootCmd.AddCommand(lintCmd)
}
