package main

import (
	"context"
	"fmt"

	fmtengine "github.com/santosr2/terratidy/internal/engines/fmt"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var fixCmd = &cobra.Command{
	Use:   "fix [paths...]",
	Short: "Auto-fix all fixable issues",
	Long:  `Automatically fix formatting and style issues. Runs fmt + style --fix.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine paths to fix
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

		fmt.Printf("🔧 Fixing %d file(s)...\n\n", len(files))

		ctx := context.Background()
		var allFindings []sdk.Finding
		totalFixed := 0

		// 1. Run formatter (auto-fix by default)
		fmt.Println("1️⃣  Formatting files...")
		fmtEngine := fmtengine.New(&fmtengine.Config{Check: false})
		fmtFindings, err := fmtEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("formatting failed: %w", err)
		}

		// Count formatted files
		formatted := 0
		for _, f := range fmtFindings {
			if f.Rule == "fmt.formatted" {
				formatted++
			}
		}
		fmt.Printf("   ✓ Formatted %d file(s)\n\n", formatted)
		totalFixed += formatted
		allFindings = append(allFindings, fmtFindings...)

		// 2. Run style fixes
		fmt.Println("2️⃣  Fixing style issues...")
		styleEngine := style.New(&style.Config{
			Fix:   true,
			Rules: make(map[string]style.RuleConfig),
		})
		styleFindings, err := styleEngine.Run(ctx, files)
		if err != nil {
			return fmt.Errorf("style fixes failed: %w", err)
		}

		// Count fixed style issues
		styleFixed := 0
		for _, f := range styleFindings {
			if f.Fixable && f.FixFunc != nil {
				styleFixed++
			}
		}
		fmt.Printf("   ✓ Fixed %d style issue(s)\n\n", styleFixed)
		totalFixed += styleFixed
		allFindings = append(allFindings, styleFindings...)

		// Display summary
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("✨ Summary: Fixed %d issue(s)\n", totalFixed)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Check for remaining issues
		remainingIssues := 0
		for _, f := range allFindings {
			if !f.Fixable || f.FixFunc == nil {
				remainingIssues++
			}
		}

		if remainingIssues > 0 {
			fmt.Printf("\n⚠️  %d issue(s) require manual attention\n", remainingIssues)
			fmt.Println("\nRun 'terratidy check' to see remaining issues")
		} else {
			fmt.Println("\n✅ All fixable issues resolved!")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)
}
