package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachCountFirstRule(t *testing.T) {
	rule := &ForEachCountFirstRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.for-each-count-first", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "for_each is first",
			content: `resource "aws_instance" "example" {
  for_each      = var.instances
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "for_each is not first",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  for_each      = var.instances
  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name: "count is first",
			content: `resource "aws_instance" "example" {
  count         = 3
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "count is not first",
			content: `resource "aws_instance" "example" {
  ami   = "ami-123"
  count = 3
}`,
			wantFindings: 1,
		},
		{
			name: "no for_each or count",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "module with for_each not first",
			content: `module "example" {
  source   = "./module"
  for_each = var.items
}`,
			wantFindings: 1,
		},
		{
			name: "data source with count not first",
			content: `data "aws_ami" "example" {
  most_recent = true
  count       = 2
}`,
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix reorders for_each to first", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  ami      = "ami-123"
  for_each = var.instances
  name     = "test"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)
		// for_each should be near the beginning
		resultStr := string(result.Edits[0].Replacement)
		forEachIdx := indexOf(resultStr, "for_each")
		amiIdx := indexOf(resultStr, "ami")
		assert.Less(t, forEachIdx, amiIdx)
	})

	t.Run("Fix moves for_each to front and preserves leading comments on neighbors", func(t *testing.T) {
		// The historical pipeline-spanning version of this assertion (full
		// canonical ordering of source/tags/depends_on) lives as an engine-level
		// integration test in internal/engines/style/style_test.go now that the
		// rule's Fix is narrow: only for_each/count relocates here. This sibling
		// asserts the leading-comment-carriage invariant on a Move that actually
		// fires — for_each is the second attribute, with a leading comment on
		// the third — so the Move puts for_each at index 0 while the comment
		// stays attached to the attribute it belongs to.
		content := `resource "aws_instance" "example" {
  ami            = "ami-123"
  for_each       = var.instances
  # This is an important comment about the instance
  # It spans multiple lines
  instance_type  = "t3.micro"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result, "for_each not at index 0 must produce a FixResult")
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		assert.Contains(t, resultStr, "# This is an important comment about the instance")
		assert.Contains(t, resultStr, "# It spans multiple lines")

		// for_each moved to first; ami follows; the two-line comment stays
		// attached to instance_type.
		forEachIdx := indexOf(resultStr, "for_each")
		amiIdx := indexOf(resultStr, "ami")
		commentIdx := indexOf(resultStr, "# This is an important comment")
		instanceTypeIdx := indexOf(resultStr, "instance_type")
		assert.Less(t, forEachIdx, amiIdx, "for_each should be before ami after Move")
		assert.Less(t, amiIdx, commentIdx, "ami should be before the comment block")
		assert.Less(t, commentIdx, instanceTypeIdx, "comment should stay attached above instance_type")
	})

	t.Run("Fix moves count to first when for_each is absent", func(t *testing.T) {
		// Pins the count branch of moveForEachOrCountToFront. A typo in the
		// attribute name on that branch would otherwise be invisible since
		// every other Fix test uses for_each.
		content := `resource "aws_instance" "example" {
  ami   = "ami-123"
  count = 2
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		out := string(result.Edits[0].Replacement)
		countIdx := indexOf(out, "count")
		amiIdx := indexOf(out, "ami")
		assert.Less(t, countIdx, amiIdx, "count should be at front when for_each is absent")
	})

	t.Run("Fix prefers for_each over count when both are present", func(t *testing.T) {
		// for_each and count cannot coexist in valid Terraform, but Check's
		// historical policy flags for_each first; Fix mirrors that precedence
		// defensively so the rule has a single canonical winner.
		content := `resource "aws_instance" "example" {
  count    = 2
  for_each = var.instances
  ami      = "ami-123"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		out := string(result.Edits[0].Replacement)
		forEachIdx := indexOf(out, "for_each")
		countIdx := indexOf(out, "count")
		assert.Less(t, forEachIdx, countIdx, "for_each should win over count when both are present")
	})
}

// TestForEachCountFirst_ParseError_FixIsNoOp covers the cst.Build parse-error
// branch. On a partial tree, Fix must return (nil, nil).
func TestForEachCountFirst_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &ForEachCountFirstRule{}
	content := "resource \"aws_instance\" \"x\" {\n  ami = \"ami-123\"\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err, "Fix must swallow parse errors; Check surfaces them")
	assert.Nil(t, result)
}

func TestLifecycleAtEndRule(t *testing.T) {
	rule := &LifecycleAtEndRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.lifecycle-at-end", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
		wantMessage  string
	}{
		{
			name: "lifecycle at end",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "lifecycle not at end",
			content: `resource "aws_instance" "example" {
  lifecycle {
    prevent_destroy = true
  }
  ami = "ami-123"
}`,
			wantFindings: 1,
			wantMessage:  "end of the resource block",
		},
		{
			name: "no lifecycle",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "data block lifecycle not at end",
			content: `data "aws_ami" "example" {
  lifecycle {
    postcondition {
      condition     = self.id != ""
      error_message = "AMI not found"
    }
  }
  most_recent = true
  owners      = ["amazon"]
}`,
			wantFindings: 1,
			wantMessage:  "end of the data block",
		},
		{
			name: "data block lifecycle at end",
			content: `data "aws_ami" "example" {
  most_recent = true
  owners      = ["amazon"]
  lifecycle {
    postcondition {
      condition     = self.id != ""
      error_message = "AMI not found"
    }
  }
}`,
			wantFindings: 0,
		},
		{
			name: "module block lifecycle not at end",
			content: `module "vpc" {
  source = "./vpc"
  lifecycle {
    precondition {
      condition     = var.region != ""
      error_message = "region required"
    }
  }
  cidr_block = "10.0.0.0/16"
}`,
			wantFindings: 1,
			wantMessage:  "end of the module block",
		},
		{
			name: "module block lifecycle at end",
			content: `module "vpc" {
  source     = "./vpc"
  cidr_block = "10.0.0.0/16"
  lifecycle {
    precondition {
      condition     = var.region != ""
      error_message = "region required"
    }
  }
}`,
			wantFindings: 0,
		},
		{
			name: "non-host block (variable) is ignored",
			content: `variable "example" {
  type    = string
  default = "x"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
			if tt.wantMessage != "" && len(findings) > 0 {
				assert.Contains(t, findings[0].Message, tt.wantMessage)
			}
		})
	}

	t.Run("Fix moves lifecycle to end", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  lifecycle {
    prevent_destroy = true
  }
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		// Verify lifecycle is now after ami and instance_type
		resultStr := string(result.Edits[0].Replacement)
		lifecycleIdx := indexOf(resultStr, "lifecycle")
		amiIdx := indexOf(resultStr, "ami")
		assert.Greater(t, lifecycleIdx, amiIdx, "lifecycle should be after ami")
	})

	t.Run("Fix moves lifecycle to end in data block", func(t *testing.T) {
		content := `data "aws_ami" "example" {
  lifecycle {
    postcondition {
      condition     = self.id != ""
      error_message = "AMI not found"
    }
  }
  most_recent = true
  owners      = ["amazon"]
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		lifecycleIdx := indexOf(resultStr, "lifecycle")
		ownersIdx := indexOf(resultStr, "owners")
		assert.Greater(t, lifecycleIdx, ownersIdx, "lifecycle should be after owners in data block")
	})

	t.Run("Fix moves lifecycle to end in module block", func(t *testing.T) {
		content := `module "vpc" {
  source = "./vpc"
  lifecycle {
    precondition {
      condition     = var.region != ""
      error_message = "region required"
    }
  }
  cidr_block = "10.0.0.0/16"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		lifecycleIdx := indexOf(resultStr, "lifecycle")
		cidrIdx := indexOf(resultStr, "cidr_block")
		assert.Greater(t, lifecycleIdx, cidrIdx, "lifecycle should be after cidr_block in module block")
	})

	t.Run("Fix handles same labels across different block types", func(t *testing.T) {
		// Regression: resource and data both labeled "example". Both have a
		// lifecycle that needs moving. The fix must move both independently,
		// matching by (block type, labels), not labels alone.
		content := `resource "aws_instance" "example" {
  lifecycle {
    prevent_destroy = true
  }
  ami = "ami-123"
}

data "aws_ami" "example" {
  lifecycle {
    postcondition {
      condition     = self.id != ""
      error_message = "AMI not found"
    }
  }
  most_recent = true
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		// Resource block: lifecycle should come after ami.
		amiIdx := indexOf(resultStr, "ami =")
		// First lifecycle occurrence is in the resource block.
		firstLifecycleIdx := indexOf(resultStr, "lifecycle")
		assert.Greater(t, firstLifecycleIdx, amiIdx, "resource lifecycle should be after ami")
		// Data block: lifecycle should come after most_recent.
		mostRecentIdx := indexOf(resultStr, "most_recent")
		secondLifecycleIdx := strings.Index(resultStr[firstLifecycleIdx+len("lifecycle"):], "lifecycle") + firstLifecycleIdx + len("lifecycle")
		assert.Greater(t, secondLifecycleIdx, mostRecentIdx, "data lifecycle should be after most_recent")
	})
}

// TestLifecycleAtEnd_WithLeadingComment_TravelsWithBlock pins the carriage
// invariant that motivated the CST migration: a leading comment above a
// lifecycle block must travel with the block when Move relocates it to the
// end. The mechanism (Block.headerRaw encodes leading-comment bytes; Move
// preserves item raw) is shared with TagsAtEndRule and TerraformBlockFirstRule
// — each has a sibling test pinning the same invariant.
func TestLifecycleAtEnd_WithLeadingComment_TravelsWithBlock(t *testing.T) {
	t.Parallel()

	rule := &LifecycleAtEndRule{}
	content := `resource "aws_instance" "example" {
  # Prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, &hcl.File{Body: file.Body})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Edits, 1)

	out := string(result.Edits[0].Replacement)

	// Ordering: ami < instance_type < comment < lifecycle.
	amiIdx := indexOf(out, "ami")
	instanceTypeIdx := indexOf(out, "instance_type")
	commentIdx := indexOf(out, "# Prevent accidental destruction")
	lifecycleIdx := indexOf(out, "lifecycle")

	assert.Greater(t, instanceTypeIdx, amiIdx, "attributes preserve source order")
	assert.Greater(t, commentIdx, instanceTypeIdx, "leading comment moves with the block")
	assert.Greater(t, lifecycleIdx, commentIdx, "lifecycle remains immediately after its leading comment")

	// Comment must NOT also appear in its original position (no duplication).
	assert.Equal(t, 1, strings.Count(out, "# Prevent accidental destruction"),
		"comment should appear exactly once after the move")
}

// TestLifecycleAtEnd_AlreadyAtEnd_FixIsNoOp pins the WholeFileEdit nil-on-
// no-change contract for the canonical input — sibling to the idempotence
// tests on TagsAtEndRule (lines 720, 748) and TerraformBlockFirstRule (line
// 2350). Guards against a future caller assuming Fix always returns a non-nil
// FixResult on success.
func TestLifecycleAtEnd_AlreadyAtEnd_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &LifecycleAtEndRule{}
	content := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  lifecycle {
    prevent_destroy = true
  }
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, &hcl.File{Body: file.Body})
	require.NoError(t, err)
	assert.Nil(t, result, "Fix should return nil when lifecycle is already at end")
}

// TestLifecycleAtEnd_HostBlockWithoutLifecycle_FixIsNoOp covers the branch
// where the walker visits a lifecycle-host block (resource/data/module/check)
// that doesn't contain a lifecycle nested block. moveLifecycleToEnd must
// early-return on the lifecycle == nil path and Fix must surface a nil
// FixResult (no bytes changed) via WholeFileEdit's nil-on-no-change contract.
func TestLifecycleAtEnd_HostBlockWithoutLifecycle_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &LifecycleAtEndRule{}
	content := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
	require.False(t, diags.HasErrors())

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, &hcl.File{Body: file.Body})
	require.NoError(t, err)
	assert.Nil(t, result, "Fix should return nil when no host block contains lifecycle")
}

// TestLifecycleAtEnd_ParseError_FixIsNoOp covers the cst.Build parse-error
// branch. On a partial tree, Fix must return (nil, nil) — Check already
// surfaces the diagnostic and Fix preserves its no-op contract.
func TestLifecycleAtEnd_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &LifecycleAtEndRule{}
	// Unterminated block: cst.Build returns a parse error.
	content := "resource \"aws_instance\" \"x\" {\n  ami = \"ami-123\"\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err, "Fix must swallow parse errors; Check surfaces them")
	assert.Nil(t, result)
}

// TestLifecycleAtEnd_ReadError_FixSurfacesError covers the os.ReadFile error
// branch — Fix must propagate I/O errors to the caller rather than returning
// a partial result.
func TestLifecycleAtEnd_ReadError_FixSurfacesError(t *testing.T) {
	t.Parallel()

	rule := &LifecycleAtEndRule{}
	ctx := &sdk.Context{File: filepath.Join(t.TempDir(), "does-not-exist.tf")}
	result, err := rule.Fix(ctx, nil)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestTagsAtEndRule(t *testing.T) {
	rule := &TagsAtEndRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.tags-at-end", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "tags at end",
			content: `resource "aws_instance" "example" {
  ami  = "ami-123"
  tags = { Name = "test" }
}`,
			wantFindings: 0,
		},
		{
			name: "tags before lifecycle",
			content: `resource "aws_instance" "example" {
  ami  = "ami-123"
  tags = { Name = "test" }
  lifecycle {
    prevent_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "tags after lifecycle",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true
  }
  tags = { Name = "test" }
}`,
			wantFindings: 1,
		},
		{
			name: "no tags",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "labels in module",
			content: `module "example" {
  source = "./module"
  labels = { env = "prod" }
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix reorders tags to end of attributes", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  tags = { Name = "test" }
  ami  = "ami-123"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		// Verify tags is now after ami (at end of attributes)
		resultStr := string(result.Edits[0].Replacement)
		tagsIdx := indexOf(resultStr, "tags")
		amiIdx := indexOf(resultStr, "ami")
		assert.Greater(t, tagsIdx, amiIdx, "tags should be after ami")
	})

	t.Run("Fix preserves leading comments on attributes", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  tags = { Name = "test" }
  # This comment describes the AMI
  ami = "ami-123"
  # This comment describes the instance type
  instance_type = "t3.micro"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		// Comments should be preserved
		assert.Contains(t, resultStr, "# This comment describes the AMI")
		assert.Contains(t, resultStr, "# This comment describes the instance type")
		// Tags should be at end, ami should be before tags
		tagsIdx := indexOf(resultStr, "tags")
		amiIdx := indexOf(resultStr, "ami")
		assert.Less(t, amiIdx, tagsIdx, "ami should be before tags")
	})

	t.Run("Check flags single attr after tags (new threshold > 0)", func(t *testing.T) {
		// The team-tools fixture: only one attribute follows tags. With the old `> 2`
		// threshold this slipped through; with `> 0` the Check fires.
		content := `module "team-tools" {
  source = "./team-tools"
  tags   = { team = "platform" }
  count  = 1
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		// Expect the "near the end" finding; lifecycle is absent so no second finding.
		assert.Len(t, findings, 1)
		assert.Contains(t, findings[0].Message, "near the end")
	})

	t.Run("Check flags trailing nested blocks after tags", func(t *testing.T) {
		// aws_security_group with tags in the middle, ingress/egress AFTER tags but BEFORE lifecycle.
		// Previously countAttrsAfterTags returned 0 (no attrs trailing); now we count blocks too.
		content := `resource "aws_security_group" "vpce" {
  vpc_id      = "vpc-1"
  name        = "x"
  description = "y"
  tags        = { Name = "test" }
  ingress {
    from_port = 80
  }
  egress {
    to_port = 443
  }
  lifecycle {
    prevent_destroy = true
  }
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.Len(t, findings, 1, "trailing non-lifecycle blocks should flag the rule")
		assert.Contains(t, findings[0].Message, "near the end")
	})

	t.Run("Fix moves tags to just before lifecycle, other items stay put", func(t *testing.T) {
		// Mirrors aws_security_group.vpce. Tags moves from position 4 to position 6
		// (between egress and lifecycle). vpc_id, name, description, ingress, egress
		// keep their source positions. Use alignment-agnostic anchors because hclwrite
		// re-aligns `=` columns after the move.
		content := `resource "aws_security_group" "vpce" {
  vpc_id      = "vpc-1"
  name        = "x"
  description = "y"
  tags        = { Name = "test" }
  ingress {
    from_port = 80
  }
  egress {
    to_port = 443
  }
  lifecycle {
    prevent_destroy = true
  }
}
`
		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  vpc_id",
			"\n  name",
			"\n  description",
			"\n  ingress {",
			"from_port = 80",
			"\n  egress {",
			"to_port = 443",
			"\n  tags",
			`Name = "test"`,
			"\n  lifecycle {",
			"prevent_destroy = true",
		})
	})

	t.Run("Fix preserves leading comment on tags when moving", func(t *testing.T) {
		// Mirrors a real ACM module fixture where tags carries a hint comment.
		content := `resource "aws_acm_certificate" "x" {
  domain_name       = "example.com"
  # remove * from the beginning before importing into Vanta
  tags              = { Name = "cert" }
  validation_method = "DNS"
  lifecycle {
    create_before_destroy = true
  }
}
`
		out := runRuleFix(t, rule, content)
		assert.Contains(t, out, "# remove * from the beginning before importing into Vanta")
		assertOrderedSubstrings(t, out, []string{
			"\n  domain_name",
			"\n  validation_method",
			"\n  # remove * from the beginning before importing into Vanta",
			"\n  tags",
			"\n  lifecycle {",
		})
	})

	t.Run("Fix moves tags to end when no lifecycle present", func(t *testing.T) {
		content := `resource "aws_instance" "x" {
  ami           = "ami-123"
  tags          = { Name = "test" }
  instance_type = "t3.medium"
}
`
		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  ami",
			"\n  instance_type",
			"\n  tags",
		})
	})

	t.Run("Fix is idempotent", func(t *testing.T) {
		content := `resource "aws_security_group" "x" {
  vpc_id      = "vpc-1"
  name        = "x"
  description = "y"
  tags        = { Name = "test" }
  ingress {
    from_port = 80
  }
  lifecycle {
    prevent_destroy = true
  }
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(x)) must equal Fix(x)")
	})

	t.Run("Fix is idempotent after first-pass move with leading comment", func(t *testing.T) {
		// Starts from a violating state (tags+comment in the middle) so the first Fix
		// performs a real move. The second Fix must produce identical output.
		content := `resource "aws_security_group" "x" {
  vpc_id      = "vpc-1"
  name        = "x"
  # important: tag this carefully
  tags        = { Name = "test" }
  description = "y"
  ingress {
    from_port = 80
  }
  lifecycle {
    prevent_destroy = true
  }
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(x)) must equal Fix(x) after a real move")
		// Comment must still be present after both passes.
		assert.Contains(t, string(first.Edits[0].Replacement), "# important: tag this carefully")
	})

	t.Run("findTagsAttribute prefers tags over tags_all", func(t *testing.T) {
		// A resource where both tags and tags_all are present should have Fix() operate
		// on `tags`, not `tags_all` (which is provider-managed). The reviewer flagged a
		// non-deterministic map-iteration bug here; this test locks the priority order.
		content := `resource "aws_instance" "x" {
  ami            = "ami-123"
  tags           = { Name = "user-tag" }
  instance_type  = "t3.medium"
  tags_all       = { Name = "user-tag", Inherited = "x" }
}
`
		out := runRuleFix(t, rule, content)

		// `tags` moves to the end; `tags_all` stays in its original (after-`instance_type`)
		// position since findTagsAttribute targets `tags` first.
		assertOrderedSubstrings(t, out, []string{
			"\n  ami",
			"\n  instance_type",
			"\n  tags_all",
			"\n  tags ",
		})
	})

	t.Run("Check emits a single finding when tags is after lifecycle (no double-fire)", func(t *testing.T) {
		// Previously this case produced two findings on the same line ("before lifecycle"
		// + "near the end"). The Check now picks the more specific message and skips the
		// general one.
		content := `resource "aws_instance" "x" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true
  }
  tags = { Name = "test" }
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.Len(t, findings, 1, "should produce exactly one finding, not two, for a single misplacement")
		assert.Contains(t, findings[0].Message, "before lifecycle")
	})
}

// TestTagsAtEnd_BelowLifecycle_RelocatesAbove pins the 2026-05-20 bug fix.
//
// Pre-CST, the line-based Fix short-circuited as "already adjacent" when tags
// was authored below lifecycle because its line-counting check only handled
// one direction (attrEnd >= insertBefore). The CST Move is direction-agnostic,
// so the same call that handles "tags above misplaced" also handles "tags
// below misplaced". The fixture mirrors a terraform/modules/acm/main.tf shape
// that triggered the original no-op.
func TestTagsAtEnd_BelowLifecycle_RelocatesAbove(t *testing.T) {
	rule := &TagsAtEndRule{}
	content := `resource "aws_acm_certificate" "x" {
  domain_name       = "example.com"
  validation_method = "DNS"
  lifecycle {
    create_before_destroy = true
  }
  tags = { Name = "cert" }
}
`
	out := runRuleFix(t, rule, content)

	// tags now sits immediately above lifecycle.
	assertOrderedSubstrings(t, out, []string{
		"\n  domain_name",
		"\n  validation_method",
		"\n  tags",
		"\n  lifecycle {",
		"create_before_destroy = true",
	})

	// Exact-alignment assertions are deliberate: unchanged items must
	// round-trip byte-for-byte through the CST, never re-aligned. Do not
	// relax these to alignment-agnostic anchors — that would weaken
	// coverage of the CST's unchanged-region preservation invariant.
	assert.Contains(t, out, `domain_name       = "example.com"`)
	assert.Contains(t, out, `validation_method = "DNS"`)
	assert.Contains(t, out, `tags = { Name = "cert" }`)
	assert.Contains(t, out, "create_before_destroy = true")

	// No comment loss: the input has none, and none must appear.
	assert.NotContains(t, out, "#")
	assert.NotContains(t, out, "//")
}

// TestTagsAtEnd_BelowLifecycle_WithLeadingComment_RelocatesAbove verifies
// that a leading comment on tags travels with the attribute when the
// direction-agnostic Move relocates tags from below lifecycle to above it.
// The pre-CST line-based path would have re-discovered the comment via line
// scanning; the CST path carries it as part of the attribute's raw bytes.
// The "free carriage" claim of the migration needs explicit coverage for
// the upward-move direction.
func TestTagsAtEnd_BelowLifecycle_WithLeadingComment_RelocatesAbove(t *testing.T) {
	rule := &TagsAtEndRule{}
	content := `resource "aws_acm_certificate" "x" {
  domain_name       = "example.com"
  validation_method = "DNS"
  lifecycle {
    create_before_destroy = true
  }
  # remove * from the beginning before importing into Vanta
  tags = { Name = "cert" }
}
`
	out := runRuleFix(t, rule, content)

	// Both the comment and tags now sit immediately above lifecycle, in
	// their original relative order.
	assertOrderedSubstrings(t, out, []string{
		"\n  domain_name",
		"\n  validation_method",
		"\n  # remove * from the beginning before importing into Vanta",
		"\n  tags",
		"\n  lifecycle {",
	})

	// Comment content preserved verbatim.
	assert.Contains(t, out, "# remove * from the beginning before importing into Vanta")
}

// TestTagsAtEnd_Labels_BelowLifecycle_RelocatesAbove covers the `labels` arm
// of findTagsCSTAttribute's priority list. The pre-existing tags-vs-tags_all
// priority test stops at the first iteration; a labels-only fixture is the
// only way the second iteration's return path is reached.
func TestTagsAtEnd_Labels_BelowLifecycle_RelocatesAbove(t *testing.T) {
	rule := &TagsAtEndRule{}
	content := `module "cluster" {
  source = "./cluster"
  lifecycle {
    create_before_destroy = true
  }
  labels = { env = "prod" }
}
`
	out := runRuleFix(t, rule, content)

	assertOrderedSubstrings(t, out, []string{
		"\n  source",
		"\n  labels",
		"\n  lifecycle {",
		"create_before_destroy = true",
	})
}

// TestTagsAtEnd_Fix_SkipsBlocksWithoutTags pins the no-op path in
// moveTagsAttrToEnd / findTagsCSTAttribute for resource and module blocks
// that have no tags-family attribute. Fix iterates every resource/module
// block (Check's filter is not reused), so this path is real, not defensive.
// The second resource carries tags below lifecycle so Fix produces a real
// edit; the first must round-trip byte-for-byte through the no-op path.
func TestTagsAtEnd_Fix_SkipsBlocksWithoutTags(t *testing.T) {
	rule := &TagsAtEndRule{}
	content := `resource "aws_vpc" "no_tags" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_acm_certificate" "with_tags" {
  domain_name = "example.com"
  lifecycle {
    create_before_destroy = true
  }
  tags = { Name = "cert" }
}
`
	out := runRuleFix(t, rule, content)

	// The first block is untouched: no tags to move, no shape change.
	assert.Contains(t, out, `resource "aws_vpc" "no_tags" {
  cidr_block = "10.0.0.0/16"
}`)

	// The second block has tags relocated above lifecycle.
	assertOrderedSubstrings(t, out, []string{
		"aws_acm_certificate",
		"\n  domain_name",
		"\n  tags",
		"\n  lifecycle {",
	})
}

// TestTagsAtEnd_Fix_ParseErrorReturnsNoOp pins the contract documented on
// Fix: when cst.Build fails to parse, Fix returns (nil, nil) rather than
// surfacing the diagnostic. Check has already reported the parse error, and
// mutating a partial tree would be unsafe.
func TestTagsAtEnd_Fix_ParseErrorReturnsNoOp(t *testing.T) {
	rule := &TagsAtEndRule{}
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	// Unclosed block: hclsyntax cannot produce a usable tree.
	require.NoError(t, os.WriteFile(tmpFile, []byte(`resource "aws_x" "y" {
`), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDependsOnOrderRule(t *testing.T) {
	rule := &DependsOnOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.depends-on-order", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "depends_on at end",
			content: `resource "aws_instance" "example" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
}`,
			wantFindings: 0,
		},
		{
			name: "depends_on not at end",
			content: `resource "aws_instance" "example" {
  depends_on = [aws_vpc.main]
  ami        = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "no depends_on",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix moves depends_on to end", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  depends_on = [aws_vpc.main]
  ami        = "ami-123"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		// Verify depends_on is now after ami
		resultStr := string(result.Edits[0].Replacement)
		dependsOnIdx := indexOf(resultStr, "depends_on")
		amiIdx := indexOf(resultStr, "ami")
		assert.Greater(t, dependsOnIdx, amiIdx, "depends_on should be after ami")
	})

	t.Run("Fix preserves leading comments on attributes", func(t *testing.T) {
		content := `resource "aws_instance" "example" {
  depends_on = [aws_vpc.main]
  # This is the AMI comment
  ami = "ami-123"
  # This describes the instance type
  instance_type = "t3.micro"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		// Comments should be preserved
		assert.Contains(t, resultStr, "# This is the AMI comment")
		assert.Contains(t, resultStr, "# This describes the instance type")
		// depends_on should be at end
		dependsOnIdx := indexOf(resultStr, "depends_on")
		amiIdx := indexOf(resultStr, "ami")
		assert.Greater(t, dependsOnIdx, amiIdx, "depends_on should be after ami")
	})

	t.Run("Check emits a single finding when depends_on is after lifecycle", func(t *testing.T) {
		// Previously this case produced two findings; the single-finding policy now picks
		// the more specific "before lifecycle" message and skips the general one.
		content := `resource "aws_instance" "x" {
  ami = "ami-123"
  lifecycle {
    prevent_destroy = true
  }
  depends_on = [aws_vpc.main]
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.Len(t, findings, 1, "should produce exactly one finding, not two, for a single misplacement")
		assert.Contains(t, findings[0].Message, "before lifecycle")
	})

	t.Run("Check flags trailing nested blocks after depends_on", func(t *testing.T) {
		// New behavior: non-lifecycle nested blocks after depends_on count as "items
		// after depends_on", so the rule flags them even though no attrs trail.
		content := `resource "aws_ecs_service" "elixir" {
  name        = "elixir"
  cluster     = "main"
  depends_on  = [aws_lb_listener.elixir]
  ordered_placement_strategy {
    type = "spread"
  }
  lifecycle {
    create_before_destroy = true
  }
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.Len(t, findings, 1)
		assert.Contains(t, findings[0].Message, "after non-lifecycle nested blocks")
		assert.Equal(t, sdk.SeverityWarning, findings[0].Severity, "trailing-blocks finding should be Warning (Fix will rewrite the file)")
	})

	t.Run("Fix moves depends_on to before lifecycle, sub-blocks stay put", func(t *testing.T) {
		// Mirrors aws_ecs_service.elixir. depends_on moves from position 3 to just
		// before lifecycle. ordered_placement_strategy stays in its source position.
		content := `resource "aws_ecs_service" "elixir" {
  name       = "elixir"
  cluster    = "main"
  depends_on = [aws_lb_listener.elixir]
  ordered_placement_strategy {
    type  = "spread"
    field = "instanceId"
  }
  lifecycle {
    create_before_destroy = true
  }
}
`
		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  name",
			"\n  cluster",
			"\n  ordered_placement_strategy {",
			`type  = "spread"`,
			`field = "instanceId"`,
			"\n  depends_on",
			"\n  lifecycle {",
		})
	})

	t.Run("Fix preserves leading comment on depends_on when moving", func(t *testing.T) {
		content := `resource "aws_ecs_service" "x" {
  name       = "x"
  cluster    = "main"
  # waits for the listener to be ready first
  depends_on = [aws_lb_listener.x]
  ordered_placement_strategy {
    type = "spread"
  }
  lifecycle {
    create_before_destroy = true
  }
}
`
		out := runRuleFix(t, rule, content)
		assert.Contains(t, out, "# waits for the listener to be ready first")
		assertOrderedSubstrings(t, out, []string{
			"\n  name",
			"\n  cluster",
			"\n  ordered_placement_strategy {",
			"\n  # waits for the listener to be ready first",
			"\n  depends_on",
			"\n  lifecycle {",
		})
	})

	t.Run("Fix moves depends_on to end when no lifecycle present", func(t *testing.T) {
		content := `resource "aws_instance" "x" {
  ami           = "ami-123"
  depends_on    = [aws_vpc.main]
  instance_type = "t3.medium"
}
`
		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  ami",
			"\n  instance_type",
			"\n  depends_on",
		})
	})

	t.Run("Fix is idempotent after first-pass move with leading comment", func(t *testing.T) {
		content := `resource "aws_instance" "x" {
  ami = "ami-123"
  # waits for vpc
  depends_on = [aws_vpc.main]
  instance_type = "t3.medium"
  lifecycle {
    prevent_destroy = true
  }
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(x)) must equal Fix(x) after a real move")
		assert.Contains(t, string(first.Edits[0].Replacement), "# waits for vpc")
	})

	t.Run("Check accepts canonical depends_on then tags then lifecycle layout", func(t *testing.T) {
		// Regression lock: countItemsAfterDependsOn must exclude the tags family because
		// the canonical layout places tags between depends_on and lifecycle. Without this
		// exclusion the rule would flag every well-formed resource that also has tags.
		content := `resource "aws_instance" "x" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
  tags       = { Name = "x" }
  lifecycle {
    prevent_destroy = true
  }
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Empty(t, findings, "depends_on → tags → lifecycle is canonical and must not be flagged")
	})

	t.Run("Fix handles no-lifecycle resource with trailing nested blocks", func(t *testing.T) {
		// depends_on followed by a nested block, no lifecycle present. The fix should
		// move depends_on to right before the closing brace.
		content := `resource "aws_ecs_service" "x" {
  name       = "x"
  cluster    = "main"
  depends_on = [aws_lb_listener.x]
  ordered_placement_strategy {
    type = "spread"
  }
}
`
		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  name",
			"\n  cluster",
			"\n  ordered_placement_strategy {",
			"\n  depends_on",
			"\n}",
		})
	})

	t.Run("Fix handles two resources in one file, both needing the move", func(t *testing.T) {
		// Validates the bottom-up sort: rewriting block N must not invalidate the
		// line ranges of blocks above it. Without bottom-up, the second block's
		// recorded range would be stale after the first rewrite.
		content := `resource "aws_instance" "first" {
  ami        = "ami-1"
  depends_on = [aws_vpc.main]
  instance_type = "t3.medium"
}

resource "aws_instance" "second" {
  ami        = "ami-2"
  depends_on = [aws_vpc.main]
  instance_type = "t3.large"
}
`
		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			`"first"`,
			"\n  ami",
			"\n  instance_type",
			"\n  depends_on",
			`"second"`,
			"\n  ami",
			"\n  instance_type",
			"\n  depends_on",
		})
	})

	t.Run("Fix handles multi-line depends_on list intact", func(t *testing.T) {
		// depends_on value spans multiple lines; the entire range must move together.
		content := `resource "aws_instance" "x" {
  ami = "ami-123"
  depends_on = [
    aws_vpc.main,
    aws_subnet.public,
    aws_security_group.app,
  ]
  instance_type = "t3.medium"
}
`
		out := runRuleFix(t, rule, content)
		// All three dependency lines must be present and contiguous after the move.
		assert.Contains(t, out, "aws_vpc.main,")
		assert.Contains(t, out, "aws_subnet.public,")
		assert.Contains(t, out, "aws_security_group.app,")
		assertOrderedSubstrings(t, out, []string{
			"\n  ami",
			"\n  instance_type",
			"\n  depends_on = [",
			"aws_vpc.main,",
			"aws_subnet.public,",
			"aws_security_group.app,",
			"]",
		})
	})

	t.Run("Check and Fix work on module blocks", func(t *testing.T) {
		content := `module "team-tools" {
  source     = "./team-tools"
  depends_on = [aws_iam_role.team]
  count      = 1
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 1, "module with depends_on then count should flag")

		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  source",
			"\n  count",
			"\n  depends_on",
		})
	})

	t.Run("Check and Fix work on data blocks", func(t *testing.T) {
		content := `data "aws_ami" "x" {
  most_recent = true
  depends_on  = [aws_iam_role.x]
  owners      = ["amazon"]
}`
		file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: "test.tf"}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Len(t, findings, 1, "data with depends_on then owners should flag")

		out := runRuleFix(t, rule, content)
		assertOrderedSubstrings(t, out, []string{
			"\n  most_recent",
			"\n  owners",
			"\n  depends_on",
		})
	})

	t.Run("Fix is a no-op (no diff) when depends_on is already adjacent to lifecycle with a blank gap", func(t *testing.T) {
		// Closes the Fix/Check semantic gap the reviewer flagged: previously Fix would
		// run the splice on this layout (because attrEnd+1 != insertBefore), produce a
		// visually-equivalent output, then FormatAndCleanBlankLines would collapse the
		// blank — so the first pass produced a non-trivial diff. The tightened no-op
		// guard now correctly recognizes this as already-canonical.
		content := `resource "aws_instance" "x" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]

  lifecycle {
    prevent_destroy = true
  }
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "Fix should be a no-op when depends_on is already correctly placed (even with blank-line gap) — nil FixResult under the new contract")
	})
}

// TestDependsOn_CanonicalWithTagsBetween_FixIsNoOp pins the
// isDependsOnCanonicallyPlaced helper's tags-family arm. Check accepts
// `depends_on → tags → lifecycle` as canonical (countItemsAfterDependsOn
// excludes the tags family); Fix must match that policy and return nil
// instead of shuffling depends_on past tags. Without the canonical-placement
// guard, MoveBefore(depends_on, lifecycle) would compute target = lifecycle-1
// and reorder the items to `tags → depends_on → lifecycle`, producing a
// spurious diff on Check-clean input.
func TestDependsOn_CanonicalWithTagsBetween_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &DependsOnOrderRule{}
	content := `resource "aws_instance" "x" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
  tags       = { Name = "x" }
  lifecycle {
    prevent_destroy = true
  }
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result, "Fix on canonical depends_on → tags → lifecycle layout must return nil")
}

// TestDependsOn_NoLifecycle_OnlyTagsAfter_FixIsNoOp pins the helper's
// end-of-body arm: when no lifecycle exists and only a tags-family attribute
// follows depends_on, the layout is canonical (mirrors Check, where
// countItemsAfterDependsOn excludes tags-family). Without the guard,
// Move(depends_on, len-1) would shuffle depends_on past tags, producing a
// spurious diff.
func TestDependsOn_NoLifecycle_OnlyTagsAfter_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &DependsOnOrderRule{}
	content := `resource "aws_instance" "x" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
  tags       = { Name = "x" }
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result, "Fix on depends_on → tags (no lifecycle) layout must return nil")
}

// TestDependsOn_StandaloneCommentBetween_FixIsNoOp pins the helper's
// StandaloneComment-skip arm. A bare comment line (not attached to any
// attribute) between depends_on and lifecycle is treated by Check as a
// non-violating intervening item (countItemsAfterDependsOn counts only
// attrs and blocks). The CST encodes it as a *cst.StandaloneComment item
// — distinct from BlankLine — so the helper needs an independent
// passthrough arm.
func TestDependsOn_StandaloneCommentBetween_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &DependsOnOrderRule{}
	content := `resource "aws_instance" "x" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
  # marker between the depends_on attribute and the next block
  lifecycle {
    prevent_destroy = true
  }
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result, "Fix on depends_on → standalone comment → lifecycle layout must return nil")
}

// TestDependsOn_CanonicalWithTagsAllBetween_FixIsNoOp pins the `tags_all`
// arm of isDependsOnCanonicallyPlaced's tags-family enumeration. A typo in
// the literal string would silently break the no-op guarantee for inputs
// that use the provider-managed `tags_all` attribute.
func TestDependsOn_CanonicalWithTagsAllBetween_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &DependsOnOrderRule{}
	content := `resource "aws_instance" "x" {
  ami        = "ami-123"
  depends_on = [aws_vpc.main]
  tags_all   = { Name = "x" }
  lifecycle {
    prevent_destroy = true
  }
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result, "Fix on canonical depends_on → tags_all → lifecycle layout must return nil")
}

// TestDependsOn_CanonicalWithLabelsBetween_FixIsNoOp pins the `labels` arm
// of isDependsOnCanonicallyPlaced's tags-family enumeration (GCP-style
// labels treated the same as AWS-style tags). Together with the tags and
// tags_all sibling tests, the three-member enumeration is fully covered
// against silent string-literal typos.
func TestDependsOn_CanonicalWithLabelsBetween_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &DependsOnOrderRule{}
	content := `resource "google_compute_instance" "x" {
  name       = "x"
  depends_on = [google_network.main]
  labels     = { env = "prod" }
  lifecycle {
    prevent_destroy = true
  }
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result, "Fix on canonical depends_on → labels → lifecycle layout must return nil")
}

func TestSourceVersionGroupedRule(t *testing.T) {
	rule := &SourceVersionGroupedRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.source-version-grouped", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "source and version grouped at start",
			content: `module "example" {
  source  = "./module"
  version = "1.0.0"
  name    = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "source not at start",
			content: `module "example" {
  name    = "test"
  source  = "./module"
  version = "1.0.0"
}`,
			wantFindings: 1,
		},
		{
			name: "version not immediately after source",
			content: `module "example" {
  source  = "./module"
  name    = "test"
  version = "1.0.0"
}`,
			wantFindings: 1,
		},
		{
			name: "source only",
			content: `module "example" {
  source = "./module"
  name   = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "for_each before source is ok",
			content: `module "example" {
  for_each = var.items
  source   = "./module"
  version  = "1.0.0"
}`,
			wantFindings: 0,
		},
		{
			// `for_each` and `count` cannot coexist in valid Terraform, but
			// the Check policy accepts either as a predecessor of source; pin
			// the count-before-source path explicitly so a regression in
			// `allowedBefore` would surface as a Check finding here.
			name: "count before source is ok",
			content: `module "example" {
  count   = 2
  source  = "./module"
  version = "1.0.0"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix moves source and version to start", func(t *testing.T) {
		content := `module "example" {
  name    = "test"
  source  = "./module"
  version = "1.0.0"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		// Verify source is now before name
		resultStr := string(result.Edits[0].Replacement)
		sourceIdx := indexOf(resultStr, "source")
		nameIdx := indexOf(resultStr, "name")
		assert.Less(t, sourceIdx, nameIdx, "source should be before name")
	})

	t.Run("Fix preserves leading comments on attributes", func(t *testing.T) {
		content := `module "example" {
  # Comment about module identifier
  name = "test"
  source = "./module"
  version = "1.0.0"
  # Comment about settings
  config = {}
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		// Comments should be preserved
		assert.Contains(t, resultStr, "# Comment about module identifier")
		assert.Contains(t, resultStr, "# Comment about settings")
		// source should be before name after fix
		sourceIdx := indexOf(resultStr, "source")
		nameIdx := indexOf(resultStr, "name =")
		assert.Less(t, sourceIdx, nameIdx, "source should be before name")
	})

	t.Run("Fix anchors source after for_each", func(t *testing.T) {
		// Pins the MoveAfter(source, forEach) branch of
		// groupSourceVersionInModule. Without this test the for_each anchor
		// path is unexercised at the Fix level.
		content := `module "example" {
  for_each = var.items
  name     = "test"
  source   = "./module"
  version  = "1.0.0"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		out := string(result.Edits[0].Replacement)
		forEachIdx := indexOf(out, "for_each")
		sourceIdx := indexOf(out, "source")
		nameIdx := indexOf(out, "name")
		versionIdx := indexOf(out, "version")
		assert.Less(t, forEachIdx, sourceIdx, "for_each stays before source")
		assert.Less(t, sourceIdx, nameIdx, "source moves after for_each but before name")
		assert.Less(t, versionIdx, nameIdx, "version follows source, both before name")
	})

	t.Run("Fix anchors source after count", func(t *testing.T) {
		// Same shape as the for_each test but exercising the
		// else-if-count branch of groupSourceVersionInModule. Terraform-
		// invalid but the rule is defensive.
		content := `module "example" {
  count   = 2
  name    = "test"
  source  = "./module"
  version = "1.0.0"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		out := string(result.Edits[0].Replacement)
		countIdx := indexOf(out, "count")
		sourceIdx := indexOf(out, "source")
		nameIdx := indexOf(out, "name")
		assert.Less(t, countIdx, sourceIdx, "count stays before source")
		assert.Less(t, sourceIdx, nameIdx, "source moves after count but before name")
	})

	t.Run("Fix is a no-op when source and version are already canonical", func(t *testing.T) {
		// Pins the WholeFileEdit nil-on-no-change contract for canonical
		// input. Sibling of the AlreadyAtEnd tests in TagsAtEndRule /
		// LifecycleAtEndRule / DependsOnOrderRule.
		content := `module "example" {
  source  = "./module"
  version = "1.0.0"
  name    = "test"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "already-canonical input must produce no edits")
	})

	t.Run("Fix is a no-op when source is absent", func(t *testing.T) {
		// Pins the early return in groupSourceVersionInModule when no
		// source is present. WholeFileEdit must return nil since file
		// bytes do not change.
		content := `module "example" {
  version = "1.0.0"
  name    = "test"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "source-absent module must produce no edits")
	})
}

// TestSourceVersionGrouped_ParseError_FixIsNoOp covers the cst.Build parse-
// error branch. On a partial tree, Fix must return (nil, nil).
func TestSourceVersionGrouped_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &SourceVersionGroupedRule{}
	content := "module \"example\" {\n  source = \"./module\"\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestVariableOrderRule(t *testing.T) {
	rule := &VariableOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.variable-order", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "correct order",
			content: `variable "example" {
  description = "Example variable"
  type        = string
  default     = "value"
}`,
			wantFindings: 0,
		},
		{
			name: "type before description",
			content: `variable "example" {
  type        = string
  description = "Example variable"
}`,
			wantFindings: 1,
		},
		{
			name: "default before type",
			content: `variable "example" {
  default = "value"
  type    = string
}`,
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	fixTests := []struct {
		name      string
		input     string
		wantOrder []string // substrings that must appear in this top-down order in the output
		wantKeep  []string // substrings that must remain present anywhere
		wantNoOp  bool     // true when the input is already canonical; Fix must return nil
	}{
		{
			name: "type before description gets reordered",
			input: `variable "example" {
  type        = string
  description = "Example variable"
  default     = "value"
}
`,
			wantOrder: []string{`description = "Example variable"`, `type        = string`, `default     = "value"`},
		},
		{
			name: "heredoc description survives reorder",
			input: `variable "example" {
  type        = string
  description = <<-EOT
    A multi-line
    description with # hashes and // slashes
  EOT
  default     = "value"
}
`,
			wantOrder: []string{
				`description = <<-EOT`,
				`A multi-line`,
				`description with # hashes and // slashes`,
				`EOT`,
				`type        = string`,
				`default     = "value"`,
			},
			wantKeep: []string{`# hashes and // slashes`},
		},
		{
			name: "multi-line validation block keeps body and moves after attrs",
			input: `variable "name" {
  validation {
    condition     = length(var.name) > 0
    error_message = "must not be empty"
  }
  type        = string
  description = "Name of the thing"
}
`,
			wantOrder: []string{
				`description = "Name of the thing"`,
				`type        = string`,
				`validation {`,
				`condition     = length(var.name) > 0`,
				`error_message = "must not be empty"`,
			},
		},
		{
			name: "validation-only variable preserves block",
			input: `variable "name" {
  type = string
  validation {
    condition     = length(var.name) > 0
    error_message = "must not be empty"
  }
}
`,
			wantOrder: []string{`type = string`, `validation {`, `condition     = length(var.name) > 0`},
			wantNoOp:  true,
		},
		{
			name: "description-only variable left unchanged",
			input: `variable "name" {
  description = "Name of the thing"
}
`,
			wantOrder: []string{`description = "Name of the thing"`},
			wantNoOp:  true,
		},
		{
			name:  "empty variable body left unchanged",
			input: "variable \"name\" {}\n",
			wantOrder: []string{
				`variable "name" {`,
			},
			wantNoOp: true,
		},
		{
			name: "interleaved validation moves to end after all known attrs",
			input: `variable "name" {
  description = "Name"
  validation {
    condition     = length(var.name) > 0
    error_message = "must not be empty"
  }
  type    = string
  default = "x"
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				"\n  default",
				`validation {`,
			},
		},
		{
			name: "multiple validation blocks keep relative order",
			input: `variable "name" {
  validation {
    condition     = length(var.name) > 0
    error_message = "first"
  }
  type = string
  validation {
    condition     = length(var.name) < 64
    error_message = "second"
  }
  description = "Name"
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				`error_message = "first"`,
				`error_message = "second"`,
			},
		},
		{
			name: "all five known attributes reorder to canonical sequence",
			input: `variable "name" {
  nullable    = false
  sensitive   = true
  default     = "x"
  type        = string
  description = "Name"
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				"\n  default",
				"\n  sensitive",
				"\n  nullable",
			},
		},
		{
			name: "heredoc inside validation condition is preserved",
			input: `variable "name" {
  validation {
    condition = (
      length(var.name) > 0 &&
      length(var.name) < 64
    )
    error_message = <<-EOT
      name must be 1-63 characters.
      See policy doc for # rationale.
    EOT
  }
  type        = string
  description = "Name"
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				`validation {`,
				`length(var.name) > 0 &&`,
				`length(var.name) < 64`,
				`name must be 1-63 characters.`,
				`See policy doc for # rationale.`,
			},
			wantKeep: []string{`See policy doc for # rationale.`},
		},
		{
			name: "comment inside validation body survives reorder",
			input: `variable "name" {
  validation {
    # check non-empty
    condition     = length(var.name) > 0
    error_message = "must not be empty"
  }
  type        = string
  description = "Name"
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				`validation {`,
				`# check non-empty`,
				`condition     = length(var.name) > 0`,
			},
		},
		{
			name: "trailing comment on closing brace is preserved",
			input: `variable "name" {
  type        = string
  description = "Name"
} # end of name
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				`} # end of name`,
			},
			wantKeep: []string{`# end of name`},
		},
		{
			name: "orphan comment after last attr is preserved (no following region)",
			input: `variable "name" {
  type        = string
  description = "Name"

  # forgotten note pinned to the bottom
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				`# forgotten note pinned to the bottom`,
			},
			wantKeep: []string{`# forgotten note pinned to the bottom`},
		},
		{
			name: "orphan comment between regions follows reordered next region",
			input: `variable "name" {
  type        = string

  # comment naturally attached to default below
  default     = "x"
  description = "Name"
}
`,
			wantOrder: []string{
				`description = "Name"`,
				"\n  type",
				`# comment naturally attached to default below`,
				"\n  default",
			},
		},
		{
			name: "leading comments on attrs and validation block are preserved",
			input: `variable "name" {
  # validation comment
  validation {
    condition     = length(var.name) > 0
    error_message = "must not be empty"
  }
  # type comment
  type = string
  # description comment
  description = "Name"
}
`,
			wantOrder: []string{
				`# description comment`,
				`description = "Name"`,
				`# type comment`,
				`type = string`,
				`# validation comment`,
				`validation {`,
			},
			wantKeep: []string{
				`# validation comment`,
				`# type comment`,
				`# description comment`,
			},
		},
	}

	for _, tt := range fixTests {
		t.Run("Fix/"+tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.input), 0o644))

			ctx := &sdk.Context{File: tmpFile}
			result, err := rule.Fix(ctx, nil)
			require.NoError(t, err)

			output := tt.input
			if tt.wantNoOp {
				assert.Nil(t, result, "already-canonical input must produce no edits")
			} else {
				require.NotNil(t, result, "mutating case must produce a FixResult")
				require.Len(t, result.Edits, 1)
				output = string(result.Edits[0].Replacement)
			}
			assertOrderedSubstrings(t, output, tt.wantOrder)
			for _, want := range tt.wantKeep {
				assert.Contains(t, output, want, "should retain: %s", want)
			}
		})
	}

	t.Run("Fix is idempotent", func(t *testing.T) {
		input := `variable "name" {
  validation {
    condition     = length(var.name) > 0
    error_message = "must not be empty"
  }
  type        = string
  description = <<-EOT
    Multi-line
    with # marker
  EOT
  default     = "x"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(content)) must equal Fix(content)")
	})

	t.Run("Fix handles multiple variables in one file", func(t *testing.T) {
		input := `variable "first" {
  type        = string
  description = "First var"
}

variable "second" {
  default     = "x"
  type        = string
  description = "Second var"
}
`
		output := runRuleFix(t, rule, input)
		assertOrderedSubstrings(t, output, []string{
			`description = "First var"`,
			`type        = string`,
			`description = "Second var"`,
			`type        = string`,
			`default     = "x"`,
		})
	})
}

// assertOrderedSubstrings asserts that each substring appears in the given top-down order.
func assertOrderedSubstrings(t *testing.T, haystack string, needles []string) {
	t.Helper()
	prev := 0
	prevNeedle := ""
	for _, needle := range needles {
		if prev > len(haystack) {
			t.Fatalf("expected %q to appear after %q but reached end of input:\n%s", needle, prevNeedle, haystack)
		}
		idx := strings.Index(haystack[prev:], needle)
		require.NotEqual(t, -1, idx, "expected %q to appear after %q in:\n%s", needle, prevNeedle, haystack)
		prev += idx + len(needle)
		prevNeedle = needle
	}
}

func TestOutputOrderRule(t *testing.T) {
	rule := &OutputOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.output-order", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "correct order",
			content: `output "example" {
  description = "Example output"
  value       = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "value before description",
			content: `output "example" {
  value       = "test"
  description = "Example output"
}`,
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	outputFixTests := []struct {
		name      string
		input     string
		wantOrder []string
		wantKeep  []string
		wantNoOp  bool // true when the input is already canonical; Fix must return nil
	}{
		{
			name: "value before description gets reordered",
			input: `output "example" {
  value       = "test"
  description = "Example output"
}
`,
			wantOrder: []string{`description = "Example output"`, `value       = "test"`},
		},
		{
			name: "all four known attrs reorder to canonical sequence",
			input: `output "example" {
  depends_on  = [aws_s3_bucket.x]
  sensitive   = true
  value       = "test"
  description = "Example output"
}
`,
			wantOrder: []string{
				`description = "Example output"`,
				"\n  value",
				"\n  sensitive",
				"\n  depends_on",
			},
		},
		{
			name: "precondition block moves after attrs",
			input: `output "example" {
  precondition {
    condition     = var.x != ""
    error_message = "x must be set"
  }
  value       = "test"
  description = "Example output"
}
`,
			wantOrder: []string{
				`description = "Example output"`,
				"\n  value",
				`precondition {`,
				`condition     = var.x != ""`,
			},
		},
		{
			name: "heredoc inside value attribute is preserved",
			input: `output "example" {
  value       = <<-EOT
    A multi-line value
    with # markers
  EOT
  description = "Example output"
}
`,
			wantOrder: []string{
				`description = "Example output"`,
				`value       = <<-EOT`,
				`A multi-line value`,
				`with # markers`,
				`EOT`,
			},
			wantKeep: []string{`with # markers`},
		},
		{
			name: "leading comments on attrs and precondition are preserved",
			input: `output "example" {
  # precondition comment
  precondition {
    condition     = var.x != ""
    error_message = "x must be set"
  }
  # value comment
  value = "test"
  # description comment
  description = "Example output"
}
`,
			wantOrder: []string{
				`# description comment`,
				`description = "Example output"`,
				`# value comment`,
				`value = "test"`,
				`# precondition comment`,
				`precondition {`,
			},
		},
		{
			name: "orphan trailing comment is preserved",
			input: `output "example" {
  value       = "test"
  description = "Example output"

  # forgotten trailing note
}
`,
			wantOrder: []string{
				`description = "Example output"`,
				`value       = "test"`,
				`# forgotten trailing note`,
			},
		},
		{
			// Pins the WholeFileEdit nil-on-no-change contract for the
			// trivially-canonical case where reorderOutputBody finds only
			// `value` and Move(value, 0) is a no-op.
			name: "value-only output left unchanged",
			input: `output "example" {
  value = "test"
}
`,
			wantOrder: []string{`value = "test"`},
			wantNoOp:  true,
		},
		{
			// Sibling of VariableOrder's "multiple validation blocks keep
			// relative order": multiple precondition blocks must keep their
			// source order across the canonical reorder so error messages
			// remain in their authored order.
			name: "multiple precondition blocks keep relative order",
			input: `output "example" {
  precondition {
    condition     = var.x != ""
    error_message = "first"
  }
  value       = "test"
  precondition {
    condition     = var.y != ""
    error_message = "second"
  }
  description = "Example output"
}
`,
			wantOrder: []string{
				`description = "Example output"`,
				`value       = "test"`,
				`error_message = "first"`,
				`error_message = "second"`,
			},
		},
	}

	for _, tt := range outputFixTests {
		t.Run("Fix/"+tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.input), 0o644))

			ctx := &sdk.Context{File: tmpFile}
			result, err := rule.Fix(ctx, nil)
			require.NoError(t, err)

			output := tt.input
			if tt.wantNoOp {
				assert.Nil(t, result, "already-canonical input must produce no edits")
			} else {
				require.NotNil(t, result, "mutating case must produce a FixResult")
				require.Len(t, result.Edits, 1)
				output = string(result.Edits[0].Replacement)
			}
			assertOrderedSubstrings(t, output, tt.wantOrder)
			for _, want := range tt.wantKeep {
				assert.Contains(t, output, want, "should retain: %s", want)
			}
		})
	}

	t.Run("Fix is idempotent", func(t *testing.T) {
		input := `output "example" {
  precondition {
    condition     = var.x != ""
    error_message = "x must be set"
  }
  depends_on  = [aws_s3_bucket.x]
  sensitive   = true
  value       = "test"
  description = "Example output"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(content)) must equal Fix(content)")
	})

	t.Run("Fix handles multiple outputs in one file", func(t *testing.T) {
		input := `output "first" {
  value       = "1"
  description = "First"
}

output "second" {
  sensitive   = true
  value       = "2"
  description = "Second"
}
`
		assertOrderedSubstrings(t, runRuleFix(t, rule, input), []string{
			`description = "First"`,
			`value       = "1"`,
			`description = "Second"`,
			`value       = "2"`,
			`sensitive   = true`,
		})
	})
}

// TestOutputOrder_ParseError_FixIsNoOp covers the cst.Build parse-error
// branch. On a partial tree, Fix must return (nil, nil).
func TestOutputOrder_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &OutputOrderRule{}
	content := "output \"example\" {\n  value = \"test\"\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestVariableOrder_ParseError_FixIsNoOp covers the cst.Build parse-error
// branch. On a partial tree, Fix must return (nil, nil).
func TestVariableOrder_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &VariableOrderRule{}
	content := "variable \"name\" {\n  type = string\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestTerraformBlockFirstRule(t *testing.T) {
	rule := &TerraformBlockFirstRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.terraform-block-first", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "terraform block first",
			content: `terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "terraform block not first",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}

terraform {
  required_version = ">= 1.0"
}`,
			wantFindings: 1,
		},
		{
			name: "no terraform block",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix moves terraform block to first position", func(t *testing.T) {
		input := `resource "aws_instance" "x" {
  ami = "ami-123"
}

terraform {
  required_version = ">= 1.0"
}
`
		out := runRuleFix(t, rule, input)
		tfIdx := strings.Index(out, "terraform {")
		rIdx := strings.Index(out, "resource ")
		require.NotEqual(t, -1, tfIdx)
		require.NotEqual(t, -1, rIdx)
		assert.Less(t, tfIdx, rIdx, "terraform must precede resource after fix")
	})

	t.Run("Fix preserves comments inside resource bodies", func(t *testing.T) {
		input := `resource "aws_instance" "x" {
  # important: choose AMI carefully
  ami           = "ami-123"
  instance_type = "t3.medium" # production size
}

terraform {
  required_version = ">= 1.0"
}
`
		out := runRuleFix(t, rule, input)
		assert.Contains(t, out, "# important: choose AMI carefully")
		assert.Contains(t, out, "# production size")
		// The buggy old helper would reshuffle attrs via map iteration; line-range never touches block bodies.
		amiIdx := strings.Index(out, "ami           = ")
		instIdx := strings.Index(out, "instance_type = ")
		require.NotEqual(t, -1, amiIdx)
		require.NotEqual(t, -1, instIdx)
		assert.Less(t, amiIdx, instIdx, "resource body attribute order must be untouched")
	})

	t.Run("Fix preserves standalone comments above blocks", func(t *testing.T) {
		// Mirrors a real-world backup-style file: standalone section header comments
		// and an external link comment must remain anchored to their blocks after reorder.
		//nolint:dupword // HCL content intentionally contains repeated identifiers
		input := `# https://docs.example.com/backup-policy

# Section: SNS Notifications
resource "aws_sns_topic" "backup" {
  name = "backup"
}

# A note about the module
module "backup_vault" {
  source = "./vault"
  name   = "default"
}

terraform {
  required_version = ">= 1.0"
}
`
		out := runRuleFix(t, rule, input)
		// The Slab-style URL comment lives in the file header.
		assert.Contains(t, out, "# https://docs.example.com/backup-policy")
		// Section header travels with its sns_topic resource.
		assert.Contains(t, out, "# Section: SNS Notifications")
		// Module-attached comment stays with the module.
		assert.Contains(t, out, "# A note about the module")
		// Resulting order: terraform, resource, module
		assertOrderedSubstrings(t, out, []string{
			"terraform {",
			"# Section: SNS Notifications",
			`resource "aws_sns_topic"`,
			"# A note about the module",
			`module "backup_vault"`,
		})
	})

	t.Run("Fix is idempotent", func(t *testing.T) {
		input := `resource "aws_instance" "x" {
  ami = "ami-123"
}

terraform {
  required_version = ">= 1.0"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(content)) must equal Fix(content)")
	})

	t.Run("Fix surfaces read error for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := &sdk.Context{File: filepath.Join(tmpDir, "does-not-exist.tf")}
		result, err := rule.Fix(ctx, nil)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Fix returns no-op on parse error", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "broken.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte("terraform {\n"), 0o644))
		result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Fix returns no-op when no terraform block is present", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "no-terraform.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(`resource "aws_instance" "x" {
  ami = "ami-123"
}
`), 0o644))
		result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

// TestTerraformBlockFirst_FloatingSectionHeader_Survives pins the 2026-05-20
// floating-section-header bug fix. The fixture mirrors a workera-iac
// terraform/modules/backup/main.tf shape: a standalone `### SNS Notifications`
// header comment sits between two resource blocks, separated by blank lines
// on both sides, and a misplaced terraform block trails them. Pre-CST, the
// line-based reorder helper folded the standalone comment into the
// surrounding move and lost it from the output. The CST Move targets only
// the terraform block; StandaloneComment items keep their content and stay
// flanked by their blank lines.
func TestTerraformBlockFirst_FloatingSectionHeader_Survives(t *testing.T) {
	rule := &TerraformBlockFirstRule{}
	content := `resource "aws_backup_vault" "primary" {
  name = "primary"
}

### SNS Notifications

resource "aws_sns_topic" "backup" {
  name = "backup-events"
}

terraform {
  required_version = ">= 1.0"
}
`
	out := runRuleFix(t, rule, content)

	// terraform block is now first.
	assertOrderedSubstrings(t, out, []string{
		"terraform {",
		`resource "aws_backup_vault"`,
		"### SNS Notifications",
		`resource "aws_sns_topic"`,
	})

	// The standalone section header is still present, verbatim.
	assert.Contains(t, out, "### SNS Notifications")

	// The standalone comment is still flanked by blank lines on both sides
	// — the byte-exact sandwich that defines it as standalone (vs a leading
	// comment attached to the next block). Pre-CST, the line-based reorder
	// could collapse one or both blank lines as a side-effect of the move,
	// dropping the comment into the next block's leading-comment slot or
	// losing it outright. The CST keeps StandaloneComment a distinct item
	// with its own raw bytes — including the surrounding blank lines.
	assert.Contains(t, out, "\n\n### SNS Notifications\n\n")

	// Resource bodies are untouched: byte-for-byte preservation invariant.
	assert.Contains(t, out, `name = "primary"`)
	assert.Contains(t, out, `name = "backup-events"`)

	// Idempotence: a second pass on the already-corrected output is a no-op
	// (Fix returns nil), so the standalone comment must not be perturbed by a
	// re-run after the first fix lands.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(out), 0o644))
	second, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, second, "Fix(Fix(content)) must equal Fix(content)")
}

// TestTerraformBlockFirst_LeadingCommentOnTerraform_TravelsWithBlock verifies
// the symmetric carriage invariant for the terraform block itself: a leading
// comment immediately above `terraform {` with no blank line between them
// (the binding rule under DefaultTopLevelPolicy: StrictAdjacency=true treats
// "no blank above" as always-attach in classifyGap) is carried along when the
// block moves to position 0. The existing "Fix preserves standalone comments
// above blocks" subtest covers leading comments on resource/module blocks via
// the same mechanism; this one pins the same contract for the rule's actual
// subject — the terraform block itself.
func TestTerraformBlockFirst_LeadingCommentOnTerraform_TravelsWithBlock(t *testing.T) {
	rule := &TerraformBlockFirstRule{}
	content := `resource "aws_instance" "x" {
  ami = "ami-123"
}
# Configure Terraform settings
terraform {
  required_version = ">= 1.0"
}
`
	out := runRuleFix(t, rule, content)

	assertOrderedSubstrings(t, out, []string{
		"# Configure Terraform settings",
		"terraform {",
		`resource "aws_instance"`,
	})

	// The leading comment must immediately precede the terraform header with
	// no intervening blank line — exactly as authored.
	assert.Contains(t, out, "# Configure Terraform settings\nterraform {")
}

func TestProviderBlockOrderRule(t *testing.T) {
	rule := &ProviderBlockOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.provider-block-order", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "provider after terraform, before resources",
			content: `terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "provider before terraform",
			content: `provider "aws" {
  region = "us-east-1"
}

terraform {
  required_version = ">= 1.0"
}`,
			wantFindings: 1,
		},
		{
			name: "provider after resources",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}

provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.content), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: "test.tf"}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix reorders terraform, resource, provider to terraform, provider, resource", func(t *testing.T) {
		input := `terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "x" {
  ami = "ami-123"
}

provider "aws" {
  region = "us-east-1"
}
`
		out := runRuleFix(t, rule, input)
		assertOrderedSubstrings(t, out, []string{
			"terraform {",
			`provider "aws"`,
			`resource "aws_instance"`,
		})
	})

	t.Run("Fix preserves all comments through reorder", func(t *testing.T) {
		// Comments with a blank line above are StandaloneComments under
		// DefaultTopLevelPolicy (StrictAdjacency=true) and stay in their slot
		// as blocks reshuffle past them — the same mechanism that fixes the
		// floating section-header bug in style.terraform-block-first. All four
		// comments are still present in the output; only their position
		// relative to blocks changes from the pre-CST line-based reorder.
		//nolint:dupword // HCL content intentionally contains repeated block-type identifiers
		input := `# File-level note at top.

# About the resource
resource "aws_instance" "x" {
  ami = "ami-123"
}

# About the provider
provider "aws" {
  region = "us-east-1"
}

# About terraform
terraform {
  required_version = ">= 1.0"
}
`
		out := runRuleFix(t, rule, input)
		assert.Contains(t, out, "# File-level note at top.")
		assert.Contains(t, out, "# About the resource")
		assert.Contains(t, out, "# About the provider")
		assert.Contains(t, out, "# About terraform")
		assertOrderedSubstrings(t, out, []string{
			"# File-level note at top.",
			"# About the resource",
			"terraform {",
			`provider "aws"`,
			`resource "aws_instance"`,
			"# About the provider",
			"# About terraform",
		})
	})

	t.Run("Fix does not touch attributes inside untouched blocks", func(t *testing.T) {
		input := `resource "aws_instance" "x" {
  ami           = "ami-123"
  instance_type = "t3.medium"
  tags = {
    Name = "test"
    Env  = "prod"
  }
}

terraform {
  required_version = ">= 1.0"
}
`
		out := runRuleFix(t, rule, input)
		// Resource body order untouched.
		amiIdx := strings.Index(out, "ami           = ")
		instIdx := strings.Index(out, "instance_type = ")
		tagsIdx := strings.Index(out, "tags = {")
		require.NotEqual(t, -1, amiIdx)
		require.NotEqual(t, -1, instIdx)
		require.NotEqual(t, -1, tagsIdx)
		assert.Less(t, amiIdx, instIdx)
		assert.Less(t, instIdx, tagsIdx)
		// Nested map order preserved
		nameIdx := strings.Index(out, `Name = "test"`)
		envIdx := strings.Index(out, `Env  = "prod"`)
		assert.Less(t, nameIdx, envIdx)
	})

	t.Run("Fix is idempotent", func(t *testing.T) {
		input := `provider "aws" {
  region = "us-east-1"
}

terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "x" {
  ami = "ami-123"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		ctx := &sdk.Context{File: tmpFile}
		first, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tmpFile, first.Edits[0].Replacement, 0o644))

		second, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, second, "Fix(Fix(content)) must equal Fix(content)")
	})

	t.Run("Fix is a no-op when blocks are already in canonical order", func(t *testing.T) {
		input := `terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "x" {
  ami = "ami-123"
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

		result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "Fix on already-canonical input must return nil via WholeFileEdit no-change guard")
	})

	t.Run("Fix surfaces read error for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := &sdk.Context{File: filepath.Join(tmpDir, "does-not-exist.tf")}
		result, err := rule.Fix(ctx, nil)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestProviderBlockOrder_NoBlocksInFile_FixIsNoOp pins the firstBlockIdx < 0
// guard: a comment-only file with no *cst.Block items must return (nil, nil)
// without mutation.
func TestProviderBlockOrder_NoBlocksInFile_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &ProviderBlockOrderRule{}
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "comments-only.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte("# just a comment\n"), 0o644))

	result, err := rule.Fix(&sdk.Context{File: tmpFile}, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestProviderBlockOrder_ParseError_FixIsNoOp covers the cst.Build parse-error
// branch. On a partial tree, Fix must return (nil, nil).
func TestProviderBlockOrder_ParseError_FixIsNoOp(t *testing.T) {
	t.Parallel()

	rule := &ProviderBlockOrderRule{}
	content := "provider \"aws\" {\n  region = \"us-east-1\"\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := rule.Fix(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// runRuleFix writes content to a tmp file, runs rule.Fix, and returns the output as a string.
// Asserts exactly one TextEdit in the result, so it cannot be used for no-op cases
// (rules that return nil) or rules that emit multiple edits per pass.
func runRuleFix(t *testing.T, rule sdk.Rule, content string) string {
	t.Helper()
	fixer, ok := rule.(sdk.Fixer)
	require.True(t, ok, "rule must implement sdk.Fixer")

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := &sdk.Context{File: tmpFile}
	result, err := fixer.Fix(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Edits, 1)
	return string(result.Edits[0].Replacement)
}

func TestIsDependsOnRelevantBlock(t *testing.T) {
	tests := []struct {
		blockType string
		expected  bool
	}{
		{"resource", true},
		{"module", true},
		{"data", true},
		{"variable", false},
		{"output", false},
		{"terraform", false},
		{"locals", false},
		{"provider", false},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			result := IsDependsOnRelevantBlock(tt.blockType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAttributeGroupSpacingRule(t *testing.T) {
	rule := &AttributeGroupSpacingRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.attribute-group-spacing", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "properly spaced groups",
			content: `resource "aws_instance" "example" {
  for_each = var.instances

  ami           = "ami-123"
  instance_type = "t2.micro"

  tags = {
    Name = "test"
  }
}`,
			wantFindings: 0,
		},
		{
			name: "missing blank line after for_each",
			content: `resource "aws_instance" "example" {
  for_each      = var.instances
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name: "missing blank line before tags",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  tags = {
    Name = "test"
  }
}`,
			wantFindings: 1,
		},
		{
			name: "module with source/version needs spacing",
			content: `module "example" {
  source  = "./module"
  version = "1.0.0"
  name    = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "module properly spaced",
			content: `module "example" {
  source  = "./module"
  version = "1.0.0"

  name = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "missing blank line before depends_on",
			content: `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  depends_on    = [aws_vpc.main]
}`,
			wantFindings: 1,
		},
		{
			name: "no attributes - no findings",
			content: `resource "aws_instance" "example" {
}`,
			wantFindings: 0,
		},
		{
			name: "single attribute - no findings",
			content: `resource "aws_instance" "example" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings, "unexpected number of findings")
		})
	}
}

func TestAttributeGroupSpacingRule_Fix(t *testing.T) {
	rule := &AttributeGroupSpacingRule{}

	t.Run("adds blank line after for_each", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")

		content := `resource "aws_instance" "example" {
  for_each      = var.instances
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		// Check that blank line was added after for_each
		resultStr := string(result.Edits[0].Replacement)
		assert.Contains(t, resultStr, "for_each      = var.instances\n\n  ami")
	})

	t.Run("adds blank line before tags", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")

		content := `resource "aws_instance" "example" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  tags = {
    Name = "test"
  }
}
`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		// Check that blank line was added before tags
		resultStr := string(result.Edits[0].Replacement)
		assert.Contains(t, resultStr, "instance_type = \"t2.micro\"\n\n  tags")
	})

	// Idempotency: an already-spaced block must produce a nil FixResult. This
	// exercises hasBlankLineBetween's true-return path (the double-insertion
	// guard) and the WholeFileEdit early-return.
	t.Run("Fix is a no-op when blank lines already present", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")

		content := `resource "aws_instance" "example" {
  for_each = var.instances

  ami           = "ami-123"
  instance_type = "t2.micro"

  tags = {
    Name = "test"
  }
}
`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		assert.Nil(t, result, "Fix on already-canonical content must return nil (no edits)")
	})

	// Parse-error contract: a partial tree returns (nil, nil) — Check already
	// surfaced the diagnostic, and Fix must not mutate a broken file.
	t.Run("Fix returns nil result and nil error on parse error", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "broken.tf")

		// Unterminated string in attribute value forces hclsyntax to error.
		content := `resource "aws_instance" "broken" {
  ami = "unterminated
}
`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		ctx := &sdk.Context{File: tmpFile}
		result, err := rule.Fix(ctx, &hcl.File{Body: nil})
		require.NoError(t, err)
		assert.Nil(t, result, "Fix on parse error must return nil result")
	})

	// Multiple insertion points in one block — exercises the reverse-walk
	// algorithm at insertAttributeGroupSpacing across three group boundaries.
	// Each iteration's snapshot index must remain valid under prior inserts.
	t.Run("inserts blank lines at every cross-group boundary in one pass", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")

		content := `resource "aws_instance" "example" {
  for_each      = var.instances
  ami           = "ami-123"
  instance_type = "t2.micro"
  depends_on    = [aws_iam_role.x]
  tags = {
    Name = "test"
  }
}
`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)

		resultStr := string(result.Edits[0].Replacement)
		// Three group boundaries: meta→main, main→dependsOn, dependsOn→tags.
		assert.Contains(t, resultStr, "for_each      = var.instances\n\n  ami",
			"meta-arg / main-attr boundary missing blank line")
		assert.Contains(t, resultStr, "instance_type = \"t2.micro\"\n\n  depends_on",
			"main-attr / depends_on boundary missing blank line")
		assert.Contains(t, resultStr, "depends_on    = [aws_iam_role.x]\n\n  tags",
			"depends_on / tags boundary missing blank line")
	})
}

// Helper function to find index of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
