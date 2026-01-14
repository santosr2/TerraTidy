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

	// Advanced naming rules
	ResourceNameMatchesTypeRule = rules.ResourceNameMatchesTypeRule
	OutputPrefixRule            = rules.OutputPrefixRule
	ModuleNameConventionRule    = rules.ModuleNameConventionRule

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

	// Block organization rules
	MetaArgumentsOrderRule      = rules.MetaArgumentsOrderRule
	LifecycleAttributeOrderRule = rules.LifecycleAttributeOrderRule
	NestedBlockOrderRule        = rules.NestedBlockOrderRule
	OneLineAttributeSpacingRule = rules.OneLineAttributeSpacingRule

	// File organization rules
	VariablesInFileRule         = rules.VariablesInFileRule
	OutputsInFileRule           = rules.OutputsInFileRule
	ProvidersInFileRule         = rules.ProvidersInFileRule
	ScopedFileOrganizationRule  = rules.ScopedFileOrganizationRule
	TerraformFilesStructureRule = rules.TerraformFilesStructureRule

	// Documentation rules
	RequireVariableDescriptionRule = rules.RequireVariableDescriptionRule
	RequireOutputDescriptionRule   = rules.RequireOutputDescriptionRule
	RequireVariableTypeRule        = rules.RequireVariableTypeRule

	// Comment and format rules
	CommentSyntaxRule           = rules.CommentSyntaxRule
	NoTrailingWhitespaceRule    = rules.NoTrailingWhitespaceRule
	ConsistentQuotesRule        = rules.ConsistentQuotesRule
	NoConsecutiveBlankLinesRule = rules.NoConsecutiveBlankLinesRule
)
