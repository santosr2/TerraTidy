// Package style implements the style-checking engine for HCL files.
package style

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/santosr2/TerraTidy/internal/engines/style/rules"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// Engine represents the style engine
type Engine struct {
	config *Config
	rules  []sdk.Rule
}

// Config holds the style engine configuration
type Config struct {
	Fix   bool // Auto-fix mode
	Diff  bool // Show diff of changes
	Rules map[string]RuleConfig
}

// RuleConfig holds configuration for a single rule
type RuleConfig struct {
	Enabled  bool
	Severity string
	Options  map[string]any
}

// New creates a new style engine
func New(config *Config) *Engine {
	if config == nil {
		config = &Config{
			Rules: make(map[string]RuleConfig),
		}
	}

	engine := &Engine{
		config: config,
		rules:  []sdk.Rule{},
	}

	// Register built-in rules
	engine.registerRules()

	return engine
}

// Name returns the engine name
func (e *Engine) Name() string {
	return "style"
}

// Run executes the style engine on the given files
func (e *Engine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding

	parser := hclparse.NewParser()

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		findings, err := e.checkFile(parser, file)
		if err != nil {
			return nil, fmt.Errorf("checking %s: %w", file, err)
		}

		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// checkFile checks a single file against all enabled rules
func (e *Engine) checkFile(parser *hclparse.Parser, path string) ([]sdk.Finding, error) {
	// Create context for rule execution
	ruleCtx := &sdk.Context{
		Config:  make(map[string]any),
		WorkDir: ".",
		File:    path,
	}

	// Capture original content before any fixes for diff generation
	var originalContent []byte
	if e.config.Diff && e.config.Fix {
		var err error
		originalContent, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file for diff: %w", err)
		}
	}

	var allFindings []sdk.Finding
	maxPasses := 3 // Limit passes to prevent infinite loops

	for pass := 0; pass < maxPasses; pass++ {
		// Always read fresh content for each pass
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}

		// Parse fresh for each pass
		file, diags := parser.ParseHCL(content, path)
		if diags.HasErrors() {
			// Try as JSON for .tf.json files
			file, diags = parser.ParseJSON(content, path)
		}

		if diags.HasErrors() {
			return []sdk.Finding{{
				Rule:     "style.parse-error",
				Message:  fmt.Sprintf("Failed to parse file: %s", diags.Error()),
				File:     path,
				Severity: sdk.SeverityError,
				Fixable:  false,
			}}, nil
		}

		if file == nil {
			return []sdk.Finding{{
				Rule:     "style.parse-error",
				Message:  "Failed to parse file: unknown error",
				File:     path,
				Severity: sdk.SeverityError,
				Fixable:  false,
			}}, nil
		}

		// Run all enabled rules
		var findings []sdk.Finding
		for _, rule := range e.rules {
			ruleConfig := e.getRuleConfig(rule.Name())
			if !ruleConfig.Enabled {
				continue
			}

			ruleCtx.Config = ruleConfig.Options

			ruleFindings, err := rule.Check(ruleCtx, file)
			if err != nil {
				return nil, fmt.Errorf("rule %s: %w", rule.Name(), err)
			}

			// Apply severity override from config if specified
			if ruleConfig.Severity != "" {
				for i := range ruleFindings {
					ruleFindings[i].Severity = sdk.ParseSeverity(ruleConfig.Severity, ruleFindings[i].Severity)
				}
			}

			findings = append(findings, ruleFindings...)
		}

		// On first pass, collect all findings for reporting
		if pass == 0 {
			allFindings = findings
		}

		// In fix mode, apply fixes and potentially loop for another pass
		if e.config.Fix && len(findings) > 0 {
			fixedCount, err := e.applyFixes(ruleCtx, file, findings)
			if err != nil {
				return nil, fmt.Errorf("applying fixes: %w", err)
			}

			// If we applied fixes, continue to next pass to catch any new issues
			if fixedCount > 0 {
				continue
			}
		}

		// No fixes applied or not in fix mode, we're done
		break
	}

	// Generate diff if requested and fixes were applied
	if e.config.Diff && e.config.Fix && originalContent != nil {
		fixedContent, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading fixed file for diff: %w", err)
		}

		if !bytes.Equal(originalContent, fixedContent) {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(originalContent)),
				B:        difflib.SplitLines(string(fixedContent)),
				FromFile: path,
				ToFile:   path,
				Context:  3,
			}
			diffText, err := difflib.GetUnifiedDiffString(diff)
			if err != nil {
				return nil, fmt.Errorf("generating diff: %w", err)
			}
			if diffText != "" {
				allFindings = append(allFindings, sdk.Finding{
					Rule:     "style.diff",
					Message:  diffText,
					File:     path,
					Severity: sdk.SeverityInfo,
					Fixable:  false,
				})
			}
		}
	}

	return allFindings, nil
}

// applyFixes applies auto-fixes to the file in one optimized pass.
// Returns the number of fixes applied.
func (e *Engine) applyFixes(ctx *sdk.Context, _ *hcl.File, findings []sdk.Finding) (int, error) {
	// Group findings by fixability
	var fixableFindings []sdk.Finding
	for i := range findings {
		if findings[i].Fixable && findings[i].FixFunc != nil {
			fixableFindings = append(fixableFindings, findings[i])
		}
	}

	if len(fixableFindings) == 0 {
		return 0, nil
	}

	// Deduplicate findings by rule to avoid redundant fixes
	// Many findings might result in the same fix operation
	seenRules := make(map[string]bool)
	var uniqueFindings []sdk.Finding
	for _, f := range fixableFindings {
		if !seenRules[f.Rule] {
			seenRules[f.Rule] = true
			uniqueFindings = append(uniqueFindings, f)
		}
	}

	// Apply each unique fix sequentially
	// Each FixFunc reads from disk, so we write intermediate results
	fixedCount := 0
	for _, finding := range uniqueFindings {
		fixed, err := finding.FixFunc()
		if err != nil {
			return fixedCount, fmt.Errorf("fixing %s: %w", finding.Rule, err)
		}

		// Write intermediate result so next FixFunc sees updated content
		if fixed != nil {
			if err := os.WriteFile(ctx.File, fixed, 0o600); err != nil {
				return fixedCount, fmt.Errorf("writing intermediate fix: %w", err)
			}
			fixedCount++
		}
	}

	return fixedCount, nil
}

// getRuleConfig returns the configuration for a rule
func (e *Engine) getRuleConfig(ruleName string) RuleConfig {
	if cfg, ok := e.config.Rules[ruleName]; ok {
		return cfg
	}

	// Rules that are disabled by default (opt-in)
	disabledByDefault := map[string]bool{
		// File organization rules
		"style.variables-in-file":         true,
		"style.outputs-in-file":           true,
		"style.providers-in-file":         true,
		"style.scoped-file-organization":  true,
		"style.terraform-files-structure": true,
		// Advanced naming rules (can be noisy)
		"style.resource-name-matches-type": true,
		"style.output-prefix":              true,
		"style.module-name-convention":     true,
		// Comment and format rules
		"style.comment-syntax":             true,
		"style.no-trailing-whitespace":     true,
		"style.consistent-quotes":          true,
		"style.no-consecutive-blank-lines": true,
		// Block organization rules (informational)
		"style.meta-arguments-order":       true,
		"style.lifecycle-attribute-order":  true,
		"style.nested-block-order":         true,
		"style.one-line-attribute-spacing": true,
	}

	if disabledByDefault[ruleName] {
		return RuleConfig{
			Enabled:  false,
			Severity: "info",
			Options:  make(map[string]any),
		}
	}

	// Return default config (enabled by default)
	return RuleConfig{
		Enabled:  true,
		Severity: "warning",
		Options:  make(map[string]any),
	}
}

// registerRules registers all built-in style rules
func (e *Engine) registerRules() {
	// Block spacing between blocks
	e.rules = append(e.rules, &rules.BlankLineBetweenBlocksRule{})

	// Naming conventions
	e.rules = append(e.rules, &rules.BlockLabelCaseRule{})
	e.rules = append(e.rules, &rules.VariableNamingRule{})
	e.rules = append(e.rules, &rules.OutputNamingRule{})
	e.rules = append(e.rules, &rules.LocalNamingRule{})

	// Block ordering
	e.rules = append(e.rules, &rules.TerraformBlockFirstRule{})
	e.rules = append(e.rules, &rules.ProviderBlockOrderRule{})

	// Attribute ordering within blocks (runs first to reorder attributes)
	e.rules = append(e.rules, &rules.ForEachCountFirstRule{})
	e.rules = append(e.rules, &rules.SourceVersionGroupedRule{})
	e.rules = append(e.rules, &rules.TagsAtEndRule{})
	e.rules = append(e.rules, &rules.DependsOnOrderRule{})
	e.rules = append(e.rules, &rules.LifecycleAtEndRule{})

	// Variable and output ordering
	e.rules = append(e.rules, &rules.VariableOrderRule{})
	e.rules = append(e.rules, &rules.OutputOrderRule{})

	// Attribute group spacing (runs after ordering to add blank lines between groups)
	e.rules = append(e.rules, &rules.AttributeGroupSpacingRule{})

	// Cleanup rules (run last to fix any blank line issues from reordering)
	e.rules = append(e.rules, &rules.NoLeadingTrailingBlankLinesRule{})
	e.rules = append(e.rules, &rules.NoEmptyBlocksRule{})

	// File organization rules (disabled by default - enable via config)
	e.rules = append(e.rules, &rules.VariablesInFileRule{})
	e.rules = append(e.rules, &rules.OutputsInFileRule{})
	e.rules = append(e.rules, &rules.ProvidersInFileRule{})
	e.rules = append(e.rules, &rules.ScopedFileOrganizationRule{})
	e.rules = append(e.rules, &rules.TerraformFilesStructureRule{})

	// Advanced naming rules (disabled by default - enable via config)
	e.rules = append(e.rules, &rules.ResourceNameMatchesTypeRule{})
	e.rules = append(e.rules, &rules.OutputPrefixRule{})
	e.rules = append(e.rules, &rules.ModuleNameConventionRule{})

	// Block organization rules (disabled by default - enable via config)
	e.rules = append(e.rules, &rules.MetaArgumentsOrderRule{})
	e.rules = append(e.rules, &rules.LifecycleAttributeOrderRule{})
	e.rules = append(e.rules, &rules.NestedBlockOrderRule{})
	e.rules = append(e.rules, &rules.OneLineAttributeSpacingRule{})

	// Comment and format rules (disabled by default - enable via config)
	e.rules = append(e.rules, &rules.CommentSyntaxRule{})
	e.rules = append(e.rules, &rules.NoTrailingWhitespaceRule{})
	e.rules = append(e.rules, &rules.ConsistentQuotesRule{})
	e.rules = append(e.rules, &rules.NoConsecutiveBlankLinesRule{})
}

// GetAllRules returns all registered rules for listing/documentation
func (e *Engine) GetAllRules() []sdk.Rule {
	return e.rules
}
