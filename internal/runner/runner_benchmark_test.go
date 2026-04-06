package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fmtengine "github.com/santosr2/TerraTidy/internal/engines/format"
	"github.com/santosr2/TerraTidy/internal/engines/style"
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

resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  tags = {
    Environment = var.environment
  }
}
`)
}

func BenchmarkRunnerSequential(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New().
			AddEngine(fmtengine.New(&fmtengine.Config{Check: true})).
			AddEngine(style.New(&style.Config{Fix: false, Rules: make(map[string]style.RuleConfig)})).
			SetParallel(false)

		_, err := r.Run(ctx, files)
		if err != nil {
			b.Fatalf("runner error: %v", err)
		}
	}
}

func BenchmarkRunnerParallel(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New().
			AddEngine(fmtengine.New(&fmtengine.Config{Check: true})).
			AddEngine(style.New(&style.Config{Fix: false, Rules: make(map[string]style.RuleConfig)})).
			SetParallel(true)

		_, err := r.Run(ctx, files)
		if err != nil {
			b.Fatalf("runner error: %v", err)
		}
	}
}

func BenchmarkRunnerSingleEngine(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New().
			AddEngine(fmtengine.New(&fmtengine.Config{Check: true}))

		_, err := r.Run(ctx, files)
		if err != nil {
			b.Fatalf("runner error: %v", err)
		}
	}
}

func BenchmarkRunnerMultipleEngines(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New().
			AddEngine(fmtengine.New(&fmtengine.Config{Check: true})).
			AddEngine(style.New(&style.Config{Fix: false, Rules: make(map[string]style.RuleConfig)})).
			SetParallel(true)

		_, err := r.Run(ctx, files)
		if err != nil {
			b.Fatalf("runner error: %v", err)
		}
	}
}
