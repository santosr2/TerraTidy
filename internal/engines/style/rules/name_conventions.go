package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// labelNoun returns the human-facing noun for a named block type, used to make
// naming findings read the same way as the variable/output/local rules.
func labelNoun(blockType string) string {
	switch blockType {
	case "data":
		return "Data source"
	case "resource":
		return "Resource"
	default:
		return blockType
	}
}

// checkLabelNaming validates the name label (the second block label) of every
// block of blockType against the configured naming convention. An empty name
// yields a single SeverityError finding; a mis-cased name yields a single
// SeverityWarning finding. Both are attributed to ruleName so callers surface
// their own rule identifier.
func checkLabelNaming(ctx *sdk.Context, file *hcl.File, blockType, ruleName string) []sdk.Finding {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings
	}

	convention, customPattern := getNamingConventionFromConfig(ctx.Options)
	noun := labelNoun(blockType)

	for _, block := range hclFile.Blocks {
		if block.Type != blockType {
			continue
		}

		if len(block.Labels) < 2 {
			continue
		}

		name := block.Labels[1]

		if name == "" {
			findings = append(findings, sdk.Finding{
				Rule:     ruleName,
				Message:  noun + " name cannot be empty",
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityError,
			})
			// Skip the naming check so an empty name reports exactly one finding.
			continue
		}

		valid, caseName := validateNaming(name, convention, customPattern)
		if !valid {
			findings = append(findings, sdk.Finding{
				Rule:     ruleName,
				Message:  noun + " name should be " + caseName + ": " + name,
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityWarning,
			})
		}
	}

	return findings
}

// nameOccurrence is a single identifier to validate together with the range a
// finding should be reported against.
type nameOccurrence struct {
	name string
	rng  hcl.Range
}

// checkNaming validates every identifier that extract pulls from each block
// against the configured naming convention, emitting one SeverityWarning per
// mis-cased name attributed to ruleName. noun is the leading text of the message
// (e.g. "Variable name"), matching the phrasing of the label-based rules. Blocks
// the rule doesn't care about return no occurrences.
func checkNaming(ctx *sdk.Context, file *hcl.File, ruleName, noun string, extract func(*hclsyntax.Block) []nameOccurrence) []sdk.Finding {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings
	}

	convention, customPattern := getNamingConventionFromConfig(ctx.Options)

	for _, block := range hclFile.Blocks {
		for _, occ := range extract(block) {
			valid, caseName := validateNaming(occ.name, convention, customPattern)
			if !valid {
				findings = append(findings, sdk.Finding{
					Rule:     ruleName,
					Message:  noun + " should be " + caseName + ": " + occ.name,
					File:     ctx.File,
					Location: sdk.LocationFromRange(occ.rng),
					Severity: sdk.SeverityWarning,
				})
			}
		}
	}

	return findings
}

// ResourceNameConventionRule ensures resource names follow naming conventions.
type ResourceNameConventionRule struct{}

// Name returns the rule identifier.
func (r *ResourceNameConventionRule) Name() string {
	return "style.resource-name-convention"
}

// Description returns a human-readable description of the rule.
func (r *ResourceNameConventionRule) Description() string {
	return "Ensures resource names follow naming conventions (snake_case by default)"
}

// Check examines resource block names for naming convention violations.
func (r *ResourceNameConventionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	return checkLabelNaming(ctx, file, "resource", r.Name()), nil
}

// DataNameConventionRule ensures data source names follow naming conventions.
type DataNameConventionRule struct{}

// Name returns the rule identifier.
func (r *DataNameConventionRule) Name() string {
	return "style.data-name-convention"
}

// Description returns a human-readable description of the rule.
func (r *DataNameConventionRule) Description() string {
	return "Ensures data source names follow naming conventions (snake_case by default)"
}

// Check examines data source block names for naming convention violations.
func (r *DataNameConventionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	return checkLabelNaming(ctx, file, "data", r.Name()), nil
}

// VariableNameConventionRule ensures variable names follow naming conventions.
type VariableNameConventionRule struct{}

// Name returns the rule identifier.
func (r *VariableNameConventionRule) Name() string {
	return "style.variable-name-convention"
}

// Description returns a human-readable description of the rule.
func (r *VariableNameConventionRule) Description() string {
	return "Variable names should use snake_case"
}

// Check examines variable names for naming convention compliance.
func (r *VariableNameConventionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	return checkNaming(ctx, file, r.Name(), "Variable name", func(block *hclsyntax.Block) []nameOccurrence {
		if block.Type != "variable" || len(block.Labels) == 0 {
			return nil
		}
		return []nameOccurrence{{name: block.Labels[0], rng: block.Range()}}
	}), nil
}

// OutputNameConventionRule ensures output names follow naming conventions.
type OutputNameConventionRule struct{}

// Name returns the rule identifier.
func (r *OutputNameConventionRule) Name() string {
	return "style.output-name-convention"
}

// Description returns a human-readable description of the rule.
func (r *OutputNameConventionRule) Description() string {
	return "Output names should use snake_case"
}

// Check examines output names for naming convention compliance.
func (r *OutputNameConventionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	return checkNaming(ctx, file, r.Name(), "Output name", func(block *hclsyntax.Block) []nameOccurrence {
		if block.Type != "output" || len(block.Labels) == 0 {
			return nil
		}
		return []nameOccurrence{{name: block.Labels[0], rng: block.Range()}}
	}), nil
}

// LocalNameConventionRule ensures local value names follow naming conventions.
type LocalNameConventionRule struct{}

// Name returns the rule identifier.
func (r *LocalNameConventionRule) Name() string {
	return "style.local-name-convention"
}

// Description returns a human-readable description of the rule.
func (r *LocalNameConventionRule) Description() string {
	return "Local value names should use snake_case"
}

// Check examines local value names for naming convention compliance.
func (r *LocalNameConventionRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	return checkNaming(ctx, file, r.Name(), "Local value name", func(block *hclsyntax.Block) []nameOccurrence {
		if block.Type != "locals" {
			return nil
		}
		occ := make([]nameOccurrence, 0, len(block.Body.Attributes))
		for name, attr := range block.Body.Attributes {
			occ = append(occ, nameOccurrence{name: name, rng: attr.Range()})
		}
		return occ
	}), nil
}
