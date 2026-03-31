package main

import (
	"context"
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/engines/policy"
	"github.com/santosr2/terratidy/internal/output"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	policyDirs     []string
	policyFiles    []string
	policyShowJSON bool
)

var policyCmd = &cobra.Command{
	Use:   "policy [paths...]",
	Short: "Run policy checks using OPA/Rego",
	Long: `Run OPA policy checks against Terraform configurations.

Policy checks use Rego (the OPA policy language) to evaluate Terraform code
against custom policies. This enables organization-specific compliance and
security checks.

Built-in policies include:
  - Required terraform block with required_version
  - Required required_providers block
  - Security checks (no public SSH, no public S3, no public RDS)
  - Required tags on resources
  - Module version constraints

Custom policies can be provided via --policy-dir or --policy-file flags.
Use --changed to only check files that have been modified in git.`,
	Example: `  # Run policy checks on current directory
  terratidy policy

  # Run with custom policies
  terratidy policy --policy-dir ./policies

  # Only check changed files
  terratidy policy --changed

  # Show input JSON for debugging policies
  terratidy policy --show-input`,
	RunE: func(_ *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

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

		// Build policy config from terratidy config, then apply CLI overrides
		policyCfg := buildPolicyConfig(cfg)

		// CLI flags override config file settings
		if len(policyDirs) > 0 {
			policyCfg.PolicyDirs = policyDirs
		}
		if len(policyFiles) > 0 {
			policyCfg.PolicyFiles = policyFiles
		}

		// Create policy engine
		engine := policy.New(policyCfg)

		// Show input JSON if requested
		if policyShowJSON {
			jsonData, err := engine.GetInput(files)
			if err != nil {
				return fmt.Errorf("generating input JSON: %w", err)
			}
			fmt.Println(string(jsonData))
			return nil
		}

		modeMsg := ""
		if changed {
			modeMsg = " (changed files only)"
		}
		fmt.Printf("Running policy checks on %s%s...\n\n", formatFileCount(len(files)), modeMsg)

		// Run policy checks
		ctx := context.Background()
		findings, err := engine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("policy check failed: %w", err)
		}

		// Apply severity threshold filtering
		threshold := getEffectiveSeverityThreshold(cfg)
		findings = filterFindingsBySeverity(findings, threshold)

		// Output results using formatter
		return outputPolicyResults(findings)
	},
}

func outputPolicyResults(findings []sdk.Finding) error {
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
			fmt.Printf("Policy check summary: %d error(s), %d warning(s), %d info\n", errors, warnings, info)
		}

		if errors > 0 {
			return &sdk.ExitError{Code: 1}
		}
	} else {
		// Return exit error if there are errors (for structured output)
		for _, finding := range findings {
			if finding.Severity == sdk.SeverityError {
				return &sdk.ExitError{Code: 1}
			}
		}
	}

	return nil
}

func init() {
	policyCmd.Flags().StringSliceVar(&policyDirs, "policy-dir", nil, "directories containing Rego policy files")
	policyCmd.Flags().StringSliceVar(&policyFiles, "policy-file", nil, "individual Rego policy files")
	policyCmd.Flags().BoolVar(&policyShowJSON, "show-input", false, "show input JSON for debugging policies")
	rootCmd.AddCommand(policyCmd)
}
