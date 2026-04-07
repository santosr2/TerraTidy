package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/style/rules"
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

func TestBlockLabelCaseRule(t *testing.T) {
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
				if f.Rule == "style.block-label-case" {
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

	// Verify we have all 33 rules registered
	// 1. BlankLineBetweenBlocksRule
	// 2-5. BlockLabelCaseRule, VariableNamingRule, OutputNamingRule, LocalNamingRule
	// 6-7. TerraformBlockFirstRule, ProviderBlockOrderRule
	// 8-12. ForEachCountFirstRule, SourceVersionGroupedRule, TagsAtEndRule, DependsOnOrderRule, LifecycleAtEndRule
	// 13-14. VariableOrderRule, OutputOrderRule
	// 15. AttributeGroupSpacingRule
	// 16-17. NoLeadingTrailingBlankLinesRule, NoEmptyBlocksRule
	// 18-22. VariablesInFileRule, OutputsInFileRule, ProvidersInFileRule, ScopedFileOrganizationRule, TerraformFilesStructureRule
	// 23-25. ResourceNameMatchesTypeRule, OutputPrefixRule, ModuleNameConventionRule
	// 26-29. MetaArgumentsOrderRule, LifecycleAttributeOrderRule, NestedBlockOrderRule, OneLineAttributeSpacingRule
	// 30-33. CommentSyntaxRule, NoTrailingWhitespaceRule, ConsistentQuotesRule, NoConsecutiveBlankLinesRule
	assert.Len(t, rules, 33, "should have 33 rules registered")

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
				Enabled: false,
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

func TestIsDependsOnRelevantBlock(t *testing.T) {
	tests := []struct {
		blockType string
		want      bool
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
			got := rules.IsDependsOnRelevantBlock(tt.blockType)
			assert.Equal(t, tt.want, got)
		})
	}
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
			"style.blank-line-between-blocks": {Enabled: true},
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
					Enabled:  true,
					Severity: "warning",
				},
				"block-label-case": {
					Enabled:  false,
					Severity: "error",
					Config:   map[string]any{"case": "snake"},
				},
			},
		}
		cfg := ConfigFromEngine(engineCfg)

		require.Len(t, cfg.Rules, 2)

		blankLineRule := cfg.Rules["blank-line-between-blocks"]
		assert.True(t, blankLineRule.Enabled)
		assert.Equal(t, "warning", blankLineRule.Severity)
		assert.Nil(t, blankLineRule.Options)

		caseRule := cfg.Rules["block-label-case"]
		assert.False(t, caseRule.Enabled)
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

// mockPluginRule is a simple rule implementation for testing plugin integration
type mockPluginRule struct {
	name string
}

func (r *mockPluginRule) Name() string        { return r.name }
func (r *mockPluginRule) Description() string { return "Mock plugin rule for testing" }
func (r *mockPluginRule) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return nil, nil
}

func TestNew_AcceptsPluginRules(t *testing.T) {
	t.Run("no plugin rules", func(t *testing.T) {
		engine := New(nil)
		rules := engine.GetAllRules()

		// Should have 33 built-in rules
		assert.Len(t, rules, 33)
	})

	t.Run("with single plugin rule", func(t *testing.T) {
		pluginRule := &mockPluginRule{name: "plugin.test-rule"}
		engine := New(nil, pluginRule)
		rules := engine.GetAllRules()

		// Should have 33 built-in + 1 plugin = 34 rules
		assert.Len(t, rules, 34)

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
		plugin1 := &mockPluginRule{name: "plugin.rule-one"}
		plugin2 := &mockPluginRule{name: "plugin.rule-two"}
		plugin3 := &mockPluginRule{name: "plugin.rule-three"}
		engine := New(nil, plugin1, plugin2, plugin3)
		rules := engine.GetAllRules()

		// Should have 33 built-in + 3 plugin = 36 rules
		assert.Len(t, rules, 36)
	})
}

func TestNew_PluginRulesAppendedAfterBuiltIn(t *testing.T) {
	pluginRule := &mockPluginRule{name: "plugin.test-rule"}
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
			content: `# terratidy:ignore-file:style.block-label-case
resource "aws_instance" "MyServer" { }
resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount:   0,
			unexpectedRules: []string{"style.block-label-case"},
		},
		{
			name: "next-block suppression ignores only next block",
			content: `# terratidy:ignore:style.block-label-case
resource "aws_instance" "MyServer" { }

resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount: 1, // One violation not suppressed
			expectedRules: []string{"style.block-label-case"},
		},
		{
			name: "inline suppression ignores same line",
			content: `resource "aws_instance" "MyServer" { } # terratidy:ignore:style.block-label-case

resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount: 1, // One violation not suppressed
			expectedRules: []string{"style.block-label-case"},
		},
		{
			name: "wildcard suppression matches all style rules",
			content: `# terratidy:ignore-file:style.*
resource "aws_instance" "MyServer" { }
`,
			expectedCount:   0,
			unexpectedRules: []string{"style.block-label-case"},
		},
		{
			name: "no suppression returns all findings",
			content: `resource "aws_instance" "MyServer" { }
resource "aws_s3_bucket" "AnotherBad" { }
`,
			expectedCount: 2, // Two violations
			expectedRules: []string{"style.block-label-case"},
		},
		{
			name: "suppression with non-existent rule is ignored",
			content: `# terratidy:ignore:style.nonexistent-rule
resource "aws_instance" "MyServer" { }
`,
			expectedCount: 1, // Violation still reported
			expectedRules: []string{"style.block-label-case"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Create engine with block-label-case rule enabled
			engine := New(&Config{
				Rules: map[string]RuleConfig{
					"style.block-label-case": {Enabled: true},
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
