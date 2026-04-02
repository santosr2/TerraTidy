// require-tags - TerraTidy Plugin
// Build with: go build -buildmode=plugin -o require-tags.so

package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/internal/plugins"
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
		hasTags := false
		for _, attr := range block.Body.Attributes {
			if attr.Name == "tags" {
				hasTags = true
				break
			}
		}
		if !hasTags {
			findings = append(findings, sdk.Finding{
				Rule:     "require-tags",
				Message:  fmt.Sprintf("Resource %q is missing a tags attribute", block.Labels[0]),
				File:     ctx.File,
				Location: block.DefRange(),
				Severity: sdk.SeverityWarning,
			})
		}
	}
	return findings, nil
}

func (r *RequireTagsRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
