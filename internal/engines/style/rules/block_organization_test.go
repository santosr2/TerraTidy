package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaArgumentsOrderRule(t *testing.T) {
	rule := &MetaArgumentsOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.meta-arguments-order", rule.Name())
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
			name: "correct order for_each first",
			content: `resource "aws_instance" "web" {
  for_each = var.instances
  provider = aws.west

  ami           = "ami-123"
  instance_type = "t2.micro"

  depends_on = [aws_vpc.main]
}`,
			wantFindings: 0,
		},
		{
			name: "correct order count first",
			content: `resource "aws_instance" "web" {
  count    = 3
  provider = aws.west

  ami           = "ami-123"
  instance_type = "t2.micro"

  depends_on = [aws_vpc.main]
}`,
			wantFindings: 0,
		},
		{
			name: "wrong order depends_on before for_each",
			content: `resource "aws_instance" "web" {
  depends_on = [aws_vpc.main]
  for_each   = var.instances

  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name: "wrong order provider before count",
			content: `resource "aws_instance" "web" {
  provider = aws.west
  count    = 3

  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name: "single meta-argument is fine",
			content: `resource "aws_instance" "web" {
  count = 3

  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "no meta-arguments is fine",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "data source with correct order",
			content: `data "aws_ami" "latest" {
  for_each = var.regions

  most_recent = true

  depends_on = [aws_vpc.main]
}`,
			wantFindings: 0,
		},
		{
			name: "module with correct order",
			content: `module "vpc" {
  for_each = var.vpcs
  source   = "./modules/vpc"

  depends_on = [aws_iam_role.main]
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

	t.Run("Fix reorders meta-arguments", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `resource "aws_instance" "web" {
  depends_on = [aws_vpc.main]
  for_each   = var.instances

  ami           = "ami-123"
  instance_type = "t2.micro"
}`
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Parse the result and verify for_each comes before depends_on
		fixedFile, diags := hclsyntax.ParseConfig(result, tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := fixedFile.Body.(*hclsyntax.Body)
		require.Len(t, body.Blocks, 1)

		block := body.Blocks[0]
		forEachLine := 0
		dependsOnLine := 0
		for name, attr := range block.Body.Attributes {
			if name == "for_each" {
				forEachLine = attr.Range().Start.Line
			}
			if name == "depends_on" {
				dependsOnLine = attr.Range().Start.Line
			}
		}

		assert.Greater(t, dependsOnLine, forEachLine, "depends_on should come after for_each")
	})
}

func TestLifecycleAttributeOrderRule(t *testing.T) {
	rule := &LifecycleAttributeOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.lifecycle-attribute-order", rule.Name())
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
			name: "correct lifecycle attribute order",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
    ignore_changes        = [tags]
  }
}`,
			wantFindings: 0,
		},
		{
			name: "wrong lifecycle attribute order",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    ignore_changes        = [tags]
    create_before_destroy = true
  }
}`,
			wantFindings: 1,
		},
		{
			name: "single lifecycle attribute is fine",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    prevent_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "no lifecycle block is fine",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "data source without lifecycle is fine",
			content: `data "aws_ami" "latest" {
  most_recent = true
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

	t.Run("Fix reorders lifecycle attributes", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		content := `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    ignore_changes        = [tags]
    create_before_destroy = true
  }
}`
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)

		file, diags := hclsyntax.ParseConfig([]byte(content), tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		hclFile := &hcl.File{Body: file.Body}
		ctx := &sdk.Context{File: tmpFile}

		result, err := rule.Fix(ctx, hclFile)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the fixed content has correct order
		fixedFile, diags := hclsyntax.ParseConfig(result, tmpFile, hcl.InitialPos)
		require.False(t, diags.HasErrors())

		body := fixedFile.Body.(*hclsyntax.Body)
		require.Len(t, body.Blocks, 1)

		resourceBlock := body.Blocks[0]
		var lifecycleBlock *hclsyntax.Block
		for _, nested := range resourceBlock.Body.Blocks {
			if nested.Type == "lifecycle" {
				lifecycleBlock = nested
				break
			}
		}
		require.NotNil(t, lifecycleBlock)

		createBeforeDestroyLine := 0
		ignoreChangesLine := 0
		for name, attr := range lifecycleBlock.Body.Attributes {
			if name == "create_before_destroy" {
				createBeforeDestroyLine = attr.Range().Start.Line
			}
			if name == "ignore_changes" {
				ignoreChangesLine = attr.Range().Start.Line
			}
		}

		assert.Greater(t, ignoreChangesLine, createBeforeDestroyLine, "ignore_changes should come after create_before_destroy")
	})
}

func TestNestedBlockOrderRule(t *testing.T) {
	rule := &NestedBlockOrderRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.nested-block-order", rule.Name())
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
			name: "correct nested block order",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  timeouts {
    create = "30m"
  }

  connection {
    type = "ssh"
  }

  provisioner "local-exec" {
    command = "echo hello"
  }

  lifecycle {
    create_before_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "wrong order lifecycle before provisioner",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    create_before_destroy = true
  }

  provisioner "local-exec" {
    command = "echo hello"
  }
}`,
			wantFindings: 1,
		},
		{
			name: "wrong order lifecycle before timeouts",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    create_before_destroy = true
  }

  timeouts {
    create = "30m"
  }
}`,
			wantFindings: 1,
		},
		{
			name: "single nested block is fine",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"

  lifecycle {
    create_before_destroy = true
  }
}`,
			wantFindings: 0,
		},
		{
			name: "no nested blocks is fine",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
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

	t.Run("Fix returns nil", func(t *testing.T) {
		result, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestOneLineAttributeSpacingRule(t *testing.T) {
	rule := &OneLineAttributeSpacingRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.one-line-attribute-spacing", rule.Name())
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
			name: "all one-line attributes",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name: "multi-line followed by one-line with blank",
			content: `resource "aws_instance" "web" {
  tags = {
    Name = "web"
  }

  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "multi-line followed by one-line without blank",
			content: `resource "aws_instance" "web" {
  tags = {
    Name = "web"
  }
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "no multi-line attributes",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  count         = 3
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			file, diags := hclsyntax.ParseConfig([]byte(tt.content), tmpFile, hcl.InitialPos)
			require.False(t, diags.HasErrors())

			hclFile := &hcl.File{Body: file.Body}
			ctx := &sdk.Context{File: tmpFile}

			findings, err := rule.Check(ctx, hclFile)
			require.NoError(t, err)
			assert.Len(t, findings, tt.wantFindings)
		})
	}

	t.Run("Fix returns nil", func(t *testing.T) {
		result, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}
