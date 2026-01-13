package main

import (
	"context"
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/output"
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

		// For structured output formats, skip the progress messages
		useStructuredOutput := format != "" && format != "text"

		if !useStructuredOutput {
			modeMsg := ""
			if changed {
				modeMsg = " (changed files only)"
			}
			fmt.Printf("Running linter on %s%s...\n\n", formatFileCount(len(files)), modeMsg)
		}

		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return fmt.Errorf("running linter: %w", err)
		}

		// Output results using formatter
		return outputLintResults(findings)
	},
}

func outputLintResults(findings []sdk.Finding) error {
	formatter, err := output.GetFormatterWithColor(format, true, version, color)
	if err != nil {
		return fmt.Errorf("getting formatter: %w", err)
	}

	if err := formatter.Format(findings, os.Stdout); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}

	// For text format, add summary
	if format == "" || format == "text" {
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

		if len(findings) > 0 {
			fmt.Println()
			fmt.Println("---")
			fmt.Printf("Lint summary: %d error(s), %d warning(s), %d info\n", errors, warnings, info)
		}

		if errors > 0 {
			os.Exit(1)
		}
	} else {
		// Exit with error code if there are errors (for structured output)
		for _, finding := range findings {
			if finding.Severity == sdk.SeverityError {
				os.Exit(1)
			}
		}
	}

	return nil
}

func init() {
	lintCmd.Flags().StringVar(&lintConfigFile, "config-file", ".tflint.hcl", "path to TFLint config file")
	lintCmd.Flags().StringSliceVar(&lintPlugins, "plugin", []string{}, "plugins to enable (aws, google, azurerm)")
	lintCmd.Flags().StringSliceVar(&lintRules, "rule", []string{}, "specific rules to enable")
	rootCmd.AddCommand(lintCmd)
}
