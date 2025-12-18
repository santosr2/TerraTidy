// Package main provides the init-rule command for TerraTidy.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initRuleName   string
	initRuleType   string
	initRuleOutput string
)

var initRuleCmd = &cobra.Command{
	Use:   "init-rule",
	Short: "Initialize a new custom rule",
	Long: `Generate scaffolding for a new custom rule.

This command creates template files for implementing custom rules.
Supported rule types:
  - go:     Go-based rule (most powerful, requires compilation)
  - rego:   Rego policy rule (for policy engine)
  - yaml:   YAML-based rule configuration`,
	Example: `  # Create a Go-based rule
  terratidy init-rule --name my-rule --type go

  # Create a Rego policy
  terratidy init-rule --name require-encryption --type rego

  # Create rule in specific directory
  terratidy init-rule --name my-rule --output ./rules`,
	RunE: runInitRule,
}

func init() {
	initRuleCmd.Flags().StringVar(&initRuleName, "name", "", "rule name (required)")
	initRuleCmd.Flags().StringVar(&initRuleType, "type", "rego", "rule type (go|rego|yaml)")
	initRuleCmd.Flags().StringVar(&initRuleOutput, "output", ".", "output directory")
	initRuleCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(initRuleCmd)
}

func runInitRule(cmd *cobra.Command, args []string) error {
	// Validate name
	if initRuleName == "" {
		return fmt.Errorf("rule name is required")
	}

	// Normalize name
	normalizedName := strings.ToLower(strings.ReplaceAll(initRuleName, " ", "-"))

	// Create output directory
	if err := os.MkdirAll(initRuleOutput, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	fmt.Printf("Creating %s rule: %s\n\n", initRuleType, normalizedName)

	switch strings.ToLower(initRuleType) {
	case "go":
		return createGoRule(normalizedName)
	case "rego":
		return createRegoRule(normalizedName)
	case "yaml":
		return createYAMLRule(normalizedName)
	default:
		return fmt.Errorf("unsupported rule type: %s (use go, rego, or yaml)", initRuleType)
	}
}

func createGoRule(name string) error {
	// Create rule directory
	ruleDir := filepath.Join(initRuleOutput, "rules", name)
	if err := os.MkdirAll(ruleDir, 0755); err != nil {
		return fmt.Errorf("creating rule directory: %w", err)
	}

	// Generate Go file
	goContent := fmt.Sprintf(`// Package %s implements the %s custom rule.
package %s

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// Rule implements the %s check.
type Rule struct{}

// Name returns the rule name.
func (r *Rule) Name() string {
	return "%s"
}

// Description returns the rule description.
func (r *Rule) Description() string {
	return "Custom rule: %s"
}

// Check implements the rule logic.
func (r *Rule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	// TODO: Implement your rule logic here
	//
	// Example: Check for a specific pattern
	// body, ok := file.Body.(*hclsyntax.Body)
	// if !ok {
	//     return nil, nil
	// }
	//
	// for _, block := range body.Blocks {
	//     if block.Type == "resource" {
	//         // Your validation logic here
	//     }
	// }

	return findings, nil
}
`, toGoPackageName(name), name, toGoPackageName(name), name, name, name)

	goFile := filepath.Join(ruleDir, "rule.go")
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		return fmt.Errorf("writing rule.go: %w", err)
	}
	fmt.Printf("  Created %s\n", goFile)

	// Generate test file
	testContent := fmt.Sprintf(`package %s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRule_Name(t *testing.T) {
	r := &Rule{}
	assert.Equal(t, "%s", r.Name())
}

func TestRule_Check(t *testing.T) {
	// TODO: Add test cases
	t.Run("passes valid config", func(t *testing.T) {
		// r := &Rule{}
		// findings, err := r.Check(ctx, file)
		// require.NoError(t, err)
		// assert.Empty(t, findings)
	})

	t.Run("catches violation", func(t *testing.T) {
		// r := &Rule{}
		// findings, err := r.Check(ctx, file)
		// require.NoError(t, err)
		// assert.Len(t, findings, 1)
	})
}
`, toGoPackageName(name), name)

	testFile := filepath.Join(ruleDir, "rule_test.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		return fmt.Errorf("writing rule_test.go: %w", err)
	}
	fmt.Printf("  Created %s\n", testFile)

	fmt.Println()
	fmt.Println("Go rule scaffolding created!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Implement the Check() method in rule.go")
	fmt.Println("  2. Add test cases in rule_test.go")
	fmt.Println("  3. Register the rule in your engine")

	return nil
}

func createRegoRule(name string) error {
	// Create policies directory
	policyDir := filepath.Join(initRuleOutput, "policies")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		return fmt.Errorf("creating policies directory: %w", err)
	}

	// Generate Rego file
	regoContent := fmt.Sprintf(`# Policy: %s
# Description: Custom policy for %s
package terraform

# Deny rule - returns violations
deny[msg] {
    # Get resources from input
    resource := input.resources[_]

    # TODO: Add your condition here
    # Example: Check for missing encryption
    # resource.type == "aws_s3_bucket"
    # not resource.encryption

    # This is a placeholder condition that never matches
    false

    msg := {
        "msg": "Violation message for %s",
        "rule": "%s",
        "severity": "error",
        "file": resource._file
    }
}

# Warn rule - returns warnings (optional)
warn[msg] {
    # Add warning conditions here
    false

    msg := {
        "msg": "Warning message for %s",
        "rule": "%s",
        "severity": "warning"
    }
}
`, name, name, name, name, name, name)

	regoFile := filepath.Join(policyDir, name+".rego")
	if err := os.WriteFile(regoFile, []byte(regoContent), 0644); err != nil {
		return fmt.Errorf("writing %s.rego: %w", name, err)
	}
	fmt.Printf("  Created %s\n", regoFile)

	// Generate test file
	testContent := fmt.Sprintf(`# Tests for %s policy
package terraform

# Test data - valid configuration
test_valid_config {
    # TODO: Add test for valid configuration
    # count(deny) == 0 with input as {...}
    true
}

# Test data - invalid configuration
test_invalid_config {
    # TODO: Add test for invalid configuration
    # count(deny) > 0 with input as {...}
    true
}
`, name)

	testFile := filepath.Join(policyDir, name+"_test.rego")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		return fmt.Errorf("writing %s_test.rego: %w", name, err)
	}
	fmt.Printf("  Created %s\n", testFile)

	fmt.Println()
	fmt.Println("Rego policy scaffolding created!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s to add your policy logic\n", regoFile)
	fmt.Printf("  2. Add test cases in %s\n", testFile)
	fmt.Println("  3. Run: terratidy policy --policy-dir ./policies")

	return nil
}

func createYAMLRule(name string) error {
	// Create rules directory
	rulesDir := filepath.Join(initRuleOutput, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("creating rules directory: %w", err)
	}

	// Generate YAML file
	yamlContent := fmt.Sprintf(`# Custom Rule Configuration: %s
# This file configures a custom rule for TerraTidy

name: %s
description: "Custom rule: %s"
severity: warning  # info, warning, or error

# Pattern-based matching
patterns:
  # Match specific resource types
  resource_types:
    # - aws_s3_bucket
    # - aws_instance

  # Required attributes
  required_attributes:
    # - encryption
    # - tags

# Message shown when rule is violated
message: "Resource violates %s rule"

# Examples
examples:
  # Good example (passes the rule)
  good: |
    resource "aws_s3_bucket" "example" {
      bucket = "my-bucket"
      # This passes because it has required attributes
    }

  # Bad example (fails the rule)
  bad: |
    resource "aws_s3_bucket" "example" {
      bucket = "my-bucket"
      # This fails because it's missing required attributes
    }

# Tags for categorization
tags:
  - custom
  # - security
  # - compliance
`, name, name, name, name)

	yamlFile := filepath.Join(rulesDir, name+".yaml")
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("writing %s.yaml: %w", name, err)
	}
	fmt.Printf("  Created %s\n", yamlFile)

	fmt.Println()
	fmt.Println("YAML rule configuration created!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Edit %s to configure your rule\n", yamlFile)
	fmt.Println("  2. Add the rule to your .terratidy.yaml:")
	fmt.Printf("     custom_rules:\n")
	fmt.Printf("       %s:\n", name)
	fmt.Printf("         enabled: true\n")

	return nil
}

// toGoPackageName converts a rule name to a valid Go package name.
func toGoPackageName(name string) string {
	// Replace hyphens with underscores
	name = strings.ReplaceAll(name, "-", "_")
	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result.WriteRune(c)
		}
	}
	return result.String()
}
