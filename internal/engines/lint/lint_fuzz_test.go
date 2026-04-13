package lint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// FuzzLintCheck exercises lint rules against arbitrary HCL input.
// Uses built-in rules only (no TFLint) to ensure no panics on malformed input.
func FuzzLintCheck(f *testing.F) {
	// Valid HCL seeds with potential lint issues
	f.Add([]byte(`resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`))

	// Deprecated resource
	f.Add([]byte(`resource "aws_s3_bucket_object" "deprecated" {
  bucket = "my-bucket"
  key    = "key"
}
`))

	// Hardcoded AMI
	f.Add([]byte(`resource "aws_instance" "hardcoded" {
  ami           = "ami-0123456789abcdef0"
  instance_type = "t2.micro"
}
`))

	// Empty description
	f.Add([]byte(`variable "name" {
  type = string
}
`))

	// Sensitive variable without description
	f.Add([]byte(`variable "password" {
  type      = string
  sensitive = true
}
`))

	// Module without version
	f.Add([]byte(`module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`))

	// Module with count
	f.Add([]byte(`module "servers" {
  source = "./modules/server"
  count  = 3
}
`))

	// Data source
	f.Add([]byte(`data "aws_ami" "latest" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn2-ami-*"]
  }
}
`))

	// Provider configuration
	f.Add([]byte(`provider "aws" {
  region = "us-west-2"
}
`))

	// Terraform block
	f.Add([]byte(`terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`))

	// Edge cases (invalid/malformed HCL)
	f.Add([]byte(``))              // Empty
	f.Add([]byte(`{`))             // Invalid HCL
	f.Add([]byte(`}`))             // Invalid HCL
	f.Add([]byte(`= value`))       // Invalid HCL
	f.Add([]byte(`resource {}`))   // Missing labels
	f.Add([]byte(`"just string"`)) // Just a string

	// Complex expressions
	f.Add([]byte(`locals {
  enabled = var.enabled && length(var.items) > 0
  name    = coalesce(var.name, "default")
  tags    = merge(var.base_tags, { Name = local.name })
}
`))

	// Deep nesting
	f.Add([]byte(`resource "test" "nested" {
  dynamic "block" {
    for_each = var.items
    content {
      nested {
        value = block.value
      }
    }
  }
}
`))

	// Unicode
	f.Add([]byte(`variable "日本語" {
  default = "テスト"
}
`))

	// Heredoc
	f.Add([]byte(`resource "aws_iam_policy" "policy" {
  name = "test"
  policy = <<-EOF
    {
      "Version": "2012-10-17",
      "Statement": []
    }
  EOF
}
`))

	// Create engine with built-in rules only (no TFLint)
	engine := New(&Config{
		UseTFLint:       false,
		FallbackBuiltin: true,
		Rules:           make(map[string]RuleConfig),
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Write to temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "main.tf")
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Run lint checks - should not panic regardless of input
		_, _ = engine.Run(context.Background(), []string{tmpFile})
	})
}
