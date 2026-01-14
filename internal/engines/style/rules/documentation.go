// Package rules provides style rules for TerraTidy.
package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// RequireVariableDescriptionRule ensures variable blocks have descriptions.
type RequireVariableDescriptionRule struct{}

// Name returns the rule identifier.
func (r *RequireVariableDescriptionRule) Name() string {
	return "style.require-variable-description"
}

// Description returns a human-readable description of the rule.
func (r *RequireVariableDescriptionRule) Description() string {
	return "Ensures all variable blocks have a description attribute"
}

// Check examines variable blocks for missing description attributes.
func (r *RequireVariableDescriptionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "variable" {
			continue
		}

		// Check if description attribute exists
		hasDescription := false
		for name := range block.Body.Attributes {
			if name == "description" {
				hasDescription = true
				break
			}
		}

		if !hasDescription {
			varName := ""
			if len(block.Labels) > 0 {
				varName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Variable \"" + varName + "\" is missing a description",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as adding descriptions requires manual input.
func (r *RequireVariableDescriptionRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// RequireOutputDescriptionRule ensures output blocks have descriptions.
type RequireOutputDescriptionRule struct{}

// Name returns the rule identifier.
func (r *RequireOutputDescriptionRule) Name() string {
	return "style.require-output-description"
}

// Description returns a human-readable description of the rule.
func (r *RequireOutputDescriptionRule) Description() string {
	return "Ensures all output blocks have a description attribute"
}

// Check examines output blocks for missing description attributes.
func (r *RequireOutputDescriptionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "output" {
			continue
		}

		// Check if description attribute exists
		hasDescription := false
		for name := range block.Body.Attributes {
			if name == "description" {
				hasDescription = true
				break
			}
		}

		if !hasDescription {
			outputName := ""
			if len(block.Labels) > 0 {
				outputName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Output \"" + outputName + "\" is missing a description",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as adding descriptions requires manual input.
func (r *RequireOutputDescriptionRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// RequireVariableTypeRule ensures variable blocks have explicit types.
type RequireVariableTypeRule struct{}

// Name returns the rule identifier.
func (r *RequireVariableTypeRule) Name() string {
	return "style.require-variable-type"
}

// Description returns a human-readable description of the rule.
func (r *RequireVariableTypeRule) Description() string {
	return "Ensures all variable blocks have an explicit type attribute"
}

// Check examines variable blocks for missing type attributes.
func (r *RequireVariableTypeRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	for _, block := range hclFile.Blocks {
		if block.Type != "variable" {
			continue
		}

		// Check if type attribute exists
		hasType := false
		for name := range block.Body.Attributes {
			if name == "type" {
				hasType = true
				break
			}
		}

		if !hasType {
			varName := ""
			if len(block.Labels) > 0 {
				varName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Variable \"" + varName + "\" is missing an explicit type",
				File:     ctx.File,
				Location: block.Range(),
				Severity: sdk.SeverityWarning,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as adding types requires manual input.
func (r *RequireVariableTypeRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
