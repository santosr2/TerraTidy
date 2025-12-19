// Package main provides the check command for TerraTidy.
package main

import (
	"context"
	"fmt"
	"os"

	fmtengine "github.com/santosr2/terratidy/internal/engines/fmt"
	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/engines/policy"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	checkSkipFmt    bool
	checkSkipStyle  bool
	checkSkipLint   bool
	checkSkipPolicy bool
)

var checkCmd = &cobra.Command{
	Use:   "check [paths...]",
	Short: "Run all checks (fmt, style, lint, policy)",
	Long: `Run all enabled engines in check mode. This is the recommended command for CI/CD.

Use --changed to only check files that have been modified in git.
Use --skip-* flags to skip specific engines.`,
	Example: `  # Run all checks
  terratidy check

  # Check specific paths
  terratidy check ./modules ./environments

  # Only check changed files (git)
  terratidy check --changed

  # Skip policy checks
  terratidy check --skip-policy`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkSkipFmt, "skip-fmt", false, "skip formatting checks")
	checkCmd.Flags().BoolVar(&checkSkipStyle, "skip-style", false, "skip style checks")
	checkCmd.Flags().BoolVar(&checkSkipLint, "skip-lint", false, "skip linting")
	checkCmd.Flags().BoolVar(&checkSkipPolicy, "skip-policy", false, "skip policy checks")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(_ *cobra.Command, args []string) error {
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

	modeMsg := ""
	if changed {
		modeMsg = " (changed files only)"
	}
	fmt.Printf("Checking %s%s...\n\n", formatFileCount(len(files)), modeMsg)

	ctx := context.Background()
	var allFindings []sdk.Finding

	step := 1

	// 1. Run formatter check
	if !checkSkipFmt {
		fmt.Printf("%d. Checking formatting...\n", step)
		step++

		fmtEngine := fmtengine.New(&fmtengine.Config{Check: true})
		fmtFindings, err := fmtEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("fmt check failed: %w", err)
		}
		allFindings = append(allFindings, fmtFindings...)
		fmt.Printf("   Found %d issue(s)\n\n", len(fmtFindings))
	}

	// 2. Run style checks
	if !checkSkipStyle {
		fmt.Printf("%d. Checking style...\n", step)
		step++

		styleEngine := style.New(&style.Config{
			Fix:   false,
			Rules: make(map[string]style.RuleConfig),
		})
		styleFindings, err := styleEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("style check failed: %w", err)
		}
		allFindings = append(allFindings, styleFindings...)
		fmt.Printf("   Found %d issue(s)\n\n", len(styleFindings))
	}

	// 3. Run linting
	if !checkSkipLint {
		fmt.Printf("%d. Running linter...\n", step)
		step++

		lintEngine := lint.New(&lint.Config{ConfigFile: ".tflint.hcl"})
		lintFindings, err := lintEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("lint check failed: %w", err)
		}
		allFindings = append(allFindings, lintFindings...)
		fmt.Printf("   Found %d issue(s)\n\n", len(lintFindings))
	}

	// 4. Run policy checks
	if !checkSkipPolicy {
		fmt.Printf("%d. Running policy checks...\n", step)

		policyEngine := policy.New(&policy.Config{})
		policyFindings, err := policyEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("policy check failed: %w", err)
		}
		allFindings = append(allFindings, policyFindings...)
		fmt.Printf("   Found %d issue(s)\n\n", len(policyFindings))
	}

	// Display summary
	fmt.Println("---")
	fmt.Printf("Summary: %d total issue(s)\n", len(allFindings))

	if len(allFindings) == 0 {
		fmt.Println("All checks passed!")
		return nil
	}

	// Group findings by severity
	errors := 0
	warnings := 0
	info := 0

	for _, finding := range allFindings {
		switch finding.Severity {
		case sdk.SeverityError:
			errors++
		case sdk.SeverityWarning:
			warnings++
		case sdk.SeverityInfo:
			info++
		}
	}

	fmt.Println()
	if errors > 0 {
		fmt.Printf("  Errors:   %d\n", errors)
	}
	if warnings > 0 {
		fmt.Printf("  Warnings: %d\n", warnings)
	}
	if info > 0 {
		fmt.Printf("  Info:     %d\n", info)
	}

	fmt.Println()
	fmt.Println("Run individual commands for details:")
	fmt.Println("  terratidy fmt --check")
	fmt.Println("  terratidy style")
	fmt.Println("  terratidy lint")
	fmt.Println("  terratidy policy")

	if errors > 0 {
		os.Exit(1)
	}

	return nil
}
