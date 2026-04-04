package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// BlockLabelCaseRule ensures block labels follow naming conventions.
type BlockLabelCaseRule struct{}

// Name returns the rule identifier.
func (r *BlockLabelCaseRule) Name() string {
	return "style.block-label-case"
}

// Description returns a human-readable description of the rule.
func (r *BlockLabelCaseRule) Description() string {
	return "Ensures block labels follow naming conventions (snake_case for resources/data)"
}

// Check examines block labels for naming convention violations.
func (r *BlockLabelCaseRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get naming convention from config (defaults to snake_case)
	convention, customPattern := GetNamingConventionFromConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		blockType := block.Type

		if blockType != "resource" && blockType != "data" && blockType != "module" {
			continue
		}

		if len(block.Labels) < 2 {
			continue
		}

		name := block.Labels[1]

		if name == "" {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Block label cannot be empty",
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityError,
				Fixable:  false,
			})
			continue
		}

		// Validate naming convention for resources and data sources
		if blockType == "resource" || blockType == "data" {
			valid, caseName := ValidateNaming(name, convention, customPattern)
			if !valid {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "Block label should be " + caseName + ": " + name,
					File:     ctx.File,
					Location: sdk.LocationFromRange(block.Range()),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as block label renaming requires manual review.
func (r *BlockLabelCaseRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// VariableNamingRule ensures variable names follow naming conventions.
type VariableNamingRule struct{}

// Name returns the rule identifier.
func (r *VariableNamingRule) Name() string {
	return "style.variable-naming"
}

// Description returns a human-readable description of the rule.
func (r *VariableNamingRule) Description() string {
	return "Variable names should use snake_case"
}

// Check examines variable names for naming convention compliance.
func (r *VariableNamingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get naming convention from config (defaults to snake_case)
	convention, customPattern := GetNamingConventionFromConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		if block.Type != "variable" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		name := block.Labels[0]
		valid, caseName := ValidateNaming(name, convention, customPattern)
		if !valid {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Variable name should be " + caseName + ": " + name,
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming requires manual review.
func (r *VariableNamingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// OutputNamingRule ensures output names follow naming conventions.
type OutputNamingRule struct{}

// Name returns the rule identifier.
func (r *OutputNamingRule) Name() string {
	return "style.output-naming"
}

// Description returns a human-readable description of the rule.
func (r *OutputNamingRule) Description() string {
	return "Output names should use snake_case"
}

// Check examines output names for naming convention compliance.
func (r *OutputNamingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get naming convention from config (defaults to snake_case)
	convention, customPattern := GetNamingConventionFromConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		if block.Type != "output" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		name := block.Labels[0]
		valid, caseName := ValidateNaming(name, convention, customPattern)
		if !valid {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Output name should be " + caseName + ": " + name,
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming requires manual review.
func (r *OutputNamingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// LocalNamingRule ensures local value names follow naming conventions.
type LocalNamingRule struct{}

// Name returns the rule identifier.
func (r *LocalNamingRule) Name() string {
	return "style.local-naming"
}

// Description returns a human-readable description of the rule.
func (r *LocalNamingRule) Description() string {
	return "Local value names should use snake_case"
}

// Check examines local value names for naming convention compliance.
func (r *LocalNamingRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Get naming convention from config (defaults to snake_case)
	convention, customPattern := GetNamingConventionFromConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		if block.Type != "locals" {
			continue
		}

		// Check each attribute in the locals block
		for name, attr := range block.Body.Attributes {
			valid, caseName := ValidateNaming(name, convention, customPattern)
			if !valid {
				findings = append(findings, sdk.Finding{
					Rule:     r.Name(),
					Message:  "Local value name should be " + caseName + ": " + name,
					File:     ctx.File,
					Location: sdk.LocationFromRange(attr.Range()),
					Severity: sdk.SeverityWarning,
					Fixable:  false,
				})
			}
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as renaming requires manual review.
func (r *LocalNamingRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
