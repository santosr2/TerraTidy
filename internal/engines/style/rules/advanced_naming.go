package rules

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// ResourceNameMatchesTypeRule ensures resource names relate to their type.
// For example, aws_instance.web_server is better than aws_instance.foo.
type ResourceNameMatchesTypeRule struct{}

// Name returns the rule identifier.
func (r *ResourceNameMatchesTypeRule) Name() string {
	return "style.resource-name-matches-type"
}

// Description returns a human-readable description of the rule.
func (r *ResourceNameMatchesTypeRule) Description() string {
	return "Ensures resource names relate to their resource type"
}

// genericNames are names that don't provide meaningful context
var genericNames = map[string]bool{
	"this":     true,
	"main":     true,
	"default":  true,
	"primary":  true,
	"foo":      true,
	"bar":      true,
	"baz":      true,
	"test":     true,
	"example":  true,
	"temp":     true,
	"tmp":      true,
	"resource": true,
	"data":     true,
	"module":   true,
	"my":       true,
	"new":      true,
	"old":      true,
	"a":        true,
	"b":        true,
	"c":        true,
	"x":        true,
	"y":        true,
	"z":        true,
	"instance": true, // generic when used alone
	"cluster":  true, // generic when used alone
	"bucket":   true, // generic when used alone
	"policy":   true, // generic when used alone
	"role":     true, // generic when used alone
	"group":    true, // generic when used alone
	"rule":     true, // generic when used alone
}

// Check examines resource blocks for meaningful names.
func (r *ResourceNameMatchesTypeRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "data" {
			continue
		}

		if len(block.Labels) < 2 {
			continue
		}

		resourceType := block.Labels[0]
		resourceName := block.Labels[1]

		// Check for generic/meaningless names
		if genericNames[strings.ToLower(resourceName)] {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Resource name '%s' is too generic; consider a more descriptive name for %s", resourceName, resourceType),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
			continue
		}

		// Check if name is just repeating the resource type
		typeWords := r.extractTypeWords(resourceType)
		if r.nameJustRepeatsType(resourceName, typeWords) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Resource name '%s' just repeats the type; consider adding purpose (e.g., %s_web, %s_api)", resourceName, resourceName, resourceName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// extractTypeWords extracts meaningful words from a resource type (e.g., "aws_instance" -> ["instance"])
func (r *ResourceNameMatchesTypeRule) extractTypeWords(resourceType string) []string {
	parts := strings.Split(resourceType, "_")
	var words []string

	// Skip provider prefix (aws_, google_, azurerm_, etc.)
	startIdx := 0
	if len(parts) > 0 {
		providers := []string{"aws", "google", "azurerm", "azuread", "kubernetes", "helm", "vault", "null", "random", "local", "template", "tls", "archive", "external", "http", "time"}
		for _, p := range providers {
			if parts[0] == p {
				startIdx = 1
				break
			}
		}
	}

	for i := startIdx; i < len(parts); i++ {
		if parts[i] != "" {
			words = append(words, strings.ToLower(parts[i]))
		}
	}

	return words
}

// nameJustRepeatsType checks if the resource name is just repeating words from the type
func (r *ResourceNameMatchesTypeRule) nameJustRepeatsType(name string, typeWords []string) bool {
	nameLower := strings.ToLower(name)
	nameParts := strings.Split(nameLower, "_")

	// If name is exactly one of the type words, it's too repetitive
	for _, word := range typeWords {
		if nameLower == word {
			return true
		}
	}

	// If all name parts are from type words, it's repetitive
	allFromType := true
	for _, part := range nameParts {
		found := false
		for _, word := range typeWords {
			if part == word {
				found = true
				break
			}
		}
		if !found {
			allFromType = false
			break
		}
	}

	return allFromType && len(nameParts) > 0
}

// Fix is a no-op for this rule as renaming resources requires manual review.
func (r *ResourceNameMatchesTypeRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// OutputPrefixRule ensures outputs follow a naming pattern.
type OutputPrefixRule struct{}

// Name returns the rule identifier.
func (r *OutputPrefixRule) Name() string {
	return "style.output-prefix"
}

// Description returns a human-readable description of the rule.
func (r *OutputPrefixRule) Description() string {
	return "Ensures output names follow a consistent pattern"
}

// Check examines output blocks for naming patterns.
func (r *OutputPrefixRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get configuration
	prefix, suffix := r.getPatternConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		if block.Type != "output" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		outputName := block.Labels[0]

		// Check prefix if configured
		if prefix != "" && !strings.HasPrefix(outputName, prefix) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Output '%s' should have prefix '%s'", outputName, prefix),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}

		// Check suffix if configured
		if suffix != "" && !strings.HasSuffix(outputName, suffix) {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Output '%s' should have suffix '%s'", outputName, suffix),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

func (r *OutputPrefixRule) getPatternConfig(config map[string]any) (prefix, suffix string) {
	if config == nil {
		return "", ""
	}

	options, ok := config["options"].(map[string]any)
	if !ok {
		return "", ""
	}

	if p, ok := options["prefix"].(string); ok {
		prefix = p
	}
	if s, ok := options["suffix"].(string); ok {
		suffix = s
	}

	return prefix, suffix
}

// Fix is a no-op for this rule as renaming outputs requires manual review.
func (r *OutputPrefixRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// ModuleNameConventionRule ensures module names follow conventions.
type ModuleNameConventionRule struct{}

// Name returns the rule identifier.
func (r *ModuleNameConventionRule) Name() string {
	return "style.module-name-convention"
}

// Description returns a human-readable description of the rule.
func (r *ModuleNameConventionRule) Description() string {
	return "Ensures module names follow naming conventions"
}

// Check examines module blocks for naming conventions.
func (r *ModuleNameConventionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get naming convention from config
	convention, customPattern := GetNamingConventionFromConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		if block.Type != "module" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		moduleName := block.Labels[0]

		// Validate naming convention
		isValid, caseName := ValidateNaming(moduleName, convention, customPattern)
		if !isValid {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Module name '%s' should use %s", moduleName, caseName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}

		// Check for generic names
		if genericNames[strings.ToLower(moduleName)] {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Module name '%s' is too generic; consider a more descriptive name", moduleName),
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming modules requires manual review.
func (r *ModuleNameConventionRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
