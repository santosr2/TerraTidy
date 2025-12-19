package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_New(t *testing.T) {
	// Test with nil config
	engine := New(nil)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.config)
	assert.NotNil(t, engine.parser)

	// Test with config
	config := &Config{
		PolicyDirs: []string{"./policies"},
	}
	engine = New(config)
	assert.NotNil(t, engine)
	assert.Equal(t, []string{"./policies"}, engine.config.PolicyDirs)
}

func TestEngine_Name(t *testing.T) {
	engine := New(nil)
	assert.Equal(t, "policy", engine.Name())
}

func TestEngine_Run_NoPolices(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Engine with no custom policies will use built-in policies
	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Should have findings from built-in policies (missing tags, missing terraform block, etc.)
	assert.NotNil(t, findings)
}

func TestEngine_Run_WithTerraformBlock(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "example"
  }
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Should have fewer findings since we have terraform block and tags
	for _, f := range findings {
		t.Logf("Finding: %s - %s", f.Rule, f.Message)
	}
}

func TestEngine_Run_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
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

func TestEngine_ParseModuleToJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main.tf
	mainContent := `terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}

data "aws_ami" "latest" {
  most_recent = true
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}
`
	// Create variables.tf
	varsContent := `variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "t2.micro"
}

output "instance_id" {
  description = "The instance ID"
  value       = aws_instance.example.id
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(varsContent), 0o644))

	engine := New(nil)
	data, err := engine.parseModuleToJSON([]string{
		filepath.Join(tmpDir, "main.tf"),
		filepath.Join(tmpDir, "variables.tf"),
	})
	require.NoError(t, err)

	// Verify resources
	resources := data["resources"].([]any)
	assert.Len(t, resources, 1)

	// Verify data sources
	dataSources := data["data"].([]any)
	assert.Len(t, dataSources, 1)

	// Verify modules
	modules := data["modules"].([]any)
	assert.Len(t, modules, 1)

	// Verify variables
	variables := data["variables"].([]any)
	assert.Len(t, variables, 1)

	// Verify outputs
	outputs := data["outputs"].([]any)
	assert.Len(t, outputs, 1)
}

func TestEngine_GetInput(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)
	jsonData, err := engine.GetInput([]string{tmpFile})
	require.NoError(t, err)

	assert.Contains(t, string(jsonData), "resources")
	assert.Contains(t, string(jsonData), "aws_instance")
}

func TestEngine_CustomPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Terraform file
	tfFile := filepath.Join(tmpDir, "main.tf")
	tfContent := `resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	require.NoError(t, os.WriteFile(tfFile, []byte(tfContent), 0o644))

	// Create custom policy
	policyDir := filepath.Join(tmpDir, "policies")
	require.NoError(t, os.MkdirAll(policyDir, 0o755))

	policyContent := `package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    msg := {
        "msg": "Custom policy: EC2 instance detected",
        "rule": "custom-ec2-check",
        "severity": "warning"
    }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(policyDir, "custom.rego"), []byte(policyContent), 0o644))

	engine := New(&Config{
		PolicyDirs: []string{policyDir},
	})

	findings, err := engine.Run(context.Background(), []string{tfFile})
	require.NoError(t, err)

	// Should find the custom policy violation
	found := false
	for _, f := range findings {
		if f.Rule == "policy.custom-ec2-check" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find custom policy violation")
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
		{"unknown", "error"}, // defaults to error for policies
		{"", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSeverity(tt.input)
			assert.Equal(t, tt.want, string(result))
		})
	}
}

func TestViolationToFinding_String(t *testing.T) {
	engine := New(nil)

	finding := engine.violationToFinding("Simple error message", "/path/to/dir")

	assert.Equal(t, "Simple error message", finding.Message)
	assert.Equal(t, "/path/to/dir", finding.File)
	assert.Equal(t, "policy.violation", finding.Rule)
}

func TestViolationToFinding_Map(t *testing.T) {
	engine := New(nil)

	violation := map[string]any{
		"msg":      "Security violation",
		"rule":     "no-public-ssh",
		"severity": "error",
		"file":     "/path/to/main.tf",
		"line":     float64(10),
	}

	finding := engine.violationToFinding(violation, "/path/to/dir")

	assert.Equal(t, "Security violation", finding.Message)
	assert.Equal(t, "policy.no-public-ssh", finding.Rule)
	assert.Equal(t, "/path/to/main.tf", finding.File)
	assert.Equal(t, 10, finding.Location.Start.Line)
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
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name        = "example"
    Environment = "dev"
  }
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

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(varsContent), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{
		filepath.Join(tmpDir, "main.tf"),
		filepath.Join(tmpDir, "variables.tf"),
	})
	require.NoError(t, err)

	// Log all findings for debugging
	for _, f := range findings {
		t.Logf("Finding: %s - %s", f.Rule, f.Message)
	}
}

func TestBuiltinPolicies(t *testing.T) {
	// Verify built-in policies are defined
	assert.Greater(t, len(builtinPolicies), 0, "should have built-in policies")

	// Each policy should contain package declaration
	for i, policy := range builtinPolicies {
		assert.Contains(t, policy, "package terraform", "policy %d should have package declaration", i)
	}
}
