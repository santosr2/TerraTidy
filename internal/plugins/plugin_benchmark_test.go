package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadYAMLRule(b *testing.B) {
	tmpDir := b.TempDir()

	ruleContent := `name: test-rule
description: A test rule for benchmarking
severity: warning
enabled: true
message: "Test finding"
patterns:
  required_attributes:
    - description
`
	rulePath := filepath.Join(tmpDir, "test-rule.yaml")
	if err := os.WriteFile(rulePath, []byte(ruleContent), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loadYAMLRule(rulePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadYAMLRuleComplex(b *testing.B) {
	tmpDir := b.TempDir()

	ruleContent := `name: complex-rule
description: A complex rule with all pattern types
severity: warning
enabled: true
message: "Complex finding"
tags:
  - security
  - compliance
  - best-practice
patterns:
  block_types:
    - resource
    - data
  resource_types:
    - aws_s3_bucket
    - aws_instance
    - aws_iam_role
  required_attributes:
    - description
    - tags
  forbidden_attributes:
    - deprecated_field
    - legacy_option
  attribute_patterns:
    - attribute: name
      pattern: "^[a-z][a-z0-9-]+$"
      message: "Name must be lowercase with hyphens"
    - attribute: bucket
      pattern: "^[a-z0-9][a-z0-9.-]+$"
      message: "Bucket name must follow S3 naming rules"
`
	rulePath := filepath.Join(tmpDir, "complex-rule.yaml")
	if err := os.WriteFile(rulePath, []byte(ruleContent), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loadYAMLRule(rulePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginManagerLoadDirectory(b *testing.B) {
	tmpDir := b.TempDir()

	// Create multiple YAML rules
	for i := 0; i < 5; i++ {
		ruleContent := `name: rule-` + string(rune('a'+i)) + `
description: Test rule
severity: warning
enabled: true
message: "Finding"
patterns:
  required_attributes:
    - description
`
		rulePath := filepath.Join(tmpDir, "rule-"+string(rune('a'+i))+".yaml")
		if err := os.WriteFile(rulePath, []byte(ruleContent), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := NewManager([]string{tmpDir}, false)
		err := manager.LoadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginManagerLoadMultipleDirectories(b *testing.B) {
	// Create multiple directories with rules
	var dirs []string
	for d := 0; d < 3; d++ {
		tmpDir := b.TempDir()
		dirs = append(dirs, tmpDir)

		for i := 0; i < 3; i++ {
			ruleContent := `name: dir` + string(rune('0'+d)) + `-rule-` + string(rune('a'+i)) + `
description: Test rule
severity: warning
enabled: true
message: "Finding"
patterns:
  required_attributes:
    - description
`
			rulePath := filepath.Join(tmpDir, "rule-"+string(rune('a'+i))+".yaml")
			if err := os.WriteFile(rulePath, []byte(ruleContent), 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := NewManager(dirs, false)
		err := manager.LoadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}
