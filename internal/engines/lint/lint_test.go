package lint

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
		config      *Config
		wantErr     bool
		wantFinding bool
	}{
		{
			name: "valid terraform file with version",
			content: `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			config:      nil,
			wantErr:     false,
			wantFinding: false,
		},
		{
			name: "missing required_version",
			content: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			config:      nil,
			wantErr:     false,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			engine := New(tt.config)
			findings, err := engine.Run(context.Background(), []string{tmpFile})

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantFinding {
				assert.NotEmpty(t, findings, "expected to find issues")
			} else {
				assert.Empty(t, findings, "expected no issues")
			}
		})
	}
}

func TestTerraformRequiredVersionRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "has required_version",
			content: `terraform {
  required_version = ">= 1.0"
}
`,
			wantFinding: false,
		},
		{
			name: "missing required_version",
			content: `terraform {
}
`,
			wantFinding: true,
		},
		{
			name: "no terraform block",
			content: `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-required-version" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformDocumentedVariablesRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "variable with description",
			content: `variable "instance_type" {
  description = "The instance type to use"
  type        = string
  default     = "t2.micro"
}
`,
			wantFinding: false,
		},
		{
			name: "variable without description",
			content: `variable "instance_type" {
  type    = string
  default = "t2.micro"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "variables.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-documented-variables" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformTypedVariablesRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "variable with type",
			content: `variable "instance_type" {
  description = "The instance type"
  type        = string
}
`,
			wantFinding: false,
		},
		{
			name: "variable without type",
			content: `variable "instance_type" {
  description = "The instance type"
  default     = "t2.micro"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "variables.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-typed-variables" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformNamingConventionRule(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-naming-convention" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformModulePinnedSourceRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "module with version",
			content: `module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}
`,
			wantFinding: false,
		},
		{
			name: "registry module without version",
			content: `module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`,
			wantFinding: true,
		},
		{
			name: "local module without version (allowed)",
			content: `module "local" {
  source = "./modules/vpc"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-module-pinned-source" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformDeprecatedSyntaxRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "no deprecated syntax",
			content: `resource "aws_instance" "example" {
  ami           = var.ami_id
  instance_type = local.instance_type
}
`,
			wantFinding: false,
		},
		{
			name: "deprecated interpolation syntax",
			content: `resource "aws_instance" "example" {
  ami           = "${var.ami_id}"
  instance_type = "t2.micro"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-deprecated-syntax" {
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

	// Verify we have all 10 rules registered
	assert.Len(t, rules, 10, "should have 10 rules registered")

	// Verify each rule has required methods
	for _, rule := range rules {
		assert.NotEmpty(t, rule.Name(), "rule name should not be empty")
		assert.NotEmpty(t, rule.Description(), "rule description should not be empty")
	}
}

func TestEngine_RuleDisabling(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	// Test with required_version rule disabled
	engine := New(&Config{
		Rules: map[string]RuleConfig{
			"lint.terraform-required-version": {
				Enabled: false,
			},
		},
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Should not find required_version findings since it's disabled
	for _, f := range findings {
		assert.NotEqual(t, "lint.terraform-required-version", f.Rule)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	engine := New(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := engine.Run(ctx, []string{tmpFile})
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGroupFilesByDirectory(t *testing.T) {
	engine := New(nil)

	files := []string{
		"/project/modules/vpc/main.tf",
		"/project/modules/vpc/variables.tf",
		"/project/modules/ec2/main.tf",
		"/project/environments/dev/main.tf",
	}

	result := engine.groupFilesByDirectory(files)

	assert.Len(t, result, 3, "should have 3 directories")
	assert.Len(t, result["/project/modules/vpc"], 2)
	assert.Len(t, result["/project/modules/ec2"], 1)
	assert.Len(t, result["/project/environments/dev"], 1)
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"error", "error"},
		{"warning", "warning"},
		{"info", "info"},
		{"ERROR", "error"},
		{"WARNING", "warning"},
		{"unknown", "warning"}, // defaults to warning
		{"", "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSeverity(tt.input)
			assert.Equal(t, tt.want, string(result))
		})
	}
}

func TestTerraformResourceCountRule(t *testing.T) {
	// Create content with many resources
	content := ""
	for i := 0; i < 20; i++ {
		content += `resource "aws_instance" "instance_` + string(rune('a'+i)) + `" {
  ami = "ami-12345"
}

`
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	found := false
	for _, f := range findings {
		if f.Rule == "lint.terraform-resource-count" {
			found = true
			break
		}
	}

	assert.True(t, found, "should find resource count warning")
}

func TestIsSimpleReference(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"var.name", true},
		{"local.value", true},
		{"data.aws_ami.latest", true},
		{"module.vpc.id", true},
		{"each.key", true},
		{"count.index", true},
		{"aws_instance.example.id", false},
		{"hello", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isSimpleReference(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTerraformDocumentedOutputsRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "output with description",
			content: `output "instance_id" {
  description = "The instance ID"
  value       = aws_instance.example.id
}
`,
			wantFinding: false,
		},
		{
			name: "output without description",
			content: `output "instance_id" {
  value = aws_instance.example.id
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "outputs.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-documented-outputs" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestEngine_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// main.tf with terraform block
	mainContent := `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

resource "aws_instance" "example" {
  ami           = var.ami_id
  instance_type = var.instance_type
}
`

	// variables.tf
	varsContent := `variable "ami_id" {
  description = "The AMI ID"
  type        = string
}

variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "t2.micro"
}
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(varsContent), 0644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{
		filepath.Join(tmpDir, "main.tf"),
		filepath.Join(tmpDir, "variables.tf"),
	})
	require.NoError(t, err)

	// Should have no major findings (everything is well-documented)
	for _, f := range findings {
		t.Logf("Finding: %s - %s", f.Rule, f.Message)
	}
}
