// Package style implements the style-checking engine for HCL files.
package style

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sort"

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
	// writeFn is the function used to persist fixed file content. Defaults to
	// os.WriteFile in New(); tests substitute a capturing closure that records
	// invocation count and the content written, so applyFixes's
	// one-write-per-pass invariant can be asserted directly.
	writeFn func(name string, data []byte, perm os.FileMode) error
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
		config:  config,
		rules:   []sdk.Rule{},
		writeFn: os.WriteFile,
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

	// Capture the original file mode once. All write paths below (preview
	// restore and per-pass fix apply) honor it. Falls back to 0o600 if Stat
	// fails so we still produce a sensibly-permissioned file.
	mode := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
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
	var lastAppliedRules []string

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
			// lastAppliedRules is always populated when this branch fires.
			allFindings = append(allFindings, hashCycleFixLoopFinding(path, lastAppliedRules))
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
			appliedRules, err := e.applyFixes(ruleCtx, file, findings, mode)
			if err != nil {
				return nil, fmt.Errorf("applying fixes: %w", err)
			}

			// If we applied any fixes, loop again to catch any new issues.
			if len(appliedRules) > 0 {
				lastAppliedRules = appliedRules
				continue
			}

			// applyFixes produced no edits despite Fixable findings being
			// present, which means a rule keeps reporting an issue it never
			// converges on. Surface it as a fix-loop instead of dropping it.
			if stuck := stuckRuleFixLoopFinding(findings, path); stuck != nil {
				allFindings = append(allFindings, *stuck)
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
				// Errors are silent by design: the preview contract prefers
				// returning the diff finding over surfacing restore failures.
				// Both calls preserve the captured original mode.
				_ = os.WriteFile(path, originalContent, mode)
				_ = os.Chmod(path, mode)
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

// collectStuckFixableRules returns the deduplicated, source-ordered set of
// rule names from findings that are marked Fixable. The checkFile loop calls
// this when applyFixes produced no edits despite Fixable findings being
// present, which signals a stuck-rule cycle (the rule keeps reporting an
// issue it never converges on). Returns nil when no fixable rule is present.
func collectStuckFixableRules(findings []sdk.Finding) []string {
	var stuck []string
	seen := make(map[string]struct{})
	for i := range findings {
		if !findings[i].Fixable {
			continue
		}
		name := findings[i].Rule
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		stuck = append(stuck, name)
	}
	return stuck
}

// hashCycleFixLoopFinding builds the fix-loop finding emitted when checkFile
// reads file content whose hash matches a previously-seen pass — the classic
// ping-pong (or self-undo) cycle the hash-based guard exists to catch.
// lastAppliedRules names every rule whose edits landed in the pass just
// before the cycle was detected.
func hashCycleFixLoopFinding(path string, lastAppliedRules []string) sdk.Finding {
	return sdk.Finding{
		Rule: "style.fix-loop",
		Message: fmt.Sprintf(
			"fix loop detected: applying %v reproduced a previously-seen file state. "+
				"These are the rules applied in the last pass before the cycle was detected. "+
				"Aborting to avoid an infinite fix loop.",
			lastAppliedRules,
		),
		File:     path,
		Severity: sdk.SeverityError,
	}
}

// stuckRuleFixLoopFinding builds the degenerate-fix-loop finding emitted when
// applyFixes returns no edits despite Fixable findings being reported. Returns
// nil in the common "no fix needed this pass" case so callers can branch on
// the return value without pre-checking. Severity mirrors the hash-collision
// fix-loop emission in checkFile.
func stuckRuleFixLoopFinding(findings []sdk.Finding, path string) *sdk.Finding {
	stuck := collectStuckFixableRules(findings)
	if len(stuck) == 0 {
		return nil
	}
	return &sdk.Finding{
		Rule: "style.fix-loop",
		Message: fmt.Sprintf(
			"fix loop detected: %v reported fixable findings but produced no edits this pass. "+
				"The file state did not converge. Aborting to avoid an infinite fix loop.",
			stuck,
		),
		File:     path,
		Severity: sdk.SeverityError,
	}
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

// applyFixes collects byte-range edits from every fixable finding, resolves
// overlaps, and applies the non-conflicting edits in a single write. It
// returns the deduplicated set of rule names whose edits were applied this
// pass — an empty slice when no fix ran.
//
// Algorithm (in order):
//  1. For each finding marked Fixable, invoke the originating rule's
//     Fixer.Fix(ctx, file) and collect every emitted TextEdit. A nil
//     FixResult or empty Edits slice contributes no edits; a Fix error
//     aborts the pass and propagates up.
//  2. Bounds-check every collected edit (Start >= 0, Start <= End,
//     End <= len(content)). An out-of-bounds edit returns an error naming
//     the offending rule.
//  3. Whole-file exclusivity: if any collected edit covers the full file
//     (Start == 0 && End == len(content)), apply it alone and discard every
//     other edit this pass. Whole-file ties resolve in source order. Narrow
//     edits re-emit against the rewritten content on the next pass.
//  4. Overlap resolution: walk collected edits in source order and keep an
//     edit iff it does not conflict with any already-kept edit. Two edits
//     conflict iff they share a Start offset or their half-open byte ranges
//     overlap. Dropped edits re-emit on the next pass.
//  5. Sort retained edits by Start in descending order, then splice each into
//     a fresh copy of the content. Descending order keeps the byte offsets
//     of remaining edits valid as earlier (right-side) splices land.
//  6. Write the spliced content with the captured mode, then defensively
//     Chmod to restore the mode in case WriteFile recreated the file.
//
// The hash-based fix-loop guard in checkFile catches rules that re-emit
// indefinitely against the updated content; this function is responsible
// only for converging a single pass's edits.
func (e *Engine) applyFixes(ctx *sdk.Context, file *hcl.File, findings []sdk.Finding, mode os.FileMode) ([]string, error) {
	type collectedEdit struct {
		rule string
		edit sdk.TextEdit
	}

	content := file.Bytes

	var collected []collectedEdit
	for i := range findings {
		if !findings[i].Fixable {
			continue
		}
		fixer := e.Fixer(findings[i].Rule)
		if fixer == nil {
			continue
		}
		result, err := fixer.Fix(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("computing fix for %s: %w", findings[i].Rule, err)
		}
		if result == nil {
			continue
		}
		ruleName := findings[i].Rule
		for _, te := range result.Edits {
			collected = append(collected, collectedEdit{rule: ruleName, edit: te})
		}
	}

	if len(collected) == 0 {
		return nil, nil
	}

	for _, ce := range collected {
		if ce.edit.Start < 0 {
			return nil, fmt.Errorf("fix from %s: edit start %d is negative", ce.rule, ce.edit.Start)
		}
		if ce.edit.End < ce.edit.Start {
			return nil, fmt.Errorf("fix from %s: edit end %d precedes start %d", ce.rule, ce.edit.End, ce.edit.Start)
		}
		if ce.edit.End > len(content) {
			return nil, fmt.Errorf("fix from %s: edit end %d exceeds content length %d", ce.rule, ce.edit.End, len(content))
		}
	}

	// Whole-file exclusivity requires End == len(content) — a zero-width
	// insertion at offset 0 (End == 0 on a non-empty file) does NOT qualify.
	for _, ce := range collected {
		if ce.edit.Start == 0 && ce.edit.End == len(content) {
			return e.writeFixed(ctx.File, ce.edit.Replacement, mode, []string{ce.rule})
		}
	}

	retained := make([]collectedEdit, 0, len(collected))
	for _, ce := range collected {
		conflict := false
		for _, kept := range retained {
			if editsConflict(ce.edit, kept.edit) {
				conflict = true
				break
			}
		}
		if !conflict {
			retained = append(retained, ce)
		}
	}

	sort.SliceStable(retained, func(i, j int) bool {
		return retained[i].edit.Start > retained[j].edit.Start
	})

	newContent := append([]byte(nil), content...)
	for _, ce := range retained {
		// Splice in three parts: newContent[:Start] + Replacement + newContent[End:].
		// Descending-Start ordering plus the no-overlap invariant guarantees
		// ce.edit.End <= len(newContent) at every iteration, so the right slice is
		// always valid.
		leftLen := ce.edit.Start
		rightLen := len(newContent) - ce.edit.End
		spliced := make([]byte, 0, leftLen+len(ce.edit.Replacement)+rightLen)
		spliced = append(spliced, newContent[:ce.edit.Start]...)
		spliced = append(spliced, ce.edit.Replacement...)
		spliced = append(spliced, newContent[ce.edit.End:]...)
		newContent = spliced
	}

	applied := make([]string, 0, len(retained))
	seen := make(map[string]struct{}, len(retained))
	for _, ce := range retained {
		if _, dup := seen[ce.rule]; dup {
			continue
		}
		seen[ce.rule] = struct{}{}
		applied = append(applied, ce.rule)
	}

	return e.writeFixed(ctx.File, newContent, mode, applied)
}

// editsConflict reports whether two edits cannot be co-applied in a single
// pass under the descending-Start splice rule. The relation is symmetric, so
// caller ordering does not matter. Conflicts come in three flavors:
//
//  1. Same-Start: no defined application order between them.
//  2. Half-open ranges share at least one byte position
//     (max(Start) < min(End)). Touching ranges (a.End == b.Start) do not
//     conflict.
//  3. One edit is zero-width (Start == End) and its offset lies strictly
//     inside the other edit's half-open range (open interval (Start, End)).
//     A zero-width insertion at a position about to be deleted is ambiguous:
//     the descending-Start splice produces a deterministic result, but the
//     insertion point's anchor is gone from the original document. The
//     conflict drops the insertion so it can re-anchor against the rewritten
//     content on the next pass. Boundary cases do not conflict in this
//     branch: a zero-width edit at the other's Start is caught by branch 1
//     (same-Start), and a zero-width edit at the other's End is the
//     adjacent-touching case (a.End == b.Start), which is always allowed.
func editsConflict(a, b sdk.TextEdit) bool {
	if a.Start == b.Start {
		return true
	}
	if max(a.Start, b.Start) < min(a.End, b.End) {
		return true
	}
	if a.Start == a.End && a.Start > b.Start && a.Start < b.End {
		return true
	}
	if b.Start == b.End && b.Start > a.Start && b.Start < a.End {
		return true
	}
	return false
}

// writeFixed persists the spliced content with the captured mode and reports
// which rules contributed. Writes go through e.writeFn (test seam, defaults to
// os.WriteFile). The defensive Chmod after the write guards against the file
// being recreated (umask wins over the mode otherwise). The applied rule names
// are interpolated into the error so a write failure points at which fixes
// were in flight.
func (e *Engine) writeFixed(path string, content []byte, mode os.FileMode, applied []string) ([]string, error) {
	if err := e.writeFn(path, content, mode); err != nil {
		return nil, fmt.Errorf("writing fixes for %v: %w", applied, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return nil, fmt.Errorf("restoring mode after fixes for %v: %w", applied, err)
	}
	return applied, nil
}

// Fixer returns the rule registered under the given name as an sdk.Fixer, or
// nil if no rule with that name is registered or it does not implement Fixer.
// The lookup matches against Rule.Name(); for built-in rules this is identical
// to the diagnostic Code reported in findings, but plugin authors should be
// aware the two are conceptually distinct.
//
// Callers outside this package (notably the LSP server's code-action handler)
// must snapshot the engine pointer under the appropriate read lock before
// invoking this method, since concurrent reloads may swap the rule set.
func (e *Engine) Fixer(name string) sdk.Fixer {
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

// RegisterFixerForTesting is test-only and not goroutine-safe. Tests using it
// must not run in parallel with other code that reads the registry (LSP
// CodeAction handlers, format/style runs). Use a single-threaded test or a
// per-test Engine instance.
//
// The seam wraps the supplied Fixer in a shim that satisfies sdk.Rule under the
// given name, then appends it to the engine's rule slice so Fixer finds it.
// The shim's Check returns no findings, so the registered name does
// not produce diagnostics on its own; callers exercising the Fix path must
// seed the diagnostic through a real rule or drive it directly.
func (e *Engine) RegisterFixerForTesting(name string, fixer sdk.Fixer) {
	e.rules = append(e.rules, &fixerForTesting{name: name, fixer: fixer})
}

// fixerForTesting adapts an sdk.Fixer into an sdk.Rule that can be appended to
// Engine.rules. Test-only — created exclusively by RegisterFixerForTesting.
type fixerForTesting struct {
	name  string
	fixer sdk.Fixer
}

func (f *fixerForTesting) Name() string        { return f.name }
func (f *fixerForTesting) Description() string { return "test-only fixer shim" }

func (f *fixerForTesting) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return nil, nil
}

func (f *fixerForTesting) Fix(ctx *sdk.Context, file *hcl.File) (*sdk.FixResult, error) {
	return f.fixer.Fix(ctx, file)
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
	e.rules = append(e.rules, &rules.TerragruntIncludeFirstRule{})

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
