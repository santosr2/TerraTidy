package config

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadConfig(b *testing.B) {
	// Create a temporary config file
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	configContent := `version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
  policy:
    enabled: false

severity_threshold: warning
fail_fast: false
parallel: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadConfigWithProfiles(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	configContent := `version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true

profiles:
  ci:
    description: "CI checks"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true

  development:
    description: "Fast dev checks"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: false
      policy:
        enabled: false

  production:
    description: "Production checks"
    inherits: ci
    engines:
      policy:
        enabled: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadConfigWithImports(b *testing.B) {
	tmpDir := b.TempDir()

	// Create imported config
	importDir := filepath.Join(tmpDir, ".terratidy")
	if err := os.MkdirAll(importDir, 0o755); err != nil {
		b.Fatal(err)
	}

	importContent := `overrides:
  rules:
    style.blank-lines:
      enabled: true
      severity: warning
    style.block-label-case:
      enabled: true
      severity: error
`
	importPath := filepath.Join(importDir, "style-rules.yaml")
	if err := os.WriteFile(importPath, []byte(importContent), 0o644); err != nil {
		b.Fatal(err)
	}

	// Create main config with import
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")
	configContent := `version: 1

imports:
  - .terratidy/style-rules.yaml

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplyProfile(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, ".terratidy.yaml")

	configContent := `version: 1

engines:
  fmt:
    enabled: true
  style:
    enabled: true
  lint:
    enabled: true

profiles:
  strict:
    description: "Strict mode"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		b.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.ApplyProfile("strict")
	}
}
