package cst

import (
	"fmt"
	"strings"
	"testing"
)

// These two benchmarks pin the cost of Build (HCL → CST) and File.Bytes
// (CST → HCL) so future structural-fix work can measure overhead against
// a stable baseline. Sub-benchmarks across Small/Medium/Large input sizes
// catch regressions that only show on one shape (e.g. a per-block
// quadratic gap-scan that disappears at low item counts).
//
// Reference snapshot (NOT absolute thresholds — machine-specific):
// captured 2026-06-15 on linux/amd64, Intel Core i7-9750H @ 2.60GHz, via
// `go test -bench=BenchmarkCST -benchmem -benchtime=3s -count=2 ./internal/cst/`,
// median of two runs reported:
//
//	BenchmarkCSTBuild/Small:        1.05 ms/op    347 KB/op    1267 allocs/op
//	BenchmarkCSTBuild/Medium:      13.00 ms/op   2504 KB/op   10281 allocs/op
//	BenchmarkCSTBuild/Large:      124.86 ms/op  14036 KB/op   40297 allocs/op
//	BenchmarkCSTSerialize/Small:   14.75 µs/op      6 KB/op       5 allocs/op
//	BenchmarkCSTSerialize/Medium: 125.59 µs/op     49 KB/op       8 allocs/op
//	BenchmarkCSTSerialize/Large:  754.01 µs/op    196 KB/op      10 allocs/op
//
// The snapshot documents shape (super-linear Build, near-flat Serialize) so
// reviewers can sanity-check whether their local numbers look like the same
// algorithm. Compare relative deltas on the same hardware, not absolute
// numbers across machines. The repo's benchmarks/baseline.txt is
// darwin/arm64 only at the time of writing, which is why this snapshot is
// inline here.
//
// Build cost grows super-linearly with block count: the gap-scan in
// build.go iterates all tokens per body via commentsInRange, so the
// asymptote is O(blocks × tokens). This isn't optimized here — the
// baseline is just for tracking. If a downstream rule needs Build on hot
// paths, file a follow-up to index tokens by byte range up-front.

// benchSizes generates HCL inputs of varying block counts representative
// of the file shapes downstream rules see in the wild — small modules,
// mid-size workspaces, and the large hand-rolled Terraform files where
// the existing line-based Fix paths burn the most CPU.
var benchSizes = []struct {
	name   string
	blocks int
}{
	{name: "Small", blocks: 5},
	{name: "Medium", blocks: 50},
	{name: "Large", blocks: 200},
}

// generateBenchmarkHCL produces a deterministic Terraform file with the
// given number of resource blocks plus a terraform block, a variable,
// blank-line separators, and a leading section comment per resource.
// The shape is chosen to exercise every BodyItem variant (Attribute,
// Block, BlankLine, StandaloneComment) so a regression that only shows
// on one shape gets caught at every size.
func generateBenchmarkHCL(blocks int) []byte {
	var b strings.Builder
	b.WriteString(`# Generated benchmark fixture
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

`)
	for i := 0; i < blocks; i++ {
		fmt.Fprintf(&b, `### Resource group %d

# Web server instance for tier %d
resource "aws_instance" "web_%d" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  tags = {
    Name        = "web-server-%d"
    Environment = var.environment
  }

  lifecycle {
    create_before_destroy = true
  }
}

`, i, i, i, i)
	}
	return []byte(b.String())
}

// BenchmarkCSTBuild measures Build (lex + parse + CST construction). The
// hot path is the gap-classification walk per body, so input size scales
// the per-block work rather than just the lex/parse budget. Allocs
// reported here are the floor for any structural rule that calls Build
// even once per file.
func BenchmarkCSTBuild(b *testing.B) {
	for _, sz := range benchSizes {
		content := generateBenchmarkHCL(sz.blocks)
		b.Run(sz.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Build(content, "bench.tf", DefaultTopLevelPolicy())
				if err != nil {
					b.Fatalf("Build: %v", err)
				}
			}
		})
	}
}

// BenchmarkCSTSerialize measures File.Bytes on a fully-unchanged CST —
// the fast path where every BodyItem writes its raw bytes through
// verbatim. This is the floor; once a rule mutates an item (raw = nil)
// the regenerated branches add their own cost, but the unchanged-item
// path dominates production traffic since most files have zero or one
// mutation per Fix call.
//
// Build is hoisted inside b.Run (before b.ResetTimer) so a Build failure
// reports against the inner sub-benchmark and the timed loop measures
// only Bytes.
func BenchmarkCSTSerialize(b *testing.B) {
	for _, sz := range benchSizes {
		content := generateBenchmarkHCL(sz.blocks)
		b.Run(sz.name, func(b *testing.B) {
			f, err := Build(content, "bench.tf", DefaultTopLevelPolicy())
			if err != nil {
				b.Fatalf("Build: %v", err)
			}
			b.SetBytes(int64(len(content)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.Bytes()
			}
		})
	}
}
