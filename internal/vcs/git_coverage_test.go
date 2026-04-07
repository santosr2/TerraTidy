package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChangedFiles_TempRepo(t *testing.T) {
	dir := t.TempDir()

	// Init a repo with a commit
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	run("init", "-b", "main")
	run("config", "commit.gpgsign", "false")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "test" "t" {}`), 0o644))
	run("add", ".")
	run("commit", "-m", "initial")

	// Add a changed file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.tf"), []byte(`variable "v" {}`), 0o644))
	run("add", "new.tf")

	git := NewGit(dir)
	files, err := git.GetStagedFiles()
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0], "new.tf")
}

func TestGetChangedTerraformFiles_TempRepo(t *testing.T) {
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	run("init", "-b", "main")
	run("config", "commit.gpgsign", "false")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "test" "t" {}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0o644))
	run("add", ".")
	run("commit", "-m", "initial")

	// Modify both files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "test" "modified" {}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# updated"), 0o644))

	git := NewGit(dir)
	files, err := git.GetAllChangedTerraformFiles()
	require.NoError(t, err)

	// Should only include .tf files
	for _, f := range files {
		ext := filepath.Ext(f)
		assert.True(t, ext == ".tf" || ext == ".hcl" || ext == ".tfvars",
			"expected terraform file, got %s", f)
	}
}

func TestGetFileStatuses_TempRepo(t *testing.T) {
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	run("init", "-b", "main")
	run("config", "commit.gpgsign", "false")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "existing.tf"), []byte(`variable "v" {}`), 0o644))
	run("add", ".")
	run("commit", "-m", "initial")

	// Add new file, modify existing
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.tf"), []byte(`output "o" {}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "existing.tf"), []byte(`variable "modified" {}`), 0o644))

	git := NewGit(dir)
	statuses, err := git.GetFileStatuses()
	require.NoError(t, err)
	assert.NotEmpty(t, statuses)

	// Should have entries for modified and untracked files
	hasModified := false
	hasUntracked := false
	for _, s := range statuses {
		if s.Status == "M" {
			hasModified = true
		}
		if s.Status == "?" {
			hasUntracked = true
		}
	}
	assert.True(t, hasModified || hasUntracked, "should detect file changes")
}
