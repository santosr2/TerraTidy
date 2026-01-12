// Package style provides the style engine and rules for TerraTidy.
// It enforces consistent code style and formatting conventions in Terraform
// configurations, such as attribute ordering, block spacing, and naming conventions.
//
// This file re-exports rules from the rules subpackage for backwards compatibility.
package style

import (
	"github.com/santosr2/terratidy/internal/engines/style/rules"
)

// Type aliases for backwards compatibility
type (
	// Spacing rules
	BlankLineBetweenBlocksRule      = rules.BlankLineBetweenBlocksRule
	NoLeadingTrailingBlankLinesRule = rules.NoLeadingTrailingBlankLinesRule
	NoEmptyBlocksRule               = rules.NoEmptyBlocksRule

	// Naming rules
	BlockLabelCaseRule = rules.BlockLabelCaseRule
	VariableNamingRule = rules.VariableNamingRule
	OutputNamingRule   = rules.OutputNamingRule
	LocalNamingRule    = rules.LocalNamingRule

	// Ordering rules
	ForEachCountFirstRule     = rules.ForEachCountFirstRule
	LifecycleAtEndRule        = rules.LifecycleAtEndRule
	TagsAtEndRule             = rules.TagsAtEndRule
	DependsOnOrderRule        = rules.DependsOnOrderRule
	SourceVersionGroupedRule  = rules.SourceVersionGroupedRule
	VariableOrderRule         = rules.VariableOrderRule
	OutputOrderRule           = rules.OutputOrderRule
	TerraformBlockFirstRule   = rules.TerraformBlockFirstRule
	ProviderBlockOrderRule    = rules.ProviderBlockOrderRule
	AttributeGroupSpacingRule = rules.AttributeGroupSpacingRule

	// File organization rules
	VariablesInFileRule = rules.VariablesInFileRule
	OutputsInFileRule   = rules.OutputsInFileRule
	ProvidersInFileRule = rules.ProvidersInFileRule
)
