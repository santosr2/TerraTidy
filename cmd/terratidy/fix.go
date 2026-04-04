// Package main provides the fix command for TerraTidy.
package main

import (
	"context"
	"fmt"

	"github.com/santosr2/TerraTidy/internal/config"
	fmtengine "github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var fixCmd = &cobra.Command{
	Use:   "fix [paths...]",
	Short: "Auto-fix all fixable issues",
	Long: `Automatically fix formatting and style issues. Runs fmt + style --fix.

Use --changed to only fix files that have been modified in git.`,
	Example: `  # Fix all files
  terratidy fix

  # Fix specific paths
  terratidy fix ./modules

  # Only fix changed files (git)
  terratidy fix --changed`,
	RunE: runFix,
}

func init() {
	rootCmd.AddCommand(fixCmd)
}

func runFix(_ *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	files, err := getTargetFiles(args, changed)
	if err != nil {
		return fmt.Errorf("finding files: %w", err)
	}

	if len(files) == 0 {
		printNoFilesMessage()
		return nil
	}

	printFixHeader(len(files))

	allFindings, totalFixed, err := runAllFixesWithConfig(cfg, files)
	if err != nil {
		return err
	}

	printFixSummary(allFindings, totalFixed)
	return nil
}

func printFixHeader(fileCount int) {
	modeMsg := ""
	if changed {
		modeMsg = " (changed files only)"
	}
	fmt.Printf("Fixing %s%s...\n\n", formatFileCount(fileCount), modeMsg)
}

func runAllFixesWithConfig(cfg *config.Config, files []string) ([]sdk.Finding, int, error) {
	ctx := context.Background()
	var allFindings []sdk.Finding
	totalFixed := 0

	fmtFindings, formatted, err := runFmtFix(ctx, files)
	if err != nil {
		return nil, 0, err
	}
	allFindings = append(allFindings, fmtFindings...)
	totalFixed += formatted

	styleFindings, styleFixed, err := runStyleFixWithConfig(ctx, cfg, files)
	if err != nil {
		return nil, 0, err
	}
	allFindings = append(allFindings, styleFindings...)
	totalFixed += styleFixed

	// Re-run fmt after style fixes to restore proper HCL formatting
	// (style fixes may disrupt equal sign alignment)
	if styleFixed > 0 {
		fmt.Println("3. Re-formatting files...")
		fmtEngine := fmtengine.New(&fmtengine.Config{Check: false})
		if _, err := fmtEngine.Run(ctx, files); err != nil {
			return nil, 0, fmt.Errorf("re-formatting failed: %w", err)
		}
		fmt.Println("   Done")
		fmt.Println()
	}

	return allFindings, totalFixed, nil
}

func runFmtFix(ctx context.Context, files []string) ([]sdk.Finding, int, error) {
	fmt.Println("1. Formatting files...")
	fmtEngine := fmtengine.New(&fmtengine.Config{Check: false})
	findings, err := fmtEngine.Run(ctx, files)
	if err != nil {
		return nil, 0, fmt.Errorf("formatting files: %w", err)
	}

	formatted := countFormattedFiles(findings)
	fmt.Printf("   Formatted %d file(s)\n\n", formatted)
	return findings, formatted, nil
}

func countFormattedFiles(findings []sdk.Finding) int {
	count := 0
	for _, f := range findings {
		if f.Rule == "fmt.formatted" {
			count++
		}
	}
	return count
}

func runStyleFixWithConfig(ctx context.Context, cfg *config.Config, files []string) ([]sdk.Finding, int, error) {
	fmt.Println("2. Fixing style issues...")
	styleCfg := buildStyleConfig(cfg, true)
	styleEngine := style.New(styleCfg)
	findings, err := styleEngine.Run(ctx, files)
	if err != nil {
		return nil, 0, fmt.Errorf("style fixes failed: %w", err)
	}

	fixed := countFixedStyleIssues(findings)
	fmt.Printf("   Fixed %d style issue(s)\n\n", fixed)
	return findings, fixed, nil
}

func countFixedStyleIssues(findings []sdk.Finding) int {
	count := 0
	for _, f := range findings {
		if f.Fix != nil {
			count++
		}
	}
	return count
}

func printFixSummary(allFindings []sdk.Finding, totalFixed int) {
	fmt.Println("---")
	fmt.Printf("Summary: Fixed %d issue(s)\n", totalFixed)

	remainingIssues := countRemainingIssues(allFindings)
	if remainingIssues > 0 {
		fmt.Printf("\n%d issue(s) require manual attention\n", remainingIssues)
		fmt.Println("\nRun 'terratidy check' to see remaining issues")
	} else {
		fmt.Println("\nAll fixable issues resolved!")
	}
}

func countRemainingIssues(findings []sdk.Finding) int {
	count := 0
	for _, f := range findings {
		if f.Fix == nil {
			count++
		}
	}
	return count
}
