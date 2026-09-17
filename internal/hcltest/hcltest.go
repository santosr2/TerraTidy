// Package hcltest holds HCL helpers that tests and fuzz targets in several
// packages share. Only test code imports it.
package hcltest

import (
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// IsValid reports whether data is UTF-8 HCL that hclsyntax parses without
// diagnostics. Fuzz targets use it to narrow random input down to valid HCL.
//
// The UTF-8 check matters: hclsyntax accepts an isolated continuation byte
// (e.g. a lone 0xC9 inside an expression) the first time, but once content is
// spliced in after it, the lexer drifts on that byte and reports "Missing
// newline after argument". The HCL spec requires UTF-8 source, so the filter
// holds input to that rather than depending on the parser's leniency.
func IsValid(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	_, diags := hclsyntax.ParseConfig(data, "fuzz.tf", hcl.InitialPos)
	return !diags.HasErrors()
}

// BlockKey identifies a block by type and labels, in the form
// block:<type>:<label1>:<label2>:...
func BlockKey(blockType string, labels []string) string {
	key := "block:" + blockType
	for _, l := range labels {
		key += ":" + l
	}
	return key
}

// TopLevelNames returns the attributes and blocks declared directly in body,
// keyed as attr:<name> or as BlockKey returns them, so the two kinds never
// collide.
func TopLevelNames(body *hclsyntax.Body) map[string]bool {
	out := make(map[string]bool, len(body.Attributes)+len(body.Blocks))
	for name := range body.Attributes {
		out["attr:"+name] = true
	}
	for _, blk := range body.Blocks {
		out[BlockKey(blk.Type, blk.Labels)] = true
	}
	return out
}

// AllNames is TopLevelNames at every depth. A nested name is prefixed with the
// keys of its enclosing blocks, joined by " > ", so moving an attribute into a
// different block changes its key.
func AllNames(body *hclsyntax.Body) map[string]bool {
	out := make(map[string]bool)
	collectNames(body, "", out)
	return out
}

func collectNames(body *hclsyntax.Body, prefix string, out map[string]bool) {
	for name := range body.Attributes {
		out[prefix+"attr:"+name] = true
	}
	for _, blk := range body.Blocks {
		key := prefix + BlockKey(blk.Type, blk.Labels)
		out[key] = true
		collectNames(blk.Body, key+" > ", out)
	}
}
