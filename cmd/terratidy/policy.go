package main

import (
	"context"
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/engines/policy"
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

		// Create policy engine
		engine := policy.New(&policy.Config{
			PolicyDirs:  policyDirs,
			PolicyFiles: policyFiles,
		})

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

		// Display results
		if len(findings) == 0 {
			fmt.Println("All policy checks passed!")
			return nil
		}

		// Group by severity
		errors := 0
		warnings := 0
		info := 0

		for _, finding := range findings {
			icon := ""
			switch finding.Severity {
			case sdk.SeverityError:
				icon = "!"
				errors++
			case sdk.SeverityWarning:
				icon = "!"
				warnings++
			case sdk.SeverityInfo:
				icon = "i"
				info++
			}

			// Print finding
			fmt.Printf("  [%s] %s\n", icon, finding.Rule)
			fmt.Printf("      %s\n", finding.Message)
			if finding.File != "" {
				fmt.Printf("      File: %s", finding.File)
				if finding.Location.Start.Line > 0 {
					fmt.Printf(":%d", finding.Location.Start.Line)
				}
				fmt.Println()
			}
			fmt.Println()
		}

		// Summary
		fmt.Println("---")
		fmt.Printf("Policy check summary: %d error(s), %d warning(s), %d info\n",
			errors, warnings, info)

		if errors > 0 {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	policyCmd.Flags().StringSliceVar(&policyDirs, "policy-dir", nil, "directories containing Rego policy files")
	policyCmd.Flags().StringSliceVar(&policyFiles, "policy-file", nil, "individual Rego policy files")
	policyCmd.Flags().BoolVar(&policyShowJSON, "show-input", false, "show input JSON for debugging policies")
	rootCmd.AddCommand(policyCmd)
}
