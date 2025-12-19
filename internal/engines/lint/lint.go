// Package lint provides linting capabilities for Terraform configurations.
// It performs AST-based analysis to detect common issues, best practice violations,
// and potential bugs in Terraform code. It can also integrate with TFLint when available.
package lint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// Engine represents the linting engine with AST-based analysis
type Engine struct {
	config *Config
	rules  []Rule
	parser *hclparse.Parser
}

// Config holds the linting engine configuration
type Config struct {
	ConfigFile      string                 // Path to configuration file
	Plugins         []string               // List of plugins to enable
	Rules           map[string]RuleConfig  // Rule-specific configuration
	Options         map[string]interface{} // Additional options
	UseTFLint       bool                   // Enable TFLint integration
	TFLintPath      string                 // Custom path to TFLint binary
	TFLintConfig    string                 // Path to TFLint config file
	FallbackBuiltin bool                   // Use built-in rules if TFLint unavailable
}

// RuleConfig holds configuration for a single rule
type RuleConfig struct {
	Enabled  bool
	Severity string
	Options  map[string]interface{}
}

// Rule defines the interface for lint rules
type Rule interface {
	Name() string
	Description() string
	Check(ctx *RuleContext) []sdk.Finding
}

// RuleContext provides context for rule execution
type RuleContext struct {
	File     string
	Content  []byte
	HCLFile  *hcl.File
	Body     *hclsyntax.Body
	Config   RuleConfig
	WorkDir  string
	AllFiles map[string]*hcl.File // All files in the module
}

// New creates a new linting engine
func New(config *Config) *Engine {
	if config == nil {
		config = &Config{
			ConfigFile: ".tflint.hcl",
			Rules:      make(map[string]RuleConfig),
		}
	}
	if config.Rules == nil {
		config.Rules = make(map[string]RuleConfig)
	}

	engine := &Engine{
		config: config,
		parser: hclparse.NewParser(),
		rules:  []Rule{},
	}

	// Register built-in rules
	engine.registerRules()

	return engine
}

// Name returns the engine name
func (e *Engine) Name() string {
	return "lint"
}

// Run executes the linting engine on the given files
func (e *Engine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding

	// Use TFLint integration if enabled and available
	if e.config.UseTFLint {
		if e.IsTFLintAvailable() {
			return e.RunWithTFLint(ctx, files)
		}
		// If TFLint requested but not available, fallback to built-in if enabled
		if !e.config.FallbackBuiltin {
			return nil, fmt.Errorf("TFLint integration enabled but tflint not found in PATH")
		}
		// Fall through to built-in rules
	}

	// Group files by directory for module-level analysis
	dirFiles := e.groupFilesByDirectory(files)

	for dir, dirFileList := range dirFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		findings, err := e.lintModule(ctx, dir, dirFileList)
		if err != nil {
			return nil, fmt.Errorf("linting %s: %w", dir, err)
		}

		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// lintModule runs linting checks on a Terraform module (directory)
func (e *Engine) lintModule(ctx context.Context, dir string, files []string) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	// Parse all files in the module first for cross-file analysis
	moduleFiles := make(map[string]*hcl.File)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		hclFile, diags := e.parser.ParseHCL(content, file)
		if diags.HasErrors() {
			findings = append(findings, sdk.Finding{
				Rule:     "lint.parse-error",
				Message:  fmt.Sprintf("Failed to parse file: %s", diags.Error()),
				File:     file,
				Severity: sdk.SeverityError,
				Fixable:  false,
			})
			continue
		}
		moduleFiles[file] = hclFile
	}

	// Process each file with module context
	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		hclFile, ok := moduleFiles[file]
		if !ok {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		body, ok := hclFile.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		ruleCtx := &RuleContext{
			File:     file,
			Content:  content,
			HCLFile:  hclFile,
			Body:     body,
			WorkDir:  dir,
			AllFiles: moduleFiles,
		}

		// Run all enabled rules
		for _, rule := range e.rules {
			ruleConfig := e.getRuleConfig(rule.Name())
			if !ruleConfig.Enabled {
				continue
			}

			ruleCtx.Config = ruleConfig
			ruleFindings := rule.Check(ruleCtx)
			findings = append(findings, ruleFindings...)
		}
	}

	return findings, nil
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

// registerRules registers all built-in lint rules
func (e *Engine) registerRules() {
	e.rules = append(e.rules,
		&TerraformRequiredVersionRule{},
		&TerraformRequiredProvidersRule{},
		&TerraformDeprecatedSyntaxRule{},
		&TerraformDocumentedVariablesRule{},
		&TerraformTypedVariablesRule{},
		&TerraformDocumentedOutputsRule{},
		&TerraformModulePinnedSourceRule{},
		&TerraformNamingConventionRule{},
		&TerraformUnusedDeclarationsRule{},
		&TerraformResourceCountRule{},
		&TerraformHardcodedSecretsRule{},
	)
}

// GetAllRules returns all registered rules
func (e *Engine) GetAllRules() []Rule {
	return e.rules
}

// groupFilesByDirectory groups files by their parent directory
func (e *Engine) groupFilesByDirectory(files []string) map[string][]string {
	dirFiles := make(map[string][]string)

	for _, file := range files {
		dir := filepath.Dir(file)
		dirFiles[dir] = append(dirFiles[dir], file)
	}

	return dirFiles
}

// parseSeverity converts string severity to sdk.Severity
func parseSeverity(severity string) sdk.Severity {
	switch strings.ToLower(severity) {
	case "error":
		return sdk.SeverityError
	case "warning":
		return sdk.SeverityWarning
	case "info":
		return sdk.SeverityInfo
	default:
		return sdk.SeverityWarning
	}
}

// ============================================================================
// Built-in Lint Rules
// ============================================================================

// TerraformRequiredVersionRule checks for terraform required_version constraint.
type TerraformRequiredVersionRule struct{}

// Name returns the rule identifier.
func (r *TerraformRequiredVersionRule) Name() string {
	return "lint.terraform-required-version"
}

// Description returns a human-readable description of the rule.
func (r *TerraformRequiredVersionRule) Description() string {
	return "Ensures terraform block contains a required_version constraint"
}

// Check examines files for required_version constraints.
func (r *TerraformRequiredVersionRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	// Look for terraform block
	for _, block := range ctx.Body.Blocks {
		if block.Type == "terraform" {
			// Check for required_version attribute
			for name := range block.Body.Attributes {
				if name == "required_version" {
					return findings // Found it
				}
			}
		}
	}

	// Only report once per module (check main.tf or first file)
	basename := filepath.Base(ctx.File)
	if basename == "main.tf" || basename == "versions.tf" {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "Missing terraform required_version constraint",
			File:     ctx.File,
			Location: hcl.Range{Filename: ctx.File, Start: hcl.Pos{Line: 1, Column: 1}},
			Severity: parseSeverity(ctx.Config.Severity),
			Fixable:  false,
		})
	}

	return findings
}

// TerraformRequiredProvidersRule checks for required_providers block.
type TerraformRequiredProvidersRule struct{}

// Name returns the rule identifier.
func (r *TerraformRequiredProvidersRule) Name() string {
	return "lint.terraform-required-providers"
}

// Description returns a human-readable description of the rule.
func (r *TerraformRequiredProvidersRule) Description() string {
	return "Ensures terraform block contains required_providers with version constraints"
}

// Check examines files for required_providers configuration.
func (r *TerraformRequiredProvidersRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	// Look for terraform block with required_providers
	for _, block := range ctx.Body.Blocks {
		if block.Type == "terraform" {
			for _, nested := range block.Body.Blocks {
				if nested.Type == "required_providers" {
					return findings // Found it
				}
			}
		}
	}

	// Only report once per module
	basename := filepath.Base(ctx.File)
	if basename == "main.tf" || basename == "versions.tf" {
		findings = append(findings, sdk.Finding{
			Rule:     r.Name(),
			Message:  "Missing required_providers block in terraform configuration",
			File:     ctx.File,
			Location: hcl.Range{Filename: ctx.File, Start: hcl.Pos{Line: 1, Column: 1}},
			Severity: sdk.SeverityInfo,
			Fixable:  false,
		})
	}

	return findings
}

// TerraformDeprecatedSyntaxRule checks for deprecated interpolation syntax.
type TerraformDeprecatedSyntaxRule struct{}

// Name returns the rule identifier.
func (r *TerraformDeprecatedSyntaxRule) Name() string {
	return "lint.terraform-deprecated-syntax"
}

// Description returns a human-readable description of the rule.
func (r *TerraformDeprecatedSyntaxRule) Description() string {
	return "Detects deprecated interpolation-only expressions like \"${var.x}\""
}

// deprecatedInterpolationRegex matches "${...}" patterns that should be simplified.
var deprecatedInterpolationRegex = regexp.MustCompile(`"\$\{([^}]+)\}"`)

// Check examines files for deprecated syntax patterns.
func (r *TerraformDeprecatedSyntaxRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding
	lines := strings.Split(string(ctx.Content), "\n")

	for i, line := range lines {
		matches := deprecatedInterpolationRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				inner := line[match[2]:match[3]]
				// Only flag if it's a simple reference (var.x, local.x, etc.)
				if isSimpleReference(inner) {
					msg := fmt.Sprintf(
						"Deprecated interpolation-only expression: use %s instead of \"${%s}\"",
						inner, inner,
					)
					findings = append(findings, sdk.Finding{
						Rule:    r.Name(),
						Message: msg,
						File:    ctx.File,
						Location: hcl.Range{
							Filename: ctx.File,
							Start:    hcl.Pos{Line: i + 1, Column: match[0] + 1},
						},
						Severity: sdk.SeverityWarning,
						Fixable:  true,
					})
				}
			}
		}
	}

	return findings
}

// isSimpleReference checks if the string is a simple variable/local reference.
func isSimpleReference(s string) bool {
	s = strings.TrimSpace(s)
	// Simple references: var.x, local.x, data.x.y, module.x.y
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}

	prefix := parts[0]
	return prefix == "var" || prefix == "local" || prefix == "data" ||
		prefix == "module" || prefix == "each" || prefix == "count"
}

// TerraformDocumentedVariablesRule ensures variables have descriptions.
type TerraformDocumentedVariablesRule struct{}

// Name returns the rule identifier.
func (r *TerraformDocumentedVariablesRule) Name() string {
	return "lint.terraform-documented-variables"
}

// Description returns a human-readable description of the rule.
func (r *TerraformDocumentedVariablesRule) Description() string {
	return "Ensures all variables have description attributes"
}

// Check examines variable blocks for description attributes.
func (r *TerraformDocumentedVariablesRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	for _, block := range ctx.Body.Blocks {
		if block.Type != "variable" {
			continue
		}

		varName := ""
		if len(block.Labels) > 0 {
			varName = block.Labels[0]
		}

		hasDescription := false
		for name := range block.Body.Attributes {
			if name == "description" {
				hasDescription = true
				break
			}
		}

		if !hasDescription {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Variable '%s' is missing a description", varName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: parseSeverity(ctx.Config.Severity),
				Fixable:  false,
			})
		}
	}

	return findings
}

// TerraformTypedVariablesRule ensures variables have type constraints.
type TerraformTypedVariablesRule struct{}

// Name returns the rule identifier.
func (r *TerraformTypedVariablesRule) Name() string {
	return "lint.terraform-typed-variables"
}

// Description returns a human-readable description of the rule.
func (r *TerraformTypedVariablesRule) Description() string {
	return "Ensures all variables have explicit type constraints"
}

// Check examines variable blocks for type constraints.
func (r *TerraformTypedVariablesRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	for _, block := range ctx.Body.Blocks {
		if block.Type != "variable" {
			continue
		}

		varName := ""
		if len(block.Labels) > 0 {
			varName = block.Labels[0]
		}

		hasType := false
		for name := range block.Body.Attributes {
			if name == "type" {
				hasType = true
				break
			}
		}

		if !hasType {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Variable '%s' should have an explicit type constraint", varName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings
}

// TerraformDocumentedOutputsRule ensures outputs have descriptions.
type TerraformDocumentedOutputsRule struct{}

// Name returns the rule identifier.
func (r *TerraformDocumentedOutputsRule) Name() string {
	return "lint.terraform-documented-outputs"
}

// Description returns a human-readable description of the rule.
func (r *TerraformDocumentedOutputsRule) Description() string {
	return "Ensures all outputs have description attributes"
}

// Check examines output blocks for description attributes.
func (r *TerraformDocumentedOutputsRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	for _, block := range ctx.Body.Blocks {
		if block.Type != "output" {
			continue
		}

		outputName := ""
		if len(block.Labels) > 0 {
			outputName = block.Labels[0]
		}

		hasDescription := false
		for name := range block.Body.Attributes {
			if name == "description" {
				hasDescription = true
				break
			}
		}

		if !hasDescription {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Output '%s' is missing a description", outputName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings
}

// TerraformModulePinnedSourceRule ensures module sources are pinned to versions.
type TerraformModulePinnedSourceRule struct{}

// Name returns the rule identifier.
func (r *TerraformModulePinnedSourceRule) Name() string {
	return "lint.terraform-module-pinned-source"
}

// Description returns a human-readable description of the rule.
func (r *TerraformModulePinnedSourceRule) Description() string {
	return "Ensures module sources are pinned to specific versions"
}

// Check examines module blocks for version pinning.
func (r *TerraformModulePinnedSourceRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	for _, block := range ctx.Body.Blocks {
		if block.Type != "module" {
			continue
		}

		moduleName := ""
		if len(block.Labels) > 0 {
			moduleName = block.Labels[0]
		}

		var sourceAttr *hclsyntax.Attribute
		var versionAttr *hclsyntax.Attribute

		for name, attr := range block.Body.Attributes {
			if name == "source" {
				sourceAttr = attr
			}
			if name == "version" {
				versionAttr = attr
			}
		}

		if sourceAttr == nil {
			continue
		}

		// Get source value - this is simplified, real implementation would evaluate
		sourceExpr := string(ctx.Content[sourceAttr.Expr.Range().Start.Byte:sourceAttr.Expr.Range().End.Byte])

		// Check if it's a registry module (needs version)
		isRegistryModule := !strings.HasPrefix(sourceExpr, "\"./") &&
			!strings.HasPrefix(sourceExpr, "\"../") &&
			!strings.Contains(sourceExpr, "github.com") &&
			!strings.Contains(sourceExpr, "git::") &&
			!strings.Contains(sourceExpr, "s3::") &&
			!strings.Contains(sourceExpr, "gcs::")

		if isRegistryModule && versionAttr == nil {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Module '%s' from registry should have a version constraint", moduleName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}

		// Check for git sources without ref
		if strings.Contains(sourceExpr, "git::") || strings.Contains(sourceExpr, "github.com") {
			if !strings.Contains(sourceExpr, "?ref=") && !strings.Contains(sourceExpr, "//") {
				msg := fmt.Sprintf(
					"Module '%s' from git should specify a ref (tag, branch, or commit)",
					moduleName,
				)
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  msg,
					File:     ctx.File,
					Location: block.Range(),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	return findings
}

// TerraformNamingConventionRule checks resource naming conventions.
type TerraformNamingConventionRule struct{}

// Name returns the rule identifier.
func (r *TerraformNamingConventionRule) Name() string {
	return "lint.terraform-naming-convention"
}

// Description returns a human-readable description of the rule.
func (r *TerraformNamingConventionRule) Description() string {
	return "Ensures resources follow naming conventions (snake_case)"
}

// snakeCasePattern matches valid snake_case identifiers.
var snakeCasePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Check examines resource names for naming convention compliance.
func (r *TerraformNamingConventionRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	for _, block := range ctx.Body.Blocks {
		if block.Type != "resource" && block.Type != "data" && block.Type != "module" {
			continue
		}

		// Get the name label (second for resource/data, first for module)
		var name string
		if block.Type == "module" && len(block.Labels) > 0 {
			name = block.Labels[0]
		} else if len(block.Labels) > 1 {
			name = block.Labels[1]
		} else {
			continue
		}

		if !snakeCasePattern.MatchString(name) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("%s name '%s' should use snake_case", block.Type, name),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings
}

// TerraformUnusedDeclarationsRule checks for unused variables and locals.
type TerraformUnusedDeclarationsRule struct{}

// Name returns the rule identifier.
func (r *TerraformUnusedDeclarationsRule) Name() string {
	return "lint.terraform-unused-declarations"
}

// Description returns a human-readable description of the rule.
func (r *TerraformUnusedDeclarationsRule) Description() string {
	return "Detects declared but unused variables and locals"
}

// Check examines variables and locals for usage.
func (r *TerraformUnusedDeclarationsRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	// Collect all declared variables
	declaredVars := make(map[string]hcl.Range)
	for _, block := range ctx.Body.Blocks {
		if block.Type == "variable" && len(block.Labels) > 0 {
			declaredVars[block.Labels[0]] = block.Range()
		}
	}

	// Search for var.X references in all module files
	contentStr := ""
	for _, hclFile := range ctx.AllFiles {
		if hclFile == nil {
			continue
		}
		// Get the raw bytes from the file
		for path := range ctx.AllFiles {
			if content, err := os.ReadFile(path); err == nil {
				contentStr += string(content) + "\n"
			}
		}
		break // Only need to do this once
	}

	// Check each variable
	for varName, varRange := range declaredVars {
		// Check if var.X appears in content
		pattern := fmt.Sprintf("var\\.%s[^a-zA-Z0-9_]", regexp.QuoteMeta(varName))
		if matched, _ := regexp.MatchString(pattern, contentStr); !matched {
			// Also check for var.X at end of expression
			patternEnd := fmt.Sprintf("var\\.%s$", regexp.QuoteMeta(varName))
			if matchedEnd, _ := regexp.MatchString(patternEnd, contentStr); !matchedEnd {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  fmt.Sprintf("Variable '%s' is declared but never used", varName),
					File:     ctx.File,
					Location: varRange,
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	return findings
}

// TerraformResourceCountRule checks for high resource counts.
type TerraformResourceCountRule struct{}

// Name returns the rule identifier.
func (r *TerraformResourceCountRule) Name() string {
	return "lint.terraform-resource-count"
}

// Description returns a human-readable description of the rule.
func (r *TerraformResourceCountRule) Description() string {
	return "Warns when a file has too many resources (suggests splitting)"
}

// Check counts resources in a file and warns if above threshold.
func (r *TerraformResourceCountRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding

	resourceCount := 0
	for _, block := range ctx.Body.Blocks {
		if block.Type == "resource" {
			resourceCount++
		}
	}

	// Default threshold of 15 resources per file
	threshold := 15
	if t, ok := ctx.Config.Options["threshold"].(int); ok && t > 0 {
		threshold = t
	}

	if resourceCount > threshold {
		msg := fmt.Sprintf(
			"File has %d resources (threshold: %d). Consider splitting into multiple files",
			resourceCount, threshold,
		)
		findings = append(findings, sdk.Finding{
			Rule:    r.Name(),
			Message: msg,
			File:    ctx.File,
			Location: hcl.Range{
				Filename: ctx.File,
				Start:    hcl.Pos{Line: 1, Column: 1},
			},
			Severity: sdk.SeverityInfo,
			Fixable:  false,
		})
	}

	return findings
}

// TerraformHardcodedSecretsRule detects potential hardcoded secrets.
type TerraformHardcodedSecretsRule struct{}

// Name returns the rule identifier.
func (r *TerraformHardcodedSecretsRule) Name() string {
	return "lint.terraform-hardcoded-secrets"
}

// Description returns a human-readable description of the rule.
func (r *TerraformHardcodedSecretsRule) Description() string {
	return "Detects potential hardcoded secrets like AWS keys, passwords, and API tokens"
}

// Secret patterns to detect
var secretPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"AWS Access Key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"AWS Secret Key", regexp.MustCompile(`(?i)(aws_secret_access_key|secret_key)\s*=\s*"[A-Za-z0-9/+=]{40}"`)},
	{"Generic API Key", regexp.MustCompile(`(?i)(api_key|apikey|api_token)\s*=\s*"[A-Za-z0-9_\-]{20,}"`)},
	{"Private Key", regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)},
	{"Generic Secret", regexp.MustCompile(`(?i)(password|passwd|secret|token)\s*=\s*"[^"$]{8,}"`)},
}

// Sensitive attribute names that should use variables
var sensitiveAttributes = []string{
	"password", "secret", "token", "api_key", "apikey",
	"access_key", "secret_key", "private_key", "auth_token",
	"credentials", "connection_string",
}

// Check examines files for hardcoded secrets.
func (r *TerraformHardcodedSecretsRule) Check(ctx *RuleContext) []sdk.Finding {
	var findings []sdk.Finding
	lines := strings.Split(string(ctx.Content), "\n")

	// Check for secret patterns in file content
	for i, line := range lines {
		for _, sp := range secretPatterns {
			if sp.pattern.MatchString(line) {
				// Skip if it's using a variable reference
				if strings.Contains(line, "var.") || strings.Contains(line, "local.") ||
					strings.Contains(line, "data.") || strings.Contains(line, "module.") {
					continue
				}

				findings = append(findings, sdk.Finding{
					Rule:    r.Name(),
					Message: fmt.Sprintf("Potential hardcoded %s detected", sp.name),
					File:    ctx.File,
					Location: hcl.Range{
						Filename: ctx.File,
						Start:    hcl.Pos{Line: i + 1, Column: 1},
					},
					Severity: sdk.SeverityError,
					Fixable:  false,
				})
			}
		}
	}

	// Check for sensitive attributes with literal string values
	for _, block := range ctx.Body.Blocks {
		r.checkBlockForSecrets(ctx, block, &findings)
	}

	return findings
}

// checkBlockForSecrets recursively checks blocks for hardcoded secrets.
func (r *TerraformHardcodedSecretsRule) checkBlockForSecrets(
	ctx *RuleContext,
	block *hclsyntax.Block,
	findings *[]sdk.Finding,
) {
	for attrName, attr := range block.Body.Attributes {
		// Check if attribute name is sensitive
		isSensitive := false
		for _, sensitive := range sensitiveAttributes {
			if strings.Contains(strings.ToLower(attrName), sensitive) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			continue
		}

		// Check if the value is a literal string (not a variable reference)
		if templateExpr, ok := attr.Expr.(*hclsyntax.TemplateExpr); ok {
			// Template with only literal parts is a hardcoded string
			allLiteral := true
			for _, part := range templateExpr.Parts {
				if _, isLit := part.(*hclsyntax.LiteralValueExpr); !isLit {
					allLiteral = false
					break
				}
			}
			if allLiteral && len(templateExpr.Parts) > 0 {
				// Get the string value to check if it's not empty/placeholder
				exprBytes := ctx.Content[attr.Expr.Range().Start.Byte:attr.Expr.Range().End.Byte]
				exprStr := string(exprBytes)

				// Skip if it looks like a placeholder
				if strings.Contains(exprStr, "CHANGE_ME") ||
					strings.Contains(exprStr, "REPLACE") ||
					strings.Contains(exprStr, "TODO") ||
					exprStr == `""` {
					continue
				}

				msg := fmt.Sprintf(
					"Sensitive attribute '%s' has a hardcoded value. "+
						"Use a variable or secret manager instead",
					attrName,
				)
				*findings = append(*findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  msg,
					File:     ctx.File,
					Location: attr.Range(),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	// Recursively check nested blocks
	for _, nested := range block.Body.Blocks {
		r.checkBlockForSecrets(ctx, nested, findings)
	}
}

// ============================================================================
// TFLint Integration
// ============================================================================

// TFLintOutput represents the JSON output from TFLint
type TFLintOutput struct {
	Issues []TFLintIssue `json:"issues"`
	Errors []TFLintError `json:"errors"`
}

// TFLintIssue represents a single issue from TFLint
type TFLintIssue struct {
	Rule    TFLintRule    `json:"rule"`
	Message string        `json:"message"`
	Range   TFLintRange   `json:"range"`
	Callers []TFLintRange `json:"callers"`
}

// TFLintRule represents rule information in TFLint output
type TFLintRule struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Link     string `json:"link"`
}

// TFLintRange represents a location in TFLint output
type TFLintRange struct {
	Filename string    `json:"filename"`
	Start    TFLintPos `json:"start"`
	End      TFLintPos `json:"end"`
}

// TFLintPos represents a position in TFLint output
type TFLintPos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// TFLintError represents an error from TFLint
type TFLintError struct {
	Summary  string       `json:"summary"`
	Detail   string       `json:"detail"`
	Range    *TFLintRange `json:"range"`
	Severity string       `json:"severity"`
}

// IsTFLintAvailable checks if TFLint is available on the system
func (e *Engine) IsTFLintAvailable() bool {
	path := e.getTFLintPath()
	_, err := exec.LookPath(path)
	return err == nil
}

// getTFLintPath returns the path to the TFLint binary
func (e *Engine) getTFLintPath() string {
	if e.config.TFLintPath != "" {
		return e.config.TFLintPath
	}
	return "tflint"
}

// RunTFLint executes TFLint on the given directory and returns findings
func (e *Engine) RunTFLint(ctx context.Context, dir string) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	tflintPath := e.getTFLintPath()

	// Build command arguments
	args := []string{
		"--format=json",
		"--force",
	}

	// Add config file if specified
	if e.config.TFLintConfig != "" {
		args = append(args, "--config="+e.config.TFLintConfig)
	} else if e.config.ConfigFile != "" {
		// Check if .tflint.hcl exists
		configPath := filepath.Join(dir, e.config.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			args = append(args, "--config="+configPath)
		}
	}

	// Create command
	cmd := exec.CommandContext(ctx, tflintPath, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run TFLint (it returns non-zero exit code when issues are found)
	err := cmd.Run()

	// Parse JSON output even if there's an error (TFLint returns non-zero on issues)
	output := stdout.Bytes()
	if len(output) == 0 {
		// If no stdout, check if it's a real error
		if err != nil && stderr.Len() > 0 {
			return nil, fmt.Errorf("tflint failed: %s", stderr.String())
		}
		return findings, nil
	}

	var tflintOutput TFLintOutput
	if err := json.Unmarshal(output, &tflintOutput); err != nil {
		return nil, fmt.Errorf("parsing tflint output: %w", err)
	}

	// Convert TFLint issues to sdk.Finding
	for _, issue := range tflintOutput.Issues {
		severity := sdk.SeverityWarning
		switch strings.ToLower(issue.Rule.Severity) {
		case "error":
			severity = sdk.SeverityError
		case "warning":
			severity = sdk.SeverityWarning
		case "info", "notice":
			severity = sdk.SeverityInfo
		}

		finding := sdk.Finding{
			Rule:    "tflint." + issue.Rule.Name,
			Message: issue.Message,
			File:    filepath.Join(dir, issue.Range.Filename),
			Location: hcl.Range{
				Filename: filepath.Join(dir, issue.Range.Filename),
				Start: hcl.Pos{
					Line:   issue.Range.Start.Line,
					Column: issue.Range.Start.Column,
				},
				End: hcl.Pos{
					Line:   issue.Range.End.Line,
					Column: issue.Range.End.Column,
				},
			},
			Severity: severity,
			Fixable:  false,
		}

		findings = append(findings, finding)
	}

	// Convert TFLint errors to findings
	for _, tflintErr := range tflintOutput.Errors {
		finding := sdk.Finding{
			Rule:     "tflint.error",
			Message:  tflintErr.Summary + ": " + tflintErr.Detail,
			Severity: sdk.SeverityError,
			Fixable:  false,
		}

		if tflintErr.Range != nil {
			finding.File = filepath.Join(dir, tflintErr.Range.Filename)
			finding.Location = hcl.Range{
				Filename: filepath.Join(dir, tflintErr.Range.Filename),
				Start: hcl.Pos{
					Line:   tflintErr.Range.Start.Line,
					Column: tflintErr.Range.Start.Column,
				},
			}
		}

		findings = append(findings, finding)
	}

	return findings, nil
}

// RunWithTFLint runs linting using TFLint integration
func (e *Engine) RunWithTFLint(ctx context.Context, files []string) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding

	// Group files by directory
	dirFiles := e.groupFilesByDirectory(files)

	for dir := range dirFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		findings, err := e.RunTFLint(ctx, dir)
		if err != nil {
			// If TFLint fails and fallback is enabled, use built-in rules
			if e.config.FallbackBuiltin {
				builtinFindings, builtinErr := e.lintModule(ctx, dir, dirFiles[dir])
				if builtinErr != nil {
					return nil, builtinErr
				}
				allFindings = append(allFindings, builtinFindings...)
				continue
			}
			return nil, err
		}

		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}
