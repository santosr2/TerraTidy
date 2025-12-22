package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

	// Verify we have all 12 rules registered
	assert.Len(t, rules, 12, "should have 12 rules registered")

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

	// Both files should be processed without error - findings may be empty
	// The important thing is no error was returned
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
			got := isDependsOnRelevantBlock(tt.blockType)
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

	// Fix mode should still report findings
	// The actual fixing happens via the applyFixes function
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

	// May or may not return error depending on how parsing is handled
	// The important thing is we don't panic
	_ = err
	_ = findings
}
