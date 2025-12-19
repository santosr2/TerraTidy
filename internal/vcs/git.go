// Package vcs provides version control system integration.
// It supports detecting changed files for incremental checks.
package vcs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git provides Git-specific VCS operations
type Git struct {
	workDir string
}

// NewGit creates a new Git VCS instance
func NewGit(workDir string) *Git {
	if workDir == "" {
		workDir = "."
	}
	return &Git{workDir: workDir}
}

// IsGitRepo checks if the working directory is inside a Git repository
func (g *Git) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = g.workDir
	return cmd.Run() == nil
}

// GetRepoRoot returns the root directory of the Git repository
func (g *Git) GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting repo root: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentBranch returns the current branch name
func (g *Git) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetDefaultBranch returns the default branch (main or master)
func (g *Git) GetDefaultBranch() string {
	// Try to find the default branch from origin
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		if strings.HasPrefix(branch, "origin/") {
			return strings.TrimPrefix(branch, "origin/")
		}
		return branch
	}

	// Fall back to checking if main or master exists
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = g.workDir
		if cmd.Run() == nil {
			return branch
		}
	}

	return "main" // Default assumption
}

// GetChangedFiles returns files that have changed compared to the given base ref
// If base is empty, it compares to the default branch
func (g *Git) GetChangedFiles(base string) ([]string, error) {
	if base == "" {
		base = g.GetDefaultBranch()
	}

	// Get merge base to find common ancestor
	mergeBaseCmd := exec.Command("git", "merge-base", base, "HEAD")
	mergeBaseCmd.Dir = g.workDir
	mergeBaseOut, err := mergeBaseCmd.Output()
	if err != nil {
		// If merge-base fails, try diff against base directly
		return g.getChangedFilesDirect(base)
	}
	mergeBase := strings.TrimSpace(string(mergeBaseOut))

	// Get changed files since merge base
	cmd := exec.Command("git", "diff", "--name-only", mergeBase, "HEAD")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}

	return g.parseFileList(out)
}

// getChangedFilesDirect gets changed files by diffing directly against base
func (g *Git) getChangedFilesDirect(base string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", base)
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}
	return g.parseFileList(out)
}

// GetStagedFiles returns files that are staged for commit
func (g *Git) GetStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--cached")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting staged files: %w", err)
	}
	return g.parseFileList(out)
}

// GetUnstagedFiles returns files that have unstaged changes
func (g *Git) GetUnstagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting unstaged files: %w", err)
	}
	return g.parseFileList(out)
}

// GetUntrackedFiles returns untracked files
func (g *Git) GetUntrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting untracked files: %w", err)
	}
	return g.parseFileList(out)
}

// GetAllChanges returns all changed, staged, and untracked files
func (g *Git) GetAllChanges() ([]string, error) {
	files := make(map[string]bool)

	// Staged
	staged, err := g.GetStagedFiles()
	if err == nil {
		for _, f := range staged {
			files[f] = true
		}
	}

	// Unstaged
	unstaged, err := g.GetUnstagedFiles()
	if err == nil {
		for _, f := range unstaged {
			files[f] = true
		}
	}

	// Untracked
	untracked, err := g.GetUntrackedFiles()
	if err == nil {
		for _, f := range untracked {
			files[f] = true
		}
	}

	result := make([]string, 0, len(files))
	for f := range files {
		result = append(result, f)
	}
	return result, nil
}

// GetChangedTerraformFiles returns only .tf files that have changed
func (g *Git) GetChangedTerraformFiles(base string) ([]string, error) {
	files, err := g.GetChangedFiles(base)
	if err != nil {
		return nil, err
	}
	return g.filterTerraformFiles(files), nil
}

// GetAllChangedTerraformFiles returns all changed .tf files (including staged/unstaged)
func (g *Git) GetAllChangedTerraformFiles() ([]string, error) {
	files, err := g.GetAllChanges()
	if err != nil {
		return nil, err
	}
	return g.filterTerraformFiles(files), nil
}

// filterTerraformFiles filters a list to only include Terraform files
func (g *Git) filterTerraformFiles(files []string) []string {
	var result []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".tf" || ext == ".tfvars" || ext == ".hcl" {
			result = append(result, f)
		}
	}
	return result
}

// parseFileList parses git output into a list of files
func (g *Git) parseFileList(out []byte) ([]string, error) {
	var files []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// Make paths absolute
			if !filepath.IsAbs(line) {
				repoRoot, err := g.GetRepoRoot()
				if err == nil {
					line = filepath.Join(repoRoot, line)
				}
			}
			files = append(files, line)
		}
	}
	return files, scanner.Err()
}

// ToAbsolutePaths converts relative paths to absolute paths
func (g *Git) ToAbsolutePaths(files []string) ([]string, error) {
	repoRoot, err := g.GetRepoRoot()
	if err != nil {
		return nil, err
	}

	result := make([]string, len(files))
	for i, f := range files {
		if filepath.IsAbs(f) {
			result[i] = f
		} else {
			result[i] = filepath.Join(repoRoot, f)
		}
	}
	return result, nil
}

// FileStatus represents the Git status of a file (M=modified, A=added, D=deleted, etc.)
type FileStatus struct {
	Path   string
	Status string
}

// GetFileStatuses returns the Git status of all changed files in the repository.
func (g *Git) GetFileStatuses() ([]FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting file statuses: %w", err)
	}

	var statuses []FileStatus
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) >= 3 {
			status := strings.TrimSpace(line[:2])
			path := strings.TrimSpace(line[3:])
			if path != "" {
				statuses = append(statuses, FileStatus{
					Path:   path,
					Status: status,
				})
			}
		}
	}
	return statuses, scanner.Err()
}

// FilterExisting filters a list of files to only those that exist
func FilterExisting(files []string) []string {
	var result []string
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			result = append(result, f)
		}
	}
	return result
}
