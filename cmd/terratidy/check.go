// Package main provides the check command for TerraTidy.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/santosr2/TerraTidy/internal/config"
	fmtengine "github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/lint"
	"github.com/santosr2/TerraTidy/internal/engines/policy"
	"github.com/santosr2/TerraTidy/internal/engines/style"
	"github.com/santosr2/TerraTidy/internal/output"
	"github.com/santosr2/TerraTidy/internal/plugins"
	"github.com/santosr2/TerraTidy/internal/runner"
	"github.com/santosr2/TerraTidy/pkg/sdk"
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
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Load plugin rules if plugins are enabled
	pluginRules, err := loadPluginRules(cfg)
	if err != nil {
		return fmt.Errorf("loading plugins: %w", err)
	}

	files, err := getTargetFilesWithExcludes(args, changed, cfg.Exclude)
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

	allFindings, err := runAllChecksWithConfig(cfg, files, useStructuredOutput, pluginRules)
	if err != nil {
		return err
	}

	// Apply severity threshold filtering
	threshold := getEffectiveSeverityThreshold(cfg)
	allFindings = filterFindingsBySeverity(allFindings, threshold)

	return outputCheckResults(allFindings, useStructuredOutput)
}

// loadPluginRules loads plugin rules from the configured plugin directories.
// Returns an empty slice if plugins are not enabled.
// Filters rules based on plugins.rules config (enabled/disabled).
// Severity overrides are applied by the style engine via buildStyleConfig.
// TaggedRule is an optional interface for rules that have tags.
type TaggedRule interface {
	Tags() []string
}

func loadPluginRules(cfg *config.Config) ([]sdk.Rule, error) {
	if cfg == nil || !cfg.Plugins.Enabled {
		return nil, nil
	}

	mgr := plugins.NewManager(cfg.Plugins.Directories, cfg.Plugins.ShouldVerifyIntegrity())
	if err := mgr.LoadAll(); err != nil {
		return nil, err
	}

	// Convert map to slice, filtering disabled rules and by tags
	rulesMap := mgr.GetRules()
	rules := make([]sdk.Rule, 0, len(rulesMap))
	for _, rule := range rulesMap {
		// Check if rule is disabled in config
		if ruleConfig, exists := cfg.Plugins.Rules[rule.Name()]; exists {
			if !ruleConfig.Enabled {
				continue // Skip disabled rules
			}
		}

		// Filter by tags if configured
		if len(cfg.Plugins.Tags) > 0 {
			if taggedRule, ok := rule.(TaggedRule); ok {
				if !hasMatchingTag(taggedRule.Tags(), cfg.Plugins.Tags) {
					continue // Skip rules that don't have any of the required tags
				}
			} else {
				// Rule doesn't support tags, skip it when tag filter is active
				continue
			}
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// hasMatchingTag returns true if any tag in ruleTags matches any tag in filterTags.
func hasMatchingTag(ruleTags, filterTags []string) bool {
	for _, rt := range ruleTags {
		for _, ft := range filterTags {
			if rt == ft {
				return true
			}
		}
	}
	return false
}

func printCheckHeader(fileCount int) {
	modeMsg := ""
	if changed {
		modeMsg = " (changed files only)"
	}
	fmt.Printf("Checking %s%s...\n\n", formatFileCount(fileCount), modeMsg)
}

func runAllChecksWithConfig(cfg *config.Config, files []string, quiet bool, pluginRules []sdk.Rule) ([]sdk.Finding, error) {
	ctx := context.Background()

	// Determine if parallel execution should be used
	useParallel := getEffectiveParallel(cfg, checkParallel)

	if useParallel {
		return runAllChecksParallelWithConfig(ctx, cfg, files, quiet, pluginRules)
	}
	return runAllChecksSequentialWithConfig(ctx, cfg, files, quiet, pluginRules)
}

func runAllChecksParallelWithConfig(ctx context.Context, cfg *config.Config, files []string, quiet bool, pluginRules []sdk.Rule) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Println("Running checks in parallel mode...")
	}

	r := runner.New().SetParallel(true)

	// Check if engines are enabled (CLI skip flags override config)
	if !checkSkipFmt && isEngineEnabled(cfg, "fmt") {
		r.AddEngine(fmtengine.New(&fmtengine.Config{Check: true}))
	}
	if !checkSkipStyle && isEngineEnabled(cfg, "style") {
		r.AddEngine(style.New(buildStyleConfig(cfg, false), pluginRules...))
	}
	if !checkSkipLint && isEngineEnabled(cfg, "lint") {
		r.AddEngine(lint.New(buildLintConfig(cfg), pluginRules...))
	}
	if !checkSkipPolicy && isEngineEnabled(cfg, "policy") {
		r.AddEngine(policy.New(buildPolicyConfig(cfg)))
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

func runAllChecksSequentialWithConfig(ctx context.Context, cfg *config.Config, files []string, quiet bool, pluginRules []sdk.Rule) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding
	step := 1
	failFast := shouldFailFast(cfg)

	if !checkSkipFmt && isEngineEnabled(cfg, "fmt") {
		findings, err := runFmtCheckWithConfig(ctx, cfg, files, step, quiet)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		step++

		// Fail fast if enabled and there are errors
		if failFast && hasErrors(findings) {
			return allFindings, nil
		}
	}

	if !checkSkipStyle && isEngineEnabled(cfg, "style") {
		findings, err := runStyleCheckWithConfig(ctx, cfg, files, step, quiet, pluginRules)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		step++

		if failFast && hasErrors(findings) {
			return allFindings, nil
		}
	}

	if !checkSkipLint && isEngineEnabled(cfg, "lint") {
		findings, err := runLintCheckWithConfig(ctx, cfg, files, step, quiet, pluginRules)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
		step++

		if failFast && hasErrors(findings) {
			return allFindings, nil
		}
	}

	if !checkSkipPolicy && isEngineEnabled(cfg, "policy") {
		findings, err := runPolicyCheckWithConfig(ctx, cfg, files, step, quiet)
		if err != nil {
			return nil, err
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// hasErrors returns true if any finding has error severity.
func hasErrors(findings []sdk.Finding) bool {
	for i := range findings {
		if findings[i].Severity == sdk.SeverityError {
			return true
		}
	}
	return false
}

func runFmtCheckWithConfig(ctx context.Context, _ *config.Config, files []string, step int, quiet bool) ([]sdk.Finding, error) {
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

func runStyleCheckWithConfig(ctx context.Context, cfg *config.Config, files []string, step int, quiet bool, pluginRules []sdk.Rule) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Checking style...\n", step)
	}
	styleEngine := style.New(buildStyleConfig(cfg, false), pluginRules...)
	findings, err := styleEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("style check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

func runLintCheckWithConfig(ctx context.Context, cfg *config.Config, files []string, step int, quiet bool, pluginRules []sdk.Rule) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Running linter...\n", step)
	}
	lintEngine := lint.New(buildLintConfig(cfg), pluginRules...)
	findings, err := lintEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("lint check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

func runPolicyCheckWithConfig(ctx context.Context, cfg *config.Config, files []string, step int, quiet bool) ([]sdk.Finding, error) {
	if !quiet {
		fmt.Printf("%d. Running policy checks...\n", step)
	}
	policyEngine := policy.New(buildPolicyConfig(cfg))
	findings, err := policyEngine.Run(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("policy check failed: %w", err)
	}
	if !quiet {
		fmt.Printf("   Found %d issue(s)\n\n", len(findings))
	}
	return findings, nil
}

// buildStyleConfig creates a style.Config from the terratidy config.
// CLI flags (fix, diff) take precedence over config file values.
func buildStyleConfig(cfg *config.Config, fix bool, diff ...bool) *style.Config {
	if cfg == nil {
		return &style.Config{
			Fix:   fix,
			Diff:  len(diff) > 0 && diff[0],
			Rules: make(map[string]style.RuleConfig),
		}
	}

	// Use engine's ConfigFromEngine for base conversion
	styleCfg := style.ConfigFromEngine(cfg.Engines.Style)

	// Apply CLI flag overrides (CLI takes precedence)
	if fix {
		styleCfg.Fix = true
	}
	if len(diff) > 0 && diff[0] {
		styleCfg.Diff = true
	}

	// Merge override rules (overrides take precedence over engine config)
	for ruleName, ruleCfg := range cfg.Overrides.Rules {
		styleCfg.Rules[ruleName] = style.RuleConfig{
			Enabled:  ruleCfg.Enabled,
			Severity: ruleCfg.Severity,
			Options:  ruleCfg.Config,
		}
	}

	// Merge plugin rule configs (plugins.rules takes precedence for plugin rules)
	for ruleName, ruleCfg := range cfg.Plugins.Rules {
		styleCfg.Rules[ruleName] = style.RuleConfig{
			Enabled:  ruleCfg.Enabled,
			Severity: ruleCfg.Severity,
			Options:  ruleCfg.Config,
		}
	}

	return styleCfg
}

// buildLintConfig creates a lint.Config from the terratidy config.
func buildLintConfig(cfg *config.Config) *lint.Config {
	if cfg == nil {
		return &lint.Config{
			ConfigFile: ".tflint.hcl",
			Rules:      make(map[string]lint.RuleConfig),
		}
	}

	// Use engine's ConfigFromEngine for base conversion
	lintCfg := lint.ConfigFromEngine(cfg.Engines.Lint)

	// Merge override rules (overrides take precedence over engine config)
	for ruleName, ruleCfg := range cfg.Overrides.Rules {
		lintCfg.Rules[ruleName] = lint.RuleConfig{
			Enabled:  ruleCfg.Enabled,
			Severity: ruleCfg.Severity,
			Options:  ruleCfg.Config,
		}
	}

	// Merge plugin rule configs (plugins.rules takes precedence for plugin rules)
	for ruleName, ruleCfg := range cfg.Plugins.Rules {
		lintCfg.Rules[ruleName] = lint.RuleConfig{
			Enabled:  ruleCfg.Enabled,
			Severity: ruleCfg.Severity,
			Options:  ruleCfg.Config,
		}
	}

	return lintCfg
}

// buildPolicyConfig creates a policy.Config from the terratidy config.
func buildPolicyConfig(cfg *config.Config) *policy.Config {
	if cfg == nil {
		return &policy.Config{
			PolicyDirs:  []string{},
			PolicyFiles: []string{},
			Rules:       make(map[string]policy.RuleConfig),
		}
	}

	// Use engine's ConfigFromEngine for conversion
	return policy.ConfigFromEngine(cfg.Engines.Policy)
}

func outputCheckResults(allFindings []sdk.Finding, _ bool) error {
	formatter, err := output.GetFormatterWithColor(format, true, version, color)
	if err != nil {
		return fmt.Errorf("getting formatter: %w", err)
	}

	if err := formatter.Format(allFindings, os.Stdout); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}

	// For text format, add summary
	if format == "" || format == "text" {
		return printCheckSummary(allFindings)
	}

	// Return exit error if there are errors (for structured output)
	for i := range allFindings {
		if allFindings[i].Severity == sdk.SeverityError {
			return &sdk.ExitError{Code: 1}
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
		return &sdk.ExitError{Code: 1}
	}
	return nil
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
