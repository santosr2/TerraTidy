package style

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Run(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantFinding bool
		rulePrefix  string
	}{
		{
			name: "proper spacing between blocks",
			content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}

resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			wantErr:     false,
			wantFinding: false,
		},
		{
			name: "missing blank line between blocks",
			content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			wantErr:     false,
			wantFinding: true,
			rulePrefix:  "style.blank-line",
		},
		{
			name: "too many blank lines between blocks",
			content: `resource "aws_instance" "example1" {
  ami = "ami-12345"
}


resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`,
			wantErr:     false,
			wantFinding: true,
			rulePrefix:  "style.blank-line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Create engine
			engine := New(&Config{
				Fix:   false,
				Rules: make(map[string]RuleConfig),
			})

			// Run style checks
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check findings
			if tt.wantFinding {
				if len(findings) == 0 {
					t.Error("expected findings but got none")
					return
				}
			} else {
				if len(findings) != 0 {
					t.Errorf("expected no findings but got %d: %+v", len(findings), findings)
				}
			}
		})
	}
}

func TestBlankLineBetweenBlocksRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "proper spacing",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}

resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: false,
		},
		{
			name: "no spacing",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: true,
		},
		{
			name: "single block",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}
`,
			wantFinding: false,
		},
		{
			name: "comment with one blank line should be valid",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}

# This is a comment about the next block
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: false,
		},
		{
			name: "multiple comments with one blank line should be valid",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}

# Comment line 1
# Comment line 2
# Comment line 3
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: false,
		},
		{
			name: "comment without blank line should report missing blank line",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}
# This is a comment
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: true,
		},
		{
			name: "two blank lines with comment is allowed",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}


# This is a comment
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: false, // 2 blank lines allowed when there's a comment (common section divider pattern)
		},
		{
			name: "three blank lines with comment should report too many",
			content: `resource "aws_instance" "a" {
  ami = "ami-12345"
}



# This is a comment
resource "aws_instance" "b" {
  ami = "ami-67890"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			found := false
			for _, f := range findings {
				if f.Rule == "style.blank-line-between-blocks" {
					found = true
					break
				}
			}

			if found != tt.wantFinding {
				t.Errorf("wanted finding=%v, got finding=%v (findings: %+v)", tt.wantFinding, found, findings)
			}
		})
	}
}

func TestResourceNameConventionRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "valid snake_case",
			content: `resource "aws_instance" "my_instance" {
  ami = "ami-12345"
}
`,
			wantFinding: false,
		},
		{
			name: "invalid camelCase",
			content: `resource "aws_instance" "myInstance" {
  ami = "ami-12345"
}
`,
			wantFinding: true,
		},
		{
			name: "invalid PascalCase",
			content: `resource "aws_instance" "MyInstance" {
  ami = "ami-12345"
}
`,
			wantFinding: true,
		},
		{
			name: "valid data source name",
			content: `data "aws_ami" "latest_ami" {
  most_recent = true
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.resource-name-convention" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestForEachCountFirstRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "for_each is first",
			content: `resource "aws_instance" "example" {
  for_each = var.instances
  ami      = "ami-12345"
}
`,
			wantFinding: false,
		},
		{
			name: "for_each is not first",
			content: `resource "aws_instance" "example" {
  ami      = "ami-12345"
  for_each = var.instances
}
`,
			wantFinding: true,
		},
		{
			name: "count is first",
			content: `resource "aws_instance" "example" {
  count = 3
  ami   = "ami-12345"
}
`,
			wantFinding: false,
		},
		{
			name: "count is not first",
			content: `resource "aws_instance" "example" {
  ami   = "ami-12345"
  count = 3
}
`,
			wantFinding: true,
		},
		{
			name: "no for_each or count",
			content: `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.for-each-count-first" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestLifecycleAtEndRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "lifecycle at end",
			content: `resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  lifecycle {
    create_before_destroy = true
  }
}
`,
			wantFinding: false,
		},
		{
			name: "lifecycle not at end",
			content: `resource "aws_instance" "example" {
  ami = "ami-12345"

  lifecycle {
    create_before_destroy = true
  }

  instance_type = "t2.micro"
}
`,
			wantFinding: true,
		},
		{
			name: "no lifecycle",
			content: `resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.lifecycle-at-end" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestSourceVersionGroupedRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "source and version grouped",
			content: `module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"

  name = "my-vpc"
}
`,
			wantFinding: false,
		},
		{
			name: "source and version not grouped",
			content: `module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  name   = "my-vpc"
  version = "3.0.0"
}
`,
			wantFinding: true,
		},
		{
			name: "source only",
			content: `module "local" {
  source = "./modules/vpc"
  name   = "my-vpc"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.source-version-grouped" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestVariableOrderRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "correct order",
			content: `variable "instance_type" {
  description = "The instance type"
  type        = string
  default     = "t2.micro"
}
`,
			wantFinding: false,
		},
		{
			name: "wrong order - type before description",
			content: `variable "instance_type" {
  type        = string
  description = "The instance type"
  default     = "t2.micro"
}
`,
			wantFinding: true,
		},
		{
			name: "wrong order - default before type",
			content: `variable "instance_type" {
  description = "The instance type"
  default     = "t2.micro"
  type        = string
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.variable-order" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestOutputOrderRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "correct order",
			content: `output "instance_id" {
  description = "The instance ID"
  value       = aws_instance.example.id
  sensitive   = false
}
`,
			wantFinding: false,
		},
		{
			name: "wrong order - value before description",
			content: `output "instance_id" {
  value       = aws_instance.example.id
  description = "The instance ID"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.output-order" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformBlockFirstRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "terraform block first",
			content: `terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-west-2"
}
`,
			wantFinding: false,
		},
		{
			name: "terraform block not first",
			content: `provider "aws" {
  region = "us-west-2"
}

terraform {
  required_version = ">= 1.0"
}
`,
			wantFinding: true,
		},
		{
			name: "no terraform block",
			content: `provider "aws" {
  region = "us-west-2"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.terraform-block-first" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestNoEmptyBlocksRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "non-empty block",
			content: `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`,
			wantFinding: false,
		},
		{
			name: "empty resource block",
			content: `resource "aws_instance" "example" {
}
`,
			wantFinding: true,
		},
		{
			name: "empty lifecycle is allowed",
			content: `resource "aws_instance" "example" {
  ami = "ami-12345"

  lifecycle {
  }
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "style.no-empty-blocks" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestEngine_GetAllRules(t *testing.T) {
	engine := New(nil)
	rules := engine.GetAllRules()

	// Verify we have all 36 rules registered
	// 1. BlankLineBetweenBlocksRule
	// 2-6. ResourceNameConventionRule, DataNameConventionRule, VariableNameConventionRule, OutputNameConventionRule, LocalNameConventionRule
	// 7-9. TerraformBlockFirstRule, ProviderBlockOrderRule, TerragruntIncludeFirstRule
	// 10-14. ForEachCountFirstRule, SourceVersionGroupedRule, TagsAtEndRule, DependsOnOrderRule, LifecycleAtEndRule
	// 15-16. VariableOrderRule, OutputOrderRule
	// 17. AttributeGroupSpacingRule
	// 18-19. NoLeadingTrailingBlankLinesRule, NoEmptyBlocksRule
	// 20-24. VariablesInFileRule, OutputsInFileRule, ProvidersInFileRule, ScopedFileOrganizationRule, TerraformFilesStructureRule
	// 25-28. ResourceNameMatchesTypeRule, OutputPrefixRule, ModuleNameConventionRule, ModuleNameDescriptiveRule
	// 29-32. MetaArgumentsOrderRule, LifecycleAttributeOrderRule, NestedBlockOrderRule, OneLineAttributeSpacingRule
	// 33-36. CommentSyntaxRule, NoTrailingWhitespaceRule, ConsistentQuotesRule, NoConsecutiveBlankLinesRule
	assert.Len(t, rules, 36, "should have 36 rules registered")

	// Verify each rule has required methods
	for _, rule := range rules {
		assert.NotEmpty(t, rule.Name(), "rule name should not be empty")
		assert.NotEmpty(t, rule.Description(), "rule description should not be empty")
	}
}

func TestEngine_RuleConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Test with rule disabled
	engine := New(&Config{
		Rules: map[string]RuleConfig{
			"style.blank-line-between-blocks": {
				Enabled: config.BoolPtr(false),
			},
		},
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Should not find blank-line-between-blocks since it's disabled
	for _, f := range findings {
		assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := engine.Run(ctx, []string{tmpFile})
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestEngine_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile1 := filepath.Join(tmpDir, "main.tf")
	tmpFile2 := filepath.Join(tmpDir, "variables.tf")

	content1 := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	content2 := `variable "instance_type" {
  description = "The instance type"
  type        = string
}
`
	require.NoError(t, os.WriteFile(tmpFile1, []byte(content1), 0o644))
	require.NoError(t, os.WriteFile(tmpFile2, []byte(content2), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{tmpFile1, tmpFile2})
	require.NoError(t, err)

	// Both files processed without error; findings may be empty for valid HCL
	_ = findings
}

func TestEngine_Name(t *testing.T) {
	engine := New(nil)
	assert.Equal(t, "style", engine.Name())
}

func TestEngine_FixMode(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")

	// Create file with spacing issue
	content := `resource "aws_instance" "example1" {
  ami = "ami-12345"
}
resource "aws_instance" "example2" {
  ami = "ami-67890"
}`

	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Run in fix mode
	engine := New(&Config{
		Fix:   true,
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Fix mode ran without error; findings may be empty if nothing to fix
	_ = findings
}

func TestEngine_DisableSpecificRule(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")

	// Create file with properly formatted content
	content := `resource "aws_instance" "example1" {
  ami = "ami-12345"
}

resource "aws_instance" "example2" {
  ami = "ami-67890"
}`

	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Engine with rules configured
	engine := New(&Config{
		Rules: make(map[string]RuleConfig),
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// With properly formatted file, should have no blank-line findings
	for _, f := range findings {
		assert.NotEqual(t, "style.blank-line-between-blocks", f.Rule)
	}
}

func TestEngine_InvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.tf")

	// Create file with invalid HCL
	content := `resource "aws_instance" { this is invalid`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{tmpFile})

	// Invalid HCL should not panic; it may return an error or empty findings
	if err == nil {
		assert.NotNil(t, findings)
	}
}

// TestStyleEngine_CSTPipeline_ForEachSourceTagsDependsOn pins the full
// canonical ordering of a module body once every ordering rule has run
// through the engine pipeline. The single-rule equivalent lived in
// rules/ordering_test.go (TestForEachCountFirstRule "Fix preserves leading
// comments on attributes") back when ForEachCountFirstRule.Fix reordered the
// whole body via the shared hclwrite helper. After the CST migration each
// rule's Fix is narrow (for_each-and-count moves stay in one rule, source/
// version moves in another, tags/depends_on in their own), so the
// cross-rule ordering invariant has to be asserted at the pipeline boundary
// where all four rules participate.
func TestStyleEngine_CSTPipeline_ForEachSourceTagsDependsOn(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")

	input := `module "example" {
  for_each = var.instances
  depends_on = [module.other]
  # This is an important comment about the instance
  # It spans multiple lines
  instance_type = "t3.micro"
  source = "./module"
  tags = { Name = "test" }
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

	engine := New(&Config{Fix: true})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	fixed, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	out := string(fixed)

	// Leading comments survive every Move.
	assert.Contains(t, out, "# This is an important comment about the instance")
	assert.Contains(t, out, "# It spans multiple lines")

	// Canonical order after the pipeline runs:
	//   for_each, source, instance_type, tags, depends_on
	// for_each is moved (or already) at front; source follows the meta-arg;
	// instance_type's two-line leading comment travels with it; tags ends
	// near the tail before depends_on lands last (no lifecycle present).
	want := []string{
		"for_each",
		"source",
		"# This is an important comment about the instance",
		"# It spans multiple lines",
		"instance_type",
		"tags",
		"depends_on",
	}
	prev := 0
	for i, needle := range want {
		idx := strings.Index(out[prev:], needle)
		require.NotEqual(t, -1, idx,
			"expected substring %d (%q) to appear after %q in:\n%s",
			i, needle, want[max(0, i-1)], out)
		prev += idx + len(needle)
	}

	// Pipeline-level idempotency: a second engine pass on the fixed file
	// must produce no further changes. Individual rules pin idempotency in
	// their own test suites, but rule composition could in principle drift
	// (e.g., rule A inserts spacing that rule B then reflags) — this
	// catches that class of regression at the boundary where it would
	// actually appear.
	engine2 := New(&Config{Fix: true})
	_, err = engine2.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	second, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, fixed, second, "pipeline must be idempotent across passes")
}

// TestStyleEngine_CSTPipeline_Resource_ForEachTagsDependsOn is the resource-
// block sibling of the module pipeline test. SourceVersionGroupedRule only
// fires on module blocks, so this fixture exercises a different rule
// combination (for-each-count-first + tags-at-end + depends-on-order) and
// validates that SourceVersionGroupedRule's no-op path leaves resource
// content untouched.
func TestStyleEngine_CSTPipeline_Resource_ForEachTagsDependsOn(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")

	input := `resource "aws_instance" "example" {
  ami        = "ami-123"
  for_each   = var.instances
  tags       = { Name = "test" }
  depends_on = [aws_iam_role.x]
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

	engine := New(&Config{Fix: true})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	fixed, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	out := string(fixed)

	want := []string{"for_each", "ami", "tags", "depends_on"}
	prev := 0
	for i, needle := range want {
		idx := strings.Index(out[prev:], needle)
		require.NotEqual(t, -1, idx,
			"expected substring %d (%q) to appear after %q in:\n%s",
			i, needle, want[max(0, i-1)], out)
		prev += idx + len(needle)
	}
}

// TestStyleEngine_CSTPipeline_ForEachCountFirst_TagsAtEnd_AttributeGroupSpacing
// pins the engine-pipeline contract for the three rules that AttributeGroupSpacing
// composes with most tightly. After the CST migration each rule's Fix is narrow,
// so the canonical post-pipeline shape — for_each first, tags below the main
// attribute group, lifecycle last, and a blank line separating each group —
// is only observable when all three rules run through the engine.
//
// Idempotency on a second pass is asserted explicitly: a stable-post-fix file
// must produce no further edits, guarding against rule composition that would
// reflag a finding the previous pass just resolved.
func TestStyleEngine_CSTPipeline_ForEachCountFirst_TagsAtEnd_AttributeGroupSpacing(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")

	input := `resource "aws_instance" "example" {
  ami           = "ami-123"
  for_each      = var.instances
  tags          = { Name = "test" }
  instance_type = "t2.micro"
  lifecycle {
    create_before_destroy = true
  }
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(input), 0o644))

	engine := New(&Config{Fix: true})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	fixed, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	out := string(fixed)

	want := []string{"for_each", "ami", "instance_type", "tags", "lifecycle"}
	prev := 0
	for i, needle := range want {
		idx := strings.Index(out[prev:], needle)
		require.NotEqual(t, -1, idx,
			"expected substring %d (%q) to appear after %q in:\n%s",
			i, needle, want[max(0, i-1)], out)
		prev += idx + len(needle)
	}

	// AttributeGroupSpacing must have inserted blank lines at each attribute
	// group boundary: between the for_each meta-arg and the main attribute
	// group, and between the main attribute group and tags. The rule only
	// polices attribute-to-attribute pairs, so the tags-to-lifecycle gap is
	// not its responsibility (the lifecycle nested block carries its own
	// visual weight).
	assert.Contains(t, out, "for_each      = var.instances\n\n",
		"missing blank line between for_each (meta-arg group) and the main attribute group:\n%s", out)
	assert.Contains(t, out, "instance_type = \"t2.micro\"\n\n  tags",
		"missing blank line between the main attribute group and tags:\n%s", out)

	engine2 := New(&Config{Fix: true})
	_, err = engine2.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)
	second, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, fixed, second, "pipeline must be idempotent across passes")
}

func TestEngine_ApplyFixes_MultipleLocations(t *testing.T) {
	// Test that the same rule at different locations both get fixed (BUG-9)
	// Previously, applyFixes keyed by rule name only, so only the first fix was applied
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "multi_fix.tf")

	// Content with multiple missing blank lines (same rule, different locations)
	content := `resource "aws_instance" "one" {
  ami = "ami-1"
}
resource "aws_instance" "two" {
  ami = "ami-2"
}
resource "aws_instance" "three" {
  ami = "ami-3"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Enable fix mode and the blank-line rule
	engine := New(&Config{
		Fix: true,
		Rules: map[string]RuleConfig{
			"style.blank-line-between-blocks": {Enabled: config.BoolPtr(true)},
		},
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Read the fixed content
	fixedContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	// Count blank lines between blocks - should have blank lines between all blocks
	lines := string(fixedContent)

	// The fix should have added blank lines between blocks
	// With the bug, only the first missing blank line would be fixed
	// After fix, there should be proper spacing between all three resources
	assert.Contains(t, lines, "}\n\nresource", "should have blank line between blocks")

	// Should report findings for the issues that were fixed
	_ = findings // Findings count depends on whether fix was applied before or after check
}

func TestConfigFromEngine(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		engineCfg := config.StyleEngineConfig{}
		cfg := ConfigFromEngine(engineCfg)

		require.NotNil(t, cfg)
		assert.False(t, cfg.Fix)
		assert.False(t, cfg.Diff)
		assert.Empty(t, cfg.Rules)
	})

	t.Run("fix and diff enabled", func(t *testing.T) {
		engineCfg := config.StyleEngineConfig{
			Fix:  true,
			Diff: true,
		}
		cfg := ConfigFromEngine(engineCfg)

		assert.True(t, cfg.Fix)
		assert.True(t, cfg.Diff)
	})

	t.Run("with rules", func(t *testing.T) {
		engineCfg := config.StyleEngineConfig{
			Rules: map[string]config.RuleConfig{
				"blank-line-between-blocks": {
					Enabled:  config.BoolPtr(true),
					Severity: "warning",
				},
				"resource-name-convention": {
					Enabled:  config.BoolPtr(false),
					Severity: "error",
					Config:   map[string]any{"case": "snake"},
				},
			},
		}
		cfg := ConfigFromEngine(engineCfg)

		require.Len(t, cfg.Rules, 2)

		blankLineRule := cfg.Rules["blank-line-between-blocks"]
		assert.True(t, *blankLineRule.Enabled)
		assert.Equal(t, "warning", blankLineRule.Severity)
		assert.Nil(t, blankLineRule.Options)

		caseRule := cfg.Rules["resource-name-convention"]
		assert.False(t, *caseRule.Enabled)
		assert.Equal(t, "error", caseRule.Severity)
		assert.Equal(t, "snake", caseRule.Options["case"])
	})

	t.Run("nil rules map", func(t *testing.T) {
		engineCfg := config.StyleEngineConfig{
			Fix:   true,
			Rules: nil,
		}
		cfg := ConfigFromEngine(engineCfg)

		require.NotNil(t, cfg.Rules)
		assert.Empty(t, cfg.Rules)
	})
}

// fakePluginRule is a simple rule implementation for testing plugin integration
type fakePluginRule struct {
	name string
}

func (r *fakePluginRule) Name() string        { return r.name }
func (r *fakePluginRule) Description() string { return "Fake plugin rule for testing" }
func (r *fakePluginRule) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return nil, nil
}

func TestNew_AcceptsPluginRules(t *testing.T) {
	t.Run("no plugin rules", func(t *testing.T) {
		engine := New(nil)
		rules := engine.GetAllRules()

		// Should have 36 built-in rules
		assert.Len(t, rules, 36)
	})

	t.Run("with single plugin rule", func(t *testing.T) {
		pluginRule := &fakePluginRule{name: "plugin.test-rule"}
		engine := New(nil, pluginRule)
		rules := engine.GetAllRules()

		// Should have 36 built-in + 1 plugin = 37 rules
		assert.Len(t, rules, 37)

		// Plugin rule should be present
		found := false
		for _, r := range rules {
			if r.Name() == "plugin.test-rule" {
				found = true
				break
			}
		}
		assert.True(t, found, "plugin rule should be registered")
	})

	t.Run("with multiple plugin rules", func(t *testing.T) {
		plugin1 := &fakePluginRule{name: "plugin.rule-one"}
		plugin2 := &fakePluginRule{name: "plugin.rule-two"}
		plugin3 := &fakePluginRule{name: "plugin.rule-three"}
		engine := New(nil, plugin1, plugin2, plugin3)
		rules := engine.GetAllRules()

		// Should have 36 built-in + 3 plugin = 39 rules
		assert.Len(t, rules, 39)
	})
}

func TestNew_PluginRulesAppendedAfterBuiltIn(t *testing.T) {
	pluginRule := &fakePluginRule{name: "plugin.test-rule"}
	engine := New(nil, pluginRule)
	rules := engine.GetAllRules()

	// Plugin rules should be at the end of the slice
	// Get the last rule
	lastRule := rules[len(rules)-1]
	assert.Equal(t, "plugin.test-rule", lastRule.Name(), "plugin rule should be appended after built-in rules")

	// First rule should be a built-in rule (blank-line-between-blocks is first)
	firstRule := rules[0]
	assert.Equal(t, "style.blank-line-between-blocks", firstRule.Name(), "first rule should be built-in")
}

func TestEngine_SuppressionAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedCount   int
		expectedRules   []string
		unexpectedRules []string
	}{
		{
			name: "file-level suppression ignores all matching findings",
			content: `# terratidy:ignore-file:style.resource-name-convention
resource "aws_instance" "MyServer" { }
resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount:   0,
			unexpectedRules: []string{"style.resource-name-convention"},
		},
		{
			name: "next-block suppression ignores only next block",
			content: `# terratidy:ignore:style.resource-name-convention
resource "aws_instance" "MyServer" { }

resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount: 1, // One violation not suppressed
			expectedRules: []string{"style.resource-name-convention"},
		},
		{
			name: "inline suppression ignores same line",
			content: `resource "aws_instance" "MyServer" { } # terratidy:ignore:style.resource-name-convention

resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount: 1, // One violation not suppressed
			expectedRules: []string{"style.resource-name-convention"},
		},
		{
			name: "wildcard suppression matches all style rules",
			content: `# terratidy:ignore-file:style.*
resource "aws_instance" "MyServer" { }
`,
			expectedCount:   0,
			unexpectedRules: []string{"style.resource-name-convention"},
		},
		{
			name: "no suppression returns all findings",
			content: `resource "aws_instance" "MyServer" { }
resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount: 2, // Two violations
			expectedRules: []string{"style.resource-name-convention"},
		},
		{
			name: "suppression with non-existent rule is ignored",
			content: `# terratidy:ignore:style.nonexistent-rule
resource "aws_instance" "MyServer" { }
`,
			expectedCount: 1, // Violation still reported
			expectedRules: []string{"style.resource-name-convention"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Create engine with resource-name-convention rule enabled
			engine := New(&Config{
				Rules: map[string]RuleConfig{
					"style.resource-name-convention": {Enabled: config.BoolPtr(true)},
				},
			})

			// Run engine
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			// Count findings matching expectedRules
			var matchingCount int
			for _, f := range findings {
				for _, rule := range tt.expectedRules {
					if f.Rule == rule {
						matchingCount++
						break
					}
				}
			}

			// Check expected count
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedCount, matchingCount,
					"expected %d findings but got %d", tt.expectedCount, matchingCount)
			}

			// Check that unexpected rules are not present
			for _, unexpected := range tt.unexpectedRules {
				for _, f := range findings {
					assert.NotEqual(t, unexpected, f.Rule,
						"unexpected finding for rule %s", unexpected)
				}
			}
		})
	}
}

func TestEngine_FixPreservesComments(t *testing.T) {
	t.Run("single block with comments", func(t *testing.T) {
		content := `module "example" {
  source = "./module"
  depends_on = [module.other]
  # This is an important comment
  # that spans multiple lines
  instance_type = "t3.micro"
  for_each = var.instances
  tags = { Name = "test" }
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		engine := New(&Config{Fix: true})
		_, err := engine.Run(context.Background(), []string{tmpFile})
		require.NoError(t, err)

		result, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		resultStr := string(result)
		// Comments must be preserved
		assert.Contains(t, resultStr, "# This is an important comment")
		assert.Contains(t, resultStr, "# that spans multiple lines")
		// Order must be correct: for_each, source, regular attrs, tags, depends_on
		forEachIdx := strings.Index(resultStr, "for_each")
		sourceIdx := strings.Index(resultStr, "source")
		instanceIdx := strings.Index(resultStr, "instance_type")
		tagsIdx := strings.Index(resultStr, "tags")
		dependsOnIdx := strings.Index(resultStr, "depends_on")
		assert.Less(t, forEachIdx, sourceIdx, "for_each should be before source")
		assert.Less(t, sourceIdx, instanceIdx, "source should be before instance_type")
		assert.Less(t, instanceIdx, tagsIdx, "instance_type should be before tags")
		assert.Less(t, tagsIdx, dependsOnIdx, "tags should be before depends_on")
	})

	t.Run("multiple blocks with comments in single pass", func(t *testing.T) {
		content := `module "first" {
  source = "./first"
  depends_on = [module.zero]
  # Comment on first module
  attr1 = "value1"
  for_each = var.first
  tags = { Name = "first" }
}

module "second" {
  source = "./second"
  depends_on = [module.first]
  # Comment on second module
  attr2 = "value2"
  for_each = var.second
  tags = { Name = "second" }
}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		engine := New(&Config{Fix: true})
		_, err := engine.Run(context.Background(), []string{tmpFile})
		require.NoError(t, err)

		result, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		resultStr := string(result)
		// Both comments must be preserved
		assert.Contains(t, resultStr, "# Comment on first module")
		assert.Contains(t, resultStr, "# Comment on second module")
		// Both blocks must be correctly ordered (check depends_on is last in each)
		// Find the two depends_on occurrences
		firstDepends := strings.Index(resultStr, "depends_on = [module.zero]")
		secondDepends := strings.Index(resultStr, "depends_on = [module.first]")
		assert.Greater(t, firstDepends, 0, "first depends_on should exist")
		assert.Greater(t, secondDepends, firstDepends, "second depends_on should be after first")

		// Verify no issues remain after fix
		engine2 := New(&Config{Fix: false})
		findings, err := engine2.Run(context.Background(), []string{tmpFile})
		require.NoError(t, err)
		assert.Empty(t, findings, "should have no issues after fix")
	})
}

// TestEngine_PreservesFileMode_OnFix verifies that the per-pass fix-apply path
// preserves the file's original permission mode after writing fixed content.
// Skipped on Windows where Unix-style permission bits don't apply.
func TestEngine_PreservesFileMode_OnFix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style file permissions don't apply on Windows")
	}

	// Two adjacent blocks lacking a blank-line separator triggers a fixable
	// style finding. We need a fix to actually run so the WriteFile path is
	// exercised.
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o755))
	require.NoError(t, os.Chmod(tmpFile, 0o755), "ensure mode is set even if umask altered WriteFile's perm")

	engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	modified, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	require.NotEqual(t, content, string(modified), "test precondition: fix must have actually modified the file")

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(),
		"style fix must preserve original file mode after writing fixed content")
}

// TestEngine_PreservesFileMode_OnDiffPreviewRestore verifies that diff preview
// mode (Diff=true, Fix=false) restores the file to its original content AND
// preserves the original permission mode after applying-then-restoring.
// Skipped on Windows.
func TestEngine_PreservesFileMode_OnDiffPreviewRestore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style file permissions don't apply on Windows")
	}

	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o755))
	require.NoError(t, os.Chmod(tmpFile, 0o755), "ensure mode is set even if umask altered WriteFile's perm")

	engine := New(&Config{Fix: false, Diff: true, Rules: make(map[string]RuleConfig)})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Preview contract: file content unchanged after run.
	restored, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(restored), "diff preview must leave file content unchanged")

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(),
		"diff preview restore must preserve original file mode")
}

// TestEngine_DiffCaptureReadError verifies that when Diff=true and the target
// file is unreadable, the engine surfaces the "reading file for diff" error
// branch from checkFile. Setup: a chmod-0 fixture makes os.ReadFile fail with
// EACCES on the originalContent capture, before the per-pass loop ever runs.
// Skipped on Windows and when running as root, where mode-based access control
// does not apply.
func TestEngine_DiffCaptureReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style file permissions don't apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses mode-based access control")
	}

	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(`resource "aws_instance" "test" {}`+"\n"), 0o644))
	require.NoError(t, os.Chmod(tmpFile, 0o000), "make file unreadable so ReadFile fails")
	// Restore permissions so t.TempDir cleanup can remove the file.
	t.Cleanup(func() { _ = os.Chmod(tmpFile, 0o600) })

	engine := New(&Config{Diff: true, Rules: make(map[string]RuleConfig)})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.Error(t, err, "expected error from unreadable file in Diff mode")
	assert.Contains(t, err.Error(), "reading file for diff",
		"error should be wrapped with the diff-capture context from checkFile")
	// ErrorIs traverses %w wrapping; this catches regressions that drop the
	// wrap and replace the cause with a fmt.Errorf("%v", ...) string.
	assert.ErrorIs(t, err, fs.ErrPermission, "underlying cause should remain reachable via errors.Is")
}

// TestEngine_WriteFileError verifies that a write failure during fix-apply
// surfaces as an error from Engine.Run. Setup: a read-only fixture (0o400)
// lets ReadFile succeed (owner has read) and the rules detect a fixable
// finding, but os.WriteFile in applyFixes fails with EACCES, exercising the
// "writing fix" error branch. Skipped on Windows and when running as root,
// where mode-based access control does not apply.
func TestEngine_WriteFileError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style file permissions don't apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses mode-based access control")
	}

	// Two adjacent blocks → fixable style finding → applyFixes runs.
	content := `resource "aws_instance" "test1" {
  ami = "ami-123"
}
resource "aws_instance" "test2" {
  ami = "ami-456"
}
`
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))
	require.NoError(t, os.Chmod(tmpFile, 0o400), "make file read-only so WriteFile fails")

	engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)})
	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.Error(t, err, "expected error from read-only file write")
	assert.Contains(t, err.Error(), "writing fixes for",
		"error should be wrapped with the rule-list context from applyFixes")
	assert.Contains(t, err.Error(), "style.blank-line-between-blocks",
		"error should name the rule(s) whose edits were in flight")
}

// TestCollectStuckFixableRules exercises the helper that powers the
// stuck-rule branch of the fix-loop guard in checkFile. The function must
// (a) skip findings whose Fixable flag is false, (b) deduplicate rule names
// that appear more than once, (c) preserve source order on first occurrence,
// and (d) return nil when no Fixable finding is present.
func TestCollectStuckFixableRules(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		want     []string
	}{
		{
			name:     "empty input returns nil",
			findings: nil,
			want:     nil,
		},
		{
			name: "no fixable findings returns nil",
			findings: []sdk.Finding{
				{Rule: "rule-a", Fixable: false},
				{Rule: "rule-b", Fixable: false},
			},
			want: nil,
		},
		{
			name: "non-fixable findings are skipped",
			findings: []sdk.Finding{
				{Rule: "rule-a", Fixable: false},
				{Rule: "rule-b", Fixable: true},
				{Rule: "rule-c", Fixable: false},
			},
			want: []string{"rule-b"},
		},
		{
			name: "duplicate fixable rule names are deduplicated",
			findings: []sdk.Finding{
				{Rule: "rule-a", Fixable: true},
				{Rule: "rule-a", Fixable: true},
				{Rule: "rule-b", Fixable: true},
				{Rule: "rule-a", Fixable: true},
			},
			want: []string{"rule-a", "rule-b"},
		},
		{
			name: "source order is preserved on first occurrence",
			findings: []sdk.Finding{
				{Rule: "rule-z", Fixable: true},
				{Rule: "rule-a", Fixable: true},
				{Rule: "rule-m", Fixable: true},
			},
			want: []string{"rule-z", "rule-a", "rule-m"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectStuckFixableRules(tt.findings)
			assert.Equal(t, tt.want, got)
		})
	}
}

// stubNarrowEditRule is a parameterized Fixer that emits a single narrow
// TextEdit at a configured byte range. Tests use two instances at disjoint
// ranges to exercise applyFixes's multi-edit splice path without relying on
// real rules whose tests live in the (currently mid-migration) rules
// subpackage.
type stubNarrowEditRule struct {
	name        string
	startOffset int
	endOffset   int
	replacement []byte
}

func (r *stubNarrowEditRule) Name() string        { return r.name }
func (r *stubNarrowEditRule) Description() string { return "Test stub: emits one narrow edit" }

func (r *stubNarrowEditRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return []sdk.Finding{{
		Rule:     r.name,
		Message:  "stub narrow-edit finding",
		File:     ctx.File,
		Severity: sdk.SeverityWarning,
		Fixable:  true,
	}}, nil
}

func (r *stubNarrowEditRule) Fix(_ *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	return &sdk.FixResult{
		Edits: []sdk.TextEdit{{
			Start:       r.startOffset,
			End:         r.endOffset,
			Replacement: r.replacement,
		}},
	}, nil
}

// TestApplyFixes_MultipleNonOverlappingEdits asserts that applyFixes splices
// two narrow edits from two distinct rules in a single pass — exactly one
// os.WriteFile call, both rule names returned in appliedRules, and the
// resulting content matches the descending-Start splice of both edits.
//
// The single-write assertion is the load-bearing guarantee: before narrow
// byte-range edits landed, N independent fixes required N passes (and N
// writes). The new contract collapses them into one write per pass when
// ranges don't conflict.
//
// Two sub-tests exercise the splice math: equal-length replacements (trivial
// offset preservation) and asymmetric-length replacements (the right-edit
// shifts later content, the left-edit must still land at its original offset
// because descending-Start splices the right edge first).
func TestApplyFixes_MultipleNonOverlappingEdits(t *testing.T) {
	type editSpec struct {
		marker      string
		replacement []byte
	}

	tests := []struct {
		name     string
		original []byte
		left     editSpec
		right    editSpec
		expected []byte
	}{
		{
			name:     "equal_length_replacements",
			original: []byte("# AAAA BBBB CCCC\n"),
			left:     editSpec{marker: "AAAA", replacement: []byte("aaaa")},
			right:    editSpec{marker: "CCCC", replacement: []byte("cccc")},
			expected: []byte("# aaaa BBBB cccc\n"),
		},
		{
			// Asymmetric-length replacements stress the descending-Start
			// splice: the right edit ("CCCC" → 8 bytes) expands the tail,
			// then the left edit ("AAAA" → 2 bytes) shrinks the head. The
			// invariant being checked is that the left edit applies at its
			// ORIGINAL offset, not a post-right-splice offset.
			name:     "asymmetric_length_replacements",
			original: []byte("# AAAA BBBB CCCC\n"),
			left:     editSpec{marker: "AAAA", replacement: []byte("xx")},
			right:    editSpec{marker: "CCCC", replacement: []byte("yyyyyyyy")},
			expected: []byte("# xx BBBB yyyyyyyy\n"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tmpFile := filepath.Join(dir, "narrow.tf")
			require.NoError(t, os.WriteFile(tmpFile, tc.original, 0o644))

			leftStart := bytes.Index(tc.original, []byte(tc.left.marker))
			require.GreaterOrEqual(t, leftStart, 0, "test fixture must contain left marker")
			rightStart := bytes.Index(tc.original, []byte(tc.right.marker))
			require.GreaterOrEqual(t, rightStart, 0, "test fixture must contain right marker")

			ruleLeft := &stubNarrowEditRule{
				name:        "test.stub-narrow-left",
				startOffset: leftStart,
				endOffset:   leftStart + len(tc.left.marker),
				replacement: tc.left.replacement,
			}
			ruleRight := &stubNarrowEditRule{
				name:        "test.stub-narrow-right",
				startOffset: rightStart,
				endOffset:   rightStart + len(tc.right.marker),
				replacement: tc.right.replacement,
			}

			engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)}, ruleLeft, ruleRight)

			var writeCount int
			var capturedContent []byte
			engine.writeFn = func(name string, data []byte, perm os.FileMode) error {
				writeCount++
				capturedContent = append([]byte(nil), data...)
				return os.WriteFile(name, data, perm)
			}

			parser := hclparse.NewParser()
			file, diags := parser.ParseHCL(tc.original, tmpFile)
			require.False(t, diags.HasErrors(), "test fixture must parse: %s", diags.Error())

			ruleCtx := &sdk.Context{
				Context: context.Background(),
				Options: make(map[string]any),
				WorkDir: ".",
				File:    tmpFile,
			}
			findings := []sdk.Finding{
				{Rule: ruleLeft.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning},
				{Rule: ruleRight.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning},
			}

			applied, err := engine.applyFixes(ruleCtx, file, findings, 0o644)
			require.NoError(t, err)

			// Load-bearing assertion: one pass means one write.
			assert.Equal(t, 1, writeCount,
				"applyFixes must perform exactly one os.WriteFile call for two non-overlapping edits")
			assert.ElementsMatch(t, []string{ruleLeft.Name(), ruleRight.Name()}, applied,
				"appliedRules must contain both contributing rule names")
			assert.Equal(t, tc.expected, capturedContent,
				"single write must contain both narrow replacements spliced from one pass")

			onDisk, err := os.ReadFile(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, onDisk,
				"on-disk content must match the single write (consistency check)")
		})
	}
}

// TestApplyFixes_OverlappingEdits_DefersSecond asserts that when two edits
// conflict (same byte range, or partially-overlapping ranges), only the first
// by source order is applied this pass; the conflicting later edit is dropped
// from the retained set and re-emerges on the next pass when checkFile re-runs
// Check against the updated content. The single-pass contract is the unit
// under test here; multi-pass convergence is covered by Engine.Run tests.
//
// Two sub-cases exercise both branches of editsConflict: same-Start
// (unconditional conflict regardless of End), and the half-open range
// intersection branch (positions shared by two distinct Start offsets).
func TestApplyFixes_OverlappingEdits_DefersSecond(t *testing.T) {
	type editSpec struct {
		start, end  int
		replacement []byte
	}

	tests := []struct {
		name     string
		original []byte
		first    editSpec
		second   editSpec
		expected []byte
	}{
		{
			// Same-Start branch of editsConflict: first wins, second deferred.
			name:     "same_range",
			original: []byte("# AAAA BBBB\n"),
			first:    editSpec{start: 2, end: 6, replacement: []byte("xxxx")},
			second:   editSpec{start: 2, end: 6, replacement: []byte("yyyy")},
			expected: []byte("# xxxx BBBB\n"),
		},
		{
			// Half-open range intersection branch: original is "# abcdef\n"
			// (indices: '#'=0, ' '=1, 'a'=2, 'b'=3, 'c'=4, 'd'=5, 'e'=6, 'f'=7).
			// [3,6) covers 'bcd'; [4,7) covers 'cde'. Shared positions 4 and 5.
			// Distinct Start offsets so the same-Start short-circuit doesn't fire.
			// First retained, second deferred → only 'bcd' becomes "ZZZ".
			name:     "partial_overlap",
			original: []byte("# abcdef\n"),
			first:    editSpec{start: 3, end: 6, replacement: []byte("ZZZ")},
			second:   editSpec{start: 4, end: 7, replacement: []byte("WWW")},
			expected: []byte("# aZZZef\n"),
		},
		{
			// Zero-width-inside-range branch of editsConflict: a zero-width
			// insertion's offset lies strictly inside the other edit's
			// half-open range. The half-open intersection check at
			// max(Start) < min(End) misses this (min(End) equals the
			// insertion's offset, so 3 < 3 is false); the dedicated
			// zero-width branch catches it. Without this branch
			// FuzzApplyFixesSorting found a divergence between the engine's
			// descending splice and an ascending-with-shift reference.
			//
			// Original "# abcde\n" with first = replace [2,5)="abc" → "X"
			// and second = insert "Y" at offset 3 (strictly inside [2,5)).
			// First retained, second deferred → "# Xde\n"; the insertion
			// re-anchors next pass against the rewritten content.
			name:     "zero_width_inside_range",
			original: []byte("# abcde\n"),
			first:    editSpec{start: 2, end: 5, replacement: []byte("X")},
			second:   editSpec{start: 3, end: 3, replacement: []byte("Y")},
			expected: []byte("# Xde\n"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tmpFile := filepath.Join(dir, "overlap.tf")
			require.NoError(t, os.WriteFile(tmpFile, tc.original, 0o644))

			ruleFirst := &stubNarrowEditRule{
				name:        "test.stub-overlap-first",
				startOffset: tc.first.start,
				endOffset:   tc.first.end,
				replacement: tc.first.replacement,
			}
			ruleSecond := &stubNarrowEditRule{
				name:        "test.stub-overlap-second",
				startOffset: tc.second.start,
				endOffset:   tc.second.end,
				replacement: tc.second.replacement,
			}

			engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)}, ruleFirst, ruleSecond)

			var writeCount int
			var capturedContent []byte
			engine.writeFn = func(name string, data []byte, perm os.FileMode) error {
				writeCount++
				capturedContent = append([]byte(nil), data...)
				return os.WriteFile(name, data, perm)
			}

			parser := hclparse.NewParser()
			file, diags := parser.ParseHCL(tc.original, tmpFile)
			require.False(t, diags.HasErrors(), "test fixture must parse: %s", diags.Error())

			ruleCtx := &sdk.Context{
				Context: context.Background(),
				Options: make(map[string]any),
				WorkDir: ".",
				File:    tmpFile,
			}
			// Order is load-bearing: ruleFirst's finding precedes ruleSecond's,
			// so the conflict-resolution loop retains ruleFirst's edit and
			// drops ruleSecond's.
			findings := []sdk.Finding{
				{Rule: ruleFirst.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning},
				{Rule: ruleSecond.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning},
			}

			applied, err := engine.applyFixes(ruleCtx, file, findings, 0o644)
			require.NoError(t, err)

			assert.Equal(t, 1, writeCount,
				"applyFixes must perform exactly one os.WriteFile call even when edits conflict")
			assert.Equal(t, []string{ruleFirst.Name()}, applied,
				"appliedRules must contain only the first rule by source order; the deferred edit does not appear in this pass's report")
			assert.Equal(t, tc.expected, capturedContent,
				"single write must contain only the first rule's replacement; the second's edit is deferred to a later pass")

			onDisk, err := os.ReadFile(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, onDisk,
				"on-disk content must match the single write (consistency check)")
		})
	}
}

// TestApplyFixes_WholeFileEdit_ExclusiveOfOthers asserts that when one rule
// emits a whole-file edit (Start=0, End=len(content)) and another rule emits
// a narrow edit in the same pass, applyFixes applies the whole-file edit
// alone and defers the narrow edit. The single-pass contract under test:
// appliedRules contains only the whole-file rule, exactly one write occurs,
// and the written content matches the whole-file replacement verbatim.
//
// Exclusivity must be source-order independent: the whole-file branch at
// style.go scans every collected edit looking for one that covers the full
// file, so registering the narrow rule first must not let the narrow edit
// sneak through ahead of the whole-file replacement. Two sub-cases cover
// both orderings to pin this invariant.
//
// Multi-pass convergence (next pass picks up the deferred narrow edit
// re-emitted against the rewritten content) is covered by Engine.Run tests;
// this single-pass unit test exercises the exclusivity branch directly.
func TestApplyFixes_WholeFileEdit_ExclusiveOfOthers(t *testing.T) {
	original := []byte("# AAAA BBBB\n")
	wholeFileReplacement := []byte("ENTIRELY NEW\n")
	narrowMarker := "AAAA"
	narrowReplacement := []byte("aaaa")

	tests := []struct {
		name string
		// orderWholeFileFirst controls the source order of the findings slice.
		// True: whole-file finding precedes narrow finding, so the whole-file
		// edit is collected[0] and the exclusivity loop finds it on its first
		// iteration.
		// False: narrow finding precedes whole-file finding, so the loop must
		// scan past collected[0] before finding the whole-file edit at
		// collected[1]. Both orderings must yield the same result — the
		// invariant under test is source-order independence.
		orderWholeFileFirst bool
	}{
		{name: "whole_file_registered_first", orderWholeFileFirst: true},
		{name: "narrow_registered_first", orderWholeFileFirst: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tmpFile := filepath.Join(dir, "whole.tf")
			require.NoError(t, os.WriteFile(tmpFile, original, 0o644))

			narrowStart := bytes.Index(original, []byte(narrowMarker))
			require.GreaterOrEqual(t, narrowStart, 0, "test fixture must contain narrow marker")

			wholeFileRule := &stubNarrowEditRule{
				name:        "test.stub-whole-file",
				startOffset: 0,
				endOffset:   len(original),
				replacement: wholeFileReplacement,
			}
			narrowRule := &stubNarrowEditRule{
				name:        "test.stub-narrow",
				startOffset: narrowStart,
				endOffset:   narrowStart + len(narrowMarker),
				replacement: narrowReplacement,
			}

			engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)}, wholeFileRule, narrowRule)

			var writeCount int
			var capturedContent []byte
			engine.writeFn = func(name string, data []byte, perm os.FileMode) error {
				writeCount++
				capturedContent = append([]byte(nil), data...)
				return os.WriteFile(name, data, perm)
			}

			parser := hclparse.NewParser()
			file, diags := parser.ParseHCL(original, tmpFile)
			require.False(t, diags.HasErrors(), "test fixture must parse: %s", diags.Error())

			ruleCtx := &sdk.Context{
				Context: context.Background(),
				Options: make(map[string]any),
				WorkDir: ".",
				File:    tmpFile,
			}

			wholeFinding := sdk.Finding{Rule: wholeFileRule.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning}
			narrowFinding := sdk.Finding{Rule: narrowRule.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning}

			var findings []sdk.Finding
			if tc.orderWholeFileFirst {
				findings = []sdk.Finding{wholeFinding, narrowFinding}
			} else {
				findings = []sdk.Finding{narrowFinding, wholeFinding}
			}

			applied, err := engine.applyFixes(ruleCtx, file, findings, 0o644)
			require.NoError(t, err)

			assert.Equal(t, 1, writeCount,
				"applyFixes must perform exactly one os.WriteFile call when a whole-file edit is present")
			assert.Equal(t, []string{wholeFileRule.Name()}, applied,
				"appliedRules must contain only the whole-file rule; the narrow edit is deferred")
			assert.Equal(t, wholeFileReplacement, capturedContent,
				"single write must contain the whole-file replacement only; the narrow edit's bytes must not appear")

			onDisk, err := os.ReadFile(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, wholeFileReplacement, onDisk,
				"on-disk content must match the single write (consistency check)")
		})
	}
}

// TestApplyFixes_OutOfBoundsEdit_Errors asserts that applyFixes rejects any
// edit violating the half-open [Start, End) invariants BEFORE writing,
// surfacing a rule-attributed error and leaving the file untouched.
//
// The three bounds-check branches at style.go:478-488 are:
//
//   - Start < 0
//   - End < Start
//   - End > len(content)
//
// One sub-case per branch pins both the trigger condition and the branch's
// specific error wording so a future refactor cannot silently widen the
// accepted range. Per sub-case the assertions verify (a) an error is
// returned, (b) the error contains the originating rule name, (c) the error
// matches the branch's exact wording, (d) no file write occurs (writeFn seam
// counter remains zero), (e) the file's mode is unchanged, and (f) the
// file's content is unchanged. The mode and content checks pin the
// no-side-effects invariant — applyFixes must abort before writeFixed runs,
// so the defensive Chmod inside writeFixed never fires either.
func TestApplyFixes_OutOfBoundsEdit_Errors(t *testing.T) {
	// 12-byte original: positions 0..11; len(content) == 12.
	original := []byte("# AAAA BBBB\n")

	tests := []struct {
		name     string
		ruleName string
		// start, end describe the (deliberately invalid) edit emitted by
		// the stub rule. Replacement is irrelevant — the bounds-check fires
		// before any splice attempt — so it stays constant across sub-cases.
		start, end int
		// wantErrFragment pins the specific branch's error wording. The
		// bounds-check order in applyFixes is (1) Start<0, (2) End<Start,
		// (3) End>len, so each sub-case is constructed to trip exactly one.
		wantErrFragment string
	}{
		{
			name:            "start_negative",
			ruleName:        "test.stub-negative-start",
			start:           -1, // Start<0 fires first regardless of End
			end:             4,
			wantErrFragment: "edit start -1 is negative",
		},
		{
			name:            "end_precedes_start",
			ruleName:        "test.stub-end-before-start",
			start:           6,
			end:             2, // End<Start fires before End>len would
			wantErrFragment: "edit end 2 precedes start 6",
		},
		{
			name:            "end_exceeds_content_length",
			ruleName:        "test.stub-end-past-eof",
			start:           2,
			end:             len(original) + 5, // 17 > 12
			wantErrFragment: "edit end 17 exceeds content length 12",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tmpFile := filepath.Join(dir, "oob.tf")
			require.NoError(t, os.WriteFile(tmpFile, original, 0o644))

			// Snapshot the post-WriteFile mode rather than asserting against
			// a literal 0o644 — keeps the test independent of the runner's
			// umask while still letting us detect a stray Chmod on the
			// error path.
			beforeStat, err := os.Stat(tmpFile)
			require.NoError(t, err)
			beforeMode := beforeStat.Mode().Perm()

			rule := &stubNarrowEditRule{
				name:        tc.ruleName,
				startOffset: tc.start,
				endOffset:   tc.end,
				replacement: []byte("UNUSED"),
			}

			engine := New(&Config{Fix: true, Rules: make(map[string]RuleConfig)}, rule)

			var writeCount int
			engine.writeFn = func(name string, data []byte, perm os.FileMode) error {
				writeCount++
				return os.WriteFile(name, data, perm)
			}

			parser := hclparse.NewParser()
			file, diags := parser.ParseHCL(original, tmpFile)
			require.False(t, diags.HasErrors(), "test fixture must parse: %s", diags.Error())

			ruleCtx := &sdk.Context{
				Context: context.Background(),
				Options: make(map[string]any),
				WorkDir: ".",
				File:    tmpFile,
			}
			findings := []sdk.Finding{
				{Rule: rule.Name(), File: tmpFile, Fixable: true, Severity: sdk.SeverityWarning},
			}

			// Pass a captured mode (0o600) distinct from the on-disk mode
			// (0o644 minus umask). If applyFixes ever invoked writeFixed on
			// the error path, writeFixed's defensive Chmod would force the
			// file to 0o600 — the mode-preservation assert below would
			// catch the regression.
			applied, err := engine.applyFixes(ruleCtx, file, findings, 0o600)
			require.Error(t, err, "applyFixes must reject out-of-bounds edits")
			assert.Nil(t, applied, "appliedRules must be nil on the bounds-check error path")

			assert.Contains(t, err.Error(), "fix from",
				"error must use the consistent 'fix from <rule>' prefix from applyFixes")
			assert.Contains(t, err.Error(), tc.ruleName,
				"error must name the originating rule so the user can locate the offending fixer")
			assert.Contains(t, err.Error(), tc.wantErrFragment,
				"error must identify the specific bounds-check branch")

			assert.Zero(t, writeCount,
				"writeFn must not be invoked when the bounds-check fails — the abort happens before writeFixed runs")

			afterStat, err := os.Stat(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, beforeMode, afterStat.Mode().Perm(),
				"file mode must be preserved on the bounds-check error path (no write, no Chmod)")

			onDisk, err := os.ReadFile(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, original, onDisk,
				"file content must be preserved on the bounds-check error path")
		})
	}
}

// TestEngine_MultiFinding_SinglePass closes the integration loop for the
// byte-range-textedits change. It exercises Engine.Run on a multi-rule
// fixture with the writeFn seam wired to a capturing closure, and pins
// convergence + a bounded write count.
//
// While every Fixer here returns a single whole-file edit, the whole-file
// exclusivity check in applyFixes picks one rule per pass — so writeCount
// == 3 currently. Once rules return narrow byte-range edits instead of
// whole-file edits, the same fixture converges in one pass with one write;
// the upper bound and convergence assertions remain valid then, and the
// test naturally observes writeCount == 1 without any code change.
//
// Companion: TestApplyFixes_MultipleNonOverlappingEdits pins the
// writeCount == 1 contract for narrow edits via stub rules at the
// applyFixes boundary. This test closes the integration loop with real
// rules through Engine.Run.
func TestEngine_MultiFinding_SinglePass(t *testing.T) {
	// Fixture triggers three independent opt-in fixable rules:
	//   1. style.comment-syntax        — `// header` → `# header`
	//   2. style.no-trailing-whitespace — trailing spaces after `ami = "ami-123"`
	//   3. style.no-consecutive-blank-lines — two blank lines collapse to one
	//
	// The `triggers = { foo = "bar" }` populated map keeps the fixture immune
	// to a future no-empty-blocks rule extension that might match `{}`. No
	// default-on rule fires here: blank-line-between-blocks is satisfied (a
	// blank line exists between the resources) and attribute-group-spacing
	// has nothing to group (one attribute per block).
	fixture := "// header comment\n" +
		"resource \"aws_instance\" \"test\" {\n" +
		"  ami = \"ami-123\"   \n" +
		"}\n" +
		"\n" +
		"\n" +
		"resource \"null_resource\" \"two\" {\n" +
		"  triggers = { foo = \"bar\" }\n" +
		"}\n"

	expectedCanonical := "# header comment\n" +
		"resource \"aws_instance\" \"test\" {\n" +
		"  ami = \"ami-123\"\n" +
		"}\n" +
		"\n" +
		"resource \"null_resource\" \"two\" {\n" +
		"  triggers = { foo = \"bar\" }\n" +
		"}\n"

	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(fixture), 0o644))

	// Enable the three opt-in rules; everything else falls back to the engine's
	// default config (which leaves them at their registered defaults).
	engine := New(&Config{
		Fix: true,
		Rules: map[string]RuleConfig{
			"style.comment-syntax":             {Enabled: config.BoolPtr(true)},
			"style.no-trailing-whitespace":     {Enabled: config.BoolPtr(true)},
			"style.no-consecutive-blank-lines": {Enabled: config.BoolPtr(true)},
		},
	})

	var writeCount int
	var writtenContents [][]byte
	engine.writeFn = func(name string, data []byte, perm os.FileMode) error {
		writeCount++
		writtenContents = append(writtenContents, append([]byte(nil), data...))
		return os.WriteFile(name, data, perm)
	}

	_, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Engine must have written at least once — the fixture is not canonical.
	require.Positive(t, writeCount,
		"engine must invoke writeFn at least once to fix the non-canonical fixture")

	// Three distinct fixable rules fire on the original fixture. Whole-file
	// exclusivity caps applyFixes at one rule per pass, so writeCount is
	// bounded above by 3 while rules emit whole-file edits. Narrow byte-range
	// edits tighten this to 1.
	const fixableRuleCount = 3
	assert.LessOrEqual(t, writeCount, fixableRuleCount,
		"writeCount must not exceed the count of distinct fixable rules with findings on the fixture")

	// Convergence: the on-disk file matches the canonical form, and the loop
	// terminated on a no-edit pass (so the last captured write IS the canonical
	// form — no thrashing after the converged state was reached).
	onDisk, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, expectedCanonical, string(onDisk),
		"engine.Run must converge the fixture to canonical form")

	require.NotEmpty(t, writtenContents,
		"writtenContents must have at least one entry because writeCount > 0")
	assert.Equal(t, expectedCanonical, string(writtenContents[len(writtenContents)-1]),
		"the last write captured by writeFn must equal the canonical form")

	// Per-pass single-write invariant: writeFn is the engine's only path to
	// os.WriteFile from the fix loop, and writeFixed (its sole caller inside
	// applyFixes) is invoked at most once per pass. Therefore the captured
	// writeCount equals the number of fix-applying passes — and each pass
	// contributed exactly one write. Successive writes must produce distinct
	// content (each pass advances the file state), guarding against a buggy
	// applyFixes that wrote the same bytes twice within a single pass.
	for i := 1; i < len(writtenContents); i++ {
		assert.NotEqual(t, writtenContents[i-1], writtenContents[i],
			"consecutive writes must produce distinct content — each fix-applying pass must advance the file state, not re-emit the previous state")
	}
}

// stubFixer is a minimal sdk.Fixer used to drive RegisterFixerForTesting in
// unit tests. The Fix method returns the configured result and error verbatim;
// FixCalled records that Fix ran so tests can assert delegation.
type stubFixer struct {
	result    *sdk.FixResult
	err       error
	fixCalled bool
}

func (s *stubFixer) Fix(_ *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	s.fixCalled = true
	return s.result, s.err
}

// TestEngine_RegisterFixerForTesting pins the contract of the test-only seam:
// the registered name is discoverable via Engine.Fixer, and the returned
// Fixer delegates Fix to the supplied stub. The shim's Check is a no-op (no
// findings, no error) so the registered name does not produce diagnostics on
// its own, which the second sub-test asserts directly.
func TestEngine_RegisterFixerForTesting(t *testing.T) {
	t.Run("delegates Fix to registered fixer and propagates errors", func(t *testing.T) {
		engine := New(&Config{Rules: make(map[string]RuleConfig)})

		wantErr := errors.New("simulated fix failure")
		stub := &stubFixer{result: nil, err: wantErr}
		engine.RegisterFixerForTesting("test.simulated-failure", stub)

		fixer := engine.Fixer("test.simulated-failure")
		require.NotNil(t, fixer, "Engine.Fixer must discover the registered name")

		result, err := fixer.Fix(&sdk.Context{Context: context.Background()}, nil)
		assert.Nil(t, result, "stub returns nil FixResult; the shim must propagate it without wrapping")
		assert.ErrorIs(t, err, wantErr, "shim must propagate the underlying Fixer's error verbatim")
		assert.True(t, stub.fixCalled, "delegation must invoke the registered fixer's Fix")
	})

	t.Run("Check returns no findings so the shim does not emit diagnostics", func(t *testing.T) {
		engine := New(&Config{Rules: make(map[string]RuleConfig)})
		engine.RegisterFixerForTesting("test.no-diagnostic-shim", &stubFixer{})

		// Find the registered shim in the rule slice and call Check directly.
		// The shim is the last appended rule, so iterate from the end.
		var shim sdk.Rule
		for i := len(engine.rules) - 1; i >= 0; i-- {
			if engine.rules[i].Name() == "test.no-diagnostic-shim" {
				shim = engine.rules[i]
				break
			}
		}
		require.NotNil(t, shim, "registered shim must be present in engine.rules")

		findings, err := shim.Check(&sdk.Context{Context: context.Background()}, nil)
		assert.NoError(t, err, "shim Check must not error")
		assert.Empty(t, findings, "shim Check must return no findings — diagnostics come from real rules, not the test seam")
	})
}
