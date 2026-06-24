package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/internal/cst"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerragruntIncludeFirstRule(t *testing.T) {
	rule := &TerragruntIncludeFirstRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.terragrunt-include-first", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		filename     string
		content      string
		wantFindings int
	}{
		{
			name:     "canonical order: include, dependency, inputs",
			filename: "terragrunt.hcl",
			content: `include "root" {
  path = find_in_parent_folders()
}

dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
  vpc_id = dependency.vpc.outputs.vpc_id
}
`,
			wantFindings: 0,
		},
		{
			name:     "dependency before include",
			filename: "terragrunt.hcl",
			content: `dependency "vpc" {
  config_path = "../vpc"
}

include "root" {
  path = find_in_parent_folders()
}
`,
			wantFindings: 1,
		},
		{
			name:     "include after locals",
			filename: "terragrunt.hcl",
			content: `locals {
  region = "us-east-1"
}

include "root" {
  path = find_in_parent_folders()
}
`,
			wantFindings: 1,
		},
		{
			name:     "dependency after locals (no include)",
			filename: "terragrunt.hcl",
			content: `locals {
  region = "us-east-1"
}

dependency "vpc" {
  config_path = "../vpc"
}
`,
			wantFindings: 1,
		},
		{
			name:     "pure terraform file with no terragrunt blocks",
			filename: "main.tf",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}

variable "name" {
  type = string
}
`,
			wantFindings: 0,
		},
		{
			name:     "include block in a .tf file out of order",
			filename: "main.tf",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}

include "root" {
  path = find_in_parent_folders()
}
`,
			wantFindings: 1,
		},
		{
			name:     "multiple includes, all before non-include",
			filename: "terragrunt.hcl",
			content: `include "root" {
  path = find_in_parent_folders()
}

include "region" {
  path = find_in_parent_folders("region.hcl")
}

inputs = {
  foo = "bar"
}
`,
			wantFindings: 0,
		},
		{
			name:         "empty file",
			filename:     "terragrunt.hcl",
			content:      ``,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tt.filename, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tt.filename}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix reorders dependency before include into canonical order", func(t *testing.T) {
		input := `dependency "vpc" {
  config_path = "../vpc"
}

include "root" {
  path = find_in_parent_folders()
}
`
		out := runTerragruntFix(t, rule, "terragrunt.hcl", input)
		incIdx := strings.Index(out, "include ")
		depIdx := strings.Index(out, "dependency ")
		require.NotEqual(t, -1, incIdx, "include must remain in output")
		require.NotEqual(t, -1, depIdx, "dependency must remain in output")
		assert.Less(t, incIdx, depIdx, "include must precede dependency after fix")
	})

	t.Run("Fix moves include and dependency ahead of locals and inputs", func(t *testing.T) {
		input := `locals {
  region = "us-east-1"
}

dependency "vpc" {
  config_path = "../vpc"
}

include "root" {
  path = find_in_parent_folders()
}

inputs = {
  region = local.region
}
`
		out := runTerragruntFix(t, rule, "terragrunt.hcl", input)
		incIdx := strings.Index(out, "include ")
		depIdx := strings.Index(out, "dependency ")
		locIdx := strings.Index(out, "locals ")
		require.NotEqual(t, -1, incIdx)
		require.NotEqual(t, -1, depIdx)
		require.NotEqual(t, -1, locIdx)
		assert.Less(t, incIdx, depIdx, "include must precede dependency after fix")
		assert.Less(t, depIdx, locIdx, "dependency must precede locals after fix")
		assert.Contains(t, out, "inputs = {", "inputs attribute survives the reorder")
	})

	t.Run("Fix is no-op on canonical order", func(t *testing.T) {
		input := `include "root" {
  path = find_in_parent_folders()
}

dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
  vpc_id = dependency.vpc.outputs.vpc_id
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "terragrunt.hcl")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "canonical order should produce no edits")
	})

	t.Run("Fix is no-op on pure Terraform file with no Terragrunt blocks", func(t *testing.T) {
		input := `resource "aws_instance" "example" {
  ami = "ami-123"
}

variable "name" {
  type = string
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "main.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "non-Terragrunt files should not be modified")
	})

	t.Run("Fix applies to .tf file when include block is present", func(t *testing.T) {
		input := `resource "aws_instance" "example" {
  ami = "ami-123"
}

include "root" {
  path = find_in_parent_folders()
}
`
		out := runTerragruntFix(t, rule, "main.tf", input)
		incIdx := strings.Index(out, "include ")
		resIdx := strings.Index(out, "resource ")
		require.NotEqual(t, -1, incIdx)
		require.NotEqual(t, -1, resIdx)
		assert.Less(t, incIdx, resIdx, "include must precede resource after fix on .tf file")
	})

	t.Run("Fix preserves standalone section-header comments in place", func(t *testing.T) {
		// Under DefaultTopLevelPolicy a comment separated from a block by a
		// blank line is a StandaloneComment and stays where it is when blocks
		// reorder around it. Assert both survival AND position: the header
		// must land after the moved include/dependency blocks and above the
		// inputs attribute that originally followed it.
		input := `dependency "vpc" {
  config_path = "../vpc"
}

include "root" {
  path = find_in_parent_folders()
}

### Inputs

inputs = {
  vpc_id = dependency.vpc.outputs.vpc_id
}
`
		out := runTerragruntFix(t, rule, "terragrunt.hcl", input)
		assertOrderedSubstrings(t, out, []string{
			"include ",
			"dependency ",
			"### Inputs",
			"inputs = {",
		})
	})

	t.Run("Fix reorders dependency-only when no include blocks present", func(t *testing.T) {
		input := `locals {
  region = "us-east-1"
}

dependency "vpc" {
  config_path = "../vpc"
}
`
		out := runTerragruntFix(t, rule, "terragrunt.hcl", input)
		assertOrderedSubstrings(t, out, []string{
			"dependency ",
			"locals ",
		})
	})

	t.Run("Fix is idempotent", func(t *testing.T) {
		input := `locals {
  region = "us-east-1"
}

dependency "vpc" {
  config_path = "../vpc"
}

include "root" {
  path = find_in_parent_folders()
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "terragrunt.hcl")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		firstResult, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, firstResult)
		require.Len(t, firstResult.Edits, 1)
		// Persist the first-pass output so the second pass reads canonical input.
		require.NoError(t, os.WriteFile(tmpFile, firstResult.Edits[0].Replacement, 0o644))

		secondResult, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, secondResult, "second Fix pass on canonical input must be a no-op")
	})
}

// TestTerragruntIncludeFirst_ParseError_FixIsNoOp covers the cst.Build
// parse-error branch. On a partial tree, Fix must return (nil, nil) — Check
// already surfaces the diagnostic and Fix preserves its no-op contract.
func TestTerragruntIncludeFirst_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &TerragruntIncludeFirstRule{}
	// Unterminated block: cst.Build returns a parse error.
	content := "include \"root\" {\n  path = find_in_parent_folders()\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.hcl")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err, "Fix must swallow parse errors; Check surfaces them")
	assert.Nil(t, result)
}

// TestTerragruntIncludeFirst_ReadError_FixSurfacesError covers the
// os.ReadFile error branch — Fix must propagate I/O errors to the caller
// rather than returning a partial result.
func TestTerragruntIncludeFirst_ReadError_FixSurfacesError(t *testing.T) {
	t.Parallel()

	rule := &TerragruntIncludeFirstRule{}
	ctx := &sdk.Context{File: filepath.Join(t.TempDir(), "does-not-exist.hcl")}
	result, err := rule.Fix(ctx, nil)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestTerragruntBlocksAreCanonical(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "include, dependency, other",
			content: `include "root" { path = "x" }
dependency "vpc" { config_path = "../vpc" }
locals { region = "us-east-1" }`,
			want: true,
		},
		{
			name: "dependency before include",
			content: `dependency "vpc" { config_path = "../vpc" }
include "root" { path = "x" }`,
			want: false,
		},
		{
			name: "other before include",
			content: `locals { region = "us-east-1" }
include "root" { path = "x" }`,
			want: false,
		},
		{
			name: "only other blocks",
			content: `locals { region = "us-east-1" }
inputs = { foo = "bar" }`,
			want: true,
		},
		{
			name:    "empty",
			content: ``,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, parseErr := cst.Build([]byte(tt.content), "terragrunt.hcl", cst.DefaultTopLevelPolicy())
			require.NoError(t, parseErr)
			got := terragruntBlocksAreCanonical(file.Body)
			assert.Equal(t, tt.want, got)
		})
	}
}

// runTerragruntFix is a Terragrunt variant of runRuleFix that lets the test
// pick the on-disk filename so cst.IsTerragruntFile's filename-check path is
// exercised alongside the block-content path.
func runTerragruntFix(t *testing.T, rule sdk.Rule, filename, content string) string {
	t.Helper()
	fixer, ok := rule.(sdk.Fixer)
	require.True(t, ok, "rule must implement sdk.Fixer")

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, filename)
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := fixer.Fix(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Edits, 1)
	return string(result.Edits[0].Replacement)
}
