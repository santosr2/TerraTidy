package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceNameConventionRule(t *testing.T) {
	rule := &ResourceNameConventionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.resource-name-convention", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case resource",
			content: `resource "aws_instance" "my_server" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase resource",
			content: `resource "aws_instance" "myServer" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "invalid PascalCase resource",
			content: `resource "aws_instance" "MyServer" {
  ami = "ami-123"
}`,
			wantFindings: 1,
		},
		{
			name: "data source is ignored",
			content: `data "aws_ami" "latestUbuntu" {
  most_recent = true
}`,
			wantFindings: 0,
		},
		{
			name: "module is ignored",
			content: `module "myModule" {
  source = "./module"
}`,
			wantFindings: 0,
		},
		{
			name: "variable is ignored",
			content: `variable "MyVariable" {
  type = string
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := runRuleOnContent(t, rule, tt.content, nil)
			assert.Len(t, findings, tt.wantFindings)
		})
	}
}

func TestDataNameConventionRule(t *testing.T) {
	rule := &DataNameConventionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.data-name-convention", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case data source",
			content: `data "aws_ami" "latest_ubuntu" {
  most_recent = true
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase data source",
			content: `data "aws_ami" "latestUbuntu" {
  most_recent = true
}`,
			wantFindings: 1,
		},
		{
			name: "resource is ignored",
			content: `resource "aws_instance" "myServer" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := runRuleOnContent(t, rule, tt.content, nil)
			assert.Len(t, findings, tt.wantFindings)
		})
	}
}

// TestNameConventionEmptyLabel verifies an empty resource/data label produces
// exactly one SeverityError finding at the rule level (no double-fire), and that
// the error surfaces on the correct rule.
func TestNameConventionEmptyLabel(t *testing.T) {
	tests := []struct {
		name    string
		rule    sdk.Rule
		content string
	}{
		{
			name: "empty resource label",
			rule: &ResourceNameConventionRule{},
			content: `resource "aws_instance" "" {
  ami = "ami-123"
}`,
		},
		{
			name: "empty data label",
			rule: &DataNameConventionRule{},
			content: `data "aws_ami" "" {
  most_recent = true
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := runRuleOnContent(t, tt.rule, tt.content, nil)
			require.Len(t, findings, 1, "empty label must report exactly one finding")
			assert.Equal(t, sdk.SeverityError, findings[0].Severity)
			assert.Equal(t, tt.rule.Name(), findings[0].Rule)
		})
	}
}

// TestNameConventionIndependentCase proves resource and data naming can be
// configured with different case conventions and are enforced independently.
func TestNameConventionIndependentCase(t *testing.T) {
	// snake_case name: valid under snake_case, invalid under kebab-case.
	snakeContent := `resource "aws_instance" "my_server" {
  ami = "ami-123"
}`
	// kebab-case name: valid under kebab-case, invalid under snake_case.
	kebabContent := `data "aws_ami" "my-ami" {
  most_recent = true
}`

	snakeCfg := map[string]any{"options": map[string]any{"case": "snake_case"}}
	kebabCfg := map[string]any{"options": map[string]any{"case": "kebab-case"}}

	resourceRule := &ResourceNameConventionRule{}
	dataRule := &DataNameConventionRule{}

	// Resource enforces snake_case: the snake_case name is clean.
	assert.Empty(t, runRuleOnContent(t, resourceRule, snakeContent, snakeCfg),
		"snake_case resource name valid under snake_case config")

	// Data enforces kebab-case simultaneously: the kebab-case name is clean.
	assert.Empty(t, runRuleOnContent(t, dataRule, kebabContent, kebabCfg),
		"kebab-case data name valid under kebab-case config")

	// The conventions are independent: swap the configs and both now flag.
	assert.Len(t, runRuleOnContent(t, resourceRule, snakeContent, kebabCfg), 1,
		"snake_case resource name invalid under kebab-case config")
	assert.Len(t, runRuleOnContent(t, dataRule, kebabContent, snakeCfg), 1,
		"kebab-case data name invalid under snake_case config")
}

func TestVariableNameConventionRule(t *testing.T) {
	rule := &VariableNameConventionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.variable-name-convention", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case variable",
			content: `variable "my_variable" {
  type = string
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase variable",
			content: `variable "myVariable" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name: "invalid PascalCase variable",
			content: `variable "MyVariable" {
  type = string
}`,
			wantFindings: 1,
		},
		{
			name: "resource is ignored",
			content: `resource "aws_instance" "myServer" {
  ami = "ami-123"
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := runRuleOnContent(t, rule, tt.content, nil)
			assert.Len(t, findings, tt.wantFindings)
		})
	}
}

func TestOutputNameConventionRule(t *testing.T) {
	rule := &OutputNameConventionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.output-name-convention", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case output",
			content: `output "my_output" {
  value = "test"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase output",
			content: `output "myOutput" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "invalid PascalCase output",
			content: `output "MyOutput" {
  value = "test"
}`,
			wantFindings: 1,
		},
		{
			name: "variable is ignored",
			content: `variable "myVariable" {
  type = string
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := runRuleOnContent(t, rule, tt.content, nil)
			assert.Len(t, findings, tt.wantFindings)
		})
	}
}

func TestLocalNameConventionRule(t *testing.T) {
	rule := &LocalNameConventionRule{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "style.local-name-convention", rule.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.NotEmpty(t, rule.Description())
	})

	tests := []struct {
		name         string
		content      string
		wantFindings int
	}{
		{
			name: "valid snake_case local",
			content: `locals {
  my_local = "value"
}`,
			wantFindings: 0,
		},
		{
			name: "invalid camelCase local",
			content: `locals {
  myLocal = "value"
}`,
			wantFindings: 1,
		},
		{
			name: "multiple locals mixed",
			content: `locals {
  valid_name   = "ok"
  invalidName  = "bad"
  another_good = "ok"
}`,
			wantFindings: 1,
		},
		{
			name: "variable block is ignored",
			content: `variable "myVariable" {
  type = string
}`,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := runRuleOnContent(t, rule, tt.content, nil)
			assert.Len(t, findings, tt.wantFindings)
		})
	}
}

// TestNameConventionRulesWithConfig tests naming rules with different naming
// conventions from config.
func TestNameConventionRulesWithConfig(t *testing.T) {
	camelCfg := map[string]any{"options": map[string]any{"case": "camelCase"}}

	t.Run("VariableNameConventionRule with camelCase config", func(t *testing.T) {
		content := `variable "myVariable" {
  type = string
}`
		findings := runRuleOnContent(t, &VariableNameConventionRule{}, content, camelCfg)
		assert.Empty(t, findings, "camelCase should be valid with camelCase config")
	})

	t.Run("VariableNameConventionRule snake_case invalid with camelCase config", func(t *testing.T) {
		content := `variable "my_variable" {
  type = string
}`
		findings := runRuleOnContent(t, &VariableNameConventionRule{}, content, camelCfg)
		assert.Len(t, findings, 1, "snake_case should be invalid with camelCase config")
	})

	t.Run("LocalNameConventionRule with kebab-case config", func(t *testing.T) {
		content := `locals {
  my-local = "value"
}`
		cfg := map[string]any{"options": map[string]any{"case": "kebab-case"}}
		findings := runRuleOnContent(t, &LocalNameConventionRule{}, content, cfg)
		assert.Empty(t, findings, "kebab-case should be valid with kebab-case config")
	})

	t.Run("OutputNameConventionRule with PascalCase config", func(t *testing.T) {
		content := `output "MyOutput" {
  value = "test"
}`
		cfg := map[string]any{"options": map[string]any{"case": "PascalCase"}}
		findings := runRuleOnContent(t, &OutputNameConventionRule{}, content, cfg)
		assert.Empty(t, findings, "PascalCase should be valid with PascalCase config")
	})

	t.Run("ResourceNameConventionRule with custom pattern config", func(t *testing.T) {
		content := `resource "aws_instance" "prefix_server" {
  ami = "ami-123"
}`
		cfg := map[string]any{"options": map[string]any{"case": "custom", "pattern": "^prefix_"}}
		findings := runRuleOnContent(t, &ResourceNameConventionRule{}, content, cfg)
		assert.Empty(t, findings, "prefix_server should match custom pattern ^prefix_")
	})

	t.Run("ResourceNameConventionRule custom pattern invalid", func(t *testing.T) {
		content := `resource "aws_instance" "server" {
  ami = "ami-123"
}`
		cfg := map[string]any{"options": map[string]any{"case": "custom", "pattern": "^prefix_"}}
		findings := runRuleOnContent(t, &ResourceNameConventionRule{}, content, cfg)
		assert.Len(t, findings, 1, "server should not match custom pattern ^prefix_")
	})
}

// runRuleOnContent parses content as HCL and runs rule against it, returning the
// findings. options may be nil.
func runRuleOnContent(t *testing.T, rule sdk.Rule, content string, options map[string]any) []sdk.Finding {
	t.Helper()

	file, diags := hclsyntax.ParseConfig([]byte(content), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	hclFile := &hcl.File{Body: file.Body}
	ctx := &sdk.Context{File: "test.tf", Options: options}

	findings, err := rule.Check(ctx, hclFile)
	require.NoError(t, err)
	return findings
}
