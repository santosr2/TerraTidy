package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedFileOrganizationRule(t *testing.T) {
	rule := &ScopedFileOrganizationRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.scoped-file-organization", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		fileName     string
		content      string
		wantFindings int
	}{
		{
			name:     "network resources in network.tf",
			fileName: "network.tf",
			content: `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}`,
			wantFindings: 0,
		},
		{
			name:     "compute resources in network.tf",
			fileName: "network.tf",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 1,
		},
		{
			name:     "database resources in database.tf",
			fileName: "database.tf",
			content: `resource "aws_dynamodb_table" "main" {
  name     = "my-table"
  hash_key = "id"
}`,
			wantFindings: 0,
		},
		{
			name:     "storage resources in storage.tf",
			fileName: "storage.tf",
			content: `resource "aws_s3_bucket" "logs" {
  bucket = "my-logs-bucket"
}`,
			wantFindings: 0,
		},
		{
			name:     "iam resources in iam.tf",
			fileName: "iam.tf",
			content: `resource "aws_iam_role" "main" {
  name = "main-role"
  assume_role_policy = jsonencode({})
}`,
			wantFindings: 0,
		},
		{
			name:     "non-scoped file allows anything",
			fileName: "main.tf",
			content: `resource "aws_instance" "web" {
  ami = "ami-123"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}`,
			wantFindings: 0,
		},
		{
			name:     "data source in network.tf",
			fileName: "network.tf",
			content: `data "aws_vpc" "existing" {
  default = true
}`,
			wantFindings: 0, // VPC data in network.tf is correct
		},
		{
			name:     "lambda resources in lambda.tf",
			fileName: "lambda.tf",
			content: `resource "aws_lambda_function" "main" {
  function_name = "my-function"
  handler       = "index.handler"
  runtime       = "nodejs18.x"
}`,
			wantFindings: 0,
		},
		{
			name:     "container resources in container.tf",
			fileName: "container.tf",
			content: `resource "aws_ecr_repository" "main" {
  name = "my-repo"
}`,
			wantFindings: 0,
		},
		{
			name:     "monitoring resources in monitoring.tf",
			fileName: "monitoring.tf",
			content: `resource "aws_cloudwatch_metric_alarm" "cpu" {
  alarm_name          = "cpu-alarm"
  comparison_operator = "GreaterThanThreshold"
}`,
			wantFindings: 0,
		},
		{
			name:     "security resources in security.tf",
			fileName: "security.tf",
			content: `resource "aws_kms_key" "main" {
  description = "Main encryption key"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.fileName)
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
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

func TestScopedFileOrganizationRule_GetFileScope(t *testing.T) {
	rule := &ScopedFileOrganizationRule{}

	tests := []struct {
		fileName string
		expected string
	}{
		{"network.tf", "network"},
		{"network_public.tf", "network"},
		{"public_network.tf", "network"},
		{"database.tf", "database"},
		{"compute.tf", "compute"},
		{"storage.tf", "storage"},
		{"iam.tf", "iam"},
		{"security.tf", "security"},
		{"monitoring.tf", "monitoring"},
		{"lambda.tf", "lambda"},
		{"container.tf", "container"},
		{"dns.tf", "dns"},
		{"main.tf", ""},        // Not a scoped file
		{"outputs.tf", ""},     // Not a scoped file
		{"variables.tf", ""},   // Not a scoped file
		{"providers.tf", ""},   // Not a scoped file
		{"random_name.tf", ""}, // Unknown scope
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			result := rule.getFileScope(tt.fileName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScopedFileOrganizationRule_GetResourceScope(t *testing.T) {
	rule := &ScopedFileOrganizationRule{}

	// Note: The scope keywords in the rule may have overlaps and map iteration is non-deterministic
	// So we test that any result is non-empty for resources that should have a scope,
	// and test specific unique keywords
	t.Run("returns empty for unscoped resources", func(t *testing.T) {
		result := rule.getResourceScope("null_resource")
		assert.Equal(t, "", result)
	})

	t.Run("returns empty for random resources", func(t *testing.T) {
		result := rule.getResourceScope("random_string")
		assert.Equal(t, "", result)
	})

	t.Run("returns non-empty for vpc resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_vpc")
		assert.NotEmpty(t, result)
		assert.Equal(t, "network", result)
	})

	t.Run("returns non-empty for subnet resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_subnet")
		assert.NotEmpty(t, result)
		assert.Equal(t, "network", result)
	})

	t.Run("returns non-empty for s3 resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_s3_bucket")
		assert.NotEmpty(t, result)
		assert.Equal(t, "storage", result)
	})

	t.Run("returns non-empty for dynamodb resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_dynamodb_table")
		assert.NotEmpty(t, result)
		assert.Equal(t, "database", result)
	})

	t.Run("returns non-empty for iam resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_iam_role")
		assert.NotEmpty(t, result)
		assert.Equal(t, "iam", result)
	})

	t.Run("returns non-empty for kms resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_kms_key")
		assert.NotEmpty(t, result)
		assert.Equal(t, "security", result)
	})

	t.Run("returns non-empty for cloudwatch resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_cloudwatch_metric_alarm")
		assert.NotEmpty(t, result)
		assert.Equal(t, "monitoring", result)
	})

	t.Run("returns non-empty for lambda resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_lambda_function")
		assert.NotEmpty(t, result)
		assert.Equal(t, "lambda", result)
	})

	t.Run("returns non-empty for ecr resources", func(t *testing.T) {
		result := rule.getResourceScope("aws_ecr_repository")
		assert.NotEmpty(t, result)
		assert.Equal(t, "container", result)
	})
}

func TestTerraformFilesStructureRule(t *testing.T) {
	rule := &TerraformFilesStructureRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.terraform-files-structure", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		fileName     string
		content      string
		setupFiles   []string // Additional files to create in the temp dir
		wantFindings int
	}{
		{
			name:     "variables in variables.tf is fine",
			fileName: "variables.tf",
			content: `variable "region" {
  type    = string
  default = "us-east-1"
}`,
			wantFindings: 0,
		},
		{
			name:     "outputs in outputs.tf is fine",
			fileName: "outputs.tf",
			content: `output "vpc_id" {
  value = aws_vpc.main.id
}`,
			wantFindings: 0,
		},
		{
			name:     "providers in providers.tf is fine",
			fileName: "providers.tf",
			content: `provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 0,
		},
		{
			name:     "terraform in versions.tf is fine",
			fileName: "versions.tf",
			content: `terraform {
  required_version = ">= 1.0"
}`,
			wantFindings: 0,
		},
		{
			name:     "locals in locals.tf is fine",
			fileName: "locals.tf",
			content: `locals {
  environment = "production"
}`,
			wantFindings: 0,
		},
		{
			name:     "variable in main.tf when variables.tf exists",
			fileName: "main.tf",
			content: `variable "region" {
  type = string
}`,
			setupFiles:   []string{"variables.tf"},
			wantFindings: 1,
		},
		{
			name:     "output in main.tf when outputs.tf exists",
			fileName: "main.tf",
			content: `output "vpc_id" {
  value = aws_vpc.main.id
}`,
			setupFiles:   []string{"outputs.tf"},
			wantFindings: 1,
		},
		{
			name:     "provider in main.tf when providers.tf exists",
			fileName: "main.tf",
			content: `provider "aws" {
  region = "us-east-1"
}`,
			setupFiles:   []string{"providers.tf"},
			wantFindings: 1,
		},
		{
			name:     "resources in main.tf is fine",
			fileName: "main.tf",
			content: `resource "aws_instance" "web" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}`,
			wantFindings: 0,
		},
		{
			name:     "variable in main.tf suggests standard file",
			fileName: "main.tf",
			content: `variable "region" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name:     "output in main.tf suggests standard file",
			fileName: "main.tf",
			content: `output "vpc_id" {
  value = "vpc-123"
}`,
			wantFindings: 1,
		},
		{
			name:     "provider in main.tf suggests standard file",
			fileName: "main.tf",
			content: `provider "aws" {
  region = "us-east-1"
}`,
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.fileName)
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Create any additional setup files
			for _, setupFile := range tt.setupFiles {
				setupPath := filepath.Join(tmpDir, setupFile)
				err := os.WriteFile(setupPath, []byte("# placeholder"), 0o644)
				require.NoError(t, err)
			}

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

func TestTerraformFilesStructureRule_FileExists(t *testing.T) {
	rule := &TerraformFilesStructureRule{}

	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.tf")
	err := os.WriteFile(existingFile, []byte("# test"), 0o644)
	require.NoError(t, err)

	nonExistingFile := filepath.Join(tmpDir, "nonexisting.tf")

	assert.True(t, rule.fileExists(existingFile))
	assert.False(t, rule.fileExists(nonExistingFile))
}

func TestTerraformFilesStructureRule_ShouldSuggestStandardFile(t *testing.T) {
	rule := &TerraformFilesStructureRule{}

	tests := []struct {
		blockType string
		expected  bool
	}{
		{"variable", true},
		{"output", true},
		{"provider", true},
		{"resource", false},
		{"data", false},
		{"module", false},
		{"locals", false},
		{"terraform", false},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			result := rule.shouldSuggestStandardFile(tt.blockType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
