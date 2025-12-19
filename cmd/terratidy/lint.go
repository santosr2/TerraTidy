package main

import (
	"context"
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	lintConfigFile string
	lintPlugins    []string
	lintRules      []string
)

var lintCmd = &cobra.Command{
	Use:   "lint [paths...]",
	Short: "Run linting checks",
	Long: `Run linting checks to detect errors and best practice violations.

Linting performs static analysis of Terraform code to find potential issues,
security vulnerabilities, and violations of best practices.

Use --changed to only lint files that have been modified in git.`,
	Example: `  # Lint current directory
  terratidy lint

  # Lint specific paths
  terratidy lint ./modules ./environments

  # Only lint changed files
  terratidy lint --changed

  # Enable specific rules
  terratidy lint --rule terraform_required_version`,
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

		// Create rule config
		ruleConfig := make(map[string]lint.RuleConfig)
		for _, rule := range lintRules {
			ruleConfig[rule] = lint.RuleConfig{
				Enabled:  true,
				Severity: "warning",
			}
		}

		// Create lint engine
		engine := lint.New(&lint.Config{
			ConfigFile: lintConfigFile,
			Plugins:    lintPlugins,
			Rules:      ruleConfig,
		})

		modeMsg := ""
		if changed {
			modeMsg = " (changed files only)"
		}
		fmt.Printf("Running linter on %s%s...\n\n", formatFileCount(len(files)), modeMsg)

		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("running linter: %w", err)
		}

		// Display results
		if len(findings) == 0 {
			fmt.Println("No linting issues found")
			return nil
		}

		// Group findings by severity
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

			fmt.Printf("  [%s] %s:%d:%d - %s (%s)\n",
				icon,
				finding.File,
				finding.Location.Start.Line,
				finding.Location.Start.Column,
				finding.Message,
				finding.Rule,
			)
		}

		// Display summary
		fmt.Println()
		fmt.Println("---")
		fmt.Printf("Lint summary: %d error(s), %d warning(s), %d info\n", errors, warnings, info)

		if errors > 0 {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	lintCmd.Flags().StringVar(&lintConfigFile, "config-file", ".tflint.hcl", "path to TFLint config file")
	lintCmd.Flags().StringSliceVar(&lintPlugins, "plugin", []string{}, "plugins to enable (aws, google, azurerm)")
	lintCmd.Flags().StringSliceVar(&lintRules, "rule", []string{}, "specific rules to enable")
	rootCmd.AddCommand(lintCmd)
}
