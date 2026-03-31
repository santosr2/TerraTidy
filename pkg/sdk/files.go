package sdk

import (
	"path/filepath"
	"strings"
)

// IsHCLFile checks if a file has a Terraform/HCL extension (.tf, .hcl, .tfvars).
func IsHCLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tf" || ext == ".hcl" || ext == ".tfvars"
}
