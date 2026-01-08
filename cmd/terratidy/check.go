// Package main provides the check command for TerraTidy.
package main

import (
	"context"
	"fmt"
	"os"

	fmtengine "github.com/santosr2/terratidy/internal/engines/format"
	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/engines/policy"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/internal/output"
	"github.com/santosr2/terratidy/internal/runner"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	checkSkipFmt    bool
	checkSkipStyle  bool
	checkSkipLint   bool
	checkSkipPolicy bool
	checkParallel   bool
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
  terratidy check --skip-policy

  # Run engines in parallel for faster execution
  terratidy check --parallel`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkSkipFmt, "skip-fmt", false, "skip formatting checks")
	checkCmd.Flags().BoolVar(&checkSkipStyle, "skip-style", false, "skip style checks")
	checkCmd.Flags().BoolVar(&checkSkipLint, "skip-lint", false, "skip linting")
	checkCmd.Flags().BoolVar(&checkSkipPolicy, "skip-policy", false, "skip policy checks")
	checkCmd.Flags().BoolVarP(&checkParallel, "parallel", "p", false, "run engines in parallel")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(_ *cobra.Command, args []string) error {
	files, err := getTargetFiles(args, changed)
	if err != nil {
		return fmt.Errorf("finding files: %w", err)
	}

	if len(files) == 0 {
		printNoFilesMessage()
		return nil
	}

	// For structured output formats, skip the progress messages
	useStructuredOutput := format != "" && format != "text"

	if !useStructuredOutput {
		printCheckHeader(len(files))
	}

	allFindings, err := runAllChecks(files, useStructuredOutput)
	if err != nil {
		return err
	}

	return outputCheckResults(allFindings, useStructuredOutput)
}

func printNoFilesMessage() {
	if changed {
		fmt.Println("No changed HCL files found")
	} else {
		fmt.Println("No HCL files found")
	}
}

func printCheckHeader(fileCount int) {
	modeMsg := ""
	if changed {
		modeMsg = " (changed files only)"
	}
	fmt.Printf("Checking %s%s...\n\n", formatFileCount(fileCount), modeMsg)
}

func runAllChecks(files []string, quiet bool) ([]sdk.Finding, error) {
	ctx := context.Background()

	if checkParallel {
		return runAllChecksParallel(ctx, files, quiet)
	}
	return runAllChecksSequential(ctx, files, quiet)
}

func runAllChecksParallel(ctx context.Context, files []string, quiet bool) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Println("Running checks in parallel mode...")
	}

	r := runner.New().SetParallel(true)

	if !checkSkipFmt {
		r.AddEngine(fmtengine.New(&fmtengine.Config{Check: true}))
	}
	if !checkSkipStyle {
		r.AddEngine(style.New(&style.Config{Fix: false, Rules: make(map[string]style.RuleConfig)}))
	}
	if !checkSkipLint {
		r.AddEngine(lint.New(&lint.Config{ConfigFile: ".tflint.hcl"}))
	}
	if !checkSkipPolicy {
		r.AddEngine(policy.New(&policy.Config{}))
	}

	results := r.RunWithResults(ctx, files)

	var allFindings []sdk.Finding
	for _, result := range results {
		if result.Error != nil {
			return nil, fmt.Errorf("%s check failed: %w", result.Engine, result.Error)
		}
		if !quiet {
			fmt.Printf("  %s: %d issue(s)\n", result.Engine, len(result.Findings))
		}
		allFindings = append(allFindings, result.Findings...)
	}
	if !quiet {
		fmt.Println()
	}

	return allFindings, nil
}

func runAllChecksSequential(ctx context.Context, files []string, quiet bool) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding
	step := 1

	if !checkSkipFmt {
		findings, err := runFmtCheck(ctx, files, step, quiet)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		step++
	}

	if !checkSkipStyle {
		findings, err := runStyleCheck(ctx, files, step, quiet)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		step++
	}

	if !checkSkipLint {
		findings, err := runLintCheck(ctx, files, step, quiet)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		step++
	}

	if !checkSkipPolicy {
		findings, err := runPolicyCheck(ctx, files, step, quiet)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

func runFmtCheck(ctx context.Context, files []string, step int, quiet bool) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Checking formatting...\n", step)
	}
	fmtEngine := fmtengine.New(&fmtengine.Config{Check: true})
	findings, err := fmtEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("fmt check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

func runStyleCheck(ctx context.Context, files []string, step int, quiet bool) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Checking style...\n", step)
	}
	styleEngine := style.New(&style.Config{
		Fix:   false,
		Rules: make(map[string]style.RuleConfig),
	})
	findings, err := styleEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("style check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

func runLintCheck(ctx context.Context, files []string, step int, quiet bool) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Running linter...\n", step)
	}
	lintEngine := lint.New(&lint.Config{ConfigFile: ".tflint.hcl"})
	findings, err := lintEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("lint check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

func runPolicyCheck(ctx context.Context, files []string, step int, quiet bool) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Running policy checks...\n", step)
	}
	policyEngine := policy.New(&policy.Config{})
	findings, err := policyEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("policy check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

func outputCheckResults(allFindings []sdk.Finding, useStructuredOutput bool) error {
	if useStructuredOutput {
		return outputStructuredResults(allFindings)
	}
	return printCheckSummary(allFindings)
}

func outputStructuredResults(findings []sdk.Finding) error {
	formatter, err := output.GetFormatter(format, true, version)
	if err != nil {
		return fmt.Errorf("getting formatter: %w", err)
	}

	if err := formatter.Format(findings, os.Stdout); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}

	// Exit with error code if there are errors
	for _, finding := range findings {
		if finding.Severity == sdk.SeverityError {
			os.Exit(1)
		}
	}
	return nil
}

func printCheckSummary(allFindings []sdk.Finding) error {
	fmt.Println("---")
	fmt.Printf("Summary: %d total issue(s)\n", len(allFindings))

	if len(allFindings) == 0 {
		fmt.Println("All checks passed!")
		return nil
	}

	errors, warnings, info := countBySeverity(allFindings)
	printSeverityCounts(errors, warnings, info)
	printCheckHints()

	if errors > 0 {
		os.Exit(1)
	}
	return nil
}

func countBySeverity(findings []sdk.Finding) (errors, warnings, info int) {
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
	return
}

func printSeverityCounts(errors, warnings, info int) {
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
}

func printCheckHints() {
	fmt.Println()
	fmt.Println("Run individual commands for details:")
	fmt.Println("  terratidy fmt --check")
	fmt.Println("  terratidy style")
	fmt.Println("  terratidy lint")
	fmt.Println("  terratidy policy")
}
