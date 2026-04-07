package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santosr2/TerraTidy/internal/engines/lint"
	"github.com/santosr2/TerraTidy/internal/engines/style"
)

// benchmarkTFContent is a realistic Terraform configuration for benchmarking
const benchmarkTFContent = `
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
  description = "Number of instances"
  default     = 1
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_instance" "web_server" {
  count         = var.instance_count
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  tags = merge(local.common_tags, {
    Name = "web-${count.index}"
  })
}

resource "aws_instance" "app_server" {
  ami           = "ami-12345678"
  instance_type = "t3.small"

  tags = local.common_tags
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]
}

output "instance_ids" {
  value       = aws_instance.web_server[*].id
  description = "The IDs of the web instances"
}
`

func setupBenchmarkServer(b *testing.B) (*Server, string, string) {
	b.Helper()

	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(testFile, []byte(benchmarkTFContent), 0o644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	out := &bytes.Buffer{}
	server := NewServer(strings.NewReader(""), out)
	server.initialized = true
	server.workspaceRoot = tmpDir

	// Initialize engines
	server.lintEngine = lint.New(nil)
	server.styleEngine = style.New(nil)

	// Enable engines via initOptions
	server.initOptions = &InitializationOptions{
		Engines: EngineToggles{
			Fmt:    true,
			Style:  true,
			Lint:   true,
			Policy: false,
		},
	}

	uri := pathToFileURI(testFile)
	server.docMu.Lock()
	server.documents[uri] = &Document{
		URI:     uri,
		Content: benchmarkTFContent,
		Version: 1,
	}
	server.docMu.Unlock()

	return server, uri, tmpDir
}

func BenchmarkGetDiagnostics(b *testing.B) {
	server, uri, _ := setupBenchmarkServer(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.getDiagnostics(uri)
	}
}

func BenchmarkGetDiagnostics_StyleOnly(b *testing.B) {
	server, uri, _ := setupBenchmarkServer(b)
	server.lintEngine = nil
	server.initOptions.Engines.Lint = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.getDiagnostics(uri)
	}
}

func BenchmarkGetDiagnostics_LintOnly(b *testing.B) {
	server, uri, _ := setupBenchmarkServer(b)
	server.styleEngine = nil
	server.initOptions.Engines.Style = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.getDiagnostics(uri)
	}
}

func BenchmarkGetDiagnostics_NoEngines(b *testing.B) {
	server, uri, _ := setupBenchmarkServer(b)
	server.initOptions.Engines.Style = false
	server.initOptions.Engines.Lint = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.getDiagnostics(uri)
	}
}
