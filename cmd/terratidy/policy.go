package main

import (
	"context"
	"fmt"

	"github.com/santosr2/TerraTidy/internal/engines/policy"
	"github.com/santosr2/TerraTidy/pkg/sdk"
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
			printNoFilesMessage()
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
			jsonData, jsonErr := engine.GetInput(files)
			if jsonErr != nil {
				return fmt.Errorf("generating input JSON: %w", jsonErr)
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
	return outputResults(findings, "Policy check summary")
}

func init() {
	policyCmd.Flags().StringSliceVar(&policyDirs, "policy-dir", nil, "directories containing Rego policy files")
	policyCmd.Flags().StringSliceVar(&policyFiles, "policy-file", nil, "individual Rego policy files")
	policyCmd.Flags().BoolVar(&policyShowJSON, "show-input", false, "show input JSON for debugging policies")
	rootCmd.AddCommand(policyCmd)
}
