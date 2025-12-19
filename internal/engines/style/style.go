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
	// Read and parse the file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var file *hcl.File
	var diags hcl.Diagnostics

	// Try parsing as HCL first
	file, diags = parser.ParseHCL(content, path)
	if diags.HasErrors() {
		// If that fails, try as JSON (for .tf.json files)
		file, diags = parser.ParseJSON(content, path)
		if diags.HasErrors() {
			// If both fail, return a parsing error finding
			return []sdk.Finding{{
				Rule:     "style.parse-error",
				Message:  fmt.Sprintf("Failed to parse file: %s", diags.Error()),
				File:     path,
				Severity: sdk.SeverityError,
				Fixable:  false,
			}}, nil
		}
	}

	// Create context for rule execution
	ruleCtx := &sdk.Context{
		Config:  make(map[string]interface{}),
		WorkDir: ".",
		File:    path,
	}

	// Run all enabled rules
	var findings []sdk.Finding
	for _, rule := range e.rules {
		ruleConfig := e.getRuleConfig(rule.Name())
		if !ruleConfig.Enabled {
			continue
		}

		// Set rule config in context
		ruleCtx.Config = ruleConfig.Options

		// Check the rule
		ruleFindings, err := rule.Check(ruleCtx, file)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.Name(), err)
		}

		findings = append(findings, ruleFindings...)
	}

	// In fix mode, apply fixes
	if e.config.Fix && len(findings) > 0 {
		if err := e.applyFixes(ruleCtx, file, findings); err != nil {
			return nil, fmt.Errorf("applying fixes: %w", err)
		}
	}

	return findings, nil
}

// applyFixes applies auto-fixes to the file in one optimized pass.
func (e *Engine) applyFixes(ctx *sdk.Context, _ *hcl.File, findings []sdk.Finding) error {
	// Group findings by fixability
	var fixableFindings []sdk.Finding
	for _, f := range findings {
		if f.Fixable && f.FixFunc != nil {
			fixableFindings = append(fixableFindings, f)
		}
	}

	if len(fixableFindings) == 0 {
		return nil
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

	// Apply fixes in a single pass by chaining transformations
	// Read the file once
	content, err := os.ReadFile(ctx.File)
	if err != nil {
		return fmt.Errorf("reading file for fixes: %w", err)
	}

	// Apply each unique fix
	// Note: Since fixes operate on file content, we apply them sequentially
	// but only write to disk once at the end
	for _, finding := range uniqueFindings {
		fixed, err := finding.FixFunc()
		if err != nil {
			return fmt.Errorf("fixing %s: %w", finding.Rule, err)
		}

		// Update content for next iteration
		content = fixed
	}

	// Write the final fixed content back to the file once
	if err := os.WriteFile(ctx.File, content, 0o644); err != nil {
		return fmt.Errorf("writing fixed file: %w", err)
	}

	return nil
}

// getRuleConfig returns the configuration for a rule
func (e *Engine) getRuleConfig(ruleName string) RuleConfig {
	if cfg, ok := e.config.Rules[ruleName]; ok {
		return cfg
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
	// Block spacing and structure
	e.rules = append(e.rules, &BlankLineBetweenBlocksRule{})
	e.rules = append(e.rules, &NoEmptyBlocksRule{})

	// Naming conventions
	e.rules = append(e.rules, &BlockLabelCaseRule{})

	// Block ordering
	e.rules = append(e.rules, &TerraformBlockFirstRule{})
	e.rules = append(e.rules, &ProviderBlockOrderRule{})

	// Attribute ordering within blocks
	e.rules = append(e.rules, &ForEachCountFirstRule{})
	e.rules = append(e.rules, &SourceVersionGroupedRule{})
	e.rules = append(e.rules, &TagsAtEndRule{})
	e.rules = append(e.rules, &DependsOnOrderRule{})
	e.rules = append(e.rules, &LifecycleAtEndRule{})

	// Variable and output ordering
	e.rules = append(e.rules, &VariableOrderRule{})
	e.rules = append(e.rules, &OutputOrderRule{})
}

// GetAllRules returns all registered rules for listing/documentation
func (e *Engine) GetAllRules() []sdk.Rule {
	return e.rules
}
