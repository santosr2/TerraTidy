// require-tags - TerraTidy Plugin
// Build with: go build -buildmode=plugin -o require-tags.so

package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/plugins"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// PluginMetadata provides information about this plugin.
var PluginMetadata = &plugins.PluginMetadata{
	Name:        "require-tags",
	Version:     "1.0.0",
	Description: "Checks that resources have a tags attribute",
	Author:      "Your Name",
	Type:        plugins.PluginTypeRule,
}

// Plugin implements the RulePlugin interface.
type Plugin struct {
	rules []sdk.Rule
}

// New creates a new instance of the plugin.
func New() plugins.RulePlugin {
	return &Plugin{rules: []sdk.Rule{&RequireTagsRule{}}}
}

// GetRules returns all rules provided by this plugin.
func (p *Plugin) GetRules() []sdk.Rule { return p.rules }

// RequireTagsRule checks that resource blocks include a tags attribute.
type RequireTagsRule struct{}

func (r *RequireTagsRule) Name() string        { return "require-tags" }
func (r *RequireTagsRule) Description() string { return "Resources must have a tags attribute" }

func (r *RequireTagsRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil
	}

	var findings []sdk.Finding
	for _, block := range body.Blocks {
		if block.Type != "resource" {
			continue
		}
		// Guard against blocks with no labels (malformed HCL)
		if len(block.Labels) == 0 {
			continue
		}
		hasTags := false
		for _, attr := range block.Body.Attributes {
			if attr.Name == "tags" {
				hasTags = true
				break
			}
		}
		if !hasTags {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  fmt.Sprintf("Resource %q is missing a tags attribute", block.Labels[0]),
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.DefRange()),
				Severity: sdk.SeverityWarning,
			})
		}
	}
	return findings, nil
}

// Optional: implement sdk.Fixer interface for auto-fix support
// func (r *RequireTagsRule) Fix(ctx *sdk.Context, file *hcl.File) (*sdk.FixResult, error) {
//     // Return nil for no-op, or build a *sdk.FixResult with one or more
//     // sdk.TextEdit byte-range edits to splice into the file.
//     return nil, nil
// }
