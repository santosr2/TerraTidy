package style

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// Engine represents the style engine
type Engine struct {
	config *Config
	rules  []sdk.Rule
}

// Config holds the style engine configuration
type Config struct {
	Fix   bool // Auto-fix mode
	Rules map[string]RuleConfig
}

// RuleConfig holds configuration for a single rule
type RuleConfig struct {
	Enabled  bool
	Severity string
	Options  map[string]interface{}
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
		Config:  make(map[string]interface{}),
		WorkDir: ".",
		File:    path,
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

	return allFindings, nil
}

// applyFixes applies auto-fixes to the file in one optimized pass.
// Returns the number of fixes applied.
func (e *Engine) applyFixes(ctx *sdk.Context, _ *hcl.File, findings []sdk.Finding) (int, error) {
	// Group findings by fixability
	var fixableFindings []sdk.Finding
	for _, f := range findings {
		if f.Fixable && f.FixFunc != nil {
			fixableFindings = append(fixableFindings, f)
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
			if err := os.WriteFile(ctx.File, fixed, 0o644); err != nil {
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

	// File organization rules are disabled by default (opt-in)
	disabledByDefault := map[string]bool{
		"style.variables-in-file": true,
		"style.outputs-in-file":   true,
		"style.providers-in-file": true,
	}

	if disabledByDefault[ruleName] {
		return RuleConfig{
			Enabled:  false,
			Severity: "info",
			Options:  make(map[string]interface{}),
		}
	}

	// Return default config (enabled by default)
	return RuleConfig{
		Enabled:  true,
		Severity: "warning",
		Options:  make(map[string]interface{}),
	}
}

// registerRules registers all built-in style rules
func (e *Engine) registerRules() {
	// Block spacing between blocks
	e.rules = append(e.rules, &BlankLineBetweenBlocksRule{})

	// Naming conventions
	e.rules = append(e.rules, &BlockLabelCaseRule{})
	e.rules = append(e.rules, &VariableNamingRule{})
	e.rules = append(e.rules, &OutputNamingRule{})
	e.rules = append(e.rules, &LocalNamingRule{})

	// Block ordering
	e.rules = append(e.rules, &TerraformBlockFirstRule{})
	e.rules = append(e.rules, &ProviderBlockOrderRule{})

	// Attribute ordering within blocks (runs first to reorder attributes)
	e.rules = append(e.rules, &ForEachCountFirstRule{})
	e.rules = append(e.rules, &SourceVersionGroupedRule{})
	e.rules = append(e.rules, &TagsAtEndRule{})
	e.rules = append(e.rules, &DependsOnOrderRule{})
	e.rules = append(e.rules, &LifecycleAtEndRule{})

	// Variable and output ordering
	e.rules = append(e.rules, &VariableOrderRule{})
	e.rules = append(e.rules, &OutputOrderRule{})

	// Attribute group spacing (runs after ordering to add blank lines between groups)
	e.rules = append(e.rules, &AttributeGroupSpacingRule{})

	// Cleanup rules (run last to fix any blank line issues from reordering)
	e.rules = append(e.rules, &NoLeadingTrailingBlankLinesRule{})
	e.rules = append(e.rules, &NoEmptyBlocksRule{})

	// File organization rules (disabled by default - enable via config)
	e.rules = append(e.rules, &VariablesInFileRule{})
	e.rules = append(e.rules, &OutputsInFileRule{})
	e.rules = append(e.rules, &ProvidersInFileRule{})
}

// GetAllRules returns all registered rules for listing/documentation
func (e *Engine) GetAllRules() []sdk.Rule {
	return e.rules
}
