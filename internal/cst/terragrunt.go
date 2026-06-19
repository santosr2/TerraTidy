package cst

import "strings"

// TerragruntBlockTypes lists the block types specific to Terragrunt configs.
// Rules that want to discriminate Terragrunt-aware logic — e.g.
// style.terragrunt-include-first — consult this set after Build classifies
// the body items.
//
// "terraform" is included because Terragrunt overloads the block name with
// extra fields (source, extra_arguments, before_hook, after_hook). The same
// block name is also legal in standard Terraform configs, so IsTerragruntFile
// excludes "terraform" alone from triggering detection; otherwise every .tf
// file with a top-level terraform block would falsely register as Terragrunt.
//
// Scope: covers Terragrunt blocks stable through v0.50 (catalog). The v0.67+
// blocks (errors, exclude, feature, engine) and the v0.66+ terragrunt.stack.hcl
// filename are out of scope until a consumer rule needs them; add together
// when that target Terragrunt version becomes the baseline.
//
// This map is treated as immutable by every caller. Mutating it from outside
// the package is undefined behavior; the type stays a plain map only so the
// natural `TerragruntBlockTypes[blockType]` lookup pattern works at call sites.
var TerragruntBlockTypes = map[string]bool{
	"include":      true,
	"dependency":   true,
	"dependencies": true,
	"remote_state": true,
	"generate":     true,
	"catalog":      true,
	"terraform":    true,
}

// IsTerragruntFile reports whether the given file looks like a Terragrunt
// configuration. The heuristic checks the filename first, then scans the
// top-level body for any non-"terraform" Terragrunt block.
//
// Only top-level Body.Items are inspected because Terragrunt's structural
// blocks are always top-level in canonical use. A nested block named
// "include" inside a resource body would not register here, which avoids
// false positives on hand-written Terraform that happens to reuse the name.
//
// A nil f, or a File with nil Body, is tolerated: detection falls back to
// the filename check alone. This lets callers query path-based intent before
// (or independently of) Build.
func IsTerragruntFile(f *File, path string) bool {
	if isTerragruntFilename(path) {
		return true
	}
	if f == nil || f.Body == nil {
		return false
	}
	for _, item := range f.Body.Items {
		block, ok := item.(*Block)
		if !ok {
			continue
		}
		if block.Type == "terraform" {
			continue
		}
		if TerragruntBlockTypes[block.Type] {
			return true
		}
	}
	return false
}

// isTerragruntFilename returns true when path's basename is exactly
// "terragrunt.hcl". Both Unix ('/') and Windows ('\\') path separators are
// recognized so fixtures or fuzz inputs that mix conventions across platforms
// classify the same way; path/filepath.Base would be OS-dependent and would
// fail to split a Windows path on Linux.
func isTerragruntFilename(path string) bool {
	base := path
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		base = path[i+1:]
	}
	return base == "terragrunt.hcl"
}
