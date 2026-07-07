package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Sample Terraform configurations for benchmarks
var (
	benchSimpleConfig = `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`

	benchConfigWithTerraformBlock = `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name        = "web"
    Environment = "production"
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"

  tags = {
    Name = "data"
  }
}

output "instance_id" {
  description = "Instance ID"
  value       = aws_instance.web.id
}
`

	benchComplexConfig = `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 3
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-gp2"]
  }
}

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags = local.common_tags
}

resource "aws_subnet" "public" {
  count = 3

  vpc_id     = aws_vpc.main.id
  cidr_block = cidrsubnet("10.0.0.0/16", 8, count.index)

  tags = merge(local.common_tags, {
    Name = "public-${count.index}"
  })
}

resource "aws_security_group" "web" {
  name        = "web-sg"
  description = "Security group for web servers"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

resource "aws_instance" "web" {
  count = var.instance_count

  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.public[count.index % 3].id
  vpc_security_group_ids = [aws_security_group.web.id]

  tags = merge(local.common_tags, {
    Name = "web-${count.index}"
    Role = "web"
  })
}

resource "aws_s3_bucket" "logs" {
  bucket = "my-logs-bucket"

  tags = local.common_tags
}

resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"

  tags = local.common_tags
}

module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 5.0"

  identifier = "mydb"
}

output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "instance_ids" {
  description = "Web instance IDs"
  value       = aws_instance.web[*].id
}
`

	// Custom policy with multiple rules for complex benchmark.
	// Note: Some rules (2, 4, 5) won't match due to data model mismatches, mirroring
	// the behavior of builtin policies. This is intentional for realistic benchmarks.
	benchComplexPolicy = `package terraform

import rego.v1

# Rule 1: Check for required tags on resources
deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not resource.tags
    msg := {
        "msg": sprintf("EC2 instance %s is missing tags", [resource.name]),
        "rule": "require-tags",
        "severity": "warning",
        "file": resource._file
    }
}

# Rule 2: Check for VPC CIDR blocks
warn contains msg if {
    some resource in input.resources
    resource.type == "aws_vpc"
    contains(resource.cidr_block, "/8")
    msg := {
        "msg": sprintf("VPC %s uses a /8 CIDR block which may be too large", [resource.name]),
        "rule": "vpc-cidr-size",
        "severity": "info",
        "file": resource._file
    }
}

# Rule 3: Check for module version constraints
warn contains msg if {
    some module in input.modules
    not module.version
    not startswith(module.source, "\"./")
    not startswith(module.source, "\"../")
    msg := {
        "msg": sprintf("Module %s should have a version constraint", [module.name]),
        "rule": "module-version",
        "severity": "warning",
        "file": module._file
    }
}

# Rule 4: Check for security group rules
deny contains msg if {
    some resource in input.resources
    resource.type == "aws_security_group"
    contains(resource.ingress, "0.0.0.0/0")
    contains(resource.ingress, "22")
    msg := {
        "msg": sprintf("Security group %s allows SSH from 0.0.0.0/0", [resource.name]),
        "rule": "no-public-ssh",
        "severity": "error",
        "file": resource._file
    }
}

# Rule 5: Check for S3 bucket naming
warn contains msg if {
    some resource in input.resources
    resource.type == "aws_s3_bucket"
    not contains(resource.bucket, "\"production")
    not contains(resource.bucket, "\"staging")
    not contains(resource.bucket, "\"dev")
    msg := {
        "msg": sprintf("S3 bucket %s should include environment in name", [resource.name]),
        "rule": "s3-naming",
        "severity": "info",
        "file": resource._file
    }
}
`
)

// BenchmarkPolicyEngine_SimpleConfig benchmarks policy evaluation with a simple config.
func BenchmarkPolicyEngine_SimpleConfig(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(benchSimpleConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil) // Uses built-in policies
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkPolicyEngine_MediumConfig benchmarks policy evaluation with a medium config.
func BenchmarkPolicyEngine_MediumConfig(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(benchConfigWithTerraformBlock), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkPolicyEngine_ComplexConfig benchmarks policy evaluation with a complex config.
func BenchmarkPolicyEngine_ComplexConfig(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(benchComplexConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkPolicyEngine_ComplexPolicy benchmarks with multiple custom policy rules.
func BenchmarkPolicyEngine_ComplexPolicy(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(benchComplexConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	// Write custom policy with multiple rules
	policyFile := filepath.Join(tmpDir, "policy.rego")
	if err := os.WriteFile(policyFile, []byte(benchComplexPolicy), 0o644); err != nil {
		b.Fatalf("failed to create policy file: %v", err)
	}

	engine := New(&Config{
		PolicyFiles: []string{policyFile},
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkPolicyEngine_MultipleFiles benchmarks policy evaluation across multiple files.
func BenchmarkPolicyEngine_MultipleFiles(b *testing.B) {
	tmpDir := b.TempDir()

	files := []struct {
		name    string
		content string
	}{
		{"main.tf", benchComplexConfig},
		{"variables.tf", `variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "env" {
  description = "Environment name"
  type        = string
}
`},
		{"outputs.tf", `output "region" {
  description = "AWS region"
  value       = var.region
}

output "env" {
  description = "Environment name"
  value       = var.env
}
`},
	}

	var filePaths []string
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
			b.Fatalf("failed to create file %s: %v", f.name, err)
		}
		filePaths = append(filePaths, path)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, filePaths)
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkPolicyEngine_InvalidHCL benchmarks error handling with malformed HCL input.
func BenchmarkPolicyEngine_InvalidHCL(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")

	// Invalid HCL that will fail parsing
	invalidContent := `resource "aws_instance" "web" {
  ami = "ami-12345"
  this is not valid HCL syntax {
`
	if err := os.WriteFile(tmpFile, []byte(invalidContent), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We expect this to not error (graceful handling) but produce findings
		_, _ = engine.Run(ctx, []string{tmpFile})
	}
}

// BenchmarkPolicyEngine_NoFindings benchmarks when all policies pass.
func BenchmarkPolicyEngine_NoFindings(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")

	// A config that should pass most built-in policies
	compliantConfig := `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name        = "web"
    Environment = "production"
    Team        = "engineering"
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "my-secure-bucket"

  tags = {
    Name = "data"
  }
}
`
	if err := os.WriteFile(tmpFile, []byte(compliantConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

// BenchmarkPolicyEngine_ParseModuleToJSONInternal benchmarks the HCL to JSON conversion.
// Note: This is a white-box benchmark of an internal method. Production code uses
// parseModuleToJSONWithSuppressions which includes annotation parsing overhead.
func BenchmarkPolicyEngine_ParseModuleToJSONInternal(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(tmpFile, []byte(benchComplexConfig), 0o644); err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	engine := New(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.parseModuleToJSON([]string{tmpFile})
	}
}
