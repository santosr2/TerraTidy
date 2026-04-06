package plugins

import (
	"fmt"
	"os"
	"regexp"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
)

// YAMLRuleConfig represents the configuration for a YAML-defined rule.
type YAMLRuleConfig struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Severity    string       `yaml:"severity"`
	Enabled     bool         `yaml:"enabled"`
	Message     string       `yaml:"message"`
	Patterns    YAMLPatterns `yaml:"patterns"`
	Tags        []string     `yaml:"tags"`
}

// YAMLPatterns defines pattern-based matching for YAML rules.
type YAMLPatterns struct {
	// BlockTypes filters by HCL block type (resource, variable, output, data, locals, module).
	// If empty, matches all block types.
	BlockTypes []string `yaml:"block_types"`
	// ResourceTypes filters by Terraform resource type (e.g., aws_s3_bucket).
	ResourceTypes []string `yaml:"resource_types"`
	// RequiredAttributes lists attributes that must be present.
	RequiredAttributes []string `yaml:"required_attributes"`
	// ForbiddenAttributes lists attributes that must NOT be present.
	ForbiddenAttributes []string `yaml:"forbidden_attributes"`
	// AttributePatterns validates attribute values against regex patterns.
	AttributePatterns []AttributePattern `yaml:"attribute_patterns"`
}

// AttributePattern defines a regex pattern to validate an attribute's value.
type AttributePattern struct {
	// Attribute is the name of the attribute to check.
	Attribute string `yaml:"attribute"`
	// Pattern is the regex pattern the attribute value must match.
	Pattern string `yaml:"pattern"`
	// Message is the custom message for findings (optional).
	Message string `yaml:"message"`
}

// compiledPattern holds an AttributePattern with its compiled regex.
type compiledPattern struct {
	AttributePattern
	regex *regexp.Regexp
}

// YAMLRule implements sdk.Rule from a YAML configuration file.
type YAMLRule struct {
	config           YAMLRuleConfig
	compiledPatterns []compiledPattern
}

// Name returns the rule name.
func (r *YAMLRule) Name() string { return r.config.Name }

// Description returns the rule description.
func (r *YAMLRule) Description() string { return r.config.Description }

// Tags returns the tags associated with this rule.
func (r *YAMLRule) Tags() []string { return r.config.Tags }

// Check evaluates HCL files against the YAML-defined patterns.
func (r *YAMLRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	if !r.config.Enabled {
		return nil, nil
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	severity := parseSeverity(r.config.Severity)
	var findings []sdk.Finding

	for _, block := range body.Blocks {
		if !r.matchesBlockType(block) {
			continue
		}
		if !r.matchesResourceType(block) {
			continue
		}

		for _, required := range r.config.Patterns.RequiredAttributes {
			if !blockHasAttribute(block, required) {
				msg := r.config.Message
				if msg == "" {
					msg = fmt.Sprintf("Missing required attribute: %s", required)
				}
				findings = append(findings, sdk.Finding{
					Rule:     r.config.Name,
					Message:  msg,
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.DefRange()),
					Severity: severity,
				})
			}
		}

		for _, forbidden := range r.config.Patterns.ForbiddenAttributes {
			if attr := getBlockAttribute(block, forbidden); attr != nil {
				msg := r.config.Message
				if msg == "" {
					msg = fmt.Sprintf("Forbidden attribute present: %s", forbidden)
				}
				findings = append(findings, sdk.Finding{
					Rule:     r.config.Name,
					Message:  msg,
					File:     ctx.File,
					Location: sdk.LocationFromRange(attr.SrcRange),
					Severity: severity,
				})
			}
		}

		for _, cp := range r.compiledPatterns {
			attr := getBlockAttribute(block, cp.Attribute)
			if attr == nil {
				// Attribute not present, skip (don't fail on optional attrs)
				continue
			}
			value := getAttributeStringValue(attr)
			if value == "" {
				// Can't extract string value, skip
				continue
			}
			if !cp.regex.MatchString(value) {
				msg := cp.Message
				if msg == "" {
					msg = fmt.Sprintf("Attribute %q value %q does not match pattern %q", cp.Attribute, value, cp.Pattern)
				}
				findings = append(findings, sdk.Finding{
					Rule:     r.config.Name,
					Message:  msg,
					File:     ctx.File,
					Location: sdk.LocationFromRange(attr.SrcRange),
					Severity: severity,
				})
			}
		}
	}

	return findings, nil
}

// Fix is not supported for YAML rules.
func (r *YAMLRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// matchesBlockType checks if a block matches the configured block types.
// If no block types are configured, matches all block types.
func (r *YAMLRule) matchesBlockType(block *hclsyntax.Block) bool {
	blockTypes := r.config.Patterns.BlockTypes
	if len(blockTypes) == 0 {
		return true // No filter = match all block types
	}
	return slices.Contains(blockTypes, block.Type)
}

// matchesResourceType checks if a block matches the configured resource types.
// Only applies to blocks that have labels (resource, data, variable, etc.).
// If no resource types are configured, all blocks of the matching type match.
func (r *YAMLRule) matchesResourceType(block *hclsyntax.Block) bool {
	if len(r.config.Patterns.ResourceTypes) == 0 {
		return true
	}
	if len(block.Labels) < 1 {
		return true // No label to filter on, allow it
	}
	return slices.Contains(r.config.Patterns.ResourceTypes, block.Labels[0])
}

// blockHasAttribute checks if an HCL block contains a given attribute.
func blockHasAttribute(block *hclsyntax.Block, name string) bool {
	return getBlockAttribute(block, name) != nil
}

// getBlockAttribute returns the attribute with the given name, or nil if not found.
func getBlockAttribute(block *hclsyntax.Block, name string) *hclsyntax.Attribute {
	if block.Body == nil {
		return nil
	}
	for _, attr := range block.Body.Attributes {
		if attr.Name == name {
			return attr
		}
	}
	return nil
}

// getAttributeStringValue extracts the string value from an attribute.
// Returns empty string if the value is not a simple string literal.
func getAttributeStringValue(attr *hclsyntax.Attribute) string {
	if attr == nil || attr.Expr == nil {
		return ""
	}
	// Handle template expressions (most string literals)
	if tmpl, ok := attr.Expr.(*hclsyntax.TemplateExpr); ok {
		if len(tmpl.Parts) == 1 {
			if lit, ok := tmpl.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				val := lit.Val
				if val.Type().Equals(cty.String) {
					return val.AsString()
				}
			}
		}
	}
	// Handle direct literal values
	if lit, ok := attr.Expr.(*hclsyntax.LiteralValueExpr); ok {
		val := lit.Val
		if val.Type().Equals(cty.String) {
			return val.AsString()
		}
	}
	return ""
}

// loadYAMLRule loads a YAML rule definition from a file.
func loadYAMLRule(path string) (*YAMLRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading YAML rule %s: %w", path, err)
	}

	var config YAMLRuleConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML rule %s: %w", path, err)
	}

	if config.Name == "" {
		return nil, fmt.Errorf("YAML rule %s is missing required 'name' field", path)
	}

	// Compile attribute patterns
	compiled, err := compileAttributePatterns(config.Patterns.AttributePatterns)
	if err != nil {
		return nil, fmt.Errorf("YAML rule %s: %w", path, err)
	}

	return &YAMLRule{config: config, compiledPatterns: compiled}, nil
}

// compileAttributePatterns compiles regex patterns from AttributePatterns.
func compileAttributePatterns(patterns []AttributePattern) ([]compiledPattern, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	compiled := make([]compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		if p.Attribute == "" {
			return nil, fmt.Errorf("attribute_pattern missing required 'attribute' field")
		}
		if p.Pattern == "" {
			return nil, fmt.Errorf("attribute_pattern for %q missing required 'pattern' field", p.Attribute)
		}
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern for attribute %q: %w", p.Attribute, err)
		}
		compiled = append(compiled, compiledPattern{
			AttributePattern: p,
			regex:            re,
		})
	}
	return compiled, nil
}

// parseSeverity converts a severity string to an sdk.Severity value (defaults to warning).
func parseSeverity(s string) sdk.Severity {
	return sdk.ParseSeverity(s, sdk.SeverityWarning)
}
