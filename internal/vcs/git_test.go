package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGit(t *testing.T) {
	// Test with empty work dir (defaults to ".")
	git := NewGit("")
	assert.Equal(t, ".", git.workDir)

	// Test with specific work dir
	git = NewGit("/tmp")
	assert.Equal(t, "/tmp", git.workDir)
}

func TestGit_IsGitRepo(t *testing.T) {
	// Current directory should be a git repo (TerraTidy project)
	git := NewGit(".")
	assert.True(t, git.IsGitRepo())

	// Non-existent directory should not be a git repo
	git = NewGit("/nonexistent")
	assert.False(t, git.IsGitRepo())
}

func TestGit_GetRepoRoot(t *testing.T) {
	git := NewGit(".")
	root, err := git.GetRepoRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
	assert.True(t, filepath.IsAbs(root))

	// Verify it's actually a directory
	info, err := os.Stat(root)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGit_GetCurrentBranch(t *testing.T) {
	git := NewGit(".")
	branch, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.NotEmpty(t, branch)
}

func TestGit_GetDefaultBranch(t *testing.T) {
	git := NewGit(".")
	branch := git.GetDefaultBranch()
	// Should return main or master
	assert.True(t, branch == "main" || branch == "master",
		"expected main or master, got %s", branch)
}

func TestGit_FilterTerraformFiles(t *testing.T) {
	git := NewGit(".")

	files := []string{
		"main.tf",
		"variables.tf",
		"README.md",
		"config.yaml",
		"terraform.tfvars",
		".tflint.hcl",
		"test.go",
	}

	filtered := git.filterTerraformFiles(files)

	assert.Len(t, filtered, 4)
	assert.Contains(t, filtered, "main.tf")
	assert.Contains(t, filtered, "variables.tf")
	assert.Contains(t, filtered, "terraform.tfvars")
	assert.Contains(t, filtered, ".tflint.hcl")
	assert.NotContains(t, filtered, "README.md")
	assert.NotContains(t, filtered, "config.yaml")
	assert.NotContains(t, filtered, "test.go")
}

func TestFilterExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	existingFile := filepath.Join(tmpDir, "exists.tf")
	require.NoError(t, os.WriteFile(existingFile, []byte(""), 0o644))

	files := []string{
		existingFile,
		filepath.Join(tmpDir, "nonexistent.tf"),
	}

	filtered := FilterExisting(files)

	assert.Len(t, filtered, 1)
	assert.Contains(t, filtered, existingFile)
}

func TestGit_ParseFileList(t *testing.T) {
	git := NewGit(".")

	// Mock git output
	output := []byte("main.tf\nvariables.tf\nmodules/vpc/main.tf\n\n")

	files, err := git.parseFileList(output)
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestGit_InRealRepo(t *testing.T) {
	// Skip if not running in a real git repo
	git := NewGit(".")
	if !git.IsGitRepo() {
		t.Skip("Not running in a git repository")
	}

	t.Run("GetChangedFiles", func(_ *testing.T) {
		// Result depends on repo state; we just verify it doesn't panic.
		// May error if no default branch found, that's acceptable.
		_, _ = git.GetChangedFiles("")
	})

	t.Run("GetStagedFiles", func(t *testing.T) {
		files, err := git.GetStagedFiles()
		require.NoError(t, err)
		// Result depends on repo state
		_ = files
	})

	t.Run("GetUnstagedFiles", func(t *testing.T) {
		files, err := git.GetUnstagedFiles()
		require.NoError(t, err)
		_ = files
	})

	t.Run("GetUntrackedFiles", func(t *testing.T) {
		files, err := git.GetUntrackedFiles()
		require.NoError(t, err)
		_ = files
	})

	t.Run("GetAllChanges", func(t *testing.T) {
		files, err := git.GetAllChanges()
		require.NoError(t, err)
		_ = files
	})

	t.Run("GetFileStatuses", func(t *testing.T) {
		statuses, err := git.GetFileStatuses()
		require.NoError(t, err)
		_ = statuses
	})
}

func TestGit_NewTempRepo(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git user for commits and disable GPG signing
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "commit.gpgsign", "false")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	git := NewGit(tmpDir)

	t.Run("IsGitRepo", func(t *testing.T) {
		assert.True(t, git.IsGitRepo())
	})

	t.Run("GetRepoRoot", func(t *testing.T) {
		root, err := git.GetRepoRoot()
		require.NoError(t, err)
		// Resolve symlinks for comparison (macOS /tmp is a symlink)
		expectedDir, _ := filepath.EvalSymlinks(tmpDir)
		actualDir, _ := filepath.EvalSymlinks(root)
		assert.Equal(t, expectedDir, actualDir)
	})

	// Create and stage a file
	testFile := filepath.Join(tmpDir, "main.tf")
	require.NoError(t, os.WriteFile(testFile, []byte("resource {}"), 0o644))

	cmd = exec.Command("git", "add", "main.tf")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	t.Run("GetStagedFiles", func(t *testing.T) {
		files, err := git.GetStagedFiles()
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	// Commit the file
	cmd = exec.Command("git", "commit", "-m", "initial", "--no-verify")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create an unstaged change
	require.NoError(t, os.WriteFile(testFile, []byte("resource { updated }"), 0o644))

	t.Run("GetUnstagedFiles", func(t *testing.T) {
		files, err := git.GetUnstagedFiles()
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	// Create an untracked file
	untrackedFile := filepath.Join(tmpDir, "new.tf")
	require.NoError(t, os.WriteFile(untrackedFile, []byte("new resource"), 0o644))

	t.Run("GetUntrackedFiles", func(t *testing.T) {
		files, err := git.GetUntrackedFiles()
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("GetAllChanges", func(t *testing.T) {
		files, err := git.GetAllChanges()
		require.NoError(t, err)
		// Should have both unstaged and untracked
		assert.Len(t, files, 2)
	})

	t.Run("GetAllChangedTerraformFiles", func(t *testing.T) {
		files, err := git.GetAllChangedTerraformFiles()
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("GetFileStatuses", func(t *testing.T) {
		statuses, err := git.GetFileStatuses()
		require.NoError(t, err)
		assert.Len(t, statuses, 2)

		// Verify status types
		statusMap := make(map[string]string)
		for _, s := range statuses {
			statusMap[filepath.Base(s.Path)] = s.Status
		}

		assert.Equal(t, "M", statusMap["main.tf"]) // Modified
		assert.Equal(t, "??", statusMap["new.tf"]) // Untracked
	})
}

func TestGit_ToAbsolutePaths(t *testing.T) {
	git := NewGit(".")

	if !git.IsGitRepo() {
		t.Skip("Not running in a git repository")
	}

	files := []string{"main.tf", "modules/vpc/main.tf"}
	absPaths, err := git.ToAbsolutePaths(files)
	require.NoError(t, err)

	for _, p := range absPaths {
		assert.True(t, filepath.IsAbs(p), "expected absolute path, got %s", p)
	}
}

func TestSanitizeGitError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := sanitizeGitError(nil)
		assert.Nil(t, result)
	})

	t.Run("simple error without path", func(t *testing.T) {
		err := fmt.Errorf("git command failed")
		result := sanitizeGitError(err)
		assert.EqualError(t, result, "git command failed")
	})

	t.Run("error with absolute path", func(t *testing.T) {
		err := fmt.Errorf("fatal: could not read '/home/user/secret/repo/.git/config'")
		result := sanitizeGitError(err)
		assert.Contains(t, result.Error(), "config")
		assert.NotContains(t, result.Error(), "/home/user/secret")
	})

	t.Run("error with multiple paths", func(t *testing.T) {
		err := fmt.Errorf("error: could not read '/etc/passwd' or '/home/user/.ssh/id_rsa'")
		result := sanitizeGitError(err)
		assert.NotContains(t, result.Error(), "/etc/")
		assert.NotContains(t, result.Error(), "/home/user/")
	})

	t.Run("truncates long error", func(t *testing.T) {
		longMsg := strings.Repeat("x", 600)
		err := fmt.Errorf("error: %s", longMsg)
		result := sanitizeGitError(err)
		assert.LessOrEqual(t, len(result.Error()), 520) // 500 + "... (truncated)"
		assert.Contains(t, result.Error(), "(truncated)")
	})
}

func TestValidateGitRef(t *testing.T) {
	tests := []struct {
		ref     string
		wantErr bool
	}{
		// Valid refs
		{"", false},                  // Empty is valid (uses default)
		{"main", false},              // Simple branch
		{"master", false},            // Simple branch
		{"origin/main", false},       // Remote branch
		{"feature/add-tests", false}, // Feature branch with slash
		{"v1.0.0", false},            // Tag
		{"HEAD", false},              // HEAD ref
		{"HEAD@{1}", false},          // Reflog syntax
		{"HEAD@{upstream}", false},   // Upstream ref
		{"refs/heads/main", false},   // Full ref path
		{"abc123", false},            // Short SHA
		{"abc123def456", false},      // Longer SHA
		{"release-1.0", false},       // Branch with hyphen
		{"feature_branch", false},    // Branch with underscore
		{"v1.2.3-rc1", false},        // Prerelease tag

		// Invalid refs (shell injection attempts)
		{"; rm -rf /", true},             // Command injection
		{"$(whoami)", true},              // Command substitution
		{"`id`", true},                   // Backtick command
		{"main; echo pwned", true},       // Semicolon injection
		{"main | cat /etc/passwd", true}, // Pipe injection
		{"main && echo pwned", true},     // AND injection
		{"main || echo pwned", true},     // OR injection
		{"main\necho pwned", true},       // Newline injection
		{"main'", true},                  // Single quote
		{"main\"", true},                 // Double quote
		{"$(cat /etc/passwd)", true},     // Nested command
		{"main > /tmp/out", true},        // Redirect
		{"main < /etc/passwd", true},     // Input redirect
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			err := ValidateGitRef(tt.ref)
			if tt.wantErr {
				assert.Error(t, err, "ValidateGitRef(%q) should return error", tt.ref)
				if err != nil {
					assert.Contains(t, err.Error(), "invalid git ref")
				}
			} else {
				assert.NoError(t, err, "ValidateGitRef(%q) should not return error", tt.ref)
			}
		})
	}
}

func TestGetChangedFiles_InvalidRef(t *testing.T) {
	git := NewGit(".")

	if !git.IsGitRepo() {
		t.Skip("Not running in a git repository")
	}

	// Test that invalid refs are rejected before being passed to git
	_, err := git.GetChangedFiles("; rm -rf /")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid git ref")

	_, err = git.GetChangedFiles("$(whoami)")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid git ref")
}
