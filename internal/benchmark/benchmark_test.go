// Package benchmark provides performance benchmarks for TerraTidy components.
package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/terratidy/internal/cache"
	fmtengine "github.com/santosr2/terratidy/internal/engines/format"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/santosr2/terratidy/internal/runner"
)

// setupTestFiles creates temporary Terraform files for benchmarking
func setupTestFiles(b *testing.B, count int) (string, []string) {
	b.Helper()

	tmpDir := b.TempDir()
	files := make([]string, count)

	for i := 0; i < count; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i%10))+string(rune('0'+i/10))+".tf")
		content := generateTerraformContent(i)
		if err := os.WriteFile(filename, content, 0o644); err != nil {
			b.Fatalf("failed to create test file: %v", err)
		}
		files[i] = filename
	}

	return tmpDir, files
}

// generateTerraformContent creates realistic Terraform content
func generateTerraformContent(seed int) []byte {
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

// BenchmarkFmtEngine benchmarks the format engine
func BenchmarkFmtEngine(b *testing.B) {
	_, files := setupTestFiles(b, 10)
	engine := fmtengine.New(&fmtengine.Config{Check: true})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, files)
		if err != nil {
			b.Fatalf("engine error: %v", err)
		}
	}
}

// BenchmarkStyleEngine benchmarks the style engine
func BenchmarkStyleEngine(b *testing.B) {
	_, files := setupTestFiles(b, 10)
	engine := style.New(&style.Config{
		Fix:   false,
		Rules: make(map[string]style.RuleConfig),
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Run(ctx, files)
		if err != nil {
			b.Fatalf("engine error: %v", err)
		}
	}
}

// BenchmarkRunnerSequential benchmarks sequential execution
func BenchmarkRunnerSequential(b *testing.B) {
	_, files := setupTestFiles(b, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := runner.New().
			AddEngine(fmtengine.New(&fmtengine.Config{Check: true})).
			AddEngine(style.New(&style.Config{Fix: false, Rules: make(map[string]style.RuleConfig)})).
			SetParallel(false)

		_, err := r.Run(ctx, files)
		if err != nil {
			b.Fatalf("runner error: %v", err)
		}
	}
}

// BenchmarkRunnerParallel benchmarks parallel execution
func BenchmarkRunnerParallel(b *testing.B) {
	_, files := setupTestFiles(b, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := runner.New().
			AddEngine(fmtengine.New(&fmtengine.Config{Check: true})).
			AddEngine(style.New(&style.Config{Fix: false, Rules: make(map[string]style.RuleConfig)})).
			SetParallel(true)

		_, err := r.Run(ctx, files)
		if err != nil {
			b.Fatalf("runner error: %v", err)
		}
	}
}

// BenchmarkCacheHit benchmarks cache hit performance
func BenchmarkCacheHit(b *testing.B) {
	_, files := setupTestFiles(b, 10)
	c := cache.NewDefault()

	// Warm up cache
	for _, file := range files {
		_, err := c.GetOrParse(file)
		if err != nil {
			b.Fatalf("cache error: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			_, ok := c.Get(file)
			if !ok {
				b.Fatal("expected cache hit")
			}
		}
	}
}

// BenchmarkCacheMiss benchmarks cache miss (parse) performance
func BenchmarkCacheMiss(b *testing.B) {
	_, files := setupTestFiles(b, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := cache.NewDefault() // Fresh cache each iteration
		for _, file := range files {
			_, err := c.GetOrParse(file)
			if err != nil {
				b.Fatalf("cache error: %v", err)
			}
		}
	}
}

// BenchmarkFileCount benchmarks performance with varying file counts
func BenchmarkFileCount(b *testing.B) {
	fileCounts := []int{1, 5, 10, 25, 50}

	for _, count := range fileCounts {
		b.Run("files="+string(rune('0'+count/10))+string(rune('0'+count%10)), func(b *testing.B) {
			_, files := setupTestFiles(b, count)
			engine := fmtengine.New(&fmtengine.Config{Check: true})
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
