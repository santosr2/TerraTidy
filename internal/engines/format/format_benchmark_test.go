package format

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// setupBenchmarkFiles creates temporary Terraform files for benchmarking
func setupBenchmarkFiles(b *testing.B, count int) []string {
	b.Helper()

	tmpDir := b.TempDir()
	files := make([]string, count)

	for i := 0; i < count; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i%10))+string(rune('0'+i/10))+".tf")
		content := generateBenchmarkContent(i)
		if err := os.WriteFile(filename, content, 0o644); err != nil {
			b.Fatalf("failed to create test file: %v", err)
		}
		files[i] = filename
	}

	return files
}

// generateBenchmarkContent creates realistic Terraform content
func generateBenchmarkContent(seed int) []byte {
	return []byte(`# Terraform configuration file ` + string(rune('0'+seed%10)) + `

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "environment" {
  type        = string
  description = "The deployment environment"
  default     = "dev"
}

variable "instance_count" {
  type        = number
  description = "Number of instances to create"
  default     = 1
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_instance" "example" {
  count         = var.instance_count
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  tags = merge(local.common_tags, {
    Name = "instance-${count.index}"
  })
}

output "instance_ids" {
  value       = aws_instance.example[*].id
  description = "The IDs of the created instances"
}
`)
}

func BenchmarkFormatEngine(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	engine := New(&Config{Check: true})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, files)
		if err != nil {
			b.Fatalf("engine error: %v", err)
		}
	}
}

func BenchmarkFormatEngineWithWrite(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	engine := New(&Config{Check: false}) // Actually write formatted content
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, files)
		if err != nil {
			b.Fatalf("engine error: %v", err)
		}
	}
}

func BenchmarkFormatFileCount(b *testing.B) {
	fileCounts := []int{1, 5, 10, 25, 50}

	for _, count := range fileCounts {
		b.Run("files="+formatCount(count), func(b *testing.B) {
			files := setupBenchmarkFiles(b, count)
			engine := New(&Config{Check: true})
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := engine.Run(ctx, files)
				if err != nil {
					b.Fatalf("engine error: %v", err)
				}
			}
		})
	}
}

func formatCount(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// generateLargeContent creates a large Terraform file with 1000+ lines.
// Generates approximately 50 resources with nested blocks to exceed 1000 lines.
func generateLargeContent() []byte {
	var content []byte

	// Header (~20 lines)
	content = append(content, []byte(`# Large Terraform configuration for benchmark testing
# This file contains multiple resources to simulate real-world modules

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

variable "environment" {
  type        = string
  description = "The deployment environment"
  default     = "production"
}

variable "region" {
  type        = string
  description = "AWS region"
  default     = "us-west-2"
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
    Project     = "benchmark"
  }
}

`)...)

	// Generate 50 resources (~20 lines each = ~1000 lines)
	for i := 0; i < 50; i++ {
		resourceBlock := []byte(`resource "aws_instance" "server_` + formatCount(i) + `" {
  ami           = "ami-0123456789abcdef` + string(rune('0'+i%10)) + `"
  instance_type = "t3.medium"
  subnet_id     = aws_subnet.main.id

  root_block_device {
    volume_size           = 100
    volume_type           = "gp3"
    encrypted             = true
    delete_on_termination = true
  }

  tags = merge(local.common_tags, {
    Name = "server-` + formatCount(i) + `-${var.environment}"
    Role = "application"
  })

  lifecycle {
    create_before_destroy = true
  }
}

`)
		content = append(content, resourceBlock...)
	}

	// Add outputs (~5 lines each, 10 outputs = 50 lines)
	for i := 0; i < 10; i++ {
		outputBlock := []byte(`output "server_` + formatCount(i) + `_id" {
  value       = aws_instance.server_` + formatCount(i) + `.id
  description = "Instance ID for server ` + formatCount(i) + `"
}

`)
		content = append(content, outputBlock...)
	}

	return content
}

func BenchmarkFormatLargeFile(b *testing.B) {
	tmpDir := b.TempDir()
	largeFile := filepath.Join(tmpDir, "large_module.tf")

	content := generateLargeContent()
	if err := os.WriteFile(largeFile, content, 0o644); err != nil {
		b.Fatalf("failed to create large test file: %v", err)
	}

	b.Logf("Large file size: %d bytes, ~%d lines", len(content), countLines(content))

	engine := New(&Config{Check: true})
	ctx := context.Background()
	files := []string{largeFile}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, files)
		if err != nil {
			b.Fatalf("engine error: %v", err)
		}
	}
}

func countLines(content []byte) int {
	count := 0
	for _, b := range content {
		if b == '\n' {
			count++
		}
	}
	return count
}
