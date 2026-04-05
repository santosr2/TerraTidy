package lint

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Run(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		config      *Config
		wantErr     bool
		wantFinding bool
	}{
		{
			name: "valid terraform file with version",
			content: `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			config:      nil,
			wantErr:     false,
			wantFinding: false,
		},
		{
			name: "missing required_version",
			content: `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`,
			config:      nil,
			wantErr:     false,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			engine := New(tt.config)
			findings, err := engine.Run(context.Background(), []string{tmpFile})

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantFinding {
				assert.NotEmpty(t, findings, "expected to find issues")
			} else {
				assert.Empty(t, findings, "expected no issues")
			}
		})
	}
}

func TestTerraformRequiredVersionRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "has required_version",
			content: `terraform {
  required_version = ">= 1.0"
}
`,
			wantFinding: false,
		},
		{
			name: "missing required_version",
			content: `terraform {
}
`,
			wantFinding: true,
		},
		{
			name: "no terraform block",
			content: `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-required-version" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformDocumentedVariablesRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "variable with description",
			content: `variable "instance_type" {
  description = "The instance type to use"
  type        = string
  default     = "t2.micro"
}
`,
			wantFinding: false,
		},
		{
			name: "variable without description",
			content: `variable "instance_type" {
  type    = string
  default = "t2.micro"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "variables.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-documented-variables" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformTypedVariablesRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "variable with type",
			content: `variable "instance_type" {
  description = "The instance type"
  type        = string
}
`,
			wantFinding: false,
		},
		{
			name: "variable without type",
			content: `variable "instance_type" {
  description = "The instance type"
  default     = "t2.micro"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "variables.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-typed-variables" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformNamingConventionRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "valid snake_case",
			content: `resource "aws_instance" "my_instance" {
  ami = "ami-12345"
}
`,
			wantFinding: false,
		},
		{
			name: "invalid camelCase",
			content: `resource "aws_instance" "myInstance" {
  ami = "ami-12345"
}
`,
			wantFinding: true,
		},
		{
			name: "invalid PascalCase",
			content: `resource "aws_instance" "MyInstance" {
  ami = "ami-12345"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-naming-convention" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformModulePinnedSourceRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "module with version",
			content: `module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}
`,
			wantFinding: false,
		},
		{
			name: "registry module without version",
			content: `module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`,
			wantFinding: true,
		},
		{
			name: "local module without version (allowed)",
			content: `module "local" {
  source = "./modules/vpc"
}
`,
			wantFinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-module-pinned-source" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestTerraformDeprecatedSyntaxRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "no deprecated syntax",
			content: `resource "aws_instance" "example" {
  ami           = var.ami_id
  instance_type = local.instance_type
}
`,
			wantFinding: false,
		},
		{
			name: "deprecated interpolation syntax",
			content: `resource "aws_instance" "example" {
  ami           = "${var.ami_id}"
  instance_type = "t2.micro"
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "main.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-deprecated-syntax" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestEngine_GetAllRules(t *testing.T) {
	engine := New(nil)
	rules := engine.GetAllRules()

	// Verify we have all 11 rules registered
	assert.Len(t, rules, 11, "should have 11 rules registered")

	// Verify each rule has required methods
	for _, rule := range rules {
		assert.NotEmpty(t, rule.Name(), "rule name should not be empty")
		assert.NotEmpty(t, rule.Description(), "rule description should not be empty")
	}
}

func TestEngine_RuleDisabling(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	// Test with required_version rule disabled
	engine := New(&Config{
		Rules: map[string]RuleConfig{
			"lint.terraform-required-version": {
				Enabled: false,
			},
		},
	})

	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	// Should not find required_version findings since it's disabled
	for _, f := range findings {
		assert.NotEqual(t, "lint.terraform-required-version", f.Rule)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "example" {
  ami = "ami-12345"
}
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := engine.Run(ctx, []string{tmpFile})
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGroupFilesByDirectory(t *testing.T) {
	engine := New(nil)

	files := []string{
		filepath.Join("project", "modules", "vpc", "main.tf"),
		filepath.Join("project", "modules", "vpc", "variables.tf"),
		filepath.Join("project", "modules", "ec2", "main.tf"),
		filepath.Join("project", "environments", "dev", "main.tf"),
	}

	result := engine.groupFilesByDirectory(files)

	assert.Len(t, result, 3, "should have 3 directories")
	assert.Len(t, result[filepath.Join("project", "modules", "vpc")], 2)
	assert.Len(t, result[filepath.Join("project", "modules", "ec2")], 1)
	assert.Len(t, result[filepath.Join("project", "environments", "dev")], 1)
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"error", "error"},
		{"warning", "warning"},
		{"info", "info"},
		{"ERROR", "error"},
		{"WARNING", "warning"},
		{"unknown", "warning"}, // defaults to warning
		{"", "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSeverity(tt.input)
			assert.Equal(t, tt.want, string(result))
		})
	}
}

func TestTerraformResourceCountRule(t *testing.T) {
	// Create content with many resources
	content := ""
	for i := 0; i < 20; i++ {
		content += `resource "aws_instance" "instance_` + string(rune('a'+i)) + `" {
  ami = "ami-12345"
}

`
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{tmpFile})
	require.NoError(t, err)

	found := false
	for _, f := range findings {
		if f.Rule == "lint.terraform-resource-count" {
			found = true
			break
		}
	}

	assert.True(t, found, "should find resource count warning")
}

func TestIsSimpleReference(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"var.name", true},
		{"local.value", true},
		{"data.aws_ami.latest", true},
		{"module.vpc.id", true},
		{"each.key", true},
		{"count.index", true},
		{"aws_instance.example.id", false},
		{"hello", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isSimpleReference(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTerraformDocumentedOutputsRule(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFinding bool
	}{
		{
			name: "output with description",
			content: `output "instance_id" {
  description = "The instance ID"
  value       = aws_instance.example.id
}
`,
			wantFinding: false,
		},
		{
			name: "output without description",
			content: `output "instance_id" {
  value = aws_instance.example.id
}
`,
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "outputs.tf")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			engine := New(nil)
			findings, err := engine.Run(context.Background(), []string{tmpFile})
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Rule == "lint.terraform-documented-outputs" {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFinding, found, "findings: %+v", findings)
		})
	}
}

func TestEngine_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// main.tf with terraform block
	mainContent := `terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

resource "aws_instance" "example" {
  ami           = var.ami_id
  instance_type = var.instance_type
}
`

	// variables.tf
	varsContent := `variable "ami_id" {
  description = "The AMI ID"
  type        = string
}

variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "t2.micro"
}
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(varsContent), 0o644))

	engine := New(nil)
	findings, err := engine.Run(context.Background(), []string{
		filepath.Join(tmpDir, "main.tf"),
		filepath.Join(tmpDir, "variables.tf"),
	})
	require.NoError(t, err)

	// Should have no major findings (everything is well-documented)
	for _, f := range findings {
		t.Logf("Finding: %s - %s", f.Rule, f.Message)
	}
}

func TestEngine_ValidateTFLintPath(t *testing.T) {
	t.Run("default path not in PATH", func(t *testing.T) {
		// Use a path that definitely doesn't exist
		engine := New(&Config{
			UseTFLint: true,
			// TFLintPath empty means use "tflint" from PATH
		})
		// IsTFLintAvailable will return false if tflint not in PATH
		// This test verifies the validation doesn't crash
		_ = engine.IsTFLintAvailable()
	})

	t.Run("custom path does not exist", func(t *testing.T) {
		engine := New(&Config{
			UseTFLint:  true,
			TFLintPath: "/nonexistent/path/to/tflint",
		})

		err := engine.validateTFLintPath()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Contains(t, err.Error(), "/nonexistent/path/to/tflint")
	})

	t.Run("custom path is directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		engine := New(&Config{
			UseTFLint:  true,
			TFLintPath: tmpDir,
		})

		err := engine.validateTFLintPath()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is a directory")
	})

	t.Run("custom path not executable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("executable permission check not applicable on Windows")
		}

		tmpDir := t.TempDir()
		fakeBinary := filepath.Join(tmpDir, "tflint")
		require.NoError(t, os.WriteFile(fakeBinary, []byte("#!/bin/sh\necho test"), 0o644))

		engine := New(&Config{
			UseTFLint:  true,
			TFLintPath: fakeBinary,
		})

		err := engine.validateTFLintPath()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not executable")
	})

	t.Run("custom path is valid executable", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeBinary := filepath.Join(tmpDir, "tflint")
		require.NoError(t, os.WriteFile(fakeBinary, []byte("#!/bin/sh\necho test"), 0o755))

		engine := New(&Config{
			UseTFLint:  true,
			TFLintPath: fakeBinary,
		})

		err := engine.validateTFLintPath()
		require.NoError(t, err)
		assert.True(t, engine.IsTFLintAvailable())
	})
}

func TestTerraformUnusedDeclarationsRule(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("detects unused variable", func(t *testing.T) {
		// Create file with unused variable
		content := `
variable "used_var" {
  type = string
}

variable "unused_var" {
  type = string
}

resource "test" "example" {
  name = var.used_var
}
`
		tmpFile := filepath.Join(tmpDir, "unused.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		engine := New(&Config{
			UseTFLint: false,
			Rules: map[string]RuleConfig{
				"lint.terraform-unused-declarations": {Enabled: true},
			},
		})

		findings, err := engine.Run(context.Background(), []string{tmpFile})
		require.NoError(t, err)

		// Should find the unused variable
		var foundUnused bool
		for _, f := range findings {
			if f.Rule == "lint.terraform-unused-declarations" && strings.Contains(f.Message, "unused_var") {
				foundUnused = true
			}
		}
		assert.True(t, foundUnused, "should detect unused_var as unused")
	})

	t.Run("no false positive for used variable", func(t *testing.T) {
		content := `
variable "my_var" {
  type = string
}

resource "test" "example" {
  name = var.my_var
}
`
		tmpFile := filepath.Join(tmpDir, "used.tf")
		require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

		engine := New(&Config{
			UseTFLint: false,
			Rules: map[string]RuleConfig{
				"lint.terraform-unused-declarations": {Enabled: true},
			},
		})

		findings, err := engine.Run(context.Background(), []string{tmpFile})
		require.NoError(t, err)

		// Should not report my_var as unused
		for _, f := range findings {
			if f.Rule == "lint.terraform-unused-declarations" {
				assert.NotContains(t, f.Message, "my_var", "should not report used variable")
			}
		}
	})

	t.Run("handles multiple files correctly", func(t *testing.T) {
		// Variable declared in one file, used in another
		varFile := filepath.Join(tmpDir, "variables.tf")
		mainFile := filepath.Join(tmpDir, "main.tf")

		varContent := `
variable "cross_file_var" {
  type = string
}
`
		mainContent := `
resource "test" "example" {
  name = var.cross_file_var
}
`
		require.NoError(t, os.WriteFile(varFile, []byte(varContent), 0o644))
		require.NoError(t, os.WriteFile(mainFile, []byte(mainContent), 0o644))

		engine := New(&Config{
			UseTFLint: false,
			Rules: map[string]RuleConfig{
				"lint.terraform-unused-declarations": {Enabled: true},
			},
		})

		findings, err := engine.Run(context.Background(), []string{varFile, mainFile})
		require.NoError(t, err)

		// Should not report cross_file_var as unused
		for _, f := range findings {
			if f.Rule == "lint.terraform-unused-declarations" {
				assert.NotContains(t, f.Message, "cross_file_var", "should find usage across files")
			}
		}
	})
}

func TestEngine_Run_TFLintPathError(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(`resource "test" "x" {}`), 0o644))

	t.Run("invalid TFLintPath with no fallback", func(t *testing.T) {
		engine := New(&Config{
			UseTFLint:       true,
			TFLintPath:      "/nonexistent/tflint",
			FallbackBuiltin: false,
		})

		_, err := engine.Run(context.Background(), []string{tmpFile})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TFLint integration enabled")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid TFLintPath with fallback", func(t *testing.T) {
		engine := New(&Config{
			UseTFLint:       true,
			TFLintPath:      "/nonexistent/tflint",
			FallbackBuiltin: true,
		})

		// Should not error - falls back to built-in rules
		_, err := engine.Run(context.Background(), []string{tmpFile})
		require.NoError(t, err)
	})
}

func TestConfigFromEngine(t *testing.T) {
	t.Run("empty config uses defaults", func(t *testing.T) {
		engineCfg := config.LintEngineConfig{}
		cfg := ConfigFromEngine(engineCfg)

		require.NotNil(t, cfg)
		assert.Equal(t, ".tflint.hcl", cfg.ConfigFile)
		assert.Empty(t, cfg.Plugins)
		assert.Empty(t, cfg.Args)
		assert.False(t, cfg.UseTFLint)
		assert.Empty(t, cfg.TFLintPath)
		assert.False(t, cfg.FallbackBuiltin)
		assert.Empty(t, cfg.Rules)
	})

	t.Run("all fields copied", func(t *testing.T) {
		engineCfg := config.LintEngineConfig{
			ConfigFile:      "/custom/.tflint.hcl",
			Plugins:         []string{"aws", "google"},
			Args:            []string{"--force", "--no-color"},
			UseTFLint:       true,
			TFLintPath:      "/usr/local/bin/tflint",
			FallbackBuiltin: true,
		}
		cfg := ConfigFromEngine(engineCfg)

		assert.Equal(t, "/custom/.tflint.hcl", cfg.ConfigFile)
		assert.Equal(t, []string{"aws", "google"}, cfg.Plugins)
		assert.Equal(t, []string{"--force", "--no-color"}, cfg.Args)
		assert.True(t, cfg.UseTFLint)
		assert.Equal(t, "/usr/local/bin/tflint", cfg.TFLintPath)
		assert.True(t, cfg.FallbackBuiltin)
	})

	t.Run("with rules", func(t *testing.T) {
		engineCfg := config.LintEngineConfig{
			Rules: map[string]config.RuleConfig{
				"terraform-required-version": {
					Enabled:  true,
					Severity: "error",
				},
				"terraform-required-providers": {
					Enabled:  false,
					Severity: "warning",
					Config:   map[string]any{"source_required": true},
				},
			},
		}
		cfg := ConfigFromEngine(engineCfg)

		require.Len(t, cfg.Rules, 2)

		versionRule := cfg.Rules["terraform-required-version"]
		assert.True(t, versionRule.Enabled)
		assert.Equal(t, "error", versionRule.Severity)

		providersRule := cfg.Rules["terraform-required-providers"]
		assert.False(t, providersRule.Enabled)
		assert.Equal(t, "warning", providersRule.Severity)
		assert.Equal(t, true, providersRule.Options["source_required"])
	})

	t.Run("nil rules map", func(t *testing.T) {
		engineCfg := config.LintEngineConfig{
			UseTFLint: true,
			Rules:     nil,
		}
		cfg := ConfigFromEngine(engineCfg)

		require.NotNil(t, cfg.Rules)
		assert.Empty(t, cfg.Rules)
	})
}

func TestTerraformRequiredProvidersRule(t *testing.T) {
	rule := &TerraformRequiredProvidersRule{}

	t.Run("name and description", func(t *testing.T) {
		assert.Equal(t, "lint.terraform-required-providers", rule.Name())
		assert.Contains(t, rule.Description(), "required_providers")
	})

	t.Run("no finding when required_providers exists", func(t *testing.T) {
		content := `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "versions.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("finding when required_providers missing in versions.tf", func(t *testing.T) {
		content := `
terraform {
  required_version = ">= 1.0"
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "versions.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.Len(t, findings, 1)
		assert.Contains(t, findings[0].Message, "Missing required_providers")
		assert.Equal(t, sdk.SeverityInfo, findings[0].Severity)
	})

	t.Run("no finding for non-main/versions files", func(t *testing.T) {
		content := `
resource "aws_instance" "test" {
  ami = "ami-123"
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "instance.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Empty(t, findings, "should not report for non-main/versions files")
	})
}

func TestTerraformHardcodedSecretsRule(t *testing.T) {
	rule := &TerraformHardcodedSecretsRule{}

	t.Run("name and description", func(t *testing.T) {
		assert.Equal(t, "lint.terraform-hardcoded-secrets", rule.Name())
		assert.Contains(t, rule.Description(), "secrets")
	})

	t.Run("detects AWS access key", func(t *testing.T) {
		content := `
variable "aws_access_key" {
  default = "AKIAIOSFODNN7EXAMPLE"
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.NotEmpty(t, findings)
		assert.Contains(t, findings[0].Message, "AWS Access Key")
		assert.Equal(t, sdk.SeverityError, findings[0].Severity)
	})

	t.Run("no finding for variable references", func(t *testing.T) {
		content := `
resource "aws_instance" "test" {
  ami = var.ami_id
  tags = {
    secret = var.my_secret
  }
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Empty(t, findings, "should not flag variable references")
	})

	t.Run("detects hardcoded password", func(t *testing.T) {
		content := `
resource "aws_db_instance" "test" {
  password = "mysecretpassword123"
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		require.NotEmpty(t, findings)
	})

	t.Run("no finding for clean config", func(t *testing.T) {
		content := `
resource "aws_instance" "test" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

		parser := hclparse.NewParser()
		hclFile, diags := parser.ParseHCLFile(file)
		require.False(t, diags.HasErrors())

		ctx := &sdk.Context{
			File:     file,
			AllFiles: map[string][]byte{file: []byte(content)},
		}

		findings, err := rule.Check(ctx, hclFile)
		require.NoError(t, err)
		assert.Empty(t, findings)
	})
}

// mockPluginRule is a simple rule implementation for testing plugin integration
type mockPluginRule struct {
	name string
}

func (r *mockPluginRule) Name() string        { return r.name }
func (r *mockPluginRule) Description() string { return "Mock plugin rule for testing" }
func (r *mockPluginRule) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return nil, nil
}

func TestNew_AcceptsPluginRules(t *testing.T) {
	t.Run("no plugin rules", func(t *testing.T) {
		engine := New(nil)

		rules := engine.GetAllRules()
		// Should have 11 built-in rules
		assert.Equal(t, 11, len(rules))
	})

	t.Run("with single plugin rule", func(t *testing.T) {
		pluginRule := &mockPluginRule{name: "plugin.test-rule"}
		engine := New(nil, pluginRule)

		rules := engine.GetAllRules()
		// Should have 11 built-in + 1 plugin = 12 rules
		assert.Equal(t, 12, len(rules))

		// Plugin rule should be present
		var found bool
		for _, r := range rules {
			if r.Name() == "plugin.test-rule" {
				found = true
				break
			}
		}
		assert.True(t, found, "plugin rule should be registered")
	})

	t.Run("with multiple plugin rules", func(t *testing.T) {
		plugin1 := &mockPluginRule{name: "plugin.rule-one"}
		plugin2 := &mockPluginRule{name: "plugin.rule-two"}
		plugin3 := &mockPluginRule{name: "plugin.rule-three"}

		engine := New(nil, plugin1, plugin2, plugin3)

		rules := engine.GetAllRules()
		// Should have 11 built-in + 3 plugin = 14 rules
		assert.Equal(t, 14, len(rules))
	})
}

func TestNew_PluginRulesAppendedAfterBuiltIn(t *testing.T) {
	pluginRule := &mockPluginRule{name: "plugin.test-rule"}
	engine := New(nil, pluginRule)

	rules := engine.GetAllRules()

	// Plugin rules should be at the end of the slice
	lastRule := rules[len(rules)-1]

	assert.Equal(t, "plugin.test-rule", lastRule.Name(), "plugin rule should be appended after built-in rules")
}
