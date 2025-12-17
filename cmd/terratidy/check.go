package main

import (
	"context"
	"fmt"

	fmtengine "github.com/santosr2/terratidy/internal/engines/fmt"
	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check [paths...]",
	Short: "Run all checks (fmt, style, lint, policy)",
	Long:  `Run all enabled engines in check mode. This is the recommended command for CI/CD.`,
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

		fmt.Printf("🔍 Checking %d file(s)...\n\n", len(files))

		ctx := context.Background()
		var allFindings []sdk.Finding
		hasErrors := false

		// 1. Run formatter check
		fmt.Println("1️⃣  Checking formatting...")
		fmtEngine := fmtengine.New(&fmtengine.Config{Check: true})
		fmtFindings, err := fmtEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("fmt check failed: %w", err)
		}
		allFindings = append(allFindings, fmtFindings...)
		fmt.Printf("   Found %d issue(s)\n\n", len(fmtFindings))

		// 2. Run style checks
		fmt.Println("2️⃣  Checking style...")
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

		// 3. Run linting
		fmt.Println("3️⃣  Running linter...")
		lintEngine := lint.New(&lint.Config{ConfigFile: ".tflint.hcl"})
		lintFindings, err := lintEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("lint check failed: %w", err)
		}
		allFindings = append(allFindings, lintFindings...)
		fmt.Printf("   Found %d issue(s)\n\n", len(lintFindings))

		// Display summary
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("📊 Summary: %d total issue(s)\n", len(allFindings))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if len(allFindings) == 0 {
			fmt.Println("✅ All checks passed!")
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
				hasErrors = true
			case sdk.SeverityWarning:
				warnings++
			case sdk.SeverityInfo:
				info++
			}
		}

		fmt.Printf("\n")
		if errors > 0 {
			fmt.Printf("❌ Errors:   %d\n", errors)
		}
		if warnings > 0 {
			fmt.Printf("⚠️  Warnings: %d\n", warnings)
		}
		if info > 0 {
			fmt.Printf("ℹ️  Info:     %d\n", info)
		}

		fmt.Println("\nRun individual commands for details:")
		if len(fmtFindings) > 0 {
			fmt.Println("  terratidy fmt --check")
		}
		if len(styleFindings) > 0 {
			fmt.Println("  terratidy style")
		}
		if len(lintFindings) > 0 {
			fmt.Println("  terratidy lint")
		}

		if hasErrors {
			return fmt.Errorf("checks failed with %d error(s)", errors)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
