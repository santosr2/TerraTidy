package rules

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// VariablesInFileRule ensures variables are defined in variables.tf.
type VariablesInFileRule struct{}

// Name returns the rule identifier.
func (r *VariablesInFileRule) Name() string {
	return "style.variables-in-file"
}

// Description returns a human-readable description of the rule.
func (r *VariablesInFileRule) Description() string {
	return "Variables should be defined in variables.tf"
}

// Check examines if variables are in the correct file.
func (r *VariablesInFileRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Skip if this is variables.tf
	basename := filepath.Base(ctx.File)
	if basename == "variables.tf" {
		return findings, nil
	}

	// Check for variable blocks in non-variables.tf files
	for _, block := range hclFile.Blocks {
		if block.Type == "variable" {
			varName := ""
			if len(block.Labels) > 0 {
				varName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Variable '" + varName + "' should be defined in variables.tf",
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityInfo,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as moving blocks requires manual review.
func (r *VariablesInFileRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// OutputsInFileRule ensures outputs are defined in outputs.tf.
type OutputsInFileRule struct{}

// Name returns the rule identifier.
func (r *OutputsInFileRule) Name() string {
	return "style.outputs-in-file"
}

// Description returns a human-readable description of the rule.
func (r *OutputsInFileRule) Description() string {
	return "Outputs should be defined in outputs.tf"
}

// Check examines if outputs are in the correct file.
func (r *OutputsInFileRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Skip if this is outputs.tf
	basename := filepath.Base(ctx.File)
	if basename == "outputs.tf" {
		return findings, nil
	}

	// Check for output blocks in non-outputs.tf files
	for _, block := range hclFile.Blocks {
		if block.Type == "output" {
			outputName := ""
			if len(block.Labels) > 0 {
				outputName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Output '" + outputName + "' should be defined in outputs.tf",
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityInfo,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as moving blocks requires manual review.
func (r *OutputsInFileRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// ProvidersInFileRule ensures providers are defined in providers.tf or versions.tf.
type ProvidersInFileRule struct{}

// Name returns the rule identifier.
func (r *ProvidersInFileRule) Name() string {
	return "style.providers-in-file"
}

// Description returns a human-readable description of the rule.
func (r *ProvidersInFileRule) Description() string {
	return "Provider configurations should be in providers.tf or versions.tf"
}

// Check examines if providers are in the correct file.
func (r *ProvidersInFileRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Skip if this is providers.tf or versions.tf
	basename := filepath.Base(ctx.File)
	if basename == "providers.tf" || basename == "versions.tf" {
		return findings, nil
	}

	// Check for provider blocks in other files
	for _, block := range hclFile.Blocks {
		if block.Type == "provider" {
			providerName := ""
			if len(block.Labels) > 0 {
				providerName = block.Labels[0]
			}
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  "Provider '" + providerName + "' should be defined in providers.tf or versions.tf",
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityInfo,
			})
		}
	}

	return findings, nil
}

// Fix is a no-op for this rule as moving blocks requires manual review.
func (r *ProvidersInFileRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
