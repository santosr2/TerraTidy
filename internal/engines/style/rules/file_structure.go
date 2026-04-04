package rules

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// ScopedFileOrganizationRule ensures related resources/data/modules are in scoped files.
// For example, network-related resources should be in network.tf, database resources in database.tf, etc.
type ScopedFileOrganizationRule struct{}

// Name returns the rule identifier.
func (r *ScopedFileOrganizationRule) Name() string {
	return "style.scoped-file-organization"
}

// Description returns a human-readable description of the rule.
func (r *ScopedFileOrganizationRule) Description() string {
	return "Ensures related resources/data/modules are organized in scoped files (e.g., network.tf for networking)"
}

// scopeKeywords maps file scope names to related resource type keywords
var scopeKeywords = map[string][]string{
	"network": {
		"vpc", "subnet", "route", "gateway", "nat", "internet", "eip", "elastic_ip",
		"network", "security_group", "nacl", "acl", "peering", "endpoint", "transit",
		"vpn", "direct_connect", "cloudfront", "alb", "elb", "nlb", "load_balancer",
		"lb", "target_group", "listener",
	},
	"compute": {
		"instance", "ec2", "launch_template", "autoscaling", "asg", "spot",
		"ami", "ebs", "volume", "snapshot", "placement",
	},
	"storage": {
		"s3", "bucket", "object", "glacier", "efs", "fsx", "storage",
	},
	"database": {
		"rds", "aurora", "dynamodb", "elasticache", "redis", "memcached",
		"neptune", "documentdb", "database", "db", "cluster", "instance",
	},
	"iam": {
		"iam", "role", "policy", "user", "group", "permission", "assume",
		"identity", "access", "principal",
	},
	"security": {
		"kms", "key", "secret", "ssm", "parameter", "certificate", "acm",
		"waf", "shield", "guardduty", "inspector", "macie",
	},
	"monitoring": {
		"cloudwatch", "alarm", "metric", "log", "dashboard", "event",
		"sns", "sqs", "notification",
	},
	"lambda": {
		"lambda", "function", "layer", "permission", "event_source",
	},
	"container": {
		"ecs", "ecr", "eks", "fargate", "task", "service", "cluster",
		"container", "docker", "kubernetes", "k8s",
	},
	"dns": {
		"route53", "hosted_zone", "record", "domain", "dns",
	},
}

// Check examines files for misplaced resources based on file scope.
func (r *ScopedFileOrganizationRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	// Determine file scope from filename
	fileName := filepath.Base(ctx.File)
	fileScope := r.getFileScope(fileName)

	// If file doesn't have a recognized scope, skip check
	if fileScope == "" {
		return findings, nil
	}

	// Check each block to see if it belongs in this file
	for _, block := range hclFile.Blocks {
		if block.Type != "resource" && block.Type != "data" && block.Type != "module" {
			continue
		}

		var resourceType string
		if len(block.Labels) > 0 {
			resourceType = block.Labels[0]
		}

		// Check if this resource matches the file scope
		resourceScope := r.getResourceScope(resourceType)
		if resourceScope != "" && resourceScope != fileScope {
			findings = append(findings, sdk.Finding{
				Rule:     r.Name(),
				Message:  resourceType + " resource may be better placed in " + resourceScope + ".tf",
				File:     ctx.File,
				Location: sdk.LocationFromRange(block.Range()),
				Severity: sdk.SeverityInfo,
				Fixable:  false,
			})
		}
	}

	return findings, nil
}

// getFileScope extracts the scope from a filename (e.g., "network" from "network.tf")
func (r *ScopedFileOrganizationRule) getFileScope(fileName string) string {
	// Remove .tf extension
	name := strings.TrimSuffix(fileName, ".tf")

	// Check if this matches any known scope
	for scope := range scopeKeywords {
		if name == scope || strings.HasPrefix(name, scope+"_") || strings.HasSuffix(name, "_"+scope) {
			return scope
		}
	}

	return ""
}

// getResourceScope determines what scope a resource type belongs to
func (r *ScopedFileOrganizationRule) getResourceScope(resourceType string) string {
	resourceLower := strings.ToLower(resourceType)

	for scope, keywords := range scopeKeywords {
		for _, keyword := range keywords {
			if strings.Contains(resourceLower, keyword) {
				return scope
			}
		}
	}

	return ""
}

// Fix is a no-op for this rule as moving resources between files requires manual review.
func (r *ScopedFileOrganizationRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// TerraformFilesStructureRule ensures standard file structure is followed.
// Standard files: variables.tf, outputs.tf, providers.tf, versions.tf, main.tf, locals.tf
type TerraformFilesStructureRule struct{}

// Name returns the rule identifier.
func (r *TerraformFilesStructureRule) Name() string {
	return "style.terraform-files-structure"
}

// Description returns a human-readable description of the rule.
func (r *TerraformFilesStructureRule) Description() string {
	return "Ensures standard Terraform file structure (variables.tf, outputs.tf, providers.tf, etc.)"
}

// standardFileBlocks maps standard file names to expected block types
var standardFileBlocks = map[string][]string{
	"variables.tf": {"variable"},
	"outputs.tf":   {"output"},
	"providers.tf": {"provider", "terraform"},
	"versions.tf":  {"terraform"},
	"locals.tf":    {"locals"},
	"data.tf":      {"data"},
}

// Check examines files for blocks that should be in standard files.
func (r *TerraformFilesStructureRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	hclFile, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return findings, nil
	}

	fileName := filepath.Base(ctx.File)

	// Get list of expected block types for this file
	expectedTypes := standardFileBlocks[fileName]

	// Skip if this is a standard file
	if len(expectedTypes) > 0 {
		return findings, nil
	}

	// Check if blocks in this file should be in a standard file
	for _, block := range hclFile.Blocks {
		for standardFile, types := range standardFileBlocks {
			for _, blockType := range types {
				if block.Type == blockType {
					// Check if the standard file exists
					standardPath := filepath.Join(filepath.Dir(ctx.File), standardFile)
					if r.fileExists(standardPath) || r.shouldSuggestStandardFile(block.Type) {
						findings = append(findings, sdk.Finding{
							Rule:     r.Name(),
							Message:  block.Type + " block should be in " + standardFile,
							File:     ctx.File,
							Location: sdk.LocationFromRange(block.Range()),
							Severity: sdk.SeverityInfo,
							Fixable:  false,
						})
					}
				}
			}
		}
	}

	return findings, nil
}

func (r *TerraformFilesStructureRule) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// shouldSuggestStandardFile returns true if this block type is commonly placed in standard files
func (r *TerraformFilesStructureRule) shouldSuggestStandardFile(blockType string) bool {
	// Always suggest standard files for these common block types
	commonTypes := map[string]bool{
		"variable": true,
		"output":   true,
		"provider": true,
	}
	return commonTypes[blockType]
}

// Fix is a no-op for this rule as moving blocks between files requires manual review.
func (r *TerraformFilesStructureRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}
