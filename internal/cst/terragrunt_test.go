package cst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsTerragruntFile exercises both detection paths (filename and
// top-level block scan) together. Empty content is fed through Build so the
// table also doubles as the "non-Terragrunt but legal HCL" baseline.
func TestIsTerragruntFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		content string
		want    bool
	}{
		// Filename-based detection.
		{"bare terragrunt.hcl", "terragrunt.hcl", "", true},
		{"unix path to terragrunt.hcl", "/some/dir/terragrunt.hcl", "", true},
		{"windows path to terragrunt.hcl", `C:\projects\foo\terragrunt.hcl`, "", true},
		{"deep unix path", "/a/b/c/d/terragrunt.hcl", "", true},

		// Negatives on the filename check.
		{"path with terragrunt.hcl as directory", "/foo/terragrunt.hcl/main.tf", "", false},
		{"prefix-style false match", "myterragrunt.hcl", "", false},
		{"suffix-style false match", "terragrunt.hcl.bak", "", false},
		{"different extension", "terragrunt.tf", "", false},
		{"empty path", "", "", false},

		// Block-based detection (top-level Terragrunt blocks trigger).
		{
			"include block",
			"main.hcl",
			"include \"root\" {\n  path = \"../\"\n}\n",
			true,
		},
		{
			"dependency block",
			"main.hcl",
			"dependency \"vpc\" {\n  config_path = \"../vpc\"\n}\n",
			true,
		},
		{
			"dependencies block",
			"main.hcl",
			"dependencies {\n  paths = [\"../vpc\"]\n}\n",
			true,
		},
		{
			"remote_state block",
			"main.hcl",
			"remote_state {\n  backend = \"s3\"\n}\n",
			true,
		},
		{
			"generate block",
			"main.hcl",
			"generate \"provider\" {\n  path = \"provider.tf\"\n  if_exists = \"overwrite\"\n  contents  = \"\"\n}\n",
			true,
		},
		{
			"catalog block",
			"main.hcl",
			"catalog {\n  urls = [\"github.com/example/modules\"]\n}\n",
			true,
		},
		{
			"include after non-terragrunt block",
			"main.hcl",
			"locals {\n  x = 1\n}\n\ninclude \"root\" {\n  path = \"../\"\n}\n",
			true,
		},
		{
			// Exercises the terraform-skip branch in the block scan together
			// with a real Terragrunt block: the loop must not exit on the
			// terraform block and must reach include on a later iteration.
			"terraform block before include does not short-circuit",
			"main.hcl",
			"terraform {\n  source = \".\"\n}\n\ninclude \"root\" {\n  path = \"../\"\n}\n",
			true,
		},

		// terraform alone must not trigger detection.
		{
			"terraform block alone is neutral",
			"main.tf",
			"terraform {\n  required_version = \">= 1.5\"\n}\n",
			false,
		},
		{
			"resource block is neutral",
			"main.tf",
			"resource \"aws_instance\" \"this\" {\n  ami = \"ami-123\"\n}\n",
			false,
		},
		{
			"locals only is neutral",
			"main.tf",
			"locals {\n  x = 1\n}\n",
			false,
		},

		// Filename wins over neutral content.
		{
			"path wins over terraform-only content",
			"terragrunt.hcl",
			"terraform {\n  required_version = \">= 1.5\"\n}\n",
			true,
		},

		// Nested usage is ignored — only top-level blocks count.
		{
			"nested include block does not trigger",
			"main.tf",
			"resource \"aws_instance\" \"this\" {\n  include {\n    foo = \"bar\"\n  }\n}\n",
			false,
		},
		{
			"include attribute name does not trigger",
			"main.tf",
			"resource \"aws_instance\" \"this\" {\n  include = \"x\"\n}\n",
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, err := Build([]byte(tc.content), tc.path, DefaultTopLevelPolicy())
			require.NoError(t, err)

			assert.Equal(t, tc.want, IsTerragruntFile(f, tc.path))
		})
	}
}

// TestIsTerragruntFile_NilTolerance documents the nil-File contract:
// detection still answers via the filename check. Rules that probe path
// before Build (e.g. to skip work entirely on non-Terragrunt files) depend
// on this behavior.
func TestIsTerragruntFile_NilTolerance(t *testing.T) {
	t.Parallel()

	assert.True(t, IsTerragruntFile(nil, "terragrunt.hcl"), "nil File with terragrunt path should detect")
	assert.True(t, IsTerragruntFile(nil, "/projects/x/terragrunt.hcl"), "nil File with nested path should detect")
	assert.False(t, IsTerragruntFile(nil, "main.tf"), "nil File with non-terragrunt path must not detect")
	assert.False(t, IsTerragruntFile(&File{}, "main.tf"), "empty File with non-terragrunt path must not detect")
	assert.True(t, IsTerragruntFile(&File{}, "terragrunt.hcl"), "empty File with terragrunt path should detect via filename")
}

// TestTerragruntBlockTypes_Members pins the membership of the lookup map so
// a silent rename or addition is caught here rather than as a behavior shift
// inside IsTerragruntFile or downstream rules. If a real new Terragrunt
// block type appears (e.g. a future spec gains one), update both this
// assertion and the doc comment on TerragruntBlockTypes.
func TestTerragruntBlockTypes_Members(t *testing.T) {
	t.Parallel()

	want := map[string]bool{
		"include":      true,
		"dependency":   true,
		"dependencies": true,
		"remote_state": true,
		"generate":     true,
		"catalog":      true,
		"terraform":    true,
	}

	assert.Equal(t, want, TerragruntBlockTypes)
}

// TestIsTerragruntFilename pins the basename-extraction deviation from the
// spec template (which used two strings.HasSuffix checks and would miss a
// bare "terragrunt.hcl" input). Test the helper directly so a regression in
// the separator handling fails here rather than being diluted inside the
// full IsTerragruntFile table.
func TestIsTerragruntFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"terragrunt.hcl", true},
		{"./terragrunt.hcl", true},
		{"/abs/path/terragrunt.hcl", true},
		{`relative\dir\terragrunt.hcl`, true},
		{`C:\projects\foo\terragrunt.hcl`, true},
		{"", false},
		{"myterragrunt.hcl", false},
		{"terragrunt.hcl.bak", false},
		{"terragrunt.tf", false},
		{"/foo/terragrunt.hcl/main.tf", false},
		{"/foo/bar.hcl", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isTerragruntFilename(tc.path))
		})
	}
}
