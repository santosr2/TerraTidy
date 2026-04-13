package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func FuzzStyleCheck(f *testing.F) {
	// Use benchmark configs as seeds (comprehensive and realistic)
	f.Add([]byte(mediumConfig))  // provider, count, lifecycle, tags
	f.Add([]byte(complexConfig)) // for_each, depends_on, multiple providers

	// Additional basic seeds
	f.Add([]byte(`resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  tags = {
    Name = "test"
  }
}
`))
	f.Add([]byte(`variable "name" {
  type    = string
  default = "hello"
}
`))
	f.Add([]byte(`output "id" {
  value       = aws_s3_bucket.example.id
  description = "The bucket ID"
}
`))
	f.Add([]byte(`data "aws_caller_identity" "current" {}
`))

	// Edge cases (invalid/malformed HCL)
	f.Add([]byte(``))              // Empty
	f.Add([]byte(`{`))             // Invalid HCL
	f.Add([]byte(`}`))             // Invalid HCL
	f.Add([]byte(`= value`))       // Invalid HCL
	f.Add([]byte(`resource {}`))   // Missing labels
	f.Add([]byte(`"just string"`)) // Just a string

	// Deep nesting
	f.Add([]byte(`resource "test" "nested" {
  block1 {
    block2 {
      block3 {
        attr = "deep"
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

	// Comments
	f.Add([]byte(`# Comment
resource "test" "example" {} // inline
/* block */
`))

	// Heredoc
	f.Add([]byte(`resource "test" "heredoc" {
  content = <<-EOF
    line1
    line2
  EOF
}
`))

	// Complex expressions
	f.Add([]byte(`locals {
  complex = merge(
    { for k, v in var.map : k => v if v != null },
    try(var.other, {})
  )
}
`))

	// Create engine once outside fuzz function for performance
	engine := New(&Config{Rules: make(map[string]RuleConfig)})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Write to temp file (style engine works on files)
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Run style checks - should not panic regardless of input
		_, _ = engine.Run(context.Background(), []string{tmpFile})
	})
}

// FuzzStyleFix exercises the fix-mode code path which involves:
// - Multi-pass loop (up to 3 passes)
// - File write-back via applyFixes
// - Re-read and re-parse after each fix
// - Diff generation
func FuzzStyleFix(f *testing.F) {
	// Use same seeds as FuzzStyleCheck
	f.Add([]byte(mediumConfig))
	f.Add([]byte(complexConfig))

	// Seeds with known style issues that trigger fixes
	f.Add([]byte(`resource "aws_instance" "web" {
tags = {
Name = "test"
}
ami = "ami-123"
}
`)) // tags not at end

	f.Add([]byte(`resource "aws_instance" "web" {
ami = "ami-123"


instance_type = "t2.micro"
}
`)) // consecutive blank lines

	f.Add([]byte(`


resource "test" "example" {}


`)) // leading/trailing blank lines

	f.Add([]byte(`variable "a" {}
variable "b" {}
`)) // no blank line between blocks

	// Edge cases
	f.Add([]byte(``))
	f.Add([]byte(`{`))
	f.Add([]byte(`resource {}`))

	// Create engine with Fix and Diff enabled
	engine := New(&Config{
		Fix:   true,
		Diff:  true,
		Rules: make(map[string]RuleConfig),
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Write to temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.tf")
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Run style checks with fix mode - should not panic
		_, _ = engine.Run(context.Background(), []string{tmpFile})
	})
}
