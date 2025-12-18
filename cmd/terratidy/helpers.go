// Package main provides CLI helpers for TerraTidy commands.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santosr2/terratidy/internal/vcs"
)

// getTargetFiles returns the list of files to process based on the provided paths
// and global flags. When --changed is set, it uses VCS to detect changed files.
func getTargetFiles(paths []string, changedOnly bool) ([]string, error) {
	if changedOnly {
		return getChangedFiles(paths)
	}
	return findHCLFilesFromPaths(paths)
}

// getChangedFiles uses VCS to get only changed Terraform/HCL files.
// If paths are provided, it filters the changed files to only those within the paths.
func getChangedFiles(filterPaths []string) ([]string, error) {
	git := vcs.NewGit(".")

	// Check if we're in a git repo
	if !git.IsGitRepo() {
		return nil, fmt.Errorf("not a git repository; --changed requires git")
	}

	// Get all changed Terraform files
	changedFiles, err := git.GetAllChangedTerraformFiles()
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}

	// If no filter paths provided, return all changed files
	if len(filterPaths) == 0 || (len(filterPaths) == 1 && filterPaths[0] == ".") {
		return vcs.FilterExisting(changedFiles), nil
	}

	// Filter changed files to only those within the specified paths
	var filteredFiles []string
	for _, file := range changedFiles {
		for _, filterPath := range filterPaths {
			absFilterPath, err := filepath.Abs(filterPath)
			if err != nil {
				continue
			}

			// Check if the file is within the filter path
			if isPathWithin(file, absFilterPath) {
				filteredFiles = append(filteredFiles, file)
				break
			}
		}
	}

	return vcs.FilterExisting(filteredFiles), nil
}

// isPathWithin checks if a file path is within a directory path.
func isPathWithin(filePath, dirPath string) bool {
	// Clean and normalize paths
	filePath = filepath.Clean(filePath)
	dirPath = filepath.Clean(dirPath)

	// Check if file starts with directory path
	if strings.HasPrefix(filePath, dirPath) {
		// Make sure it's actually within (not just a prefix match)
		remainder := strings.TrimPrefix(filePath, dirPath)
		return remainder == "" || strings.HasPrefix(remainder, string(filepath.Separator))
	}
	return false
}

// findHCLFilesFromPaths is a helper that handles default paths and delegates to findHCLFiles.
func findHCLFilesFromPaths(paths []string) ([]string, error) {
	targetPaths := paths
	if len(targetPaths) == 0 {
		targetPaths = []string{"."}
	}
	return findHCLFiles(targetPaths)
}

// findHCLFiles recursively finds all .tf and .hcl files in the given paths.
func findHCLFiles(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}

		if info.IsDir() {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Skip hidden directories and common non-terraform directories
				if info.IsDir() && shouldSkipDir(p, info.Name()) {
					return filepath.SkipDir
				}
				if !info.IsDir() && isHCLFile(p) && !seen[p] {
					absPath, err := filepath.Abs(p)
					if err != nil {
						absPath = p
					}
					files = append(files, absPath)
					seen[absPath] = true
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walking %s: %w", path, err)
			}
		} else if isHCLFile(path) && !seen[path] {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			files = append(files, absPath)
			seen[absPath] = true
		}
	}

	return files, nil
}

// shouldSkipDir returns true if the directory should be skipped during traversal.
func shouldSkipDir(_ string, name string) bool {
	// Skip hidden directories
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}
	// Skip common non-terraform directories
	skipDirs := map[string]bool{
		"node_modules":      true,
		"vendor":            true,
		".terraform":        true,
		".terragrunt-cache": true,
		"__pycache__":       true,
	}
	return skipDirs[name]
}

// isHCLFile checks if a file has .tf, .tfvars, or .hcl extension.
func isHCLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tf" || ext == ".hcl" || ext == ".tfvars"
}

// formatFileCount returns a human-readable file count string.
func formatFileCount(count int) string {
	if count == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", count)
}

// filterBySeverityThreshold filters findings based on the severity threshold.
// Returns findings that meet or exceed the threshold severity.
func filterBySeverityThreshold(threshold string, findings []FindingWithSeverity) []FindingWithSeverity {
	if threshold == "" {
		return findings
	}

	thresholdLevel := severityLevel(threshold)
	var filtered []FindingWithSeverity

	for _, f := range findings {
		if severityLevel(f.Severity) >= thresholdLevel {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

// FindingWithSeverity is a helper interface for filtering findings.
type FindingWithSeverity struct {
	Severity string
}

// severityLevel converts severity string to a numeric level for comparison.
func severityLevel(severity string) int {
	switch strings.ToLower(severity) {
	case "info":
		return 1
	case "warning":
		return 2
	case "error":
		return 3
	default:
		return 0
	}
}
