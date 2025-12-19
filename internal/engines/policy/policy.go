// Package policy provides policy enforcement for Terraform configurations using OPA/Rego.
// It allows users to define custom policies and evaluate Terraform code against them.
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// Engine represents the policy engine with OPA/Rego support
type Engine struct {
	config *Config
	parser *hclparse.Parser
}

// Config holds the policy engine configuration
type Config struct {
	PolicyDirs  []string              // Directories containing Rego policy files
	PolicyFiles []string              // Individual policy files
	DataFiles   []string              // Additional data files
	Options     map[string]any        // Additional options
	Rules       map[string]RuleConfig // Rule-specific configuration
}

// RuleConfig holds configuration for a single policy rule
type RuleConfig struct {
	Enabled  bool
	Severity string
	Options  map[string]any
}

// New creates a new policy engine
func New(config *Config) *Engine {
	if config == nil {
		config = &Config{
			PolicyDirs:  []string{},
			PolicyFiles: []string{},
			Rules:       make(map[string]RuleConfig),
		}
	}
	if config.Rules == nil {
		config.Rules = make(map[string]RuleConfig)
	}

	return &Engine{
		config: config,
		parser: hclparse.NewParser(),
	}
}

// Name returns the engine name
func (e *Engine) Name() string {
	return "policy"
}

// Run executes policy checks on the given files
func (e *Engine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	allFindings := []sdk.Finding{}

	// Load policies
	policies, err := e.loadPolicies()
	if err != nil {
		return nil, fmt.Errorf("loading policies: %w", err)
	}

	if len(policies) == 0 {
		// No policies configured - return empty
		return allFindings, nil
	}

	// Group files by directory for module-level analysis
	dirFiles := e.groupFilesByDirectory(files)

	for dir, dirFileList := range dirFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Parse and convert all files in the module to JSON representation
		moduleData, err := e.parseModuleToJSON(dirFileList)
		if err != nil {
			allFindings = append(allFindings, sdk.Finding{
				Rule:     "policy.parse-error",
				Message:  fmt.Sprintf("Failed to parse module: %v", err),
				File:     dir,
				Severity: sdk.SeverityError,
				Fixable:  false,
			})
			continue
		}

		// Evaluate policies against the module data
		findings, err := e.evaluatePolicies(ctx, policies, moduleData, dir)
		if err != nil {
			return nil, fmt.Errorf("evaluating policies for %s: %w", dir, err)
		}

		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// loadPolicies loads all Rego policy files
func (e *Engine) loadPolicies() ([]string, error) {
	var policies []string

	// Load from policy directories
	for _, dir := range e.config.PolicyDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".rego") {
				content, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading %s: %w", path, err)
				}
				policies = append(policies, string(content))
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("walking %s: %w", dir, err)
		}
	}

	// Load individual policy files
	for _, file := range e.config.PolicyFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}
		policies = append(policies, string(content))
	}

	// Add built-in policies if no custom policies provided
	if len(policies) == 0 {
		policies = append(policies, builtinPolicies...)
	}

	return policies, nil
}

// parseModuleToJSON parses Terraform files and converts to JSON representation for OPA
func (e *Engine) parseModuleToJSON(files []string) (map[string]any, error) {
	moduleData := newModuleData()

	for _, file := range files {
		e.parseFileIntoModule(file, moduleData)
	}

	return moduleData, nil
}

func newModuleData() map[string]any {
	return map[string]any{
		"resources": []any{},
		"data":      []any{},
		"modules":   []any{},
		"variables": []any{},
		"outputs":   []any{},
		"locals":    []any{},
		"providers": []any{},
		"terraform": map[string]any{},
		"_files":    []string{},
	}
}

func (e *Engine) parseFileIntoModule(file string, moduleData map[string]any) {
	content, err := os.ReadFile(file)
	if err != nil {
		return
	}

	hclFile, diags := e.parser.ParseHCL(content, file)
	if diags.HasErrors() {
		return
	}

	body, ok := hclFile.Body.(*hclsyntax.Body)
	if !ok {
		return
	}

	moduleData["_files"] = append(moduleData["_files"].([]string), file)

	for _, block := range body.Blocks {
		blockData := e.extractBlockData(block, content)
		blockData["_file"] = file
		e.addBlockToModule(block, blockData, moduleData)
	}
}

func (e *Engine) addBlockToModule(block *hclsyntax.Block, blockData, moduleData map[string]any) {
	switch block.Type {
	case "resource":
		addLabeledBlock(blockData, block.Labels, 2, "type", "name")
		appendToSlice(moduleData, "resources", blockData)
	case "data":
		addLabeledBlock(blockData, block.Labels, 2, "type", "name")
		appendToSlice(moduleData, "data", blockData)
	case "module":
		addLabeledBlock(blockData, block.Labels, 1, "name")
		appendToSlice(moduleData, "modules", blockData)
	case "variable":
		addLabeledBlock(blockData, block.Labels, 1, "name")
		appendToSlice(moduleData, "variables", blockData)
	case "output":
		addLabeledBlock(blockData, block.Labels, 1, "name")
		appendToSlice(moduleData, "outputs", blockData)
	case "locals":
		appendToSlice(moduleData, "locals", blockData)
	case "provider":
		addLabeledBlock(blockData, block.Labels, 1, "name")
		appendToSlice(moduleData, "providers", blockData)
	case "terraform":
		for k, v := range blockData {
			moduleData["terraform"].(map[string]any)[k] = v
		}
	}
}

func addLabeledBlock(blockData map[string]any, labels []string, minLabels int, keys ...string) {
	if len(labels) >= minLabels {
		for i, key := range keys {
			if i < len(labels) {
				blockData[key] = labels[i]
			}
		}
	}
}

func appendToSlice(moduleData map[string]any, key string, blockData map[string]any) {
	moduleData[key] = append(moduleData[key].([]any), blockData)
}

// extractBlockData extracts data from an HCL block into a map
func (e *Engine) extractBlockData(block *hclsyntax.Block, content []byte) map[string]any {
	data := map[string]any{
		"_block_type": block.Type,
		"_range": map[string]any{
			"start_line":   block.Range().Start.Line,
			"end_line":     block.Range().End.Line,
			"start_column": block.Range().Start.Column,
			"end_column":   block.Range().End.Column,
		},
	}

	// Extract attributes
	for name, attr := range block.Body.Attributes {
		// Get the raw expression text
		exprRange := attr.Expr.Range()
		if exprRange.Start.Byte < len(content) && exprRange.End.Byte <= len(content) {
			exprText := string(content[exprRange.Start.Byte:exprRange.End.Byte])
			data[name] = exprText
		}
	}

	// Extract nested blocks
	for _, nested := range block.Body.Blocks {
		nestedData := e.extractBlockData(nested, content)
		key := nested.Type
		if len(nested.Labels) > 0 {
			key = nested.Type + "_" + nested.Labels[0]
		}

		if existing, ok := data[nested.Type]; ok {
			// If already exists as slice, append
			if slice, ok := existing.([]any); ok {
				data[nested.Type] = append(slice, nestedData)
			} else {
				// Convert to slice
				data[nested.Type] = []any{existing, nestedData}
			}
		} else {
			data[key] = nestedData
		}
	}

	return data
}

// policyEvalContext holds context for policy evaluation.
type policyEvalContext struct {
	ctx        context.Context
	moduleData map[string]any
	dir        string
}

// evaluatePolicies evaluates all policies against the module data.
func (e *Engine) evaluatePolicies(
	ctx context.Context,
	policies []string,
	moduleData map[string]any,
	dir string,
) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	evalCtx := &policyEvalContext{ctx: ctx, moduleData: moduleData, dir: dir}

	for _, policy := range policies {
		// Evaluate deny rules
		denyFindings := e.evaluateQuery(evalCtx, policy, "data.terraform.deny", sdk.SeverityError)
		findings = append(findings, denyFindings...)

		// Evaluate warn rules
		warnFindings := e.evaluateQuery(evalCtx, policy, "data.terraform.warn", sdk.SeverityWarning)
		findings = append(findings, warnFindings...)
	}

	return findings, nil
}

// evaluateQuery evaluates a single Rego query and returns findings.
func (e *Engine) evaluateQuery(
	evalCtx *policyEvalContext,
	policy string,
	query string,
	severity sdk.Severity,
) []sdk.Finding {
	r := rego.New(
		rego.Query(query),
		rego.Module("policy.rego", policy),
		rego.Input(evalCtx.moduleData),
	)

	rs, err := r.Eval(evalCtx.ctx)
	if err != nil {
		return nil
	}

	return e.extractFindings(rs, evalCtx.dir, severity)
}

// extractFindings extracts findings from Rego result set.
func (e *Engine) extractFindings(rs rego.ResultSet, dir string, severity sdk.Severity) []sdk.Finding {
	var findings []sdk.Finding

	for _, result := range rs {
		for _, expr := range result.Expressions {
			violations, ok := expr.Value.([]any)
			if !ok {
				continue
			}
			for _, v := range violations {
				finding := e.violationToFinding(v, dir)
				finding.Severity = severity
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// violationToFinding converts a policy violation to a Finding
func (e *Engine) violationToFinding(violation any, dir string) sdk.Finding {
	finding := sdk.Finding{
		Rule:     "policy.violation",
		Severity: sdk.SeverityError,
		Fixable:  false,
	}

	switch v := violation.(type) {
	case string:
		finding.Message = v
		finding.File = dir

	case map[string]any:
		if msg, ok := v["msg"].(string); ok {
			finding.Message = msg
		}
		if rule, ok := v["rule"].(string); ok {
			finding.Rule = "policy." + rule
		}
		if file, ok := v["file"].(string); ok {
			finding.File = file
		} else {
			finding.File = dir
		}
		if severity, ok := v["severity"].(string); ok {
			finding.Severity = parseSeverity(severity)
		}
		if line, ok := v["line"].(float64); ok {
			finding.Location = hcl.Range{
				Filename: finding.File,
				Start:    hcl.Pos{Line: int(line), Column: 1},
			}
		}
	}

	return finding
}

// groupFilesByDirectory groups files by their parent directory
func (e *Engine) groupFilesByDirectory(files []string) map[string][]string {
	dirFiles := make(map[string][]string)

	for _, file := range files {
		dir := filepath.Dir(file)
		dirFiles[dir] = append(dirFiles[dir], file)
	}

	return dirFiles
}

// parseSeverity converts string severity to sdk.Severity
func parseSeverity(severity string) sdk.Severity {
	switch strings.ToLower(severity) {
	case "error":
		return sdk.SeverityError
	case "warning":
		return sdk.SeverityWarning
	case "info":
		return sdk.SeverityInfo
	default:
		return sdk.SeverityError
	}
}

// GetInput returns the module data as JSON for debugging
func (e *Engine) GetInput(files []string) ([]byte, error) {
	data, err := e.parseModuleToJSON(files)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(data, "", "  ")
}

// builtinPolicies contains default policies (OPA v1 Rego syntax)
var builtinPolicies = []string{
	// Required version policy
	`package terraform

import rego.v1

deny contains msg if {
    count(input.terraform) == 0
    msg := {
        "msg": "Missing terraform block with required_version",
        "rule": "required-terraform-block",
        "severity": "warning"
    }
}

deny contains msg if {
    tf := input.terraform
    not tf.required_version
    msg := {
        "msg": "Missing required_version in terraform block",
        "rule": "required-version",
        "severity": "warning"
    }
}
`,
	// Required providers policy
	`package terraform

import rego.v1

deny contains msg if {
    count(input.providers) > 0
    count(input.terraform) == 0
    msg := {
        "msg": "Provider used without required_providers block",
        "rule": "required-providers",
        "severity": "warning"
    }
}
`,
	// Security policies
	`package terraform

import rego.v1

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_security_group"
    contains(resource.ingress, "0.0.0.0/0")
    contains(resource.ingress, "22")
    msg := {
        "msg": sprintf("Security group %s allows SSH from 0.0.0.0/0", [resource.name]),
        "rule": "no-public-ssh",
        "severity": "error",
        "file": resource._file
    }
}

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_s3_bucket"
    resource.acl == "\"public-read\""
    msg := {
        "msg": sprintf("S3 bucket %s has public-read ACL", [resource.name]),
        "rule": "no-public-s3",
        "severity": "error",
        "file": resource._file
    }
}

deny contains msg if {
    some resource in input.resources
    resource.type == "aws_db_instance"
    resource.publicly_accessible == "true"
    msg := {
        "msg": sprintf("RDS instance %s is publicly accessible", [resource.name]),
        "rule": "no-public-rds",
        "severity": "error",
        "file": resource._file
    }
}
`,
	// Tagging policies
	`package terraform

import rego.v1

warn contains msg if {
    some resource in input.resources
    resource.type == "aws_instance"
    not resource.tags
    msg := {
        "msg": sprintf("EC2 instance %s is missing tags", [resource.name]),
        "rule": "required-tags",
        "severity": "warning",
        "file": resource._file
    }
}

warn contains msg if {
    some resource in input.resources
    resource.type == "aws_s3_bucket"
    not resource.tags
    msg := {
        "msg": sprintf("S3 bucket %s is missing tags", [resource.name]),
        "rule": "required-tags",
        "severity": "warning",
        "file": resource._file
    }
}
`,
	// Module source policy
	`package terraform

import rego.v1

warn contains msg if {
    some module in input.modules
    not module.version
    not startswith(module.source, "\"./")
    not startswith(module.source, "\"../")
    msg := {
        "msg": sprintf("Module %s should have a version constraint", [module.name]),
        "rule": "module-version",
        "severity": "warning",
        "file": module._file
    }
}
`,
}
