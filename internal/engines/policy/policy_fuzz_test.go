package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// FuzzPolicyEval exercises OPA/Rego policy evaluation with arbitrary input.
// Tests both the Rego policy parsing and HCL input parsing paths.
func FuzzPolicyEval(f *testing.F) {
	// Valid Rego policy seeds
	validPolicy := `package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not resource.tags
    msg := {
        "msg": sprintf("EC2 instance %s is missing tags", [resource.name]),
        "rule": "require-tags",
        "severity": "warning"
    }
}
`

	simplePolicy := `package terraform

import rego.v1

deny contains msg if {
    input.foo == "bar"
    msg := {"msg": "found foo", "rule": "test"}
}
`

	warnPolicy := `package terraform

import rego.v1

warn contains msg if {
    some resource in input.resources
    msg := {"msg": "warning", "rule": "warn-test", "severity": "info"}
}
`

	// Valid HCL seeds
	validHCL := `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`

	hclWithTags := `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "web"
  }
}
`

	// Add seed combinations (policy, hcl)
	f.Add([]byte(validPolicy), []byte(validHCL))
	f.Add([]byte(validPolicy), []byte(hclWithTags))
	f.Add([]byte(simplePolicy), []byte(validHCL))
	f.Add([]byte(warnPolicy), []byte(validHCL))

	// Edge case policies
	f.Add([]byte(``), []byte(validHCL))                  // Empty policy
	f.Add([]byte(`package terraform`), []byte(validHCL)) // Minimal policy
	f.Add([]byte(`{`), []byte(validHCL))                 // Invalid Rego
	f.Add([]byte(`package`), []byte(validHCL))           // Incomplete Rego
	f.Add([]byte(`package 日本語`), []byte(validHCL))       // Unicode in policy
	f.Add([]byte("package terraform\n# comment"), []byte(validHCL))

	// Edge case HCL
	f.Add([]byte(validPolicy), []byte(``))              // Empty HCL
	f.Add([]byte(validPolicy), []byte(`{`))             // Invalid HCL
	f.Add([]byte(validPolicy), []byte(`resource {}`))   // Missing labels
	f.Add([]byte(validPolicy), []byte(`"just string"`)) // Just a string

	// Both invalid
	f.Add([]byte(`{`), []byte(`{`))
	f.Add([]byte(``), []byte(``))

	// Complex HCL
	f.Add([]byte(validPolicy), []byte(`terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

variable "count" {
  type    = number
  default = 3
}

resource "aws_instance" "web" {
  count         = var.count
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "web-${count.index}"
  }

  lifecycle {
    create_before_destroy = true
  }
}
`))

	// Policy with complex Rego
	f.Add([]byte(`package terraform

import rego.v1

# Multiple rules with different outputs
deny contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not resource.tags
    msg := {"msg": "no tags", "rule": "tags", "severity": "error"}
}

warn contains msg if {
    some resource in input.resources
    resource.type == "aws_s3_bucket"
    msg := {"msg": "bucket found", "rule": "bucket", "severity": "warning"}
}

# Helper rule
has_tag(resource, key) if {
    resource.tags[key]
}
`), []byte(validHCL))

	f.Fuzz(func(t *testing.T, policyData, hclData []byte) {
		tmpDir := t.TempDir()

		// Write policy file
		policyFile := filepath.Join(tmpDir, "policy.rego")
		if err := os.WriteFile(policyFile, policyData, 0o600); err != nil {
			t.Fatalf("failed to write policy file: %v", err)
		}

		// Write HCL file
		hclFile := filepath.Join(tmpDir, "main.tf")
		if err := os.WriteFile(hclFile, hclData, 0o600); err != nil {
			t.Fatalf("failed to write HCL file: %v", err)
		}

		// Create engine with the policy file
		engine := New(&Config{
			PolicyFiles: []string{policyFile},
			Rules:       make(map[string]RuleConfig),
		})

		// Run policy evaluation - should not panic regardless of input
		_, _ = engine.Run(context.Background(), []string{hclFile})
	})
}
