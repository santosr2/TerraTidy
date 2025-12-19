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

func TestNewModuleData(t *testing.T) {
	data := newModuleData()

	// Verify all expected keys exist
	expectedKeys := []string{"resources", "data", "modules", "variables", "outputs", "locals", "providers", "terraform", "_files"}
	for _, key := range expectedKeys {
		_, ok := data[key]
		assert.True(t, ok, "should have key %s", key)
	}

	// Verify slices are initialized empty
	assert.Empty(t, data["resources"].([]any))
	assert.Empty(t, data["data"].([]any))
	assert.Empty(t, data["modules"].([]any))
	assert.Empty(t, data["variables"].([]any))
	assert.Empty(t, data["outputs"].([]any))
	assert.Empty(t, data["locals"].([]any))
	assert.Empty(t, data["providers"].([]any))
	assert.Empty(t, data["_files"].([]string))

	// Verify terraform is an empty map
	tfBlock := data["terraform"].(map[string]any)
	assert.Empty(t, tfBlock)
}

func TestAddLabeledBlock(t *testing.T) {
	tests := []struct {
		name      string
		labels    []string
		minLabels int
		keys      []string
		wantKeys  map[string]string
	}{
		{
			name:      "resource with type and name",
			labels:    []string{"aws_instance", "example"},
			minLabels: 2,
			keys:      []string{"type", "name"},
			wantKeys:  map[string]string{"type": "aws_instance", "name": "example"},
		},
		{
			name:      "module with single label",
			labels:    []string{"vpc"},
			minLabels: 1,
			keys:      []string{"name"},
			wantKeys:  map[string]string{"name": "vpc"},
		},
		{
			name:      "insufficient labels",
			labels:    []string{"only_one"},
			minLabels: 2,
			keys:      []string{"type", "name"},
			wantKeys:  map[string]string{}, // No keys added
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockData := make(map[string]any)
			addLabeledBlock(blockData, tt.labels, tt.minLabels, tt.keys...)

			for key, want := range tt.wantKeys {
				got, ok := blockData[key]
				assert.True(t, ok, "should have key %s", key)
				assert.Equal(t, want, got)
			}

			if len(tt.wantKeys) == 0 {
				// Verify no keys were added for insufficient labels
				for _, key := range tt.keys {
					_, ok := blockData[key]
					assert.False(t, ok, "should not have key %s", key)
				}
			}
		})
	}
}

func TestAppendToSlice(t *testing.T) {
	moduleData := newModuleData()

	// Append to resources
	blockData1 := map[string]any{"type": "aws_instance", "name": "test1"}
	blockData2 := map[string]any{"type": "aws_s3_bucket", "name": "test2"}

	appendToSlice(moduleData, "resources", blockData1)
	appendToSlice(moduleData, "resources", blockData2)

	resources := moduleData["resources"].([]any)
	assert.Len(t, resources, 2)
	assert.Equal(t, "aws_instance", resources[0].(map[string]any)["type"])
	assert.Equal(t, "aws_s3_bucket", resources[1].(map[string]any)["type"])
}

func TestEvaluateQuery_InvalidPolicy(t *testing.T) {
	engine := New(nil)

	evalCtx := &policyEvalContext{
		ctx:        context.Background(),
		moduleData: newModuleData(),
		dir:        "/test/dir",
	}

	// Invalid Rego policy should return nil (error handled gracefully)
	invalidPolicy := "this is not valid rego {"
	findings := engine.evaluateQuery(evalCtx, invalidPolicy, "data.terraform.deny", "error")

	assert.Nil(t, findings)
}

func TestEvaluateQuery_ValidPolicy(t *testing.T) {
	engine := New(nil)

	moduleData := newModuleData()
	moduleData["resources"] = []any{
		map[string]any{
			"type":  "aws_instance",
			"name":  "test",
			"_file": "/test/main.tf",
		},
	}

	evalCtx := &policyEvalContext{
		ctx:        context.Background(),
		moduleData: moduleData,
		dir:        "/test/dir",
	}

	// Policy that always triggers on aws_instance
	policy := `package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    msg := {
        "msg": "Found EC2 instance",
        "rule": "test-rule"
    }
}
`
	findings := engine.evaluateQuery(evalCtx, policy, "data.terraform.deny", "error")

	assert.Len(t, findings, 1)
	assert.Equal(t, "Found EC2 instance", findings[0].Message)
	assert.Equal(t, "policy.test-rule", findings[0].Rule)
}

func TestExtractFindings_EmptyResults(t *testing.T) {
	engine := New(nil)

	// Empty result set
	findings := engine.extractFindings(nil, "/test/dir", "error")
	assert.Empty(t, findings)
}

func TestParseFileIntoModule_NonexistentFile(t *testing.T) {
	engine := New(nil)
	moduleData := newModuleData()

	// Should not panic on nonexistent file
	engine.parseFileIntoModule("/nonexistent/file.tf", moduleData)

	// Module data should remain unchanged
	assert.Empty(t, moduleData["resources"].([]any))
	assert.Empty(t, moduleData["_files"].([]string))
}

func TestParseFileIntoModule_InvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.tf")

	// Write invalid HCL
	require.NoError(t, os.WriteFile(tmpFile, []byte("this is { not valid hcl"), 0o644))

	engine := New(nil)
	moduleData := newModuleData()

	// Should not panic on invalid HCL
	engine.parseFileIntoModule(tmpFile, moduleData)

	// File should not be added since parsing failed
	assert.Empty(t, moduleData["_files"].([]string))
}
