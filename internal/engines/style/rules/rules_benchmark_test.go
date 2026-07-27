package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// benchmarkHCL is a realistic Terraform configuration for benchmarking
const benchmarkHCL = `
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

// largeHCL generates a larger configuration for scaling tests
const largeHCL = `
terraform {
  required_version = ">= 1.0"
}

variable "v1" { type = string }
variable "v2" { type = string }
variable "v3" { type = string }
variable "v4" { type = string }
variable "v5" { type = string }

locals {
  l1 = "value1"
  l2 = "value2"
  l3 = "value3"
}

resource "aws_instance" "r1" { ami = "ami-1" }
resource "aws_instance" "r2" { ami = "ami-2" }
resource "aws_instance" "r3" { ami = "ami-3" }
resource "aws_instance" "r4" { ami = "ami-4" }
resource "aws_instance" "r5" { ami = "ami-5" }
resource "aws_instance" "r6" { ami = "ami-6" }
resource "aws_instance" "r7" { ami = "ami-7" }
resource "aws_instance" "r8" { ami = "ami-8" }
resource "aws_instance" "r9" { ami = "ami-9" }
resource "aws_instance" "r10" { ami = "ami-10" }

data "aws_ami" "d1" { most_recent = true }
data "aws_ami" "d2" { most_recent = true }
data "aws_ami" "d3" { most_recent = true }

output "o1" { value = aws_instance.r1.id }
output "o2" { value = aws_instance.r2.id }
output "o3" { value = aws_instance.r3.id }
`

func parseHCL(b *testing.B, content string) *hcl.File {
	b.Helper()
	file, diags := hclsyntax.ParseConfig([]byte(content), "benchmark.tf", hcl.InitialPos)
	if diags.HasErrors() {
		b.Fatalf("failed to parse HCL: %v", diags)
	}
	return file
}

func BenchmarkResourceNameConventionRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &ResourceNameConventionRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkBlankLineBetweenBlocksRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &BlankLineBetweenBlocksRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkMetaArgumentsOrderRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &MetaArgumentsOrderRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkForEachCountFirstRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &ForEachCountFirstRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkNestedBlockOrderRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &NestedBlockOrderRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkCommentSyntaxRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &CommentSyntaxRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkScopedFileOrganizationRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &ScopedFileOrganizationRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkVariablesInFileRule(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	rule := &VariablesInFileRule{}
	ctx := &sdk.Context{File: "benchmark.tf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Check(ctx, file)
	}
}

func BenchmarkAllRules_SmallConfig(b *testing.B) {
	file := parseHCL(b, benchmarkHCL)
	ctx := &sdk.Context{File: "benchmark.tf"}

	rules := []sdk.Rule{
		&ResourceNameConventionRule{},
		&BlankLineBetweenBlocksRule{},
		&MetaArgumentsOrderRule{},
		&ForEachCountFirstRule{},
		&NestedBlockOrderRule{},
		&CommentSyntaxRule{},
		&ScopedFileOrganizationRule{},
		&VariablesInFileRule{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, rule := range rules {
			_, _ = rule.Check(ctx, file)
		}
	}
}

func BenchmarkAllRules_LargeConfig(b *testing.B) {
	file := parseHCL(b, largeHCL)
	ctx := &sdk.Context{File: "benchmark.tf"}

	rules := []sdk.Rule{
		&ResourceNameConventionRule{},
		&BlankLineBetweenBlocksRule{},
		&MetaArgumentsOrderRule{},
		&ForEachCountFirstRule{},
		&NestedBlockOrderRule{},
		&CommentSyntaxRule{},
		&ScopedFileOrganizationRule{},
		&VariablesInFileRule{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, rule := range rules {
			_, _ = rule.Check(ctx, file)
		}
	}
}

// singleFindingTagsAtEndHCL is one resource block with tags placed after lifecycle —
// produces exactly one finding from TagsAtEndRule.Check and one target for Fix.
const singleFindingTagsAtEndHCL = `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  lifecycle {
    prevent_destroy = true
  }

  tags = {
    Name = "example"
  }
}
`

// generateMultiFindingTagsAtEndHCL returns HCL with n resource blocks, each with tags
// placed after lifecycle. Produces n findings and n Fix targets for the multi-finding
// arm of BenchmarkStructuralFix.
func generateMultiFindingTagsAtEndHCL(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `resource "aws_instance" "r%d" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  lifecycle {
    prevent_destroy = true
  }

  tags = {
    Name = "r%d"
  }
}

`, i, i)
	}
	return sb.String()
}

// BenchmarkStructuralFix measures a Fix that performs structural ordering
// changes — the representative shape upcoming CST-based rules will follow.
// Acts as the pre-migration baseline that follow-up CST rule work compares
// against.
//
// Pre-migration baseline captured 2026-06-16 on the line-based TagsAtEndRule.Fix path (linux/amd64, Intel i7-9750H @ 2.60GHz):
//
//	SingleFinding:  99,803 ns/op,  80,568 B/op,   242 allocs/op
//	MultiFinding:  833,152 ns/op, 798,777 B/op, 2,907 allocs/op
//
// Source: go test -bench=BenchmarkStructuralFix -benchmem -benchtime=3s -count=3 -cpu=1 ./internal/engines/style/rules/
// Performance budget for the CST migration: SingleFinding regression ≤10% vs. these numbers on the same hardware. Use -cpu=1 — default scheduling has 2-3x variance on this hardware.
func BenchmarkStructuralFix(b *testing.B) {
	rule := &TagsAtEndRule{}

	b.Run("SingleFinding", func(b *testing.B) {
		tmpDir := b.TempDir()
		tmpFile := filepath.Join(tmpDir, "single.tf")
		if err := os.WriteFile(tmpFile, []byte(singleFindingTagsAtEndHCL), 0o600); err != nil {
			b.Fatalf("write fixture: %v", err)
		}
		ctx := &sdk.Context{File: tmpFile}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := rule.Fix(ctx, nil); err != nil {
				b.Fatalf("fix: %v", err)
			}
		}
	})

	b.Run("MultiFinding", func(b *testing.B) {
		content := generateMultiFindingTagsAtEndHCL(10)
		tmpDir := b.TempDir()
		tmpFile := filepath.Join(tmpDir, "multi.tf")
		if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
			b.Fatalf("write fixture: %v", err)
		}
		ctx := &sdk.Context{File: tmpFile}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := rule.Fix(ctx, nil); err != nil {
				b.Fatalf("fix: %v", err)
			}
		}
	})
}
