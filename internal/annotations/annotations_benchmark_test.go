package annotations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// generateHCLWithAnnotations creates HCL content with a specified number of resources
// and annotations per resource.
func generateHCLWithAnnotations(resources, annotationsPerResource int) []byte {
	var sb strings.Builder

	// Add a file-level suppression
	sb.WriteString("# terratidy:ignore-file:style.variable-name-convention\n\n")

	for i := range resources {
		// Add next-block annotations
		for j := range annotationsPerResource / 2 {
			fmt.Fprintf(&sb, "# terratidy:ignore:style.rule-%d\n", j)
		}

		// Add a resource with inline annotation
		if annotationsPerResource > 0 {
			fmt.Fprintf(&sb, `resource "aws_instance" "server_%d" { # terratidy:ignore:style.resource-name-convention
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}

`, i)
		} else {
			fmt.Fprintf(&sb, `resource "aws_instance" "server_%d" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}

`, i)
		}
	}

	return []byte(sb.String())
}

// generateFindings creates a slice of findings for benchmarking.
func generateFindings(n int) []sdk.Finding {
	findings := make([]sdk.Finding, n)
	for i := range n {
		sev := sdk.SeverityWarning
		switch i % 3 {
		case 0:
			sev = sdk.SeverityError
		case 2:
			sev = sdk.SeverityInfo
		}
		findings[i] = sdk.Finding{
			Rule:     fmt.Sprintf("style.rule-%d", i%10),
			Message:  fmt.Sprintf("Issue %d in resource block", i),
			File:     fmt.Sprintf("modules/service-%d/main.tf", i%5),
			Severity: sev,
			Location: sdk.Location{
				StartLine:   i + 1,
				StartColumn: 1,
				EndLine:     i + 1,
				EndColumn:   40,
			},
		}
	}
	return findings
}

// generateSuppressions creates suppressions for benchmarking.
func generateSuppressions(n int) []Suppression {
	suppressions := make([]Suppression, n)
	for i := range n {
		typ := Type(i % 3) // Cycle through NextBlock, Inline, File
		suppressions[i] = Suppression{
			Rule:       fmt.Sprintf("style.rule-%d", i%10),
			Line:       i + 1,
			TargetLine: i + 2,
			Type:       typ,
		}
	}
	return suppressions
}

func BenchmarkAnnotationParse(b *testing.B) {
	tests := []struct {
		name                   string
		resources              int
		annotationsPerResource int
	}{
		{"Small_10Resources_FileAnnotationOnly", 10, 0},
		{"Small_10Resources_2Annotations", 10, 2},
		{"Medium_50Resources_4Annotations", 50, 4},
		{"Large_100Resources_6Annotations", 100, 6},
		{"XLarge_500Resources_2Annotations", 500, 2},
	}

	for _, tc := range tests {
		content := generateHCLWithAnnotations(tc.resources, tc.annotationsPerResource)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for range b.N {
				suppressions := Parse(content)
				_ = suppressions
			}
		})
	}
}

func BenchmarkFilterFindings(b *testing.B) {
	tests := []struct {
		name         string
		findings     int
		suppressions int
	}{
		{"100Findings_10Suppressions", 100, 10},
		{"100Findings_50Suppressions", 100, 50},
		{"1000Findings_10Suppressions", 1000, 10},
		{"1000Findings_100Suppressions", 1000, 100},
		{"5000Findings_50Suppressions", 5000, 50},
	}

	for _, tc := range tests {
		findings := generateFindings(tc.findings)
		suppressions := generateSuppressions(tc.suppressions)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				filtered := FilterFindings(findings, suppressions)
				_ = filtered
			}
		})
	}
}

func BenchmarkIsSuppressed(b *testing.B) {
	finding := sdk.Finding{
		Rule:     "style.resource-name-convention",
		Location: sdk.Location{StartLine: 5},
	}

	tests := []struct {
		name         string
		suppressions []Suppression
	}{
		{
			"NoMatch_5Suppressions",
			[]Suppression{
				{Rule: "style.other-rule", TargetLine: 5, Type: NextBlock},
				{Rule: "lint.some-rule", TargetLine: 5, Type: NextBlock},
				{Rule: "policy.require-tags", TargetLine: 5, Type: NextBlock},
				{Rule: "style.variable-name-convention", TargetLine: 10, Type: NextBlock},
				{Rule: "style.resource-name-convention", TargetLine: 10, Type: NextBlock}, // Wrong line
			},
		},
		{
			"Match_FileLevel",
			[]Suppression{
				{Rule: "style.resource-name-convention", Type: File},
			},
		},
		{
			"Match_Wildcard",
			[]Suppression{
				{Rule: "style.*", Type: File},
			},
		},
		{
			"Match_LineSpecific",
			[]Suppression{
				{Rule: "style.resource-name-convention", TargetLine: 5, Type: NextBlock},
			},
		},
		{
			"NoMatch_LinearScan20",
			generateSuppressions(20),
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := IsSuppressed(finding, tc.suppressions)
				_ = result
			}
		})
	}
}

func BenchmarkRuleMatches(b *testing.B) {
	tests := []struct {
		name            string
		findingRule     string
		suppressionRule string
	}{
		{"ExactMatch", "style.resource-name-convention", "style.resource-name-convention"},
		{"NoMatch", "style.resource-name-convention", "style.variable-name-convention"},
		{"WildcardMatch", "style.resource-name-convention", "style.*"},
		{"WildcardNoMatch", "lint.some-rule", "style.*"},
		{"LongRuleName", "style.very-long-rule-name-for-testing", "style.very-long-rule-name-for-testing"},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result := RuleMatches(tc.findingRule, tc.suppressionRule)
				_ = result
			}
		})
	}
}

// BenchmarkAnnotationParseAndFilter combines parsing and filtering in a realistic workflow.
func BenchmarkAnnotationParseAndFilter(b *testing.B) {
	content := generateHCLWithAnnotations(100, 4)
	findings := generateFindings(500)

	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()

	for range b.N {
		suppressions := Parse(content)
		filtered := FilterFindings(findings, suppressions)
		_ = filtered
	}
}

// BenchmarkAnnotationParse_RealWorld simulates real-world HCL files with mixed content.
func BenchmarkAnnotationParse_RealWorld(b *testing.B) {
	content := []byte(`# terratidy:ignore-file:policy.require-tags
# This is a typical Terraform file with various resources

# terratidy:ignore:style.resource-name-convention
resource "aws_vpc" "Main" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "main-vpc"
  }
}

# terratidy:ignore:style.variable-name-convention
# terratidy:ignore:lint.deprecated-resource
resource "aws_instance" "WebServer" { # terratidy:ignore:style.resource-name-convention
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  vpc_id        = aws_vpc.Main.id

  tags = {
    Name = "web-server"
  }
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

output "vpc_id" { # terratidy:ignore:style.output-name-convention
  value       = aws_vpc.Main.id
  description = "The VPC ID"
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# terratidy:ignore:lint.unused-variable
variable "unused_var" {
  description = "This is intentionally unused"
  type        = string
  default     = ""
}

module "database" {
  source = "./modules/database"

  vpc_id      = aws_vpc.Main.id
  environment = var.environment
}
`)

	b.ReportAllocs()
	b.SetBytes(int64(len(content)))

	b.ResetTimer()
	for range b.N {
		suppressions := Parse(content)
		_ = suppressions
	}
}
