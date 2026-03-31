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

// GroupFilesByDirectory groups file paths by their parent directory.
func GroupFilesByDirectory(files []string) map[string][]string {
	dirFiles := make(map[string][]string)
	for _, file := range files {
		dir := filepath.Dir(file)
		dirFiles[dir] = append(dirFiles[dir], file)
	}
	return dirFiles
}
