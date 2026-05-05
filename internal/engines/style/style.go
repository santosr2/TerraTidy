// Package style implements the style-checking engine for HCL files.
package style

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/santosr2/TerraTidy/internal/annotations"
	"github.com/santosr2/TerraTidy/internal/config"
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
	Enabled  *bool
	Severity string
	Options  map[string]any
}

// IsEnabled returns whether the rule is enabled.
// If Enabled is nil (not explicitly set), returns defaultEnabled.
func (r RuleConfig) IsEnabled(defaultEnabled bool) bool {
	if r.Enabled == nil {
		return defaultEnabled
	}
	return *r.Enabled
}

// ConfigFromEngine creates a style.Config from the config package's StyleEngineConfig.
// This converts the typed config struct used for YAML parsing into the engine's
// internal Config type. CLI flag overrides (fix, diff) and plugin rules should be
// applied by the caller after this conversion.
func ConfigFromEngine(engineCfg config.StyleEngineConfig) *Config {
	cfg := &Config{
		Fix:   engineCfg.Fix,
		Diff:  engineCfg.Diff,
		Rules: make(map[string]RuleConfig),
	}

	for ruleName, ruleCfg := range engineCfg.Rules {
		cfg.Rules[ruleName] = RuleConfig{
			Enabled:  ruleCfg.Enabled,
			Severity: ruleCfg.Severity,
			Options:  ruleCfg.Config,
		}
	}

	return cfg
}

// New creates a new style engine.
// Plugin rules can be passed as optional additional arguments; they are appended after built-in rules.
func New(config *Config, pluginRules ...sdk.Rule) *Engine {
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

	// Append plugin rules after built-in rules
	engine.rules = append(engine.rules, pluginRules...)

	return engine
}

// Name returns the engine name
func (e *Engine) Name() string {
	return "style"
}

// Run executes the style engine on the given files
func (e *Engine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		findings, err := e.checkFile(file)
		if err != nil {
			return nil, fmt.Errorf("checking %s: %w", file, err)
		}

		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// checkFile checks a single file against all enabled rules
func (e *Engine) checkFile(path string) ([]sdk.Finding, error) {
	// Create context for rule execution
	ruleCtx := &sdk.Context{
		Context: context.Background(),
		Options: make(map[string]any),
		WorkDir: ".",
		File:    path,
	}

	// Capture original content before any fixes for diff generation
	// When Diff=true, we capture content regardless of Fix flag to support preview mode
	var originalContent []byte
	if e.config.Diff {
		var err error
		originalContent, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file for diff: %w", err)
		}
	}

	var allFindings []sdk.Finding
	var suppressions []annotations.Suppression
	// Hash-based fixed-point detection. The set of pre-fix file states we have
	// seen during this checkFile invocation. If a later pass reads content whose
	// hash is already in this set, we have cycled back to a state we have been
	// in before — typically a rule whose Fix() re-triggers its own finding, or
	// two rules ping-ponging the same content. Aborting on the cycle is the
	// right behavior: we emit a style.fix-loop error finding instead of either
	// silently truncating (the old 10-pass cap) or looping forever.
	seenHashes := make(map[[sha256.Size]byte]struct{})
	var lastAppliedRule string

	for pass := 0; ; pass++ {
		// Always read fresh content for each pass
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}

		contentHash := sha256.Sum256(content)
		if _, seen := seenHashes[contentHash]; seen {
			// The hash check runs before applying a fix this pass, so a repeat
			// implies that at least one prior pass applied a fix — therefore
			// lastAppliedRule is always populated when this branch fires.
			allFindings = append(allFindings, sdk.Finding{
				Rule: "style.fix-loop",
				Message: fmt.Sprintf(
					"fix loop detected: applying %q reproduced a previously-seen file state. "+
						"This is the last rule applied before the cycle was detected; another "+
						"rule may also be involved in a ping-pong cycle. Aborting to avoid an "+
						"infinite fix loop.",
					lastAppliedRule,
				),
				File:     path,
				Severity: sdk.SeverityError,
			})
			break
		}
		seenHashes[contentHash] = struct{}{}

		// Parse suppression annotations on first pass
		if pass == 0 {
			suppressions = annotations.Parse(content)
		}

		// Create fresh parser for each pass to avoid cached results
		parser := hclparse.NewParser()
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
			}}, nil
		}

		if file == nil {
			return []sdk.Finding{{
				Rule:     "style.parse-error",
				Message:  "Failed to parse file: unknown error",
				File:     path,
				Severity: sdk.SeverityError,
			}}, nil
		}

		findings, err := e.runEnabledRules(ruleCtx, file)
		if err != nil {
			return nil, err
		}

		// On first pass, collect all findings for reporting
		if pass == 0 {
			// Filter out suppressed findings based on annotations
			allFindings = annotations.FilterFindings(findings, suppressions)
		}

		// In fix mode or diff preview mode, apply fixes and potentially loop for another pass
		// When Diff=true with Fix=false, we still apply fixes to generate preview diff
		if (e.config.Fix || e.config.Diff) && len(findings) > 0 {
			appliedRule, err := e.applyFixes(ruleCtx, file, findings)
			if err != nil {
				return nil, fmt.Errorf("applying fixes: %w", err)
			}

			// If we applied a fix, loop again to catch any new issues
			if appliedRule != "" {
				lastAppliedRule = appliedRule
				continue
			}
		}

		// No fixes applied or not in fix mode, we're done
		break
	}

	// Generate diff if requested (works in both fix mode and preview mode)
	if e.config.Diff && originalContent != nil {
		// In preview mode (Diff=true, Fix=false), always restore the original content
		// even if diff generation fails - the preview contract guarantees no file changes
		if !e.config.Fix {
			defer func() {
				// Restore is best-effort in defer; primary error takes precedence
				_ = os.WriteFile(path, originalContent, 0o600)
			}()
		}

		diffFinding, err := e.generateDiff(path, originalContent)
		if err != nil {
			return nil, err
		}
		if diffFinding != nil {
			allFindings = append(allFindings, *diffFinding)
		}
	}

	return allFindings, nil
}

// runEnabledRules invokes Check() on every enabled rule against the parsed file
// and returns the aggregated findings. It applies the per-rule severity override
// from config and stamps Fixable based on whether the rule implements sdk.Fixer.
func (e *Engine) runEnabledRules(ruleCtx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding
	for _, rule := range e.rules {
		ruleConfig := e.getRuleConfig(rule.Name())
		if !ruleConfig.IsEnabled(true) { // Default enabled if not explicitly set
			continue
		}

		ruleCtx.Options = ruleConfig.Options

		ruleFindings, err := rule.Check(ruleCtx, file)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.Name(), err)
		}

		if ruleConfig.Severity != "" {
			for i := range ruleFindings {
				ruleFindings[i].Severity = sdk.ParseSeverity(ruleConfig.Severity, ruleFindings[i].Severity)
			}
		}

		// Stamp Fixable: engine owns the signal, not the rule. Set true for Fixer
		// rules (engine dispatches to Fixer.Fix(ctx, file) lazily in applyFixes);
		// false otherwise, regardless of what the rule may have set.
		_, isFixer := rule.(sdk.Fixer)
		for i := range ruleFindings {
			ruleFindings[i].Fixable = isFixer
		}

		findings = append(findings, ruleFindings...)
	}
	return findings, nil
}

// generateDiff creates a diff finding comparing original content with the current file.
func (e *Engine) generateDiff(path string, originalContent []byte) (*sdk.Finding, error) {
	fixedContent, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fixed file for diff: %w", err)
	}

	if bytes.Equal(originalContent, fixedContent) {
		return nil, nil
	}

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
	if diffText == "" {
		return nil, nil
	}

	return &sdk.Finding{
		Rule:     "style.diff",
		Message:  diffText,
		File:     path,
		Severity: sdk.SeverityInfo,
		IsDiff:   true,
	}, nil
}

// applyFixes applies ONE auto-fix to the file per pass.
// Returns the name of the rule whose fix was applied, or "" if no fix ran.
//
// Dispatch: for each finding marked Fixable, look up the originating rule by name
// and invoke its Fixer.Fix(ctx, file) method. The first non-nil byte content
// returned wins; it is written to disk and the rule name is returned. The
// multi-pass loop in checkFile re-reads the file and re-runs rules after each
// fix, so subsequent fixes are computed against the updated content.
//
// Note: each rule's Fix() must converge — repeated applications must eventually
// reach a fixed point. If the file returns to a previously-seen state, the
// multi-pass loop's hash-based cycle detector fires and emits a style.fix-loop
// error finding instead of looping forever.
func (e *Engine) applyFixes(ctx *sdk.Context, file *hcl.File, findings []sdk.Finding) (string, error) {
	for i := range findings {
		if !findings[i].Fixable {
			continue
		}

		fixer := e.findFixerByRuleName(findings[i].Rule)
		if fixer == nil {
			continue
		}

		content, err := fixer.Fix(ctx, file)
		if err != nil {
			return "", fmt.Errorf("computing fix for %s: %w", findings[i].Rule, err)
		}
		if content == nil {
			continue
		}

		if err := os.WriteFile(ctx.File, content, 0o600); err != nil {
			return "", fmt.Errorf("writing fix for %s: %w", findings[i].Rule, err)
		}
		return findings[i].Rule, nil
	}

	return "", nil
}

// findFixerByRuleName returns the rule registered under the given name as an
// sdk.Fixer, or nil if no rule with that name is registered or it does not
// implement Fixer.
func (e *Engine) findFixerByRuleName(name string) sdk.Fixer {
	for _, r := range e.rules {
		if r.Name() != name {
			continue
		}
		if f, ok := r.(sdk.Fixer); ok {
			return f
		}
		return nil
	}
	return nil
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
			Enabled:  config.BoolPtr(false),
			Severity: "info",
			Options:  make(map[string]any),
		}
	}

	// Return default config (enabled by default)
	return RuleConfig{
		Enabled:  config.BoolPtr(true),
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
