package cache

import (
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

resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
}
`)
}

func BenchmarkCacheHit(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	c := NewDefault()

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

func BenchmarkCacheMiss(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := NewDefault() // Fresh cache each iteration
		for _, file := range files {
			_, err := c.GetOrParse(file)
			if err != nil {
				b.Fatalf("cache error: %v", err)
			}
		}
	}
}

func BenchmarkCacheGetOrParse(b *testing.B) {
	files := setupBenchmarkFiles(b, 10)
	c := NewDefault()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			_, err := c.GetOrParse(file)
			if err != nil {
				b.Fatalf("cache error: %v", err)
			}
		}
	}
}
