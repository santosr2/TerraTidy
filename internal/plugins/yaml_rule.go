package plugins

import (
	"fmt"
	"os"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
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
	ResourceTypes      []string `yaml:"resource_types"`
	RequiredAttributes []string `yaml:"required_attributes"`
}

// YAMLRule implements sdk.Rule from a YAML configuration file.
type YAMLRule struct {
	config YAMLRuleConfig
}

// Name returns the rule name.
func (r *YAMLRule) Name() string { return r.config.Name }

// Description returns the rule description.
func (r *YAMLRule) Description() string { return r.config.Description }

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
					Location: block.DefRange(),
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

// matchesResourceType checks if a block matches the configured resource types.
// If no resource types are configured, all resource blocks match.
func (r *YAMLRule) matchesResourceType(block *hclsyntax.Block) bool {
	if block.Type != "resource" || len(block.Labels) < 1 {
		return false
	}
	if len(r.config.Patterns.ResourceTypes) == 0 {
		return true
	}
	return slices.Contains(r.config.Patterns.ResourceTypes, block.Labels[0])
}

// blockHasAttribute checks if an HCL block contains a given attribute.
func blockHasAttribute(block *hclsyntax.Block, name string) bool {
	for _, attr := range block.Body.Attributes {
		if attr.Name == name {
			return true
		}
	}
	return false
}

// loadYAMLRule loads a YAML rule definition from a file.
func loadYAMLRule(path string) (*YAMLRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading YAML rule %s: %w", path, err)
	}

	var config YAMLRuleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML rule %s: %w", path, err)
	}

	if config.Name == "" {
		return nil, fmt.Errorf("YAML rule %s is missing required 'name' field", path)
	}

	return &YAMLRule{config: config}, nil
}

// parseSeverity converts a severity string to an sdk.Severity value (defaults to warning).
func parseSeverity(s string) sdk.Severity {
	return sdk.ParseSeverity(s, sdk.SeverityWarning)
}
